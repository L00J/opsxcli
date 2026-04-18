package net

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"opsxcli/internal/ui"
)

// drawRealtime 绘制实时流量界面（类似iftop，使用统一边框）
func drawRealtime(screen tcell.Screen, data *NetworkData, selectedIf int, width, height int) {
	y := 4

	if len(data.Interfaces) == 0 {
		drawText(screen, 2, y, "正在收集网络数据...", ui.ColorMuted)
		return
	}

	// 绘制边框
	ui.DrawBox(screen, 2, y, width-4, height-y-2, " 实时流量 ", ui.ColorSecondary)

	contentY := y + 2

	// 创建接口流量列表并按速率排序(最大流量在top)
	type ifaceWithTraffic struct {
		info    *InterfaceInfo
		traffic *InterfaceTraffic
		total   float64
	}

	ifaceList := make([]ifaceWithTraffic, 0)
	for _, iface := range data.Interfaces {
		if traffic, ok := data.InterfaceTraffic[iface.Name]; ok {
			// 只显示有流量或活跃的接口
			if iface.IsUp || traffic.SendRate > 0 || traffic.RecvRate > 0 {
				ifaceList = append(ifaceList, ifaceWithTraffic{
					info:    iface,
					traffic: traffic,
					total:   traffic.SendRate + traffic.RecvRate,
				})
			}
		}
	}

	// 按总流量降序排序(最大流量在上面)
	sort.Slice(ifaceList, func(i, j int) bool {
		return ifaceList[i].total > ifaceList[j].total
	})

	// 绘制接口列表
	maxRows := (height - y - 6) / 5 // 每个接口占5行
	for i, item := range ifaceList {
		if i >= maxRows {
			break
		}

		iface := item.info
		traffic := item.traffic

		// 背景高亮选中项
		if i == selectedIf {
			for x := 4; x < width-6; x++ {
				for dy := 0; dy < 4; dy++ {
					screen.SetContent(x, contentY+dy, ' ', nil, tcell.StyleDefault.Background(tcell.ColorDarkBlue))
				}
			}
		}

		// 接口名称、状态和IP
		statusIcon := ui.ArrowDown
		if iface.IsUp {
			statusIcon = ui.ArrowUp
		}

		nameStyle := tcell.StyleDefault.Foreground(ui.ColorInfo).Bold(true)
		if i == selectedIf {
			nameStyle = nameStyle.Background(tcell.ColorDarkBlue)
		}

		ipStr := ""
		if len(iface.Addrs) > 0 {
			ipStr = iface.Addrs[0]
			if len(ipStr) > 22 {
				ipStr = ipStr[:22]
			}
		}

		ifaceHeader := fmt.Sprintf("%c %-12s  %s", statusIcon, iface.Name, ipStr)
		drawTextWithStyle(screen, 4, contentY, ifaceHeader, nameStyle)
		contentY++

		// TX (发送) 流量条
		txLabel := fmt.Sprintf("  TX ► %-14s", formatBytes(traffic.SendRate)+"/s")
		txStyle := tcell.StyleDefault.Foreground(ui.ColorSuccess)
		if i == selectedIf {
			txStyle = txStyle.Background(tcell.ColorDarkBlue)
		}
		drawTextWithStyle(screen, 4, contentY, txLabel, txStyle)

		// TX 进度条
		barWidth := width - 48
		barX := 24
		if barWidth > 0 && barWidth < 80 {
			maxBandwidth := 100.0 * 1024 * 1024 // 100MB/s
			txPercent := (traffic.SendRate / maxBandwidth) * 100
			if txPercent > 100 {
				txPercent = 100
			}
			drawProgressBar(screen, barX, contentY, barWidth, txPercent, ui.ColorSuccess)

			// 显示峰值
			peakStr := fmt.Sprintf("峰值:%s/s", formatBytes(traffic.PeakSendRate))
			peakStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)
			if i == selectedIf {
				peakStyle = peakStyle.Background(tcell.ColorDarkBlue)
			}
			drawTextWithStyle(screen, barX+barWidth+2, contentY, peakStr, peakStyle)
		}
		contentY++

		// RX (接收) 流量条
		rxLabel := fmt.Sprintf("  RX ◄ %-14s", formatBytes(traffic.RecvRate)+"/s")
		rxStyle := tcell.StyleDefault.Foreground(ui.ColorInfo)
		if i == selectedIf {
			rxStyle = rxStyle.Background(tcell.ColorDarkBlue)
		}
		drawTextWithStyle(screen, 4, contentY, rxLabel, rxStyle)

		// RX 进度条
		if barWidth > 0 && barWidth < 80 {
			maxBandwidth := 100.0 * 1024 * 1024
			rxPercent := (traffic.RecvRate / maxBandwidth) * 100
			if rxPercent > 100 {
				rxPercent = 100
			}
			drawProgressBar(screen, barX, contentY, barWidth, rxPercent, ui.ColorInfo)

			// 显示峰值
			peakStr := fmt.Sprintf("峰值:%s/s", formatBytes(traffic.PeakRecvRate))
			peakStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)
			if i == selectedIf {
				peakStyle = peakStyle.Background(tcell.ColorDarkBlue)
			}
			drawTextWithStyle(screen, barX+barWidth+2, contentY, peakStr, peakStyle)
		}
		contentY++

		// 累计流量
		totalStr := fmt.Sprintf("  累计: ↑%s  ↓%s",
			formatBytes(float64(traffic.TotalSent)),
			formatBytes(float64(traffic.TotalRecv)))
		totalStyle := tcell.StyleDefault.Foreground(ui.ColorMuted)
		if i == selectedIf {
			totalStyle = totalStyle.Background(tcell.ColorDarkBlue)
		}
		drawTextWithStyle(screen, 4, contentY, totalStr, totalStyle)
		contentY++

		// 分隔线
		if i < len(ifaceList)-1 && contentY < height-4 {
			separator := strings.Repeat("─", width-8)
			drawText(screen, 4, contentY, separator, ui.ColorMuted)
			contentY++
		}
	}

	// 底部全局统计面板
	statsY := height - 3
	ui.DrawDoubleHorizontalLine(screen, 3, statsY-1, width-6, ui.ColorSecondary)

	globalStats := fmt.Sprintf(" 全局统计: 总发送 %s | 总接收 %s | 丢包率 %.2f%% | 错误率 %.2f%% ",
		formatBytes(float64(data.TotalBytesSent)),
		formatBytes(float64(data.TotalBytesRecv)),
		data.PacketLossRate,
		data.ErrorRate)
	drawText(screen, 4, statsY, globalStats, tcell.ColorYellow)
}
