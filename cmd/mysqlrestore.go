package cmd

import (
	opsxmysql "opsxcli/plugins/mysql"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("mysqlrestore", "数据库", "导入MySQL数据库", NewMysqlrestoreCmd)
}

// NewMysqlrestoreCmd 创建 mysqlrestore 顶级命令
func NewMysqlrestoreCmd() *cobra.Command {
	var inputPath string

	cmd := &cobra.Command{
		Use:   "mysqlrestore",
		Short: "从SQL文件导入数据到MySQL数据库",
		Long: `从SQL文件导入数据到MySQL数据库（纯Go实现）

Examples:
  # 从SQL文件导入
  opsxcli mysqlrestore -h localhost -u root -p -d mydb -i dump.sql

  # 导入并指定端口
  opsxcli mysqlrestore -h 192.168.1.100 -P 3307 -u admin -p -d mydb -i dump.sql`,
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

			return opsxmysql.RestoreDatabase(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, inputPath)
		},
	}

	// 数据库连接参数（MySQL 风格）
	addSubcommandDBFlags(cmd, DatabaseDefaults{
		Host: "127.0.0.1",
		Port: 3306,
		User: "root",
	})

	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "输入SQL文件路径")

	return cmd
}
