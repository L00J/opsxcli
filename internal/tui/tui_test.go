package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ===== Markdown 测试 =====

func TestContainsMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"空字符串", "", false},
		{"纯文本", "hello world", false},
		{"标题", "## 标题", true},
		{"粗体", "**粗体文本**", true},
		{"代码块", "```go\nfmt.Println()\n```", true},
		{"列表", "- 列表项", true},
		{"星号列表", "* 列表项", true},
		{"有序列表", "1. 第一项", true},
		{"链接", "[文本](url)", true},
		{"表格", "| 列1 | 列2 |", true},
		{"引用", "> 引用内容", true},
		{"下划线粗体", "__粗体__", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsMarkdown(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewMarkdownRenderer(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	assert.NoError(t, err)
	assert.NotNil(t, renderer)
}

func TestMarkdownRenderer_Render_PlainText(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	assert.NoError(t, err)

	// 纯文本不包含 Markdown 标记，应直接返回
	result, err := renderer.Render("plain text")
	assert.NoError(t, err)
	assert.Equal(t, "plain text", result)
}

func TestMarkdownRenderer_Render_MarkdownContent(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	assert.NoError(t, err)

	// 包含 Markdown 标记的内容应被渲染
	md := "**bold text**"
	result, err := renderer.Render(md)
	assert.NoError(t, err)
	// 渲染后的内容应该与原始内容不同（添加了 ANSI 转义码）
	assert.NotEqual(t, md, result)
}

func TestRenderSimple(t *testing.T) {
	// 纯文本
	result := RenderSimple("hello")
	assert.Equal(t, "hello", result)

	// Markdown 文本应被渲染
	result = RenderSimple("**bold**")
	assert.NotEqual(t, "**bold**", result)
}

func TestFormatTitle(t *testing.T) {
	result := FormatTitle("测试标题")
	assert.Contains(t, result, "测试标题")
	assert.True(t, len(result) > len("测试标题")) // 应有 ANSI 转义码
}

func TestFormatSuccess(t *testing.T) {
	result := FormatSuccess("操作成功")
	assert.Contains(t, result, "操作成功")
	assert.Contains(t, result, "✓")
}

func TestFormatError(t *testing.T) {
	result := FormatError("出错了")
	assert.Contains(t, result, "出错了")
	assert.Contains(t, result, "✗")
}

func TestFormatWarning(t *testing.T) {
	result := FormatWarning("警告信息")
	assert.Contains(t, result, "警告信息")
	assert.Contains(t, result, "⚠")
}

func TestFormatInfo(t *testing.T) {
	result := FormatInfo("信息提示")
	assert.Contains(t, result, "信息提示")
	assert.Contains(t, result, "ℹ")
}

// ===== formatDuration 测试 =====

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{"0秒", 0 * time.Second, "0s"},
		{"1秒", 1 * time.Second, "1s"},
		{"30秒", 30 * time.Second, "30s"},
		{"59秒", 59 * time.Second, "59s"},
		{"1分钟", 60 * time.Second, "1m 0s"},
		{"90秒", 90 * time.Second, "1m 30s"},
		{"9分59秒", 599 * time.Second, "9m 59s"},
		{"10分钟", 600 * time.Second, "10m"},
		{"15分钟", 900 * time.Second, "15m"},
		{"1小时", 3600 * time.Second, "60m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ===== formatNumber 测试 =====

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"0", 0, "0"},
		{"个位数", 5, "5"},
		{"两位数", 99, "99"},
		{"三位数", 999, "999"},
		{"1000", 1000, "1.0k"},
		{"1500", 1500, "1.5k"},
		{"9999", 9999, "10.0k"},
		{"999999", 999999, "1000.0k"},
		{"1000000", 1000000, "1.0M"},
		{"2500000", 2500000, "2.5M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ===== ToolStatus 测试 =====

func TestNewToolStatus(t *testing.T) {
	ts := NewToolStatus()
	assert.NotNil(t, ts)
	assert.False(t, ts.IsRunning())
	assert.Equal(t, "", ts.GetCurrentTool())
}

func TestToolStatus_GetToolIcon(t *testing.T) {
	ts := NewToolStatus()

	tests := []struct {
		name     string
		tool     string
		expected string
	}{
		{"curl", "curl", "🌐"},
		{"SSH", "ssh", "🔐"},
		{"bash", "bash", "⚙️"},
		{"docker", "docker", "🐳"},
		{"redis", "redis", "🔴"},
		{"mysql", "mysql", "🐬"},
		{"kubectl", "kubectl", "☸️"},
		{"未知工具", "unknown_tool", "●"}, // 默认图标
		{"大小写不敏感", "DOCKER", "🐳"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := ts.getToolIcon(tt.tool)
			assert.Equal(t, tt.expected, icon)
		})
	}
}

func TestToolStatus_GetToolDisplayName(t *testing.T) {
	ts := NewToolStatus()

	tests := []struct {
		name     string
		tool     string
		expected string
	}{
		{"curl", "curl", "HTTP Request"},
		{"wget", "wget", "Download"},
		{"ssh", "ssh", "SSH"},
		{"bash", "bash", "Execute Command"},
		{"docker", "docker", "Docker"},
		{"redis", "redis", "Redis"},
		{"mysql", "mysql", "MySQL"},
		{"kubectl", "kubectl", "Kubernetes"},
		{"file_read", "file_read", "Read"},
		{"大小写不敏感", "REDIS", "Redis"},
		{"未知工具", "custom_tool", "Custom Tool"}, // Title(ReplaceAll(toolName, "_", " "))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := ts.getToolDisplayName(tt.tool)
			assert.Equal(t, tt.expected, name)
		})
	}
}

func TestToolStatus_StartStop(t *testing.T) {
	ts := NewToolStatus()

	// 启动
	ts.Start("curl")
	assert.True(t, ts.IsRunning())
	assert.Equal(t, "curl", ts.GetCurrentTool())

	// 等一小段时间确保动画 goroutine 启动
	time.Sleep(50 * time.Millisecond)

	// 停止（成功）
	ts.Stop(true, "完成")
	assert.False(t, ts.IsRunning())
}

func TestToolStatus_StopWhenNotRunning(t *testing.T) {
	ts := NewToolStatus()
	// 不应该 panic
	ts.Stop(true, "未运行时停止")
	assert.False(t, ts.IsRunning())
}

func TestToolStatus_SetDisplayMode(t *testing.T) {
	ts := NewToolStatus()
	ts.SetDisplayMode("simple")
	assert.Equal(t, "simple", ts.displayMode)
	ts.SetDisplayMode("detailed")
	assert.Equal(t, "detailed", ts.displayMode)
}

// ===== InputModel 测试 =====

func TestNewInputModel(t *testing.T) {
	model := NewInputModel("请输入:")
	assert.NotNil(t, model)
	assert.Equal(t, "请输入:", model.prompt)
	assert.Equal(t, "", model.GetValue())
	assert.False(t, model.IsSubmitted())
	assert.False(t, model.IsCancelled())
}

func TestDefaultInputStyles(t *testing.T) {
	styles := DefaultInputStyles()
	// 验证所有样式都被初始化
	assert.NotNil(t, styles.Separator)
	assert.NotNil(t, styles.Prompt)
	assert.NotNil(t, styles.Input)
	assert.NotNil(t, styles.Placeholder)
	assert.NotNil(t, styles.Cursor)
}

func TestInputModel_Reset(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune{'h', 'e', 'l', 'l', 'o'}
	model.cursor = 5
	model.submitted = true

	model.Reset()

	assert.Equal(t, "", model.GetValue())
	assert.Equal(t, 0, model.cursor)
	assert.False(t, model.IsSubmitted())
	assert.False(t, model.IsCancelled())
}

func TestInputModel_SetEnabled(t *testing.T) {
	model := NewInputModel("测试")
	assert.True(t, model.enabled)

	model.SetEnabled(false)
	assert.False(t, model.enabled)

	model.SetEnabled(true)
	assert.True(t, model.enabled)
}

// ===== ConfirmDialog 测试 =====

func TestNewConfirmDialog(t *testing.T) {
	dialog := NewConfirmDialog("确认执行?", "rm -rf /", "删除所有文件")
	assert.NotNil(t, dialog)
	assert.Equal(t, "确认执行?", dialog.Title)
	assert.Equal(t, "rm -rf /", dialog.Command)
	assert.Equal(t, "删除所有文件", dialog.CommandDesc)
	assert.Equal(t, 0, dialog.Selected)
	assert.Len(t, dialog.Options, 3)
}

func TestConfirmDialog_Options(t *testing.T) {
	dialog := NewConfirmDialog("确认?", "ls", "列出文件")

	// 验证三个选项
	assert.Equal(t, "yes", dialog.Options[0].Value)
	assert.Equal(t, "yes_remember", dialog.Options[1].Value)
	assert.Equal(t, "custom", dialog.Options[2].Value)

	// 验证选项标签
	assert.Equal(t, "Yes", dialog.Options[0].Label)
	assert.Contains(t, dialog.Options[1].Label, "allow similar")
	assert.Contains(t, dialog.Options[2].Label, "Type here")
}

// ===== Screen 测试 =====

func TestNewScreen(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	assert.NotNil(t, screen)
	assert.Equal(t, "⏵ ", screen.inputPrompt)
	assert.NotNil(t, screen.messages)
	assert.NotNil(t, screen.inputChan)
	assert.NotNil(t, screen.quitChan)

	// 关闭
	screen.Close()
}

func TestScreen_AddMessage(t *testing.T) {
	screen, err := NewScreen()
	assert.NoError(t, err)
	defer screen.Close()

	screen.AddMessage("user", "hello")
	screen.AddMessage("assistant", "world")

	assert.Len(t, screen.messages, 2)
	assert.Equal(t, "user", screen.messages[0].Role)
	assert.Equal(t, "hello", screen.messages[0].Content)
	assert.Equal(t, "assistant", screen.messages[1].Role)
	assert.Equal(t, "world", screen.messages[1].Content)
}

// ===== SimpleProgress 测试 =====

func TestSimpleProgress(t *testing.T) {
	result := SimpleProgress(2, 10, 500, 3*time.Second)
	assert.Contains(t, result, "Thinking")
	assert.Contains(t, result, "step 2/10")
	assert.Contains(t, result, "3s")
	assert.Contains(t, result, "500 tokens")
}

func TestSimpleProgress_NoTokens(t *testing.T) {
	result := SimpleProgress(1, 5, 0, 1*time.Second)
	assert.Contains(t, result, "Thinking")
	assert.Contains(t, result, "step 1/5")
	assert.NotContains(t, result, "tokens")
}

// ===== Session 测试 =====

func TestSession_Struct(t *testing.T) {
	// 验证 Message 结构
	msg := Message{
		Role:    "user",
		Content: "测试消息",
		Time:    time.Now(),
	}
	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "测试消息", msg.Content)
	assert.False(t, msg.Time.IsZero())
}

// ===== 集成测试: Markdown 与 ToolStatus =====

func TestMarkdownAndToolStatus_Integration(t *testing.T) {
	// 测试 Markdown 渲染器与工具状态的组合使用
	renderer, err := NewMarkdownRenderer()
	assert.NoError(t, err)

	ts := NewToolStatus()
	icon := ts.getToolIcon("redis")
	name := ts.getToolDisplayName("redis")

	mdContent := "**工具状态**: " + icon + " " + name + " 完成"
	result, err := renderer.Render(mdContent)
	assert.NoError(t, err)
	assert.Contains(t, result, "完成")
}

// ===== 字符串处理辅助测试 =====

func TestStringsContainInToolNames(t *testing.T) {
	ts := NewToolStatus()

	// 确保所有图标映射键都是小写的
	testTools := []string{"curl", "wget", "ping", "telnet", "ssh", "bash", "docker", "redis", "mysql", "kubectl"}
	for _, tool := range testTools {
		icon := ts.getToolIcon(strings.ToUpper(tool))
		assert.NotEqual(t, "●", icon, "工具 %s 的大小写映射应正常工作", tool)
	}
}

func TestToolStatus_ToolNameNormalization(t *testing.T) {
	ts := NewToolStatus()

	// 测试下划线转空格 + Title
	name := ts.getToolDisplayName("my_custom_tool")
	assert.Equal(t, "My Custom Tool", name)
}
