package net

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"opsxcli/internal/ui"
)

// drawLeftPanel 绘制左侧面板(累计统计和质量指标)
func drawLeftPanel(screen tcell.Screen, data *NetworkData, x, y, width, height int, uptime string) {
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

	contentY = drawSpeedSection(screen, data, x, contentY, width, totalSendRate, totalRecvRate, peakSendRate, peakRecvRate)
	contentY = drawConnectionSection(screen, x, contentY, width, data.ConnectionStats)
	contentY = drawCumulativeSection(screen, data, x, contentY, width, uptime)
	contentY = drawQualitySection(screen, data, x, contentY, width)
}

// drawSpeedSection 绘制速率区域
func drawSpeedSection(screen tcell.Screen, data *NetworkData, x, contentY, width int, totalSendRate, totalRecvRate, peakSendRate, peakRecvRate float64) int {
	// 当前速率
	drawText(screen, x+2, contentY, "当前速率:", ui.ColorAccent)
	rateStr := fmt.Sprintf("↑%s/s ↓%s/s", formatBytes(totalSendRate), formatBytes(totalRecvRate))
	rateColor := ui.ColorText
	if totalSendRate+totalRecvRate > 1024*1024 {
		rateColor = ui.ColorWarning
	}
	if totalSendRate+totalRecvRate > 10*1024*1024 {
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

	ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorSecondary)
	return contentY + 1
}

// drawConnectionSection 绘制连接统计区域
func drawConnectionSection(screen tcell.Screen, x, contentY, width int, connectionStats map[string]int) int {
	drawText(screen, x+2, contentY, "连接统计:", ui.ColorAccent)
	contentY++

	// 活跃连接
	drawText(screen, x+2, contentY, "活跃连接:", ui.ColorAccent)
	establishedCount := connectionStats["ESTABLISHED"]
	connColor := ui.ColorText
	if establishedCount > 1000 {
		connColor = ui.ColorDanger
	} else if establishedCount > 500 {
		connColor = ui.ColorWarning
	}
	drawText(screen, x+14, contentY, formatNumber(uint64(establishedCount)), connColor)
	contentY++

	// TIME_WAIT
	drawText(screen, x+2, contentY, "TIME_WAIT:", ui.ColorAccent)
	timewaitCount := connectionStats["TIME_WAIT"]
	twColor := ui.ColorText
	if timewaitCount > 5000 {
		twColor = ui.ColorDanger
	} else if timewaitCount > 1000 {
		twColor = ui.ColorWarning
	}
	drawText(screen, x+14, contentY, formatNumber(uint64(timewaitCount)), twColor)
	contentY++

	// 监听端口
	drawText(screen, x+2, contentY, "监听端口:", ui.ColorAccent)
	drawText(screen, x+14, contentY, formatNumber(uint64(connectionStats["LISTEN"])), ui.ColorText)
	contentY += 2

	ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorSecondary)
	return contentY + 1
}

// drawCumulativeSection 绘制累计数据区域
func drawCumulativeSection(screen tcell.Screen, data *NetworkData, x, contentY, width int, uptime string) int {
	drawText(screen, x+2, contentY, "累计数据:", ui.ColorAccent)
	contentY++

	drawText(screen, x+2, contentY, "运行时间:", ui.ColorMuted)
	drawText(screen, x+14, contentY, uptime, ui.ColorInfo)
	contentY++

	totalBytes := data.TotalBytesSent + data.TotalBytesRecv
	drawText(screen, x+2, contentY, "总流量:", ui.ColorMuted)
	drawText(screen, x+14, contentY, formatBytes(float64(totalBytes)), ui.ColorText)
	contentY++

	totalPackets := data.TotalPacketsSent + data.TotalPacketsRecv
	drawText(screen, x+2, contentY, "总包数:", ui.ColorMuted)
	drawText(screen, x+14, contentY, formatNumber(totalPackets), ui.ColorText)
	contentY += 2

	ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorSecondary)
	return contentY + 1
}

// drawQualitySection 绘制质量指标区域
func drawQualitySection(screen tcell.Screen, data *NetworkData, x, contentY, width int) int {
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

	ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorSecondary)
	contentY++

	drawText(screen, x+2, contentY, "总错误:", ui.ColorMuted)
	drawText(screen, x+14, contentY, formatNumber(data.TotalErrors), ui.ColorDanger)
	contentY++

	drawText(screen, x+2, contentY, "总丢包:", ui.ColorMuted)
	drawText(screen, x+14, contentY, formatNumber(data.TotalDrops), ui.ColorWarning)

	return contentY
}
