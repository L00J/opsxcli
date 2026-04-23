// dashboard.go - TUI 仪表板视图
// v0.6.0: 记忆层状态仪表板 + Skill 库浏览器
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"opsxcli/internal/agent/evolver"
)

// dashboardTab 仪表板标签页
type dashboardTab int

const (
	tabOverview dashboardTab = iota // 概览
	tabSkills                       // 技能库
)

// DashboardData 仪表板数据
type DashboardData struct {
	// 记忆层统计
	ShortTermSessions int
	ShortTermSizeMB   float64
	LongTermItems     int
	LongTermSizeMB    float64
	TotalSizeMB       float64
	TotalLimitMB      float64
	UsagePercent      float64

	// 技能库数据
	Skills     []*evolver.SkillEntry
	SkillCount int
	Categories map[string]int // 分类 → 数量

	// 加载错误
	LoadErr string
}

// loadDashboardCmd 加载仪表板数据
func (m Model) loadDashboardCmd() tea.Cmd {
	return func() tea.Msg {
		data := DashboardData{}

		// 加载记忆层统计
		homeDir, err := os.UserHomeDir()
		if err != nil {
			data.LoadErr = fmt.Sprintf("获取主目录失败: %v", err)
			return dashboardLoadedMsg{data: data}
		}

		// 尝试加载程序层记忆（Skill 库）
		procDir := filepath.Join(homeDir, ".opsxcli", "agent")
		pm, err := evolver.LoadProceduralMemory(procDir)
		if err == nil {
			pm.SeedBuiltinSkills()
			data.Skills = pm.GetAllSkills()
			data.SkillCount = pm.Count()
			data.Categories = make(map[string]int)
			for _, s := range data.Skills {
				data.Categories[s.Category]++
			}
		}

		// 尝试获取记忆层统计（需要 DB 实例，在 TUI 中可能不可用）
		// 如果 MemoryManager 未注入，则只显示技能库数据
		if m.memoryStatsLoader != nil {
			stats, err := m.memoryStatsLoader()
			if err == nil {
				if v, ok := stats["short_term_sessions"].(int); ok {
					data.ShortTermSessions = v
				}
				if v, ok := stats["short_term_size_mb"].(float64); ok {
					data.ShortTermSizeMB = v
				}
				if v, ok := stats["long_term_items"].(int64); ok {
					data.LongTermItems = int(v)
				}
				if v, ok := stats["long_term_size_mb"].(float64); ok {
					data.LongTermSizeMB = v
				}
				if v, ok := stats["total_size_mb"].(float64); ok {
					data.TotalSizeMB = v
				}
				if v, ok := stats["total_limit_mb"].(float64); ok {
					data.TotalLimitMB = v
				}
				if v, ok := stats["usage_percent"].(float64); ok {
					data.UsagePercent = v
				}
			}
		}

		return dashboardLoadedMsg{data: data}
	}
}

// handleDashboardKeys 处理仪表板视图的按键
func (m Model) handleDashboardKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc, tea.KeyCtrlD, tea.KeyCtrlO:
		m.viewState = viewChat
		m.viewport.SetContent(m.renderMessages())
		return m, nil
	case tea.KeyUp:
		if m.dashCursor > 0 {
			m.dashCursor--
		}
		return m, nil
	case tea.KeyDown:
		maxCursor := 0
		if m.dashTab == tabSkills && m.dashboardData.Skills != nil {
			maxCursor = len(m.dashboardData.Skills) - 1
		}
		if m.dashCursor < maxCursor {
			m.dashCursor++
		}
		return m, nil
	}

	switch msg.String() {
	case "1":
		m.dashTab = tabOverview
		m.dashCursor = 0
		return m, nil
	case "2":
		m.dashTab = tabSkills
		m.dashCursor = 0
		return m, nil
	case "r":
		// 刷新数据
		return m, m.loadDashboardCmd()
	case "enter":
		// Tab 切换
		if m.dashTab == tabOverview {
			m.dashTab = tabSkills
		} else {
			m.dashTab = tabOverview
		}
		m.dashCursor = 0
		return m, nil
	}

	return m, nil
}

// renderDashboard 渲染仪表板视图
func (m Model) renderDashboard() string {
	if m.width == 0 {
		return "初始化中..."
	}

	var b strings.Builder
	contentWidth := m.width - 6
	if contentWidth < 30 {
		contentWidth = 30
	}

	// 标题
	b.WriteString(dashTitleStyle.Render("📊 控制面板"))
	b.WriteString("\n\n")

	// 标签页指示器
	tabOverviewLabel := " 1.概览 "
	tabSkillsLabel := " 2.技能库 "
	if m.dashTab == tabOverview {
		tabOverviewLabel = dashTabActiveStyle.Render(tabOverviewLabel)
		tabSkillsLabel = dashTabInactiveStyle.Render(tabSkillsLabel)
	} else {
		tabOverviewLabel = dashTabInactiveStyle.Render(tabOverviewLabel)
		tabSkillsLabel = dashTabActiveStyle.Render(tabSkillsLabel)
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, tabOverviewLabel, tabSkillsLabel))
	b.WriteString("\n")
	b.WriteString(dashDividerStyle.Render(strings.Repeat("─", contentWidth)))
	b.WriteString("\n\n")

	if m.dashboardData.LoadErr != "" {
		b.WriteString(errorStyle.Render("⚠️ 加载错误: " + m.dashboardData.LoadErr))
		b.WriteString("\n\n")
	}

	switch m.dashTab {
	case tabOverview:
		b.WriteString(m.renderDashboardOverview(contentWidth))
	case tabSkills:
		b.WriteString(m.renderDashboardSkills(contentWidth))
	}

	b.WriteString("\n")
	b.WriteString(dashDividerStyle.Render(strings.Repeat("─", contentWidth)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("1/2 切换标签页 | ↑↓ 导航 | Enter 切换标签 | R 刷新 | Ctrl+M/Esc 返回"))
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

// renderDashboardOverview 渲染概览标签页
func (m Model) renderDashboardOverview(width int) string {
	var b strings.Builder
	data := m.dashboardData

	// 记忆层状态卡片
	b.WriteString(dashSectionStyle.Render("🧠 记忆层状态"))
	b.WriteString("\n\n")

	// 使用两列布局
	leftCol := fmt.Sprintf("  短期记忆会话: %d\n  短期记忆大小: %.2f MB", data.ShortTermSessions, data.ShortTermSizeMB)
	rightCol := fmt.Sprintf("  长期记忆条目: %d\n  长期记忆大小: %.2f MB", data.LongTermItems, data.LongTermSizeMB)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left,
		dashCardStyle.Width(width/2-2).Render(leftCol),
		dashCardStyle.Width(width/2-2).Render(rightCol),
	))
	b.WriteString("\n\n")

	// 容量使用条
	usageStr := fmt.Sprintf("%.1f%%", data.UsagePercent)
	if data.UsagePercent == 0 && data.SkillCount > 0 {
		usageStr = "N/A"
	}
	usageLabel := fmt.Sprintf("  总容量: %.2f / %.0f MB (%s)", data.TotalSizeMB, data.TotalLimitMB, usageStr)
	b.WriteString(dashCardStyle.Width(width).Render(usageLabel))
	b.WriteString("\n\n")

	// 使用率进度条
	if data.UsagePercent > 0 {
		barWidth := width - 20
		if barWidth < 10 {
			barWidth = 10
		}
		filled := int(data.UsagePercent / 100 * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		barStyle := dashBarNormalStyle
		if data.UsagePercent > 80 {
			barStyle = dashBarDangerStyle
		} else if data.UsagePercent > 50 {
			barStyle = dashBarWarnStyle
		}
		b.WriteString("  ")
		b.WriteString(barStyle.Render(bar))
		b.WriteString("\n\n")
	}

	// 技能库摘要
	b.WriteString(dashSectionStyle.Render("📚 技能库摘要"))
	b.WriteString("\n\n")

	summaryLine := fmt.Sprintf("  技能总数: %d", data.SkillCount)
	if len(data.Categories) > 0 {
		cats := make([]string, 0, len(data.Categories))
		for cat, count := range data.Categories {
			cats = append(cats, fmt.Sprintf("%s(%d)", cat, count))
		}
		sort.Strings(cats)
		summaryLine += "  |  分类: " + strings.Join(cats, ", ")
	}
	b.WriteString(dashCardStyle.Width(width).Render(summaryLine))
	b.WriteString("\n\n")

	// 快速统计
	b.WriteString(dashSectionStyle.Render("📈 会话统计"))
	b.WriteString("\n\n")
	sessionInfo := fmt.Sprintf("  当前会话: %s", m.currentSessionID)
	if m.currentSessionID == "" {
		sessionInfo = "  当前会话: 无"
	}
	sessionInfo += fmt.Sprintf("\n  本次消息数: %d", len(m.messages))
	sessionInfo += fmt.Sprintf("\n  累计 Token: %d / %d", m.totalTokens, m.maxTokens)
	b.WriteString(dashCardStyle.Width(width).Render(sessionInfo))
	b.WriteString("\n")

	return b.String()
}

// renderDashboardSkills 渲染技能库标签页
func (m Model) renderDashboardSkills(width int) string {
	var b strings.Builder
	data := m.dashboardData

	if len(data.Skills) == 0 {
		b.WriteString(itemStyle.Render("  暂无技能数据"))
		b.WriteString("\n")
		return b.String()
	}

	// 按分类分组显示
	categorized := make(map[string][]*evolver.SkillEntry)
	var categories []string
	for _, s := range data.Skills {
		if _, ok := categorized[s.Category]; !ok {
			categories = append(categories, s.Category)
		}
		categorized[s.Category] = append(categorized[s.Category], s)
	}
	sort.Strings(categories)

	cursor := 0
	for _, cat := range categories {
		catLabel := fmt.Sprintf("📁 %s (%d)", cat, len(categorized[cat]))
		b.WriteString(dashSectionStyle.Render(catLabel))
		b.WriteString("\n")

		for _, skill := range categorized[cat] {
			prefix := "  "
			if cursor == m.dashCursor {
				prefix = "▸ "
			}

			// 成功率指示器
			rateStr := fmt.Sprintf("%.0f%%", skill.SuccessRate*100)
			rateIndicator := ""
			if skill.SuccessRate >= 0.9 {
				rateIndicator = "🟢"
			} else if skill.SuccessRate >= 0.8 {
				rateIndicator = "🟡"
			} else {
				rateIndicator = "🔴"
			}

			// 来源标签
			sourceTag := ""
			switch skill.Source {
			case "seed":
				sourceTag = "[内置]"
			case "learned":
				sourceTag = "[学习]"
			case "manual":
				sourceTag = "[自建]"
			}

			line := fmt.Sprintf("%s%s %s %s (v%d, 使用%d次, 成功率%s) %s",
				prefix, rateIndicator, skill.Name, sourceTag,
				skill.Version, skill.UsageCount, rateStr,
				skill.UpdatedAt.Format("01-02"))

			if cursor == m.dashCursor {
				line = selectedStyle.Render(line)
			} else {
				line = itemStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")

			// 如果选中，显示详细信息
			if cursor == m.dashCursor {
				desc := skill.Description
				if len(desc) > width-8 {
					desc = desc[:width-8] + "..."
				}
				b.WriteString(dashDetailStyle.Render(fmt.Sprintf("    %s", desc)))
				b.WriteString("\n")

				if len(skill.Steps) > 0 {
					stepsStr := strings.Join(skill.Steps, " → ")
					if len(stepsStr) > width-8 {
						stepsStr = stepsStr[:width-8] + "..."
					}
					b.WriteString(dashDetailStyle.Render(fmt.Sprintf("    流程: %s", stepsStr)))
					b.WriteString("\n")
				}

				if len(skill.Triggers) > 0 {
					triggersStr := strings.Join(skill.Triggers, ", ")
					if len(triggersStr) > width-12 {
						triggersStr = triggersStr[:width-12] + "..."
					}
					b.WriteString(dashDetailStyle.Render(fmt.Sprintf("    触发: %s", triggersStr)))
					b.WriteString("\n")
				}
				b.WriteString("\n")
			}

			cursor++
		}
		b.WriteString("\n")
	}

	return b.String()
}

// createDashboardViewport 创建仪表板专用的 viewport
func createDashboardViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height)
	return vp
}
