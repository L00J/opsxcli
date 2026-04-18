package cmd

import (
	"opsxcli/plugins/busybox"

	"github.com/spf13/cobra"
)

// === 文件查看/编辑 ===

func NewMoreCmd() *cobra.Command {
	return createForwardCmd("more", "分页显示文件", "分页查看文件内容")
}

func NewLessCmd() *cobra.Command {
	return createForwardCmd("less", "分页显示文件（增强版）", "分页查看文件内容，支持向前翻页")
}

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

func NewAwkCmd() *cobra.Command {
	return createForwardCmd("awk", "文本处理工具", "强大的文本处理和数据提取工具")
}

func NewSedCmd() *cobra.Command {
	return createForwardCmd("sed", "流编辑器", "非交互式文本编辑器")
}

func NewViCmd() *cobra.Command {
	return createForwardCmd("vi", "文本编辑器", "经典的 vi 文本编辑器")
}

func NewVimCmd() *cobra.Command {
	return createForwardCmd("vim", "增强版文本编辑器", "vi 的增强版本")
}
