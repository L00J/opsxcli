package llm

import (
	"strings"
	"testing"
)

// TestRegisterProvider_CustomProvider 测试注册自定义 provider
func TestRegisterProvider_CustomProvider(t *testing.T) {
	// 注册一个自定义 provider
	RegisterProvider("test-custom-provider", func(config *ProviderConfig) (Client, error) {
		return NewOpenAIClient(config.BaseURL, config.APIKey, config.Model), nil
	})
	defer func() {
		// 清理：从注册表中移除
		providerRegistry.Lock()
		delete(providerRegistry.factories, "test-custom-provider")
		providerRegistry.Unlock()
	}()

	factory, ok := getProvider("test-custom-provider")
	if !ok {
		t.Fatal("getProvider(\"test-custom-provider\") should return true")
	}

	client, err := factory(&ProviderConfig{
		BaseURL: "http://localhost:8080/v1",
		APIKey:  "test-key",
		Model:   "test-model",
	})
	if err != nil {
		t.Fatalf("factory() unexpected error: %v", err)
	}
	if client.Name() != "openai-compatible" {
		t.Errorf("client.Name() = %q, want %q", client.Name(), "openai-compatible")
	}
}

// TestGetRegisteredProviders 测试获取已注册 provider 列表
func TestGetRegisteredProviders(t *testing.T) {
	providers := getRegisteredProviders()

	// 验证返回的是排序列表
	for i := 1; i < len(providers); i++ {
		if providers[i] < providers[i-1] {
			t.Errorf("getRegisteredProviders() not sorted: %q before %q", providers[i-1], providers[i])
		}
	}

	// 验证核心 provider 都已注册
	expectedProviders := []string{
		"deepseek", "openai", "gpt", "kimi", "qwen",
		"yi", "baichuan", "doubao", "llama", "ollama",
		"glm", "minimax", "claude", "anthropic",
		"gemini",
	}

	providerSet := make(map[string]bool)
	for _, p := range providers {
		providerSet[p] = true
	}

	for _, expected := range expectedProviders {
		if !providerSet[expected] {
			t.Errorf("expected provider %q to be registered, but it was not found in %v", expected, providers)
		}
	}
}

// TestCreateFromConfig_UnregisteredProvider 测试未注册 provider 的错误信息
func TestCreateFromConfig_UnregisteredProvider(t *testing.T) {
	f := NewClientFactory(nil)

	config := &ProviderConfig{
		Type:    "nonexistent-provider",
		BaseURL: "http://localhost",
		APIKey:  "test",
		Model:   "test",
	}

	client, err := f.CreateFromConfig(config)
	if err == nil {
		t.Fatal("CreateFromConfig() should return error for unregistered provider")
	}
	if client != nil {
		t.Fatal("CreateFromConfig() should return nil client for unregistered provider")
	}

	// 验证错误信息包含 provider 类型
	if !strings.Contains(err.Error(), "nonexistent-provider") {
		t.Errorf("error should contain provider type, got: %v", err)
	}

	// 验证错误信息包含已注册 provider 列表
	if !strings.Contains(err.Error(), "已注册:") {
		t.Errorf("error should contain registered providers list, got: %v", err)
	}
}

// TestRegisterProvider_Overwrite 测试覆盖注册 provider
func TestRegisterProvider_Overwrite(t *testing.T) {
	// 保存原始 factory
	originalFactory, ok := getProvider("gemini")
	if !ok {
		t.Fatal("gemini provider should be registered")
	}

	// 覆盖注册
	var callCount int
	RegisterProvider("gemini", func(config *ProviderConfig) (Client, error) {
		callCount++
		return NewGeminiClient(config.APIKey, config.Model), nil
	})

	f := NewClientFactory(nil)
	_, err := f.CreateFromConfig(&ProviderConfig{
		Type:   "gemini",
		APIKey: "test",
		Model:  "test-model",
	})
	if err != nil {
		t.Fatalf("CreateFromConfig() error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected custom factory to be called once, got %d", callCount)
	}

	// 恢复原始 factory
	providerRegistry.Lock()
	providerRegistry.factories["gemini"] = originalFactory
	providerRegistry.Unlock()
}

// TestRegistryConcurrency 测试注册表并发安全
func TestRegistryConcurrency(t *testing.T) {
	const goroutines = 50

	done := make(chan bool, goroutines)

	// 并发读取
	for i := 0; i < goroutines/2; i++ {
		go func() {
			getProvider("deepseek")
			getRegisteredProviders()
			done <- true
		}()
	}

	// 并发写入（注册临时 provider 然后清理）
	for i := 0; i < goroutines/2; i++ {
		go func(id int) {
			name := "concurrent-test-" + string(rune('a'+id%26))
			RegisterProvider(name, func(config *ProviderConfig) (Client, error) {
				return NewOpenAIClient(config.BaseURL, config.APIKey, config.Model), nil
			})
			getProvider(name)
			providerRegistry.Lock()
			delete(providerRegistry.factories, name)
			providerRegistry.Unlock()
			done <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
}
