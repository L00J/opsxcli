package cmd

import (
	"opsxcli/plugins/busybox"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("tree", "文件", "以树形结构显示目录内容", NewTreeCmd)
	RegisterCommand("du", "文件", "显示目录空间使用情况", NewDuCmd)
}

// ==================== tree ====================

func NewTreeCmd() *cobra.Command {
	var opts busybox.TreeOptions
	cmd := &cobra.Command{
		Use:   "tree [flags] [path]",
		Short: "以树形结构显示目录内容",
		Long:  "以树形结构递归显示目录和文件（Go 原生实现）",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Tree(args, opts)
		},
	}
	cmd.Flags().IntVarP(&opts.MaxDepth, "depth", "L", 0, "最大显示深度 (0=无限)")
	cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "显示隐藏文件")
	cmd.Flags().BoolVarP(&opts.DirsOnly, "dirs-only", "d", false, "只显示目录")
	cmd.Flags().BoolVarP(&opts.Fullpath, "fullpath", "f", false, "显示完整路径")
	return cmd
}

// ==================== du ====================

func NewDuCmd() *cobra.Command {
	var opts busybox.DuOptions
	cmd := &cobra.Command{
		Use:   "du [flags] [path...]",
		Short: "显示目录空间使用情况",
		Long:  "显示文件和目录的磁盘空间使用情况（Go 原生实现）",
		Args:  cobra.MaximumNArgs(10),
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Du(args, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Human, "human", "h", false, "人类可读的大小 (如 1.2G)")
	cmd.Flags().BoolVarP(&opts.Summarize, "summarize", "s", false, "只显示总计")
	cmd.Flags().IntVar(&opts.MaxDepth, "max-depth", 0, "最大显示深度 (0=无限)")
	cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "显示单个文件大小")
	return cmd
}
