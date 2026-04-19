package cmd

import (
	"os"

	opsxpostgres "opsxcli/plugins/postgres"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("pgrestore", "数据库", "导入PostgreSQL数据库", NewPgrestoreCmd)
}

// NewPgrestoreCmd 创建 pgrestore 顶级命令
func NewPgrestoreCmd() *cobra.Command {
	var inputPath string

	cmd := &cobra.Command{
		Use:   "pgrestore",
		Short: "从SQL文件导入数据到PostgreSQL数据库",
		Long: `从SQL文件导入数据到PostgreSQL数据库（纯Go实现）

Examples:
  # 从SQL文件导入
  opsxcli pgrestore -h localhost -U postgres -W -d mydb -i dump.sql

  # 使用环境变量密码
  export PGPASSWORD=***
  opsxcli pgrestore -h localhost -U postgres -d mydb -i dump.sql`,
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

			return opsxpostgres.RestoreDatabase(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, inputPath)
		},
	}

	// PostgreSQL 风格的数据库连接参数
	cmd.Flags().StringP("host", "h", "127.0.0.1", "PostgreSQL主机地址")
	cmd.Flags().IntP("port", "P", 5432, "PostgreSQL端口")
	cmd.Flags().StringP("user", "U", "postgres", "PostgreSQL用户名")
	cmd.Flags().StringP("password", "W", "", "PostgreSQL密码（或使用环境变量 PGPASSWORD）")
	cmd.Flags().Lookup("password").NoOptDefVal = "ASK"
	cmd.Flags().StringP("database", "d", "", "数据库名称")

	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "输入SQL文件路径")
	cmd.Flags().Bool("help", false, "help for pgrestore")

	return cmd
}
