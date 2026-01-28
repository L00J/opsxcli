package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

// InteractiveUI 交互式 UI
type InteractiveUI struct {
	rl            *readline.Instance
	statusLine    string
	statusMutex   sync.RWMutex
	userInputChan chan string
	cancelFunc    context.CancelFunc
	isThinking    bool
	thinkingStart time.Time
	tokens        int
	step          int
	maxSteps      int
}

// NewInteractiveUI 创建交互式 UI
func NewInteractiveUI(prompt string) (*InteractiveUI, error) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          color.GreenString(prompt),
		HistoryFile:     "/tmp/opsxcli_history",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return nil, err
	}

	ui := &InteractiveUI{
		rl:            rl,
		userInputChan: make(chan string, 10),
	}

	return ui, nil
}

// Close 关闭 UI
func (ui *InteractiveUI) Close() {
	if ui.rl != nil {
		ui.rl.Close()
	}
}

// ReadInput 读取用户输入（非阻塞）
func (ui *InteractiveUI) ReadInput() string {
	line, err := ui.rl.Readline()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(line)
}

// StartThinking 开始思考状态
func (ui *InteractiveUI) StartThinking(maxSteps int) {
	ui.statusMutex.Lock()
	defer ui.statusMutex.Unlock()

	ui.isThinking = true
	ui.thinkingStart = time.Now()
	ui.tokens = 0
	ui.step = 0
	ui.maxSteps = maxSteps

	// 启动状态更新 goroutine
	go ui.updateStatus()
}

// UpdateProgress 更新进度
func (ui *InteractiveUI) UpdateProgress(step int, tokens int) {
	ui.statusMutex.Lock()
	defer ui.statusMutex.Unlock()

	ui.step = step
	ui.tokens = tokens
}

// StopThinking 停止思考状态
func (ui *InteractiveUI) StopThinking() {
	ui.statusMutex.Lock()
	defer ui.statusMutex.Unlock()

	ui.isThinking = false
	// 清除状态行
	fmt.Print("\r\033[K")
}

// updateStatus 更新状态显示（私有方法）
func (ui *InteractiveUI) updateStatus() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frameIndex := 0

	for range ticker.C {
		ui.statusMutex.RLock()
		if !ui.isThinking {
			ui.statusMutex.RUnlock()
			return
		}

		elapsed := time.Since(ui.thinkingStart)
		spinner := color.CyanString(spinnerFrames[frameIndex%len(spinnerFrames)])

		statusMsg := fmt.Sprintf("\r%s Thinking… (%s",
			spinner,
			formatDuration(elapsed))

		if ui.step > 0 {
			statusMsg += fmt.Sprintf(" · step %d/%d", ui.step, ui.maxSteps)
		}

		if ui.tokens > 0 {
			statusMsg += fmt.Sprintf(" · ↓ %s tokens", formatNumber(ui.tokens))
		}

		statusMsg += ") " + color.HiBlackString("[type to add context, Ctrl+C to cancel]")

		ui.statusMutex.RUnlock()

		fmt.Print(statusMsg)
		frameIndex++
	}
}

// ShowMessage 显示消息
func (ui *InteractiveUI) ShowMessage(role, content string) {
	// 清除状态行
	fmt.Print("\r\033[K")

	var roleColor *color.Color
	var roleIcon string

	switch role {
	case "user":
		roleColor = color.New(color.FgGreen)
		roleIcon = "👤"
	case "assistant":
		roleColor = color.New(color.FgCyan)
		roleIcon = "🤖"
	case "tool":
		roleColor = color.New(color.FgMagenta)
		roleIcon = "→"
	case "error":
		roleColor = color.New(color.FgRed)
		roleIcon = "❌"
	case "info":
		roleColor = color.New(color.FgYellow)
		roleIcon = "ℹ️"
	default:
		roleColor = color.New(color.FgWhite)
		roleIcon = "·"
	}

	if role == "tool" {
		fmt.Printf("  %s %s\n", roleColor.Sprint(roleIcon), content)
	} else if role != "" {
		fmt.Printf("\n%s %s:\n%s\n", roleIcon, roleColor.Sprint(strings.Title(role)), content)
	} else {
		fmt.Println(content)
	}
}

// Prompt 显示提示并等待输入
func (ui *InteractiveUI) Prompt(prompt string) (string, error) {
	ui.rl.SetPrompt(color.GreenString(prompt))
	return ui.rl.Readline()
}

// formatDuration 格式化时长
func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	secs := seconds % 60
	if minutes < 10 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	return fmt.Sprintf("%dm", minutes)
}

// formatNumber 格式化数字
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}
