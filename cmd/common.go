package cmd

import (
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// securePassword 安全密码存储：设置后仅可读取一次，读取后立即清零内存
// 替代原来的全局 string 变量，避免密码长期驻留在进程内存中
var securePassword []byte

// SetDatabasePassword 设置数据库密码（供 main.go 预处理使用）
// 密码以 []byte 存储，便于用后即清
func SetDatabasePassword(pwd string) {
	securePassword = []byte(pwd)
}

// GetDatabasePassword 获取预处理时提取的密码
// 保留向后兼容：仅用于不需要立即清零的场景
func GetDatabasePassword() string {
	if securePassword == nil {
		return ""
	}
	return string(securePassword)
}

// FetchAndClearDatabasePassword 获取密码后立即从内存中清零
// 适用于密码使用后不再需要的场景（推荐使用此方法）
// 返回密码字符串并从 securePassword 中擦除原始数据
func FetchAndClearDatabasePassword() string {
	if securePassword == nil {
		return ""
	}
	pwd := string(securePassword)
	// 内存清零：覆写原始字节
	for i := range securePassword {
		securePassword[i] = 0
	}
	securePassword = nil
	return pwd
}

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

	// 密码 flag 支持可选参数：-p 不带值时使用空字符串标记（后续检测并提示输入）
	cmd.Flags().StringP("password", "p", "", "数据库密码")
	cmd.Flags().Lookup("password").NoOptDefVal = "ASK" // 当 -p 不带参数时的默认值

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

	// 如果密码是 "ASK" 且我们之前从 -pXXX 中提取过密码，则使用提取的密码
	// 使用 FetchAndClearDatabasePassword 获取后立即清零，避免密码长期驻留内存
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
		Execute:  execute,
	}
}

// PromptPassword 提示用户输入密码
func PromptPassword(prompt string) string {
	fmt.Printf("%s: ", prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // 换行
	if err != nil {
		return ""
	}
	return string(bytePassword)
}

// DatabaseDefaults 数据库默认配置
type DatabaseDefaults struct {
	Host string
	Port int
	User string
}
