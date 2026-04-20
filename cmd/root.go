package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"opsxcli/internal/logger"
	"opsxcli/internal/output"
)

// NewRootCmd 创建根命令
func NewRootCmd(version string) *cobra.Command {
	var (
		upgradeFlag bool
		quietFlag   bool
		outputFlag  string
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
			DisableDefaultCmd: false, // 启用 completion 子命令（通过 cmd/completion.go 自定义实现）
			HiddenDefaultCmd:  true,  // 隐藏 Cobra 默认的 completion 命令
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
				// 检测是否已完成首次配置（有 providers 目录且内有 .json 文件）
				homeDir, _ := os.UserHomeDir()
				if homeDir != "" {
					providersDir := homeDir + "/.opsxcli/providers"
					configured := false
					if entries, err := os.ReadDir(providersDir); err == nil {
						for _, e := range entries {
							if !e.IsDir() && len(e.Name()) > 5 && e.Name()[len(e.Name())-5:] == ".json" {
								configured = true
								break
							}
						}
					}
					if !configured {
						fmt.Println()
						fmt.Println(color.CyanString("🤖 欢迎使用 OpsX CLI - AI 智能运维助手"))
						fmt.Println()
						fmt.Println(color.YellowString("⚠️  未检测到 LLM 配置"))
						fmt.Println()
						fmt.Println("  首次使用请先配置 LLM 提供商：")
						fmt.Println()
						fmt.Printf("    %s\n\n", color.GreenString("opsxcli setup"))
						return nil
					}
				}
				if err := cmd.Help(); err != nil {
					return err
				}
			}
			return nil
		},
	}

	rootCmd.Flags().BoolVar(&upgradeFlag, "upgrade", false, "升级到最新版本")
	rootCmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "静默模式，抑制非必要输出")
	rootCmd.Flags().StringVar(&outputFlag, "output", "text", "输出格式 (text/json)")

	// 动态注册已自注册的命令
	var names []string
	for name := range commandRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		rootCmd.AddCommand(commandRegistry[name].Factory())
	}

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

	// 定义命令分组
	type commandGroup struct {
		name     string
		commands []string
		desc     string
	}

	// 收集并分类命令
	categoryCommands := make(map[string][]string)
	categoryDescs := make(map[string][]string)

	for _, subCmd := range cmd.Commands() {
		if subCmd.Hidden {
			continue
		}
		if entry, ok := commandRegistry[subCmd.Name()]; ok {
			categoryCommands[entry.Category] = append(categoryCommands[entry.Category], subCmd.Name())
			categoryDescs[entry.Category] = append(categoryDescs[entry.Category], entry.Description)
		} else {
			categoryCommands["其他"] = append(categoryCommands["其他"], subCmd.Name())
		}
	}

	categoryOrder := []string{"AI", "数据库", "网络", "监控", "文件", "系统", "Docker", "Kubernetes", "工具", "管理", "其他"}

	groups := make([]commandGroup, 0)
	for _, cat := range categoryOrder {
		cmds, ok := categoryCommands[cat]
		if !ok || len(cmds) == 0 {
			continue
		}
		desc := ""
		if cat == "其他" {
			desc = "其他命令"
		} else {
			seen := make(map[string]bool)
			uniqueDescs := make([]string, 0)
			for _, d := range categoryDescs[cat] {
				if !seen[d] {
					seen[d] = true
					uniqueDescs = append(uniqueDescs, d)
				}
			}
			desc = strings.Join(uniqueDescs, ", ")
		}
		groups = append(groups, commandGroup{name: cat, commands: cmds, desc: desc})
	}

	// 打印分组命令（紧凑格式）
	fmt.Println("Commands:")
	for _, group := range groups {
		fmt.Printf("\n  %s:  %s\n", color.YellowString(group.name), color.New(color.FgHiBlack).Sprint(group.desc))
		fmt.Printf("    %s\n", color.CyanString(formatCommandList(group.commands)))
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
	return strings.Join(commands, ", ")
}
