package cmd

import (
	"os"
	"strings"

	"opsxcli/plugins/postgres"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("psql", "数据库", "PostgreSQL数据库操作", NewPsqlCmd)
}

// NewPsqlCmd 创建PostgreSQL命令（主命令名为psql）
func NewPsqlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "psql",
		Short:   "PostgreSQL数据库操作工具",
		Aliases: []string{"postgres"},
		Long: `PostgreSQL工具支持以下功能：
  - 交互式PostgreSQL shell（支持自动补全、历史记录）
  - 执行SQL语句
  - 查询数据库和表
  - 支持psql特殊命令（\l, \dt, \d等）
  - 数据库导入导出（dump/restore）

Examples:
  # 进入交互式PostgreSQL shell（提示输入密码）
  opsxcli psql -h localhost -P 5432 -U postgres -W

  # 使用环境变量密码（官方标准）
  export PGPASSWORD=***
  opsxcli psql -h localhost -P 5432 -U postgres

  # 执行SQL语句
  opsxcli psql -h localhost -U postgres -d testdb -c "SELECT version();"

  # 连接指定数据库
  opsxcli psql -h 192.168.1.100 -P 5432 -U admin -d production

  # 快速查询
  opsxcli psql -h localhost -U postgres -c "\l"  # 列出所有数据库
  opsxcli psql -h localhost -U postgres -d mydb -c "\dt"  # 列出表

  # 导出数据库
  opsxcli psql dump -h localhost -U postgres -W -d mydb -o dump.sql
  opsxcli psql dump -h localhost -U postgres -W -d mydb --format csv -o ./csv_output/

  # 导入数据库
  opsxcli psql restore -h localhost -U postgres -W -d mydb -i dump.sql`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := getSubcommandDBFlags(cmd)

			// PostgreSQL 密码处理（按优先级）：
			// 1. password 为 "ASK"：用户使用了 -W 但没有值，提示输入密码
			// 2. password 有值：使用命令行提供的密码
			// 3. PGPASSWORD 环境变量（官方标准）
			// 4. 无密码（尝试连接）

			if flags.Password == "ASK" {
				// -W 被设置但没有值，提示输入密码
				flags.Password = PromptPassword("Password")
			} else if flags.Password == "" {
				// 尝试从环境变量读取（官方标准）
				flags.Password = os.Getenv("PGPASSWORD")
			}

			execute, _ := cmd.Flags().GetString("execute")
			if execute != "" {
				return postgres.Execute(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, execute)
			}

			return postgres.Interactive(flags.Host, flags.Port, flags.User, flags.Password, flags.Database)
		},
		DisableFlagParsing: false,
	}

	// PostgreSQL 使用不同的 flag shorthand（-U, -W, -c）
	cmd.Flags().StringP("host", "h", "127.0.0.1", "PostgreSQL主机地址")
	cmd.Flags().IntP("port", "P", 5432, "PostgreSQL端口")
	cmd.Flags().StringP("user", "U", "postgres", "PostgreSQL用户名")

	// 密码 flag 支持可选参数
	cmd.Flags().StringP("password", "W", "", "PostgreSQL密码（或使用环境变量 PGPASSWORD）")
	cmd.Flags().Lookup("password").NoOptDefVal = "ASK" // 当 -W 不带参数时的默认值

	cmd.Flags().StringP("database", "d", "", "数据库名称")
	cmd.Flags().StringP("execute", "c", "", "执行SQL语句")
	cmd.Flags().Bool("help", false, "help for postgres")

	// 添加子命令
	cmd.AddCommand(newPsqlDumpCmd())
	cmd.AddCommand(newPsqlRestoreCmd())
	cmd.AddCommand(newPsqlAnalyzeCmd())

	return cmd
}

// newPsqlDumpCmd 创建 psql dump 子命令
func newPsqlDumpCmd() *cobra.Command {
	var (
		format       string
		tables       string
		ignoreTables string
		noData       bool
		noSchema     bool
		outputPath   string
	)

	dumpCmd := &cobra.Command{
		Use:   "dump",
		Short: "导出PostgreSQL数据库",
		Long: `导出PostgreSQL数据库到SQL或CSV文件（纯Go实现，不依赖pg_dump）

支持导出表结构和数据，可选择SQL或CSV格式。

Examples:
  # 导出整个数据库
  opsxcli psql dump -h localhost -U postgres -W -d mydb -o dump.sql

  # 导出指定表
  opsxcli psql dump -h localhost -U postgres -W -d mydb --tables users,orders -o dump.sql

  # 只导出结构
  opsxcli psql dump -h localhost -U postgres -W -d mydb --no-data -o schema.sql

  # 只导出数据
  opsxcli psql dump -h localhost -U postgres -W -d mydb --no-schema -o data.sql

  # 导出为CSV格式
  opsxcli psql dump -h localhost -U postgres -W -d mydb --format csv -o ./csv_output/`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := getPsqlDBFlags(cmd)

			if flags.Password == "ASK" {
				flags.Password = PromptPassword("Password")
			} else if flags.Password == "" {
				flags.Password = os.Getenv("PGPASSWORD")
			}

			if flags.Database == "" {
				return cmd.Help()
			}
			if outputPath == "" {
				if format == "csv" {
					outputPath = flags.Database + "/"
				} else {
					outputPath = flags.Database + ".sql"
				}
			}

			opts := postgres.DefaultDumpOptions()
			opts.Format = format
			opts.WithData = !noData
			opts.WithSchema = !noSchema
			if tables != "" {
				opts.Tables = strings.Split(tables, ",")
			}
			if ignoreTables != "" {
				opts.IgnoreTables = strings.Split(ignoreTables, ",")
			}

			return postgres.DumpDatabase(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, outputPath, opts)
		},
	}

	// PostgreSQL 风格的数据库连接参数
	dumpCmd.Flags().StringP("host", "h", "127.0.0.1", "PostgreSQL主机地址")
	dumpCmd.Flags().IntP("port", "P", 5432, "PostgreSQL端口")
	dumpCmd.Flags().StringP("user", "U", "postgres", "PostgreSQL用户名")
	dumpCmd.Flags().StringP("password", "W", "", "PostgreSQL密码")
	dumpCmd.Flags().Lookup("password").NoOptDefVal = "ASK"
	dumpCmd.Flags().StringP("database", "d", "", "数据库名称")

	// dump 特有参数
	dumpCmd.Flags().StringVarP(&outputPath, "output", "o", "", "输出文件路径（SQL）或目录（CSV）")
	dumpCmd.Flags().StringVar(&format, "format", "sql", "输出格式: sql 或 csv")
	dumpCmd.Flags().StringVar(&tables, "tables", "", "指定导出的表，逗号分隔")
	dumpCmd.Flags().StringVar(&ignoreTables, "ignore-tables", "", "忽略的表，逗号分隔")
	dumpCmd.Flags().BoolVar(&noData, "no-data", false, "不包含数据")
	dumpCmd.Flags().BoolVar(&noSchema, "no-schema", false, "不包含表结构")
	dumpCmd.Flags().Bool("help", false, "help for dump")

	return dumpCmd
}

// newPsqlRestoreCmd 创建 psql restore 子命令
func newPsqlRestoreCmd() *cobra.Command {
	var inputPath string

	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "导入PostgreSQL数据库",
		Long: `从SQL文件导入数据到PostgreSQL数据库

Examples:
  # 从SQL文件导入
  opsxcli psql restore -h localhost -U postgres -W -d mydb -i dump.sql`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := getPsqlDBFlags(cmd)

			if flags.Password == "ASK" {
				flags.Password = PromptPassword("Password")
			} else if flags.Password == "" {
				flags.Password = os.Getenv("PGPASSWORD")
			}

			if flags.Database == "" || inputPath == "" {
				return cmd.Help()
			}

			return postgres.RestoreDatabase(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, inputPath)
		},
	}

	// PostgreSQL 风格的数据库连接参数
	restoreCmd.Flags().StringP("host", "h", "127.0.0.1", "PostgreSQL主机地址")
	restoreCmd.Flags().IntP("port", "P", 5432, "PostgreSQL端口")
	restoreCmd.Flags().StringP("user", "U", "postgres", "PostgreSQL用户名")
	restoreCmd.Flags().StringP("password", "W", "", "PostgreSQL密码")
	restoreCmd.Flags().Lookup("password").NoOptDefVal = "ASK"
	restoreCmd.Flags().StringP("database", "d", "", "数据库名称")

	restoreCmd.Flags().StringVarP(&inputPath, "input", "i", "", "输入SQL文件路径")
	restoreCmd.Flags().Bool("help", false, "help for restore")

	return restoreCmd
}

// getPsqlDBFlags 从 psql 子命令获取数据库连接参数
func getPsqlDBFlags(cmd *cobra.Command) DatabaseFlags {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	user, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	database, _ := cmd.Flags().GetString("database")

	if password == "ASK" {
		if extracted := FetchAndClearDatabasePassword(); extracted != "" {
			password = extracted
		}
	}

	return DatabaseFlags{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: database,
	}
}
