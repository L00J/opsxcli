package cmd

import (
	"os"
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

Examples:
  # 进入交互式PostgreSQL shell（提示输入密码）
  opsxcli psql -h localhost -P 5432 -U postgres -W

  # 使用环境变量密码（官方标准）
  export PGPASSWORD=your_password
  opsxcli psql -h localhost -P 5432 -U postgres

  # 执行SQL语句
  opsxcli psql -h localhost -U postgres -d testdb -c "SELECT version();"

  # 连接指定数据库
  opsxcli psql -h 192.168.1.100 -P 5432 -U admin -d production

  # 快速查询
  opsxcli psql -h localhost -U postgres -c "\l"  # 列出所有数据库
  opsxcli psql -h localhost -U postgres -d mydb -c "\dt"  # 列出表`,
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := GetDatabaseFlags(cmd)

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

			if flags.Execute != "" {
				return postgres.Execute(flags.Host, flags.Port, flags.User, flags.Password, flags.Database, flags.Execute)
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

	return cmd
}
