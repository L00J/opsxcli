package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"opsxcli/internal/agent/session"
	"opsxcli/internal/llm"
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
	if m.modelName != "deepseek-chat" {
		t.Errorf("默认模型名应为 deepseek-chat, 实际为 %s", m.modelName)
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
	// token 应被估算累加
	if model.totalTokens <= 0 {
		t.Error("收到流式片段后 totalTokens 应大于 0")
	}
}

func TestThinkDoneMsg(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.state = stateStreaming

	newM, _ := m.Update(thinkDoneMsg{})
	model := newM.(Model)

	if model.state != stateCompleted {
		t.Error("thinkDoneMsg 后状态应为 completed")
	}
}

func TestErrorMsg(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24

	newM, _ := m.Update(errorMsg{err: context.Canceled})
	model := newM.(Model)

	if model.state != stateError {
		t.Error("errorMsg 后状态应为 error")
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

// === Markdown 渲染测试 ===

func TestRenderMarkdownHeaders(t *testing.T) {
	s := RenderMarkdown("# H1\n## H2\n### H3", 80)
	if !strings.Contains(s, "H1") {
		t.Error("H1 渲染应包含文本")
	}
	if !strings.Contains(s, "H2") {
		t.Error("H2 渲染应包含文本")
	}
	if !strings.Contains(s, "H3") {
		t.Error("H3 渲染应包含文本")
	}
}

func TestRenderMarkdownBold(t *testing.T) {
	s := RenderMarkdown("这是 **粗体** 文本", 80)
	if !strings.Contains(s, "粗体") {
		t.Error("粗体渲染应包含文本")
	}
}

func TestRenderMarkdownInlineCode(t *testing.T) {
	s := RenderMarkdown("使用 `kubectl get pods` 命令", 80)
	if !strings.Contains(s, "kubectl get pods") {
		t.Error("行内代码渲染应包含文本")
	}
}

func TestRenderMarkdownCodeBlock(t *testing.T) {
	input := "```bash\n$ uptime\n```"
	s := RenderMarkdown(input, 80)
	if !strings.Contains(s, "uptime") {
		t.Error("代码块渲染应包含代码内容")
	}
	if !strings.Contains(s, "bash") {
		t.Error("代码块渲染应包含语言标签")
	}
}

func TestRenderMarkdownList(t *testing.T) {
	input := "- item1\n- item2\n1. item3"
	s := RenderMarkdown(input, 80)
	if !strings.Contains(s, "item1") {
		t.Error("列表渲染应包含 item1")
	}
	if !strings.Contains(s, "item2") {
		t.Error("列表渲染应包含 item2")
	}
	if !strings.Contains(s, "item3") {
		t.Error("列表渲染应包含 item3")
	}
}

func TestRenderMarkdownQuote(t *testing.T) {
	input := "> 这是一条引用\n> 第二行"
	s := RenderMarkdown(input, 80)
	if !strings.Contains(s, "引用") {
		t.Error("引用渲染应包含文本")
	}
}

func TestRenderMarkdownTable(t *testing.T) {
	input := "| A | B |\n|---|---|\n| 1 | 2 |"
	s := RenderMarkdown(input, 80)
	if !strings.Contains(s, "A") {
		t.Error("表格渲染应包含 A")
	}
	if !strings.Contains(s, "1") {
		t.Error("表格渲染应包含 1")
	}
}

func TestRenderMarkdownDivider(t *testing.T) {
	s := RenderMarkdown("---", 80)
	if s == "" {
		t.Error("分隔线渲染不应为空")
	}
}

func TestRenderMessagesWithMarkdown(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.messages = []ChatMessage{
		{Role: "user", Content: "查看负载"},
		{Role: "assistant", Content: "## 系统负载\n\n```bash\n$ uptime\n```"},
	}
	s := m.renderMessages()
	if !strings.Contains(s, "查看负载") {
		t.Error("用户消息应被渲染")
	}
	if !strings.Contains(s, "系统负载") {
		t.Error("助手消息标题应被渲染")
	}
	if !strings.Contains(s, "uptime") {
		t.Error("助手消息代码块应被渲染")
	}
}

func TestHasAgentPrefix(t *testing.T) {
	if !hasAgentPrefix("\x1b[36m🤖 Agent\x1b[0m hello") {
		t.Error("应检测到 stream.go 输出的前缀")
	}
	if hasAgentPrefix("hello world") {
		t.Error("普通内容不应被误判为包含前缀")
	}
}

func TestEstimateTokens(t *testing.T) {
	// ASCII 文本
	asciiTokens := estimateTokens("hello world")
	if asciiTokens <= 0 {
		t.Error("ASCII 文本估算 token 应大于 0")
	}
	// 中文文本
	zhTokens := estimateTokens("你好世界")
	if zhTokens <= 0 {
		t.Error("中文文本估算 token 应大于 0")
	}
	// 空文本
	emptyTokens := estimateTokens("   \n\t  ")
	if emptyTokens != 0 {
		t.Error("空白文本估算 token 应为 0")
	}
}

// === Footer 渲染测试 ===

func TestRenderFooter(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.currentModel = "deepseek-chat"
	m.maxTokens = 6000
	m.totalTokens = 3000
	m.currentHost = "local"

	footer := m.renderFooter()
	if footer == "" {
		t.Error("renderFooter 不应返回空字符串")
	}
	if !strings.Contains(footer, "deepseek-chat") {
		t.Error("footer 应包含模型名称")
	}
	if !strings.Contains(footer, "3000") {
		t.Error("footer 应包含 token 使用量")
	}
	if !strings.Contains(footer, "就绪") {
		t.Error("footer 应包含状态")
	}
}

func TestStateString(t *testing.T) {
	m := NewModel(nil, context.Background())

	states := map[sessionState]string{
		stateIdle:       "● 就绪",
		statePlanning:   "◐ 规划任务",
		stateThinking:   "◐ 思考中",
		stateExecuting:  "◒ 执行 ",
		stateObserving:  "◓ 分析结果",
		stateReviewing:  "⚠ 等待审批",
		stateStreaming:  "◑ 输出中",
		stateCompacting: "◎ 压缩上下文",
		stateCompleted:  "✓ 任务完成",
		stateError:      "✗ 执行出错",
	}

	for s, expected := range states {
		m.state = s
		if got := m.stateString(); got != expected {
			t.Errorf("state %d: expected %q, got %q", s, expected, got)
		}
	}
}

func TestTokenColorByPercent(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.maxTokens = 100

	// < 50%: 绿色
	m.totalTokens = 30
	footer := m.renderFooter()
	if footer == "" {
		t.Error("footer 不应为空")
	}

	// 50-80%: 橙色
	m.totalTokens = 60
	footer = m.renderFooter()
	if footer == "" {
		t.Error("footer 不应为空")
	}

	// > 80%: 红色
	m.totalTokens = 90
	footer = m.renderFooter()
	if footer == "" {
		t.Error("footer 不应为空")
	}
}

func TestRenderFooterEmptyModel(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.currentModel = ""
	m.modelName = ""

	footer := m.renderFooter()
	if !strings.Contains(footer, "未连接") {
		t.Error("模型为空时 footer 应显示 '未连接'")
	}
}

func TestUserMessageTokenCount(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24
	m.textarea.SetValue("hello world")

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newM.(Model)

	if model.totalTokens <= 0 {
		t.Error("用户发送消息后 totalTokens 应大于 0")
	}
}

// === 审批弹窗测试 ===

func TestConfirmModalMsg(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24

	resultCh := make(chan bool, 1)
	newM, _ := m.Update(confirmModalMsg{toolName: "test_tool", args: map[string]interface{}{"key": "value"}, resultCh: resultCh})
	model := newM.(Model)

	if !model.showConfirm {
		t.Error("收到 confirmModalMsg 后应显示审批弹窗")
	}
	if model.state != stateReviewing {
		t.Errorf("收到 confirmModalMsg 后状态应为 reviewing, 实际为 %d", model.state)
	}
	if model.pendingTool != "test_tool" {
		t.Errorf("pendingTool 应为 'test_tool', 实际为 %s", model.pendingTool)
	}
}

func TestConfirmModalApprove(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24
	m.showConfirm = true
	m.state = stateReviewing
	m.confirmResult = make(chan bool, 1)

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newM.(Model)

	if model.showConfirm {
		t.Error("按 Enter 后应关闭审批弹窗")
	}
	if model.state != stateExecuting {
		t.Errorf("批准后状态应为 executing, 实际为 %d", model.state)
	}

	select {
	case result := <-m.confirmResult:
		if !result {
			t.Error("Enter 应发送 true 到 resultCh")
		}
	default:
		t.Error("resultCh 应收到值")
	}
}

func TestConfirmModalApproveWithY(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24
	m.showConfirm = true
	m.state = stateReviewing
	m.confirmResult = make(chan bool, 1)

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model := newM.(Model)

	if model.showConfirm {
		t.Error("按 Y 后应关闭审批弹窗")
	}
	if model.state != stateExecuting {
		t.Errorf("批准后状态应为 executing, 实际为 %d", model.state)
	}

	select {
	case result := <-m.confirmResult:
		if !result {
			t.Error("Y 应发送 true 到 resultCh")
		}
	default:
		t.Error("resultCh 应收到值")
	}
}

func TestConfirmModalRejectWithN(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24
	m.showConfirm = true
	m.state = stateReviewing
	m.confirmResult = make(chan bool, 1)

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model := newM.(Model)

	if model.showConfirm {
		t.Error("按 N 后应关闭审批弹窗")
	}
	if model.state != stateIdle {
		t.Errorf("拒绝后状态应为 idle, 实际为 %d", model.state)
	}

	select {
	case result := <-m.confirmResult:
		if result {
			t.Error("N 应发送 false 到 resultCh")
		}
	default:
		t.Error("resultCh 应收到值")
	}
}

func TestConfirmModalRejectWithEsc(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24
	m.showConfirm = true
	m.state = stateReviewing
	m.confirmResult = make(chan bool, 1)

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newM.(Model)

	if model.showConfirm {
		t.Error("按 Esc 后应关闭审批弹窗")
	}
	if model.state != stateIdle {
		t.Errorf("拒绝后状态应为 idle, 实际为 %d", model.state)
	}

	select {
	case result := <-m.confirmResult:
		if result {
			t.Error("Esc 应发送 false 到 resultCh")
		}
	default:
		t.Error("resultCh 应收到值")
	}
}

func TestConfirmModalBlocksOtherKeys(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24
	m.showConfirm = true
	m.state = stateReviewing
	m.confirmResult = make(chan bool, 1)

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model := newM.(Model)

	if !model.showConfirm {
		t.Error("按其他键不应关闭审批弹窗")
	}
	if model.state != stateReviewing {
		t.Errorf("按其他键后状态应保持 reviewing, 实际为 %d", model.state)
	}
}

func TestRenderConfirmModal(t *testing.T) {
	m := NewModel(nil, context.Background())
	m.width = 80
	m.height = 24
	m.pendingTool = "ssh_execute"
	m.pendingArgs = map[string]interface{}{"command": "uptime"}

	modal := m.renderConfirmModal()
	if !strings.Contains(modal, "ssh_execute") {
		t.Error("弹窗应包含工具名")
	}
	if !strings.Contains(modal, "uptime") {
		t.Error("弹窗应包含参数")
	}
	if !strings.Contains(modal, "安全确认请求") {
		t.Error("弹窗应包含标题")
	}
}

func TestFormatArgs(t *testing.T) {
	args := map[string]interface{}{"key1": "value1", "key2": 42}
	s := formatArgs(args)
	if !strings.Contains(s, "key1=") {
		t.Error("formatArgs 应包含 key1")
	}
	if !strings.Contains(s, "key2=") {
		t.Error("formatArgs 应包含 key2")
	}

	empty := formatArgs(map[string]interface{}{})
	if empty != "{}" {
		t.Errorf("空参数应返回 {}, 实际为 %s", empty)
	}
}

// === convertLLMMessagesToChatMessages 测试 ===

func TestConvertLLMMessagesToChatMessages_过滤System消息(t *testing.T) {
	msgs := []llm.Message{
		{Role: "system", Content: "你是助手"},
		{Role: "user", Content: "你好"},
		{Role: "assistant", Content: "你好！"},
	}
	result := convertLLMMessagesToChatMessages(msgs)
	if len(result) != 2 {
		t.Fatalf("应过滤 system 消息，期望 2 条，实际 %d 条", len(result))
	}
	if result[0].Role != "user" {
		t.Errorf("第一条消息角色应为 user，实际为 %s", result[0].Role)
	}
	if result[1].Role != "assistant" {
		t.Errorf("第二条消息角色应为 assistant，实际为 %s", result[1].Role)
	}
}

func TestConvertLLMMessagesToChatMessages_空列表(t *testing.T) {
	result := convertLLMMessagesToChatMessages(nil)
	if len(result) != 0 {
		t.Errorf("空输入应返回空切片，实际长度 %d", len(result))
	}
}

func TestConvertLLMMessagesToChatMessages_ToolCalls附加到内容(t *testing.T) {
	msgs := []llm.Message{
		{
			Role:    "assistant",
			Content: "让我帮你查看",
			ToolCalls: []llm.ToolCall{
				{ID: "tc1", Type: "function", Function: llm.FunctionCall{Name: "local_bash", Arguments: `{"command":"ls"}`}},
				{ID: "tc2", Type: "function", Function: llm.FunctionCall{Name: "ssh_execute", Arguments: `{"command":"uptime"}`}},
			},
		},
	}
	result := convertLLMMessagesToChatMessages(msgs)
	if len(result) != 1 {
		t.Fatalf("期望 1 条消息，实际 %d 条", len(result))
	}
	if !strings.Contains(result[0].Content, "让我帮你查看") {
		t.Error("内容应包含原始文本")
	}
	if !strings.Contains(result[0].Content, "[工具调用: local_bash]") {
		t.Error("内容应附加工具调用 local_bash")
	}
	if !strings.Contains(result[0].Content, "[工具调用: ssh_execute]") {
		t.Error("内容应附加工具调用 ssh_execute")
	}
}

func TestConvertLLMMessagesToChatMessages_Timestamp非零(t *testing.T) {
	msgs := []llm.Message{
		{Role: "user", Content: "test"},
	}
	result := convertLLMMessagesToChatMessages(msgs)
	if result[0].Timestamp.IsZero() {
		t.Error("Timestamp 不应为零值")
	}
}

// === formatToolArgs 测试 ===

func TestFormatToolArgs_空参数(t *testing.T) {
	result := formatToolArgs("my_tool", nil)
	if result != "my_tool" {
		t.Errorf("空参数应只返回工具名，实际为 %s", result)
	}
	result = formatToolArgs("my_tool", map[string]interface{}{})
	if result != "my_tool" {
		t.Errorf("空 map 应只返回工具名，实际为 %s", result)
	}
}

func TestFormatToolArgs_跳过内部字段(t *testing.T) {
	args := map[string]interface{}{
		"_i":      123,
		"_intent": "query",
		"command": "ls",
	}
	result := formatToolArgs("tool", args)
	if strings.Contains(result, "_i=") {
		t.Error("应跳过 _i 字段")
	}
	if strings.Contains(result, "_intent=") {
		t.Error("应跳过 _intent 字段")
	}
	if !strings.Contains(result, "command=ls") {
		t.Error("应包含 command 字段")
	}
}

func TestFormatToolArgs_截断长值(t *testing.T) {
	longVal := strings.Repeat("a", 80)
	args := map[string]interface{}{"data": longVal}
	result := formatToolArgs("tool", args)
	if !strings.Contains(result, "...") {
		t.Error("超过 60 字符的值应被截断并添加省略号")
	}
	// 格式: tool(data=aaa...aaa...)
	if !strings.HasPrefix(result, "tool(") {
		t.Errorf("结果应以 'tool(' 开头，实际为 %s", result)
	}
}

func TestFormatToolArgs_正常格式(t *testing.T) {
	args := map[string]interface{}{
		"host": "server1",
		"port": 8080,
	}
	result := formatToolArgs("ssh_execute", args)
	if !strings.HasPrefix(result, "ssh_execute(") {
		t.Errorf("结果应以 'ssh_execute(' 开头，实际为 %s", result)
	}
	if !strings.Contains(result, "host=server1") {
		t.Error("应包含 host=server1")
	}
	if !strings.Contains(result, "port=8080") {
		t.Error("应包含 port=8080")
	}
}

// === truncateOutput 测试 ===

func TestTruncateOutput_移除空行(t *testing.T) {
	output := "line1\n\nline2\n\n\nline3"
	result := truncateOutput(output, 10)
	if strings.Contains(result, "\n\n") {
		t.Error("应移除空行")
	}
	if !strings.Contains(result, "line1") || !strings.Contains(result, "line2") || !strings.Contains(result, "line3") {
		t.Error("应保留所有非空行")
	}
}

func TestTruncateOutput_超过MaxLines截断(t *testing.T) {
	output := "line1\nline2\nline3\nline4\nline5"
	result := truncateOutput(output, 3)
	if !strings.Contains(result, "... (2 more lines)") {
		t.Errorf("超过 maxLines 应显示剩余行数，实际为 %s", result)
	}
	if !strings.Contains(result, "line1") || !strings.Contains(result, "line2") || !strings.Contains(result, "line3") {
		t.Error("应保留前 3 行")
	}
	if strings.Contains(result, "line4") {
		t.Error("不应包含被截断的行")
	}
}

func TestTruncateOutput_不超过MaxLines(t *testing.T) {
	output := "line1\nline2"
	result := truncateOutput(output, 5)
	if result != "line1\nline2" {
		t.Errorf("不超过 maxLines 应原样返回，实际为 %s", result)
	}
}

func TestTruncateOutput_空输出(t *testing.T) {
	result := truncateOutput("", 3)
	if result != "" {
		t.Errorf("空输入应返回空字符串，实际为 %s", result)
	}
}

// === formatDuration 测试 ===

func TestFormatDuration_毫秒(t *testing.T) {
	result := formatDuration(500 * time.Millisecond)
	if result != "500ms" {
		t.Errorf("500ms 应返回 '500ms'，实际为 %s", result)
	}
}

func TestFormatDuration_秒(t *testing.T) {
	result := formatDuration(1500 * time.Millisecond)
	if result != "1.5s" {
		t.Errorf("1.5s 应返回 '1.5s'，实际为 %s", result)
	}
}

func TestFormatDuration_整秒(t *testing.T) {
	result := formatDuration(5 * time.Second)
	if result != "5.0s" {
		t.Errorf("5s 应返回 '5.0s'，实际为 %s", result)
	}
}

func TestFormatDuration_分钟(t *testing.T) {
	result := formatDuration(2 * time.Minute)
	if result != "2.0m" {
		t.Errorf("2m 应返回 '2.0m'，实际为 %s", result)
	}
}

func TestFormatDuration_分钟秒(t *testing.T) {
	result := formatDuration(90 * time.Second)
	if result != "1.5m" {
		t.Errorf("90s 应返回 '1.5m'，实际为 %s", result)
	}
}

func TestFormatDuration_零(t *testing.T) {
	result := formatDuration(0)
	if result != "0ms" {
		t.Errorf("0 应返回 '0ms'，实际为 %s", result)
	}
}

// TestParseListItem 测试列表项解析
func TestParseListItem(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantMarker   string
		wantText     string
	}{
		{"dash列表", "- item text", "•", "item text"},
		{"star列表", "* item text", "•", "item text"},
		{"数字列表", "1. first item", "1.", "first item"},
		{"数字列表_多位", "12. twelfth item", "12.", "twelfth item"},
		{"无前缀_回退", "plain text", "•", "plain text"},
		{"空字符串", "", "•", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marker, text := parseListItem(tt.input)
			if marker != tt.wantMarker {
				t.Errorf("parseListItem(%q) marker = %q, want %q", tt.input, marker, tt.wantMarker)
			}
			if text != tt.wantText {
				t.Errorf("parseListItem(%q) text = %q, want %q", tt.input, text, tt.wantText)
			}
		})
	}
}
