package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== ProviderConfig.IsValid 测试 =====

func TestProviderConfig_IsValid_Ollama(t *testing.T) {
	// Ollama 本地服务不需要 API key
	config := &ProviderConfig{
		Type:    "ollama",
		BaseURL: "http://localhost:11434/v1",
		APIKey:  "",
		Model:   "qwen3:14b",
	}
	assert.True(t, config.IsValid())
}

func TestProviderConfig_IsValid_LocalService(t *testing.T) {
	// 本地 vLLM 等服务，有 BaseURL 但 key 为空
	config := &ProviderConfig{
		Type:    "openai",
		BaseURL: "http://localhost:8000/v1",
		APIKey:  "",
		Model:   "test-model",
	}
	assert.True(t, config.IsValid())
}

func TestProviderConfig_IsValid_OllamaKey(t *testing.T) {
	// 本地服务，APIKey="ollama" 也视为有效
	config := &ProviderConfig{
		Type:    "openai",
		BaseURL: "http://localhost:11434/v1",
		APIKey:  "ollama",
		Model:   "test",
	}
	assert.True(t, config.IsValid())
}

func TestProviderConfig_IsValid_CloudService(t *testing.T) {
	// 云服务需要有效 key（>=10 字符）
	config := &ProviderConfig{
		Type:    "deepseek",
		APIKey:  "sk-1234567890abcdef1234567890abcdef",
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-chat",
	}
	assert.True(t, config.IsValid())
}

func TestProviderConfig_IsValid_EmptyKey(t *testing.T) {
	// 云服务空 key 无效
	config := &ProviderConfig{
		Type:   "deepseek",
		APIKey: "",
		Model:  "deepseek-chat",
	}
	assert.False(t, config.IsValid())
}

func TestProviderConfig_IsValid_ShortKey(t *testing.T) {
	// key 太短（<10 字符）无效
	config := &ProviderConfig{
		Type:   "deepseek",
		APIKey: "short",
		Model:  "deepseek-chat",
	}
	assert.False(t, config.IsValid())
}

func TestProviderConfig_IsValid_PlaceholderKey(t *testing.T) {
	placeholders := []string{
		"YOUR_API_KEY_HERE",
		"placeholder_key_1234567890",
		"changeme_please_1234567890",
		"example_api_key_value_12345",
		"xxx_api_key_value_123456789",
		"test_api_key_value_123456789",
		"demo_api_key_value_123456789",
		"sk-test_api_key_value_123456",
		"sk-xxx_api_key_value_1234567",
		"sk-123456_API_KEY_1234567890",
	}

	for _, key := range placeholders {
		config := &ProviderConfig{
			Type:   "deepseek",
			APIKey: key,
			Model:  "test",
		}
		assert.False(t, config.IsValid(), "key=%q 应该是无效的占位符", key)
	}
}

// ===== ConfigManager 测试 =====

func TestNewConfigManager(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, cm)
	assert.Equal(t, tmpDir, cm.configDir)
}

func TestNewConfigManager_EmptyDir(t *testing.T) {
	// 空目录也能正常创建
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, cm.List())
}

func TestConfigManager_SaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	config := &ProviderConfig{
		Type:        "deepseek",
		BaseURL:     "https://api.deepseek.com/v1",
		APIKey:      "sk-test-1234567890abcdef1234567890abcdef",
		Model:       "deepseek-chat",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// 保存
	err = cm.Save("test-provider", config)
	require.NoError(t, err)

	// 读取
	got, err := cm.Get("test-provider")
	require.NoError(t, err)
	assert.Equal(t, config.Type, got.Type)
	assert.Equal(t, config.BaseURL, got.BaseURL)
	assert.Equal(t, config.APIKey, got.APIKey)
	assert.Equal(t, config.Model, got.Model)
	assert.Equal(t, config.Temperature, got.Temperature)
	assert.Equal(t, config.MaxTokens, got.MaxTokens)
}

func TestConfigManager_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	_, err = cm.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestConfigManager_List(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	// 保存多个配置
	configs := map[string]*ProviderConfig{
		"zeta":  {Type: "deepseek", APIKey: "sk-1234567890abcdef1234567890", Model: "test"},
		"alpha": {Type: "claude", APIKey: "sk-1234567890abcdef1234567890", Model: "test"},
		"beta":  {Type: "openai", APIKey: "sk-1234567890abcdef1234567890", Model: "test"},
	}

	for name, cfg := range configs {
		err := cm.Save(name, cfg)
		require.NoError(t, err)
	}

	// 列表应按字母排序
	list := cm.List()
	expected := []string{"alpha", "beta", "zeta"}
	assert.Equal(t, expected, list)
}

func TestConfigManager_LoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 手动写入配置文件
	config := &ProviderConfig{
		Type:    "deepseek",
		BaseURL: "https://api.deepseek.com/v1",
		APIKey:  "sk-test-1234567890abcdef1234567890",
		Model:   "deepseek-chat",
	}
	data, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "myprovider.json"), data, 0600)
	require.NoError(t, err)

	// 加载
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	got, err := cm.Get("myprovider")
	require.NoError(t, err)
	assert.Equal(t, "deepseek", got.Type)
	assert.Equal(t, "deepseek-chat", got.Model)
}

func TestConfigManager_IgnoresNonJSONFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// 写入非 JSON 文件
	err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("test"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "subdir"), []byte("test"), 0700)
	require.NoError(t, err)

	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, cm.List())
}

func TestConfigManager_GetValidProviders(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	// 保存一个有效配置
	cm.Save("valid-provider", &ProviderConfig{
		Type:    "deepseek",
		APIKey:  "sk-1234567890abcdef1234567890abcdef",
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-chat",
	})

	// 保存一个无效配置（占位符 key）
	cm.Save("invalid-provider", &ProviderConfig{
		Type:   "deepseek",
		APIKey: "YOUR_API_KEY",
		Model:  "deepseek-chat",
	})

	// 保存一个 Ollama 配置（无 key 也有效）
	cm.Save("local-ollama", &ProviderConfig{
		Type:    "ollama",
		APIKey:  "",
		BaseURL: "http://localhost:11434/v1",
		Model:   "qwen3:14b",
	})

	valid := cm.GetValidProviders()
	expected := []string{"local-ollama", "valid-provider"}
	sort.Strings(expected)
	assert.Equal(t, expected, valid)
}

func TestConfigManager_HasAnyValidProvider(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	// 初始无有效 provider
	assert.False(t, cm.HasAnyValidProvider())

	// 添加有效 provider
	cm.Save("test", &ProviderConfig{
		Type:    "deepseek",
		APIKey:  "sk-1234567890abcdef1234567890abcdef",
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "test",
	})
	assert.True(t, cm.HasAnyValidProvider())
}

func TestConfigManager_SaveDefaultProvider(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	// 保存默认 provider
	err = cm.SaveDefaultProvider("deepseek")
	require.NoError(t, err)

	// 读取
	got := cm.GetDefaultProvider()
	assert.Equal(t, "deepseek", got)
}

func TestConfigManager_GetDefaultProvider_ReadsGlobalConfig(t *testing.T) {
	// GetDefaultProvider 读取全局配置 (~/.opsxcli/config.json)，
	// 不依赖 cm.configDir。我们验证函数可正常调用即可。
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	// 不应 panic，返回一个字符串
	got := cm.GetDefaultProvider()
	// 值取决于全局配置文件是否存在
	_ = got
}

func TestConfigManager_SaveAndGetRegion(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	// 保存区域
	err = cm.SaveRegion("cn")
	require.NoError(t, err)

	// 读取
	got := cm.GetRegion()
	assert.Equal(t, "cn", got)
}

func TestConfigManager_CreateDefaultConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	// 创建默认配置
	err = cm.CreateDefaultConfigs()
	require.NoError(t, err)

	// 应该有 8 个默认配置
	list := cm.List()
	assert.Len(t, list, 8)

	// 验证包含关键配置
	expectedConfigs := []string{"deepseek", "claude", "gpt", "kimi", "gemini", "glm", "minimax", "ollama"}
	for _, name := range expectedConfigs {
		assert.Contains(t, list, name, "应包含 %s 配置", name)
	}

	// 默认配置的 key 应该是占位符（无效）
	for _, name := range list {
		cfg, err := cm.Get(name)
		require.NoError(t, err)
		if name != "ollama" {
			assert.False(t, cfg.IsValid(), "默认配置 %s 应使用占位符 key", name)
		}
	}
}

func TestConfigManager_CreateDefaultConfigs_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	// 先保存一个自定义配置
	customConfig := &ProviderConfig{
		Type:    "deepseek",
		APIKey:  "sk-my-real-key-1234567890abcdef",
		BaseURL: "https://custom.api.com/v1",
		Model:   "custom-model",
	}
	cm.Save("deepseek", customConfig)

	// CreateDefaultConfigs 不应覆盖已有配置
	err = cm.CreateDefaultConfigs()
	require.NoError(t, err)

	got, err := cm.Get("deepseek")
	require.NoError(t, err)
	assert.Equal(t, "sk-my-real-key-1234567890abcdef", got.APIKey)
	assert.Equal(t, "custom-model", got.Model)
}

// ===== ClientFactory 测试补充 =====

func TestClientFactory_Create_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	factory := NewClientFactory(cm)
	_, err = factory.Create("nonexistent")
	assert.Error(t, err)
}

func TestClientFactory_CreateFromConfig_GLMDefaultURL(t *testing.T) {
	factory := NewClientFactory(nil)
	config := &ProviderConfig{
		Type:   "glm",
		APIKey: "test-key",
		Model:  "glm-5.1",
		// 不设置 BaseURL，应使用默认
	}
	client, err := factory.CreateFromConfig(config)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "claude", client.Name())
}

func TestClientFactory_CreateFromConfig_MiniMaxDefaultURL(t *testing.T) {
	factory := NewClientFactory(nil)
	config := &ProviderConfig{
		Type:   "minimax",
		APIKey: "test-key",
		Model:  "MiniMax-M2.7",
		// 不设置 BaseURL
	}
	client, err := factory.CreateFromConfig(config)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "claude", client.Name())
}

func TestClientFactory_CreateFromConfig_ClaudeDefaultURL(t *testing.T) {
	factory := NewClientFactory(nil)
	config := &ProviderConfig{
		Type:   "claude",
		APIKey: "test-key",
		Model:  "claude-opus-4",
		// 不设置 BaseURL
	}
	client, err := factory.CreateFromConfig(config)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "claude", client.Name())
}

// ===== globalConfigPath 测试 =====

func TestClientFactory_ListProviders(t *testing.T) {
	tmpDir := t.TempDir()
	cm, err := NewConfigManager(tmpDir)
	require.NoError(t, err)

	factory := NewClientFactory(cm)

	// 初始无 provider
	assert.Empty(t, factory.ListProviders())

	// 添加 provider 后可见
	cm.Save("test1", &ProviderConfig{Type: "deepseek", APIKey: "sk-1234567890abcdef1234567890", Model: "test"})
	cm.Save("test2", &ProviderConfig{Type: "claude", APIKey: "sk-1234567890abcdef1234567890", Model: "test"})
	assert.Equal(t, []string{"test1", "test2"}, factory.ListProviders())
}

func TestGlobalConfigPath(t *testing.T) {
	path, err := globalConfigPath()
	require.NoError(t, err)
	assert.Contains(t, path, ".opsxcli")
	assert.Contains(t, path, "config.json")
}
