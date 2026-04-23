package llm

import "testing"

// TestClientFactory_CreateFromConfig 测试工厂创建不同提供商客户端
func TestClientFactory_CreateFromConfig(t *testing.T) {
	f := NewClientFactory(nil)

	tests := []struct {
		name     string
		config   *ProviderConfig
		wantName string
		wantErr  bool
	}{
		{
			name:     "deepseek",
			config:   &ProviderConfig{Type: "deepseek", BaseURL: "https://api.deepseek.com/v1", APIKey: "test", Model: "deepseek-chat"},
			wantName: "openai-compatible",
			wantErr:  false,
		},
		{
			name:     "openai",
			config:   &ProviderConfig{Type: "openai", BaseURL: "https://api.openai.com/v1", APIKey: "test", Model: "gpt-4"},
			wantName: "openai-compatible",
			wantErr:  false,
		},
		{
			name:     "ollama",
			config:   &ProviderConfig{Type: "ollama", BaseURL: "http://localhost:11434/v1", APIKey: "", Model: "llama2"},
			wantName: "openai-compatible",
			wantErr:  false,
		},
		{
			name:     "claude",
			config:   &ProviderConfig{Type: "claude", BaseURL: "https://api.anthropic.com/v1", APIKey: "test", Model: "claude-3"},
			wantName: "claude",
			wantErr:  false,
		},
		{
			name:     "gemini",
			config:   &ProviderConfig{Type: "gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta", APIKey: "test", Model: "gemini-pro"},
			wantName: "gemini",
			wantErr:  false,
		},
		{
			name:     "unsupported",
			config:   &ProviderConfig{Type: "unknown", BaseURL: "http://test", APIKey: "test", Model: "test"},
			wantName: "",
			wantErr:  true,
		},
		{
			name:     "kimi (openai compatible)",
			config:   &ProviderConfig{Type: "kimi", BaseURL: "https://api.moonshot.cn/v1", APIKey: "test", Model: "moonshot"},
			wantName: "openai-compatible",
			wantErr:  false,
		},
		{
			name:     "qwen (openai compatible)",
			config:   &ProviderConfig{Type: "qwen", BaseURL: "https://dashscope.aliyuncs.com/v1", APIKey: "test", Model: "qwen-turbo"},
			wantName: "openai-compatible",
			wantErr:  false,
		},
		// GLM now uses Anthropic-compatible protocol (not OpenAI)
		{
			name:     "glm (anthropic compatible)",
			config:   &ProviderConfig{Type: "glm", BaseURL: "https://open.bigmodel.cn/api/anthropic", APIKey: "test", Model: "glm-4"},
			wantName: "claude",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := f.CreateFromConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("CreateFromConfig() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateFromConfig() unexpected error: %v", err)
			}
			if client.Name() != tt.wantName {
				t.Errorf("client.Name() = %q, want %q", client.Name(), tt.wantName)
			}
		})
	}
}

// TestClientFactory_CreateFromConfig_allOpenAICompat 测试所有 OpenAI 兼容提供商
func TestClientFactory_CreateFromConfig_allOpenAICompat(t *testing.T) {
	f := NewClientFactory(nil)

	openAICompatTypes := []string{
		"deepseek", "openai", "gpt", "kimi", "qwen", "glm",
		"yi", "baichuan", "minimax", "doubao", "llama", "ollama",
	}

	// GLM and MiniMax now use Anthropic-compatible protocol (not OpenAI)
	anthropicCompatTypes := map[string]bool{"glm": true, "minimax": true}

	for _, providerType := range openAICompatTypes {
		t.Run(providerType, func(t *testing.T) {
			config := &ProviderConfig{
				Type:    providerType,
				BaseURL: "http://localhost:8080/v1",
				APIKey:  "test",
				Model:   "test-model",
			}
			client, err := f.CreateFromConfig(config)
			if err != nil {
				t.Fatalf("CreateFromConfig(%q) error: %v", providerType, err)
			}
			expected := "openai-compatible"
			if anthropicCompatTypes[providerType] {
				expected = "claude"
			}
			if client.Name() != expected {
				t.Errorf("provider %q: expected %s, got %q", providerType, expected, client.Name())
			}
		})
	}
}
