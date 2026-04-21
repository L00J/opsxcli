package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== InteractiveSession 构造函数测试 =====

func TestNewInteractiveSession(t *testing.T) {
	s := NewInteractiveSession(true)
	assert.NotNil(t, s)
	assert.Equal(t, SessionStateIdle, s.GetState())
	assert.True(t, s.allowInput)
	assert.Empty(t, s.GetMessages())
	assert.False(t, s.IsRunning())
	assert.False(t, s.IsWaiting())
	assert.False(t, s.IsDone())
}

func TestNewInteractiveSession_NoInput(t *testing.T) {
	s := NewInteractiveSession(false)
	assert.False(t, s.allowInput)
}

// ===== Start/Stop 测试 =====

func TestInteractiveSession_StartStop(t *testing.T) {
	s := NewInteractiveSession(false)
	err := s.Start(context.Background())
	require.NoError(t, err)
	assert.Equal(t, SessionStateRunning, s.GetState())
	assert.True(t, s.IsRunning())

	s.Stop()
	assert.Equal(t, SessionStateDone, s.GetState())
	assert.True(t, s.IsDone())
	assert.False(t, s.IsRunning())
}

func TestInteractiveSession_StartTwice(t *testing.T) {
	s := NewInteractiveSession(false)
	err := s.Start(context.Background())
	require.NoError(t, err)

	// 二次启动应失败
	err = s.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已启动")
}

func TestInteractiveSession_StopWithoutStart(t *testing.T) {
	s := NewInteractiveSession(false)
	// 不应该 panic
	s.Stop()
	assert.True(t, s.IsDone())
}

// ===== AddMessage/GetMessages 测试 =====

func TestInteractiveSession_AddGetMessages(t *testing.T) {
	s := NewInteractiveSession(false)

	s.AddMessage("user", "hello")
	s.AddMessage("assistant", "world")
	s.AddMessage("system", "init")

	msgs := s.GetMessages()
	assert.Len(t, msgs, 3)
	assert.Equal(t, "user", msgs[0].Role)
	assert.Equal(t, "hello", msgs[0].Content)
	assert.Equal(t, "assistant", msgs[1].Role)
	assert.Equal(t, "world", msgs[1].Content)
	assert.Equal(t, "system", msgs[2].Role)
}

func TestInteractiveSession_GetMessages_ReturnsCopy(t *testing.T) {
	s := NewInteractiveSession(false)
	s.AddMessage("user", "hello")

	msgs := s.GetMessages()
	msgs[0].Content = "modified"

	// 原始数据不变
	original := s.GetMessages()
	assert.Equal(t, "hello", original[0].Content)
}

// ===== SendOutput/SendError 测试 =====

func TestInteractiveSession_SendOutput(t *testing.T) {
	s := NewInteractiveSession(false)
	s.SendOutput("test output")

	msgs := s.GetMessages()
	assert.Len(t, msgs, 1)
	assert.Equal(t, "assistant", msgs[0].Role)
	assert.Equal(t, "test output", msgs[0].Content)

	// 也应该发送到 outputChan
	select {
	case out := <-s.outputChan:
		assert.Equal(t, "test output", out)
	default:
		t.Fatal("expected output in channel")
	}
}

func TestInteractiveSession_SendError(t *testing.T) {
	s := NewInteractiveSession(false)
	testErr := errors.New("test error")
	s.SendError(testErr)

	select {
	case err := <-s.errorChan:
		assert.Equal(t, testErr, err)
	default:
		t.Fatal("expected error in channel")
	}
}

func TestInteractiveSession_SendOutput_ChannelFull(t *testing.T) {
	s := NewInteractiveSession(false)
	// outputChan 缓冲区为 10
	for i := 0; i < 15; i++ {
		s.SendOutput("msg") // 不应该 panic 或阻塞
	}
}

func TestInteractiveSession_SendError_ChannelFull(t *testing.T) {
	s := NewInteractiveSession(false)
	// errorChan 缓冲区为 10
	for i := 0; i < 15; i++ {
		s.SendError(errors.New("err")) // 不应该 panic 或阻塞
	}
}

// ===== WaitForInput 测试 =====

func TestInteractiveSession_WaitForInput_WithChannelInput(t *testing.T) {
	s := NewInteractiveSession(false)
	err := s.Start(context.Background())
	require.NoError(t, err)

	// 预填输入到通道
	s.inputChan <- "pre-filled"

	input, err := s.WaitForInput("> ")
	require.NoError(t, err)
	assert.Equal(t, "pre-filled", input)

	s.Stop()
}

// ===== Context 取消测试 =====

func TestInteractiveSession_CancelContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewInteractiveSession(false)

	err := s.Start(ctx)
	require.NoError(t, err)
	assert.True(t, s.IsRunning())

	// 取消 context
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Stop 应该仍然能正常工作
	s.Stop()
	assert.True(t, s.IsDone())
}

// ===== SessionState 类型测试 =====

func TestSessionState_Values(t *testing.T) {
	assert.Equal(t, SessionState("idle"), SessionStateIdle)
	assert.Equal(t, SessionState("running"), SessionStateRunning)
	assert.Equal(t, SessionState("waiting"), SessionStateWaiting)
	assert.Equal(t, SessionState("done"), SessionStateDone)
}

// ===== SessionMessage 结构测试 =====

func TestSessionMessage_Struct(t *testing.T) {
	msg := SessionMessage{
		Role:    "user",
		Content: "test",
	}
	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "test", msg.Content)
}
