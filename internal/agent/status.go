package agent

import (
	"fmt"
	"time"

	"github.com/fatih/color"
)

// StatusType 状态类型
type StatusType string

const (
	StatusThinking      StatusType = "thinking"       // 思考中
	StatusReading       StatusType = "reading"        // 读取文件
	StatusWriting       StatusType = "writing"        // 写入文件
	StatusBashing       StatusType = "bashing"        // 执行bash命令
	StatusInstalling    StatusType = "installing"     // 安装软件
	StatusDownloading   StatusType = "downloading"    // 下载文件
	StatusSearching     StatusType = "searching"      // 搜索
	StatusAnalyzing     StatusType = "analyzing"      // 分析
	StatusDiagnosing    StatusType = "diagnosing"     // 诊断
	StatusDeploying     StatusType = "deploying"      // 部署
	StatusConfiguring   StatusType = "configuring"    // 配置
	StatusVerifying     StatusType = "verifying"      // 验证
	StatusCompacting    StatusType = "compacting"     // 压缩对话
	StatusReticulating  StatusType = "reticulating"   // 模拟网络(幽默状态)
	StatusHerding       StatusType = "herding"        // 整理中(幽默状态)
	StatusMeandering    StatusType = "meandering"     // 漫游思考(Claude CLI特有)
	StatusDoing         StatusType = "doing"          // 执行中(通用状态)
	StatusEditing       StatusType = "editing"        // 编辑文件
	StatusBuilding      StatusType = "building"       // 构建项目
	StatusTesting       StatusType = "testing"        // 运行测试
	StatusDebugging     StatusType = "debugging"      // 调试
	StatusOptimizing    StatusType = "optimizing"     // 优化
	StatusRefactoring   StatusType = "refactoring"    // 重构代码
	StatusExploring     StatusType = "exploring"      // 探索代码库
	StatusPlanning      StatusType = "planning"       // 规划任务
	StatusReviewing     StatusType = "reviewing"      // 审查代码
)

// StatusDisplay 状态显示器
type StatusDisplay struct {
	currentStatus   StatusType
	startTime       time.Time
	totalTokens     int
	step            int
	maxSteps        int
	contextPercent  int    // 上下文使用百分比
	showContext     bool   // 是否显示上下文信息
}

// NewStatusDisplay 创建状态显示器
func NewStatusDisplay(maxSteps int) *StatusDisplay {
	return &StatusDisplay{
		currentStatus:  StatusThinking,
		startTime:      time.Now(),
		maxSteps:       maxSteps,
		showContext:    true,
		contextPercent: 100, // 初始100%可用
	}
}

// getStatusIcon 获取状态图标和显示文本
func (s *StatusDisplay) getStatusIcon(status StatusType) (string, string) {
	cyan := color.New(color.FgCyan).Sprint
	magenta := color.New(color.FgMagenta).Sprint
	green := color.New(color.FgGreen).Sprint
	yellow := color.New(color.FgYellow).Sprint
	blue := color.New(color.FgBlue).Sprint

	switch status {
	case StatusThinking:
		return cyan("✻"), "Thinking"
	case StatusReading:
		return magenta("●"), "Reading"
	case StatusWriting:
		return magenta("●"), "Writing"
	case StatusBashing:
		return magenta("●"), "Bash"
	case StatusInstalling:
		return yellow("⚙"), "Installing"
	case StatusDownloading:
		return blue("↓"), "Downloading"
	case StatusSearching:
		return cyan("⌕"), "Searching"
	case StatusAnalyzing:
		return green("⌬"), "Analyzing"
	case StatusDiagnosing:
		return yellow("⚕"), "Diagnosing"
	case StatusDeploying:
		return green("🚀"), "Deploying"
	case StatusConfiguring:
		return yellow("⚙"), "Configuring"
	case StatusVerifying:
		return green("✓"), "Verifying"
	case StatusCompacting:
		return cyan("✻"), "Compacting conversation"
	case StatusReticulating:
		return cyan("◌"), "Reticulating"
	case StatusHerding:
		return cyan("◌"), "Herding"
	case StatusMeandering:
		return cyan("✻"), "Meandering"
	case StatusDoing:
		return cyan("✻"), "Doing"
	case StatusEditing:
		return magenta("●"), "Editing"
	case StatusBuilding:
		return yellow("⚙"), "Building"
	case StatusTesting:
		return green("⌬"), "Testing"
	case StatusDebugging:
		return yellow("⚕"), "Debugging"
	case StatusOptimizing:
		return green("⚡"), "Optimizing"
	case StatusRefactoring:
		return blue("♻"), "Refactoring"
	case StatusExploring:
		return cyan("⌕"), "Exploring"
	case StatusPlanning:
		return cyan("✻"), "Planning"
	case StatusReviewing:
		return green("👁"), "Reviewing"
	default:
		return cyan("●"), "Working"
	}
}

// getToolStatus 根据工具名称返回对应的状态类型
func getToolStatus(toolName string) StatusType {
	switch toolName {
	case "cat", "grep", "ls", "head", "tail", "tree", "file_read":
		return StatusReading
	case "file_write", "file_edit", "mkdir", "touch", "chmod", "chown":
		return StatusWriting
	case "bash":
		return StatusBashing
	case "install":
		return StatusInstalling
	case "wget", "curl":
		return StatusDownloading
	case "code_search":
		return StatusSearching
	case "ps", "top", "free", "df", "du":
		return StatusAnalyzing
	case "kubectl_describe", "kubectl_logs":
		return StatusDiagnosing
	case "kubectl_get":
		return StatusSearching
	default:
		return StatusThinking
	}
}

// Show 显示当前状态
func (s *StatusDisplay) Show() {
	icon, text := s.getStatusIcon(s.currentStatus)
	elapsed := time.Since(s.startTime)

	progressMsg := fmt.Sprintf("\r%s %s…", icon, text)

	// 添加时间和步骤信息
	if s.step > 0 {
		progressMsg += fmt.Sprintf(" (%s · step %d",
			formatDuration(elapsed),
			s.step)

		if s.totalTokens > 0 {
			progressMsg += fmt.Sprintf(" · ↓ %s tokens", formatNumber(s.totalTokens))
		}
		progressMsg += ")"
	}

	fmt.Print(progressMsg)
}

// Clear 清除当前行
func (s *StatusDisplay) Clear() {
	fmt.Print("\r\033[K")
}

// UpdateStatus 更新状态
func (s *StatusDisplay) UpdateStatus(status StatusType) {
	s.currentStatus = status
	s.Show()
}

// UpdateStep 更新步骤
func (s *StatusDisplay) UpdateStep(step int, tokens int) {
	s.step = step
	s.totalTokens = tokens
	s.Show()
}

// UpdateForTool 根据工具更新状态
func (s *StatusDisplay) UpdateForTool(toolName string) {
	status := getToolStatus(toolName)
	s.UpdateStatus(status)
}

// ShowToolCall 显示工具调用（简化风格）
func (s *StatusDisplay) ShowToolCall(toolName string) {
	s.Clear()

	// 根据工具类型选择颜色
	var toolColor *color.Color
	status := getToolStatus(toolName)

	switch status {
	case StatusReading:
		toolColor = color.New(color.FgCyan)
	case StatusWriting:
		toolColor = color.New(color.FgGreen)
	case StatusBashing:
		toolColor = color.New(color.FgYellow)
	case StatusInstalling:
		toolColor = color.New(color.FgMagenta)
	case StatusDownloading:
		toolColor = color.New(color.FgBlue)
	default:
		toolColor = color.New(color.FgMagenta)
	}

	icon, _ := s.getStatusIcon(status)
	fmt.Printf("  %s %s\n", icon, toolColor.Sprint(toolName))
}

// ShowWarning 显示警告信息
func (s *StatusDisplay) ShowWarning(msg string) {
	s.Clear()
	color.New(color.FgYellow).Println("⚠️  " + msg)
}

// ShowError 显示错误信息
func (s *StatusDisplay) ShowError(msg string) {
	s.Clear()
	color.New(color.FgRed).Println("❌ " + msg)
}

// ShowSuccess 显示成功信息
func (s *StatusDisplay) ShowSuccess(msg string) {
	s.Clear()
	color.New(color.FgGreen).Println("✅ " + msg)
}

// ShowInfo 显示信息
func (s *StatusDisplay) ShowInfo(msg string) {
	s.Clear()
	color.New(color.FgCyan).Println("ℹ️  " + msg)
}

// UpdateContextPercent 更新上下文使用百分比
func (s *StatusDisplay) UpdateContextPercent(currentTokens, maxTokens int) {
	if maxTokens > 0 {
		usedPercent := (currentTokens * 100) / maxTokens
		s.contextPercent = 100 - usedPercent // 剩余百分比
		if s.contextPercent < 0 {
			s.contextPercent = 0
		}
	}
}

// ShowBottomStatus 显示底部状态栏(类似 Claude CLI)
func (s *StatusDisplay) ShowBottomStatus() {
	if !s.showContext {
		return
	}

	// 底部状态栏
	gray := color.New(color.FgHiBlack).Sprint

	statusLine := fmt.Sprintf("\n%s Context left until auto-compact: %d%%",
		gray("⏵⏵"),
		s.contextPercent)

	fmt.Print(statusLine)
}

// HideBottomStatus 隐藏底部状态栏
func (s *StatusDisplay) HideBottomStatus() {
	// 清除底部状态行
	fmt.Print("\r\033[K\033[1A\r\033[K") // 清除当前行和上一行
}
