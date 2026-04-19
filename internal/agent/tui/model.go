package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"opsxcli/internal/agent/session"
	"opsxcli/internal/agent/tools"
	"opsxcli/internal/llm"
)

// ChatMessage 对话消息
type ChatMessage struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// sessionState 会话状态
type sessionState int

const (
	stateIdle sessionState = iota
	stateThinking  // LLM 思考中
	stateStreaming // 流式输出中
	stateExecuting // 工具执行中
)

// viewState 视图状态
type viewState int

const (
	viewChat viewState = iota // 聊天视图（默认）
	viewSessionList           // 会话列表视图
)

// AgentRunner 定义 Agent 的流式执行接口，避免循环导入
type AgentRunner interface {
	RunStream(ctx context.Context, query string, out io.Writer) (*tools.Result, error)
}

// Model Bubble Tea Model
type Model struct {
	agent AgentRunner
	ctx   context.Context

	// bubbles 组件
	viewport viewport.Model // 消息历史滚动区域
	textarea textarea.Model // 输入框
	spinner  spinner.Model  // 加载动画

	// 状态
	messages []ChatMessage
	state    sessionState
	width    int
	height   int
	err      error

	// 异步消息通道
	msgChan chan tea.Msg

	// 会话管理
	viewState        viewState
	sessions         []*session.Session
	listCursor       int
	sessionMgr       session.Manager
	currentSessionID string
	deleteConfirm    bool
}

// NewModel 创建 TUI Model
func NewModel(agent AgentRunner, ctx context.Context) Model {
	ta := textarea.New()
	ta.Placeholder = "输入运维问题，按 Enter 发送..."
	ta.Focus()

	vp := viewport.New(80, 20)

	s := spinner.New()
	s.Spinner = spinner.Dot

	// 初始化 session manager
	var sessionMgr session.Manager
	homeDir, err := os.UserHomeDir()
	if err == nil {
		sessionDir := filepath.Join(homeDir, ".opsxcli", "agent", "sessions")
		store, err := session.NewJSONLStore(sessionDir)
		if err == nil {
			sessionMgr = session.NewManager(store)
		}
	}

	return Model{
		agent:      agent,
		ctx:        ctx,
		textarea:   ta,
		viewport:   vp,
		spinner:    s,
		messages:   make([]ChatMessage, 0),
		state:      stateIdle,
		msgChan:    make(chan tea.Msg, 64),
		sessionMgr: sessionMgr,
	}
}

// Init 初始化 Bubble Tea 命令
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.textarea.Focus(),
		m.viewport.Init(),
		m.spinner.Tick,
	)
}

// Update 处理消息和状态更新
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// 为输入框、状态栏、标题预留空间后计算 viewport 高度
		vpHeight := msg.Height - 14
		if vpHeight < 3 {
			vpHeight = 3
		}
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = vpHeight

		// 调整输入框尺寸
		m.textarea.SetWidth(msg.Width - 4)
		m.textarea.SetHeight(3)

		return m, nil

	case tea.KeyMsg:
		if m.viewState == viewSessionList {
			return m.handleSessionListKeys(msg)
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyCtrlO:
			if m.viewState == viewChat {
				m.viewState = viewSessionList
				m.deleteConfirm = false
				return m, m.loadSessionList()
			}
		case tea.KeyEnter:
			if m.state != stateIdle {
				return m, nil // 忙时忽略
			}
			query := strings.TrimSpace(m.textarea.Value())
			if query == "" {
				return m, nil
			}
			if query == "/exit" || query == "/quit" {
				return m, tea.Quit
			}
			if query == "/help" {
				m.messages = append(m.messages, ChatMessage{
					Role:      "system",
					Content:   "命令: /exit, /quit - 退出 | /new - 新建会话 | /sessions - 会话列表 | Enter - 发送 | ESC - 退出 | Ctrl+O - 切换会话列表",
					Timestamp: time.Now(),
				})
				m.textarea.Reset()
				m.viewport.SetContent(m.renderMessages())
				m.viewport.GotoBottom()
				return m, nil
			}
			if query == "/sessions" {
				m.textarea.Reset()
				m.viewState = viewSessionList
				m.deleteConfirm = false
				return m, m.loadSessionList()
			}
			if query == "/new" {
				m.textarea.Reset()
				return m, m.createNewSession()
			}

			// 添加用户消息
			m.messages = append(m.messages, ChatMessage{
				Role:      "user",
				Content:   query,
				Timestamp: time.Now(),
			})
			m.textarea.Reset()
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			m.state = stateThinking

			// 保存用户消息到会话
			if m.sessionMgr != nil && m.currentSessionID != "" {
				_ = m.sessionMgr.SaveMessage(m.currentSessionID, llm.Message{
					Role:    "user",
					Content: query,
				})
			}

			// 启动 Agent 查询
			return m, m.runAgent(query)
		}

	case streamChunkMsg:
		// 流式输出：追加到最后一条 assistant 消息
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
			m.messages[len(m.messages)-1].Content += msg.content
		} else {
			m.messages = append(m.messages, ChatMessage{
				Role:      "assistant",
				Content:   msg.content,
				Timestamp: time.Now(),
			})
		}
		m.state = stateStreaming
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		// 继续等待下一条消息
		return m, m.waitForMsg()

	case thinkDoneMsg:
		m.state = stateIdle
		// 保存助手回复到会话
		if m.sessionMgr != nil && m.currentSessionID != "" {
			if len(m.messages) > 0 {
				lastMsg := m.messages[len(m.messages)-1]
				if lastMsg.Role == "assistant" {
					_ = m.sessionMgr.SaveMessage(m.currentSessionID, llm.Message{
						Role:    "assistant",
						Content: lastMsg.Content,
					})
				}
			}
		}
		return m, nil

	case toolStartMsg:
		m.state = stateExecuting
		m.messages = append(m.messages, ChatMessage{
			Role:      "tool",
			Content:   fmt.Sprintf("正在执行: %s...", msg.name),
			Timestamp: time.Now(),
		})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, m.waitForMsg()

	case toolDoneMsg:
		m.state = stateIdle
		return m, nil

	case errorMsg:
		m.state = stateIdle
		m.err = msg.err
		m.messages = append(m.messages, ChatMessage{
			Role:      "system",
			Content:   fmt.Sprintf("错误: %v", msg.err),
			Timestamp: time.Now(),
		})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case sessionListMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.sessions = msg.sessions
			if m.listCursor >= len(m.sessions) {
				m.listCursor = 0
			}
		}
		return m, nil

	case sessionLoadedMsg:
		m.viewState = viewChat
		m.deleteConfirm = false
		if msg.err != nil {
			m.err = msg.err
			m.messages = append(m.messages, ChatMessage{
				Role:      "system",
				Content:   fmt.Sprintf("加载会话失败: %v", msg.err),
				Timestamp: time.Now(),
			})
		} else {
			m.currentSessionID = msg.session.ID
			m.messages = convertLLMMessagesToChatMessages(msg.messages)
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}
		return m, nil

	case sessionDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// 如果被删除的是当前会话，清空当前会话
			if msg.sessionID == m.currentSessionID {
				m.currentSessionID = ""
				m.messages = make([]ChatMessage, 0)
			}
			return m, m.loadSessionList()
		}
		return m, nil

	case sessionCreatedMsg:
		m.deleteConfirm = false
		if msg.err != nil {
			m.err = msg.err
			m.messages = append(m.messages, ChatMessage{
				Role:      "system",
				Content:   fmt.Sprintf("创建会话失败: %v", msg.err),
				Timestamp: time.Now(),
			})
		} else {
			m.currentSessionID = msg.session.ID
			m.messages = make([]ChatMessage, 0)
			m.viewport.SetContent("")
		}
		return m, nil
	}

	// 更新子组件
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleSessionListKeys 处理会话列表视图的按键
func (m Model) handleSessionListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc, tea.KeyCtrlO:
		m.viewState = viewChat
		m.deleteConfirm = false
		return m, nil
	case tea.KeyUp:
		m.deleteConfirm = false
		if m.listCursor > 0 {
			m.listCursor--
		}
		return m, nil
	case tea.KeyDown:
		m.deleteConfirm = false
		if m.listCursor < len(m.sessions)-1 {
			m.listCursor++
		}
		return m, nil
	case tea.KeyEnter:
		m.deleteConfirm = false
		if m.listCursor >= 0 && m.listCursor < len(m.sessions) {
			return m, m.loadSession(m.sessions[m.listCursor].ID)
		}
		return m, nil
	}

	switch msg.String() {
	case "d", "D":
		if len(m.sessions) == 0 {
			return m, nil
		}
		if m.deleteConfirm {
			m.deleteConfirm = false
			if m.listCursor >= 0 && m.listCursor < len(m.sessions) {
				return m, m.deleteSession(m.sessions[m.listCursor].ID)
			}
			return m, nil
		}
		m.deleteConfirm = true
		return m, nil
	case "n", "N":
		m.deleteConfirm = false
		return m, m.createNewSession()
	}

	// 其他按键取消删除确认
	m.deleteConfirm = false
	return m, nil
}

// loadSessionList 加载会话列表
func (m Model) loadSessionList() tea.Cmd {
	return func() tea.Msg {
		if m.sessionMgr == nil {
			return sessionListMsg{err: fmt.Errorf("会话管理器未初始化")}
		}
		sessions, err := m.sessionMgr.List(100)
		return sessionListMsg{sessions: sessions, err: err}
	}
}

// loadSession 加载指定会话
func (m Model) loadSession(sessionID string) tea.Cmd {
	return func() tea.Msg {
		if m.sessionMgr == nil {
			return sessionLoadedMsg{err: fmt.Errorf("会话管理器未初始化")}
		}
		sess, messages, err := m.sessionMgr.Load(sessionID)
		return sessionLoadedMsg{session: sess, messages: messages, err: err}
	}
}

// deleteSession 删除指定会话
func (m Model) deleteSession(sessionID string) tea.Cmd {
	return func() tea.Msg {
		if m.sessionMgr == nil {
			return sessionDeletedMsg{err: fmt.Errorf("会话管理器未初始化")}
		}
		err := m.sessionMgr.Delete(sessionID)
		return sessionDeletedMsg{sessionID: sessionID, err: err}
	}
}

// createNewSession 创建新会话
func (m Model) createNewSession() tea.Cmd {
	return func() tea.Msg {
		if m.sessionMgr == nil {
			return sessionCreatedMsg{err: fmt.Errorf("会话管理器未初始化")}
		}
		sess, err := m.sessionMgr.Create("", "", "")
		return sessionCreatedMsg{session: sess, err: err}
	}
}

// View 渲染界面
func (m Model) View() string {
	if m.width == 0 {
		return "初始化中..."
	}

	if m.viewState == viewSessionList {
		return m.renderSessionList()
	}

	// 标题栏
	title := titleStyle.Render("🤖 opsxcli Agent V2 — 交互模式")
	subtitle := subtitleStyle.Render("opsxcli 智能运维助手")

	// 状态栏
	var status string
	switch m.state {
	case stateThinking:
		status = fmt.Sprintf("%s 思考中...", m.spinner.View())
	case stateStreaming:
		status = fmt.Sprintf("%s 输出中...", m.spinner.View())
	case stateExecuting:
		status = fmt.Sprintf("%s 执行工具...", m.spinner.View())
	default:
		status = "就绪 | Enter 发送 | ESC 退出 | Ctrl+O 会话列表"
	}
	statusBar := statusStyle.Width(m.width - 4).Render(status)

	// 视图区域
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		subtitle,
		"",
		viewportStyle.Width(m.width).Render(m.viewport.View()),
		"",
		statusBar,
		inputStyle.Width(m.width - 2).Render(m.textarea.View()),
	)
}

// renderMessages 渲染消息历史为字符串
func (m Model) renderMessages() string {
	var b strings.Builder
	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			b.WriteString(userStyle.Render("👤 您: "))
			b.WriteString(msg.Content)
		case "assistant":
			b.WriteString(assistantStyle.Render(msg.Content))
		case "tool":
			b.WriteString(toolStyle.Render(msg.Content))
		case "system":
			b.WriteString(errorStyle.Render(msg.Content))
		}
		b.WriteString("\n\n")
	}
	return b.String()
}

// renderSessionList 渲染会话列表
func (m Model) renderSessionList() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📁 会话列表"))
	b.WriteString("\n\n")

	if len(m.sessions) == 0 {
		b.WriteString(itemStyle.Render("暂无历史会话"))
		b.WriteString("\n")
	} else {
		for i, sess := range m.sessions {
			prefix := "  "
			if i == m.listCursor {
				prefix = "> "
			}
			line := fmt.Sprintf("%s%s (%d条消息) %s",
				prefix, sess.Title, sess.MessageCount,
				sess.UpdatedAt.Format("01-02 15:04"))
			if i == m.listCursor {
				line = selectedStyle.Render(line)
			} else {
				line = itemStyle.Render(line)
			}
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	if m.deleteConfirm && len(m.sessions) > 0 {
		b.WriteString(errorStyle.Render("⚠️  按 D 再次确认删除，或按其他键取消"))
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render("↑↓ 选择 | Enter 加载 | N 新建 | D 删除 | Ctrl+O 返回"))
	return b.String()
}

// runAgent 启动 Agent 查询（在 goroutine 中执行）
func (m Model) runAgent(query string) tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					m.msgChan <- errorMsg{err: fmt.Errorf("Agent panic: %v", r)}
				}
			}()

			w := &modelWriter{ch: m.msgChan}
			_, err := m.agent.RunStream(m.ctx, query, w)
			if err != nil {
				m.msgChan <- errorMsg{err: err}
			} else {
				m.msgChan <- thinkDoneMsg{}
			}
		}()

		// 返回第一个消息（等待从 channel 读取）
		return <-m.msgChan
	}
}

// waitForMsg 等待下一条异步消息
func (m Model) waitForMsg() tea.Cmd {
	return func() tea.Msg {
		return <-m.msgChan
	}
}

// modelWriter 将输出转换为 streamChunkMsg
type modelWriter struct {
	ch chan tea.Msg
}

func (w *modelWriter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		w.ch <- streamChunkMsg{content: string(p)}
	}
	return len(p), nil
}

// convertLLMMessagesToChatMessages 将 llm.Message 转换为 ChatMessage
func convertLLMMessagesToChatMessages(msgs []llm.Message) []ChatMessage {
	result := make([]ChatMessage, 0, len(msgs))
	for _, msg := range msgs {
		role := msg.Role
		content := msg.Content
		if role == "system" {
			// 系统消息在聊天视图中不显示
			continue
		}
		// 如果有工具调用，附加到内容中
		if len(msg.ToolCalls) > 0 {
			var b strings.Builder
			b.WriteString(content)
			for _, tc := range msg.ToolCalls {
				b.WriteString(fmt.Sprintf("\n[工具调用: %s]", tc.Function.Name))
			}
			content = b.String()
		}
		result = append(result, ChatMessage{
			Role:      role,
			Content:   content,
			Timestamp: time.Now(),
		})
	}
	return result
}
