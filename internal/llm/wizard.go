package llm

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfigWizard 配置向导
type ConfigWizard struct {
	configManager *ConfigManager
	reader        *bufio.Reader
}

// NewConfigWizard 创建配置向导
func NewConfigWizard(configManager *ConfigManager) *ConfigWizard {
	return &ConfigWizard{
		configManager: configManager,
		reader:        bufio.NewReader(os.Stdin),
	}
}

// Run 运行配置向导
func (w *ConfigWizard) Run() error {
	// 打印欢迎界面
	w.printWelcome()

	// 选择提供商类型
	providerType, err := w.selectProvider()
	if err != nil {
		return err
	}

	// 根据类型配置
	var config *ProviderConfig
	switch providerType {
	case "deepseek":
		config, err = w.configureDeepSeek()
	case "claude":
		config, err = w.configureClaude()
	case "gpt":
		config, err = w.configureGPT()
	case "kimi":
		config, err = w.configureKimi()
	case "gemini":
		config, err = w.configureGemini()
	case "glm":
		config, err = w.configureGLM()
	case "minimax":
		config, err = w.configureMiniMax()
	case "ollama":
		config, err = w.configureOllama()
	default:
		return fmt.Errorf("未知的提供商类型: %s", providerType)
	}

	if err != nil {
		return err
	}

	// 保存配置
	if err := w.configManager.Save(providerType, config); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	// 打印成功信息
	w.printSuccess(providerType)

	return nil
}

// printWelcome 打印欢迎界面
func (w *ConfigWizard) printWelcome() {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println()
	fmt.Println("        🤖  欢迎使用 OpsX CLI - AI 智能运维助手  🚀")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
	fmt.Println("  首次使用需要配置大模型，请选择您喜欢的 LLM 提供商：")
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()

	// 推荐模型
	fmt.Println("  ✨ 推荐模型")
	fmt.Println()
	fmt.Println("    \033[1;36m1. DeepSeek\033[0m          ⭐⭐⭐⭐⭐ 默认推荐 | 性价比之王 | 推理强悍")
	fmt.Println("    \033[1;36m2. Claude\033[0m            ⭐⭐⭐⭐⭐ Opus 4.7 | Agentic Engineering 最强 | 代码与推理卓越")
	fmt.Println("    \033[1;36m3. ChatGPT\033[0m           ⭐⭐⭐⭐⭐ GPT-5.4 | 工具调用强 | Computer Use | 全能均衡")
	fmt.Println("    \033[1;36m4. Kimi\033[0m              ⭐⭐⭐⭐☆ K2.5-code | 超长上下文 | 代码与文档处理")
	fmt.Println("    \033[1;36m5. Gemini\033[0m            ⭐⭐⭐⭐ Gemini 3.1 Pro | 多模态强 | 推理翻倍")
	fmt.Println("    \033[1;36m6. GLM (智谱)\033[0m        ⭐⭐⭐⭐ GLM-5.1 | 国产之光 | Agentic Coding | 百万级上下文")
	fmt.Println("    \033[1;36m7. MiniMax\033[0m           ⭐⭐⭐☆ M2.7-highspeed | 速度优先 | Agent自我进化")
	fmt.Println()

	// 本地模型
	fmt.Println("  🏠 本地模型")
	fmt.Println()
	fmt.Println("    \033[1;32m8. 本地服务\033[0m          💯 Ollama | vLLM | LM Studio | 私有部署")
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
}

// printSuccess 打印成功信息
func (w *ConfigWizard) printSuccess(providerType string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
	fmt.Println("  ✅  配置完成！")
	fmt.Println()
	fmt.Println("  🎉 开始使用:")
	fmt.Printf("     \033[1;32mopsxcli agent -p %s \"你的问题\"\033[0m\n", providerType)
	fmt.Println()
	fmt.Println("  💡 快速示例:")
	fmt.Println("     • opsxcli agent \"查看系统负载\"")
	fmt.Println("     • opsxcli agent \"分析网络连接\"")
	fmt.Println("     • opsxcli agent \"优化 Docker 配置\"")
	fmt.Println()
	fmt.Println("  📚 查看更多:")
	fmt.Println("     opsxcli agent --help")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
}

// selectProvider 选择提供商
func (w *ConfigWizard) selectProvider() (string, error) {
	for {
		fmt.Print("  👉 请选择 (1-8) [默认: \033[1;33m1\033[0m]: ")
		input, err := w.reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		input = strings.TrimSpace(input)
		// 默认选择DeepSeek
		if input == "" {
			input = "1"
		}

		switch input {
		case "1":
			fmt.Println("\n  ⭐ 已选择: DeepSeek - 性价比之王 + 推理强悍")
			return "deepseek", nil
		case "2":
			fmt.Println("\n  🧠 已选择: Claude - 代码之王 + 运维专家")
			return "claude", nil
		case "3":
			fmt.Println("\n  🤖 已选择: ChatGPT - 行业标杆 + 全能型")
			return "gpt", nil
		case "4":
			fmt.Println("\n  🌙 已选择: Kimi - 长文本专家 + 日志分析")
			return "kimi", nil
		case "5":
			fmt.Println("\n  💎 已选择: Gemini - Google AI + 多模态")
			return "gemini", nil
		case "6":
			fmt.Println("\n  🇨🇳 已选择: GLM - 国产之光 + ChatGLM")
			return "glm", nil
		case "7":
			fmt.Println("\n  🚀 已选择: MiniMax - 字节跳动 + 海螺AI")
			return "minimax", nil
		case "8":
			fmt.Println("\n  🏠 已选择: 本地服务 - Ollama/vLLM/私有部署")
			return "ollama", nil
		default:
			fmt.Println("  ❌ 无效选择，请输入 1-8")
		}
	}
}

// configureOllama 配置本地模型服务
func (w *ConfigWizard) configureOllama() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  🏠 配置本地模型服务")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()

	fmt.Println("  💡 支持的本地服务:")
	fmt.Println("     • Ollama        - 最简单，推荐新手")
	fmt.Println("     • vLLM          - 高性能推理服务器")
	fmt.Println("     • LM Studio     - 图形化界面")
	fmt.Println("     • LocalAI       - OpenAI兼容服务")
	fmt.Println("     • text-generation-webui (oobabooga)")
	fmt.Println("     • 其他兼容OpenAI API的服务")
	fmt.Println()

	// 询问地址
	baseURL := w.promptWithDefault("  📍 服务地址", "http://localhost:11434/v1")

	// 询问模型
	fmt.Println()
	fmt.Println("  📌 常用模型示例:")
	fmt.Println("     Ollama:  qwen3:14b, llama3.3:70b, deepseek-r1:14b")
	fmt.Println("     vLLM:    meta-llama/Llama-3.3-70B-Instruct")
	fmt.Println("     其他:    根据您的部署配置")
	fmt.Println()
	model := w.promptWithDefault("  🤖 模型名称", "qwen3:14b")

	fmt.Println()
	fmt.Println("  📦 Ollama 快速安装:")
	fmt.Println("     \033[1;36mcurl -fsSL https://ollama.com/install.sh | sh\033[0m")
	fmt.Println("     \033[1;36mollama pull " + model + "\033[0m")
	fmt.Println()
	fmt.Println("  📚 vLLM 部署参考:")
	fmt.Println("     \033[1;36mpip install vllm\033[0m")
	fmt.Println("     \033[1;36mvllm serve <model_name> --api-key token-abc123\033[0m")
	fmt.Println()

	return &ProviderConfig{
		Type:        "ollama",
		BaseURL:     baseURL,
		APIKey:      "",
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// configureDeepSeek 配置DeepSeek
func (w *ConfigWizard) configureDeepSeek() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  ✨ 配置 DeepSeek - 性价比之王")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()

	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     1. 访问 \033[1;34mhttps://platform.deepseek.com/\033[0m")
	fmt.Println("     2. 注册/登录账号")
	fmt.Println("     3. 在 API 管理页面创建密钥")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 DeepSeek API 密钥")
	if apiKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	fmt.Println()
	fmt.Println("  📌 常用模型:")
	fmt.Println("     • deepseek-chat (推荐)")
	fmt.Println("     • deepseek-reasoner")
	fmt.Println()
	model := w.promptWithDefault("  🤖 模型名称", "deepseek-chat")

	return &ProviderConfig{
		Type:        "deepseek",
		BaseURL:     "https://api.deepseek.com/v1",
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// configureClaude 配置Claude
func (w *ConfigWizard) configureClaude() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  🧠 配置 Claude - Anthropic 代码之王")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()

	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     1. 访问 \033[1;34mhttps://console.anthropic.com/\033[0m")
	fmt.Println("     2. 注册/登录账号")
	fmt.Println("     3. 在 API Keys 页面创建密钥")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 Claude API 密钥")
	if apiKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	fmt.Println()
	fmt.Println("  📌 常用模型:")
	fmt.Println("     • claude-opus-4-20250514 (推荐，Agentic Engineering 最强)")
	fmt.Println("     • claude-sonnet-4-20250514")
	fmt.Println()
	model := w.promptWithDefault("  🤖 模型名称", "claude-opus-4-20250514")

	return &ProviderConfig{
		Type:        "claude",
		BaseURL:     "https://api.anthropic.com/v1",
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// prompt 提示输入
func (w *ConfigWizard) prompt(message string) string {
	fmt.Printf("%s: ", message)
	input, _ := w.reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// promptWithDefault 提示输入（带默认值）
func (w *ConfigWizard) promptWithDefault(message, defaultValue string) string {
	fmt.Printf("%s [%s]: ", message, defaultValue)
	input, _ := w.reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

// configureKimi 配置Kimi
func (w *ConfigWizard) configureKimi() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  🌙 配置 Kimi - 月之暗面，长文本专家")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     访问 \033[1;34mhttps://platform.moonshot.cn/\033[0m")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 Kimi API 密钥")
	if apiKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	fmt.Println()
	fmt.Println("  📌 常用模型:")
	fmt.Println("     • k2.5-code (推荐，代码+长文本)")
	fmt.Println("     • k2.6-code (最新版，更强)")
	fmt.Println()
	model := w.promptWithDefault("  🤖 模型名称", "k2.5-code")

	return &ProviderConfig{
		Type:        "kimi",
		BaseURL:     "https://api.moonshot.cn/v1",
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// configureQwen 配置Qwen
func (w *ConfigWizard) configureQwen() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  🔥 配置 Qwen - 通义千问，阿里云")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     访问 \033[1;34mhttps://dashscope.console.aliyun.com/\033[0m")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 Qwen API 密钥")
	if apiKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	fmt.Println()
	fmt.Println("  📌 常用模型:")
	fmt.Println("     • qwen-plus (推荐)")
	fmt.Println("     • qwen-max")
	fmt.Println("     • qwen-turbo (经济型)")
	fmt.Println()
	model := w.promptWithDefault("  🤖 模型名称", "qwen-plus")

	return &ProviderConfig{
		Type:        "qwen",
		BaseURL:     "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// configureGLM 配置GLM
func (w *ConfigWizard) configureGLM() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  ⚡ 配置 GLM - 智谱清言，清华背景")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     访问 \033[1;34mhttps://open.bigmodel.cn/\033[0m")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 GLM API 密钥")
	if apiKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	model := w.promptWithDefault("  🤖 模型名称", "glm-5.1")

	return &ProviderConfig{
		Type:        "glm",
		BaseURL:     "https://open.bigmodel.cn/api/paas/v4",
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// configureGPT 配置GPT
func (w *ConfigWizard) configureGPT() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  🤖 配置 ChatGPT - OpenAI 行业标杆")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     访问 \033[1;34mhttps://platform.openai.com/\033[0m")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 OpenAI API 密钥")
	if apiKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	fmt.Println()
	fmt.Println("  📌 常用模型:")
	fmt.Println("     • gpt-5.4 (推荐，工具调用+Computer Use)")
	fmt.Println("     • gpt-4o")
	fmt.Println()
	model := w.promptWithDefault("  🤖 模型名称", "gpt-5.4")

	return &ProviderConfig{
		Type:        "gpt",
		BaseURL:     "https://api.openai.com/v1",
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// configureGemini 配置Gemini
func (w *ConfigWizard) configureGemini() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  💎 配置 Gemini - Google 多模态AI")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     访问 \033[1;34mhttps://aistudio.google.com/\033[0m")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 Gemini API 密钥")
	if apiKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	fmt.Println()
	fmt.Println("  📌 常用模型:")
	fmt.Println("     • gemini-3.1-pro (推荐，多模态+推理)")
	fmt.Println("     • gemini-2.5-flash (快速)")
	fmt.Println()
	model := w.promptWithDefault("  🤖 模型名称", "gemini-3.1-pro")

	return &ProviderConfig{
		Type:        "gemini",
		BaseURL:     "https://generativelanguage.googleapis.com/v1beta",
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// configureYi 配置Yi
func (w *ConfigWizard) configureYi() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  🌟 配置 Yi - 零一万物，李开复创办")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     访问 \033[1;34mhttps://platform.lingyiwanwu.com/\033[0m")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 Yi API 密钥")
	if apiKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	model := w.promptWithDefault("  🤖 模型名称", "yi-lightning")

	return &ProviderConfig{
		Type:        "yi",
		BaseURL:     "https://api.lingyiwanwu.com/v1",
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// configureBaichuan 配置Baichuan
func (w *ConfigWizard) configureBaichuan() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  🏔️  配置 Baichuan - 百川智能，王小川创办")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     访问 \033[1;34mhttps://platform.baichuan-ai.com/\033[0m")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 Baichuan API 密钥")
	if apiKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	model := w.promptWithDefault("  🤖 模型名称", "Baichuan4")

	return &ProviderConfig{
		Type:        "baichuan",
		BaseURL:     "https://api.baichuan-ai.com/v1",
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// configureMiniMax 配置MiniMax
func (w *ConfigWizard) configureMiniMax() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  🎮 配置 MiniMax - 多模态AI")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     访问 \033[1;34mhttps://api.minimax.chat/\033[0m")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 MiniMax API 密钥")
	if apiKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	model := w.promptWithDefault("  🤖 模型名称", "MiniMax-M2.7-highspeed")

	return &ProviderConfig{
		Type:        "minimax",
		BaseURL:     "https://api.minimax.chat/v1",
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// configureDoubao 配置Doubao
func (w *ConfigWizard) configureDoubao() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  🫘 配置 Doubao - 字节豆包，火山引擎")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     访问 \033[1;34mhttps://www.volcengine.com/\033[0m")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 Doubao API 密钥")
	if apiKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	model := w.promptWithDefault("  🤖 模型名称", "doubao-pro-32k")

	return &ProviderConfig{
		Type:        "doubao",
		BaseURL:     "https://ark.cn-beijing.volces.com/api/v3",
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}

// configureLlama 配置Llama
func (w *ConfigWizard) configureLlama() (*ProviderConfig, error) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  🦙 配置 Llama - Meta 开源之光")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()
	fmt.Println("  💡 获取 API 密钥:")
	fmt.Println("     访问 \033[1;34mhttps://api.together.xyz/\033[0m (推荐)")
	fmt.Println("     或使用本地 Ollama: \033[1;36mollama pull llama3\033[0m")
	fmt.Println()

	apiKey := w.prompt("  🔑 请输入 API 密钥（本地 Ollama 可留空）")

	baseURL := w.promptWithDefault("  📍 API 地址", "https://api.together.xyz/v1")
	model := w.promptWithDefault("  🤖 模型名称", "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo")

	return &ProviderConfig{
		Type:        "llama",
		BaseURL:     baseURL,
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}, nil
}
