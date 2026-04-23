package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputModel 交互式输入框模型
type InputModel struct {
	prompt      string      // 提示文本
	value       []rune      // 当前输入值(使用rune支持中文)
	cursor      int         // 光标位置(rune索引)
	width       int         // 输入框宽度
	placeholder string      // 占位符
	submitted   bool        // 是否已提交
	cancelled   bool        // 是否已取消
	enabled     bool        // 是否启用
	styles      InputStyles // 样式
}

// InputStyles 输入框样式
type InputStyles struct {
	Separator   lipgloss.Style
	Prompt      lipgloss.Style
	Input       lipgloss.Style
	Placeholder lipgloss.Style
	Cursor      lipgloss.Style
}

// NewInputModel 创建输入框模型
func NewInputModel(prompt string) *InputModel {
	return &InputModel{
		prompt:      prompt,
		value:       []rune{},
		cursor:      0,
		width:       80,
		placeholder: "输入消息... (Enter提交, Esc取消, Shift+Tab切换)",
		enabled:     true,
		styles:      DefaultInputStyles(),
	}
}

// DefaultInputStyles 默认样式
func DefaultInputStyles() InputStyles {
	return InputStyles{
		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Bold(true),
		Prompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),
		Input: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		Placeholder: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true),
		Cursor: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Reverse(true),
	}
}

// Init 初始化
func (m *InputModel) Init() tea.Cmd {
	return nil
}

// Update 更新模型
func (m *InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.enabled {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			if len(m.value) > 0 {
				m.submitted = true
				return m, tea.Quit
			}

		case "esc":
			m.cancelled = true
			return m, tea.Quit

		case "left":
			if m.cursor > 0 {
				m.cursor--
			}

		case "right":
			if m.cursor < len(m.value) {
				m.cursor++
			}

		case "home":
			m.cursor = 0

		case "end":
			m.cursor = len(m.value)

		case "backspace":
			if m.cursor > 0 {
				m.value = append(m.value[:m.cursor-1], m.value[m.cursor:]...)
				m.cursor--
			}

		case "delete":
			if m.cursor < len(m.value) {
				m.value = append(m.value[:m.cursor], m.value[m.cursor+1:]...)
			}

		default:
			// 处理所有可打印字符(包括中文)
			// bubbletea 的 KeyMsg.String() 已经返回了完整的 UTF-8 字符
			runes := []rune(msg.String())
			if len(runes) == 1 && utf8.ValidRune(runes[0]) {
				// 在光标位置插入字符
				m.value = append(m.value[:m.cursor], append(runes, m.value[m.cursor:]...)...)
				m.cursor++
			}
		}
	}

	return m, nil
}

// View 渲染视图
func (m *InputModel) View() string {
	if !m.enabled {
		return ""
	}

	var b strings.Builder

	// 简洁的提示符,不需要复杂的分隔线
	b.WriteString(m.styles.Prompt.Render(m.prompt))
	b.WriteString("  ")

	// 如果没有输入,显示占位符(可选)
	if len(m.value) == 0 {
		// 不显示占位符,保持简洁
		b.WriteString(m.styles.Cursor.Render(" "))
	} else {
		// 显示输入内容和光标
		beforeCursor := string(m.value[:m.cursor])
		atCursor := " "
		afterCursor := ""

		if m.cursor < len(m.value) {
			atCursor = string(m.value[m.cursor])
			if m.cursor+1 < len(m.value) {
				afterCursor = string(m.value[m.cursor+1:])
			}
		}

		b.WriteString(m.styles.Input.Render(beforeCursor))
		b.WriteString(m.styles.Cursor.Render(atCursor))
		b.WriteString(m.styles.Input.Render(afterCursor))
	}

	return b.String()
}

// SetEnabled 设置是否启用
func (m *InputModel) SetEnabled(enabled bool) {
	m.enabled = enabled
}

// GetValue 获取输入值
func (m *InputModel) GetValue() string {
	return string(m.value)
}

// IsSubmitted 是否已提交
func (m *InputModel) IsSubmitted() bool {
	return m.submitted
}

// IsCancelled 是否已取消
func (m *InputModel) IsCancelled() bool {
	return m.cancelled
}

// Reset 重置输入框
func (m *InputModel) Reset() {
	m.value = []rune{}
	m.cursor = 0
	m.submitted = false
	m.cancelled = false
}

// RunInput 运行输入框并获取用户输入
func RunInput(prompt string) (string, error) {
	// 获取终端宽度,默认120字符
	width := 120

	// 打印顶部分隔线
	separator := strings.Repeat("─", width)
	fmt.Println(separator)

	// 打印提示符
	fmt.Print(prompt + "  ")

	// 读取输入
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("读取输入失败: %w", err)
	}

	// 打印底部分隔线
	fmt.Println(separator)
	fmt.Println() // 额外的空行

	return strings.TrimSpace(line), nil
}
