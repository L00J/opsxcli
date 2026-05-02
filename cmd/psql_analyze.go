package cmd

// psql_analyze.go — PostgreSQL analyze 子命令定义
//
// 提供 PostgreSQL 性能分析子命令，支持：
//   - --active: 活跃查询分析
//   - --locks:  锁等待分析
//   - --all:    全部分析（默认）

import (
	"fmt"
	"os"
	"time"

	"opsxcli/plugins/postgres"

	"github.com/spf13/cobra"
)

// psqlDBFlags 从 cobra 命令中提取 PostgreSQL 连接参数
func psqlDBFlags(cmd *cobra.Command) DatabaseFlags {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	user, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	database, _ := cmd.Flags().GetString("database")
	return DatabaseFlags{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: database,
	}
}

// newPsqlAnalyzeCmd 创建 psql analyze 子命令
func newPsqlAnalyzeCmd() *cobra.Command {
	var (
		showActive    bool // 活跃查询分析
		showLocks     bool // 锁等待分析
		showAll       bool // 全部分析
		longQueryTime int  // 长查询阈值（秒）
		threshold     time.Duration
	)

	analyzeCmd := &cobra.Command{
		Use:   "analyze",
		Short: "PostgreSQL性能分析与诊断",
		Long: `分析PostgreSQL性能指标，检测活跃查询、锁等待，生成性能报告

支持的分析模式：
  - --active:  活跃查询分析（查询 pg_stat_activity）
  - --locks:   锁等待分析（查询 pg_locks）
  - --all:     全部分析（默认，包含上述所有分析）

Examples:
  # 全部分析（默认）
  opsxcli psql analyze -h 192.168.1.100 -U postgres -W

  # 仅分析活跃查询
  opsxcli psql analyze -h localhost -U postgres -W --active

  # 仅分析锁等待
  opsxcli psql analyze -h localhost -U postgres -W --locks

  # 指定长查询阈值
  opsxcli psql analyze -h localhost -U postgres -W --long-query-time 30

  # 使用环境变量密码
  export PGPASSWORD=***; opsxcli psql analyze -h localhost -U postgres`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			threshold = time.Duration(longQueryTime) * time.Second

			// 获取数据库连接参数
			flags := psqlDBFlags(cmd)

			// PostgreSQL 密码处理（按优先级）：
			// 1. password 为 "ASK"：用户使用了 -W 但没有值，提示输入密码
			// 2. password 有值：使用命令行提供的密码
			// 3. PGPASSWORD 环境变量（官方标准）
			// 4. 无密码（尝试连接）
			switch {
			case flags.Password == "ASK":
				flags.Password = PromptPassword("Password")
			case flags.Password == "":
				flags.Password = os.Getenv("PGPASSWORD")
			}

			// 判断分析模式
			switch {
			case showAll, !showActive && !showLocks:
				// 默认或 --all：执行完整性能报告
				return postgres.RunPerformanceReport(
					flags.Host, flags.Port, flags.User, flags.Password, flags.Database, threshold,
				)
			case showActive && showLocks:
				// 同时指定两者：也是完整报告
				return postgres.RunPerformanceReport(
					flags.Host, flags.Port, flags.User, flags.Password, flags.Database, threshold,
				)
			case showActive:
				return postgres.RunActiveQueryAnalysis(
					flags.Host, flags.Port, flags.User, flags.Password, flags.Database, threshold,
				)
			case showLocks:
				return postgres.RunLockWaitAnalysis(
					flags.Host, flags.Port, flags.User, flags.Password, flags.Database,
				)
			default:
				// 不可达，仅作为安全保障
				return fmt.Errorf("未指定分析模式")
			}
		},
	}

	// PostgreSQL 风格的数据库连接参数
	analyzeCmd.Flags().StringP("host", "h", "127.0.0.1", "PostgreSQL主机地址")
	analyzeCmd.Flags().IntP("port", "P", 5432, "PostgreSQL端口")
	analyzeCmd.Flags().StringP("user", "U", "postgres", "PostgreSQL用户名")
	analyzeCmd.Flags().StringP("password", "W", "", "PostgreSQL密码（或使用环境变量 PGPASSWORD）")
	analyzeCmd.Flags().Lookup("password").NoOptDefVal = "ASK"
	analyzeCmd.Flags().StringP("database", "d", "", "数据库名称")

	// analyze 特有参数
	analyzeCmd.Flags().BoolVar(&showActive, "active", false, "分析活跃查询（pg_stat_activity）")
	analyzeCmd.Flags().BoolVar(&showLocks, "locks", false, "分析锁等待（pg_locks）")
	analyzeCmd.Flags().BoolVar(&showAll, "all", false, "全部分析（默认行为）")
	analyzeCmd.Flags().IntVar(&longQueryTime, "long-query-time", 5, "长查询阈值（秒）")
	analyzeCmd.Flags().Bool("help", false, "help for analyze")

	return analyzeCmd
}
