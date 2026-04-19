package cmd

import (
	"opsxcli/plugins/ping"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("ping", "网络", "网络连通性测试", NewPingCmd)
}

// NewPingCmd 创建ping命令
func NewPingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ping <host>",
		Short:         "网络连通性测试",
		Long:          "测试到指定主机的网络连通性",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			host := args[0]
			count, _ := cmd.Flags().GetInt("count")
			interval, _ := cmd.Flags().GetDuration("interval")
			timeout, _ := cmd.Flags().GetDuration("timeout")

			return ping.Ping(host, count, interval, timeout)
		},
	}

	cmd.Flags().IntP("count", "c", -1, "发送次数（-1表示持续）")
	cmd.Flags().DurationP("interval", "i", ping.DefaultInterval, "发送间隔")
	cmd.Flags().DurationP("timeout", "W", ping.DefaultTimeout, "超时时间")

	return cmd
}
