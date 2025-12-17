package cmd

import (
	"opsxcli/plugins/netstat"

	"github.com/spf13/cobra"
)

// NewNetstatCmd 创建netstat命令
func NewNetstatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "netstat",
		Short:         "网络连接状态查看",
		Long:          "显示网络连接、路由表和网络接口信息",
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			listen, _ := cmd.Flags().GetBool("listen")
			all, _ := cmd.Flags().GetBool("all")
			tcp, _ := cmd.Flags().GetBool("tcp")
			udp, _ := cmd.Flags().GetBool("udp")
			numeric, _ := cmd.Flags().GetBool("numeric")
			programs, _ := cmd.Flags().GetBool("programs")

			return netstat.Netstat(listen, all, tcp, udp, numeric, programs)
		},
	}

	cmd.Flags().BoolP("listen", "l", false, "只显示监听状态的连接")
	cmd.Flags().BoolP("all", "a", false, "显示所有连接")
	cmd.Flags().BoolP("tcp", "t", false, "显示TCP连接")
	cmd.Flags().BoolP("udp", "u", false, "显示UDP连接")
	cmd.Flags().BoolP("numeric", "n", false, "以数字形式显示地址和端口")
	cmd.Flags().BoolP("programs", "p", false, "显示PID和程序名")

	return cmd
}
