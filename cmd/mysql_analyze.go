package cmd

import (
	"opsxcli/plugins/mysql"

	"github.com/spf13/cobra"
)

func init() {
	// analyze 子命令通过 mysql 父命令注册，见 mysql.go 中的 AddCommand
}

// newMySQLAnalyzeCmd 创建 mysql analyze 子命令
// 用于 MySQL 慢查询分析和性能诊断
func newMySQLAnalyzeCmd() *cobra.Command {
	var (
		longQueryTime int // 长查询阈值（秒）
		showIndexes   bool
		showProcess   bool
		showLocks     bool
		outputFormat  string
	)

	analyzeCmd := &cobra.Command{
		Use:   "analyze",
		Short: "MySQL性能分析与慢查询诊断",
		Long: `分析MySQL性能指标，检测慢查询，提供索引建议

不需要连接数据库即可使用的纯分析功能：
  - SQL规范化与分类
  - 慢查询报告格式化
  - 索引建议生成

需要连接数据库的完整分析（--host 指定数据库地址）：
  - 慢查询分析（SHOW PROCESSLIST + SHOW STATUS）
  - InnoDB 缓冲池命中率检测
  - 自动索引建议（基于 information_schema.STATISTICS）
  - InnoDB 锁等待分析（SHOW ENGINE INNODB STATUS）

Examples:
  # 分析SQL语句类型
  opsxcli mysql analyze --sql "SELECT * FROM users WHERE email = 'test@example.com'"

  # 规范化SQL文本
  opsxcli mysql analyze --normalize "SELECT  *   FROM   users"

  # 连接数据库进行完整性能分析
  opsxcli mysql analyze -h 192.168.1.100 -u root -p -d mydb

  # 仅分析慢查询（阈值30秒）
  opsxcli mysql analyze -h localhost -u root -p --slow-query --long-query-time 30

  # 仅分析索引
  opsxcli mysql analyze -h localhost -u root -p --indexes

  # 分析 InnoDB 锁等待
  opsxcli mysql analyze -h localhost -u root -p --locks`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 纯文本分析模式（无需数据库连接）
			sqlInput, _ := cmd.Flags().GetString("sql")
			normalizeInput, _ := cmd.Flags().GetString("normalize")

			if sqlInput != "" {
				// SQL 分类分析
				queryType := mysql.ClassifyQueryType(sqlInput)
				normalized := mysql.NormalizeSQL(sqlInput)
				cmd.Printf("查询类型: %s\n", queryType)
				cmd.Printf("规范化SQL: %s\n", normalized)
				return nil
			}

			if normalizeInput != "" {
				// SQL 规范化
				normalized := mysql.NormalizeSQL(normalizeInput)
				cmd.Printf("规范化结果: %s\n", normalized)
				return nil
			}

			// 数据库连接分析模式
			flags := getSubcommandDBFlags(cmd)

			if flags.Password == "ASK" {
				flags.Password = PromptPassword("Enter password")
			}

			// 必须指定 host 才能连接数据库
			if flags.Host == "" {
				cmd.Println("请指定数据库连接参数: -h <host> -u <user> -p")
				cmd.Println("或使用纯分析模式: --sql <sql> 或 --normalize <sql>")
				return nil
			}

			// 按功能路由
			slowQueryOnly, _ := cmd.Flags().GetBool("slow-query")
			switch {
			case showLocks:
				return mysql.RunLockAnalysis(flags.Host, flags.Port, flags.User, flags.Password, flags.Database)
			case showIndexes && !showProcess:
				return mysql.RunIndexAnalysis(flags.Host, flags.Port, flags.User, flags.Password, flags.Database)
			case slowQueryOnly:
				return mysql.RunSlowQueryAnalysis(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, longQueryTime)
			default:
				// 完整分析（默认包含进程列表）
				return mysql.RunFullAnalysis(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, longQueryTime, showIndexes, true)
			}
		},
	}

	// 数据库连接参数
	addSubcommandDBFlags(analyzeCmd, DatabaseDefaults{
		Host: "127.0.0.1",
		Port: 3306,
		User: "root",
	})

	// analyze 特有参数
	analyzeCmd.Flags().String("sql", "", "分析指定SQL语句的类型")
	analyzeCmd.Flags().String("normalize", "", "规范化SQL文本")
	analyzeCmd.Flags().IntVar(&longQueryTime, "long-query-time", 10, "长查询阈值（秒）")
	analyzeCmd.Flags().BoolVar(&showIndexes, "indexes", false, "包含索引分析")
	analyzeCmd.Flags().BoolVar(&showProcess, "process", false, "包含进程列表分析")
	analyzeCmd.Flags().BoolVar(&showLocks, "locks", false, "分析InnoDB锁等待")
	analyzeCmd.Flags().Bool("slow-query", false, "仅分析慢查询")
	analyzeCmd.Flags().StringVar(&outputFormat, "format", "text", "输出格式: text/json")

	return analyzeCmd
}
