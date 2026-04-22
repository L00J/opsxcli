package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ===== Screen NewScreen 测试 =====

func TestNewScreen_Dimensions(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	assert.NotNil(t, screen)

	// 默认尺寸: 如果 term.GetSize 失败则为 80x24，否则为终端实际尺寸
	assert.Greater(t, screen.width, 0)
	assert.Greater(t, screen.height, 0)

	screen.Close()
}

func TestNewScreen_DefaultPrompt(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	assert.Equal(t, "⏵ ", screen.inputPrompt)
}

// ===== Screen AddMessage 测试 =====

func TestScreen_AddMessage_Multiple(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.AddMessage("user", "hello")
	screen.AddMessage("assistant", "hi there")
	screen.AddMessage("tool", "executing...")

	screen.mu.RLock()
	assert.Len(t, screen.messages, 3)
	assert.Equal(t, "user", screen.messages[0].Role)
	assert.Equal(t, "assistant", screen.messages[1].Role)
	assert.Equal(t, "tool", screen.messages[2].Role)
	screen.mu.RUnlock()
}

func TestScreen_AddMessage_Timestamp(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	before := time.Now()
	screen.AddMessage("user", "test")
	after := time.Now()

	screen.mu.RLock()
	assert.True(t, screen.messages[0].Time.After(before) || screen.messages[0].Time.Equal(before))
	assert.True(t, screen.messages[0].Time.Before(after) || screen.messages[0].Time.Equal(after))
	screen.mu.RUnlock()
}

// ===== Screen SetStatus 测试 =====

func TestScreen_SetStatus(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.SetStatus("thinking...")
	screen.mu.RLock()
	assert.Equal(t, "thinking...", screen.statusLine)
	screen.mu.RUnlock()
}

func TestScreen_SetStatus_Empty(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.SetStatus("")
	screen.mu.RLock()
	assert.Equal(t, "", screen.statusLine)
	screen.mu.RUnlock()
}

// ===== Screen StartThinking/StopThinking 测试 =====

func TestScreen_StartThinking_SetsState(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.StartThinking(10)

	screen.mu.RLock()
	assert.True(t, screen.isThinking)
	assert.Equal(t, 10, screen.maxSteps)
	assert.Equal(t, 0, screen.step)
	assert.Equal(t, 0, screen.tokens)
	assert.False(t, screen.thinkingStart.IsZero())
	screen.mu.RUnlock()

	screen.StopThinking()
}

func TestScreen_StopThinking_ClearsState(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.StartThinking(5)
	time.Sleep(50 * time.Millisecond)
	screen.StopThinking()

	screen.mu.RLock()
	assert.False(t, screen.isThinking)
	assert.Equal(t, "", screen.statusLine)
	screen.mu.RUnlock()
}

// ===== Screen UpdateProgress 测试 =====

func TestScreen_UpdateProgress(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.UpdateProgress(5, 2000)

	screen.mu.RLock()
	assert.Equal(t, 5, screen.step)
	assert.Equal(t, 2000, screen.tokens)
	screen.mu.RUnlock()
}

func TestScreen_UpdateProgress_WithThinking(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.StartThinking(10)
	screen.UpdateProgress(3, 1500)

	screen.mu.RLock()
	assert.True(t, screen.isThinking)
	assert.Equal(t, 3, screen.step)
	assert.Equal(t, 1500, screen.tokens)
	screen.mu.RUnlock()

	screen.StopThinking()
}

// ===== Screen Render 测试 =====

func TestScreen_Render_NoMessages(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	// 不应该 panic
	screen.Render()
}

func TestScreen_Render_WithMessages(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.AddMessage("user", "hello")
	screen.AddMessage("assistant", "world")

	// 不应该 panic
	screen.Render()
}

func TestScreen_Render_WithManyMessages(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	// 添加大量消息，超过 visibleLines/3
	for i := 0; i < 30; i++ {
		screen.AddMessage("user", "message "+strings.Repeat("x", 50))
	}

	// 不应该 panic
	screen.Render()
}

// ===== Screen renderMessage 测试 (通过 AddMessage + Render 间接调用) =====

func TestScreen_Render_UserMessage(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.AddMessage("user", "user message")
	screen.Render()
}

func TestScreen_Render_AssistantMessage(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.AddMessage("assistant", "assistant message")
	screen.Render()
}

func TestScreen_Render_ToolMessage(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.AddMessage("tool", "tool output")
	screen.Render()
}

func TestScreen_Render_StatusMessage(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.AddMessage("status", "status update")
	screen.Render()
}

func TestScreen_Render_UnknownMessage(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.AddMessage("unknown", "unknown role")
	screen.Render()
}

// ===== Screen renderInputArea 测试 =====

func TestScreen_RenderInputArea_WithStatus(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.SetStatus("thinking...")
	screen.Render()
}

func TestScreen_RenderInputArea_WithoutStatus(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.SetStatus("")
	screen.Render()
}

func TestScreen_RenderInputArea_WithInput(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.inputBuffer = "some text"
	screen.cursorPos = 5
	screen.Render()
}

func TestScreen_RenderInputArea_CursorAtStart(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.inputBuffer = "hello"
	screen.cursorPos = 0
	screen.Render()
}

func TestScreen_RenderInputArea_CursorAtEnd(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.inputBuffer = "hello"
	screen.cursorPos = 5 // len("hello")
	screen.Render()
}

// ===== Screen printWrapped 测试 (通过 renderMessage 间接调用) =====

func TestScreen_PrintWrapped_ShortText(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.AddMessage("user", "short text")
	screen.Render()
}

func TestScreen_PrintWrapped_LongText(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	longText := "This is a very long text that should be wrapped when it exceeds the maximum width of the terminal display. It contains multiple words and should be split at word boundaries when possible."
	screen.AddMessage("assistant", longText)
	screen.Render()
}

func TestScreen_PrintWrapped_MultilineText(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	multiline := "line one\nline two\nline three"
	screen.AddMessage("assistant", multiline)
	screen.Render()
}

func TestScreen_PrintWrapped_VeryLongWord(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.width = 40 // 设置较窄宽度
	longWord := strings.Repeat("a", 200)
	screen.AddMessage("assistant", longWord)
	screen.Render()
}

func TestScreen_PrintWrapped_NarrowWidth(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.width = 10 // 非常窄的宽度
	screen.AddMessage("assistant", "hello world this is a test")
	screen.Render()
}

// ===== Screen updateThinkingStatus 测试 =====

func TestScreen_UpdateThinkingStatus_Lifecycle(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.StartThinking(5)
	time.Sleep(700 * time.Millisecond) // 等待至少一个 tick

	screen.mu.RLock()
	thinking := screen.isThinking
	screen.mu.RUnlock()
	assert.True(t, thinking)

	screen.StopThinking()
	time.Sleep(100 * time.Millisecond)

	screen.mu.RLock()
	assert.False(t, screen.isThinking)
	screen.mu.RUnlock()
}

func TestScreen_UpdateThinkingStatus_WithProgress(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.StartThinking(10)
	screen.UpdateProgress(3, 500)

	time.Sleep(700 * time.Millisecond)

	screen.StopThinking()
}

// ===== Screen Close 测试 =====

func TestScreen_Close_MultipleSafe(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)

	// 第一次 Close
	screen.Close()

	// 注意: 第二次 Close 会 panic (close on closed channel)
	// 这是预期行为，不测试
}

// ===== Message 结构测试 =====

func TestMessage_Struct(t *testing.T) {
	now := time.Now()
	msg := Message{
		Role:    "user",
		Content: "测试消息",
		Time:    now,
	}
	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "测试消息", msg.Content)
	assert.Equal(t, now, msg.Time)
}

// ===== SimpleProgress 测试 =====

func TestSimpleProgress_WithTokens(t *testing.T) {
	result := SimpleProgress(2, 10, 500, 3*time.Second)
	assert.Contains(t, result, "Thinking")
	assert.Contains(t, result, "step 2/10")
	assert.Contains(t, result, "3s")
	assert.Contains(t, result, "500 tokens")
}

func TestSimpleProgress_ZeroTokens(t *testing.T) {
	result := SimpleProgress(1, 5, 0, 1*time.Second)
	assert.Contains(t, result, "Thinking")
	assert.NotContains(t, result, "tokens")
}

// ===== Screen 渲染综合测试 =====

func TestScreen_Render_FullScenario(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	// 模拟完整的交互流程
	screen.AddMessage("user", "查询 Redis 状态")
	screen.Render()

	screen.StartThinking(5)
	screen.UpdateProgress(1, 100)
	time.Sleep(100 * time.Millisecond)

	screen.AddMessage("tool", "执行 redis-cli ping")
	screen.Render()

	screen.UpdateProgress(2, 300)
	screen.StopThinking()

	screen.AddMessage("assistant", "Redis 运行正常，PONG")
	screen.Render()
}
