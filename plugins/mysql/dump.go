package mysql

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

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

// DumpDatabase 导出 MySQL 数据库
// 使用 information_schema 获取表结构，纯 Go 实现
func DumpDatabase(host string, port int, user, password, database, outputPath string, opts DumpOptions) error {
	// 应用默认配置
	host, port, user = db.ApplyDefaults(host, port, user, "mysql")

	// 连接数据库
	conn, err := connectDB(host, port, user, password, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 获取所有表
	tables, err := getTables(conn, database)
	if err != nil {
		return fmt.Errorf("获取表列表失败: %w", err)
	}

	// 过滤表
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
	host, port, user = db.ApplyDefaults(host, port, user, "mysql")

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

// RestoreDatabase 导入 MySQL 数据库
func RestoreDatabase(host string, port int, user, password, database, inputPath string) error {
	host, port, user = db.ApplyDefaults(host, port, user, "mysql")

	// 读取 SQL 文件
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

	// 按分号分割并执行 SQL 语句
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

// connectDB 连接 MySQL 数据库
func connectDB(host string, port int, user, password, database string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true", user, password, host, port, database)
	if database == "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=true", user, password, host, port)
	}

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("创建MySQL连接失败: %w", err)
	}

	conn.SetConnMaxLifetime(time.Second * 30)
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("连接MySQL失败: %w", err)
	}

	return conn, nil
}

// getTables 获取数据库所有表
func getTables(conn *sql.DB, database string) ([]string, error) {
	rows, err := conn.Query("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME", database)
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
	// 确保输出目录存在
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

	// 写入文件头
	writeSQLHeader(f, database)

	// 导出每张表
	for i, table := range tables {
		logger.Info("[%d/%d] 导出表: %s", i+1, len(tables), table)

		if opts.WithSchema {
			if err := dumpTableSchemaSQL(conn, database, table, f); err != nil {
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
		if err := dumpTableSchemaSQL(conn, database, table, f); err != nil {
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
	fmt.Fprintf(f, "-- opsxcli MySQL Dump\n")
	fmt.Fprintf(f, "-- 数据库: %s\n", database)
	fmt.Fprintf(f, "-- 生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f, "SET NAMES utf8mb4;\n")
	fmt.Fprintf(f, "SET FOREIGN_KEY_CHECKS = 0;\n\n")
}

// dumpTableSchemaSQL 导出表结构
func dumpTableSchemaSQL(conn *sql.DB, database, table string, f *os.File) error {
	// 使用 SHOW CREATE TABLE 获取建表语句
	var tableName, createSQL string
	err := conn.QueryRow("SHOW CREATE TABLE "+quoteIdentifier(table)).Scan(&tableName, &createSQL)
	if err != nil {
		return fmt.Errorf("获取建表语句失败: %w", err)
	}

	fmt.Fprintf(f, "-- 表结构: %s\n", table)
	fmt.Fprintf(f, "DROP TABLE IF EXISTS %s;\n", quoteIdentifier(table))
	fmt.Fprintf(f, "%s;\n\n", createSQL)

	return nil
}

// dumpTableDataSQL 导出表数据为 INSERT 语句
func dumpTableDataSQL(conn *sql.DB, table string, f *os.File) error {
	rows, err := conn.Query("SELECT * FROM " + quoteIdentifier(table))
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
		colList[i] = quoteIdentifier(c)
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
			valStrs[i] = formatSQLValue(val)
		}
		batchValues = append(batchValues, "("+strings.Join(valStrs, ", ")+")")
		rowCount++

		if len(batchValues) >= batchSize {
			fmt.Fprintf(f, "INSERT INTO %s (%s) VALUES\n", quoteIdentifier(table), colStr)
			fmt.Fprintf(f, "%s;\n\n", strings.Join(batchValues, ",\n"))
			batchValues = batchValues[:0]
		}
	}

	// 写入剩余数据
	if len(batchValues) > 0 {
		fmt.Fprintf(f, "INSERT INTO %s (%s) VALUES\n", quoteIdentifier(table), colStr)
		fmt.Fprintf(f, "%s;\n\n", strings.Join(batchValues, ",\n"))
	}

	if rowCount > 0 {
		fmt.Fprintf(f, "-- 表 %s: %d 行数据\n\n", table, rowCount)
	}

	return nil
}

// dumpDatabaseCSV 导出数据库为 CSV 格式（每张表一个文件）
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

	// 写入 BOM（UTF-8 with BOM，方便 Excel 打开）
	f.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(f)
	defer writer.Flush()

	rows, err := conn.Query("SELECT * FROM " + quoteIdentifier(table))
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// 写入表头
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
			record[i] = formatCSVValue(val)
		}
		writer.Write(record)
		rowCount++
	}

	logger.Success("表 %s 导出完成: %s (%d 行)", table, outputPath, rowCount)
	return nil
}

// formatSQLValue 格式化 SQL 值
func formatSQLValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}
	switch v := val.(type) {
	case []byte:
		// 处理二进制数据
		return "'" + escapeString(string(v)) + "'"
	case string:
		return "'" + escapeString(v) + "'"
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	case time.Time:
		return "'" + v.Format("2006-01-02 15:04:05") + "'"
	default:
		return "'" + escapeString(fmt.Sprintf("%v", v)) + "'"
	}
}

// formatCSVValue 格式化 CSV 值
func formatCSVValue(val interface{}) string {
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
			return "1"
		}
		return "0"
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// escapeString 转义 SQL 字符串中的特殊字符
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\x00", "\\0")
	return s
}

// quoteIdentifier 引用标识符
func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// splitSQL 按 SQL 分号分割语句（处理字符串和注释中的分号）
func splitSQL(content string) []string {
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(content); i++ {
		ch := content[i]

		// 处理注释
		if !inSingleQuote && !inDoubleQuote {
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

		// 处理引号
		if ch == '\'' && !inDoubleQuote {
			if inSingleQuote && i > 0 && content[i-1] == '\\' {
				// 转义的单引号
				current.WriteByte(ch)
				continue
			}
			inSingleQuote = !inSingleQuote
			current.WriteByte(ch)
			continue
		}
		if ch == '"' && !inSingleQuote {
			if inDoubleQuote && i > 0 && content[i-1] == '\\' {
				current.WriteByte(ch)
				continue
			}
			inDoubleQuote = !inDoubleQuote
			current.WriteByte(ch)
			continue
		}

		// 分号分割
		if ch == ';' && !inSingleQuote && !inDoubleQuote {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	// 处理最后一条语句
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
