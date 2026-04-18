package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	_ "github.com/lib/pq"

	"opsxcli/internal/db"
	"opsxcli/internal/logger"
)

// Execute 执行SQL语句
func Execute(host string, port int, user, password, database, sqlStr string) error {
	// 应用默认配置
	host, port, user = db.ApplyDefaults(host, port, user, "postgres")

	// 构建DSN
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database)
	if database == "" {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable",
			host, port, user, password)
	}

	// 连接数据库
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("创建PostgreSQL连接失败: %v", err)
	}
	defer db.Close()

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("连接PostgreSQL失败: %v", err)
	}

	// 执行SQL
	rows, err := db.Query(sqlStr)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// 读取所有数据
	allRows := make([][]string, 0)
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

		// 将值转换为字符串
		row := make([]string, len(columns))
		for i, val := range values {
			row[i] = formatValue(val)
		}
		allRows = append(allRows, row)
		rowCount++
	}

	// 打印表格
	printTable(columns, allRows)

	fmt.Printf("\n%d row(s) in set\n", rowCount)
	return nil
}

// Interactive 交互式PostgreSQL shell
func Interactive(host string, port int, user, password, database string) error {
	// 应用默认配置
	host, port, user = db.ApplyDefaults(host, port, user, "postgres")

	// 构建DSN
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database)
	if database == "" {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable",
			host, port, user, password)
	}

	// 连接数据库
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("创建PostgreSQL连接失败: %v", err)
	}
	defer db.Close()

	// 验证连接（设置超时）
	db.SetConnMaxLifetime(time.Second * 5)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("连接PostgreSQL失败: %v (请检查网络连接、用户名、密码和服务器状态)", err)
	}

	logger.Success("已连接到 PostgreSQL (%s:%d)", host, port)

	// 创建readline实例
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "postgres=# ",
		HistoryFile:     os.Getenv("HOME") + "/.opsxcli/postgres_history",
		AutoComplete:    getCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	// 交互式循环
	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}

		line = trimSQL(line)
		if line == "" {
			continue
		}

		// 处理特殊命令
		if shouldExit := handleSpecialCommand(db, line); shouldExit {
			break // 退出循环
		}
	}

	fmt.Println("Bye")
	return nil
}

// 辅助函数
func trimSQL(sql string) string {
	sql = strings.TrimSpace(sql)
	// 移除末尾的分号
	if strings.HasSuffix(sql, ";") {
		sql = sql[:len(sql)-1]
	}
	return strings.TrimSpace(sql)
}

func handleSpecialCommand(db *sql.DB, cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	cmdLower := strings.ToLower(cmd)

	// 处理退出命令
	if cmdLower == "exit" || cmdLower == "quit" || cmdLower == "\\q" {
		return true
	}

	// 处理帮助命令
	if cmdLower == "help" || cmdLower == "\\h" || cmdLower == "\\?" {
		printHelp()
		return false
	}

	// 处理 PostgreSQL 特殊命令
	if strings.HasPrefix(cmd, "\\") {
		handlePsqlCommand(db, cmd)
		return false
	}

	// 执行SQL
	if err := executeSQL(db, cmd); err != nil {
		logger.Error("%v", err)
	}
	return false
}

// handlePsqlCommand 处理 psql 特殊命令
func handlePsqlCommand(db *sql.DB, cmd string) {
	cmd = strings.TrimSpace(cmd)
	cmdLower := strings.ToLower(cmd)

	switch {
	case cmdLower == "\\l" || cmdLower == "\\list":
		// 列出所有数据库
		listDatabases(db)
	case cmdLower == "\\dt" || cmdLower == "\\d":
		// 列出当前数据库的所有表
		listTables(db)
	case strings.HasPrefix(cmdLower, "\\d "):
		// 描述表结构
		tableName := strings.TrimSpace(cmd[2:])
		describeTable(db, tableName)
	case strings.HasPrefix(cmdLower, "\\c ") || strings.HasPrefix(cmdLower, "\\connect "):
		// 连接数据库（这里只是提示，实际需要重新连接）
		dbName := strings.TrimSpace(cmd[strings.Index(cmd, " ")+1:])
		fmt.Printf("注意: 需要重新连接数据库 %s\n", dbName)
		fmt.Println("请使用新的连接参数重新运行命令")
	default:
		fmt.Printf("未知命令: %s\n", cmd)
		fmt.Println("输入 \\? 查看帮助")
	}
}

// listDatabases 列出所有数据库
func listDatabases(db *sql.DB) {
	query := `SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname`
	if err := executeSQL(db, query); err != nil {
		logger.Error("%v", err)
	}
}

// listTables 列出当前数据库的所有表
func listTables(db *sql.DB) {
	query := `SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name`
	if err := executeSQL(db, query); err != nil {
		logger.Error("%v", err)
	}
}

// describeTable 描述表结构
func describeTable(db *sql.DB, tableName string) {
	query := fmt.Sprintf(`
		SELECT 
			column_name,
			data_type,
			character_maximum_length,
			is_nullable,
			column_default
		FROM information_schema.columns 
		WHERE table_name = '%s' 
		ORDER BY ordinal_position`,
		tableName)
	if err := executeSQL(db, query); err != nil {
		logger.Error("%v", err)
	}
}

func executeSQL(db *sql.DB, sqlStr string) error {
	rows, err := db.Query(sqlStr)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// 读取所有数据
	allRows := make([][]string, 0)
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

		// 将值转换为字符串
		row := make([]string, len(columns))
		for i, val := range values {
			row[i] = formatValue(val)
		}
		allRows = append(allRows, row)
		rowCount++
	}

	// 打印表格
	printTable(columns, allRows)

	fmt.Printf("\n%d row(s) in set\n", rowCount)
	return nil
}

// formatValue 格式化值为字符串
func formatValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}
	switch v := val.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "t"
		}
		return "f"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// printTable 打印表格格式的数据（类似标准 psql 客户端）
func printTable(columns []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}

	// 计算每列的最大宽度
	colWidths := make([]int, len(columns))
	for i, col := range columns {
		colWidths[i] = len(col)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// 打印表格上边框
	printTableBorder(colWidths)

	// 打印表头
	printTableRow(columns, colWidths)

	// 打印表头和数据之间的分隔线
	printTableBorder(colWidths)

	// 打印数据行
	for _, row := range rows {
		printTableRow(row, colWidths)
	}

	// 打印表格下边框
	printTableBorder(colWidths)
}

// printTableBorder 打印表格边框
func printTableBorder(colWidths []int) {
	fmt.Print("+")
	for _, width := range colWidths {
		fmt.Print(strings.Repeat("-", width+2))
		fmt.Print("+")
	}
	fmt.Println()
}

// printTableRow 打印表格行
func printTableRow(row []string, colWidths []int) {
	fmt.Print("|")
	for i, cell := range row {
		fmt.Printf(" %-*s |", colWidths[i], cell)
	}
	fmt.Println()
}

func getCompleter() *readline.PrefixCompleter {
	// SQL关键字自动补全
	return readline.NewPrefixCompleter(
		readline.PcItem("SELECT"),
		readline.PcItem("INSERT"),
		readline.PcItem("UPDATE"),
		readline.PcItem("DELETE"),
		readline.PcItem("CREATE"),
		readline.PcItem("DROP"),
		readline.PcItem("ALTER"),
		readline.PcItem("\\l"),  // 列出数据库
		readline.PcItem("\\dt"), // 列出表
		readline.PcItem("\\d"),  // 描述表
		readline.PcItem("\\c"),  // 连接数据库
		readline.PcItem("\\q"),  // 退出
	)
}

func printHelp() {
	fmt.Println("PostgreSQL命令帮助:")
	fmt.Println("  help; 或 \\h 或 \\? - 显示帮助")
	fmt.Println("  exit; 或 \\q      - 退出")
	fmt.Println("  \\l                - 列出所有数据库")
	fmt.Println("  \\dt               - 列出当前数据库的所有表")
	fmt.Println("  \\d [table]        - 描述表结构")
	fmt.Println("  \\c [database]     - 连接到数据库")
	fmt.Println("  输入SQL语句，以 ; 结尾执行")
}
