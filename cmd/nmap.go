package cmd

import (
	"opsxcli/plugins/nmap"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("nmap", "网络", "端口扫描", NewNmapCmd)
}

// NewNmapCmd 创建nmap命令
func NewNmapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nmap <host>",
		Short: "网络端口扫描",
		Long: `扫描指定主机的开放端口和服务

Examples:
  # 扫描单个端口
  opsxcli nmap localhost -p 80

  # 扫描多个端口
  opsxcli nmap localhost -p 80,443,22

  # 扫描端口范围
  opsxcli nmap localhost -p 22-9999 -T 500ms -v

  # 快速扫描常用端口
  opsxcli nmap 192.168.1.1 -p 1-1000`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			host := args[0]
			ports, _ := cmd.Flags().GetString("ports")
			timeout, _ := cmd.Flags().GetDuration("timeout")
			verbose, _ := cmd.Flags().GetBool("verbose")

			return nmap.Scan(host, ports, timeout, verbose)
		},
	}

	cmd.Flags().StringP("ports", "p", "1-1000", "要扫描的端口范围（如：80,443 或 1-1000）")
	cmd.Flags().DurationP("timeout", "T", nmap.DefaultTimeout, "连接超时时间")
	cmd.Flags().BoolP("verbose", "v", false, "显示详细信息")

	return cmd
}
