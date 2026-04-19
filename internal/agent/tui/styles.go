package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7DC4E4")).
			MarginLeft(2)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E738D")).
			MarginLeft(2)

	viewportStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5B6078")).
			Padding(1)

	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5B6078"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6DA95"))

	userStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#8BD5CA"))

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CAD3F5"))

	toolStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#F5BDE6"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ED8796"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EED49F")).
			Background(lipgloss.Color("#363A4F"))

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CAD3F5"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E738D"))

	// === 新增样式 ===

	// 消息气泡
	assistantBubbleStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.Border{Left: "┃"}).
				BorderForeground(lipgloss.Color("#585b70")).
				PaddingLeft(1).
				MarginLeft(1)

	// 代码块
	codeBlockStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#1e1e2e")).
				Foreground(lipgloss.Color("#cdd6f4")).
				Padding(1, 2).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#585b70"))

	codeBlockLangStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#585b70")).
				Foreground(lipgloss.Color("#cdd6f4")).
				Padding(0, 1).
				Bold(true).
				MarginBottom(1)

	inlineCodeStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#313244")).
				Foreground(lipgloss.Color("#f5c2e7"))

	// Markdown
	h1Style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f38ba8"))
	h2Style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fab387"))
	h3Style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f9e2af"))
	boldStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cdd6f4"))
	quoteStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.Border{Left: "┃"}).
			BorderForeground(lipgloss.Color("#6c7086")).
			PaddingLeft(1).
			Foreground(lipgloss.Color("#a6adc8"))
	listStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")).
			MarginTop(1).
			MarginBottom(1)

	// Bash 高亮
	bashPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa"))
	bashCommandStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))
	bashCommentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	bashStringStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))
	bashKeywordStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))
	bashNumberStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#fab387"))

	// 其他
	placeholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6E738D")).
				Italic(true)

	tokenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E738D"))

	// Footer Powerline 样式
	footerModelStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#5B6078")).
				Foreground(lipgloss.Color("#CAD3F5")).
				Padding(0, 1)

	footerTokenStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#363A4F")).
				Foreground(lipgloss.Color("#A6DA95")).
				Padding(0, 1)

	footerTokenWarnStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#363A4F")).
				Foreground(lipgloss.Color("#F5A97F")).
				Padding(0, 1)

	footerTokenDangerStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#363A4F")).
				Foreground(lipgloss.Color("#ED8796")).
				Padding(0, 1)

	footerStatusStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#1E2030")).
				Foreground(lipgloss.Color("#8BD5CA")).
				Padding(0, 1)

	// === 弹窗样式 ===

	modalOverlayStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#000000"))

	modalBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#F5BDE6")).
			Background(lipgloss.Color("#1e1e2e")).
			Padding(2, 3).
			Width(60)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ED8796")).
			MarginBottom(1)

	modalContentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CAD3F5")).
			MarginBottom(1)

	modalButtonStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A6DA95"))

	modalRiskSafeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#A6DA95"))

	modalRiskLowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#EED49F"))

	modalRiskMediumStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#F5BDE6"))

	modalRiskHighStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ED8796"))

	modalRiskCriticalStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF0000"))
)
