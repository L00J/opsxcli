package main

import (
	"os"
	"strings"

	"opsxcli/cmd"
	"opsxcli/internal/logger"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// removeHelpFlagShorthand 移除子命令的 help flag 的 shorthand，避免与自定义 flags 冲突
// 注意：根命令的 -h 保留，只移除子命令的 -h
func removeHelpFlagShorthand(c *cobra.Command) {
	// 递归处理所有子命令
	for _, subCmd := range c.Commands() {
		// 递归处理子命令的子命令
		removeHelpFlagShorthand(subCmd)

		// 移除子命令的 help flag shorthand
		if helpFlag := subCmd.PersistentFlags().Lookup("help"); helpFlag != nil {
			helpFlag.Shorthand = ""
			helpFlag.Hidden = true
		}
		if helpFlag := subCmd.Flags().Lookup("help"); helpFlag != nil {
			helpFlag.Shorthand = ""
			helpFlag.Hidden = true
		}
	}
}

// preprocessDatabasePasswordArgs 预处理数据库命令的密码参数
// 将 -pPASSWORD 格式转换为 -p PASSWORD 以兼容 Cobra flag 解析
func preprocessDatabasePasswordArgs() {
	if len(os.Args) < 2 {
		return
	}

	// 检查是否是数据库命令 (mysql, psql, redis)
	dbCommands := map[string]bool{"mysql": true, "psql": true, "redis": true}
	if !dbCommands[os.Args[1]] {
		return
	}

	// 查找并处理 -pXXX 格式的密码参数
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]

		// 匹配 -pXXX 格式（密码紧跟在 -p 后面）
		if strings.HasPrefix(arg, "-p") && len(arg) > 2 && arg[2] != '-' {
			// 提取并保存密码到 cmd 包的全局变量
			cmd.OriginalPassword = arg[2:]

			// 将 -pPASSWORD 替换为 -p 和 PASSWORD 两个参数
			os.Args[i] = "-p"
			// 在后面插入密码参数
			os.Args = append(os.Args[:i+1], append([]string{cmd.OriginalPassword}, os.Args[i+1:]...)...)
			break // 只处理第一个匹配项
		}
	}
}

func main() {
	// 预处理数据库密码参数（必须在创建命令之前）
	// 将 -pPASSWORD 拆分为 -p PASSWORD 避免 Cobra 解析错误
	preprocessDatabasePasswordArgs()

	// 初始化日志系统
	logger.Init()
	defer logger.Close()

	// 创建根命令
	rootCmd := cmd.NewRootCmd(version)

	// 移除所有命令（包括子命令）的 help flag 短选项
	// 必须在 Execute 之前处理，确保 -h 可以用于 host 等参数
	removeHelpFlagShorthand(rootCmd)

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
