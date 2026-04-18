package sys

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
)

// drawHeader 绘制标题栏（增强版，显示实时时间）
func drawHeader(screen tcell.Screen, width int) {
	style := tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(tcell.ColorDarkBlue).
		Bold(true)

	// 绘制整行背景
	for i := 0; i < width; i++ {
		screen.SetContent(i, 0, ' ', nil, style)
	}

	// 左侧标题
	title := " ⚡ OpsX 系统监控 "
	for i, ch := range title {
		screen.SetContent(i, 0, ch, nil, style)
	}

	// 中间显示当前时间
	now := time.Now().Format("2006-01-02 15:04:05")
	timeStr := fmt.Sprintf("📅 %s", now)
	timeX := (width - len(timeStr)) / 2
	if timeX > len(title) {
		for i, ch := range timeStr {
			screen.SetContent(timeX+i, 0, ch, nil, style)
		}
	}

	// 右侧帮助
	help := " [Tab]切换 [Q]退出 "
	helpX := width - len(help)
	if helpX > 0 {
		for i, ch := range help {
			screen.SetContent(helpX+i, 0, ch, nil, style)
		}
	}
}

// drawTabs 绘制标签栏（优化：使用完整中文名称）
func drawTabs(screen tcell.Screen, currentTab TabType, width int) {
	tabs := []struct {
		name string
		tab  TabType
	}{
		{"总览", TabOverview},
		{"进程", TabProcesses},
		{"CPU", TabCPU},
		{"内存", TabMemory},
		{"磁盘", TabDisk},
	}

	y := 2
	x := 2
	for _, tab := range tabs {
		// 使用完整的标签名称
		name := tab.name
		style := tcell.StyleDefault.Foreground(tcell.ColorGray)
		if currentTab == tab.tab {
			style = style.Foreground(tcell.ColorYellow).Background(tcell.ColorDarkBlue).Bold(true)
			name = fmt.Sprintf("[ %s ]", name)
		} else {
			name = fmt.Sprintf("  %s  ", name)
		}

		// 检查是否有足够空间显示完整标签名称(使用正确的宽度计算)
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
			// 中文字符占2个宽度
			if ch >= 0x4E00 && ch <= 0x9FFF {
				col += 2
			} else {
				col++
			}
		}
		x = col + 1 // 标签之间的间距
	}
}

// drawFooter 绘制底部帮助
func drawFooter(screen tcell.Screen, width, height int) {
	y := height - 1
	help := " [←→]切换标签  [↑↓]选择  [Q/ESC/Ctrl+C]退出  [R]刷新 "
	drawText(screen, (width-len(help))/2, y, help, tcell.ColorGray)
}
