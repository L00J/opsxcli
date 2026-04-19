package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"opsxcli/internal/agentv2/session"
)

func TestNewModel(t *testing.T) {
	m := NewModel(nil, context.Background())
	if m.state != stateIdle {
		t.Error("初始状态应为 idle")
	}
	if len(m.messages) != 0 {
		t.Error("初始消息列表应为空")
	}
	if m.msgChan == nil {
		t.Error("消息通道应已初始化")
	}
}

func TestModelUpdateWindowSize(t *testing.T) {
	m := NewModel(nil, context.Background())
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model := newM.(Model)
	if model.width != 100 {
		t.Errorf("width 应为 100, 实际为 %d", model.width)
	}
	if model.height != 30 {
		t.Errorf("height 应为 30, 实际为 %d", model.height)
	}
	if model.viewport.Width != 96 {
		t.Errorf("viewport width 应为 96, 实际为 %d", model.viewport.Width)
	}
}

func TestRenderMessages(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.messages = []ChatMessage{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "world"},
	}
	s := m.renderMessages()
	if !strings.Contains(s, "hello") {
		t.Error("消息渲染应包含 'hello'")
	}
	if !strings.Contains(s, "world") {
		t.Error("消息渲染应包含 'world'")
	}
}

func TestModelEnterSendsMessage(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.textarea.SetValue("test query")

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newM.(Model)

	// 发送后输入框应被重置
	if model.textarea.Value() != "" {
		t.Error("发送后输入框应被清空")
	}

	// 用户消息应被添加
	if len(model.messages) != 1 || model.messages[0].Content != "test query" {
		t.Error("用户消息未正确添加")
	}

	// 状态应为 thinking
	if model.state != stateThinking {
		t.Error("发送后状态应为 thinking")
	}

	// 应返回命令
	if cmd == nil {
		t.Error("应按 Enter 后返回命令")
	}
}

func TestModelExitCommand(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.textarea.SetValue("/exit")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("/exit 应返回退出命令")
	}

	// 执行命令验证是否为 tea.Quit
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Error("/exit 应返回 tea.QuitMsg")
	}
}

func TestModelEscQuits(t *testing.T) {
	m := NewModel(nil, context.Background())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("ESC 应返回退出命令")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Error("ESC 应返回 tea.QuitMsg")
	}
}

func TestStreamChunkMsg(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24

	// 添加一条用户消息后再接收流式输出
	m.messages = append(m.messages, ChatMessage{Role: "user", Content: "hi"})

	newM, cmd := m.Update(streamChunkMsg{content: "hello"})
	model := newM.(Model)

	if model.state != stateStreaming {
		t.Error("收到流式片段后状态应为 streaming")
	}
	if len(model.messages) != 2 {
		t.Fatalf("消息数应为 2, 实际为 %d", len(model.messages))
	}
	if model.messages[1].Role != "assistant" {
		t.Error("流式消息角色应为 assistant")
	}
	if model.messages[1].Content != "hello" {
		t.Errorf("流式消息内容应为 'hello', 实际为 '%s'", model.messages[1].Content)
	}
	if cmd == nil {
		t.Error("流式输出应继续等待下一条消息")
	}
}

func TestThinkDoneMsg(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.state = stateStreaming

	newM, _ := m.Update(thinkDoneMsg{})
	model := newM.(Model)

	if model.state != stateIdle {
		t.Error("thinkDoneMsg 后状态应为 idle")
	}
}

func TestErrorMsg(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24

	newM, _ := m.Update(errorMsg{err: context.Canceled})
	model := newM.(Model)

	if model.state != stateIdle {
		t.Error("errorMsg 后状态应为 idle")
	}
	if len(model.messages) != 1 {
		t.Fatalf("错误消息应被添加到消息列表, 实际长度 %d", len(model.messages))
	}
	if !strings.Contains(model.messages[0].Content, "context canceled") {
		t.Error("错误消息内容应包含错误信息")
	}
}

func TestModelSessionListSwitch(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24

	// 按 Ctrl+O 切换到会话列表
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	model := newM.(Model)

	if model.viewState != viewSessionList {
		t.Error("Ctrl+O 应切换到会话列表视图")
	}
	if cmd == nil {
		t.Error("切换到会话列表应返回加载命令")
	}

	// 再按 Ctrl+O 返回聊天视图
	newM2, cmd2 := model.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	model2 := newM2.(Model)

	if model2.viewState != viewChat {
		t.Error("再次按 Ctrl+O 应返回聊天视图")
	}
	if cmd2 != nil {
		t.Error("返回聊天视图不应返回命令")
	}
}

func TestModelCreateNewSession(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24

	// 输入 /new 并发送
	m.textarea.SetValue("/new")
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newM.(Model)

	if model.textarea.Value() != "" {
		t.Error("/new 命令应清空输入框")
	}
	if cmd == nil {
		t.Error("/new 应返回创建会话命令")
	}
}

func TestRenderSessionList(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24
	m.viewState = viewSessionList
	m.sessions = []*session.Session{
		{ID: "1", Title: "测试会话", MessageCount: 3, UpdatedAt: time.Now()},
	}

	s := m.renderSessionList()
	if !strings.Contains(s, "测试会话") {
		t.Error("会话列表渲染应包含会话标题")
	}
	if !strings.Contains(s, "3条消息") {
		t.Error("会话列表渲染应包含消息数量")
	}
}

func TestModelSessionsCommand(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24

	m.textarea.SetValue("/sessions")
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newM.(Model)

	if model.textarea.Value() != "" {
		t.Error("/sessions 命令应清空输入框")
	}
	if model.viewState != viewSessionList {
		t.Error("/sessions 应切换到会话列表视图")
	}
	if cmd == nil {
		t.Error("/sessions 应返回加载命令")
	}
}

func TestModelSessionListNavigation(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24
	m.viewState = viewSessionList
	m.sessions = []*session.Session{
		{ID: "1", Title: "会话1", MessageCount: 1, UpdatedAt: time.Now()},
		{ID: "2", Title: "会话2", MessageCount: 2, UpdatedAt: time.Now()},
		{ID: "3", Title: "会话3", MessageCount: 3, UpdatedAt: time.Now()},
	}

	// 向下移动
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := newM.(Model)
	if model.listCursor != 1 {
		t.Errorf("按 Down 后光标应为 1, 实际为 %d", model.listCursor)
	}

	// 再向下移动
	newM2, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model2 := newM2.(Model)
	if model2.listCursor != 2 {
		t.Errorf("按 Down 后光标应为 2, 实际为 %d", model2.listCursor)
	}

	// 到底后不能再向下
	newM3, _ := model2.Update(tea.KeyMsg{Type: tea.KeyDown})
	model3 := newM3.(Model)
	if model3.listCursor != 2 {
		t.Errorf("到底后光标应保持 2, 实际为 %d", model3.listCursor)
	}

	// 向上移动
	newM4, _ := model3.Update(tea.KeyMsg{Type: tea.KeyUp})
	model4 := newM4.(Model)
	if model4.listCursor != 1 {
		t.Errorf("按 Up 后光标应为 1, 实际为 %d", model4.listCursor)
	}
}

func TestModelSessionListEscReturnsToChat(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24
	m.viewState = viewSessionList

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newM.(Model)

	if model.viewState != viewChat {
		t.Error("Esc 应从会话列表返回聊天视图")
	}
	if cmd != nil {
		t.Error("Esc 返回聊天视图不应有命令")
	}
}
