package cmd

import (
	"opsxcli/plugins/upgrade"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("upgrade", "管理", "升级opsxcli", NewUpgradeCmd)
}

// NewUpgradeCmd 创建升级命令
func NewUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "升级 opsxcli 到最新版本",
		Long: `从 Gitee 下载最新版本的 opsxcli 并自动替换当前二进制文件。

升级过程：
  1. 从 Gitee 下载最新版本
  2. 备份当前版本
  3. 替换二进制文件
  4. 如果失败，自动恢复备份

Examples:
  # 升级到最新版本
  opsxcli upgrade

注意：
  - 升级需要写权限，可能需要 sudo
  - 备份文件会保留，可在确认无误后手动删除`,
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			return upgrade.Run()
		},
	}

	return cmd
}
