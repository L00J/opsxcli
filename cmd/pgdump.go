package cmd

import (
	"os"
	"strings"

	opsxpostgres "opsxcli/plugins/postgres"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("pgdump", "数据库", "导出PostgreSQL数据库", NewPgdumpCmd)
}

// NewPgdumpCmd 创建 pgdump 顶级命令
func NewPgdumpCmd() *cobra.Command {
	var (
		format       string
		tables       string
		ignoreTables string
		noData       bool
		noSchema     bool
		outputPath   string
	)

	cmd := &cobra.Command{
		Use:   "pgdump",
		Short: "导出PostgreSQL数据库（纯Go实现，不依赖pg_dump）",
		Long: `导出PostgreSQL数据库到SQL或CSV文件

纯Go实现，无需安装pg_dump客户端。支持导出表结构和数据。

Examples:
  # 导出整个数据库
  opsxcli pgdump -h localhost -U postgres -W -d mydb -o dump.sql

  # 导出指定表
  opsxcli pgdump -h localhost -U postgres -W -d mydb --tables users,orders -o dump.sql

  # 只导出结构
  opsxcli pgdump -h localhost -U postgres -W -d mydb --no-data -o schema.sql

  # 只导出数据
  opsxcli pgdump -h localhost -U postgres -W -d mydb --no-schema -o data.sql

  # 导出为CSV格式
  opsxcli pgdump -h localhost -U postgres -W -d mydb --format csv -o ./csv_output/`,
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

			opts := opsxpostgres.DefaultDumpOptions()
			opts.Format = format
			opts.WithData = !noData
			opts.WithSchema = !noSchema
			if tables != "" {
				opts.Tables = strings.Split(tables, ",")
			}
			if ignoreTables != "" {
				opts.IgnoreTables = strings.Split(ignoreTables, ",")
			}

			return opsxpostgres.DumpDatabase(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, outputPath, opts)
		},
	}

	// PostgreSQL 风格的数据库连接参数
	cmd.Flags().StringP("host", "h", "127.0.0.1", "PostgreSQL主机地址")
	cmd.Flags().IntP("port", "P", 5432, "PostgreSQL端口")
	cmd.Flags().StringP("user", "U", "postgres", "PostgreSQL用户名")
	cmd.Flags().StringP("password", "W", "", "PostgreSQL密码（或使用环境变量 PGPASSWORD）")
	cmd.Flags().Lookup("password").NoOptDefVal = "ASK"
	cmd.Flags().StringP("database", "d", "", "数据库名称")

	// dump 特有参数
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "输出文件路径（SQL）或目录（CSV）")
	cmd.Flags().StringVar(&format, "format", "sql", "输出格式: sql 或 csv")
	cmd.Flags().StringVar(&tables, "tables", "", "指定导出的表，逗号分隔")
	cmd.Flags().StringVar(&ignoreTables, "ignore-tables", "", "忽略的表，逗号分隔")
	cmd.Flags().BoolVar(&noData, "no-data", false, "不包含数据")
	cmd.Flags().BoolVar(&noSchema, "no-schema", false, "不包含表结构")
	cmd.Flags().Bool("help", false, "help for pgdump")

	return cmd
}
