package cmd

import (
	"opsxcli/plugins/install"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("install", "管理", "安装系统服务", NewInstallCmd)
}

// NewInstallCmd 创建安装命令
func NewInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <package>",
		Short: "检测系统并使用对应包管理器安装服务",
		Long: `自动检测系统类型并使用对应的包管理器安装软件。

支持的系统:
  - macOS: 使用 Homebrew (brew)
  - RHEL/CentOS/Rocky/Alma/Amazon Linux/阿里云 Linux 2: 使用 yum
  - 阿里云 Linux 3/4: 使用 dnf
  - Debian/Ubuntu: 使用 apt-get

使用示例:
  opsxcli install mysql
  opsxcli install nginx
  opsxcli install docker`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			return install.Install(packageName)
		},
	}

	return cmd
}
