package sys

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"opsxcli/internal/ui"
)

// drawCPU 绘制CPU详细信息（使用统一边框）
func drawCPU(screen tcell.Screen, data *SystemData, width, height int) {
	y := 4

	// 绘制边框
	ui.DrawBox(screen, 2, y, width-4, height-y-2, " CPU详细信息 ", ui.ColorPrimary)

	// 显示更新时间
	if !data.UpdateTime.IsZero() {
		updateTimeStr := data.UpdateTime.Format("15:04:05")
		timeStr := fmt.Sprintf("更新: %s", updateTimeStr)
		drawText(screen, width-18, y, timeStr, ui.ColorMuted)
	}

	if len(data.CPUPercent) == 0 {
		drawText(screen, 4, y+2, "正在收集CPU数据...", ui.ColorMuted)
		return
	}

	contentY := y + 2

	// 计算总体 CPU 使用率
	var overallCPU float64
	sum := 0.0
	for _, cpu := range data.CPUPercent {
		sum += cpu
	}
	overallCPU = sum / float64(len(data.CPUPercent))

	// 总体 CPU 使用率
	drawText(screen, 4, contentY, "总体使用率:", tcell.ColorYellow)
	contentY++

	cpuColor := ui.ColorSuccess
	if overallCPU > 90 {
		cpuColor = ui.ColorDanger
	} else if overallCPU > 70 {
		cpuColor = ui.ColorWarning
	}

	barWidth := width - 30
	if barWidth > 0 && barWidth < width-12 {
		drawProgressBar(screen, 6, contentY, barWidth, overallCPU, cpuColor)
		percentStr := fmt.Sprintf("%.1f%%", overallCPU)
		drawText(screen, 6+barWidth+2, contentY, percentStr, tcell.ColorWhite)
	}
	contentY += 2

	// 分隔线
	ui.DrawHorizontalLine(screen, 2, contentY, width-4, ui.ColorPrimary)
	contentY++

	// 每个核心的使用率
	drawText(screen, 4, contentY, "各核心使用率:", tcell.ColorYellow)
	contentY++

	// 计算可显示的行数
	maxRows := height - contentY - 3 // 为底部负载信息预留空间
	coresPerRow := 2
	if width < 100 {
		coresPerRow = 1 // 窄屏幕时每行1个核心
	}

	coreIdx := 0
	totalCores := len(data.CPUPercent)

	for row := 0; row < maxRows && coreIdx < totalCores; row++ {
		// 左侧核心
		if coreIdx < totalCores {
			cpuVal := data.CPUPercent[coreIdx]

			// 根据使用率选择颜色
			color := ui.ColorSuccess
			if cpuVal > 90 {
				color = ui.ColorDanger
			} else if cpuVal > 70 {
				color = ui.ColorWarning
			} else if cpuVal > 50 {
				color = tcell.ColorYellow
			}

			coreLabel := fmt.Sprintf("CPU%-3d", coreIdx)
			drawText(screen, 4, contentY, coreLabel, ui.ColorInfo)

			// 进度条
			leftBarWidth := (width - 14) / coresPerRow - 4
			if leftBarWidth > 20 && leftBarWidth < width/2 {
				drawProgressBar(screen, 12, contentY, leftBarWidth, cpuVal, color)
				percentStr := fmt.Sprintf("%5.1f%%", cpuVal)
				drawText(screen, 12+leftBarWidth+2, contentY, percentStr, tcell.ColorWhite)
			}

			coreIdx++
		}

		// 右侧核心(如果空间足够)
		if coresPerRow == 2 && coreIdx < totalCores {
			cpuVal := data.CPUPercent[coreIdx]

			color := ui.ColorSuccess
			if cpuVal > 90 {
				color = ui.ColorDanger
			} else if cpuVal > 70 {
				color = ui.ColorWarning
			} else if cpuVal > 50 {
				color = tcell.ColorYellow
			}

			rightX := width/2 + 2
			coreLabel := fmt.Sprintf("CPU%-3d", coreIdx)
			drawText(screen, rightX, contentY, coreLabel, ui.ColorInfo)

			// 进度条
			rightBarWidth := width - rightX - 20
			if rightBarWidth > 20 {
				drawProgressBar(screen, rightX+8, contentY, rightBarWidth, cpuVal, color)
				percentStr := fmt.Sprintf("%5.1f%%", cpuVal)
				drawText(screen, rightX+8+rightBarWidth+2, contentY, percentStr, tcell.ColorWhite)
			}

			coreIdx++
		}

		contentY++
	}

	// 如果有未显示的核心，提示用户
	if coreIdx < totalCores {
		drawText(screen, 4, contentY, fmt.Sprintf("... 还有 %d 个核心未显示", totalCores-coreIdx), ui.ColorMuted)
		contentY++
	}

	// 底部负载信息
	if data.LoadAvg != nil {
		statsY := height - 2
		ui.DrawDoubleHorizontalLine(screen, 3, statsY-1, width-6, ui.ColorPrimary)

		load1Color := ui.ColorSuccess
		if data.LoadAvg.Load1 > 4.0 {
			load1Color = ui.ColorDanger
		} else if data.LoadAvg.Load1 > 2.0 {
			load1Color = ui.ColorWarning
		}

		loadInfo := fmt.Sprintf(" 系统负载: 1分钟 %.2f | 5分钟 %.2f | 15分钟 %.2f",
			data.LoadAvg.Load1, data.LoadAvg.Load5, data.LoadAvg.Load15)
		drawText(screen, 4, statsY, loadInfo, load1Color)
	}
}
