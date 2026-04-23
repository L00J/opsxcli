// hotswap.go - LLM Provider 热切换
// 允许在 Agent 运行时无缝切换 LLM 提供商，保留对话上下文
// 使用场景：当前 Provider 失败时降级到备用，或用户手动切换模型
package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProviderHealth Provider 健康状态
type ProviderHealth struct {
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	Healthy   bool          `json:"healthy"`
	Latency   time.Duration `json:"latency"`
	Error     string        `json:"error,omitempty"`
	LastCheck time.Time     `json:"last_check"`
}

// HotSwapEvent 热切换事件
type HotSwapEvent struct {
	FromProvider string    `json:"from_provider"`
	ToProvider   string    `json:"to_provider"`
	Reason       string    `json:"reason"` // "manual", "failover", "health_check"
	Timestamp    time.Time `json:"timestamp"`
}

// HotSwapCallback 热切换通知回调
type HotSwapCallback func(event HotSwapEvent)

// HotSwappableClient 支持运行时切换 LLM Provider 的客户端
// 实现 Client 接口，内部代理到当前活跃的底层 Client
type HotSwappableClient struct {
	mu          sync.RWMutex
	current     Client          // 当前活跃的 Client
	currentName string          // 当前 Provider 名称
	factory     *ClientFactory  // 客户端工厂，用于创建新 Client
	history     []HotSwapEvent  // 切换历史
	callback    HotSwapCallback // 切换通知回调
}

// NewHotSwappableClient 创建支持热切换的 Client
// 初始 client 为默认 Provider，factory 用于后续创建新 Client
func NewHotSwappableClient(name string, client Client, factory *ClientFactory) *HotSwappableClient {
	return &HotSwappableClient{
		current:     client,
		currentName: name,
		factory:     factory,
		history:     make([]HotSwapEvent, 0),
	}
}

// Complete 实现 Client 接口 - 代理到当前活跃的 Provider
func (h *HotSwappableClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	h.mu.RLock()
	client := h.current
	h.mu.RUnlock()

	resp, err := client.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("[%s] %w", h.currentName, err)
	}
	return resp, nil
}

// Stream 实现 Client 接口 - 代理到当前活跃的 Provider
func (h *HotSwappableClient) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	h.mu.RLock()
	client := h.current
	h.mu.RUnlock()

	ch, err := client.Stream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("[%s] %w", h.currentName, err)
	}
	return ch, nil
}

// Name 实现 Client 接口 - 返回当前 Provider 名称
func (h *HotSwappableClient) Name() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentName
}

// SwitchProvider 切换到指定的 Provider
// 保留对话上下文（由 Agent 的 messages slice 维护），仅替换底层 Client
func (h *HotSwappableClient) SwitchProvider(name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if name == h.currentName {
		return nil // 已经是当前 Provider
	}

	newClient, err := h.factory.Create(name)
	if err != nil {
		return fmt.Errorf("创建 Provider %s 失败: %w", name, err)
	}

	event := HotSwapEvent{
		FromProvider: h.currentName,
		ToProvider:   name,
		Reason:       "manual",
		Timestamp:    time.Now(),
	}

	h.current = newClient
	h.currentName = name
	h.history = append(h.history, event)

	// 通知回调
	if h.callback != nil {
		h.callback(event)
	}

	return nil
}

// SwitchProviderWithReason 带原因切换 Provider（用于 failover 等场景）
func (h *HotSwappableClient) SwitchProviderWithReason(name, reason string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if name == h.currentName {
		return nil
	}

	newClient, err := h.factory.Create(name)
	if err != nil {
		return fmt.Errorf("创建 Provider %s 失败: %w", name, err)
	}

	event := HotSwapEvent{
		FromProvider: h.currentName,
		ToProvider:   name,
		Reason:       reason,
		Timestamp:    time.Now(),
	}

	h.current = newClient
	h.currentName = name
	h.history = append(h.history, event)

	if h.callback != nil {
		h.callback(event)
	}

	return nil
}

// CurrentProvider 返回当前 Provider 名称
func (h *HotSwappableClient) CurrentProvider() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentName
}

// AvailableProviders 返回所有可用的 Provider 名称列表
func (h *HotSwappableClient) AvailableProviders() []string {
	return h.factory.ListProviders()
}

// SwitchHistory 返回切换历史
func (h *HotSwappableClient) SwitchHistory() []HotSwapEvent {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]HotSwapEvent, len(h.history))
	copy(result, h.history)
	return result
}

// SetCallback 设置热切换通知回调
func (h *HotSwappableClient) SetCallback(cb HotSwapCallback) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callback = cb
}

// CheckHealth 检查指定 Provider 的健康状态
// 发送一个最小化请求来验证可用性
func (h *HotSwappableClient) CheckHealth(ctx context.Context, name string) ProviderHealth {
	start := time.Now()
	health := ProviderHealth{
		Name:      name,
		LastCheck: start,
	}

	client, err := h.factory.Create(name)
	if err != nil {
		health.Error = err.Error()
		health.Latency = time.Since(start)
		return health
	}

	// 从配置中获取类型信息
	h.mu.RLock()
	health.Type = "unknown"
	h.mu.RUnlock()

	// 发送一个最小请求来检查可用性
	// 使用 MaxTokens=1 来最小化消耗
	_, err = client.Complete(ctx, &CompletionRequest{
		Messages:    []Message{{Role: "user", Content: "hi"}},
		MaxTokens:   1,
		Temperature: 0,
	})
	health.Latency = time.Since(start)

	if err != nil {
		health.Error = err.Error()
		health.Healthy = false
	} else {
		health.Healthy = true
	}

	return health
}

// CheckAllHealth 检查所有已配置 Provider 的健康状态
func (h *HotSwappableClient) CheckAllHealth(ctx context.Context) []ProviderHealth {
	providers := h.AvailableProviders()
	results := make([]ProviderHealth, 0, len(providers))

	for _, name := range providers {
		health := h.CheckHealth(ctx, name)
		results = append(results, health)
	}

	return results
}

// FailoverOnFailure 当当前 Provider 失败时自动降级到备用
// 尝试顺序：按 AvailableProviders 返回的顺序，跳过当前 Provider
// 返回切换后的 Provider 名称，如果所有都失败则返回错误
func (h *HotSwappableClient) FailoverOnFailure(ctx context.Context, lastErr error) (string, error) {
	providers := h.AvailableProviders()
	currentName := h.CurrentProvider()

	for _, name := range providers {
		if name == currentName {
			continue
		}

		// 检查健康状态
		health := h.CheckHealth(ctx, name)
		if health.Healthy {
			err := h.SwitchProviderWithReason(name, "failover")
			if err != nil {
				continue
			}
			return name, nil
		}
	}

	return "", fmt.Errorf("所有 Provider 均不可用 (最近错误: %w)", lastErr)
}
