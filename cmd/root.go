package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"opsxcli/internal/logger"
	"opsxcli/internal/output"
)

// NewRootCmd 创建根命令
func NewRootCmd(version string) *cobra.Command {
	var (
		upgradeFlag bool
		quietFlag  bool
		outputFlag string
	)

	rootCmd := &cobra.Command{
		Use:   "opsxcli",
		Short: "运维瑞士军刀 | 一站式命令行工具集",
		Long: `opsxcli - 面向运维和开发的集成化命令行工具集

内置数据库连接、网络调试、系统监控、文件传输等常用功能。
无需切换多种客户端，一条命令即可操作 MySQL、Redis、SSH、HTTP 等服务。

使用 'opsxcli <command> --help' 查看具体命令的帮助信息。`,
		Version: version,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true, // 禁用 completion 命令
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if outputFlag == "json" {
				output.SetFormat(output.FormatJSON)
			}
			if quietFlag {
				output.SetQuiet(true)
				logger.SetQuiet(true)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if upgradeFlag {
				return NewUpgradeCmd().RunE(cmd, args)
			}
			if !output.IsQuiet() {
				cmd.Help()
			}
			return nil
		},
	}

	rootCmd.Flags().BoolVar(&upgradeFlag, "upgrade", false, "升级到最新版本")
	rootCmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "静默模式，抑制非必要输出")
	rootCmd.Flags().StringVar(&outputFlag, "output", "text", "输出格式 (text/json)")

	// 添加所有子命令（按类型分组排序）
	// 数据库工具
	rootCmd.AddCommand(
		NewMySQLCmd(),
		NewPsqlCmd(),
		NewRedisCmd(),
	)
	// 网络工具
	rootCmd.AddCommand(
		NewSSHCmd(),
		NewTelnetCmd(),
		NewNcCmd(),
		NewPingCmd(),
		NewTracerouteCmd(),
		NewNetstatCmd(),
		NewSsCmd(),
		NewNmapCmd(),
	)
	// 监控工具
	rootCmd.AddCommand(
		NewSysCmd(),
		NewNetCmd(),
	)
	// 服务端工具
	rootCmd.AddCommand(
		NewServerCmd(),
	)
	// 其他工具
	rootCmd.AddCommand(
		NewCurlCmd(),
		NewWgetCmd(),
		NewRequestCmd(),
		NewWebSearchCmd(), // Web搜索
		NewDockerCmd(),
		NewKubectlCmd(),
		NewConsulCmd(),
		NewKubernetesCmd(),
		NewInstallCmd(),
		NewUpgradeCmd(),
		NewTestCmd(), // 测试命令
	)
	// AI 工具
	rootCmd.AddCommand(
		NewAgentCmd(),
		NewSessionCmd(),
	)

	// === Busybox 兼容命令 ===
	// 基本文件操作（Go 原生实现，跨平台兼容）
	rootCmd.AddCommand(
		NewLsCmd(),
		NewCpCmd(),
		NewMvCmd(),
		NewRmCmd(),
		NewMkdirCmd(),
		NewRmdirCmd(),
		// NewTreeCmd(),     // 系统已有 tree，移除无价值转发
		NewTouchCmd(),
		NewChmodCmd(),
		NewChownCmd(),
		NewLnCmd(),
	)
	// 文件查看/编辑
	rootCmd.AddCommand(
		NewCatCmd(),
		// NewMoreCmd(),    // 系统已有 more，移除无价值转发
		// NewLessCmd(),    // 系统已有 less，移除无价值转发
		NewHeadCmd(),
		NewTailCmd(),
		NewGrepCmd(),
		// NewAwkCmd(),     // 系统已有 awk，移除无价值转发
		// NewSedCmd(),     // 系统已有 sed，移除无价值转发
		// NewViCmd(),      // 系统已有 vi，移除无价值转发
		// NewVimCmd(),     // 系统已有 vim，移除无价值转发
	)
	// 归档压缩（系统已有，移除无价值转发）
	// rootCmd.AddCommand(
	// 	NewTarCmd(),
	// 	NewGzipCmd(),
	// 	NewUnzipCmd(),
	// )
	// 进程管理（Go 原生 ps/top/kill，保留；pstree 转发移除）
	rootCmd.AddCommand(
		NewPsCmd(),
		NewTopCmd(),
		NewKillCmd(),
		// NewPstreeCmd(),  // 系统已有 pstree，移除无价值转发
	)
	// 系统信息（系统已有，移除无价值转发）
	// rootCmd.AddCommand(
	// 	NewUnameCmd(), NewHostnameCmd(), NewWhoamiCmd(),
	// 	NewIdCmd(), NewFreeCmd(), NewDfCmd(), NewDuCmd(),
	// 	NewMountCmd(), NewUmountCmd(),
	// )
	// 磁盘工具（dd 保留）
	rootCmd.AddCommand(
		NewDdCmd(),
	)
	// 时间日期（系统已有，移除无价值转发）
	// rootCmd.AddCommand(
	// 	NewDateCmd(), NewSleepCmd(), NewWatchCmd(),
	// )
	// 网络配置（Go 原生跨平台实现，保留）
	rootCmd.AddCommand(
		NewIfconfigCmd(),
		NewRouteCmd(),
		NewIpCmd(),
	)
	// 网络传输（系统已有，移除无价值转发）
	// rootCmd.AddCommand(
	// 	NewFtpCmd(),
	// 	NewTftpCmd(),
	// )

	// 自定义版本输出
	rootCmd.SetVersionTemplate(fmt.Sprintf("%s: %s\n", color.CyanString("opsxcli"), version))

	// 自定义帮助模板，按类型分组显示命令
	rootCmd.SetHelpFunc(customHelpFunc)

	return rootCmd
}

// customHelpFunc 自定义帮助函数，按类型分组显示命令
func customHelpFunc(cmd *cobra.Command, args []string) {
	// quiet 模式下不显示帮助
	if output.IsQuiet() {
		return
	}
	// 只对根命令使用自定义帮助,子命令使用默认帮助
	if cmd.Parent() != nil {
		// 对于子命令,打印默认帮助
		cmd.OutOrStdout().Write([]byte(cmd.UsageString()))
		return
	}

	// 打印标准帮助头部
	fmt.Printf("%s\n\n", cmd.Long)
	fmt.Printf("Usage:\n  %s [command]\n\n", cmd.Use)

	// 定义命令分组（精简）
	type commandGroup struct {
		name     string
		commands []string
		desc     string // 分组描述
	}

	groups := []commandGroup{
		// === 核心工具（最常用） ===
		{"文件", []string{"ls", "cat", "grep", "vi", "cp", "mv", "rm", "mkdir", "tree"}, "文件和目录操作"},
		{"数据库", []string{"mysql", "psql", "redis"}, "MySQL, PostgreSQL, Redis"},
		{"系统", []string{"ps", "top", "free", "df", "du", "uname", "hostname"}, "进程和系统信息"},
		{"磁盘", []string{"dd"}, "磁盘读写和数据转换"},
		{"", []string{}, ""}, // 空行分隔
		// === 网络工具 ===
		{"网络", []string{"ssh", "ping", "traceroute", "telnet", "nc", "ss", "nmap"}, "SSH, Ping, 端口扫描等"},
		{"网络配置", []string{"ifconfig", "route", "ip"}, "网络接口和路由管理"},
		{"", []string{}, ""}, // 空行分隔
		// === 其他工具 ===
		{"压缩", []string{"tar", "gzip", "unzip"}, "归档和压缩工具"},
		{"服务", []string{"server"}, "HTTP/WebSocket/gRPC 服务"},
		{"工具", []string{"curl", "wget", "request", "websearch"}, "HTTP请求, 文件下载, Web搜索"},
		{"", []string{}, ""}, // 空行分隔
		// === Docker 工具 ===
		{"Docker", []string{"docker"}, "镜像管理, 多源极速下载"},
		{"", []string{}, ""}, // 空行分隔
		// === Kubernetes 工具 ===
		{"Kubernetes", []string{"kubectl", "consul", "kubernetes"}, "kubectl 命令行, 服务注册, 资源管理"},
		// === AI 工具 ===
		{"AI", []string{"agent", "agentv2", "session"}, "AI 运维助手, 会话管理"},
		{"", []string{}, ""}, // 空行分隔
		// === 监控工具 ===
		{"监控", []string{"sys", "net"}, "系统监控, 网络监控 (2秒实时刷新)"},
		{"", []string{}, ""}, // 空行分隔
		// === 管理工具 ===
		{"管理", []string{"install", "upgrade"}, "安装系统服务, 升级opsxcli"},
	}

	// 打印分组命令（紧凑格式）
	fmt.Println("Commands:")
	for _, group := range groups {
		// 跳过空行分隔符
		if group.name == "" {
			fmt.Println()
			continue
		}

		validCommands := make([]*cobra.Command, 0)
		for _, cmdName := range group.commands {
			if subCmd, _, err := cmd.Find([]string{cmdName}); err == nil && subCmd != cmd {
				validCommands = append(validCommands, subCmd)
			}
		}

		if len(validCommands) > 0 {
			// 分组标题 + 描述
			fmt.Printf("\n  %s:  %s\n", color.YellowString(group.name), color.New(color.FgHiBlack).Sprint(group.desc))

			// 命令列表（紧凑）
			cmdNames := make([]string, 0)
			for _, subCmd := range validCommands {
				cmdNames = append(cmdNames, subCmd.Name())
			}
			fmt.Printf("    %s\n", color.CyanString(formatCommandList(cmdNames)))
		}
	}

	// 底部提示
	fmt.Printf("\n%s:\n", color.GreenString("使用提示"))
	fmt.Printf("  • 查看命令帮助: %s\n", color.CyanString("opsxcli <command> --help"))
	fmt.Printf("  • AI运维助手: %s\n", color.CyanString("opsxcli agent -i"))
	fmt.Printf("  • 升级到最新版: %s 或 %s\n", color.CyanString("opsxcli --upgrade"), color.CyanString("opsxcli upgrade"))
	fmt.Printf("  • 系统监控TUI: %s\n", color.CyanString("opsxcli sys"))
	fmt.Printf("  • 网络监控TUI: %s\n", color.CyanString("opsxcli net"))
	fmt.Printf("  • 快速端口扫描: %s\n", color.CyanString("opsxcli nmap <host>"))
	fmt.Printf("  • SSH连接: %s\n", color.CyanString("opsxcli ssh user@host"))
	fmt.Printf("  • MySQL连接: %s\n", color.CyanString("opsxcli mysql -u root -h localhost"))
	fmt.Printf("  • Redis连接: %s\n", color.CyanString("opsxcli redis -h 127.0.0.1"))
}

// formatCommandList 格式化命令列表为紧凑格式
func formatCommandList(commands []string) string {
	result := ""
	for i, cmd := range commands {
		if i > 0 {
			result += ", "
		}
		result += cmd
	}
	return result
}
