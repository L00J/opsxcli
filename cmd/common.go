package cmd

import (
	"github.com/spf13/cobra"
)

// DatabaseFlags 数据库连接通用参数
type DatabaseFlags struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Execute  string
}

// AddDatabaseFlags 为命令添加数据库连接通用参数
func AddDatabaseFlags(cmd *cobra.Command, defaults DatabaseDefaults) {
	cmd.Flags().StringP("host", "h", defaults.Host, "数据库主机地址")
	cmd.Flags().IntP("port", "P", defaults.Port, "数据库端口")
	cmd.Flags().StringP("user", "u", defaults.User, "数据库用户名")
	cmd.Flags().StringP("password", "p", "", "数据库密码")
	cmd.Flags().StringP("database", "d", "", "数据库名称")
	cmd.Flags().StringP("execute", "e", "", "执行SQL语句")

	// 手动添加 help flag（没有 shorthand），避免与 -h host 冲突
	cmd.Flags().Bool("help", false, "help for "+cmd.Name())
}

// GetDatabaseFlags 从命令中获取数据库连接参数
func GetDatabaseFlags(cmd *cobra.Command) DatabaseFlags {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	user, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	database, _ := cmd.Flags().GetString("database")
	execute, _ := cmd.Flags().GetString("execute")

	return DatabaseFlags{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: database,
		Execute:  execute,
	}
}

// DatabaseDefaults 数据库默认配置
type DatabaseDefaults struct {
	Host string
	Port int
	User string
}
