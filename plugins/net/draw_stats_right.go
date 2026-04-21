package net

import (
	"fmt"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"opsxcli/internal/ui"
)

// drawRightPanel 绘制右侧面板(所有接口流量TOP)
func drawRightPanel(screen tcell.Screen, data *NetworkData, x, y, width, height int) {
	ui.DrawBox(screen, x, y, width, height, " 流量排行 ", ui.ColorSecondary)

	contentY := y + 2

	type ifaceRanking struct {
		name       string
		totalBytes uint64
		sendRate   float64
		recvRate   float64
		isUp       bool
	}

	rankings := make([]ifaceRanking, 0)
	for _, iface := range data.Interfaces {
		if traffic, ok := data.InterfaceTraffic[iface.Name]; ok {
			rankings = append(rankings, ifaceRanking{
				name:       iface.Name,
				totalBytes: traffic.TotalSent + traffic.TotalRecv,
				sendRate:   traffic.SendRate,
				recvRate:   traffic.RecvRate,
				isUp:       iface.IsUp,
			})
		}
	}

	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].totalBytes > rankings[j].totalBytes
	})

	maxRows := (height - 4) / 4
	for i, rank := range rankings {
		if i >= maxRows {
			break
		}

		rankNum := fmt.Sprintf("#%d", i+1)

		statusIcon := ui.ArrowDown
		nameColor := ui.ColorMuted
		if rank.isUp {
			statusIcon = ui.ArrowUp
			nameColor = ui.ColorSuccess
		}

		nameStr := fmt.Sprintf("%s %c %-10s", rankNum, statusIcon, rank.name)
		drawText(screen, x+2, contentY, nameStr, nameColor)
		contentY++

		totalStr := fmt.Sprintf("  总流量: %s", formatBytes(float64(rank.totalBytes)))
		drawText(screen, x+2, contentY, totalStr, ui.ColorText)
		contentY++

		rateStr := fmt.Sprintf("  速率: ↑%s/s ↓%s/s",
			formatBytes(rank.sendRate),
			formatBytes(rank.recvRate))
		drawText(screen, x+2, contentY, rateStr, ui.ColorMuted)
		contentY++

		if i < len(rankings)-1 && i < maxRows-1 {
			ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorMuted)
			contentY++
		}
	}
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d时%d分%d秒", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%d分%d秒", minutes, seconds)
	}
	return fmt.Sprintf("%d秒", seconds)
}

// formatNumber 格式化数字(添加千位分隔符)
func formatNumber(n uint64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	if n < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	return fmt.Sprintf("%.1fG", float64(n)/1000000000)
}
