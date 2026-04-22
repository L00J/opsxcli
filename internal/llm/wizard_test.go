package llm

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== 辅助函数 =====

// newTestWizard 创建带输入模拟的测试向导
func newTestWizard(cm *ConfigManager, input string) *ConfigWizard {
	return NewConfigWizardWithReader(cm, bufio.NewReader(strings.NewReader(input)))
}

func newTestConfigManager(t *testing.T) *ConfigManager {
	t.Helper()
	dir := t.TempDir()
	cm, err := NewConfigManager(dir)
	require.NoError(t, err)
	return cm
}

// ===== NewConfigWizard 测试 =====

func TestNewConfigWizard(t *testing.T) {
	// 测试创建配置向导
	cm := newTestConfigManager(t)
	wizard := NewConfigWizard(cm)
	require.NotNil(t, wizard)
	assert.Equal(t, cm, wizard.configManager)
	assert.NotNil(t, wizard.reader)
}

func TestNewConfigWizardWithReader(t *testing.T) {
	// 测试使用自定义 Reader 创建向导
	cm := newTestConfigManager(t)
	reader := bufio.NewReader(strings.NewReader("test"))
	wizard := NewConfigWizardWithReader(cm, reader)
	require.NotNil(t, wizard)
	assert.Equal(t, reader, wizard.reader)
}

// ===== detectRegion 测试 =====

func TestConfigWizard_DetectRegion(t *testing.T) {
	// 测试自动检测区域（默认中国）
	cm := newTestConfigManager(t)
	wizard := NewConfigWizardWithReader(cm, bufio.NewReader(strings.NewReader("")))

	region := wizard.detectRegion()
	// 无论环境如何，默认应返回 "cn"
	assert.Contains(t, []string{"cn", "global"}, region)
}

func TestConfigWizard_DetectRegion_ChineseLocale(t *testing.T) {
	// 测试设置中文语言环境后检测为中国区域
	cm := newTestConfigManager(t)
	wizard := NewConfigWizardWithReader(cm, bufio.NewReader(strings.NewReader("")))

	// 临时设置 LANG 环境变量
	origLang := os.Getenv("LANG")
	os.Setenv("LANG", "zh_CN.UTF-8")
	defer os.Setenv("LANG", origLang)

	region := wizard.detectRegion()
	assert.Equal(t, "cn", region)
}

func TestConfigWizard_DetectRegion_TZShanghai(t *testing.T) {
	// 测试设置亚洲/上海时区
	cm := newTestConfigManager(t)
	wizard := NewConfigWizardWithReader(cm, bufio.NewReader(strings.NewReader("")))

	origTZ := os.Getenv("TZ")
	os.Setenv("TZ", "Asia/Shanghai")
	defer os.Setenv("TZ", origTZ)

	region := wizard.detectRegion()
	assert.Equal(t, "cn", region)
}

// ===== ensureRegion 测试 =====

func TestConfigWizard_EnsureRegion_AlreadySet(t *testing.T) {
	// 已设置 region 时不再修改
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "")
	wizard.region = "cn"
	wizard.ensureRegion()
	assert.Equal(t, "cn", wizard.region)
}

func TestConfigWizard_EnsureRegion_FromConfig(t *testing.T) {
	// 从配置中读取 region
	cm := newTestConfigManager(t)
	err := cm.SaveRegion("global")
	require.NoError(t, err)

	wizard := newTestWizard(cm, "")
	wizard.ensureRegion()
	assert.Equal(t, "global", wizard.region)
}

// ===== prompt 测试 =====

func TestConfigWizard_Prompt(t *testing.T) {
	// 测试提示输入
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "用户输入\n")

	result := wizard.prompt("请输入")
	assert.Equal(t, "用户输入", result)
}

func TestConfigWizard_Prompt_Empty(t *testing.T) {
	// 测试空输入
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	result := wizard.prompt("请输入")
	assert.Equal(t, "", result)
}

func TestConfigWizard_PromptWithDefault(t *testing.T) {
	// 测试带默认值的提示（使用默认值）
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	result := wizard.promptWithDefault("请输入", "默认值")
	assert.Equal(t, "默认值", result)
}

func TestConfigWizard_PromptWithDefault_Custom(t *testing.T) {
	// 测试带默认值的提示（用户输入自定义值）
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "自定义值\n")

	result := wizard.promptWithDefault("请输入", "默认值")
	assert.Equal(t, "自定义值", result)
}

// ===== selectProvider 测试 =====

func TestConfigWizard_SelectProvider_Default(t *testing.T) {
	// 空输入默认选择 DeepSeek
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	provider, err := wizard.selectProvider()
	require.NoError(t, err)
	assert.Equal(t, "deepseek", provider)
}

func TestConfigWizard_SelectProvider_Claude(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "2\n")

	provider, err := wizard.selectProvider()
	require.NoError(t, err)
	assert.Equal(t, "claude", provider)
}

func TestConfigWizard_SelectProvider_GPT(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "3\n")

	provider, err := wizard.selectProvider()
	require.NoError(t, err)
	assert.Equal(t, "gpt", provider)
}

func TestConfigWizard_SelectProvider_Kimi(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "4\n")

	provider, err := wizard.selectProvider()
	require.NoError(t, err)
	assert.Equal(t, "kimi", provider)
}

func TestConfigWizard_SelectProvider_Gemini(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "5\n")

	provider, err := wizard.selectProvider()
	require.NoError(t, err)
	assert.Equal(t, "gemini", provider)
}

func TestConfigWizard_SelectProvider_GLM(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "6\n")

	provider, err := wizard.selectProvider()
	require.NoError(t, err)
	assert.Equal(t, "glm", provider)
}

func TestConfigWizard_SelectProvider_MiniMax(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "7\n")

	provider, err := wizard.selectProvider()
	require.NoError(t, err)
	assert.Equal(t, "minimax", provider)
}

func TestConfigWizard_SelectProvider_Ollama(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "8\n")

	provider, err := wizard.selectProvider()
	require.NoError(t, err)
	assert.Equal(t, "ollama", provider)
}

func TestConfigWizard_SelectProvider_InvalidThenValid(t *testing.T) {
	// 测试无效输入后重新输入
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "99\n1\n")

	provider, err := wizard.selectProvider()
	require.NoError(t, err)
	assert.Equal(t, "deepseek", provider)
}

// ===== configureOllama 测试 =====

func TestConfigWizard_ConfigureOllama(t *testing.T) {
	// 测试配置本地模型（使用默认值）
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n\n") // 两次回车，使用默认值

	config, err := wizard.configureOllama()
	require.NoError(t, err)
	assert.Equal(t, "ollama", config.Type)
	assert.Equal(t, "http://localhost:11434/v1", config.BaseURL)
	assert.Equal(t, "qwen3:14b", config.Model)
	assert.Equal(t, "", config.APIKey) // 本地服务无需 API Key
}

func TestConfigWizard_ConfigureOllama_Custom(t *testing.T) {
	// 测试自定义配置本地模型
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "http://192.168.1.100:8080/v1\nllama3\n")

	config, err := wizard.configureOllama()
	require.NoError(t, err)
	assert.Equal(t, "http://192.168.1.100:8080/v1", config.BaseURL)
	assert.Equal(t, "llama3", config.Model)
}

// ===== configureDeepSeek 测试 =====

func TestConfigWizard_ConfigureDeepSeek(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "sk-test-deepseek-key\n\n") // 输入 key，模型使用默认

	config, err := wizard.configureDeepSeek()
	require.NoError(t, err)
	assert.Equal(t, "deepseek", config.Type)
	assert.Equal(t, "sk-test-deepseek-key", config.APIKey)
	assert.Equal(t, "deepseek-chat", config.Model)
	assert.Equal(t, "https://api.deepseek.com/v1", config.BaseURL)
}

func TestConfigWizard_ConfigureDeepSeek_EmptyKey(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	_, err := wizard.configureDeepSeek()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API密钥不能为空")
}

func TestConfigWizard_ConfigureDeepSeek_CustomModel(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "sk-key\ndeepseek-reasoner\n")

	config, err := wizard.configureDeepSeek()
	require.NoError(t, err)
	assert.Equal(t, "deepseek-reasoner", config.Model)
}

// ===== configureClaude 测试 =====

func TestConfigWizard_ConfigureClaude(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "sk-ant-test-key\n\n")

	config, err := wizard.configureClaude()
	require.NoError(t, err)
	assert.Equal(t, "claude", config.Type)
	assert.Equal(t, "sk-ant-test-key", config.APIKey)
	assert.Equal(t, "claude-opus-4-20250514", config.Model)
}

func TestConfigWizard_ConfigureClaude_EmptyKey(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	_, err := wizard.configureClaude()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API密钥不能为空")
}

// ===== configureGPT 测试 =====

func TestConfigWizard_ConfigureGPT(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "sk-openai-key\n\n")

	config, err := wizard.configureGPT()
	require.NoError(t, err)
	assert.Equal(t, "gpt", config.Type)
	assert.Equal(t, "sk-openai-key", config.APIKey)
	assert.Equal(t, "gpt-5.4", config.Model)
	assert.Equal(t, "https://api.openai.com/v1", config.BaseURL)
}

func TestConfigWizard_ConfigureGPT_EmptyKey(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	_, err := wizard.configureGPT()
	assert.Error(t, err)
}

// ===== configureKimi 测试 =====

func TestConfigWizard_ConfigureKimi(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "moonshot-key\n\n")

	config, err := wizard.configureKimi()
	require.NoError(t, err)
	assert.Equal(t, "kimi", config.Type)
	assert.Equal(t, "moonshot-key", config.APIKey)
	assert.Equal(t, "k2.5-code", config.Model)
}

func TestConfigWizard_ConfigureKimi_EmptyKey(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	_, err := wizard.configureKimi()
	assert.Error(t, err)
}

// ===== configureGemini 测试 =====

func TestConfigWizard_ConfigureGemini(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "google-api-key\n\n")

	config, err := wizard.configureGemini()
	require.NoError(t, err)
	assert.Equal(t, "gemini", config.Type)
	assert.Equal(t, "google-api-key", config.APIKey)
	assert.Equal(t, "gemini-3.1-pro", config.Model)
}

func TestConfigWizard_ConfigureGemini_EmptyKey(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	_, err := wizard.configureGemini()
	assert.Error(t, err)
}

// ===== configureGLM 测试 =====

func TestConfigWizard_ConfigureGLM(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "glm-api-key\n\n")

	config, err := wizard.configureGLM()
	require.NoError(t, err)
	assert.Equal(t, "glm", config.Type)
	assert.Equal(t, "glm-api-key", config.APIKey)
	assert.Equal(t, "glm-5.1", config.Model)
	assert.Equal(t, "https://open.bigmodel.cn/api/anthropic", config.BaseURL)
}

func TestConfigWizard_ConfigureGLM_EmptyKey(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	_, err := wizard.configureGLM()
	assert.Error(t, err)
}

// ===== configureMiniMax 测试 =====

func TestConfigWizard_ConfigureMiniMax_CN(t *testing.T) {
	// 测试中国区域 MiniMax 配置
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "minimax-key\n\n")
	wizard.region = "cn"

	config, err := wizard.configureMiniMax()
	require.NoError(t, err)
	assert.Equal(t, "minimax", config.Type)
	assert.Equal(t, "https://api.minimaxi.com/anthropic", config.BaseURL)
}

func TestConfigWizard_ConfigureMiniMax_Global(t *testing.T) {
	// 测试全球区域 MiniMax 配置
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "minimax-key\n\n")
	wizard.region = "global"

	config, err := wizard.configureMiniMax()
	require.NoError(t, err)
	assert.Equal(t, "https://api.minimax.io/anthropic", config.BaseURL)
}

func TestConfigWizard_ConfigureMiniMax_EmptyKey(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	_, err := wizard.configureMiniMax()
	assert.Error(t, err)
}

// ===== Run 完整流程测试 =====

func TestConfigWizard_Run_DeepSeek(t *testing.T) {
	// 测试完整 DeepSeek 配置流程
	cm := newTestConfigManager(t)
	// 输入序列：提供商选择(1-DeepSeek), API Key, 模型(默认)
	// 注意：region 已预设为 "cn"，ensureRegion 会跳过区域选择
	wizard := newTestWizard(cm, "1\ntest-api-key-12345\n\n")
	wizard.region = "cn" // 预设区域跳过区域选择

	err := wizard.Run()
	require.NoError(t, err)

	// 验证配置已保存
	config, err := cm.Get("deepseek")
	require.NoError(t, err)
	assert.Equal(t, "test-api-key-12345", config.APIKey)
	assert.Equal(t, "deepseek", config.Type)
}

func TestConfigWizard_Run_Ollama(t *testing.T) {
	// 测试完整 Ollama 配置流程
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "8\n\n\n") // 选择8-Ollama，默认地址，默认模型
	wizard.region = "cn"

	err := wizard.Run()
	require.NoError(t, err)

	config, err := cm.Get("ollama")
	require.NoError(t, err)
	assert.Equal(t, "ollama", config.Type)
	assert.Equal(t, "", config.APIKey)
}

func TestConfigWizard_Run_Claude(t *testing.T) {
	// 测试完整 Claude 配置流程
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "2\ntest-claude-key-12345\n\n")
	wizard.region = "cn"

	err := wizard.Run()
	require.NoError(t, err)

	config, err := cm.Get("claude")
	require.NoError(t, err)
	assert.Equal(t, "claude", config.Type)
	assert.Equal(t, "test-claude-key-12345", config.APIKey)
}

func TestConfigWizard_Run_GPT(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "3\ntest-gpt-key-1234567890\n\n")
	wizard.region = "cn"

	err := wizard.Run()
	require.NoError(t, err)

	config, err := cm.Get("gpt")
	require.NoError(t, err)
	assert.Equal(t, "gpt", config.Type)
}

func TestConfigWizard_Run_Kimi(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "4\ntest-kimi-key-12345\n\n")
	wizard.region = "cn"

	err := wizard.Run()
	require.NoError(t, err)

	config, err := cm.Get("kimi")
	require.NoError(t, err)
	assert.Equal(t, "kimi", config.Type)
}

func TestConfigWizard_Run_Gemini(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "5\ntest-gemini-key-12345\n\n")
	wizard.region = "cn"

	err := wizard.Run()
	require.NoError(t, err)

	config, err := cm.Get("gemini")
	require.NoError(t, err)
	assert.Equal(t, "gemini", config.Type)
}

func TestConfigWizard_Run_GLM(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "6\ntest-glm-key-12345\n\n")
	wizard.region = "cn"

	err := wizard.Run()
	require.NoError(t, err)

	config, err := cm.Get("glm")
	require.NoError(t, err)
	assert.Equal(t, "glm", config.Type)
}

func TestConfigWizard_Run_MiniMax(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "7\ntest-minimax-key-12345\n\n")
	wizard.region = "cn"

	err := wizard.Run()
	require.NoError(t, err)

	config, err := cm.Get("minimax")
	require.NoError(t, err)
	assert.Equal(t, "minimax", config.Type)
}

func TestConfigWizard_Run_EmptyKeyError(t *testing.T) {
	// 测试空 API Key 返回错误
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "1\n\n") // 选择 DeepSeek 但空 key
	wizard.region = "cn"

	err := wizard.Run()
	assert.Error(t, err)
}

// ===== selectRegionWithDetect 测试 =====

func TestConfigWizard_SelectRegionWithDetect_CN(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "1\n")

	wizard.selectRegionWithDetect("cn")
	assert.Equal(t, "cn", wizard.region)
}

func TestConfigWizard_SelectRegionWithDetect_Global(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "2\n")

	wizard.selectRegionWithDetect("global")
	assert.Equal(t, "global", wizard.region)
}

func TestConfigWizard_SelectRegionWithDetect_Default(t *testing.T) {
	// 空输入使用检测到的默认值（cn）
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	wizard.selectRegionWithDetect("cn")
	assert.Equal(t, "cn", wizard.region)
}

func TestConfigWizard_SelectRegionWithDetect_DefaultGlobal(t *testing.T) {
	// 空输入使用检测到的默认值（global）
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "\n")

	wizard.selectRegionWithDetect("global")
	assert.Equal(t, "global", wizard.region)
}

func TestConfigWizard_SelectRegionWithDetect_InvalidThenValid(t *testing.T) {
	// 无效输入后重新选择
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "3\n1\n")

	wizard.selectRegionWithDetect("cn")
	assert.Equal(t, "cn", wizard.region)
}

// ===== Run 时 ensureRegion 触发区域选择 =====

func TestConfigWizard_Run_WithRegionSelection(t *testing.T) {
	// 不预设 region，让 ensureRegion 触发
	cm := newTestConfigManager(t)
	// 区域选择(1), 提供商(8-Ollama), 地址(默认), 模型(默认)
	wizard := newTestWizard(cm, "1\n8\n\n\n")

	err := wizard.Run()
	require.NoError(t, err)
	assert.Equal(t, "cn", wizard.region)
}

// ===== printWelcome / printSuccess 测试 =====

func TestConfigWizard_PrintWelcome(t *testing.T) {
	// 仅验证不 panic
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "")
	assert.NotPanics(t, func() {
		wizard.printWelcome()
	})
}

func TestConfigWizard_PrintSuccess(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "")
	assert.NotPanics(t, func() {
		wizard.printSuccess("deepseek")
	})
}

// ===== 保存默认 provider 验证 =====

func TestConfigWizard_Run_SavesDefaultProvider(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "8\n\n\n")
	wizard.region = "cn"

	err := wizard.Run()
	require.NoError(t, err)

	// 验证默认 provider 已设置
	defaultProvider := cm.GetDefaultProvider()
	assert.Equal(t, "ollama", defaultProvider)
}

// ===== 文件验证 =====

func TestConfigWizard_Run_SavesConfigFile(t *testing.T) {
	cm := newTestConfigManager(t)
	wizard := newTestWizard(cm, "8\n\n\n")
	wizard.region = "cn"

	err := wizard.Run()
	require.NoError(t, err)

	// 验证配置文件已创建
	configFile := filepath.Join(cm.configDir, "ollama.json")
	_, statErr := os.Stat(configFile)
	assert.NoError(t, statErr, "配置文件应该被创建")
}
