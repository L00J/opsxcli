package ui

import (
	"github.com/gdamore/tcell/v2"
)

// 统一主题配置

// 颜色方案
var (
	ColorPrimary      = tcell.ColorDarkBlue
	ColorSecondary    = tcell.ColorDarkGreen
	ColorAccent       = tcell.ColorYellow
	ColorSuccess      = tcell.ColorGreen
	ColorWarning      = tcell.ColorOrange
	ColorDanger       = tcell.ColorRed
	ColorInfo         = tcell.ColorAqua
	ColorMuted        = tcell.ColorGray
	ColorText         = tcell.ColorWhite
	ColorBackground   = tcell.ColorBlack
)

// 边框字符（使用 Unicode 绘图字符，跨平台兼容）
const (
	BoxTopLeft     = '┌'
	BoxTopRight    = '┐'
	BoxBottomLeft  = '└'
	BoxBottomRight = '┘'
	BoxHorizontal  = '─'
	BoxVertical    = '│'
	BoxTeeLeft     = '├'
	BoxTeeRight    = '┤'
	BoxTeeTop      = '┬'
	BoxTeeBottom   = '┴'
	BoxCross       = '┼'

	// 双线边框
	BoxDoubleHorizontal = '═'
	BoxDoubleVertical   = '║'

	// 其他符号
	ArrowUp    = '▲'
	ArrowDown  = '▼'
	ArrowRight = '►'
	ArrowLeft  = '◄'
	Bullet     = '•'
	Check      = '✓'
	Cross      = '✗'
)

// DrawBox 绘制边框
func DrawBox(screen tcell.Screen, x, y, width, height int, title string, color tcell.Color) {
	style := tcell.StyleDefault.Foreground(color)

	// 顶部
	screen.SetContent(x, y, BoxTopLeft, nil, style)
	for i := 1; i < width-1; i++ {
		screen.SetContent(x+i, y, BoxHorizontal, nil, style)
	}
	screen.SetContent(x+width-1, y, BoxTopRight, nil, style)

	// 标题 - 先用空格清除横线，然后绘制标题
	if title != "" {
		titleStyle := style.Bold(true)
		titleX := x + 2
		// 先清除标题区域的横线
		titleRunes := []rune(title)
		currentX := titleX
		for _, ch := range titleRunes {
			if currentX >= x+width-2 {
				break
			}
			// 绘制字符（tcell会自动处理宽字符）
			screen.SetContent(currentX, y, ch, nil, titleStyle)
			// 计算字符宽度（中文等宽字符占2列）
			w := runeWidth(ch)
			currentX += w
		}
	}

	// 左右边框
	for i := 1; i < height-1; i++ {
		screen.SetContent(x, y+i, BoxVertical, nil, style)
		screen.SetContent(x+width-1, y+i, BoxVertical, nil, style)
	}

	// 底部
	screen.SetContent(x, y+height-1, BoxBottomLeft, nil, style)
	for i := 1; i < width-1; i++ {
		screen.SetContent(x+i, y+height-1, BoxHorizontal, nil, style)
	}
	screen.SetContent(x+width-1, y+height-1, BoxBottomRight, nil, style)
}

// runeWidth 返回字符的显示宽度（简化版，适用于常见CJK字符）
func runeWidth(r rune) int {
	// ASCII字符宽度为1
	if r < 128 {
		return 1
	}
	// CJK字符和全角字符通常宽度为2
	// 这是一个简化实现，涵盖常见的中文、日文、韩文范围
	if (r >= 0x1100 && r <= 0x115F) || // 韩文Hangul Jamo
		(r >= 0x2E80 && r <= 0x9FFF) || // CJK统一表意文字
		(r >= 0xAC00 && r <= 0xD7AF) || // 韩文音节
		(r >= 0xF900 && r <= 0xFAFF) || // CJK兼容表意文字
		(r >= 0xFE10 && r <= 0xFE19) || // 竖排形式
		(r >= 0xFE30 && r <= 0xFE6F) || // CJK兼容形式
		(r >= 0xFF00 && r <= 0xFF60) || // 全角ASCII
		(r >= 0xFFE0 && r <= 0xFFE6) || // 全角符号
		(r >= 0x20000 && r <= 0x2FFFD) || // CJK扩展
		(r >= 0x30000 && r <= 0x3FFFD) { // CJK扩展
		return 2
	}
	return 1
}

// DrawHorizontalLine 绘制水平分隔线
func DrawHorizontalLine(screen tcell.Screen, x, y, width int, color tcell.Color) {
	style := tcell.StyleDefault.Foreground(color)
	screen.SetContent(x, y, BoxTeeLeft, nil, style)
	for i := 1; i < width-1; i++ {
		screen.SetContent(x+i, y, BoxHorizontal, nil, style)
	}
	screen.SetContent(x+width-1, y, BoxTeeRight, nil, style)
}

// DrawDoubleHorizontalLine 绘制双线水平分隔线
func DrawDoubleHorizontalLine(screen tcell.Screen, x, y, width int, color tcell.Color) {
	style := tcell.StyleDefault.Foreground(color)
	for i := 0; i < width; i++ {
		screen.SetContent(x+i, y, BoxDoubleHorizontal, nil, style)
	}
}
