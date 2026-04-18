package cmd

import (
	"opsxcli/plugins/busybox"

	"github.com/spf13/cobra"
)

// === 基本文件操作 ===

func NewLsCmd() *cobra.Command {
	var opts busybox.LsOptions
	cmd := &cobra.Command{
		Use:   "ls [flags] [files...]",
		Short: "列出目录内容",
		Long:  "列出目录中的文件和目录（Go 原生实现）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Ls(args, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "显示隐藏文件")
	cmd.Flags().BoolVarP(&opts.Long, "long", "l", false, "使用长格式")
	cmd.Flags().BoolVar(&opts.Human, "human-readable", false, "人类可读的大小")
	cmd.Flags().BoolVarP(&opts.Recursive, "recursive", "R", false, "递归列出子目录")
	cmd.Flags().BoolVarP(&opts.SortByTime, "time", "t", false, "按修改时间排序")
	cmd.Flags().BoolVarP(&opts.Reverse, "reverse", "r", false, "反向排序")
	return cmd
}

func NewCpCmd() *cobra.Command {
	var recursive bool
	cmd := &cobra.Command{
		Use:   "cp [flags] source dest",
		Short: "复制文件或目录",
		Long:  "复制文件或目录到指定位置（Go 原生实现）",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Cp(args[0], args[1], recursive)
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "递归复制目录")
	return cmd
}

func NewMvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mv source dest",
		Short: "移动或重命名文件",
		Long:  "移动文件或目录，或重命名（Go 原生实现）",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Mv(args[0], args[1])
		},
	}
}

func NewRmCmd() *cobra.Command {
	var recursive, force bool
	cmd := &cobra.Command{
		Use:   "rm [flags] files...",
		Short: "删除文件或目录",
		Long:  "删除指定的文件或目录（Go 原生实现）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Rm(args, recursive, force)
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "递归删除目录")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "强制删除，忽略错误")
	return cmd
}

func NewMkdirCmd() *cobra.Command {
	var parents bool
	cmd := &cobra.Command{
		Use:   "mkdir [flags] directories...",
		Short: "创建目录",
		Long:  "创建新目录（Go 原生实现）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Mkdir(args, parents, 0755)
		},
	}
	cmd.Flags().BoolVarP(&parents, "parents", "p", false, "自动创建父目录")
	return cmd
}

func NewRmdirCmd() *cobra.Command {
	var parents bool
	cmd := &cobra.Command{
		Use:   "rmdir [flags] directories...",
		Short: "删除空目录",
		Long:  "删除空目录（Go 原生实现）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Rmdir(args, parents)
		},
	}
	cmd.Flags().BoolVarP(&parents, "parents", "p", false, "删除目录及其祖先目录")
	return cmd
}

func NewTreeCmd() *cobra.Command {
	return createForwardCmd("tree", "树形显示目录结构", "以树形结构显示目录内容")
}

func NewTouchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "touch files...",
		Short: "创建空文件或更新时间戳",
		Long:  "创建空文件或更新文件的访问和修改时间（Go 原生实现）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Touch(args)
		},
	}
}

func NewChmodCmd() *cobra.Command {
	var recursive bool
	cmd := &cobra.Command{
		Use:   "chmod [flags] mode files...",
		Short: "修改文件权限",
		Long: `修改文件或目录的访问权限（Go 原生实现）

权限模式使用八进制表示，例如:
  755 - rwxr-xr-x
  644 - rw-r--r--
  777 - rwxrwxrwx

示例:
  opsxcli chmod 755 file.sh
  opsxcli chmod -R 644 /path/to/dir`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := args[0]
			paths := args[1:]
			return busybox.Chmod(mode, paths, recursive)
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "R", false, "递归修改目录权限")
	return cmd
}

func NewChownCmd() *cobra.Command {
	var recursive bool
	cmd := &cobra.Command{
		Use:   "chown [flags] owner[:group] files...",
		Short: "修改文件所有者",
		Long: `修改文件或目录的所有者和组（Go 原生实现）

所有者格式:
  UID       - 只改变所有者
  UID:GID   - 同时改变所有者和组

示例:
  opsxcli chown 1000 file.txt
  opsxcli chown 1000:1000 file.txt
  opsxcli chown -R 0:0 /path/to/dir`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner := args[0]
			paths := args[1:]
			return busybox.Chown(owner, paths, recursive)
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "R", false, "递归修改目录所有者")
	return cmd
}

func NewLnCmd() *cobra.Command {
	var symbolic bool
	cmd := &cobra.Command{
		Use:   "ln [flags] target link",
		Short: "创建链接",
		Long: `创建硬链接或符号链接（Go 原生实现）

示例:
  opsxcli ln file.txt hardlink.txt        # 创建硬链接
  opsxcli ln -s /path/to/file symlink     # 创建符号链接`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			link := args[1]
			return busybox.Ln(target, link, symbolic)
		},
	}
	cmd.Flags().BoolVarP(&symbolic, "symbolic", "s", false, "创建符号链接而非硬链接")
	return cmd
}

func NewCatCmd() *cobra.Command {
	var showLineNumbers bool
	cmd := &cobra.Command{
		Use:   "cat [flags] [files...]",
		Short: "显示文件内容",
		Long:  "连接文件并打印到标准输出（Go 原生实现）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.Cat(args, showLineNumbers)
		},
	}
	cmd.Flags().BoolVarP(&showLineNumbers, "number", "n", false, "显示行号")
	return cmd
}
