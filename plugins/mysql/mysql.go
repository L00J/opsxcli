package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	_ "github.com/go-sql-driver/mysql"

	"opsxcli/internal/db"
	"opsxcli/internal/logger"
)

// Execute 执行SQL语句
func Execute(host string, port int, user, password, database, sqlStr string) error {
	// 应用默认配置
	host, port, user = db.ApplyDefaults(host, port, user, "mysql")

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, database)
	if database == "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, host, port)
	}

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("创建MySQL连接失败: %v", err)
	}
	defer db.Close()

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("连接MySQL失败: %v", err)
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

// Interactive 交互式MySQL shell
func Interactive(host string, port int, user, password, database string) error {
	// 应用默认配置
	host, port, user = db.ApplyDefaults(host, port, user, "mysql")

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, database)
	if database == "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, host, port)
	}

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("创建MySQL连接失败: %v", err)
	}
	defer db.Close()

	// 验证连接（设置超时）
	db.SetConnMaxLifetime(time.Second * 5)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("连接MySQL失败: %v (请检查网络连接、用户名、密码和服务器状态)", err)
	}

	logger.Success("已连接到 MySQL (%s:%d)", host, port)

	// 创建readline实例
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "mysql> ",
		HistoryFile:     os.Getenv("HOME") + "/.opsxcli/mysql_history",
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
		if shouldExit := handleSpecialCommand(line); shouldExit {
			break // 退出循环
		}

		// 执行SQL
		if err := executeSQL(db, line); err != nil {
			logger.Error("%v", err)
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
	if strings.HasSuffix(sql, "\\g") {
		sql = sql[:len(sql)-2]
	}
	return strings.TrimSpace(sql)
}

func handleSpecialCommand(cmd string) bool {
	cmd = strings.ToLower(cmd)
	switch cmd {
	case "exit", "quit", "\\q":
		return true // 返回 true 表示需要退出
	case "help", "\\h":
		printHelp()
		return false // 返回 false 表示继续执行
	}
	return false
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
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// printTable 打印表格格式的数据（类似标准 MySQL 客户端）
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
		readline.PcItem("SHOW"),
		readline.PcItem("USE"),
		readline.PcItem("EXIT"),
	)
}

func printHelp() {
	fmt.Println("MySQL命令帮助:")
	fmt.Println("  help; 或 \\h  - 显示帮助")
	fmt.Println("  exit; 或 \\q  - 退出")
	fmt.Println("  输入SQL语句，以 ; 或 \\g 结尾执行")
}
