package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ===== InteractiveUI 构造与状态测试 =====
// 注意: NewInteractiveUI 需要创建 readline 实例，可能依赖 /tmp 写权限
// Close, ReadInput, Prompt 需要 readline 实例，通过间接方式测试

func TestInteractiveUI_StartThinking(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}

	ui.StartThinking(5)

	ui.statusMutex.RLock()
	assert.True(t, ui.isThinking)
	assert.Equal(t, 5, ui.maxSteps)
	assert.Equal(t, 0, ui.step)
	assert.Equal(t, 0, ui.tokens)
	assert.False(t, ui.thinkingStart.IsZero())
	ui.statusMutex.RUnlock()

	// 清理: 停止思考
	ui.StopThinking()
}

func TestInteractiveUI_StopThinking(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}

	ui.StartThinking(3)
	time.Sleep(50 * time.Millisecond)

	ui.StopThinking()

	ui.statusMutex.RLock()
	assert.False(t, ui.isThinking)
	ui.statusMutex.RUnlock()
}

func TestInteractiveUI_StopThinking_WithoutStart(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}
	// 不应该 panic
	ui.StopThinking()

	ui.statusMutex.RLock()
	assert.False(t, ui.isThinking)
	ui.statusMutex.RUnlock()
}

func TestInteractiveUI_UpdateProgress(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}

	ui.UpdateProgress(3, 1500)

	ui.statusMutex.RLock()
	assert.Equal(t, 3, ui.step)
	assert.Equal(t, 1500, ui.tokens)
	ui.statusMutex.RUnlock()
}

func TestInteractiveUI_UpdateProgress_Multiple(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}

	ui.UpdateProgress(1, 100)
	ui.UpdateProgress(2, 500)
	ui.UpdateProgress(3, 2000)

	ui.statusMutex.RLock()
	assert.Equal(t, 3, ui.step)
	assert.Equal(t, 2000, ui.tokens)
	ui.statusMutex.RUnlock()
}

func TestInteractiveUI_StartThinking_WithProgress(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}

	ui.StartThinking(10)
	time.Sleep(50 * time.Millisecond)

	ui.UpdateProgress(5, 3000)

	ui.statusMutex.RLock()
	assert.True(t, ui.isThinking)
	assert.Equal(t, 5, ui.step)
	assert.Equal(t, 3000, ui.tokens)
	assert.Equal(t, 10, ui.maxSteps)
	ui.statusMutex.RUnlock()

	ui.StopThinking()
}

// ===== updateStatus 间接测试 =====
// updateStatus 是一个私有方法，通过 StartThinking 启动 goroutine
// 我们通过启动然后快速停止来测试它的生命周期

func TestInteractiveUI_UpdateStatus_StopsCleanly(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}

	// StartThinking 会启动 updateStatus goroutine
	ui.StartThinking(5)

	// 等待 goroutine 至少运行一个 tick
	time.Sleep(600 * time.Millisecond)

	// 更新进度来验证 goroutine 正在读取数据
	ui.UpdateProgress(2, 1000)

	// 停止，goroutine 应该退出
	ui.StopThinking()
	time.Sleep(100 * time.Millisecond)

	ui.statusMutex.RLock()
	assert.False(t, ui.isThinking)
	ui.statusMutex.RUnlock()
}

// ===== ShowMessage 测试 =====
// ShowMessage 直接使用 fmt.Printf，我们验证它不会 panic

func TestInteractiveUI_ShowMessage_UserRole(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}
	// 不应该 panic
	ui.ShowMessage("user", "hello world")
}

func TestInteractiveUI_ShowMessage_AssistantRole(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}
	ui.ShowMessage("assistant", "response text")
}

func TestInteractiveUI_ShowMessage_ToolRole(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}
	ui.ShowMessage("tool", "tool output")
}

func TestInteractiveUI_ShowMessage_ErrorRole(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}
	ui.ShowMessage("error", "something went wrong")
}

func TestInteractiveUI_ShowMessage_InfoRole(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}
	ui.ShowMessage("info", "informational message")
}

func TestInteractiveUI_ShowMessage_UnknownRole(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}
	ui.ShowMessage("unknown", "unknown role message")
}

func TestInteractiveUI_ShowMessage_EmptyRole(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}
	ui.ShowMessage("", "no role message")
}

// ===== InteractiveUI 结构体字段测试 =====

func TestInteractiveUI_Fields(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}
	assert.NotNil(t, ui.userInputChan)
	assert.False(t, ui.isThinking)
	assert.Equal(t, 0, ui.step)
	assert.Equal(t, 0, ui.tokens)
	assert.Equal(t, 0, ui.maxSteps)
}

func TestInteractiveUI_ConcurrentAccess(t *testing.T) {
	ui := &InteractiveUI{
		userInputChan: make(chan string, 10),
	}

	// 并发读写测试
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			ui.UpdateProgress(i, i*10)
		}
		close(done)
	}()

	ui.StartThinking(100)
	time.Sleep(50 * time.Millisecond)
	ui.StopThinking()

	<-done
}
