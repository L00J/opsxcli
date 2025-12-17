package main

import (
	"os"

	"opsxcli/cmd"
	"opsxcli/internal/core"
	"opsxcli/internal/logger"
	"opsxcli/internal/security"

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

func main() {
	// 初始化日志系统
	logger.Init()

	// 初始化核心引擎（包含代码保护）
	engine := core.NewEngine(version, buildTime, gitCommit)
	if err := engine.Validate(); err != nil {
		logger.Error("引擎验证失败: %v", err)
		os.Exit(1)
	}

	// 初始化安全保护
	protection := security.NewProtection()
	if err := protection.Verify(); err != nil {
		logger.Error("安全验证失败: %v", err)
		os.Exit(1)
	}

	// 创建根命令
	rootCmd := cmd.NewRootCmd(version)

	// 移除所有命令（包括子命令）的 help flag 短选项
	// 必须在 Execute 之前处理，确保 -h 可以用于 host 等参数
	removeHelpFlagShorthand(rootCmd)

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		logger.Error("执行失败: %v", err)
		logger.Close() // 确保错误日志被写入
		os.Exit(1)
	}

	// 正常退出时也关闭日志
	logger.Close()
}
