package cmd

import (
	"opsxcli/plugins/curl"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("curl", "工具", "HTTP请求客户端", NewCurlCmd)
}

// NewCurlCmd 创建curl命令
func NewCurlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "curl <url>",
		Short:         "类curl HTTP客户端",
		Long:          "发送HTTP请求并显示响应，兼容常用curl参数",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			method, _ := cmd.Flags().GetString("request")
			header, _ := cmd.Flags().GetStringSlice("header")
			data, _ := cmd.Flags().GetString("data")
			output, _ := cmd.Flags().GetString("output")
			includeHeaders, _ := cmd.Flags().GetBool("include")
			headOnly, _ := cmd.Flags().GetBool("head")
			verbose, _ := cmd.Flags().GetBool("verbose")
			followRedirect, _ := cmd.Flags().GetBool("location")

			// 如果是 HEAD 请求，覆盖 method
			if headOnly {
				method = "HEAD"
			}

			return curl.Request(url, method, header, data, output, includeHeaders, verbose, followRedirect)
		},
	}

	cmd.Flags().StringP("request", "X", "GET", "HTTP请求方法")
	cmd.Flags().StringSliceP("header", "H", []string{}, "自定义HTTP头（可多次使用）")
	cmd.Flags().StringP("data", "d", "", "POST数据")
	cmd.Flags().StringP("output", "o", "", "输出到文件")
	cmd.Flags().BoolP("include", "i", false, "在输出中包含响应头")
	cmd.Flags().BoolP("head", "I", false, "只获取HTTP头（HEAD请求）")
	cmd.Flags().BoolP("verbose", "v", false, "显示详细请求信息")
	cmd.Flags().BoolP("location", "L", false, "跟随重定向")

	// 重置帮助函数为默认
	cmd.SetHelpFunc(nil)

	return cmd
}
