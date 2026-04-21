package llm

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ProviderFactory 创建 LLM 客户端的工厂函数
type ProviderFactory func(config *ProviderConfig) (Client, error)

// providerRegistry 全局 provider 注册表
var providerRegistry = struct {
	sync.RWMutex
	factories map[string]ProviderFactory
}{
	factories: make(map[string]ProviderFactory),
}

// RegisterProvider 注册 LLM 提供商工厂
func RegisterProvider(providerType string, factory ProviderFactory) {
	providerRegistry.Lock()
	defer providerRegistry.Unlock()
	providerRegistry.factories[providerType] = factory
}

// getProvider 获取已注册的提供商工厂
func getProvider(providerType string) (ProviderFactory, bool) {
	providerRegistry.RLock()
	defer providerRegistry.RUnlock()
	f, ok := providerRegistry.factories[providerType]
	return f, ok
}

// getRegisteredProviders 获取所有已注册的提供商类型（按字母排序）
func getRegisteredProviders() []string {
	providerRegistry.RLock()
	defer providerRegistry.RUnlock()
	types := make([]string, 0, len(providerRegistry.factories))
	for t := range providerRegistry.factories {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

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
	factory, ok := getProvider(config.Type)
	if !ok {
		available := getRegisteredProviders()
		return nil, fmt.Errorf("不支持的提供商类型: %s (已注册: %s)", config.Type, strings.Join(available, ", "))
	}
	return factory(config)
}

// ListProviders 列出所有可用的提供商
func (f *ClientFactory) ListProviders() []string {
	return f.configManager.List()
}
