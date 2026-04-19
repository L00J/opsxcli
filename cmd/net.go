package cmd

import (
	"opsxcli/plugins/net"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("net", "监控", "网络监控TUI", NewNetCmd)
}

// NewNetCmd 创建网络监控命令
func NewNetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "net",
		Short: "网络流量监控",
		Long: `实时监控网络流量情况，包括：
  - 网络接口列表
  - 实时流量统计（接收/发送）
  - 连接数统计
  - 带宽使用率

使用方向键切换网络接口，按 'q' 或 ESC 退出。`,
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			return net.Run()
		},
	}

	return cmd
}
