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
)
