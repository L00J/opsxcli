package sys

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"opsxcli/internal/ui"
)

// 排序类型
type SortType int

const (
	SortByCPU SortType = iota
	SortByMemory
	SortByDiskIO
	SortByCPUTime
)

var currentSort SortType = SortByCPU

// drawProcesses 绘制进程列表（使用统一边框）
func drawProcesses(screen tcell.Screen, data *SystemData, selected int, width, height int) {
	y := 4

	// 排序提示
	sortHint := ""
	switch currentSort {
	case SortByCPU:
		sortHint = "按CPU排序 | [M]内存 [D]磁盘I/O [T]CPU时间"
	case SortByMemory:
		sortHint = "按内存排序 | [C]CPU [D]磁盘I/O [T]CPU时间"
	case SortByDiskIO:
		sortHint = "按磁盘I/O排序 | [C]CPU [M]内存 [T]CPU时间"
	case SortByCPUTime:
		sortHint = "按CPU时间排序 | [C]CPU [M]内存 [D]磁盘I/O"
	}

	if data == nil || data.Processes == nil || len(data.Processes) == 0 {
		ui.DrawBox(screen, 2, y, width-4, height-y-2, " 进程列表 ", ui.ColorPrimary)
		drawText(screen, 4, y+2, "正在收集进程数据...", ui.ColorMuted)
		return
	}

	// 绘制边框
	title := fmt.Sprintf(" 进程列表 - %s ", sortHint)
	ui.DrawBox(screen, 2, y, width-4, height-y-2, title, ui.ColorPrimary)

	// 显示更新时间
	if !data.UpdateTime.IsZero() {
		updateTimeStr := data.UpdateTime.Format("15:04:05")
		timeStr := fmt.Sprintf("更新: %s", updateTimeStr)
		drawText(screen, width-18, y, timeStr, ui.ColorMuted)
	}

	contentY := y + 2

	// 复制并排序进程
	procs := make([]*ProcessInfo, len(data.Processes))
	copy(procs, data.Processes)

	// 根据当前排序类型排序
	switch currentSort {
	case SortByCPU:
		sort.Slice(procs, func(i, j int) bool {
			return procs[i].CPU > procs[j].CPU
		})
	case SortByMemory:
		sort.Slice(procs, func(i, j int) bool {
			return procs[i].MemRSS > procs[j].MemRSS
		})
	case SortByDiskIO:
		sort.Slice(procs, func(i, j int) bool {
			totalIO1 := procs[i].DiskReadRate + procs[i].DiskWriteRate
			totalIO2 := procs[j].DiskReadRate + procs[j].DiskWriteRate
			return totalIO1 > totalIO2
		})
	case SortByCPUTime:
		sort.Slice(procs, func(i, j int) bool {
			return procs[i].CPUTime > procs[j].CPUTime
		})
	}

	// 表头
	headerStyle := tcell.StyleDefault.
		Foreground(tcell.ColorYellow).
		Bold(true)

	header := fmt.Sprintf("  %-7s %-18s %7s %10s %5s %9s %9s %11s %9s %s",
		"PID", "名称", "CPU%", "CPU时间", "线程", "内存", "虚拟", "磁盘I/O", "运行时", "状态")
	drawTextWithStyle(screen, 4, contentY, header, headerStyle)

	// 分隔线
	ui.DrawHorizontalLine(screen, 2, contentY+1, width-4, ui.ColorPrimary)

	// 进程列表
	listY := contentY + 2
	maxRows := height - listY - 2
	startIdx := 0
	if selected >= maxRows {
		startIdx = selected - maxRows + 1
	}

	for i := startIdx; i < len(procs) && i < startIdx+maxRows; i++ {
		proc := procs[i]
		if proc == nil {
			continue
		}

		// 根据 CPU 使用率设置颜色
		color := tcell.ColorWhite
		if proc.CPU > 100 {
			color = ui.ColorDanger
		} else if proc.CPU > 50 {
			color = ui.ColorWarning
		} else if proc.CPU > 20 {
			color = tcell.ColorYellow
		}

		// 选中高亮
		style := tcell.StyleDefault.Foreground(color)
		if i == selected {
			style = style.Background(tcell.ColorDarkBlue)
		}

		// 格式化数据
		cpuTimeStr := formatCPUTime(proc.CPUTime)
		ramStr := formatBytes(float64(proc.MemRSS))
		virtStr := formatBytes(float64(proc.MemVMS))
		diskIOStr := formatDiskIO(proc.DiskReadRate, proc.DiskWriteRate)
		runtimeStr := formatRuntime(proc.RunTime)
		statusStr := truncate(proc.Status, 9)

		line := fmt.Sprintf("  %-7d %-18s %6.1f%% %10s %5d %9s %9s %11s %9s %s",
			proc.PID,
			truncate(proc.Name, 18),
			proc.CPU,
			cpuTimeStr,
			proc.Threads,
			ramStr,
			virtStr,
			diskIOStr,
			runtimeStr,
			statusStr,
		)

		drawTextWithStyle(screen, 4, listY, line, style)
		listY++
	}

	// 底部统计
	statsY := height - 2
	ui.DrawDoubleHorizontalLine(screen, 3, statsY-1, width-6, ui.ColorPrimary)

	// 构建统计信息
	stats := fmt.Sprintf(" 总计: %d 个进程", len(procs))

	// 添加系统总进程数和最大进程数对比
	if data.TotalProcesses > 0 {
		if data.MaxUserProcesses > 0 {
			// 显示总数和最大数,以及使用百分比
			usagePercent := float64(data.TotalProcesses) / float64(data.MaxUserProcesses) * 100.0
			stats = fmt.Sprintf(" 显示: %d 个进程  |  系统: %d/%d (%.1f%%)",
				len(procs), data.TotalProcesses, data.MaxUserProcesses, usagePercent)
		} else {
			// 只显示总数
			stats = fmt.Sprintf(" 显示: %d 个进程  |  系统: %d 个", len(procs), data.TotalProcesses)
		}
	}

	drawText(screen, 4, statsY, stats, tcell.ColorAqua)
}

// SetSortType 设置排序类型（从 event.go 调用）
func SetSortType(sortType SortType) {
	currentSort = sortType
}
