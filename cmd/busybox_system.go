package cmd

import (
	"opsxcli/plugins/busybox"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("free", "系统", "显示内存使用情况", NewFreeCmd)
	RegisterCommand("df", "系统", "显示磁盘空间使用情况", NewDfCmd)
	RegisterCommand("kill", "系统", "发送信号给进程", NewKillCmd)
}

// ==================== free ====================

func NewFreeCmd() *cobra.Command {
	var opts busybox.FreeOptions
	cmd := &cobra.Command{
		Use:   "free [flags]",
		Short: "显示内存使用情况",
		Long:  "显示系统物理内存和交换分区使用情况（Go 原生实现，跨平台兼容）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Free(opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Human, "human", "h", false, "人类可读的大小 (如 1.2Gi)")
	cmd.Flags().BoolVar(&opts.Total, "total", false, "显示总计行")
	cmd.Flags().StringVarP(&opts.Unit, "unit", "u", "", "指定单位: b/k/m/g")
	return cmd
}

// ==================== df ====================

func NewDfCmd() *cobra.Command {
	var opts busybox.DfOptions
	cmd := &cobra.Command{
		Use:   "df [flags] [path...]",
		Short: "显示磁盘空间使用情况",
		Long:  "显示文件系统磁盘空间使用情况（Go 原生实现，跨平台兼容）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Df(opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Human, "human", "h", false, "人类可读的大小 (如 1.2Gi)")
	cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "显示所有文件系统（包括虚拟）")
	cmd.Flags().BoolVarP(&opts.FsType, "type", "T", false, "显示文件系统类型")
	cmd.Flags().StringVarP(&opts.FilterType, "filter", "t", "", "只显示指定类型的文件系统")
	return cmd
}

// ==================== kill ====================

func NewKillCmd() *cobra.Command {
	var opts busybox.KillOptions
	cmd := &cobra.Command{
		Use:   "kill [flags] <pid> [pid...]",
		Short: "发送信号给进程",
		Long:  "向指定进程发送信号（默认 SIGTERM），Go 原生实现",
		Args:  cobra.MinimumNArgs(0), // -l 不需要参数
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Kill(args, opts)
		},
	}
	cmd.Flags().StringVarP(&opts.Signal, "signal", "s", "", "指定信号 (如 TERM, HUP, INT, KILL, USR1, USR2)")
	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "列出所有支持的信号")
	return cmd
}
