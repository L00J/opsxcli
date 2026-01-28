package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// MarkdownRenderer Markdown 渲染器
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
}

// NewMarkdownRenderer 创建 Markdown 渲染器
func NewMarkdownRenderer() (*MarkdownRenderer, error) {
	// 创建暗色主题渲染器(适配大多数终端)
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return nil, err
	}

	return &MarkdownRenderer{
		renderer: renderer,
	}, nil
}

// Render 渲染 Markdown 文本
func (mr *MarkdownRenderer) Render(markdown string) (string, error) {
	// 如果文本不包含 Markdown 标记,直接返回
	if !containsMarkdown(markdown) {
		return markdown, nil
	}

	// 渲染 Markdown
	rendered, err := mr.renderer.Render(markdown)
	if err != nil {
		// 渲染失败,返回原始文本
		return markdown, err
	}

	return rendered, nil
}

// RenderSimple 简单渲染(静态方法,无需创建实例)
func RenderSimple(markdown string) string {
	renderer, err := NewMarkdownRenderer()
	if err != nil {
		return markdown
	}

	rendered, err := renderer.Render(markdown)
	if err != nil {
		return markdown
	}

	return rendered
}

// containsMarkdown 检测文本是否包含 Markdown 标记
func containsMarkdown(text string) bool {
	// 常见的 Markdown 标记
	markers := []string{
		"##",     // 标题
		"**",     // 粗体
		"__",     // 粗体/斜体
		"```",    // 代码块
		"- ",     // 列表
		"* ",     // 列表
		"1. ",    // 有序列表
		"[",      // 链接
		"|",      // 表格
		"> ",     // 引用
	}

	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}

	return false
}

// FormatTitle 格式化标题
func FormatTitle(title string) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginTop(1).
		MarginBottom(1)

	return style.Render(title)
}

// FormatSuccess 格式化成功消息
func FormatSuccess(message string) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("42"))

	return style.Render("✓ " + message)
}

// FormatError 格式化错误消息
func FormatError(message string) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196"))

	return style.Render("✗ " + message)
}

// FormatWarning 格式化警告消息
func FormatWarning(message string) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226"))

	return style.Render("⚠ " + message)
}

// FormatInfo 格式化信息消息
func FormatInfo(message string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))

	return style.Render("ℹ " + message)
}
