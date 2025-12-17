package cmd

import (
	"opsxcli/plugins/nc"

	"github.com/spf13/cobra"
)

// NewNcCmd 创建nc命令
func NewNcCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nc [host] [port]",
		Short: "网络连接工具",
		Long: `网络连接工具，支持：
  - 连接远程主机
  - 监听端口
  - 内网反弹shell`,
		Args:          cobra.RangeArgs(0, 2),
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			listen, _ := cmd.Flags().GetBool("listen")
			verbose, _ := cmd.Flags().GetBool("verbose")
			execute, _ := cmd.Flags().GetString("execute")
			port, _ := cmd.Flags().GetInt("port")

			if listen {
				return nc.Listen(port, verbose, execute)
			}

			if len(args) < 2 {
				return cmd.Help()
			}
			return nc.Connect(args[0], args[1], verbose)
		},
	}

	cmd.Flags().BoolP("listen", "l", false, "监听模式")
	cmd.Flags().IntP("port", "p", 0, "端口号")
	cmd.Flags().BoolP("verbose", "v", false, "详细输出")
	cmd.Flags().StringP("execute", "e", "", "执行命令（监听模式）")

	return cmd
}
