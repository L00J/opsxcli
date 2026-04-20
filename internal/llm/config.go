package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProviderConfig 提供商配置
type ProviderConfig struct {
	Type        string            `json:"type"`         // deepseek, ollama, claude
	BaseURL     string            `json:"base_url"`     // API基础URL
	APIKey      string            `json:"api_key"`      // API密钥
	Model       string            `json:"model"`        // 模型名称
	Temperature float64           `json:"temperature"`  // 温度参数
	MaxTokens   int               `json:"max_tokens"`   // 最大token数
	Extra       map[string]string `json:"extra,omitempty"` // 额外参数
}

// ConfigManager 配置管理器
type ConfigManager struct {
	configDir string
	configs   map[string]*ProviderConfig
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configDir string) (*ConfigManager, error) {
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户主目录失败: %w", err)
		}
		configDir = filepath.Join(homeDir, ".opsxcli", "providers")
	}

	// 确保目录存在
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %w", err)
	}

	cm := &ConfigManager{
		configDir: configDir,
		configs:   make(map[string]*ProviderConfig),
	}

	// 加载所有配置文件
	if err := cm.loadAll(); err != nil {
		return nil, err
	}

	return cm, nil
}

// loadAll 加载所有配置
func (cm *ConfigManager) loadAll() error {
	entries, err := os.ReadDir(cm.configDir)
	if err != nil {
		return fmt.Errorf("读取配置目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		name := entry.Name()[:len(entry.Name())-5] // 去掉.json后缀
		config, err := cm.load(name)
		if err != nil {
			fmt.Printf("警告: 加载配置 %s 失败: %v\n", name, err)
			continue
		}
		cm.configs[name] = config
	}

	return nil
}

// load 加载单个配置
func (cm *ConfigManager) load(name string) (*ProviderConfig, error) {
	path := filepath.Join(cm.configDir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ProviderConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &config, nil
}

// Get 获取配置
func (cm *ConfigManager) Get(name string) (*ProviderConfig, error) {
	config, ok := cm.configs[name]
	if !ok {
		return nil, fmt.Errorf("配置 %s 不存在", name)
	}
	return config, nil
}

// List 列出所有配置（按字母排序，确保顺序稳定）
func (cm *ConfigManager) List() []string {
	names := make([]string, 0, len(cm.configs))
	for name := range cm.configs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Save 保存配置
func (cm *ConfigManager) Save(name string, config *ProviderConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	path := filepath.Join(cm.configDir, name+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	cm.configs[name] = config
	return nil
}

// placeholderPatterns 占位符 API key 模式（不区分大小写匹配）
var placeholderPatterns = []string{
	"YOUR_",
	"_API_KEY",
	"placeholder",
	"changeme",
	"example",
	"xxx",
	"test",
	"demo",
	"sk-test",
	"sk-xxx",
}

// IsValid 检查配置是否有效（非占位符）
func (c *ProviderConfig) IsValid() bool {
	// 本地服务（Ollama/vLLM）不需要 API key
	if c.Type == "ollama" {
		return true
	}
	if c.BaseURL != "" && (c.APIKey == "" || c.APIKey == "ollama") {
		// 本地服务（vLLM 等）无需 API key
		return true
	}

	// 云服务必须有效 key
	if c.APIKey == "" {
		return false
	}

	key := strings.ToUpper(c.APIKey)
	for _, pattern := range placeholderPatterns {
		if strings.Contains(key, strings.ToUpper(pattern)) {
			return false
		}
	}

	// key 太短也不行（至少 10 字符）
	if len(c.APIKey) < 10 {
		return false
	}

	return true
}

// GetValidProviders 返回所有有效（非占位符）的 provider 名称（按字母排序）
func (cm *ConfigManager) GetValidProviders() []string {
	var valid []string
	for name, config := range cm.configs {
		if config.IsValid() {
			valid = append(valid, name)
		}
	}
	sort.Strings(valid)
	return valid
}

// HasAnyValidProvider 检查是否有任何有效配置
func (cm *ConfigManager) HasAnyValidProvider() bool {
	return len(cm.GetValidProviders()) > 0
}

// globalConfigPath 返回全局配置文件路径 (~/.opsxcli/config.json)
func globalConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(homeDir, ".opsxcli", "config.json"), nil
}

// globalConfig 全局配置结构
type globalConfig struct {
	DefaultProvider string `json:"default_provider,omitempty"`
	Region          string `json:"region,omitempty"` // "cn" 或 "global"
}

// SaveDefaultProvider 保存默认 provider 到 ~/.opsxcli/config.json
func (cm *ConfigManager) SaveDefaultProvider(name string) error {
	configPath, err := globalConfigPath()
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 读取已有配置（保留其他字段）
	var gc globalConfig
	data, err := os.ReadFile(configPath)
	if err == nil {
		_ = json.Unmarshal(data, &gc)
	}

	gc.DefaultProvider = name

	data, err = json.MarshalIndent(gc, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("保存默认 provider 失败: %w", err)
	}

	return nil
}

// GetDefaultProvider 从 ~/.opsxcli/config.json 读取默认 provider
func (cm *ConfigManager) GetDefaultProvider() string {
	configPath, err := globalConfigPath()
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	var gc globalConfig
	if err := json.Unmarshal(data, &gc); err != nil {
		return ""
	}

	return gc.DefaultProvider
}

// SaveRegion 保存区域设置到 ~/.opsxcli/config.json
func (cm *ConfigManager) SaveRegion(region string) error {
	configPath, err := globalConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	var gc globalConfig
	data, err := os.ReadFile(configPath)
	if err == nil {
		_ = json.Unmarshal(data, &gc)
	}

	gc.Region = region

	data, err = json.MarshalIndent(gc, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("保存区域设置失败: %w", err)
	}

	return nil
}

// GetRegion 从 ~/.opsxcli/config.json 读取区域设置
func (cm *ConfigManager) GetRegion() string {
	configPath, err := globalConfigPath()
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	var gc globalConfig
	if err := json.Unmarshal(data, &gc); err != nil {
		return ""
	}

	return gc.Region
}

// CreateDefaultConfigs 创建默认配置示例
func (cm *ConfigManager) CreateDefaultConfigs() error {
	// 1. DeepSeek 配置（默认推荐）
	deepseekConfig := &ProviderConfig{
		Type:        "deepseek",
		BaseURL:     "https://api.deepseek.com/v1",
		APIKey:      "YOUR_DEEPSEEK_API_KEY",
		Model:       "deepseek-chat",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// 2. Claude 配置（代码之王）
	claudeConfig := &ProviderConfig{
		Type:        "claude",
		BaseURL:     "https://api.anthropic.com/v1",
		APIKey:      "YOUR_CLAUDE_API_KEY",
		Model:       "claude-opus-4-20250514",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// 3. ChatGPT 配置（行业标杆）
	gptConfig := &ProviderConfig{
		Type:        "gpt",
		BaseURL:     "https://api.openai.com/v1",
		APIKey:      "YOUR_OPENAI_API_KEY",
		Model:       "gpt-5.4",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// 4. Kimi 配置（长文本专家）
	kimiConfig := &ProviderConfig{
		Type:        "kimi",
		BaseURL:     "https://api.moonshot.cn/v1",
		APIKey:      "YOUR_KIMI_API_KEY",
		Model:       "k2.5-code",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// 5. Gemini 配置（Google AI）
	geminiConfig := &ProviderConfig{
		Type:        "gemini",
		BaseURL:     "https://generativelanguage.googleapis.com/v1beta",
		APIKey:      "YOUR_GEMINI_API_KEY",
		Model:       "gemini-3.1-pro",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// 6. GLM (智谱) 配置 — Anthropic 兼容接口
	glmConfig := &ProviderConfig{
		Type:        "glm",
		BaseURL:     "https://open.bigmodel.cn/api/anthropic",
		APIKey:      "YOUR_GLM_API_KEY",
		Model:       "glm-5.1",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// 7. MiniMax 配置 — Anthropic 兼容接口
	minimaxConfig := &ProviderConfig{
		Type:        "minimax",
		BaseURL:     "https://api.minimaxi.com/anthropic",
		APIKey:      "YOUR_MINIMAX_API_KEY",
		Model:       "MiniMax-M2.7",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// 8. 本地模型服务（Ollama/vLLM等）
	ollamaConfig := &ProviderConfig{
		Type:        "ollama",
		BaseURL:     "http://localhost:11434/v1",
		APIKey:      "",
		Model:       "qwen3:14b",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	configs := map[string]*ProviderConfig{
		"deepseek": deepseekConfig,
		"claude":   claudeConfig,
		"gpt":      gptConfig,
		"kimi":     kimiConfig,
		"gemini":   geminiConfig,
		"glm":      glmConfig,
		"minimax":  minimaxConfig,
		"ollama":   ollamaConfig,
	}

	for name, config := range configs {
		path := filepath.Join(cm.configDir, name+".json")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := cm.Save(name, config); err != nil {
				return err
			}
		}
	}

	return nil
}
