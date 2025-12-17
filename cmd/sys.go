package cmd

import (
	"opsxcli/plugins/sys"

	"github.com/spf13/cobra"
)

// NewSysCmd 创建系统监控命令
func NewSysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sys",
		Short: "系统监控可视化",
		Long: `实时监控系统资源使用情况，包括：
  - CPU使用率
  - 内存使用情况
  - 磁盘IO
  - 进程TOP列表
  - 系统负载

使用方向键和Tab键切换视图，按 'q' 或 ESC 退出。`,
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			return sys.Run()
		},
	}

	return cmd
}
