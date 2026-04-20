package llm

import (
	"fmt"
)

// ClientFactory 客户端工厂
type ClientFactory struct {
	configManager *ConfigManager
}

// NewClientFactory 创建客户端工厂
func NewClientFactory(configManager *ConfigManager) *ClientFactory {
	return &ClientFactory{
		configManager: configManager,
	}
}

// Create 根据配置名称创建客户端
func (f *ClientFactory) Create(name string) (Client, error) {
	config, err := f.configManager.Get(name)
	if err != nil {
		return nil, err
	}

	return f.CreateFromConfig(config)
}

// CreateFromConfig 根据配置创建客户端
func (f *ClientFactory) CreateFromConfig(config *ProviderConfig) (Client, error) {
	switch config.Type {
	// OpenAI 兼容的模型（使用统一的 OpenAI API 格式）
	case "deepseek", "openai", "gpt", "kimi", "qwen", "yi", "baichuan", "doubao", "llama", "ollama":
		return NewOpenAIClient(config.BaseURL, config.APIKey, config.Model), nil

	// Anthropic 兼容的模型（GLM、MiniMax 走 Anthropic 协议）
	case "glm":
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://open.bigmodel.cn/api/anthropic"
		}
		return NewClaudeClientWithBaseURL(baseURL, config.APIKey, config.Model), nil

	case "minimax":
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://api.minimaxi.com/anthropic"
		}
		return NewClaudeClientWithBaseURL(baseURL, config.APIKey, config.Model), nil

	// Claude 专用客户端（支持自定义 BaseURL）
	// 也处理 type="anthropic" 的通用 Anthropic 兼容配置
	case "claude", "anthropic":
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}
		return NewClaudeClientWithBaseURL(baseURL, config.APIKey, config.Model), nil

	// Gemini 专用客户端
	case "gemini":
		return NewGeminiClient(config.APIKey, config.Model), nil

	default:
		return nil, fmt.Errorf("不支持的提供商类型: %s", config.Type)
	}
}

// ListProviders 列出所有可用的提供商
func (f *ClientFactory) ListProviders() []string {
	return f.configManager.List()
}
