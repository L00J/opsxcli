package sys

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"opsxcli/internal/ui"
)

// drawText 绘制文本
func drawText(screen tcell.Screen, x, y int, text string, color tcell.Color) {
	style := tcell.StyleDefault.Foreground(color)
	drawTextWithStyle(screen, x, y, text, style)
}

// drawTextWithStyle 使用指定样式绘制文本
func drawTextWithStyle(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	width, _ := screen.Size()
	col := x
	for _, ch := range text {
		if col >= width {
			break
		}
		screen.SetContent(col, y, ch, nil, style)
		col += runewidth.RuneWidth(ch)
	}
}

// runeWidth 计算字符串的显示宽度
func runeWidth(s string) int {
	return runewidth.StringWidth(s)
}

// drawProgressBar 绘制进度条
func drawProgressBar(screen tcell.Screen, x, y, width int, percent float64, color tcell.Color) {
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}

	filled := int(float64(width) * percent / 100)
	empty := width - filled

	// 绘制填充部分
	fillStyle := tcell.StyleDefault.Foreground(color).Background(color)
	for i := 0; i < filled; i++ {
		screen.SetContent(x+i, y, '█', nil, fillStyle)
	}

	// 绘制空白部分
	emptyStyle := tcell.StyleDefault.Foreground(ui.ColorMuted).Background(ui.ColorMuted)
	for i := 0; i < empty; i++ {
		screen.SetContent(x+filled+i, y, '░', nil, emptyStyle)
	}
}
