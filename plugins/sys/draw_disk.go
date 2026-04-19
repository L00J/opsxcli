package sys

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"opsxcli/internal/ui"
)

// DiskSortType 磁盘排序类型
type DiskSortType int

const (
	DiskSortBySpace DiskSortType = iota
	DiskSortByInodes
	DiskSortByIO
	DiskSortByName
)

var currentDiskSort DiskSortType = DiskSortBySpace

// SetDiskSortType 设置磁盘排序类型
func SetDiskSortType(sortType DiskSortType) {
	currentDiskSort = sortType
}

// drawDisk 绘制磁盘详细信息（使用统一边框）
func drawDisk(screen tcell.Screen, data *SystemData, width, height int) {
	y := 4

	// 建立 I/O 统计映射
	ioStatsMap := make(map[string]float64)
	for _, ioStat := range data.DiskIOStats {
		ioStatsMap[ioStat.Name] = ioStat.UtilPercent
	}

	// 获取并排序磁盘列表
	diskList := getSortedDiskList(data, currentDiskSort)

	if len(diskList) == 0 {
		ui.DrawBox(screen, 2, y, width-4, height-y-2, " 磁盘详情 ", ui.ColorPrimary)
		drawText(screen, 4, y+2, "⏳ 首次加载中，请稍候 (约2秒)...", ui.ColorMuted)
		return
	}

	// 排序提示
	sortHint := ""
	switch currentDiskSort {
	case DiskSortBySpace:
		sortHint = "按空间排序 | [2]Inodes [3]I/O [4]名称"
	case DiskSortByInodes:
		sortHint = "按Inodes排序 | [1]空间 [3]I/O [4]名称"
	case DiskSortByIO:
		sortHint = "按I/O排序 | [1]空间 [2]Inodes [4]名称"
	case DiskSortByName:
		sortHint = "按名称排序 | [1]空间 [2]Inodes [3]I/O"
	}

	// 绘制边框
	title := fmt.Sprintf(" 磁盘详细信息 - %s ", sortHint)
	ui.DrawBox(screen, 2, y, width-4, height-y-2, title, ui.ColorPrimary)

	// 显示更新时间
	if !data.UpdateTime.IsZero() {
		updateTimeStr := data.UpdateTime.Format("15:04:05")
		timeStr := fmt.Sprintf("更新: %s", updateTimeStr)
		drawText(screen, width-18, y, timeStr, ui.ColorMuted)
	}

	contentY := y + 2

	// 计算汇总信息
	var totalUsed, totalTotal uint64
	for _, disk := range diskList {
		totalUsed += disk.Used
		totalTotal += disk.Total
	}

	totalUsedPercent := 0.0
	if totalTotal > 0 {
		totalUsedPercent = float64(totalUsed) / float64(totalTotal) * 100.0
	}

	// 总体磁盘使用情况
	drawText(screen, 4, contentY, "总体使用率:", ui.ColorAccent)
	contentY++

	diskColor := ui.ColorSuccess
	if totalUsedPercent > 90 {
		diskColor = ui.ColorDanger
	} else if totalUsedPercent > 70 {
		diskColor = ui.ColorWarning
	}

	barWidth := width - 40
	if barWidth > 0 && barWidth < width-12 {
		drawProgressBar(screen, 6, contentY, barWidth, totalUsedPercent, diskColor)
		diskStr := fmt.Sprintf("%.1f%% (%s/%s)", totalUsedPercent,
			formatBytesShort(float64(totalUsed)),
			formatBytesShort(float64(totalTotal)))
		drawText(screen, 6+barWidth+2, contentY, diskStr, ui.ColorText)
	}
	contentY += 2

	// 分隔线
	ui.DrawHorizontalLine(screen, 2, contentY, width-4, ui.ColorPrimary)
	contentY++

	// 表头
	headerStyle := tcell.StyleDefault.
		Foreground(ui.ColorAccent).
		Bold(true)

	header := fmt.Sprintf("  %-12s %-10s %-12s %-10s %-10s %-10s %s",
		"设备", "类型", "容量", "使用%", "Inodes%", "I/O%", "挂载点")
	drawTextWithStyle(screen, 4, contentY, header, headerStyle)

	// 分隔线
	ui.DrawHorizontalLine(screen, 2, contentY+1, width-4, ui.ColorPrimary)

	// 显示磁盘列表
	listY := contentY + 2
	maxRows := height - listY - 3

	for i := 0; i < maxRows && i < len(diskList); i++ {
		disk := diskList[i]

		// 格式化容量
		capacityStr := formatBytesShort(float64(disk.Total))

		// 格式化使用百分比
		usedPercentStr := fmt.Sprintf("%.0f%%", disk.UsedPercent)
		if disk.UsedPercent > 80 {
			usedPercentStr += "!"
		}

		// 格式化 Inode 百分比
		inodesPercentStr := fmt.Sprintf("%.0f%%", disk.InodesPercent)
		if disk.InodesPercent > 10 {
			inodesPercentStr += "!"
		}

		// 格式化 I/O 利用率
		ioUtilPercent := disk.IOUtilPercent
		if util, ok := ioStatsMap[disk.Device]; ok {
			ioUtilPercent = util
		}
		ioPercentStr := fmt.Sprintf("%.0f%%", ioUtilPercent)

		// 根据警告状态选择颜色
		color := ui.ColorText
		if disk.UsedPercent > 80 || disk.InodesPercent > 10 {
			color = ui.ColorDanger
		} else if disk.UsedPercent > 60 || ioUtilPercent > 50 {
			color = ui.ColorWarning
		}

		// 构建行
		mountPoint := disk.MountPoint
		if len(mountPoint) > 20 {
			mountPoint = mountPoint[:20]
		}

		dataRow := fmt.Sprintf("  %-12s %-10s %-12s %-10s %-10s %-10s %s",
			truncate(disk.Device, 12),
			truncate(disk.DiskType, 10),
			capacityStr,
			usedPercentStr,
			inodesPercentStr,
			ioPercentStr,
			mountPoint)

		drawText(screen, 4, listY, dataRow, color)
		listY++
	}

	// 底部统计
	statsY := height - 2
	ui.DrawDoubleHorizontalLine(screen, 3, statsY-1, width-6, ui.ColorPrimary)

	stats := fmt.Sprintf(" 磁盘总计: %s / %s | 使用率: %.1f%% | ! = 警告(空间>80%% 或 Inodes>10%%)",
		formatBytesShort(float64(totalUsed)),
		formatBytesShort(float64(totalTotal)),
		totalUsedPercent)

	statColor := ui.ColorInfo
	if totalUsedPercent > 80 {
		statColor = ui.ColorDanger
	}
	drawText(screen, 4, statsY, stats, statColor)
}

// getSortedDiskList 获取排序后的磁盘列表
func getSortedDiskList(data *SystemData, sortType DiskSortType) []DiskUsageInfo {
	diskList := make([]DiskUsageInfo, len(data.DiskUsageList))
	copy(diskList, data.DiskUsageList)

	sort.Slice(diskList, func(i, j int) bool {
		switch sortType {
		case DiskSortBySpace:
			return diskList[i].UsedPercent > diskList[j].UsedPercent
		case DiskSortByInodes:
			return diskList[i].InodesPercent > diskList[j].InodesPercent
		case DiskSortByIO:
			return diskList[i].IOUtilPercent > diskList[j].IOUtilPercent
		case DiskSortByName:
			return diskList[i].Device < diskList[j].Device
		default:
			return diskList[i].UsedPercent > diskList[j].UsedPercent
		}
	})

	return diskList
}
