package cmd

import (
	"github.com/spf13/cobra"
	"opsxcli/internal/exec"
	"opsxcli/plugins/busybox"
)

// createForwardCmd 创建一个转发到系统命令的 cobra 命令
func createForwardCmd(name, short, long string) *cobra.Command {
	return &cobra.Command{
		Use:                name,
		Short:              short,
		Long:               long,
		DisableFlagParsing: true, // 不解析参数，直接转发
		RunE: func(cmd *cobra.Command, args []string) error {
			return exec.ForwardCommand(name, args)
		},
	}
}

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
	return createForwardCmd("rmdir", "删除空目录", "删除空目录")
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
	return createForwardCmd("chmod", "修改文件权限", "修改文件或目录的访问权限")
}

func NewChownCmd() *cobra.Command {
	return createForwardCmd("chown", "修改文件所有者", "修改文件或目录的所有者和组")
}

func NewLnCmd() *cobra.Command {
	return createForwardCmd("ln", "创建链接", "创建硬链接或符号链接")
}

// === 文件查看/编辑 ===

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

func NewMoreCmd() *cobra.Command {
	return createForwardCmd("more", "分页显示文件", "分页查看文件内容")
}

func NewLessCmd() *cobra.Command {
	return createForwardCmd("less", "分页显示文件（增强版）", "分页查看文件内容，支持向前翻页")
}

func NewHeadCmd() *cobra.Command {
	return createForwardCmd("head", "显示文件开头", "显示文件的前几行")
}

func NewTailCmd() *cobra.Command {
	return createForwardCmd("tail", "显示文件末尾", "显示文件的后几行")
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

// === 归档压缩 ===

func NewTarCmd() *cobra.Command {
	return createForwardCmd("tar", "归档工具", "创建、提取或列出 tar 归档文件")
}

func NewGzipCmd() *cobra.Command {
	return createForwardCmd("gzip", "压缩文件", "使用 gzip 压缩文件")
}

func NewUnzipCmd() *cobra.Command {
	return createForwardCmd("unzip", "解压 ZIP 文件", "解压 ZIP 格式的压缩文件")
}

// === 进程管理 ===

func NewPsCmd() *cobra.Command {
	return createForwardCmd("ps", "显示进程信息", "显示当前运行的进程")
}

func NewTopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "top",
		Short: "实时系统监控",
		Long:  "实时显示系统进程信息（推荐使用 'opsxcli sys' 获得更好的体验）",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 优先使用 opsxcli sys，但保留 top 兼容性
			if len(args) == 0 {
				// 可以提示用户使用 sys 命令
				// fmt.Println("提示: 使用 'opsxcli sys' 获得更好的监控体验")
			}
			return exec.ForwardCommand("top", args)
		},
	}
}

func NewKillCmd() *cobra.Command {
	return createForwardCmd("kill", "终止进程", "发送信号到指定进程")
}

func NewPstreeCmd() *cobra.Command {
	return createForwardCmd("pstree", "显示进程树", "以树形结构显示进程关系")
}

// === 系统信息 ===

func NewUnameCmd() *cobra.Command {
	return createForwardCmd("uname", "显示系统信息", "显示系统名称和版本信息")
}

func NewHostnameCmd() *cobra.Command {
	return createForwardCmd("hostname", "显示或设置主机名", "显示或设置系统的主机名")
}

func NewWhoamiCmd() *cobra.Command {
	return createForwardCmd("whoami", "显示当前用户", "显示当前有效的用户名")
}

func NewIdCmd() *cobra.Command {
	return createForwardCmd("id", "显示用户和组信息", "显示用户的 UID、GID 和所属组")
}

func NewFreeCmd() *cobra.Command {
	return createForwardCmd("free", "显示内存使用情况", "显示系统内存和交换空间使用情况")
}

func NewDfCmd() *cobra.Command {
	return createForwardCmd("df", "显示磁盘空间", "显示文件系统的磁盘空间使用情况")
}

func NewDuCmd() *cobra.Command {
	return createForwardCmd("du", "显示目录大小", "显示目录或文件的磁盘使用情况")
}

func NewMountCmd() *cobra.Command {
	return createForwardCmd("mount", "挂载文件系统", "挂载文件系统到指定挂载点")
}

func NewUmountCmd() *cobra.Command {
	return createForwardCmd("umount", "卸载文件系统", "卸载已挂载的文件系统")
}

// === 时间日期 ===

func NewDateCmd() *cobra.Command {
	return createForwardCmd("date", "显示或设置日期时间", "显示或设置系统日期和时间")
}

func NewSleepCmd() *cobra.Command {
	return createForwardCmd("sleep", "延迟指定时间", "暂停指定的秒数")
}

func NewWatchCmd() *cobra.Command {
	return createForwardCmd("watch", "周期性执行命令", "定期执行命令并显示输出")
}

// === 网络配置 ===

func NewIfconfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ifconfig",
		Short: "网络接口配置",
		Long:  "显示网络接口配置信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.ShowInterfaces()
		},
	}
}

func NewRouteCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "route",
		Short:              "显示或修改路由表",
		Long:               "显示内核 IP 路由表",
		DisableFlagParsing: true, // 禁用参数解析
		RunE: func(cmd *cobra.Command, args []string) error {
			return busybox.ShowRoutes()
		},
	}
}

func NewIpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ip",
		Short: "网络配置工具",
		Long:  "显示网络配置信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 简化实现，只支持 ip addr show
			if len(args) > 0 && (args[0] == "addr" || args[0] == "address" || args[0] == "a") {
				return busybox.ShowIPAddr()
			}
			// 默认显示 addr
			return busybox.ShowIPAddr()
		},
	}
}

// === 网络传输 ===
// 注意: wget, telnet, nc 已经在 opsxcli 中实现，不需要再次定义

func NewFtpCmd() *cobra.Command {
	return createForwardCmd("ftp", "FTP 客户端", "文件传输协议客户端")
}

func NewTftpCmd() *cobra.Command {
	return createForwardCmd("tftp", "TFTP 客户端", "简单文件传输协议客户端")
}
