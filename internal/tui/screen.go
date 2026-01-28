package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// Screen 专业的终端屏幕管理
type Screen struct {
	mu            sync.RWMutex
	messages      []Message
	statusLine    string
	inputBuffer   string
	inputPrompt   string
	cursorPos     int
	isThinking    bool
	thinkingStart time.Time
	tokens        int
	step          int
	maxSteps      int
	height        int
	width         int
	inputChan     chan string
	quitChan      chan struct{}
}

// Message 消息结构
type Message struct {
	Role    string
	Content string
	Time    time.Time
}

// NewScreen 创建新屏幕
func NewScreen() (*Screen, error) {
	width, height, err := term.GetSize(0)
	if err != nil {
		// 默认尺寸
		width, height = 80, 24
	}

	s := &Screen{
		messages:    make([]Message, 0),
		inputPrompt: "⏵ ",
		height:      height,
		width:       width,
		inputChan:   make(chan string, 10),
		quitChan:    make(chan struct{}),
	}

	return s, nil
}

// AddMessage 添加消息
func (s *Screen) AddMessage(role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, Message{
		Role:    role,
		Content: content,
		Time:    time.Now(),
	})
}

// SetStatus 设置状态行
func (s *Screen) SetStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusLine = status
}

// StartThinking 开始思考
func (s *Screen) StartThinking(maxSteps int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isThinking = true
	s.thinkingStart = time.Now()
	s.step = 0
	s.maxSteps = maxSteps
	s.tokens = 0

	go s.updateThinkingStatus()
}

// UpdateProgress 更新进度
func (s *Screen) UpdateProgress(step, tokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.step = step
	s.tokens = tokens
}

// StopThinking 停止思考
func (s *Screen) StopThinking() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isThinking = false
	s.statusLine = ""
}

// updateThinkingStatus 更新思考状态
func (s *Screen) updateThinkingStatus() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frameIndex := 0

	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			if !s.isThinking {
				s.mu.RUnlock()
				return
			}

			elapsed := time.Since(s.thinkingStart)
			spinner := spinnerFrames[frameIndex%len(spinnerFrames)]

			status := fmt.Sprintf("%s Thinking… (%s", spinner, formatDuration(elapsed))
			if s.step > 0 {
				status += fmt.Sprintf(" · step %d/%d", s.step, s.maxSteps)
			}
			if s.tokens > 0 {
				status += fmt.Sprintf(" · ↓ %s tokens", formatNumber(s.tokens))
			}
			status += ")"

			s.mu.RUnlock()
			s.SetStatus(status)
			s.Render()

			frameIndex++

		case <-s.quitChan:
			return
		}
	}
}

// Render 渲染屏幕
func (s *Screen) Render() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 移动光标到底部
	fmt.Print("\033[H\033[J") // 清屏

	// 渲染消息（保留最近的几条）
	visibleLines := s.height - 4 // 保留底部输入区域
	messagesToShow := s.messages
	if len(messagesToShow) > visibleLines/3 {
		messagesToShow = messagesToShow[len(messagesToShow)-visibleLines/3:]
	}

	for _, msg := range messagesToShow {
		s.renderMessage(msg)
	}

	// 渲染分隔线和输入区
	s.renderInputArea()
}

// renderMessage 渲染消息
func (s *Screen) renderMessage(msg Message) {
	var roleColor *color.Color
	var roleIcon string

	switch msg.Role {
	case "user":
		roleColor = color.New(color.FgGreen, color.Bold)
		roleIcon = "👤"
	case "assistant":
		roleColor = color.New(color.FgCyan)
		roleIcon = "🤖"
	case "tool":
		roleColor = color.New(color.FgMagenta)
		roleIcon = "  →"
	case "status":
		roleColor = color.New(color.FgYellow)
		roleIcon = "✻"
	default:
		roleColor = color.New(color.FgWhite)
		roleIcon = "·"
	}

	if msg.Role == "tool" {
		fmt.Printf("%s %s\n", roleColor.Sprint(roleIcon), msg.Content)
	} else {
		fmt.Printf("\n%s %s\n", roleIcon, roleColor.Sprint(strings.Title(msg.Role))+":")
		// 自动换行
		s.printWrapped(msg.Content, 2)
	}
}

// renderInputArea 渲染输入区域
func (s *Screen) renderInputArea() {
	// 移动到倒数第3行
	rows := s.height - 3
	fmt.Printf("\033[%d;0H", rows)

	// 渲染分隔线
	separator := strings.Repeat("─", s.width)
	fmt.Println(color.HiBlackString(separator))

	// 渲染状态行（如果有）
	if s.statusLine != "" {
		fmt.Print("\r")
		statusColor := color.New(color.FgCyan)
		fmt.Print(statusColor.Sprint(s.statusLine))
		fmt.Print("  ")
		fmt.Print(color.HiBlackString("[type to add context, Ctrl+C to cancel]"))
		fmt.Println()
	} else {
		fmt.Println()
	}

	// 渲染输入行
	prompt := color.GreenString(s.inputPrompt)
	fmt.Printf("%s%s", prompt, s.inputBuffer)

	// 显示光标
	if s.cursorPos < len(s.inputBuffer) {
		fmt.Printf("\033[%dD", len(s.inputBuffer)-s.cursorPos)
	}
}

// printWrapped 打印自动换行的文本
func (s *Screen) printWrapped(text string, indent int) {
	maxWidth := s.width - indent - 2
	if maxWidth < 40 {
		maxWidth = 40
	}

	lines := strings.Split(text, "\n")
	indentStr := strings.Repeat(" ", indent)

	for _, line := range lines {
		if len(line) <= maxWidth {
			fmt.Println(indentStr + line)
			continue
		}

		// 简单的单词边界换行
		words := strings.Fields(line)
		currentLine := ""
		for _, word := range words {
			if len(currentLine)+len(word)+1 > maxWidth {
				if currentLine != "" {
					fmt.Println(indentStr + currentLine)
					currentLine = word
				} else {
					// 单词太长,强制截断
					fmt.Println(indentStr + word[:maxWidth])
					currentLine = ""
				}
			} else {
				if currentLine != "" {
					currentLine += " " + word
				} else {
					currentLine = word
				}
			}
		}
		if currentLine != "" {
			fmt.Println(indentStr + currentLine)
		}
	}
}

// Close 关闭屏幕
func (s *Screen) Close() {
	close(s.quitChan)
	fmt.Print("\033[H\033[J") // 清屏
}

// SimpleProgress 简化的进度显示（用于非交互模式）
func SimpleProgress(step, maxSteps, tokens int, elapsed time.Duration) string {
	status := fmt.Sprintf("\r%s Thinking… (%s · step %d/%d",
		color.CyanString("✻"),
		formatDuration(elapsed),
		step,
		maxSteps)

	if tokens > 0 {
		status += fmt.Sprintf(" · ↓ %s tokens", formatNumber(tokens))
	}
	status += ")"

	return status
}
