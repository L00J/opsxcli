package cmd

import (
	"opsxcli/plugins/mysql"

	"github.com/spf13/cobra"
)

// NewMySQLCmd 创建MySQL命令
func NewMySQLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mysql",
		Short: "MySQL数据库操作工具",
		Long: `MySQL工具支持以下功能：
  - 交互式MySQL shell（支持自动补全、历史记录）
  - 执行SQL语句
  - 查询数据库和表

Examples:
  # 进入交互式MySQL shell
  opsxcli mysql -h localhost -P 3306 -u root -p

  # 执行SQL语句
  opsxcli mysql -h localhost -u root -p -d testdb -e "SELECT * FROM users LIMIT 10"

  # 连接指定数据库
  opsxcli mysql -h 192.168.1.100 -P 3306 -u admin -p mypassword -d production

  # 快速查询
  opsxcli mysql -h localhost -u root -e "SHOW DATABASES;"`,
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := GetDatabaseFlags(cmd)

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

	return cmd
}
