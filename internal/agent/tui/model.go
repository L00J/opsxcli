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
	stateIdle       sessionState = iota // 就绪
	statePlanning                       // 规划任务（/plan 模式）
	stateThinking                       // LLM 思考中
	stateExecuting                      // 工具执行中
	stateObserving                      // 观察工具结果
	stateReviewing                      // 审批等待中
	stateStreaming                      // 流式输出中
	stateCompacting                     // 压缩上下文
	stateCompleted                      // 任务完成
	stateError                          // 错误状态
)

// viewState 视图状态
type viewState int

const (
	viewChat        viewState = iota // 聊天视图（默认）
	viewSessionList                  // 会话列表视图
	viewDashboard                    // 仪表板视图
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

	// 显示信息
	modelName       string
	currentModel    string // 当前 LLM 模型（如 "deepseek-chat"）
	currentProvider string // 当前提供商（如 "deepseek"）
	totalTokens     int    // 当前会话累计 token 数
	maxTokens       int    // 上下文窗口上限
	currentHost     string // 当前远程主机（如 "local"）

	// 审批弹窗
	showConfirm   bool
	pendingTool   string
	pendingArgs   map[string]interface{}
	confirmResult chan bool

	// 当前执行的工具
	currentTool string

	// 仪表板 (v0.6.0)
	dashTab           dashboardTab
	dashCursor        int
	dashboardData     DashboardData
	memoryStatsLoader func() (map[string]interface{}, error)
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
		agent:           agent,
		ctx:             ctx,
		textarea:        ta,
		viewport:        vp,
		spinner:         s,
		messages:        make([]ChatMessage, 0),
		state:           stateIdle,
		msgChan:         make(chan tea.Msg, 64),
		sessionMgr:      sessionMgr,
		modelName:       "deepseek-chat",
		currentModel:    "deepseek-chat",
		currentProvider: "deepseek",
		maxTokens:       6000,
		currentHost:     "local",
		totalTokens:     0,
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
		vpHeight := msg.Height - 12
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
		// 审批弹窗拦截所有按键
		if m.showConfirm {
			keyStr := msg.String()
			switch {
			case keyStr == "y" || keyStr == "Y" || msg.Type == tea.KeyEnter:
				m.showConfirm = false
				if m.confirmResult != nil {
					m.confirmResult <- true
				}
				m.state = stateExecuting
				return m, m.waitForMsg()
			case keyStr == "n" || keyStr == "N" || msg.Type == tea.KeyEsc:
				m.showConfirm = false
				if m.confirmResult != nil {
					m.confirmResult <- false
				}
				m.state = stateIdle
				return m, m.waitForMsg()
			}
			return m, nil // 拦截其他按键
		}

		if m.viewState == viewSessionList {
			return m.handleSessionListKeys(msg)
		}

		if m.viewState == viewDashboard {
			return m.handleDashboardKeys(msg)
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
		case tea.KeyCtrlD:
			if m.viewState == viewChat {
				m.viewState = viewDashboard
				m.dashTab = tabOverview
				m.dashCursor = 0
				return m, m.loadDashboardCmd()
			}
		case tea.KeyEnter:
			if m.state != stateIdle && m.state != stateCompleted && m.state != stateError {
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
					Content:   "命令: /exit, /quit - 退出 | /new - 新建会话 | /sessions - 会话列表 | /dashboard, /skills, /memory - 仪表板 | Enter - 发送 | ESC - 退出 | Ctrl+O - 会话列表 | Ctrl+M - 仪表板",
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
			if query == "/dashboard" || query == "/skills" || query == "/memory" {
				m.textarea.Reset()
				m.viewState = viewDashboard
				m.dashTab = tabOverview
				if query == "/skills" {
					m.dashTab = tabSkills
				}
				m.dashCursor = 0
				return m, m.loadDashboardCmd()
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
			m.totalTokens += estimateTokens(query)
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
		m.totalTokens += estimateTokens(msg.content)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		// 继续等待下一条消息
		return m, m.waitForMsg()

	case thinkDoneMsg:
		m.state = stateCompleted
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
		m.currentTool = msg.name
		// 格式化工具调用详情
		var detail string
		switch msg.name {
		case "local_bash", "bash", "shell":
			if cmd, ok := msg.args["command"].(string); ok {
				detail = fmt.Sprintf("执行命令: %s", cmd)
			} else {
				detail = fmt.Sprintf("执行: %s", formatToolArgs(msg.name, msg.args))
			}
		case "ssh_execute", "remote_bash":
			host, _ := msg.args["host"].(string)
			cmd, _ := msg.args["command"].(string)
			if host != "" && cmd != "" {
				detail = fmt.Sprintf("SSH %s → %s", host, cmd)
			} else {
				detail = fmt.Sprintf("执行: %s", formatToolArgs(msg.name, msg.args))
			}
		default:
			detail = formatToolArgs(msg.name, msg.args)
		}
		m.messages = append(m.messages, ChatMessage{
			Role:      "tool",
			Content:   detail,
			Timestamp: time.Now(),
		})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, m.waitForMsg()

	case toolDoneMsg:
		m.state = stateObserving
		m.currentTool = ""
		// 显示工具执行结果摘要
		status := "✅"
		if !msg.success {
			status = "❌"
		}
		durStr := formatDuration(msg.duration)
		var content string
		if msg.output != "" {
			preview := truncateOutput(msg.output, 3)
			content = fmt.Sprintf("%s %s 完成 (%s)\n%s", status, msg.name, durStr, preview)
		} else {
			content = fmt.Sprintf("%s %s 完成 (%s)", status, msg.name, durStr)
		}
		m.messages = append(m.messages, ChatMessage{
			Role:      "tool_result",
			Content:   content,
			Timestamp: time.Now(),
		})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case errorMsg:
		m.state = stateError
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

	case confirmModalMsg:
		m.showConfirm = true
		m.pendingTool = msg.toolName
		m.pendingArgs = msg.args
		m.confirmResult = msg.resultCh
		m.state = stateReviewing
		return m, nil

	case dashboardLoadedMsg:
		m.dashboardData = msg.data
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

	if m.viewState == viewDashboard {
		return m.renderDashboard()
	}

	title := titleStyle.Render("🤖 opsxcli 智能运维助手")

	// Powerline Footer
	footer := m.renderFooter()

	view := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		viewportStyle.Width(m.width).Render(m.viewport.View()),
		"",
		footer,
		inputStyle.Width(m.width - 2).Render(m.textarea.View()),
	)

	// 审批弹窗叠加
	if m.showConfirm {
		modal := m.renderConfirmModal()
		view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
	}

	return view
}

// renderFooter 渲染 Powerline 风格底部状态栏
func (m Model) renderFooter() string {
	if m.width == 0 {
		return ""
	}

	// 左侧: 模型信息
	modelInfo := m.currentModel
	if modelInfo == "" {
		modelInfo = m.modelName
	}
	if modelInfo == "" {
		modelInfo = "未连接"
	}
	modelStr := fmt.Sprintf(" %s ", modelInfo)

	// 中间: Token 使用率
	tokenPercent := 0
	if m.maxTokens > 0 {
		tokenPercent = (m.totalTokens * 100) / m.maxTokens
	}
	tokenStr := fmt.Sprintf(" %d/%d (%d%%) ", m.totalTokens, m.maxTokens, tokenPercent)

	// 根据使用率选择颜色
	tokenStyle := footerTokenStyle
	if tokenPercent > 80 {
		tokenStyle = footerTokenDangerStyle
	} else if tokenPercent > 50 {
		tokenStyle = footerTokenWarnStyle
	}

	// 右侧: 状态 + 主机
	statusStr := m.stateString()
	hostStr := ""
	if m.currentHost != "" && m.currentHost != "local" {
		hostStr = fmt.Sprintf(" %s", m.currentHost)
	}
	rightStr := fmt.Sprintf(" %s%s ", statusStr, hostStr)

	// Powerline 风格分段渲染
	left := footerModelStyle.Render(modelStr)
	mid := tokenStyle.Render(tokenStr)
	right := footerStatusStyle.Render(rightStr)

	return lipgloss.JoinHorizontal(lipgloss.Left, left, mid, right)
}

// stateString 返回当前状态的字符串表示
func (m Model) stateString() string {
	switch m.state {
	case stateIdle:
		return "● 就绪"
	case statePlanning:
		return "◐ 规划任务"
	case stateThinking:
		return "◐ 思考中"
	case stateExecuting:
		return fmt.Sprintf("◒ 执行 %s", m.currentTool)
	case stateObserving:
		return "◓ 分析结果"
	case stateReviewing:
		return "⚠ 等待审批"
	case stateStreaming:
		return "◑ 输出中"
	case stateCompacting:
		return "◎ 压缩上下文"
	case stateCompleted:
		return "✓ 任务完成"
	case stateError:
		return "✗ 执行出错"
	default:
		return "○ 未知"
	}
}

// renderConfirmModal 渲染审批弹窗
func (m Model) renderConfirmModal() string {
	var b strings.Builder

	b.WriteString(modalTitleStyle.Render("⚠️ 安全确认请求"))
	b.WriteString("\n\n")

	b.WriteString(modalContentStyle.Render(fmt.Sprintf("工具: %s", m.pendingTool)))
	b.WriteString("\n")

	// 格式化参数
	argsStr := formatArgs(m.pendingArgs)
	b.WriteString(modalContentStyle.Render(fmt.Sprintf("参数: %s", argsStr)))
	b.WriteString("\n\n")

	b.WriteString(modalButtonStyle.Render("[Y] 确认执行  [N] 取消"))
	b.WriteString("\n")

	return modalBoxStyle.Render(b.String())
}

// renderMessages 渲染消息历史为字符串
func (m Model) renderMessages() string {
	var b strings.Builder
	contentWidth := m.viewport.Width - 8
	if contentWidth < 20 {
		contentWidth = 20
	}

	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			b.WriteString(userStyle.Render("👤 您"))
			b.WriteString("\n")
			b.WriteString(msg.Content)
		case "assistant":
			// 检测内容中是否已包含 stream.go 输出的 🎨 Agent 前缀
			content := msg.Content
			if !hasAgentPrefix(content) {
				b.WriteString(assistantStyle.Render("🤖 Agent"))
				b.WriteString("\n")
			}
			rendered := RenderMarkdown(content, contentWidth)
			b.WriteString(assistantBubbleStyle.Render(rendered))
		case "tool":
			b.WriteString(toolStyle.Render("🔧 " + msg.Content))
		case "tool_result":
			b.WriteString(toolResultStyle.Render(msg.Content))
		case "system":
			b.WriteString(errorStyle.Render("⚠️ " + msg.Content))
		}
		b.WriteString("\n\n")
	}
	return b.String()
}

// hasAgentPrefix 检测 assistant 内容是否已包含 stream.go 输出的前缀
func hasAgentPrefix(content string) bool {
	// stream.go 输出的前缀包含 "Agent" 和 "🤖"
	// 简单检测：前 200 个字符内同时出现 "Agent" 和 "🤖"
	checkLen := 200
	if len(content) < checkLen {
		checkLen = len(content)
	}
	prefix := content[:checkLen]
	return strings.Contains(prefix, "Agent") && strings.Contains(prefix, "🤖")
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
	b.WriteString("\n")
	b.WriteString(m.renderFooter())
	return b.String()
}

// runAgent 启动 Agent 查询（在 goroutine 中执行）
func (m Model) runAgent(query string) tea.Cmd {
	return func() tea.Msg {
		// 设置 TUI 审批函数
		if setter, ok := m.agent.(interface {
			SetConfirmFn(func(toolName string, args map[string]interface{}, risk tools.RiskLevel) (bool, error))
		}); ok {
			setter.SetConfirmFn(func(toolName string, args map[string]interface{}, risk tools.RiskLevel) (bool, error) {
				resultCh := make(chan bool, 1)
				m.msgChan <- confirmModalMsg{toolName: toolName, args: args, resultCh: resultCh}
				return <-resultCh, nil
			})
		}

		// 设置工具执行回调（用于 TUI 显示工具执行进度）
		if setter, ok := m.agent.(interface {
			SetToolCallback(func(name string, args map[string]interface{}, start bool, success bool, duration time.Duration, output string))
		}); ok {
			setter.SetToolCallback(func(name string, args map[string]interface{}, start bool, success bool, duration time.Duration, output string) {
				if start {
					m.msgChan <- toolStartMsg{name: name, args: args}
				} else {
					m.msgChan <- toolDoneMsg{name: name, success: success, duration: duration, output: output}
				}
			})
		}

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

// formatArgs 格式化参数显示（本地副本，避免跨包依赖）
func formatArgs(args map[string]interface{}) string {
	if len(args) == 0 {
		return "{}"
	}

	parts := make([]string, 0, len(args))
	for k, v := range args {
		switch val := v.(type) {
		case string:
			if len(val) > 100 {
				val = val[:100] + "..."
			}
			parts = append(parts, fmt.Sprintf("%s=%q", k, val))
		default:
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// estimateTokens 简单估算 token 数
func estimateTokens(s string) int {
	// 简单估算策略：中文/符号 1 字 ≈ 1 token，英文 4 字符 ≈ 1 token
	// 这里采用更简化的方式：按 rune 计数，空白字符不计
	runes := []rune(s)
	count := 0
	for _, r := range runes {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			continue
		}
		if r <= 127 {
			// ASCII 字符（主要是英文），4 字符约 1 token
			count += 1
		} else {
			// 非 ASCII（主要是中文），1 字约 1 token
			count += 4
		}
	}
	return count / 4
}

// formatToolArgs 格式化工具名和参数为可读字符串
func formatToolArgs(name string, args map[string]interface{}) string {
	if len(args) == 0 {
		return name
	}
	parts := make([]string, 0, len(args))
	for k, v := range args {
		// 跳过内部字段
		if k == "_i" || k == "_intent" {
			continue
		}
		s := fmt.Sprintf("%v", v)
		if len(s) > 60 {
			s = s[:60] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, s))
	}
	return fmt.Sprintf("%s(%s)", name, strings.Join(parts, ", "))
}

// truncateOutput 截断工具输出到指定行数
func truncateOutput(output string, maxLines int) string {
	lines := strings.Split(output, "\n")
	// 移除空行
	var nonEmpty []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}
	if len(nonEmpty) <= maxLines {
		return strings.Join(nonEmpty, "\n")
	}
	return strings.Join(nonEmpty[:maxLines], "\n") +
		fmt.Sprintf("\n  ... (%d more lines)", len(nonEmpty)-maxLines)
}

// formatDuration 格式化耗时
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
