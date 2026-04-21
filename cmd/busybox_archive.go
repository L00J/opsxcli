package cmd

import (
	"opsxcli/plugins/busybox"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("tar", "归档", "创建或解压 tar 归档", NewTarCmd)
	RegisterCommand("gzip", "归档", "压缩或解压 gzip 文件", NewGzipCmd)
	RegisterCommand("unzip", "归档", "解压 ZIP 文件", NewUnzipCmd)
}

// ==================== tar ====================

func NewTarCmd() *cobra.Command {
	var opts busybox.TarOptions
	cmd := &cobra.Command{
		Use:   "tar [flags] [files...]",
		Short: "创建或解压 tar 归档",
		Long:  "创建、解压或列出 tar 归档文件（Go 原生实现，支持 gzip）",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Tar(args, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Create, "create", "c", false, "创建归档")
	cmd.Flags().BoolVarP(&opts.Extract, "extract", "x", false, "解压归档")
	cmd.Flags().BoolVarP(&opts.List, "list", "t", false, "列出归档内容")
	cmd.Flags().BoolVarP(&opts.Gzip, "gzip", "z", false, "使用 gzip 压缩/解压")
	cmd.Flags().StringVarP(&opts.File, "file", "f", "", "归档文件名")
	cmd.Flags().StringVarP(&opts.Directory, "directory", "C", "", "切换到指定目录")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "详细输出")
	return cmd
}

// ==================== gzip ====================

func NewGzipCmd() *cobra.Command {
	var opts busybox.GzipOptions
	cmd := &cobra.Command{
		Use:   "gzip [flags] <file> [file...]",
		Short: "压缩或解压 gzip 文件",
		Long:  "压缩或解压 gzip 文件（Go 原生实现）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Gzip(args, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Decompress, "decompress", "d", false, "解压文件")
	cmd.Flags().BoolVarP(&opts.Keep, "keep", "k", false, "保留原文件")
	cmd.Flags().BoolVarP(&opts.Stdout, "stdout", "c", false, "输出到标准输出")
	return cmd
}

// ==================== unzip ====================

func NewUnzipCmd() *cobra.Command {
	var opts busybox.UnzipOptions
	cmd := &cobra.Command{
		Use:   "unzip [flags] <zipfile>",
		Short: "解压 ZIP 文件",
		Long:  "解压或列出 ZIP 归档文件内容（Go 原生实现）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Unzip(args, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "列出内容（不解压）")
	cmd.Flags().StringVarP(&opts.Dir, "dir", "d", "", "解压到指定目录")
	cmd.Flags().BoolVarP(&opts.Quiet, "quiet", "q", false, "静默模式")
	return cmd
}
