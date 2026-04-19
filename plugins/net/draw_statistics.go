package net

import (
	"fmt"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"opsxcli/internal/ui"
	"opsxcli/plugins/netstat"
)

// drawStatistics 绘制流量统计界面(使用统一边框)
func drawStatistics(screen tcell.Screen, data *NetworkData, selectedIf int, width, height int) {
	y := 4

	if len(data.Interfaces) == 0 {
		ui.DrawBox(screen, 2, y, width-4, height-y-2, " 流量统计 ", ui.ColorSecondary)
		drawText(screen, 4, y+2, "⏳ 首次加载中，请稍候 (约2秒)...", ui.ColorMuted)
		return
	}

	// 计算运行时间
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

// drawLeftPanel 绘制左侧面板(累计统计和质量指标)
func drawLeftPanel(screen tcell.Screen, data *NetworkData, x, y, width, height int, uptime string) {
	// 绘制边框
	ui.DrawBox(screen, x, y, width, height, " 累计与质量 ", ui.ColorSecondary)

	contentY := y + 2

	// === 实时监控 ===
	drawText(screen, x+2, contentY, "实时监控:", ui.ColorAccent)
	contentY++

	// 更新时间
	updateTimeStr := data.UpdateTime.Format("15:04:05")
	drawText(screen, x+2, contentY, "更新时间:", ui.ColorMuted)
	drawText(screen, x+14, contentY, updateTimeStr, ui.ColorInfo)
	contentY += 2

	// 计算全局实时速率（所有活跃接口的速率之和）
	var totalSendRate, totalRecvRate float64
	var peakSendRate, peakRecvRate float64
	for _, traffic := range data.InterfaceTraffic {
		totalSendRate += traffic.SendRate
		totalRecvRate += traffic.RecvRate
		if traffic.PeakSendRate > peakSendRate {
			peakSendRate = traffic.PeakSendRate
		}
		if traffic.PeakRecvRate > peakRecvRate {
			peakRecvRate = traffic.PeakRecvRate
		}
	}

	// 当前速率
	drawText(screen, x+2, contentY, "当前速率:", ui.ColorAccent)
	rateStr := fmt.Sprintf("↑%s/s ↓%s/s", formatBytes(totalSendRate), formatBytes(totalRecvRate))
	rateColor := ui.ColorText
	if totalSendRate+totalRecvRate > 1024*1024 { // > 1MB/s
		rateColor = ui.ColorWarning
	}
	if totalSendRate+totalRecvRate > 10*1024*1024 { // > 10MB/s
		rateColor = ui.ColorDanger
	}
	drawText(screen, x+14, contentY, rateStr, rateColor)
	contentY++

	// 峰值速率
	drawText(screen, x+2, contentY, "峰值速率:", ui.ColorAccent)
	peakStr := fmt.Sprintf("↑%s/s ↓%s/s", formatBytes(peakSendRate), formatBytes(peakRecvRate))
	drawText(screen, x+14, contentY, peakStr, ui.ColorAccent)
	contentY++

	// 平均速率
	uptimeSec := time.Since(data.StartTime).Seconds()
	if uptimeSec == 0 {
		uptimeSec = 1
	}
	avgSendRate := float64(data.TotalBytesSent) / uptimeSec
	avgRecvRate := float64(data.TotalBytesRecv) / uptimeSec
	drawText(screen, x+2, contentY, "平均速率:", ui.ColorAccent)
	avgStr := fmt.Sprintf("↑%s/s ↓%s/s", formatBytes(avgSendRate), formatBytes(avgRecvRate))
	drawText(screen, x+14, contentY, avgStr, ui.ColorMuted)
	contentY += 2

	// 分隔线
	ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorSecondary)
	contentY++

	// === 连接统计 (实时) ===
	drawText(screen, x+2, contentY, "连接统计:", ui.ColorAccent)
	contentY++

	// 获取实时连接数
	connections := getConnectionStats()

	drawText(screen, x+2, contentY, "活跃连接:", ui.ColorAccent)
	establishedCount := connections["ESTABLISHED"]
	connColor := ui.ColorText
	if establishedCount > 1000 {
		connColor = ui.ColorDanger
	} else if establishedCount > 500 {
		connColor = ui.ColorWarning
	}
	drawText(screen, x+14, contentY, formatNumber(uint64(establishedCount)), connColor)
	contentY++

	drawText(screen, x+2, contentY, "TIME_WAIT:", ui.ColorAccent)
	timewaitCount := connections["TIME_WAIT"]
	twColor := ui.ColorText
	if timewaitCount > 5000 {
		twColor = ui.ColorDanger
	} else if timewaitCount > 1000 {
		twColor = ui.ColorWarning
	}
	drawText(screen, x+14, contentY, formatNumber(uint64(timewaitCount)), twColor)
	contentY++

	drawText(screen, x+2, contentY, "监听端口:", ui.ColorAccent)
	drawText(screen, x+14, contentY, formatNumber(uint64(connections["LISTEN"])), ui.ColorText)
	contentY += 2

	// 分隔线
	ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorSecondary)
	contentY++

	// === 累计数据 ===
	drawText(screen, x+2, contentY, "累计数据:", ui.ColorAccent)
	contentY++

	drawText(screen, x+2, contentY, "运行时间:", ui.ColorMuted)
	drawText(screen, x+14, contentY, uptime, ui.ColorInfo)
	contentY++

	// 总流量（简化显示）
	totalBytes := data.TotalBytesSent + data.TotalBytesRecv
	drawText(screen, x+2, contentY, "总流量:", ui.ColorMuted)
	drawText(screen, x+14, contentY, formatBytes(float64(totalBytes)), ui.ColorText)
	contentY++

	// 总包数（简化显示）
	totalPackets := data.TotalPacketsSent + data.TotalPacketsRecv
	drawText(screen, x+2, contentY, "总包数:", ui.ColorMuted)
	drawText(screen, x+14, contentY, formatNumber(totalPackets), ui.ColorText)
	contentY += 2

	// 分隔线
	ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorSecondary)
	contentY++

	// === 质量指标 ===
	drawText(screen, x+2, contentY, "质量指标:", ui.ColorAccent)
	contentY += 2

	// 丢包率
	lossColor := ui.ColorSuccess
	if data.PacketLossRate > 1.0 {
		lossColor = ui.ColorDanger
	} else if data.PacketLossRate > 0.1 {
		lossColor = ui.ColorWarning
	}
	drawText(screen, x+2, contentY, "丢包率:", ui.ColorAccent)
	lossStr := fmt.Sprintf("%.4f%%", data.PacketLossRate)
	drawText(screen, x+14, contentY, lossStr, lossColor)
	contentY++

	// 错误率
	errorColor := ui.ColorSuccess
	if data.ErrorRate > 1.0 {
		errorColor = ui.ColorDanger
	} else if data.ErrorRate > 0.1 {
		errorColor = ui.ColorWarning
	}
	drawText(screen, x+2, contentY, "错误率:", ui.ColorAccent)
	errorStr := fmt.Sprintf("%.4f%%", data.ErrorRate)
	drawText(screen, x+14, contentY, errorStr, errorColor)
	contentY += 2

	// 分隔线
	ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorSecondary)
	contentY++

	// 总错误和丢包数（简化）
	drawText(screen, x+2, contentY, "总错误:", ui.ColorMuted)
	drawText(screen, x+14, contentY, formatNumber(data.TotalErrors), ui.ColorDanger)
	contentY++

	drawText(screen, x+2, contentY, "总丢包:", ui.ColorMuted)
	drawText(screen, x+14, contentY, formatNumber(data.TotalDrops), ui.ColorWarning)
}

// drawRightPanel 绘制右侧面板(所有接口流量TOP)
func drawRightPanel(screen tcell.Screen, data *NetworkData, x, y, width, height int) {
	// 绘制边框
	ui.DrawBox(screen, x, y, width, height, " 流量排行 ", ui.ColorSecondary)

	contentY := y + 2

	// 创建接口列表并按总流量排序
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

	// 按总流量降序排序
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].totalBytes > rankings[j].totalBytes
	})

	// 绘制排行榜
	maxRows := (height - 4) / 4 // 每个接口占4行
	for i, rank := range rankings {
		if i >= maxRows {
			break
		}

		// 排名号和接口名称
		rankNum := fmt.Sprintf("#%d", i+1)
		if i == 0 {
			rankNum = "#1" // 第一名
		} else if i == 1 {
			rankNum = "#2" // 第二名
		}

		// 状态图标
		statusIcon := ui.ArrowDown
		nameColor := ui.ColorMuted
		if rank.isUp {
			statusIcon = ui.ArrowUp
			nameColor = ui.ColorSuccess
		}

		nameStr := fmt.Sprintf("%s %c %-10s", rankNum, statusIcon, rank.name)
		drawText(screen, x+2, contentY, nameStr, nameColor)
		contentY++

		// 总流量
		totalStr := fmt.Sprintf("  总流量: %s", formatBytes(float64(rank.totalBytes)))
		drawText(screen, x+2, contentY, totalStr, ui.ColorText)
		contentY++

		// 当前速率
		rateStr := fmt.Sprintf("  速率: ↑%s/s ↓%s/s",
			formatBytes(rank.sendRate),
			formatBytes(rank.recvRate))
		drawText(screen, x+2, contentY, rateStr, ui.ColorMuted)
		contentY++

		// 分隔线
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

// getConnectionStats 获取实时连接统计
func getConnectionStats() map[string]int {
	stats := make(map[string]int)

	// 使用netstat包获取连接信息
	connections := []netstat.SsConnection{}
	connections = append(connections, netstat.ReadTCPConnectionsWithPrograms(false, true, false)...)

	// 统计各状态的连接数
	for _, conn := range connections {
		stats[conn.State]++
	}

	return stats
}
