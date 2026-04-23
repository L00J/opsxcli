package sys

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"opsxcli/internal/ui"
)

// drawMemory 绘制内存详细信息（使用统一边框）
func drawMemory(screen tcell.Screen, data *SystemData, width, height int) {
	y := 4

	if data.MemInfo == nil {
		ui.DrawBox(screen, 2, y, width-4, height-y-2, " 内存详情 ", ui.ColorPrimary)
		drawText(screen, 4, y+2, "⏳ 首次加载中，请稍候 (约2秒)...", ui.ColorMuted)
		return
	}

	// 绘制边框
	ui.DrawBox(screen, 2, y, width-4, height-y-2, " 内存详细信息 ", ui.ColorPrimary)

	// 显示更新时间
	if !data.UpdateTime.IsZero() {
		updateTimeStr := data.UpdateTime.Format("15:04:05")
		timeStr := fmt.Sprintf("更新: %s", updateTimeStr)
		drawText(screen, width-18, y, timeStr, ui.ColorMuted)
	}

	contentY := y + 2

	// 计算内存值（GB）
	totalGB := float64(data.MemInfo.Total) / 1024 / 1024 / 1024
	availableGB := float64(data.MemInfo.Available) / 1024 / 1024 / 1024
	freeGB := float64(data.MemInfo.Free) / 1024 / 1024 / 1024
	cachedGB := float64(data.MemInfo.Cached) / 1024 / 1024 / 1024
	buffersGB := float64(data.MemInfo.Buffers) / 1024 / 1024 / 1024
	swapTotalGB := float64(data.MemInfo.SwapTotal) / 1024 / 1024 / 1024
	swapFreeGB := float64(data.MemInfo.SwapFree) / 1024 / 1024 / 1024

	// 真实使用 = MemTotal - MemAvailable
	realUsedGB := totalGB - availableGB
	realUsedPercent := (realUsedGB / totalGB) * 100.0

	// 应用程序内存 = 真实使用 - 缓存 - 缓冲
	appUsedGB := realUsedGB - cachedGB - buffersGB
	if appUsedGB < 0 {
		appUsedGB = 0
	}
	appUsedPercent := (appUsedGB / totalGB) * 100.0

	// 内存使用率
	drawText(screen, 4, contentY, "内存使用率:", ui.ColorAccent)
	contentY++

	usedColor := ui.ColorSuccess
	if realUsedPercent > 90 {
		usedColor = ui.ColorDanger
	} else if realUsedPercent > 70 {
		usedColor = ui.ColorWarning
	}

	barWidth := width - 40
	if barWidth > 0 && barWidth < width-12 {
		drawProgressBar(screen, 6, contentY, barWidth, realUsedPercent, usedColor)
		memStr := fmt.Sprintf("%.1f%% (%.1fG/%.1fG)", realUsedPercent, realUsedGB, totalGB)
		drawText(screen, 6+barWidth+2, contentY, memStr, ui.ColorText)
	}
	contentY += 2

	// 可用内存
	drawText(screen, 4, contentY, "可用内存:", ui.ColorAccent)
	contentY++

	availablePercent := 100.0 - realUsedPercent
	freeColor := ui.ColorSuccess
	if availablePercent < 10 {
		freeColor = ui.ColorDanger
	} else if availablePercent < 20 {
		freeColor = ui.ColorWarning
	}

	if barWidth > 0 && barWidth < width-12 {
		drawProgressBar(screen, 6, contentY, barWidth, availablePercent, freeColor)
		availStr := fmt.Sprintf("%.1f%% (%.1fG)", availablePercent, availableGB)
		drawText(screen, 6+barWidth+2, contentY, availStr, ui.ColorText)
	}
	contentY += 2

	// 分隔线
	ui.DrawHorizontalLine(screen, 2, contentY, width-4, ui.ColorPrimary)
	contentY++

	// 详细分布
	drawText(screen, 4, contentY, "内存分布:", ui.ColorAccent)
	contentY += 2

	// 使用两列布局
	leftX := 4
	rightX := width/2 + 2
	lineHeight := 1

	// 左列
	drawText(screen, leftX, contentY, fmt.Sprintf("应用程序:  %.2f GB (%.1f%%)", appUsedGB, appUsedPercent), ui.ColorText)
	drawText(screen, leftX, contentY+lineHeight, fmt.Sprintf("缓存:      %.2f GB (%.1f%%)", cachedGB, (cachedGB/totalGB)*100.0), ui.ColorText)
	drawText(screen, leftX, contentY+lineHeight*2, fmt.Sprintf("缓冲:      %.0f MB (%.1f%%)", buffersGB*1024, (buffersGB/totalGB)*100.0), ui.ColorText)

	// 右列
	drawText(screen, rightX, contentY, fmt.Sprintf("空闲内存:  %.2f GB (%.1f%%)", freeGB, (freeGB/totalGB)*100.0), ui.ColorText)
	drawText(screen, rightX, contentY+lineHeight, fmt.Sprintf("总容量:    %.2f GB", totalGB), ui.ColorInfo)
	drawText(screen, rightX, contentY+lineHeight*2, fmt.Sprintf("已使用:    %.2f GB", realUsedGB), ui.ColorInfo)

	contentY += lineHeight*3 + 1

	// 分隔线
	ui.DrawHorizontalLine(screen, 2, contentY, width-4, ui.ColorPrimary)
	contentY++

	// 交换空间
	drawText(screen, 4, contentY, "交换空间:", ui.ColorAccent)
	contentY++

	if swapTotalGB > 0 {
		swapUsedGB := swapTotalGB - swapFreeGB
		swapPercent := (swapUsedGB / swapTotalGB) * 100.0

		swapColor := ui.ColorInfo
		if swapPercent > 50 {
			swapColor = ui.ColorDanger
		} else if swapPercent > 20 {
			swapColor = ui.ColorWarning
		}

		if barWidth > 0 && barWidth < width-12 {
			drawProgressBar(screen, 6, contentY, barWidth, swapPercent, swapColor)
			swapStr := fmt.Sprintf("%.1f%% (%.1fG/%.1fG)", swapPercent, swapUsedGB, swapTotalGB)
			drawText(screen, 6+barWidth+2, contentY, swapStr, ui.ColorText)
		}
	} else {
		drawText(screen, 6, contentY, "未配置交换分区", ui.ColorMuted)
	}

	contentY += 2

	// 底部内存压力
	statsY := height - 2
	ui.DrawDoubleHorizontalLine(screen, 3, statsY-1, width-6, ui.ColorPrimary)

	pressurePercent := realUsedPercent
	pressureColor := ui.ColorSuccess
	pressureLevel := "低"
	if pressurePercent > 90 {
		pressureLevel = "高"
		pressureColor = ui.ColorDanger
	} else if pressurePercent > 70 {
		pressureLevel = "中"
		pressureColor = ui.ColorWarning
	}

	pressureInfo := fmt.Sprintf(" 内存压力: %.1f%% (%s) | 可用: %.1fG | 参考: /proc/meminfo",
		pressurePercent, pressureLevel, availableGB)
	drawText(screen, 4, statsY, pressureInfo, pressureColor)
}
