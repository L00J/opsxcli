package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"opsxcli/internal/agent/core"
	"opsxcli/internal/agent/safety"
	"opsxcli/internal/agent/session"
	"opsxcli/internal/agent/tools"
	"opsxcli/internal/llm"
)

func init() {
	RegisterCommand("agent", "AI", "AI运维助手", NewAgentCmd)
}

// NewAgentCmd 创建 agent 命令（统一入口）
func NewAgentCmd() *cobra.Command {
	var (
		provider       string
		safetyMode     string
		interactive    bool
		query          string
		debug          bool
		autoApprove    bool
		backgroundTasks bool
		resumeID       string
		listSessions   bool
		exportID       string
	)

	agentCmd := &cobra.Command{
		Use:           "agent [query]",
		Short:         "AI运维助手 - 使用自然语言解决运维问题",
		SilenceUsage:  true,  // 错误时不显示 Usage/Help
		SilenceErrors: false, // 显示错误信息
		Long: `agent - AI驱动的运维助手

使用自然语言描述你的运维问题，Agent会自动调用相关工具来帮你解决。

示例：
  opsxcli agent "查看根目录磁盘使用情况"
  opsxcli agent "查找所有监听80端口的进程"
  opsxcli agent "SSH到192.168.1.100查看nginx进程"
  opsxcli agent -i                          # 交互模式
  opsxcli agent --resume <session_id>       # 恢复会话
  opsxcli agent --list-sessions             # 列出历史会话`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 获取查询
			if !interactive && len(args) == 0 && query == "" && !listSessions && exportID == "" && resumeID == "" {
				return fmt.Errorf("请提供查询内容或使用 -i 进入交互模式")
			}

			if len(args) > 0 {
				query = strings.Join(args, " ")
			}

			// --list-sessions / --export 不需要 LLM 配置，提前处理
			if listSessions {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("获取用户主目录失败: %w", err)
				}
				sessionDir := filepath.Join(homeDir, ".opsxcli", "agent", "sessions")
				store, err := session.NewJSONLStore(sessionDir)
				if err != nil {
					return fmt.Errorf("创建会话存储失败: %w", err)
				}
				manager := session.NewManager(store)
				return runListSessions(manager)
			}

			if exportID != "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("获取用户主目录失败: %w", err)
				}
				sessionDir := filepath.Join(homeDir, ".opsxcli", "agent", "sessions")
				store, err := session.NewJSONLStore(sessionDir)
				if err != nil {
					return fmt.Errorf("创建会话存储失败: %w", err)
				}
				manager := session.NewManager(store)
				return runExportSession(manager, exportID)
			}

			// 1. 初始化 LLM 配置管理器
			configManager, err := llm.NewConfigManager("")
			if err != nil {
				return fmt.Errorf("初始化配置管理器失败: %w", err)
			}

			// 检查是否有有效配置（排除占位符 key）
			// 场景1: 完全没有配置 → 首次使用，启动配置向导
			// 场景2: 有配置但都是占位符 key → 引导重新配置
			// 场景3: 有有效配置 → 正常使用
			providers := configManager.List()
			validProviders := configManager.GetValidProviders()

			needsWizard := false
			if len(providers) == 0 {
				// 场景1: 完全新环境
				needsWizard = true
			} else if len(validProviders) == 0 {
				// 场景2: 有配置但 key 全是占位符
				fmt.Println()
				fmt.Println(color.YellowString("⚠️  检测到 API 密钥未配置"))
				fmt.Println(color.New(color.FgHiBlack).Sprint("   现有配置使用的是占位符密钥，需要配置真实密钥才能使用"))
				fmt.Println()
				needsWizard = true
			}

			if needsWizard {
				fmt.Println(color.CyanString("🔧 启动配置向导..."))
				fmt.Println()
				wizard := llm.NewConfigWizard(configManager)
				if err := wizard.Run(); err != nil {
					return fmt.Errorf("配置向导失败: %w", err)
				}

				// 重新加载配置
				configManager, err = llm.NewConfigManager("")
				if err != nil {
					return fmt.Errorf("重新加载配置失败: %w", err)
				}
				validProviders = configManager.GetValidProviders()

				if len(validProviders) == 0 {
					return fmt.Errorf("配置失败，请重试")
				}
			}

			// 如果未指定 provider，优先使用默认 provider，其次使用第一个有有效 key 的 provider
			if provider == "" {
				// 优先使用用户设置的默认 provider
				if defaultProvider := configManager.GetDefaultProvider(); defaultProvider != "" {
					// 验证默认 provider 是否在有效列表中
					for _, vp := range validProviders {
						if vp == defaultProvider {
							provider = defaultProvider
							break
						}
					}
				}
				// 如果默认 provider 无效或未设置，使用第一个有效 provider
				if provider == "" && len(validProviders) > 0 {
					provider = validProviders[0]
				}
			}

			// 2. 创建 LLM 客户端
			factory := llm.NewClientFactory(configManager)
			llmClient, err := factory.Create(provider)
			if err != nil {
				return fmt.Errorf("创建LLM客户端失败: %w", err)
			}

			// 3. 创建工具注册中心
			registry := tools.NewRegistry()
			registry.RegisterDefaults()

			// 4. 创建安全配置
			var mode safety.SafetyMode
			switch safetyMode {
			case "strict":
				mode = safety.SafetyModeStrict
			case "permissive":
				mode = safety.SafetyModePermissive
			default:
				mode = safety.SafetyModeBalanced
			}
			safetyCtl := safety.NewController(mode)
			if autoApprove {
				safetyCtl.SetAutoApprove(true)
			}

			// 5. 创建会话目录和管理器
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("获取用户主目录失败: %w", err)
			}
			sessionDir := filepath.Join(homeDir, ".opsxcli", "agent", "sessions")
			if err := os.MkdirAll(sessionDir, 0700); err != nil {
				return fmt.Errorf("创建会话目录失败: %w", err)
			}

			store, err := session.NewJSONLStore(sessionDir)
			if err != nil {
				return fmt.Errorf("创建会话存储失败: %w", err)
			}
			manager := session.NewManager(store)

			// 6. 创建 Agent 配置
			agentConfig := core.LoadConfig()
			if agentConfig == nil {
				agentConfig = core.NewDefaultConfig()
			}
			agentConfig.SessionDir = sessionDir
			agentConfig.AutoApprove = autoApprove
			if debug {
				agentConfig.MaxIterations = 20 // debug 模式下减少迭代次数便于观察
			}

			// 7. 创建 Agent
			ag := core.NewAgent(llmClient, registry, agentConfig, safetyCtl)
			defer ag.Close()

			// 8. 处理不同运行模式

			// 恢复会话模式
			if resumeID != "" {
				return runResumeSession(ag, manager, resumeID, args, provider)
			}

			// 交互模式
			if interactive || query == "" {
				if backgroundTasks {
					return runInteractiveWithTasks(ag, manager, provider)
				}
				return runInteractive(ag, manager, provider, debug)
			}

			// 单次查询模式
			return runSingleQuery(ag, manager, query, provider, debug)
		},
	}

	agentCmd.Flags().StringVarP(&provider, "provider", "p", "", "LLM提供商（deepseek, ollama, claude等）")
	agentCmd.Flags().StringVarP(&safetyMode, "safety", "s", "balanced", "安全模式（strict, balanced, permissive）")
	agentCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "交互模式")
	agentCmd.Flags().StringVarP(&query, "query", "q", "", "查询内容")
	agentCmd.Flags().BoolVarP(&debug, "debug", "d", false, "调试模式,显示详细的工具调用信息")
	agentCmd.Flags().BoolVarP(&autoApprove, "yes", "y", false, "自动批准所有操作,无需确认")
	agentCmd.Flags().BoolVarP(&backgroundTasks, "background", "b", false, "启用后台任务管理,支持多任务并发执行")
	agentCmd.Flags().StringVar(&resumeID, "resume", "", "恢复指定会话")
	agentCmd.Flags().BoolVar(&listSessions, "list-sessions", false, "列出历史会话")
	agentCmd.Flags().StringVar(&exportID, "export", "", "导出会话为Markdown")

	return agentCmd
}

// runInteractive 运行交互模式（使用 Bubble Tea TUI）
func runInteractive(ag *core.Agent, manager session.Manager, provider string, debug bool) error {
	// 创建新会话
	sess, err := manager.Create("交互会话", provider, "")
	if err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}
	_ = sess

	ctx := context.Background()
	return ag.RunInteractive(ctx)
}

// runInteractiveWithTasks 运行支持后台任务的交互模式
func runInteractiveWithTasks(ag *core.Agent, manager session.Manager, provider string) error {
	fmt.Println()
	fmt.Println(color.CyanString("🤖 opsxcli Agent — 后台任务模式"))
	fmt.Println(color.New(color.FgHiBlack).Sprint("   支持多个任务并发执行"))
	fmt.Println()
	fmt.Println("   命令:")
	fmt.Println("     /exit, /quit  - 退出会话")
	fmt.Println("     /tasks        - 查看任务列表")
	fmt.Println("     /switch       - 切换当前任务")
	fmt.Println()

	// 简化版后台任务：顺序执行，但支持任务列表查看
	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	var tasks []struct {
		id       string
		desc     string
		query    string
		status   string
		output   string
		err      error
	}

	for {
		fmt.Print(color.GreenString("👤 您") + ": ")

		input, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println()
				fmt.Println(color.YellowString("👋 再见！"))
				return nil
			}
			return fmt.Errorf("读取输入失败: %w", err)
		}

		input = strings.TrimSpace(input)

		switch input {
		case "/exit", "/quit":
			fmt.Println(color.YellowString("👋 再见！"))
			return nil
		case "/help":
			showAgentHelp(ag)
			continue
		case "/tasks":
			if len(tasks) == 0 {
				fmt.Println("暂无任务")
			} else {
				fmt.Println()
				fmt.Println(color.CyanString("📋 任务列表"))
				for i, t := range tasks {
					statusIcon := "⏳"
					if t.status == "completed" {
						statusIcon = "✅"
					} else if t.status == "failed" {
						statusIcon = "❌"
					}
					fmt.Printf("  %s %d. %s (%s)\n", statusIcon, i+1, t.desc, t.status)
				}
				fmt.Println()
			}
			continue
		case "":
			continue
		}

		// 创建任务
		taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
		desc := input
		if len(desc) > 40 {
			desc = desc[:40] + "..."
		}

		tasks = append(tasks, struct {
			id     string
			desc   string
			query  string
			status string
			output string
			err    error
		}{
			id:     taskID,
			desc:   desc,
			query:  input,
			status: "running",
		})

		fmt.Printf("✓ 任务 #%d 已创建: %s\n", len(tasks), desc)
		fmt.Println(color.CyanString("🤖 Agent") + ": " + color.New(color.FgHiBlack).Sprint("思考中..."))
		fmt.Println()

		result, err := ag.Run(ctx, input)

		// 更新任务状态
		taskIdx := len(tasks) - 1
		if err != nil {
			tasks[taskIdx].status = "failed"
			tasks[taskIdx].err = err
			fmt.Println(color.RedString("❌ 错误: ") + err.Error())
		} else {
			tasks[taskIdx].status = "completed"
			tasks[taskIdx].output = result.Output
			fmt.Println(color.CyanString("🤖 Agent") + ":")
			fmt.Println(result.Output)
		}
		fmt.Println()
	}
}

// runSingleQuery 运行单次查询
func runSingleQuery(ag *core.Agent, manager session.Manager, query, provider string, debug bool) error {
	// 创建会话
	sess, err := manager.Create(query, provider, "")
	if err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}
	_ = sess

	fmt.Printf("[%s] %s\n\n", color.CyanString(provider), color.GreenString(query))

	if debug {
		fmt.Println(color.New(color.FgYellow).Sprint("[DEBUG] 调试模式已开启"))
		fmt.Println()
	}

	fmt.Println(color.CyanString("🤖 Agent") + ": " + color.New(color.FgHiBlack).Sprint("思考中..."))
	fmt.Println()

	ctx := context.Background()

	// 设置工具执行回调，实时显示工具调用过程
	ag.SetToolCallback(func(name string, args map[string]interface{}, start bool, success bool, duration time.Duration, output string) {
		if start {
			// 工具开始执行
			var detail string
			switch name {
			case "local_bash", "bash", "shell":
				if cmd, ok := args["command"].(string); ok {
					detail = fmt.Sprintf("执行命令: %s", cmd)
				} else {
					detail = formatToolCallDetail(name, args)
				}
			case "ssh_execute", "remote_bash":
				host, _ := args["host"].(string)
				cmd, _ := args["command"].(string)
				if host != "" && cmd != "" {
					detail = fmt.Sprintf("SSH %s → %s", host, cmd)
				} else {
					detail = formatToolCallDetail(name, args)
				}
			default:
				detail = formatToolCallDetail(name, args)
			}
			fmt.Printf("  🔧 %s\n", color.YellowString(detail))
		} else {
			// 工具执行完成
			status := "✅"
			if !success {
				status = "❌"
			}
			if output != "" {
				preview := truncateCmdOutput(output, 3)
				fmt.Printf("  %s %s (%v)\n    %s\n", status, name, duration.Round(time.Millisecond), color.HiBlackString(preview))
			} else {
				fmt.Printf("  %s %s (%v)\n", status, name, duration.Round(time.Millisecond))
			}
		}
	})

	result, err := ag.Run(ctx, query)
	if err != nil {
		return fmt.Errorf("Agent 执行失败: %w", err)
	}

	fmt.Println(color.CyanString("🤖 Agent") + ":")
	fmt.Println(result.Output)
	fmt.Println()

	return nil
}

// runListSessions 列出历史会话
func runListSessions(manager session.Manager) error {
	sessions, err := manager.List(50)
	if err != nil {
		return fmt.Errorf("列出会话失败: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("暂无历史会话")
		return nil
	}

	fmt.Println()
	fmt.Println(color.CyanString("📋 历史会话列表"))
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("%-20s %-18s %-12s %-20s %s\n",
		color.YellowString("会话ID"),
		color.YellowString("标题"),
		color.YellowString("提供商"),
		color.YellowString("更新时间"),
		color.YellowString("消息数"),
	)
	fmt.Println(strings.Repeat("─", 80))

	for _, sess := range sessions {
		title := sess.Title
		if len(title) > 16 {
			title = title[:13] + "..."
		}
		provider := sess.Provider
		if provider == "" {
			provider = "-"
		}
		timeStr := sess.UpdatedAt.Format("01-02 15:04")
		if time.Now().Sub(sess.UpdatedAt) < 24*time.Hour {
			timeStr = sess.UpdatedAt.Format("15:04")
		}

		fmt.Printf("%-20s %-18s %-12s %-20s %d\n",
			sess.ID,
			title,
			provider,
			timeStr,
			sess.MessageCount,
		)
	}
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("共 %d 条会话\n", len(sessions))
	fmt.Println()

	return nil
}

// runExportSession 导出会话为 Markdown
func runExportSession(manager session.Manager, sessionID string) error {
	markdown, err := manager.ExportMarkdown(sessionID)
	if err != nil {
		return fmt.Errorf("导出会话失败: %w", err)
	}

	fmt.Println(markdown)
	return nil
}

// runResumeSession 恢复指定会话并继续
func runResumeSession(ag *core.Agent, manager session.Manager, sessionID string, args []string, provider string) error {
	sess, messages, err := manager.Load(sessionID)
	if err != nil {
		return fmt.Errorf("恢复会话失败: %w", err)
	}

	fmt.Printf("🔄 恢复会话: %s (%s)\n", sess.Title, sess.ID)
	fmt.Printf("   历史消息: %d 条\n", len(messages))
	fmt.Println()

	// 如果有额外参数，作为新查询执行
	if len(args) > 0 {
		query := strings.Join(args, " ")
		fmt.Printf("[%s] %s\n\n", color.CyanString(sess.Provider), color.GreenString(query))
		fmt.Println(color.CyanString("🤖 Agent") + ": " + color.New(color.FgHiBlack).Sprint("思考中..."))
		fmt.Println()

		ctx := context.Background()
		result, err := ag.Run(ctx, query)
		if err != nil {
			return fmt.Errorf("Agent 执行失败: %w", err)
		}

		fmt.Println(color.CyanString("🤖 Agent") + ":")
		fmt.Println(result.Output)
		fmt.Println()
		return nil
	}

	// 否则进入交互模式
	fmt.Println(color.CyanString("🤖 opsxcli Agent [恢复会话]"))
	fmt.Println(color.New(color.FgHiBlack).Sprint("   输入 /help 查看帮助, /exit 退出"))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Print(color.GreenString("👤 您") + ": ")

		input, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println()
				fmt.Println(color.YellowString("👋 再见！"))
				return nil
			}
			return fmt.Errorf("读取输入失败: %w", err)
		}

		input = strings.TrimSpace(input)

		switch input {
		case "/exit", "/quit":
			fmt.Println(color.YellowString("👋 再见！"))
			return nil
		case "/help":
			showAgentHelp(ag)
			continue
		case "":
			continue
		}

		fmt.Println(color.CyanString("🤖 Agent") + ": " + color.New(color.FgHiBlack).Sprint("思考中..."))
		fmt.Println()

		result, err := ag.Run(ctx, input)
		if err != nil {
			fmt.Println(color.RedString("❌ 错误: ") + err.Error())
			fmt.Println()
			continue
		}

		fmt.Println(color.CyanString("🤖 Agent") + ":")
		fmt.Println(result.Output)
		fmt.Println()
	}
}

// showAgentHelp 显示帮助信息
func showAgentHelp(ag *core.Agent) {
	fmt.Println()
	fmt.Println(color.CyanString("📖 Agent 帮助"))
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println("交互命令：")
	fmt.Println("  /help        显示此帮助")
	fmt.Println("  /exit        退出交互模式")
	fmt.Println("  /quit        同 /exit")
	fmt.Println("  /tasks       查看任务列表（后台模式）")
	fmt.Println()
	fmt.Println("用法示例：")
	fmt.Println("  opsxcli agent \"查看磁盘使用情况\"")
	fmt.Println("  opsxcli agent -i")
	fmt.Println("  opsxcli agent -p deepseek \"查看进程\"")
	fmt.Println("  opsxcli agent -b                    # 后台任务模式")
	fmt.Println("  opsxcli agent --resume sess_xxx")
	fmt.Println("  opsxcli agent --list-sessions")
	fmt.Println("  opsxcli agent --export sess_xxx > session.md")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()
}

// formatToolCallDetail 格式化工具调用详情为可读字符串
func formatToolCallDetail(name string, args map[string]interface{}) string {
	if len(args) == 0 {
		return name
	}
	parts := make([]string, 0, len(args))
	for k, v := range args {
		if k == "_i" || k == "_intent" {
			continue
		}
		s := fmt.Sprintf("%v", v)
		if len(s) > 80 {
			s = s[:80] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, s))
	}
	return fmt.Sprintf("%s(%s)", name, strings.Join(parts, ", "))
}

// truncateCmdOutput 截断命令输出到指定行数
func truncateCmdOutput(output string, maxLines int) string {
	lines := strings.Split(output, "\n")
	var nonEmpty []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}
	if len(nonEmpty) <= maxLines {
		return strings.Join(nonEmpty, "\n    ")
	}
	return strings.Join(nonEmpty[:maxLines], "\n    ") +
		fmt.Sprintf("\n    ... (%d more lines)", len(nonEmpty)-maxLines)
}
