package cmd

import (
	"strings"

	opsxmysql "opsxcli/plugins/mysql"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("mysqldump", "数据库", "导出MySQL数据库", NewMysqldumpCmd)
}

// NewMysqldumpCmd 创建 mysqldump 顶级命令
func NewMysqldumpCmd() *cobra.Command {
	var (
		format       string
		tables       string
		ignoreTables string
		noData       bool
		noSchema     bool
		outputPath   string
	)

	cmd := &cobra.Command{
		Use:   "mysqldump",
		Short: "导出MySQL数据库（纯Go实现，不依赖mysqldump）",
		Long: `导出MySQL数据库到SQL或CSV文件

纯Go实现，无需安装mysqldump客户端。支持导出表结构和数据。

Examples:
  # 导出整个数据库
  opsxcli mysqldump -h localhost -u root -p -d mydb -o dump.sql

  # 导出指定表
  opsxcli mysqldump -h localhost -u root -p -d mydb --tables users,orders -o dump.sql

  # 只导出结构
  opsxcli mysqldump -h localhost -u root -p -d mydb --no-data -o schema.sql

  # 只导出数据
  opsxcli mysqldump -h localhost -u root -p -d mydb --no-schema -o data.sql

  # 导出为CSV格式
  opsxcli mysqldump -h localhost -u root -p -d mydb --format csv -o ./csv_output/`,
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

			opts := opsxmysql.DefaultDumpOptions()
			opts.Format = format
			opts.WithData = !noData
			opts.WithSchema = !noSchema
			if tables != "" {
				opts.Tables = strings.Split(tables, ",")
			}
			if ignoreTables != "" {
				opts.IgnoreTables = strings.Split(ignoreTables, ",")
			}

			return opsxmysql.DumpDatabase(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, outputPath, opts)
		},
	}

	// 数据库连接参数（MySQL 风格）
	addSubcommandDBFlags(cmd, DatabaseDefaults{
		Host: "127.0.0.1",
		Port: 3306,
		User: "root",
	})

	// dump 特有参数
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "输出文件路径（SQL）或目录（CSV）")
	cmd.Flags().StringVar(&format, "format", "sql", "输出格式: sql 或 csv")
	cmd.Flags().StringVar(&tables, "tables", "", "指定导出的表，逗号分隔")
	cmd.Flags().StringVar(&ignoreTables, "ignore-tables", "", "忽略的表，逗号分隔")
	cmd.Flags().BoolVar(&noData, "no-data", false, "不包含数据")
	cmd.Flags().BoolVar(&noSchema, "no-schema", false, "不包含表结构")

	return cmd
}
