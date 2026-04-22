package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ===== ToolStatus StartWithMessage 测试 =====

func TestToolStatus_StartWithMessage(t *testing.T) {
	ts := NewToolStatus()

	ts.StartWithMessage("curl", "Fetching API data...")

	ts.mu.RLock()
	assert.True(t, ts.isRunning)
	assert.Equal(t, "curl", ts.currentTool)
	assert.Equal(t, "Fetching API data...", ts.customMessage)
	assert.False(t, ts.toolStart.IsZero())
	ts.mu.RUnlock()

	// 等待动画至少一个 tick
	time.Sleep(100 * time.Millisecond)

	ts.Stop(true, "完成")
	assert.False(t, ts.IsRunning())
}

func TestToolStatus_StartWithMessage_EmptyMessage(t *testing.T) {
	ts := NewToolStatus()

	ts.StartWithMessage("bash", "")
	assert.True(t, ts.IsRunning())

	time.Sleep(100 * time.Millisecond)

	ts.Stop(true, "")
	assert.False(t, ts.IsRunning())
}

func TestToolStatus_StartWithMessage_LongMessage(t *testing.T) {
	ts := NewToolStatus()

	longMsg := "This is a very long custom message that describes what the tool is doing in great detail"
	ts.StartWithMessage("docker", longMsg)

	ts.mu.RLock()
	assert.Equal(t, longMsg, ts.customMessage)
	ts.mu.RUnlock()

	ts.Stop(true, "done")
}

// ===== renderRunningStatus 测试 =====
// renderRunningStatus 直接使用 fmt.Printf，我们验证调用不 panic

func TestToolStatus_RenderRunningStatus_Standard(t *testing.T) {
	ts := NewToolStatus()
	// 标准模式 (无自定义消息)
	ts.renderRunningStatus("curl", 500*time.Millisecond, "⠋", "")
}

func TestToolStatus_RenderRunningStatus_WithCustomMessage(t *testing.T) {
	ts := NewToolStatus()
	// 自定义消息模式
	ts.renderRunningStatus("bash", 2*time.Second, "⠙", "Running script...")
}

func TestToolStatus_RenderRunningStatus_LongElapsed(t *testing.T) {
	ts := NewToolStatus()
	// 超过 1 秒的耗时，应显示时长
	ts.renderRunningStatus("redis", 5*time.Second, "⠹", "")
}

func TestToolStatus_RenderRunningStatus_ShortElapsed(t *testing.T) {
	ts := NewToolStatus()
	// 低于 1 秒的耗时，不应显示时长
	ts.renderRunningStatus("redis", 500*time.Millisecond, "⠸", "testing")
}

// ===== animate 测试 (通过 Start 间接测试) =====

func TestToolStatus_Animate_WithStop(t *testing.T) {
	ts := NewToolStatus()
	ts.Start("kubectl")

	time.Sleep(200 * time.Millisecond)

	ts.Stop(true, "completed")
	assert.False(t, ts.IsRunning())
}

func TestToolStatus_Animate_MultipleStartStop(t *testing.T) {
	ts := NewToolStatus()

	for i := 0; i < 3; i++ {
		ts.Start("curl")
		assert.True(t, ts.IsRunning())

		time.Sleep(50 * time.Millisecond)

		ts.Stop(true, "done")
		assert.False(t, ts.IsRunning())
	}
}

// ===== SimpleToolStatus 测试 =====

func TestSimpleToolStatus_WithMessage(t *testing.T) {
	// 不应该 panic
	SimpleToolStatus("curl", "Fetching data...")
}

func TestSimpleToolStatus_WithoutMessage(t *testing.T) {
	SimpleToolStatus("redis", "")
}

func TestSimpleToolStatus_UnknownTool(t *testing.T) {
	SimpleToolStatus("unknown_tool", "doing something")
}

func TestSimpleToolStatus_AllCategories(t *testing.T) {
	// 测试各类工具图标
	tools := []string{"curl", "ssh", "bash", "cat", "kubectl", "docker", "redis", "mysql", "git_status", "sys_monitor", "web_search"}
	for _, tool := range tools {
		SimpleToolStatus(tool, "testing "+tool)
	}
}

// ===== ToolStatusSuccess 测试 =====

func TestToolStatusSuccess_WithMessage(t *testing.T) {
	ToolStatusSuccess("curl", 3*time.Second, "200 OK")
}

func TestToolStatusSuccess_WithoutMessage(t *testing.T) {
	ToolStatusSuccess("bash", 100*time.Millisecond, "")
}

func TestToolStatusSuccess_LongDuration(t *testing.T) {
	ToolStatusSuccess("redis", 5*time.Minute, "Cache warmed")
}

func TestToolStatusSuccess_ShortDuration(t *testing.T) {
	ToolStatusSuccess("docker", 10*time.Millisecond, "Container started")
}

// ===== ToolStatusError 测试 =====

func TestToolStatusError(t *testing.T) {
	ToolStatusError("curl", 5*time.Second, "Connection refused")
}

func TestToolStatusError_EmptyMessage(t *testing.T) {
	ToolStatusError("bash", time.Second, "")
}

func TestToolStatusError_UnknownTool(t *testing.T) {
	ToolStatusError("my_custom_tool", 2*time.Second, "custom error")
}

// ===== ToolStatus Start + Stop 快速测试 =====

func TestToolStatus_QuickStartStop(t *testing.T) {
	ts := NewToolStatus()
	ts.Start("redis")
	ts.Stop(false, "failed immediately")
	assert.False(t, ts.IsRunning())
}

func TestToolStatus_StopNotRunning_MultipleTimes(t *testing.T) {
	ts := NewToolStatus()
	ts.Stop(true, "not running 1")
	ts.Stop(false, "not running 2")
	ts.Stop(true, "not running 3")
	assert.False(t, ts.IsRunning())
}

// ===== ToolStatus 并发测试 =====

func TestToolStatus_ConcurrentAccess(t *testing.T) {
	ts := NewToolStatus()
	done := make(chan struct{})

	go func() {
		for i := 0; i < 50; i++ {
			ts.Start("curl")
			time.Sleep(time.Millisecond)
			ts.Stop(true, "done")
		}
		close(done)
	}()

	// 同时读取状态
	go func() {
		for i := 0; i < 50; i++ {
			_ = ts.IsRunning()
			_ = ts.GetCurrentTool()
			time.Sleep(time.Millisecond)
		}
	}()

	<-done
}
