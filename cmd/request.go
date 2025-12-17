package cmd

import (
	"opsxcli/plugins/request"

	"github.com/spf13/cobra"
)

// NewRequestCmd 创建request命令
func NewRequestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "request <url>",
		Short:         "高级HTTP请求工具",
		Long:          "发送HTTP请求，支持GET、POST等方法",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
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
