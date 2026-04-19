package postgres

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"opsxcli/internal/db"
	"opsxcli/internal/logger"
)

// DumpOptions 导出选项
type DumpOptions struct {
	Format       string   // sql/csv
	WithData     bool     // 包含数据（默认true）
	WithSchema   bool     // 包含结构（默认true）
	Tables       []string // 指定表（空=全部）
	IgnoreTables []string // 忽略的表
}

// DefaultDumpOptions 返回默认导出选项
func DefaultDumpOptions() DumpOptions {
	return DumpOptions{
		Format:     "sql",
		WithData:   true,
		WithSchema: true,
	}
}

// DumpDatabase 导出 PostgreSQL 数据库
func DumpDatabase(host string, port int, user, password, database, outputPath string, opts DumpOptions) error {
	host, port, user = db.ApplyDefaults(host, port, user, "postgres")

	conn, err := connectDB(host, port, user, password, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 获取所有表
	tables, err := getTables(conn)
	if err != nil {
		return fmt.Errorf("获取表列表失败: %w", err)
	}

	tables = filterTables(tables, opts)

	if len(tables) == 0 {
		return fmt.Errorf("没有找到需要导出的表")
	}

	logger.Info("开始导出数据库 %s，共 %d 张表", database, len(tables))

	if opts.Format == "csv" {
		return dumpDatabaseCSV(conn, database, tables, outputPath, opts)
	}
	return dumpDatabaseSQL(conn, database, tables, outputPath, opts)
}

// DumpTable 导出单个表
func DumpTable(host string, port int, user, password, database, table, outputPath string, opts DumpOptions) error {
	host, port, user = db.ApplyDefaults(host, port, user, "postgres")

	conn, err := connectDB(host, port, user, password, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	logger.Info("开始导出表 %s.%s", database, table)

	if opts.Format == "csv" {
		return dumpTableCSV(conn, database, table, outputPath, opts)
	}
	return dumpTableSQL(conn, database, table, outputPath, opts)
}

// RestoreDatabase 导入 PostgreSQL 数据库
func RestoreDatabase(host string, port int, user, password, database, inputPath string) error {
	host, port, user = db.ApplyDefaults(host, port, user, "postgres")

	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	conn, err := connectDB(host, port, user, password, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	logger.Info("开始导入数据库 %s，文件: %s", database, inputPath)

	statements := splitSQL(string(content))
	total := len(statements)
	success := 0

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := conn.Exec(stmt); err != nil {
			logger.Warning("执行第 %d/%d 条语句失败: %v", i+1, total, err)
			logger.Debug("SQL: %s", truncateString(stmt, 100))
			continue
		}
		success++
		if success%50 == 0 {
			logger.Info("已执行 %d/%d 条语句", success, total)
		}
	}

	logger.Success("导入完成: 成功 %d/%d 条语句", success, total)
	return nil
}

// connectDB 连接 PostgreSQL 数据库
func connectDB(host string, port int, user, password, database string) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database)
	if database == "" {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable",
			host, port, user, password)
	}

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("创建PostgreSQL连接失败: %w", err)
	}

	conn.SetConnMaxLifetime(time.Second * 30)
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("连接PostgreSQL失败: %w", err)
	}

	return conn, nil
}

// getTables 获取数据库所有表（public schema）
func getTables(conn *sql.DB) ([]string, error) {
	rows, err := conn.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE' ORDER BY table_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

// filterTables 根据 DumpOptions 过滤表
func filterTables(tables []string, opts DumpOptions) []string {
	ignoreSet := make(map[string]bool)
	for _, t := range opts.IgnoreTables {
		ignoreSet[t] = true
	}

	includeSet := make(map[string]bool)
	if len(opts.Tables) > 0 {
		for _, t := range opts.Tables {
			includeSet[t] = true
		}
	}

	var result []string
	for _, t := range tables {
		if ignoreSet[t] {
			continue
		}
		if len(includeSet) > 0 && !includeSet[t] {
			continue
		}
		result = append(result, t)
	}
	return result
}

// dumpDatabaseSQL 导出数据库为 SQL 格式
func dumpDatabaseSQL(conn *sql.DB, database string, tables []string, outputPath string, opts DumpOptions) error {
	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %w", err)
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer f.Close()

	writeSQLHeader(f, database)

	for i, table := range tables {
		logger.Info("[%d/%d] 导出表: %s", i+1, len(tables), table)

		if opts.WithSchema {
			if err := dumpTableSchemaSQL(conn, table, f); err != nil {
				logger.Warning("导出表 %s 结构失败: %v", table, err)
				continue
			}
		}

		if opts.WithData {
			if err := dumpTableDataSQL(conn, table, f); err != nil {
				logger.Warning("导出表 %s 数据失败: %v", table, err)
			}
		}
	}

	// 写入尾部
	fmt.Fprintf(f, "-- 导入完成后恢复外键检查\n")
	fmt.Fprintf(f, "SET session_replication_role = 'origin';\n")

	logger.Success("数据库导出完成: %s", outputPath)
	return nil
}

// dumpTableSQL 导出单表为 SQL 格式
func dumpTableSQL(conn *sql.DB, database, table, outputPath string, opts DumpOptions) error {
	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %w", err)
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer f.Close()

	writeSQLHeader(f, database)

	if opts.WithSchema {
		if err := dumpTableSchemaSQL(conn, table, f); err != nil {
			return fmt.Errorf("导出表结构失败: %w", err)
		}
	}

	if opts.WithData {
		if err := dumpTableDataSQL(conn, table, f); err != nil {
			return fmt.Errorf("导出表数据失败: %w", err)
		}
	}

	logger.Success("表 %s 导出完成: %s", table, outputPath)
	return nil
}

// writeSQLHeader 写入 SQL 文件头部
func writeSQLHeader(f *os.File, database string) {
	fmt.Fprintf(f, "-- opsxcli PostgreSQL Dump\n")
	fmt.Fprintf(f, "-- 数据库: %s\n", database)
	fmt.Fprintf(f, "-- 生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f, "SET client_encoding = 'UTF8';\n")
	fmt.Fprintf(f, "SET session_replication_role = 'replica'; -- 禁用触发器以加速导入\n\n")
}

// dumpTableSchemaSQL 导出表结构
func dumpTableSchemaSQL(conn *sql.DB, table string, f *os.File) error {
	// 获取列信息
	columns, err := getTableColumns(conn, table)
	if err != nil {
		return err
	}

	if len(columns) == 0 {
		return fmt.Errorf("表 %s 没有找到列信息", table)
	}

	fmt.Fprintf(f, "-- 表结构: %s\n", table)
	fmt.Fprintf(f, "DROP TABLE IF EXISTS %s CASCADE;\n", pgQuoteIdentifier(table))

	// 构建建表语句
	fmt.Fprintf(f, "CREATE TABLE %s (\n", pgQuoteIdentifier(table))

	colDefs := make([]string, 0, len(columns))
	var primaryKeys []string

	for _, col := range columns {
		def := fmt.Sprintf("  %s %s", pgQuoteIdentifier(col.Name), col.DataType)

		if col.CharacterMaxLength > 0 {
			def = fmt.Sprintf("  %s %s(%d)", pgQuoteIdentifier(col.Name), col.DataType, col.CharacterMaxLength)
		}

		if !col.IsNullable {
			def += " NOT NULL"
		}

		if col.DefaultValue.Valid && col.DefaultValue.String != "" {
			def += " DEFAULT " + col.DefaultValue.String
		}

		colDefs = append(colDefs, def)

		if col.IsPrimaryKey {
			primaryKeys = append(primaryKeys, pgQuoteIdentifier(col.Name))
		}
	}

	// 添加主键约束
	if len(primaryKeys) > 0 {
		pkDef := fmt.Sprintf("  CONSTRAINT %s PRIMARY KEY (%s)",
			pgQuoteIdentifier("pk_"+table), strings.Join(primaryKeys, ", "))
		colDefs = append(colDefs, pkDef)
	}

	fmt.Fprintf(f, "%s\n", strings.Join(colDefs, ",\n"))
	fmt.Fprintf(f, ");\n\n")

	// 获取并导出索引
	if err := dumpTableIndexes(conn, table, f); err != nil {
		logger.Warning("导出表 %s 索引失败: %v", table, err)
	}

	return nil
}

// columnInfo 列信息
type columnInfo struct {
	Name               string
	DataType           string
	CharacterMaxLength int
	IsNullable         bool
	DefaultValue       sql.NullString
	IsPrimaryKey       bool
}

// getTableColumns 获取表的列信息
func getTableColumns(conn *sql.DB, table string) ([]columnInfo, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			COALESCE(c.character_maximum_length, 0),
			c.is_nullable = 'YES',
			c.column_default,
			CASE WHEN tc.constraint_type = 'PRIMARY KEY' THEN true ELSE false END AS is_pk
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu
			ON c.table_name = kcu.table_name AND c.column_name = kcu.column_name AND kcu.table_schema = 'public'
		LEFT JOIN information_schema.table_constraints tc
			ON kcu.constraint_name = tc.constraint_name AND tc.table_schema = 'public' AND tc.constraint_type = 'PRIMARY KEY'
		WHERE c.table_name = $1 AND c.table_schema = 'public'
		ORDER BY c.ordinal_position`

	rows, err := conn.Query(query, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []columnInfo
	for rows.Next() {
		var col columnInfo
		if err := rows.Scan(&col.Name, &col.DataType, &col.CharacterMaxLength, &col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey); err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}
	return columns, nil
}

// dumpTableIndexes 导出表索引
func dumpTableIndexes(conn *sql.DB, table string, f *os.File) error {
	query := `
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE tablename = $1 AND schemaname = 'public'`

	rows, err := conn.Query(query, table)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var indexName, indexDef string
		if err := rows.Scan(&indexName, &indexDef); err != nil {
			return err
		}
		// 跳过主键索引（已在建表语句中定义）
		if strings.HasPrefix(indexName, "pk_") {
			continue
		}
		fmt.Fprintf(f, "%s;\n", indexDef)
	}
	fmt.Fprintf(f, "\n")
	return nil
}

// dumpTableDataSQL 导出表数据为 INSERT 语句
func dumpTableDataSQL(conn *sql.DB, table string, f *os.File) error {
	rows, err := conn.Query("SELECT * FROM " + pgQuoteIdentifier(table))
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	colList := make([]string, len(columns))
	for i, c := range columns {
		colList[i] = pgQuoteIdentifier(c)
	}
	colStr := strings.Join(colList, ", ")

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	rowCount := 0
	batchSize := 100
	batchValues := make([]string, 0, batchSize)

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		valStrs := make([]string, len(columns))
		for i, val := range values {
			valStrs[i] = pgFormatSQLValue(val)
		}
		batchValues = append(batchValues, "("+strings.Join(valStrs, ", ")+")")
		rowCount++

		if len(batchValues) >= batchSize {
			fmt.Fprintf(f, "INSERT INTO %s (%s) VALUES\n", pgQuoteIdentifier(table), colStr)
			fmt.Fprintf(f, "%s;\n\n", strings.Join(batchValues, ",\n"))
			batchValues = batchValues[:0]
		}
	}

	if len(batchValues) > 0 {
		fmt.Fprintf(f, "INSERT INTO %s (%s) VALUES\n", pgQuoteIdentifier(table), colStr)
		fmt.Fprintf(f, "%s;\n\n", strings.Join(batchValues, ",\n"))
	}

	if rowCount > 0 {
		fmt.Fprintf(f, "-- 表 %s: %d 行数据\n\n", table, rowCount)
	}

	return nil
}

// dumpDatabaseCSV 导出数据库为 CSV 格式
func dumpDatabaseCSV(conn *sql.DB, database string, tables []string, outputDir string, opts DumpOptions) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	for i, table := range tables {
		logger.Info("[%d/%d] 导出表: %s", i+1, len(tables), table)

		csvPath := filepath.Join(outputDir, table+".csv")
		if err := dumpTableCSV(conn, database, table, csvPath, opts); err != nil {
			logger.Warning("导出表 %s 失败: %v", table, err)
		}
	}

	logger.Success("数据库 CSV 导出完成: %s", outputDir)
	return nil
}

// dumpTableCSV 导出单表为 CSV 格式
func dumpTableCSV(conn *sql.DB, database, table, outputPath string, opts DumpOptions) error {
	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %w", err)
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer f.Close()

	// 写入 BOM
	f.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(f)
	defer writer.Flush()

	rows, err := conn.Query("SELECT * FROM " + pgQuoteIdentifier(table))
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	writer.Write(columns)

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	rowCount := 0
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		record := make([]string, len(columns))
		for i, val := range values {
			record[i] = pgFormatCSVValue(val)
		}
		writer.Write(record)
		rowCount++
	}

	logger.Success("表 %s 导出完成: %s (%d 行)", table, outputPath, rowCount)
	return nil
}

// pgFormatSQLValue 格式化 PostgreSQL SQL 值
func pgFormatSQLValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}
	switch v := val.(type) {
	case []byte:
		return "'" + pgEscapeString(string(v)) + "'"
	case string:
		return "'" + pgEscapeString(v) + "'"
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case time.Time:
		return "'" + v.Format("2006-01-02 15:04:05") + "'"
	default:
		return "'" + pgEscapeString(fmt.Sprintf("%v", v)) + "'"
	}
}

// pgFormatCSVValue 格式化 CSV 值
func pgFormatCSVValue(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "t"
		}
		return "f"
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// pgEscapeString 转义 PostgreSQL 字符串
func pgEscapeString(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// pgQuoteIdentifier 引用 PostgreSQL 标识符
func pgQuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// splitSQL 按 SQL 分号分割语句
func splitSQL(content string) []string {
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDollarQuote := false
	dollarTag := ""
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(content); i++ {
		ch := content[i]

		// 处理注释
		if !inSingleQuote && !inDollarQuote {
			if !inLineComment && !inBlockComment && ch == '-' && i+1 < len(content) && content[i+1] == '-' {
				inLineComment = true
				current.WriteByte(ch)
				continue
			}
			if inLineComment && ch == '\n' {
				inLineComment = false
				current.WriteByte(ch)
				continue
			}
			if !inLineComment && !inBlockComment && ch == '/' && i+1 < len(content) && content[i+1] == '*' {
				inBlockComment = true
				current.WriteByte(ch)
				continue
			}
			if inBlockComment && ch == '*' && i+1 < len(content) && content[i+1] == '/' {
				inBlockComment = false
				current.WriteByte(ch)
				current.WriteByte(content[i+1])
				i++
				continue
			}
		}

		if inLineComment || inBlockComment {
			current.WriteByte(ch)
			continue
		}

		// 处理 PostgreSQL 的 $$ 引用（dollar-quoting）
		if !inSingleQuote && ch == '$' && !inDollarQuote {
			// 查找匹配的 $ 标签
			tagEnd := strings.Index(content[i+1:], "$")
			if tagEnd >= 0 && tagEnd < 64 { // 合理的标签长度
				dollarTag = content[i : i+1+tagEnd+1]
				inDollarQuote = true
				current.WriteString(dollarTag)
				i += len(dollarTag) - 1
				continue
			}
		}
		if inDollarQuote && ch == '$' {
			// 检查是否匹配结束标签
			if i+len(dollarTag) <= len(content) && content[i:i+len(dollarTag)] == dollarTag {
				current.WriteString(dollarTag)
				i += len(dollarTag) - 1
				inDollarQuote = false
				dollarTag = ""
				continue
			}
		}

		// 处理单引号（PostgreSQL 用 '' 转义）
		if ch == '\'' && !inDollarQuote {
			if inSingleQuote && i+1 < len(content) && content[i+1] == '\'' {
				// 转义引号 ''
				current.WriteString("''")
				i++
				continue
			}
			inSingleQuote = !inSingleQuote
			current.WriteByte(ch)
			continue
		}

		// 分号分割
		if ch == ';' && !inSingleQuote && !inDollarQuote {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
