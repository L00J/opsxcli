package net

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"opsxcli/internal/ui"
)

// drawHeader 绘制标题栏(增强版,显示实时时间)
func drawHeader(screen tcell.Screen, width int) {
	style := tcell.StyleDefault.
		Foreground(ui.ColorText).
		Background(ui.ColorSecondary).
		Bold(true)

	// 绘制整行背景
	for i := 0; i < width; i++ {
		screen.SetContent(i, 0, ' ', nil, style)
	}

	// 左侧标题
	title := " 🌐 OpsX 网络流量监控 "
	drawTextWithStyle(screen, 0, 0, title, style)

	// 中间显示当前时间
	now := time.Now().Format("2006-01-02 15:04:05")
	timeStr := fmt.Sprintf("📅 %s", now)
	timeX := (width - runeWidth(timeStr)) / 2
	if timeX > runeWidth(title) {
		drawTextWithStyle(screen, timeX, 0, timeStr, style)
	}

	// 右侧帮助
	help := " [Tab]切换 [Q]退出 "
	helpX := width - runeWidth(help)
	if helpX > 0 {
		drawTextWithStyle(screen, helpX, 0, help, style)
	}
}

// drawTabs 绘制标签栏（优化：使用完整中文名称）
func drawTabs(screen tcell.Screen, currentTab TabType, width int) {
	tabs := []struct {
		name string
		tab  TabType
	}{
		{"流量", TabRealtime},
		{"连接", TabConnections},
		{"统计", TabStatistics},
	}

	y := 2
	x := 2
	for _, tab := range tabs {
		// 判断是否为当前选中的标签
		isCurrent := (currentTab == tab.tab)

		// 设置样式
		style := tcell.StyleDefault.Foreground(ui.ColorMuted)
		if isCurrent {
			style = style.Foreground(ui.ColorAccent).Background(ui.ColorSecondary).Bold(true)
		}

		// 格式化标签名称
		var name string
		if isCurrent {
			name = fmt.Sprintf("[ %s ]", tab.name)
		} else {
			name = fmt.Sprintf("  %s  ", tab.name)
		}

		// 检查是否有足够空间(使用正确的宽度计算)
		nameWidth := runeWidth(name)
		if x+nameWidth > width-2 {
			break
		}

		// 逐个字符绘制
		col := x
		for _, ch := range name {
			if col >= width-2 {
				break
			}
			screen.SetContent(col, y, ch, nil, style)
			col += runewidth.RuneWidth(ch)
		}
		x = col + 1 // 标签之间的间距
	}
}

// drawFooter 绘制底部帮助
func drawFooter(screen tcell.Screen, width, height int, currentTab TabType) {
	y := height - 1
	help := " [←→/Tab]切换标签  [↑↓]选择接口  [Q/ESC/Ctrl+C]退出  [R]刷新 "
	drawText(screen, (width-runeWidth(help))/2, y, help, ui.ColorMuted)
}
