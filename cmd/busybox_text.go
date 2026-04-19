package cmd

import (
	"opsxcli/plugins/busybox"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("head", "文件", "查看文件头部", NewHeadCmd)
	RegisterCommand("tail", "文件", "查看文件尾部", NewTailCmd)
	RegisterCommand("grep", "文件", "文本搜索", NewGrepCmd)
}

// === 文件查看/编辑 ===

func NewHeadCmd() *cobra.Command {
	var lines int
	cmd := &cobra.Command{
		Use:   "head [flags] [files...]",
		Short: "显示文件开头",
		Long: `显示文件的前几行（Go 原生实现）

示例:
  opsxcli head file.txt          # 显示前 10 行
  opsxcli head -n 20 file.txt    # 显示前 20 行
  cat file.txt | opsxcli head    # 从标准输入读取`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Head(args, lines)
		},
	}
	cmd.Flags().IntVarP(&lines, "lines", "n", 10, "显示的行数")
	return cmd
}

func NewTailCmd() *cobra.Command {
	var lines int
	cmd := &cobra.Command{
		Use:   "tail [flags] [files...]",
		Short: "显示文件末尾",
		Long: `显示文件的后几行（Go 原生实现）

示例:
  opsxcli tail file.txt          # 显示后 10 行
  opsxcli tail -n 20 file.txt    # 显示后 20 行
  cat file.txt | opsxcli tail    # 从标准输入读取`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Tail(args, lines)
		},
	}
	cmd.Flags().IntVarP(&lines, "lines", "n", 10, "显示的行数")
	return cmd
}

func NewGrepCmd() *cobra.Command {
	var opts busybox.GrepOptions
	cmd := &cobra.Command{
		Use:   "grep [flags] pattern [files...]",
		Short: "搜索文本",
		Long:  "在文件中搜索匹配的文本（Go 原生实现）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := args[0]
			files := args[1:]
			if len(files) > 1 {
				opts.WithFilename = true
			}
			return busybox.Grep(pattern, files, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.IgnoreCase, "ignore-case", "i", false, "忽略大小写")
	cmd.Flags().BoolVarP(&opts.InvertMatch, "invert-match", "v", false, "反向匹配")
	cmd.Flags().BoolVarP(&opts.LineNumber, "line-number", "n", false, "显示行号")
	cmd.Flags().BoolVarP(&opts.CountOnly, "count", "c", false, "只显示匹配行数")
	return cmd
}
