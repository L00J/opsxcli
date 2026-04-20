package cmd

import (
	"strings"

	"opsxcli/plugins/mysql"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("mysql", "数据库", "MySQL数据库操作", NewMySQLCmd)
}

// NewMySQLCmd 创建MySQL命令
func NewMySQLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mysql",
		Short: "MySQL数据库操作工具",
		Long: `MySQL工具支持以下功能：
  - 交互式MySQL shell（支持自动补全、历史记录）
  - 执行SQL语句
  - 查询数据库和表
  - 数据库导入导出（dump/restore）

Examples:
  # 进入交互式MySQL shell（提示输入密码）
  opsxcli mysql -h localhost -P 3306 -u root -p

  # 直接指定密码（不安全，不推荐）
  opsxcli mysql -h localhost -P 3306 -u root -pYourPassword

  # 执行SQL语句
  opsxcli mysql -h localhost -u root -p -d testdb -e "SELECT * FROM users LIMIT 10"

  # 连接指定数据库
  opsxcli mysql -h 192.168.1.100 -P 3306 -u admin -p -d production

  # 无密码连接（不提供 -p 参数）
  opsxcli mysql -h localhost -u root -e "SHOW DATABASES;"

  # 导出数据库
  opsxcli mysql dump -h localhost -u root -p -d mydb -o dump.sql
  opsxcli mysql dump -h localhost -u root -p -d mydb --format csv -o ./csv_output/

  # 导入数据库
  opsxcli mysql restore -h localhost -u root -p -d mydb -i dump.sql`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := GetDatabaseFlags(cmd)

			if flags.Password == "ASK" {
				flags.Password = PromptPassword("Enter password")
			}

			if flags.Execute != "" {
				return mysql.Execute(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, flags.Execute)
			}

			return mysql.Interactive(flags.Host, flags.Port, flags.User, flags.Password, flags.Database)
		},
		DisableFlagParsing: false,
	}

	// 添加数据库通用参数
	AddDatabaseFlags(cmd, DatabaseDefaults{
		Host: "127.0.0.1",
		Port: 3306,
		User: "root",
	})

	// 添加子命令
	cmd.AddCommand(newMySQLDumpCmd())
	cmd.AddCommand(newMySQLRestoreCmd())

	return cmd
}

// newMySQLDumpCmd 创建 mysql dump 子命令
func newMySQLDumpCmd() *cobra.Command {
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
		Short: "导出MySQL数据库",
		Long: `导出MySQL数据库到SQL或CSV文件（纯Go实现，不依赖mysqldump）

支持导出表结构和数据，可选择SQL或CSV格式。

Examples:
  # 导出整个数据库
  opsxcli mysql dump -h localhost -u root -p -d mydb -o dump.sql

  # 导出指定表
  opsxcli mysql dump -h localhost -u root -p -d mydb --tables users,orders -o dump.sql

  # 只导出结构
  opsxcli mysql dump -h localhost -u root -p -d mydb --no-data -o schema.sql

  # 只导出数据
  opsxcli mysql dump -h localhost -u root -p -d mydb --no-schema -o data.sql

  # 导出为CSV格式
  opsxcli mysql dump -h localhost -u root -p -d mydb --format csv -o ./csv_output/`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := getSubcommandDBFlags(cmd)

			if flags.Password == "ASK" {
				flags.Password = PromptPassword("Enter password")
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

			opts := mysql.DefaultDumpOptions()
			opts.Format = format
			opts.WithData = !noData
			opts.WithSchema = !noSchema
			if tables != "" {
				opts.Tables = strings.Split(tables, ",")
			}
			if ignoreTables != "" {
				opts.IgnoreTables = strings.Split(ignoreTables, ",")
			}

			return mysql.DumpDatabase(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, outputPath, opts)
		},
	}

	// 数据库连接参数
	addSubcommandDBFlags(dumpCmd, DatabaseDefaults{
		Host: "127.0.0.1",
		Port: 3306,
		User: "root",
	})

	// dump 特有参数
	dumpCmd.Flags().StringVarP(&outputPath, "output", "o", "", "输出文件路径（SQL）或目录（CSV）")
	dumpCmd.Flags().StringVar(&format, "format", "sql", "输出格式: sql 或 csv")
	dumpCmd.Flags().StringVar(&tables, "tables", "", "指定导出的表，逗号分隔")
	dumpCmd.Flags().StringVar(&ignoreTables, "ignore-tables", "", "忽略的表，逗号分隔")
	dumpCmd.Flags().BoolVar(&noData, "no-data", false, "不包含数据")
	dumpCmd.Flags().BoolVar(&noSchema, "no-schema", false, "不包含表结构")

	return dumpCmd
}

// newMySQLRestoreCmd 创建 mysql restore 子命令
func newMySQLRestoreCmd() *cobra.Command {
	var inputPath string

	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "导入MySQL数据库",
		Long: `从SQL文件导入数据到MySQL数据库

Examples:
  # 从SQL文件导入
  opsxcli mysql restore -h localhost -u root -p -d mydb -i dump.sql`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := getSubcommandDBFlags(cmd)

			if flags.Password == "ASK" {
				flags.Password = PromptPassword("Enter password")
			}

			if flags.Database == "" || inputPath == "" {
				return cmd.Help()
			}

			return mysql.RestoreDatabase(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, inputPath)
		},
	}

	// 数据库连接参数
	addSubcommandDBFlags(restoreCmd, DatabaseDefaults{
		Host: "127.0.0.1",
		Port: 3306,
		User: "root",
	})

	restoreCmd.Flags().StringVarP(&inputPath, "input", "i", "", "输入SQL文件路径")

	return restoreCmd
}

// addSubcommandDBFlags 为子命令添加数据库连接参数
func addSubcommandDBFlags(cmd *cobra.Command, defaults DatabaseDefaults) {
	cmd.Flags().StringP("host", "h", defaults.Host, "数据库主机地址")
	cmd.Flags().IntP("port", "P", defaults.Port, "数据库端口")
	cmd.Flags().StringP("user", "u", defaults.User, "数据库用户名")
	cmd.Flags().StringP("password", "p", "", "数据库密码")
	cmd.Flags().Lookup("password").NoOptDefVal = "ASK"
	cmd.Flags().StringP("database", "d", "", "数据库名称")
	cmd.Flags().Bool("help", false, "help for "+cmd.Name())
}

// getSubcommandDBFlags 从子命令中获取数据库连接参数
func getSubcommandDBFlags(cmd *cobra.Command) DatabaseFlags {
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
