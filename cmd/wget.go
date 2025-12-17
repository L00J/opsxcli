package cmd

import (
	"opsxcli/plugins/wget"

	"github.com/spf13/cobra"
)

// NewWgetCmd 创建wget命令
func NewWgetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "wget <url>",
		Short:         "文件下载工具",
		Long:          "下载文件，支持断点续传等功能",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			output, _ := cmd.Flags().GetString("output")
			continueDownload, _ := cmd.Flags().GetBool("continue")

			return wget.Download(url, output, continueDownload)
		},
	}

	cmd.Flags().StringP("output", "O", "", "输出文件名")
	cmd.Flags().BoolP("continue", "c", false, "断点续传")

	return cmd
}
