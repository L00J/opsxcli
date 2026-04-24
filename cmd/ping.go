package cmd

import (
	"opsxcli/plugins/ping"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("ping", "网络", "网络连通性测试（ICMP/UDP/TCP）", NewPingCmd)
}

// NewPingCmd 创建ping命令
func NewPingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ping <host>",
		Short:         "网络连通性测试",
		Long:          "测试到指定主机的网络连通性，支持 ICMP 原生探测（需 root）、UDP 回退（非 root）和 TCP 模式",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			host := args[0]
			count, _ := cmd.Flags().GetInt("count")
			interval, _ := cmd.Flags().GetDuration("interval")
			timeout, _ := cmd.Flags().GetDuration("timeout")
			mode, _ := cmd.Flags().GetString("mode")
			ipVersion, _ := cmd.Flags().GetString("ip-version")
			payloadSize, _ := cmd.Flags().GetInt("payload-size")

			cfg := ping.NewPingConfig(host)
			cfg.Count = count
			cfg.Interval = interval
			cfg.Timeout = timeout
			cfg.Mode = ping.PingMode(mode)
			cfg.IPVersion = ipVersion
			cfg.PayloadSize = payloadSize

			_, err := ping.RunPing(cfg)
			return err
		},
	}

	cmd.Flags().IntP("count", "c", 4, "发送次数（-1表示持续）")
	cmd.Flags().DurationP("interval", "i", ping.DefaultInterval, "发送间隔")
	cmd.Flags().DurationP("timeout", "W", ping.DefaultTimeout, "超时时间")
	cmd.Flags().StringP("mode", "m", "", "协议模式: icmp|udp|tcp（默认自动选择）")
	cmd.Flags().String("ip-version", "4", "IP版本: 4|6")
	cmd.Flags().IntP("payload-size", "s", 56, "负载大小（字节）")

	return cmd
}
