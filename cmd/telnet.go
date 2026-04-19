package cmd

import (
	"opsxcli/plugins/telnet"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("telnet", "网络", "Telnet连接测试", NewTelnetCmd)
}

// NewTelnetCmd 创建telnet命令
func NewTelnetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telnet [host] [port]",
		Short: "Telnet客户端工具",
		Long: `Telnet工具支持以下功能：
  - 连接到远程主机的Telnet服务
  - 监听端口作为Telnet服务器
  - 交互式终端会话`,
		Args:          cobra.RangeArgs(0, 2),
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			listen, _ := cmd.Flags().GetBool("listen")
			verbose, _ := cmd.Flags().GetBool("verbose")
			timeout, _ := cmd.Flags().GetDuration("timeout")
			port, _ := cmd.Flags().GetInt("port")

			if listen {
				return telnet.Listen(port, timeout, verbose)
			}

			if len(args) < 2 {
				return cmd.Help()
			}

			if timeout == 0 {
				timeout = 10 * time.Second
			}
			return telnet.Connect(args[0], args[1], timeout, verbose)
		},
	}

	cmd.Flags().BoolP("listen", "l", false, "监听模式（服务器模式）")
	cmd.Flags().IntP("port", "p", 23, "端口号")
	cmd.Flags().BoolP("verbose", "v", false, "详细输出")
	cmd.Flags().DurationP("timeout", "t", 10*time.Second, "连接超时时间")

	return cmd
}
