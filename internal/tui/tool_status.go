package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// ToolStatus 工具调用状态提示
type ToolStatus struct {
	mu            sync.RWMutex
	currentTool   string        // 当前工具名称
	toolStart     time.Time     // 工具开始时间
	isRunning     bool          // 是否正在运行
	spinner       []string      // 动画帧
	frameIndex    int           // 当前帧索引
	stopChan      chan struct{} // 停止信号
	displayMode   string        // 显示模式: simple, detailed
	customMessage string        // 自定义消息
}

// NewToolStatus 创建工具状态提示
func NewToolStatus() *ToolStatus {
	return &ToolStatus{
		spinner: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stopChan:    make(chan struct{}),
		displayMode: "detailed", // 默认详细模式
	}
}

// Start 开始工具调用提示
func (ts *ToolStatus) Start(toolName string) {
	ts.mu.Lock()
	ts.currentTool = toolName
	ts.toolStart = time.Now()
	ts.isRunning = true
	ts.frameIndex = 0
	ts.customMessage = ""
	ts.mu.Unlock()

	// 启动动画
	go ts.animate()
}

// StartWithMessage 使用自定义消息开始
func (ts *ToolStatus) StartWithMessage(toolName, message string) {
	ts.mu.Lock()
	ts.currentTool = toolName
	ts.toolStart = time.Now()
	ts.isRunning = true
	ts.frameIndex = 0
	ts.customMessage = message
	ts.mu.Unlock()

	go ts.animate()
}

// Stop 停止工具调用提示
func (ts *ToolStatus) Stop(success bool, message string) {
	ts.mu.Lock()
	if !ts.isRunning {
		ts.mu.Unlock()
		return
	}
	ts.isRunning = false
	duration := time.Since(ts.toolStart)
	toolName := ts.currentTool
	ts.mu.Unlock()

	// 停止动画
	select {
	case ts.stopChan <- struct{}{}:
	default:
	}

	// 显示最终状态
	ts.renderFinalStatus(toolName, duration, success, message)
}

// animate 动画循环
func (ts *ToolStatus) animate() {
	ticker := time.NewTicker(80 * time.Millisecond) // 更快的动画帧率
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.mu.RLock()
			if !ts.isRunning {
				ts.mu.RUnlock()
				return
			}

			toolName := ts.currentTool
			elapsed := time.Since(ts.toolStart)
			frame := ts.spinner[ts.frameIndex%len(ts.spinner)]
			customMsg := ts.customMessage
			ts.mu.RUnlock()

			// 渲染当前状态
			ts.renderRunningStatus(toolName, elapsed, frame, customMsg)

			ts.mu.Lock()
			ts.frameIndex++
			ts.mu.Unlock()

		case <-ts.stopChan:
			return
		}
	}
}

// renderRunningStatus 渲染运行中的状态
func (ts *ToolStatus) renderRunningStatus(toolName string, elapsed time.Duration, frame, customMsg string) {
	// 清除当前行
	fmt.Print("\r\033[K")

	// 工具图标映射
	icon := ts.getToolIcon(toolName)

	// 构建状态字符串
	statusColor := color.New(color.FgCyan, color.Bold)

	if customMsg != "" {
		// 自定义消息模式
		fmt.Printf("%s %s %s",
			statusColor.Sprint(frame),
			icon,
			color.HiWhiteString(customMsg))
	} else {
		// 标准模式
		displayName := ts.getToolDisplayName(toolName)
		fmt.Printf("%s %s %s",
			statusColor.Sprint(frame),
			icon,
			color.HiWhiteString(displayName))
	}

	// 显示耗时（如果超过1秒）
	if elapsed > time.Second {
		durationStr := formatDuration(elapsed)
		fmt.Printf(" %s", color.HiBlackString("(%s)", durationStr))
	}
}

// renderFinalStatus 渲染最终状态
func (ts *ToolStatus) renderFinalStatus(toolName string, duration time.Duration, success bool, message string) {
	// 清除当前行
	fmt.Print("\r\033[K")

	icon := ts.getToolIcon(toolName)
	displayName := ts.getToolDisplayName(toolName)
	durationStr := formatDuration(duration)

	if success {
		// 成功状态 - 绿色勾
		fmt.Printf("%s %s %s %s",
			color.GreenString("✓"),
			icon,
			color.HiWhiteString(displayName),
			color.HiBlackString("in %s", durationStr))

		if message != "" {
			fmt.Printf(" — %s", color.GreenString(message))
		}
	} else {
		// 失败状态 - 红色叉
		fmt.Printf("%s %s %s %s",
			color.RedString("✗"),
			icon,
			color.HiWhiteString(displayName),
			color.HiBlackString("in %s", durationStr))

		if message != "" {
			fmt.Printf(" — %s", color.RedString(message))
		}
	}

	fmt.Println() // 换行
}

// getToolIcon 获取工具图标
func (ts *ToolStatus) getToolIcon(toolName string) string {
	iconMap := map[string]string{
		// 网络工具
		"curl":       "🌐",
		"wget":       "⬇️",
		"ping":       "📡",
		"telnet":     "🔌",
		"ssh":        "🔐",
		"request":    "🌐",

		// 系统工具
		"bash":       "⚙️",
		"shell":      "⚙️",
		"ps":         "📊",
		"top":        "📈",
		"free":       "💾",
		"df":         "💿",

		// 文件工具
		"cat":        "📄",
		"ls":         "📂",
		"grep":       "🔍",
		"file_read":  "📖",
		"file_write": "✏️",
		"file_edit":  "✏️",

		// Kubernetes
		"kubectl":         "☸️",
		"kubectl_get":     "☸️",
		"kubectl_logs":    "☸️",
		"kubectl_describe":"☸️",
		"kubectl_delete":  "☸️",

		// 容器
		"docker":     "🐳",

		// 数据库
		"redis":      "🔴",
		"mysql":      "🐬",
		"postgres":   "🐘",

		// 代码工具
		"git_status": "📝",
		"git_diff":   "📝",
		"code_search":"🔎",

		// 监控
		"sys_monitor": "📊",
		"net_monitor": "📡",

		// Web相关
		"web_search":  "🔍",
		"web_fetch":   "🌐",
		"api_call":    "🔌",
	}

	if icon, ok := iconMap[strings.ToLower(toolName)]; ok {
		return icon
	}
	return "●" // 默认圆点
}

// getToolDisplayName 获取工具显示名称
func (ts *ToolStatus) getToolDisplayName(toolName string) string {
	nameMap := map[string]string{
		// 网络工具
		"curl":    "HTTP Request",
		"wget":    "Download",
		"ping":    "Ping",
		"telnet":  "Telnet",
		"ssh":     "SSH",
		"request": "Web Request",

		// 系统工具
		"bash":  "Execute Command",
		"shell": "Shell",
		"ps":    "Process List",
		"top":   "System Monitor",
		"free":  "Memory Status",
		"df":    "Disk Usage",

		// 文件工具
		"cat":        "Read File",
		"ls":         "List Directory",
		"grep":       "Search",
		"file_read":  "Read",
		"file_write": "Write",
		"file_edit":  "Edit",

		// Kubernetes
		"kubectl":          "Kubernetes",
		"kubectl_get":      "Get Resource",
		"kubectl_logs":     "Get Logs",
		"kubectl_describe": "Describe Resource",
		"kubectl_delete":   "Delete Resource",

		// 容器
		"docker": "Docker",

		// 数据库
		"redis":    "Redis",
		"mysql":    "MySQL",
		"postgres": "PostgreSQL",

		// 代码工具
		"git_status": "Git Status",
		"git_diff":   "Git Diff",
		"code_search": "Code Search",

		// 监控
		"sys_monitor": "System Monitor",
		"net_monitor": "Network Monitor",

		// Web相关
		"web_search": "Web Search",
		"web_fetch":  "Fetch URL",
		"api_call":   "API Call",
	}

	if name, ok := nameMap[strings.ToLower(toolName)]; ok {
		return name
	}

	// 默认：首字母大写
	return strings.Title(strings.ReplaceAll(toolName, "_", " "))
}

// SetDisplayMode 设置显示模式
func (ts *ToolStatus) SetDisplayMode(mode string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.displayMode = mode
}

// IsRunning 检查是否正在运行
func (ts *ToolStatus) IsRunning() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.isRunning
}

// GetCurrentTool 获取当前工具
func (ts *ToolStatus) GetCurrentTool() string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.currentTool
}

// SimpleToolStatus 简单的工具状态提示（一次性显示）
func SimpleToolStatus(toolName string, message string) {
	ts := NewToolStatus()
	icon := ts.getToolIcon(toolName)
	displayName := ts.getToolDisplayName(toolName)

	if message != "" {
		fmt.Printf("%s %s %s — %s\n",
			color.CyanString("●"),
			icon,
			color.HiWhiteString(displayName),
			message)
	} else {
		fmt.Printf("%s %s %s\n",
			color.CyanString("●"),
			icon,
			color.HiWhiteString(displayName))
	}
}

// ToolStatusSuccess 显示工具成功状态（一次性）
func ToolStatusSuccess(toolName string, duration time.Duration, message string) {
	ts := NewToolStatus()
	icon := ts.getToolIcon(toolName)
	displayName := ts.getToolDisplayName(toolName)
	durationStr := formatDuration(duration)

	if message != "" {
		fmt.Printf("%s %s %s %s — %s\n",
			color.GreenString("✓"),
			icon,
			color.HiWhiteString(displayName),
			color.HiBlackString("in %s", durationStr),
			color.GreenString(message))
	} else {
		fmt.Printf("%s %s %s %s\n",
			color.GreenString("✓"),
			icon,
			color.HiWhiteString(displayName),
			color.HiBlackString("in %s", durationStr))
	}
}

// ToolStatusError 显示工具错误状态（一次性）
func ToolStatusError(toolName string, duration time.Duration, message string) {
	ts := NewToolStatus()
	icon := ts.getToolIcon(toolName)
	displayName := ts.getToolDisplayName(toolName)
	durationStr := formatDuration(duration)

	fmt.Printf("%s %s %s %s — %s\n",
		color.RedString("✗"),
		icon,
		color.HiWhiteString(displayName),
		color.HiBlackString("in %s", durationStr),
		color.RedString(message))
}
