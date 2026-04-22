package llm

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// mockClient 用于测试的模拟 LLM Client
type mockClient struct {
	name    string
	fail    bool
	latency time.Duration
}

func (m *mockClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if m.fail {
		return nil, fmt.Errorf("mock client %s 不可用", m.name)
	}
	if m.latency > 0 {
		select {
		case <-time.After(m.latency):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return &CompletionResponse{
		ID:    "test-id",
		Model: m.name,
		Message: Message{
			Role:    "assistant",
			Content: fmt.Sprintf("来自 %s 的响应", m.name),
		},
		Usage: Usage{TotalTokens: 10},
	}, nil
}

func (m *mockClient) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	if m.fail {
		return nil, fmt.Errorf("mock client %s 不可用", m.name)
	}
	ch := make(chan StreamChunk, 1)
	ch <- StreamChunk{
		Delta:   Message{Role: "assistant", Content: fmt.Sprintf("来自 %s 的流式响应", m.name)},
		Finish:  true,
	}
	close(ch)
	return ch, nil
}

func (m *mockClient) Name() string { return m.name }

// mockFactory 用于测试的模拟工厂
type mockFactory struct {
	clients map[string]*mockClient
}

func newMockFactory() *mockFactory {
	return &mockFactory{
		clients: make(map[string]*mockClient),
	}
}

func (f *mockFactory) addMockClient(name string, fail bool) {
	f.clients[name] = &mockClient{name: name, fail: fail}
}

// 实现 ClientFactory 类似接口的方法
// 注意：测试中我们直接构造 HotSwappableClient 而不依赖真正的 ClientFactory

// TestHotSwappableClient_SwitchProvider 测试手动切换 Provider
func TestHotSwappableClient_SwitchProvider(t *testing.T) {
	// 创建模拟 factory
	dir := t.TempDir()
	cm, err := NewConfigManager(dir)
	if err != nil {
		t.Fatalf("创建 ConfigManager 失败: %v", err)
	}

	// 注册测试 provider
	RegisterProvider("test_swap_a", func(config *ProviderConfig) (Client, error) {
		return &mockClient{name: "provider-a"}, nil
	})
	RegisterProvider("test_swap_b", func(config *ProviderConfig) (Client, error) {
		return &mockClient{name: "provider-b"}, nil
	})

	// 保存配置
	cm.Save("provider-a", &ProviderConfig{Type: "test_swap_a", Model: "model-a"})
	cm.Save("provider-b", &ProviderConfig{Type: "test_swap_b", Model: "model-b"})

	factory := NewClientFactory(cm)
	clientA, _ := factory.Create("provider-a")

	hsc := NewHotSwappableClient("provider-a", clientA, factory)

	// 验证初始 Provider
	if hsc.Name() != "provider-a" {
		t.Errorf("初始 Provider 应为 provider-a，实际为 %s", hsc.Name())
	}
	if hsc.CurrentProvider() != "provider-a" {
		t.Errorf("CurrentProvider() 应为 provider-a，实际为 %s", hsc.CurrentProvider())
	}

	// 切换到 provider-b
	err = hsc.SwitchProvider("provider-b")
	if err != nil {
		t.Fatalf("切换到 provider-b 失败: %v", err)
	}

	if hsc.Name() != "provider-b" {
		t.Errorf("切换后 Provider 应为 provider-b，实际为 %s", hsc.Name())
	}

	// 切换历史应有记录
	history := hsc.SwitchHistory()
	if len(history) != 1 {
		t.Fatalf("应有 1 条切换记录，实际为 %d", len(history))
	}
	if history[0].FromProvider != "provider-a" {
		t.Errorf("FromProvider 应为 provider-a，实际为 %s", history[0].FromProvider)
	}
	if history[0].ToProvider != "provider-b" {
		t.Errorf("ToProvider 应为 provider-b，实际为 %s", history[0].ToProvider)
	}
	if history[0].Reason != "manual" {
		t.Errorf("Reason 应为 manual，实际为 %s", history[0].Reason)
	}
}

// TestHotSwappableClient_SwitchToSame 测试切换到当前 Provider 为空操作
func TestHotSwappableClient_SwitchToSame(t *testing.T) {
	client := &mockClient{name: "current"}
	hsc := &HotSwappableClient{
		current:     client,
		currentName: "current",
		history:     make([]HotSwapEvent, 0),
	}

	err := hsc.SwitchProvider("current")
	if err != nil {
		t.Fatalf("切换到相同 Provider 不应报错: %v", err)
	}

	history := hsc.SwitchHistory()
	if len(history) != 0 {
		t.Errorf("切换到相同 Provider 不应产生历史记录，实际有 %d 条", len(history))
	}
}

// TestHotSwappableClient_SwitchToNonExistent 测试切换到不存在的 Provider
func TestHotSwappableClient_SwitchToNonExistent(t *testing.T) {
	dir := t.TempDir()
	cm, _ := NewConfigManager(dir)
	factory := NewClientFactory(cm)

	client := &mockClient{name: "default"}
	hsc := NewHotSwappableClient("default", client, factory)

	err := hsc.SwitchProvider("nonexistent")
	if err == nil {
		t.Error("切换到不存在的 Provider 应报错")
	}
}

// TestHotSwappableClient_Complete 测试 Complete 代理到当前 Provider
func TestHotSwappableClient_Complete(t *testing.T) {
	client := &mockClient{name: "test-provider"}
	hsc := &HotSwappableClient{
		current:     client,
		currentName: "test-provider",
		history:     make([]HotSwapEvent, 0),
	}

	resp, err := hsc.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "测试"}},
	})
	if err != nil {
		t.Fatalf("Complete 失败: %v", err)
	}
	if resp.Message.Content != "来自 test-provider 的响应" {
		t.Errorf("响应内容不正确: %s", resp.Message.Content)
	}
}

// TestHotSwappableClient_Stream 测试 Stream 代理到当前 Provider
func TestHotSwappableClient_Stream(t *testing.T) {
	client := &mockClient{name: "stream-provider"}
	hsc := &HotSwappableClient{
		current:     client,
		currentName: "stream-provider",
		history:     make([]HotSwapEvent, 0),
	}

	ch, err := hsc.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "测试流式"}},
	})
	if err != nil {
		t.Fatalf("Stream 失败: %v", err)
	}

	chunk, ok := <-ch
	if !ok {
		t.Fatal("Stream 通道应可读取")
	}
	if chunk.Delta.Content != "来自 stream-provider 的流式响应" {
		t.Errorf("流式响应内容不正确: %s", chunk.Delta.Content)
	}
}

// TestHotSwappableClient_CompleteError 测试 Complete 错误传递
func TestHotSwappableClient_CompleteError(t *testing.T) {
	client := &mockClient{name: "failing", fail: true}
	hsc := &HotSwappableClient{
		current:     client,
		currentName: "failing",
		history:     make([]HotSwapEvent, 0),
	}

	_, err := hsc.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "测试"}},
	})
	if err == nil {
		t.Error("期望错误但没有返回")
	}
	// 错误信息应包含 Provider 名称
	if err.Error()[:3] != "[fa" {
		t.Errorf("错误信息应包含 Provider 名称前缀: %s", err.Error())
	}
}

// TestHotSwappableClient_Callback 测试切换通知回调
func TestHotSwappableClient_Callback(t *testing.T) {
	dir := t.TempDir()
	cm, _ := NewConfigManager(dir)

	RegisterProvider("test_cb_a", func(config *ProviderConfig) (Client, error) {
		return &mockClient{name: "cb-a"}, nil
	})
	RegisterProvider("test_cb_b", func(config *ProviderConfig) (Client, error) {
		return &mockClient{name: "cb-b"}, nil
	})

	cm.Save("cb-a", &ProviderConfig{Type: "test_cb_a", Model: "m-a"})
	cm.Save("cb-b", &ProviderConfig{Type: "test_cb_b", Model: "m-b"})

	factory := NewClientFactory(cm)
	clientA, _ := factory.Create("cb-a")

	hsc := NewHotSwappableClient("cb-a", clientA, factory)

	var mu sync.Mutex
	var receivedEvent *HotSwapEvent

	hsc.SetCallback(func(event HotSwapEvent) {
		mu.Lock()
		defer mu.Unlock()
		receivedEvent = &event
	})

	err := hsc.SwitchProvider("cb-b")
	if err != nil {
		t.Fatalf("切换失败: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if receivedEvent == nil {
		t.Fatal("回调未被触发")
	}
	if receivedEvent.FromProvider != "cb-a" {
		t.Errorf("FromProvider 应为 cb-a，实际为 %s", receivedEvent.FromProvider)
	}
	if receivedEvent.ToProvider != "cb-b" {
		t.Errorf("ToProvider 应为 cb-b，实际为 %s", receivedEvent.ToProvider)
	}
	if receivedEvent.Reason != "manual" {
		t.Errorf("Reason 应为 manual，实际为 %s", receivedEvent.Reason)
	}
}

// TestHotSwappableClient_MultipleSwitches 测试多次切换
func TestHotSwappableClient_MultipleSwitches(t *testing.T) {
	dir := t.TempDir()
	cm, _ := NewConfigManager(dir)

	RegisterProvider("test_multi_a", func(config *ProviderConfig) (Client, error) {
		return &mockClient{name: "multi-a"}, nil
	})
	RegisterProvider("test_multi_b", func(config *ProviderConfig) (Client, error) {
		return &mockClient{name: "multi-b"}, nil
	})
	RegisterProvider("test_multi_c", func(config *ProviderConfig) (Client, error) {
		return &mockClient{name: "multi-c"}, nil
	})

	cm.Save("ma", &ProviderConfig{Type: "test_multi_a", Model: "m"})
	cm.Save("mb", &ProviderConfig{Type: "test_multi_b", Model: "m"})
	cm.Save("mc", &ProviderConfig{Type: "test_multi_c", Model: "m"})

	factory := NewClientFactory(cm)
	clientA, _ := factory.Create("ma")

	hsc := NewHotSwappableClient("ma", clientA, factory)

	// ma -> mb -> mc -> ma
	hsc.SwitchProvider("mb")
	hsc.SwitchProvider("mc")
	hsc.SwitchProvider("ma")

	history := hsc.SwitchHistory()
	if len(history) != 3 {
		t.Fatalf("应有 3 条切换记录，实际为 %d", len(history))
	}

	// 验证切换链
	expectedChain := []struct{ from, to string }{
		{"ma", "mb"},
		{"mb", "mc"},
		{"mc", "ma"},
	}
	for i, expected := range expectedChain {
		if history[i].FromProvider != expected.from || history[i].ToProvider != expected.to {
			t.Errorf("第 %d 次切换: 期望 %s->%s，实际 %s->%s",
				i+1, expected.from, expected.to,
				history[i].FromProvider, history[i].ToProvider)
		}
	}
}

// TestHotSwappableClient_SwitchWithReason 测试带原因切换
func TestHotSwappableClient_SwitchWithReason(t *testing.T) {
	dir := t.TempDir()
	cm, _ := NewConfigManager(dir)

	RegisterProvider("test_reason_a", func(config *ProviderConfig) (Client, error) {
		return &mockClient{name: "r-a"}, nil
	})
	RegisterProvider("test_reason_b", func(config *ProviderConfig) (Client, error) {
		return &mockClient{name: "r-b"}, nil
	})

	cm.Save("ra", &ProviderConfig{Type: "test_reason_a", Model: "m"})
	cm.Save("rb", &ProviderConfig{Type: "test_reason_b", Model: "m"})

	factory := NewClientFactory(cm)
	clientA, _ := factory.Create("ra")

	hsc := NewHotSwappableClient("ra", clientA, factory)

	err := hsc.SwitchProviderWithReason("rb", "failover")
	if err != nil {
		t.Fatalf("切换失败: %v", err)
	}

	history := hsc.SwitchHistory()
	if len(history) != 1 {
		t.Fatalf("应有 1 条记录，实际为 %d", len(history))
	}
	if history[0].Reason != "failover" {
		t.Errorf("Reason 应为 failover，实际为 %s", history[0].Reason)
	}
}

// TestHotSwappableClient_AvailableProviders 测试列出可用 Provider
func TestHotSwappableClient_AvailableProviders(t *testing.T) {
	dir := t.TempDir()
	cm, _ := NewConfigManager(dir)

	cm.Save("provider-1", &ProviderConfig{Type: "test", Model: "m1"})
	cm.Save("provider-2", &ProviderConfig{Type: "test", Model: "m2"})
	cm.Save("provider-3", &ProviderConfig{Type: "test", Model: "m3"})

	RegisterProvider("test", func(config *ProviderConfig) (Client, error) {
		return &mockClient{name: "test"}, nil
	})

	factory := NewClientFactory(cm)
	client, _ := factory.Create("provider-1")

	hsc := NewHotSwappableClient("provider-1", client, factory)

	providers := hsc.AvailableProviders()
	if len(providers) != 3 {
		t.Errorf("应有 3 个 Provider，实际为 %d: %v", len(providers), providers)
	}
}

// TestHotSwappableClient_ConcurrentAccess 测试并发访问安全性
func TestHotSwappableClient_ConcurrentAccess(t *testing.T) {
	client := &mockClient{name: "concurrent"}
	hsc := &HotSwappableClient{
		current:     client,
		currentName: "concurrent",
		history:     make([]HotSwapEvent, 0),
	}

	var wg sync.WaitGroup
	errors := make(chan error, 20)

	// 并发读取
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = hsc.Name()
			_ = hsc.CurrentProvider()
			_, err := hsc.Complete(context.Background(), &CompletionRequest{
				Messages: []Message{{Role: "user", Content: "并发测试"}},
			})
			if err != nil {
				errors <- err
			}
		}()
	}

	// 并发写入（设置回调）
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			hsc.SetCallback(func(event HotSwapEvent) {})
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("并发访问出错: %v", err)
	}
}

// TestHotSwappableClient_HistoryIsolation 测试历史记录切片隔离
func TestHotSwappableClient_HistoryIsolation(t *testing.T) {
	hsc := &HotSwappableClient{
		current:     &mockClient{name: "test"},
		currentName: "test",
		history: []HotSwapEvent{
			{FromProvider: "a", ToProvider: "b", Timestamp: time.Now()},
		},
	}

	history := hsc.SwitchHistory()
	// 修改返回的切片不应影响内部状态
	history[0].FromProvider = "modified"

	internal := hsc.SwitchHistory()
	if internal[0].FromProvider == "modified" {
		t.Error("返回的历史记录切片应与内部隔离")
	}
}
