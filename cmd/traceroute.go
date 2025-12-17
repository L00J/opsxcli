package cmd

import (
	"opsxcli/plugins/traceroute"

	"github.com/spf13/cobra"
)

// NewTracerouteCmd 创建traceroute命令
func NewTracerouteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "traceroute <host>",
		Short:         "路由追踪",
		Long:          "追踪数据包到目标主机的路由路径",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			host := args[0]
			maxHops, _ := cmd.Flags().GetInt("max-hops")
			packetSize, _ := cmd.Flags().GetInt("packet-size")

			return traceroute.Traceroute(host, maxHops, packetSize)
		},
	}

	cmd.Flags().IntP("max-hops", "m", 30, "最大跳数")
	cmd.Flags().IntP("packet-size", "s", 60, "数据包大小（字节）")

	return cmd
}
