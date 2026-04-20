package cmd

import (
	"opsxcli/plugins/request"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("requests", "工具", "HTTP请求测试", NewRequestsCmd)
}

// NewRequestsCmd 创建 requests 命令
func NewRequestsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "requests <url>",
		Short:         "高级HTTP请求工具",
		Long:          "发送HTTP请求，支持GET、POST等方法",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			method, _ := cmd.Flags().GetString("method")
			header, _ := cmd.Flags().GetStringSlice("header")
			data, _ := cmd.Flags().GetString("data")
			output, _ := cmd.Flags().GetString("output")

			return request.Send(url, method, header, data, output)
		},
	}

	cmd.Flags().StringP("method", "X", "GET", "HTTP方法")
	cmd.Flags().StringSliceP("header", "H", []string{}, "HTTP头（可多次使用）")
	cmd.Flags().StringP("data", "d", "", "请求体数据")
	cmd.Flags().StringP("output", "o", "", "输出到文件")

	return cmd
}
