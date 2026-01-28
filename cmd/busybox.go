package cmd

import (
	"fmt"
	"strings"

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
		Use:   "ip [ OPTIONS ] OBJECT { COMMAND | help }",
		Short: "显示/操作路由、网络设备、接口和隧道",
		Long: `ip - 显示/操作路由、网络设备、接口和隧道

用法: ip [ OPTIONS ] OBJECT { COMMAND | help }
      ip [ -force ] -batch filename

OBJECT := { link | address | route | help }
OPTIONS := { -V[ersion] | -h[uman-readable] | -s[tatistics] |
             -r[esolve] | -f[amily] { inet | inet6 } |
             -4 | -6 | -o[neline] | -br[ief] }

常用命令:
  ip addr           显示所有网络接口的 IP 地址
  ip addr show      显示所有网络接口的 IP 地址
  ip link           显示所有网络接口信息
  ip link show      显示所有网络接口信息
  ip route          显示路由表
  ip route show     显示路由表

简写形式:
  ip a              等同于 ip addr
  ip l              等同于 ip link
  ip r              等同于 ip route`,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleIPCommand(args)
		},
	}
}

// handleIPCommand 处理 ip 命令的各种子命令
func handleIPCommand(args []string) error {
	// 没有参数，默认显示 addr
	if len(args) == 0 {
		return busybox.ShowIPAddr()
	}

	// 过滤掉选项参数（以 - 开头的）
	var object string
	for i, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			// 显示帮助信息
			fmt.Println(getIPHelpText())
			return nil
		}
		if !strings.HasPrefix(arg, "-") {
			object = arg
			break
		}
		_ = i // 避免未使用变量警告
	}

	// 处理对象别名
	switch object {
	case "a", "add", "addr", "address":
		return busybox.ShowIPAddr()
	case "l", "link":
		return busybox.ShowInterfaces()
	case "r", "route":
		return busybox.ShowRoutes()
	case "help", "":
		fmt.Println(getIPHelpText())
		return nil
	default:
		return fmt.Errorf("对象 \"%s\" 未知，请尝试 \"ip help\"", object)
	}
}

// getIPHelpText 返回 ip 命令的帮助文本
func getIPHelpText() string {
	return `用法: ip [ OPTIONS ] OBJECT { COMMAND | help }
      ip [ -force ] -batch filename

OBJECT := { link | address | route | help }
OPTIONS := { -V[ersion] | -h[uman-readable] | -s[tatistics] |
             -r[esolve] | -f[amily] { inet | inet6 } |
             -4 | -6 | -o[neline] | -br[ief] }

常用命令:
  ip addr           显示所有网络接口的 IP 地址
  ip addr show      显示所有网络接口的 IP 地址
  ip link           显示所有网络接口信息
  ip link show      显示所有网络接口信息
  ip route          显示路由表
  ip route show     显示路由表

简写形式:
  ip a              等同于 ip addr
  ip l              等同于 ip link
  ip r              等同于 ip route`
}

// === 网络传输 ===
// 注意: wget, telnet, nc 已经在 opsxcli 中实现，不需要再次定义

func NewFtpCmd() *cobra.Command {
	return createForwardCmd("ftp", "FTP 客户端", "文件传输协议客户端")
}

func NewTftpCmd() *cobra.Command {
	return createForwardCmd("tftp", "TFTP 客户端", "简单文件传输协议客户端")
}

// === 磁盘工具 ===

func NewDdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dd [options]",
		Short: "转换和复制文件",
		Long: `转换和复制文件,支持底层数据复制

参数格式: key=value

常用选项:
  if=FILE         输入文件 (默认: stdin)
  of=FILE         输出文件 (默认: stdout)
  bs=BYTES        块大小 (默认: 512)
  count=N         复制 N 个块
  skip=N          跳过输入文件的前 N 个块
  seek=N          跳过输出文件的前 N 个块
  conv=CONVS      转换选项 (notrunc: 不截断输出文件)
  status=LEVEL    显示级别 (progress, noxfer, none)

大小单位: K (1024), M (1024*1024), G (1024*1024*1024)

示例:
  # 创建 100MB 的空文件
  opsxcli dd if=/dev/zero of=test.img bs=1M count=100

  # 复制文件
  opsxcli dd if=input.bin of=output.bin bs=4K

  # 备份磁盘分区
  opsxcli dd if=/dev/sda1 of=backup.img bs=1M status=progress

  # 创建引导盘
  opsxcli dd if=ubuntu.iso of=/dev/sdb bs=4M status=progress

  # 擦除磁盘数据
  opsxcli dd if=/dev/zero of=/dev/sdb bs=1M count=1024

  # 从文件中读取特定位置的数据
  opsxcli dd if=data.bin of=output.bin bs=512 skip=10 count=20`,
		DisableFlagParsing: true, // 禁用标准参数解析,使用自定义格式
		RunE: func(c *cobra.Command, args []string) error {
			// 处理帮助请求
			if len(args) > 0 && (args[0] == "-h" || args[0] == "--help" || args[0] == "help") {
				fmt.Println(c.Long)
				return nil
			}

			opts, err := busybox.ParseDdArgs(args)
			if err != nil {
				return fmt.Errorf("参数错误: %v", err)
			}
			return busybox.Dd(opts)
		},
	}
	return cmd
}
