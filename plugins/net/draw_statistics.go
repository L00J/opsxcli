package net

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"opsxcli/internal/ui"
)

// drawStatistics 绘制流量统计界面(使用统一边框)
func drawStatistics(screen tcell.Screen, data *NetworkData, selectedIf int, width, height int) {
	y := 4

	if len(data.Interfaces) == 0 {
		ui.DrawBox(screen, 2, y, width-4, height-y-2, " 流量统计 ", ui.ColorSecondary)
		drawText(screen, 4, y+2, "⏳ 正在收集网络数据，请稍候...", ui.ColorMuted)
		return
	}

	uptime := time.Since(data.StartTime)
	uptimeStr := formatDuration(uptime)

	// 左侧: 累计统计和质量指标 (占60%宽度)
	leftWidth := width*6/10 - 3
	drawLeftPanel(screen, data, 2, y, leftWidth, height-y-2, uptimeStr)

	// 右侧: 所有接口流量TOP (占40%宽度)
	rightX := leftWidth + 5
	rightWidth := width - rightX - 2
	drawRightPanel(screen, data, rightX, y, rightWidth, height-y-2)
}
