package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"opsxcli/internal/agent"
	"opsxcli/internal/db"
	"opsxcli/internal/llm"
	"opsxcli/internal/tools"
)

// NewAgentCmd 创建agent命令
func NewAgentCmd() *cobra.Command {
	var (
		provider       string
		safetyMode     string
		interactive    bool
		query          string
		debug          bool
		autoApprove    bool
		backgroundTasks bool
	)

	agentCmd := &cobra.Command{
		Use:   "agent [query]",
		Short: "AI运维助手 - 使用自然语言解决运维问题",
		Long: `agent - AI驱动的运维助手

使用自然语言描述你的运维问题，Agent会自动调用相关工具来帮你解决。

示例：
  opsxcli agent "查看根目录磁盘使用情况"
  opsxcli agent "查找所有监听80端口的进程"
  opsxcli agent "查看default命名空间下的所有pods"
  opsxcli agent -i  # 交互模式`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 获取查询
			if !interactive && len(args) == 0 && query == "" {
				return fmt.Errorf("请提供查询内容或使用 -i 进入交互模式")
			}

			if len(args) > 0 {
				query = strings.Join(args, " ")
			}

			// 初始化数据库
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("获取用户主目录失败: %w", err)
			}

			dataDir := homeDir + "/.opsxcli"
			database, err := db.NewDB(dataDir)
			if err != nil {
				return fmt.Errorf("初始化数据库失败: %w", err)
			}
			defer database.Close()

			// 初始化LLM配置管理器
			configManager, err := llm.NewConfigManager("")
			if err != nil {
				return fmt.Errorf("初始化配置管理器失败: %w", err)
			}

			// 检查是否有配置，如果没有则运行配置向导
			providers := configManager.List()
			if len(providers) == 0 {
				fmt.Println("🔧 首次使用，启动配置向导...")
				fmt.Println()
				wizard := llm.NewConfigWizard(configManager)
				if err := wizard.Run(); err != nil {
					return fmt.Errorf("配置向导失败: %w", err)
				}

				// 重新加载配置
				configManager, _ = llm.NewConfigManager("")
				providers = configManager.List()

				// 如果还是没有配置，退出
				if len(providers) == 0 {
					return fmt.Errorf("配置失败，请重试")
				}
			}

			// 如果未指定provider，使用第一个
			if provider == "" {
				provider = providers[0]
			}

			// 创建LLM客户端
			factory := llm.NewClientFactory(configManager)
			llmClient, err := factory.Create(provider)
			if err != nil {
				return fmt.Errorf("创建LLM客户端失败: %w", err)
			}

			// 创建工具注册表
			registry := tools.NewToolRegistry()
			tools.RegisterDefaultTools(registry)

			// 创建用户（简化处理，使用默认admin用户）
			userRepo := db.NewUserRepository(database)
			user, err := userRepo.GetByUsername("admin")
			if err != nil {
				return fmt.Errorf("获取用户失败: %w", err)
			}

			// 创建审计日志仓储
			auditRepo := db.NewAuditLogRepository(database)

			// 解析安全模式
			var mode agent.SafetyMode
			switch safetyMode {
			case "strict":
				mode = agent.SafetyModeStrict
			case "balanced":
				mode = agent.SafetyModeBalanced
			case "permissive":
				mode = agent.SafetyModePermissive
			default:
				mode = agent.SafetyModeBalanced
			}

			// 创建安全控制器
			safetyController := agent.NewSafetyController(mode, auditRepo, user.ID, user.Username)

			// 设置自动批准
			if autoApprove {
				safetyController.SetAutoApprove(true)
			}

			// 创建Agent
			ag := agent.NewAgent(llmClient, registry, safetyController)

			// 设置调试模式
			if debug {
				ag.SetDebug(true)
			}

			// 交互模式
			if interactive {
				// 根据是否启用后台任务选择不同的运行模式
				if backgroundTasks {
					return runInteractiveWithTasks(ag, provider)
				}
				return runNewInteractiveMode(ag, provider)
			}

			// 单次查询模式(支持后续对话)
			return runQueryWithFollowUp(ag, query, provider)
		},
	}

	agentCmd.Flags().StringVarP(&provider, "provider", "p", "", "LLM提供商（deepseek, ollama, claude）")
	agentCmd.Flags().StringVarP(&safetyMode, "safety", "s", "balanced", "安全模式（strict, balanced, permissive）")
	agentCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "交互模式")
	agentCmd.Flags().StringVarP(&query, "query", "q", "", "查询内容")
	agentCmd.Flags().BoolVarP(&debug, "debug", "d", false, "调试模式,显示详细的工具调用信息")
	agentCmd.Flags().BoolVarP(&autoApprove, "yes", "y", false, "自动批准所有操作,无需确认")
	agentCmd.Flags().BoolVarP(&backgroundTasks, "background", "b", false, "启用后台任务管理,支持多任务并发执行")

	return agentCmd
}

// runNewInteractiveMode 运行新的交互模式
func runNewInteractiveMode(ag *agent.Agent, provider string) error {
	fmt.Printf("🤖 opsxcli Agent [%s]\n", color.CyanString(provider))
	fmt.Println("输入 'exit' 或 'quit' 退出")
	fmt.Println()

	// 获取初始查询
	fmt.Print(color.GreenString("👤 您: "))
	reader := bufio.NewReader(os.Stdin)
	initialQuery, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("读取输入失败: %w", err)
	}

	initialQuery = strings.TrimSpace(initialQuery)
	if initialQuery == "" {
		return fmt.Errorf("请输入查询内容")
	}

	// 运行交互式会话
	ctx := context.Background()
	return ag.RunInteractive(ctx, initialQuery)
}

// runQueryWithFollowUp 运行查询并支持后续对话
func runQueryWithFollowUp(ag *agent.Agent, query, provider string) error {
	fmt.Printf("[%s] %s\n", color.CyanString(provider), color.GreenString(query))

	ctx := context.Background()
	return ag.RunWithFollowUp(ctx, query)
}

// runInteractiveWithTasks 运行支持后台任务的交互模式
func runInteractiveWithTasks(ag *agent.Agent, provider string) error {
	fmt.Printf("🤖 opsxcli Agent [%s] - 后台任务模式\n", color.CyanString(provider))
	fmt.Println("输入 'exit' 或 'quit' 退出")
	fmt.Println()

	// 获取初始查询
	fmt.Print(color.GreenString("👤 您: "))
	reader := bufio.NewReader(os.Stdin)
	initialQuery, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("读取输入失败: %w", err)
	}

	initialQuery = strings.TrimSpace(initialQuery)
	if initialQuery == "" {
		return fmt.Errorf("请输入查询内容")
	}

	// 运行交互式会话(支持后台任务)
	ctx := context.Background()
	return ag.RunInteractiveWithTasks(ctx, initialQuery)
}

// runInteractiveMode 运行交互模式(保留旧版本作为备份)
func runInteractiveMode(ag *agent.Agent, provider string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("🤖 opsxcli Agent [%s]\n", color.CyanString(provider))
	fmt.Println("输入 'exit' 或 'quit' 退出")
	fmt.Println()

	for {
		fmt.Print(color.GreenString("👤 您: "))

		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("读取输入失败: %w", err)
		}

		input = strings.TrimSpace(input)

		// 检查退出命令
		if input == "exit" || input == "quit" {
			fmt.Println(color.YellowString("👋 再见！"))
			break
		}

		if input == "" {
			continue
		}

		// 运行Agent
		ctx := context.Background()
		result, err := ag.Run(ctx, input)

		if err != nil {
			fmt.Printf("%s %v\n\n", color.RedString("❌ 错误:"), err)
			continue
		}

		fmt.Printf("\n%s\n%s\n\n", color.CyanString("🤖 Agent:"), result)
	}

	return nil
}

// runSingleQuery 运行单次查询
func runSingleQuery(ag *agent.Agent, query, provider string) error {
	fmt.Printf("[%s] %s\n", color.CyanString(provider), color.GreenString(query))

	ctx := context.Background()
	result, err := ag.Run(ctx, query)

	if err != nil {
		return fmt.Errorf("Agent执行失败: %w", err)
	}

	fmt.Printf("\n%s\n%s\n", color.CyanString("🤖 回答:"), result)

	return nil
}
