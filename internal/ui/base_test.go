package ui

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

// ===== mockBaseUI 实现 BaseUI 接口 =====

type mockBaseUI struct {
	drawCount       int
	eventHandled    bool
	returnFromEvent bool
	mu              sync.Mutex
}

func (m *mockBaseUI) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (m *mockBaseUI) Draw(screen tcell.Screen) error {
	m.mu.Lock()
	m.drawCount++
	m.mu.Unlock()
	return nil
}

func (m *mockBaseUI) HandleEvent(event *tcell.EventKey) bool {
	m.mu.Lock()
	m.eventHandled = true
	result := m.returnFromEvent
	m.mu.Unlock()
	return result
}

// ===== UIManager 构造测试 =====

func TestNewUIManager(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要 tcell 的测试")
	}

	mockUI := &mockBaseUI{returnFromEvent: true}
	mgr, err := NewUIManager(mockUI)

	// tcell.NewScreen() 在没有真实终端的环境下可能失败
	if err != nil {
		t.Skipf("无法创建 tcell screen (无终端环境): %v", err)
	}

	assert.NotNil(t, mgr)
	assert.NotNil(t, mgr.screen)
	assert.NotNil(t, mgr.ctx)
	assert.NotNil(t, mgr.cancel)
	assert.NotNil(t, mgr.refreshChan)
	assert.Equal(t, mockUI, mgr.ui)

	mgr.Finish()
}

// ===== UIManager 结构测试 =====

func TestUIManager_Struct(t *testing.T) {
	mgr := &UIManager{
		refreshChan: make(chan struct{}, 1),
	}
	assert.NotNil(t, mgr.refreshChan)
	assert.Nil(t, mgr.screen)
	assert.Nil(t, mgr.ui)
}

// ===== TriggerRefresh 测试 =====

func TestUIManager_TriggerRefresh(t *testing.T) {
	mgr := &UIManager{
		refreshChan: make(chan struct{}, 1),
	}

	// 触发刷新
	mgr.TriggerRefresh()

	// 应该能从 refreshChan 读到数据
	select {
	case <-mgr.refreshChan:
		// 成功
	default:
		t.Fatal("expected refresh signal in channel")
	}
}

func TestUIManager_TriggerRefresh_Multiple(t *testing.T) {
	mgr := &UIManager{
		refreshChan: make(chan struct{}, 1),
	}

	// 多次触发，不应阻塞
	mgr.TriggerRefresh()
	mgr.TriggerRefresh()
	mgr.TriggerRefresh()

	// 通道缓冲区为1，只有第一个信号在
	assert.Len(t, mgr.refreshChan, 1)
}

func TestUIManager_TriggerRefresh_DoesNotBlock(t *testing.T) {
	mgr := &UIManager{
		refreshChan: make(chan struct{}, 1),
	}

	// 先填满通道
	mgr.refreshChan <- struct{}{}

	// 再次触发，不应阻塞
	done := make(chan struct{})
	go func() {
		mgr.TriggerRefresh()
		close(done)
	}()

	select {
	case <-done:
		// 成功，没有阻塞
	case <-time.After(100 * time.Millisecond):
		t.Fatal("TriggerRefresh blocked")
	}
}

// ===== Stop 测试 =====

func TestUIManager_Stop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mgr := &UIManager{
		ctx:    ctx,
		cancel: cancel,
	}

	// 验证 context 未取消
	assert.NoError(t, mgr.ctx.Err())

	mgr.Stop()

	// 验证 context 已取消
	assert.Error(t, mgr.ctx.Err())
}

// ===== refresh 测试 =====

func TestUIManager_Refresh(t *testing.T) {
	mockUI := &mockBaseUI{returnFromEvent: true}
	screen := newMockScreen()

	mgr := &UIManager{
		screen: screen,
		ui:     mockUI,
	}

	mgr.refresh()

	mockUI.mu.Lock()
	assert.Equal(t, 1, mockUI.drawCount)
	mockUI.mu.Unlock()
}

func TestUIManager_Refresh_Multiple(t *testing.T) {
	mockUI := &mockBaseUI{returnFromEvent: true}
	screen := newMockScreen()

	mgr := &UIManager{
		screen: screen,
		ui:     mockUI,
	}

	mgr.refresh()
	mgr.refresh()
	mgr.refresh()

	mockUI.mu.Lock()
	assert.Equal(t, 3, mockUI.drawCount)
	mockUI.mu.Unlock()
}

// ===== Finish 测试 =====

func TestUIManager_Finish_WithScreen(t *testing.T) {
	screen := newMockScreen()
	mgr := &UIManager{
		screen: screen,
	}

	// Finish 调用 screen.Fini()，mockScreen 的 Fini 是空操作
	mgr.Finish()
}

// ===== BaseUI 接口测试 =====

func TestMockBaseUI_ImplementsInterface(t *testing.T) {
	// 编译时验证接口实现
	var _ BaseUI = &mockBaseUI{}
}

// ===== UIManager Run 测试 =====

func TestUIManager_Run_Cancelled(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要 tcell 的测试")
	}

	mockUI := &mockBaseUI{returnFromEvent: true}
	mgr, err := NewUIManager(mockUI)
	if err != nil {
		t.Skipf("无法创建 tcell screen: %v", err)
	}

	// 立即停止
	go func() {
		mgr.Stop()
	}()

	// Run 应该返回 nil
	err = mgr.Run()
	// Run 会调用 Finish，但可能已调用 Stop
	// 不验证返回值，只验证不会 panic
	_ = err
}
