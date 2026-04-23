package sys

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"opsxcli/internal/ui"
)

// drawOverview 绘制总览页面（优化布局，使用边框）
func drawOverview(screen tcell.Screen, data *SystemData, width, height int) {
	y := 4

	// 顶部系统资源面板（CPU、内存、磁盘）
	panelHeight := 12

	// 左侧：CPU 和内存
	leftWidth := width/2 - 2
	drawResourcePanel(screen, data, 2, y, leftWidth, panelHeight)

	// 右侧：系统信息和网络
	rightX := width/2 + 1
	rightWidth := width - rightX - 2
	drawSystemInfoPanel(screen, data, rightX, y, rightWidth, panelHeight)

	// 底部：TOP进程列表
	processY := y + panelHeight + 1
	processHeight := height - processY - 2
	drawTopProcessesPanel(screen, data, 2, processY, width-4, processHeight)
}

// drawResourcePanel 绘制资源使用面板（CPU和内存）
func drawResourcePanel(screen tcell.Screen, data *SystemData, x, y, width, height int) {
	// 绘制边框
	ui.DrawBox(screen, x, y, width, height, " 系统资源 ", ui.ColorPrimary)

	contentY := y + 2

	// 显示更新时间
	if !data.UpdateTime.IsZero() {
		updateTimeStr := data.UpdateTime.Format("15:04:05")
		timeStr := fmt.Sprintf("更新: %s", updateTimeStr)
		drawText(screen, x+width-14, y, timeStr, ui.ColorMuted)
	}

	barWidth := width - 20

	// CPU 使用率
	drawText(screen, x+2, contentY, "CPU 使用率:", ui.ColorAccent)
	contentY++

	if len(data.CPUPercent) == 0 {
		drawText(screen, x+4, contentY, "⏳ 等待第二次采样，约1秒后显示...", ui.ColorMuted)
		contentY += 2
	} else {
		var overallCPU float64
		sum := 0.0
		for _, cpu := range data.CPUPercent {
			sum += cpu
		}
		overallCPU = sum / float64(len(data.CPUPercent))

		cpuColor := ui.ColorSuccess
		if overallCPU > 90 {
			cpuColor = ui.ColorDanger
		} else if overallCPU > 70 {
			cpuColor = ui.ColorWarning
		}

		if barWidth > 0 {
			drawProgressBar(screen, x+4, contentY, barWidth, overallCPU, cpuColor)
			percentStr := fmt.Sprintf("%.1f%%", overallCPU)
			drawText(screen, x+barWidth+6, contentY, percentStr, ui.ColorText)
		}
		contentY++

		// 显示核心数和负载参考
		if data.CPUCores > 0 && data.LoadAvg != nil {
			loadPerCore := data.LoadAvg.Load1 / float64(data.CPUCores)
			loadColor := ui.ColorSuccess
			if loadPerCore > 1.0 {
				loadColor = ui.ColorDanger
			} else if loadPerCore > 0.7 {
				loadColor = ui.ColorWarning
			}
			coreInfo := fmt.Sprintf("  %d核心 | 负载: %.2f (%.0f%%)",
				data.CPUCores, data.LoadAvg.Load1, loadPerCore*100)
			drawText(screen, x+4, contentY, coreInfo, loadColor)
		}
		contentY++
	}

	// 内存使用
	if data.MemInfo != nil {
		memUsed := float64(data.MemInfo.Used) / 1024 / 1024 / 1024
		memTotal := float64(data.MemInfo.Total) / 1024 / 1024 / 1024
		realUsedPercent := (float64(data.MemInfo.Total-data.MemInfo.Available) / float64(data.MemInfo.Total)) * 100.0

		drawText(screen, x+2, contentY, "内存使用率:", ui.ColorAccent)
		contentY++

		memColor := ui.ColorSuccess
		if realUsedPercent > 90 {
			memColor = ui.ColorDanger
		} else if realUsedPercent > 70 {
			memColor = ui.ColorWarning
		}

		if barWidth > 0 {
			drawProgressBar(screen, x+4, contentY, barWidth, realUsedPercent, memColor)
			memStr := fmt.Sprintf("%.1fG/%.1fG", memUsed, memTotal)
			drawText(screen, x+barWidth+6, contentY, memStr, ui.ColorText)
		}
		contentY += 2

		// Swap 使用
		swapUsed := float64(data.MemInfo.SwapTotal-data.MemInfo.SwapFree) / 1024 / 1024 / 1024
		swapTotal := float64(data.MemInfo.SwapTotal) / 1024 / 1024 / 1024

		drawText(screen, x+2, contentY, "交换分区:", ui.ColorAccent)
		contentY++

		if swapTotal > 0 {
			swapPercent := float64(data.MemInfo.SwapTotal-data.MemInfo.SwapFree) / float64(data.MemInfo.SwapTotal) * 100
			swapColor := ui.ColorInfo
			if swapPercent > 50 {
				swapColor = ui.ColorDanger
			}
			if barWidth > 0 {
				drawProgressBar(screen, x+4, contentY, barWidth, swapPercent, swapColor)
				swapStr := fmt.Sprintf("%.1fG/%.1fG", swapUsed, swapTotal)
				drawText(screen, x+barWidth+6, contentY, swapStr, ui.ColorText)
			}
		} else {
			drawText(screen, x+4, contentY, "未配置交换分区", ui.ColorMuted)
		}
	}
}

// drawSystemInfoPanel 绘制系统信息面板
func drawSystemInfoPanel(screen tcell.Screen, data *SystemData, x, y, width, height int) {
	// 绘制边框
	ui.DrawBox(screen, x, y, width, height, " 系统信息 ", ui.ColorPrimary)

	contentY := y + 2

	// 系统负载
	if data.LoadAvg != nil {
		drawText(screen, x+2, contentY, "系统负载:", ui.ColorAccent)
		contentY++

		load1Color := ui.ColorSuccess
		if data.LoadAvg.Load1 > 4.0 {
			load1Color = ui.ColorDanger
		} else if data.LoadAvg.Load1 > 2.0 {
			load1Color = ui.ColorWarning
		}

		loadStr := fmt.Sprintf("  1分钟:  %.2f", data.LoadAvg.Load1)
		drawText(screen, x+2, contentY, loadStr, load1Color)
		contentY++

		loadStr = fmt.Sprintf("  5分钟:  %.2f", data.LoadAvg.Load5)
		drawText(screen, x+2, contentY, loadStr, ui.ColorText)
		contentY++

		loadStr = fmt.Sprintf("  15分钟: %.2f", data.LoadAvg.Load15)
		drawText(screen, x+2, contentY, loadStr, ui.ColorText)
		contentY += 2
	}

	// 进程统计
	drawText(screen, x+2, contentY, "进程统计:", ui.ColorAccent)
	contentY++

	// 显示总进程数
	if data.TotalProcesses > 0 {
		drawText(screen, x+2, contentY, fmt.Sprintf("  总数: %d", data.TotalProcesses), ui.ColorText)
	} else if len(data.Processes) > 0 {
		drawText(screen, x+2, contentY, fmt.Sprintf("  总数: %d", len(data.Processes)), ui.ColorText)
	} else {
		drawText(screen, x+2, contentY, "  总数: -", ui.ColorText)
	}
	contentY++

	// 显示最大进程数限制
	if data.MaxUserProcesses > 0 {
		// 计算使用百分比
		if data.TotalProcesses > 0 {
			usagePercent := float64(data.TotalProcesses) / float64(data.MaxUserProcesses) * 100.0
			limitColor := ui.ColorSuccess
			if usagePercent > 90 {
				limitColor = ui.ColorDanger
			} else if usagePercent > 70 {
				limitColor = ui.ColorWarning
			}
			drawText(screen, x+2, contentY, fmt.Sprintf("  限制: %d (%.1f%%)", data.MaxUserProcesses, usagePercent), limitColor)
		} else {
			drawText(screen, x+2, contentY, fmt.Sprintf("  限制: %d", data.MaxUserProcesses), ui.ColorText)
		}
	} else {
		drawText(screen, x+2, contentY, "  限制: unlimited", ui.ColorMuted)
	}
}

// drawTopProcessesPanel 绘制TOP进程列表面板
func drawTopProcessesPanel(screen tcell.Screen, data *SystemData, x, y, width, height int) {
	// 绘制边框
	ui.DrawBox(screen, x, y, width, height, " TOP 进程 (按CPU排序) ", ui.ColorPrimary)

	if data == nil || data.Processes == nil || len(data.Processes) == 0 {
		drawText(screen, x+2, y+2, "⏳ 正在收集进程数据，请稍候...", ui.ColorMuted)
		return
	}

	// 复制并排序进程
	procs := make([]*ProcessInfo, len(data.Processes))
	copy(procs, data.Processes)
	sort.Slice(procs, func(i, j int) bool {
		if procs[i].CPU != procs[j].CPU {
			return procs[i].CPU > procs[j].CPU
		}
		return procs[i].MemRSS > procs[j].MemRSS
	})

	// 表头
	headerY := y + 2
	headerStyle := tcell.StyleDefault.
		Foreground(ui.ColorAccent).
		Bold(true)

	header := fmt.Sprintf("  %-7s %-18s %7s %9s %9s %s",
		"PID", "名称", "CPU%", "内存", "状态", "命令")
	drawTextWithStyle(screen, x+2, headerY, header, headerStyle)

	// 分隔线
	ui.DrawHorizontalLine(screen, x, headerY+1, width, ui.ColorPrimary)

	// 进程列表
	listY := headerY + 2
	maxRows := height - 4
	displayCount := maxRows
	if displayCount > len(procs) {
		displayCount = len(procs)
	}
	if displayCount > 15 {
		displayCount = 15
	}

	for i := 0; i < displayCount && listY < y+height-1; i++ {
		proc := procs[i]
		if proc == nil {
			continue
		}

		// 根据CPU使用率设置颜色
		color := ui.ColorText
		if proc.CPU > 100 {
			color = ui.ColorDanger
		} else if proc.CPU > 50 {
			color = ui.ColorWarning
		} else if proc.CPU > 20 {
			color = ui.ColorAccent
		}

		memStr := formatBytes(float64(proc.MemRSS))
		statusStr := truncate(proc.Status, 8)
		cmdStr := truncate(proc.Command, width-60)
		if cmdStr == "" {
			cmdStr = proc.Name
		}

		line := fmt.Sprintf("  %-7d %-18s %6.1f%% %9s %-8s %s",
			proc.PID,
			truncate(proc.Name, 18),
			proc.CPU,
			memStr,
			statusStr,
			cmdStr)

		drawText(screen, x+2, listY, line, color)
		listY++
	}
}
