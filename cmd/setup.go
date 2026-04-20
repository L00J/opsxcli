package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"opsxcli/internal/llm"
)

func init() {
	RegisterCommand("setup", "管理", "配置 LLM 提供商管理", NewSetupCmd)
}

// NewSetupCmd 创建 setup 命令
func NewSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "配置 LLM 提供商",
		Long: `setup - 管理大模型提供商配置

交互式配置管理工具，支持：
  • 添加新的 LLM 提供商
  • 修改已有提供商配置
  • 设置默认提供商
  • 查看当前配置状态`,
		RunE: runSetup,
	}
}

// runSetup 运行 setup 命令
func runSetup(cmd *cobra.Command, args []string) error {
	// 初始化配置管理器
	configManager, err := llm.NewConfigManager("")
	if err != nil {
		return fmt.Errorf("初始化配置管理器失败: %w", err)
	}

	// 检查是否有有效配置
	providers := configManager.List()
	validProviders := configManager.GetValidProviders()

	// 无配置或全部占位符 → 走首次配置向导
	if len(providers) == 0 || len(validProviders) == 0 {
		fmt.Println()
		if len(providers) == 0 {
			fmt.Println(color.CyanString("🔧 未检测到 LLM 配置，启动首次配置向导..."))
		} else {
			fmt.Println(color.YellowString("⚠️  现有 API 密钥无效（占位符），启动配置向导..."))
		}
		fmt.Println()

		wizard := llm.NewConfigWizard(configManager)
		if err := wizard.Run(); err != nil {
			return fmt.Errorf("配置向导失败: %w", err)
		}

		// 配置完成后重新加载
		configManager, err = llm.NewConfigManager("")
		if err != nil {
			return fmt.Errorf("重新加载配置失败: %w", err)
		}

		// 引导进入管理菜单
		fmt.Println()
		fmt.Println(color.New(color.FgHiBlack).Sprint("  提示: 输入 opsxcli setup 可随时管理配置"))
		fmt.Println()
		return nil
	}

	// 有有效配置 → 走管理菜单
	reader := bufio.NewReader(os.Stdin)

	for {
		// 显示当前配置状态
		showConfigStatus(configManager)

		// 显示操作菜单
		fmt.Println()
		fmt.Println(color.CyanString("  📋 操作菜单:"))
		fmt.Println("    1. 添加新 provider")
		fmt.Println("    2. 修改已有 provider")
		fmt.Println("    3. 设置默认 provider")
		fmt.Println("    4. 退出")
		fmt.Println()

		fmt.Print(color.GreenString("  👉 请选择 [1-4]: "))
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("读取输入失败: %w", err)
		}
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			if err := addProvider(configManager, reader); err != nil {
				fmt.Printf("  ❌ %s\n\n", err.Error())
			}
		case "2":
			if err := modifyProvider(configManager, reader); err != nil {
				fmt.Printf("  ❌ %s\n\n", err.Error())
			}
		case "3":
			if err := setDefaultProvider(configManager, reader); err != nil {
				fmt.Printf("  ❌ %s\n\n", err.Error())
			}
		case "4", "":
			fmt.Println()
			fmt.Println(color.YellowString("  👋 再见！"))
			fmt.Println()
			return nil
		default:
			fmt.Println("  ❌ 无效选择，请输入 1-4")
		}
	}
}

// showConfigStatus 显示当前配置状态
func showConfigStatus(cm *llm.ConfigManager) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
	fmt.Println("  🔧  LLM 提供商配置管理")
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))

	providers := cm.List()
	validProviders := cm.GetValidProviders()
	defaultProvider := cm.GetDefaultProvider()

	if len(providers) == 0 {
		fmt.Println()
		fmt.Println(color.YellowString("  ⚠️  当前没有任何 provider 配置"))
		fmt.Println()
	} else {
		fmt.Println()
		fmt.Printf("  %-4s %-15s %-12s %-20s %s\n",
			"", "Provider", "状态", "模型", "默认")
		fmt.Println("  " + strings.Repeat("-", 60))

		for i, name := range providers {
			config, err := cm.Get(name)
			if err != nil {
				continue
			}

			statusIcon := "✅"
			status := "有效"
			if !config.IsValid() {
				statusIcon = "⚠️ "
				status = "占位符"
			}

			isDefault := ""
			if name == defaultProvider {
				isDefault = color.GreenString("⭐ 默认")
			}

			fmt.Printf("  %-4s %-15s %s%-10s %-20s %s\n",
				fmt.Sprintf("%d.", i+1),
				color.CyanString(name),
				statusIcon+" ",
				status,
				config.Model,
				isDefault,
			)
		}
		fmt.Println("  " + strings.Repeat("-", 60))
		fmt.Printf("  共 %d 个 provider，%d 个有效\n", len(providers), len(validProviders))
	}

	if defaultProvider != "" {
		fmt.Printf("  当前默认: %s\n", color.CyanString(defaultProvider))
	}
}

// addProvider 添加新 provider
func addProvider(cm *llm.ConfigManager, reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println(color.CyanString("  ➕ 添加新 provider"))
	fmt.Println()

	wizard := llm.NewConfigWizardWithReader(cm, reader)
	return wizard.Run()
}

// modifyProvider 修改已有 provider
func modifyProvider(cm *llm.ConfigManager, reader *bufio.Reader) error {
	providers := cm.List()
	if len(providers) == 0 {
		fmt.Println("  ⚠️  没有可修改的 provider，请先添加")
		return nil
	}

	fmt.Println()
	fmt.Println(color.CyanString("  ✏️  修改已有 provider"))
	fmt.Println()
	fmt.Printf("  %-4s %s\n", "", "Provider")
	fmt.Println("  " + strings.Repeat("-", 30))
	for i, name := range providers {
		fmt.Printf("  %-4s %s\n", fmt.Sprintf("%d.", i+1), color.CyanString(name))
	}
	fmt.Println()

	fmt.Print(color.GreenString("  👉 请选择要修改的 provider 编号: "))
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("读取输入失败: %w", err)
	}
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(providers) {
		return fmt.Errorf("无效选择")
	}

	selectedName := providers[idx-1]
	fmt.Printf("  🔄 重新配置 %s ...\n", color.CyanString(selectedName))
	fmt.Println()

	wizard := llm.NewConfigWizardWithReader(cm, reader)
	return wizard.Run()
}

// setDefaultProvider 设置默认 provider
func setDefaultProvider(cm *llm.ConfigManager, reader *bufio.Reader) error {
	validProviders := cm.GetValidProviders()
	if len(validProviders) == 0 {
		fmt.Println("  ⚠️  没有有效的 provider，请先配置一个有效的 provider")
		return nil
	}

	currentDefault := cm.GetDefaultProvider()

	fmt.Println()
	fmt.Println(color.CyanString("  ⭐ 设置默认 provider"))
	fmt.Println()
	fmt.Printf("  当前默认: %s\n", func() string {
		if currentDefault == "" {
			return color.New(color.FgHiBlack).Sprint("未设置")
		}
		return color.CyanString(currentDefault)
	}())
	fmt.Println()
	fmt.Printf("  %-4s %-15s %-20s\n", "", "Provider", "模型")
	fmt.Println("  " + strings.Repeat("-", 45))

	for i, name := range validProviders {
		config, _ := cm.Get(name)
		modelName := ""
		if config != nil {
			modelName = config.Model
		}
		marker := ""
		if name == currentDefault {
			marker = color.GreenString(" ⭐")
		}
		fmt.Printf("  %-4s %-15s %-20s%s\n",
			fmt.Sprintf("%d.", i+1),
			color.CyanString(name),
			modelName,
			marker,
		)
	}
	fmt.Println()

	fmt.Print(color.GreenString("  👉 请选择默认 provider 编号: "))
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("读取输入失败: %w", err)
	}
	input = strings.TrimSpace(input)

	if input == "" {
		fmt.Println("  ⚠️  已取消")
		return nil
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(validProviders) {
		return fmt.Errorf("无效选择: %q (请输入 1-%d)", input, len(validProviders))
	}

	selected := validProviders[idx-1]
	if err := cm.SaveDefaultProvider(selected); err != nil {
		return fmt.Errorf("保存默认 provider 失败: %w", err)
	}

	fmt.Printf("  ✅ 默认 provider 已设置为: %s\n", color.CyanString(selected))
	fmt.Println()
	return nil
}
