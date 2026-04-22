package net

import (
	"fmt"
	"sort"
	"time"

	"opsxcli/internal/ui"
	"opsxcli/plugins/netstat"

	"github.com/gdamore/tcell/v2"
)

// statPair 统计对(地址和计数)
type statPair struct {
	addr  string
	count int
}

// TrafficStat 流量统计
type TrafficStat struct {
	srcAddr string
	srcPort uint32
	dstAddr string
	dstPort uint32
	recvKB  float64
	sendKB  float64
	totalKB float64
}

// drawConnectionsDashboard 绘制连接综合仪表板
func drawConnectionsDashboard(screen tcell.Screen, width, height int, connections []netstat.SsConnection, trafficHistory map[string]*TrafficHistoryEntry, updateFunc func([]*TrafficHistoryEntry)) {
	y := 4

	if len(connections) == 0 {
		ui.DrawBox(screen, 2, y, width-4, height-y-2, " 连接监控 ", ui.ColorSecondary)
		drawText(screen, 4, y+2, "暂无连接数据...", ui.ColorMuted)
		return
	}

	// 统计TIME_WAIT
	timewaitMap := make(map[string]int)
	for _, conn := range connections {
		if conn.State == "TIME_WAIT" {
			timewaitMap[conn.ForeignAddr]++
		}
	}

	// 统计并发IP
	concurrentMap := make(map[string]int)
	for _, conn := range connections {
		if conn.ForeignAddr != "0.0.0.0" && conn.ForeignAddr != "" && conn.ForeignAddr != "*" {
			concurrentMap[conn.ForeignAddr]++
		}
	}

	// 收集所有活跃连接（ESTABLISHED状态）并更新历史
	now := time.Now()
	newEntries := make([]*TrafficHistoryEntry, 0)

	for _, conn := range connections {
		// 只记录已建立的连接，不管队列是否有数据
		if conn.State == "ESTABLISHED" {
			recvKB := float64(conn.RecvQ) / 1024.0
			sendKB := float64(conn.SendQ) / 1024.0

			// 优先使用 nettop 累计流量（macOS）
			if conn.BytesIn > 0 || conn.BytesOut > 0 {
				recvKB = float64(conn.BytesIn) / 1024.0
				sendKB = float64(conn.BytesOut) / 1024.0
			}

			entry := &TrafficHistoryEntry{
				SrcAddr:    conn.LocalAddr,
				SrcPort:    conn.LocalPort,
				DstAddr:    conn.ForeignAddr,
				DstPort:    conn.ForeignPort,
				RecvKB:     recvKB,
				SendKB:     sendKB,
				TotalKB:    recvKB + sendKB,
				LastUpdate: now,
			}
			newEntries = append(newEntries, entry)
		}
	}

	// 更新历史记录
	if updateFunc != nil {
		updateFunc(newEntries)
	}

	// 从历史记录中提取流量统计(包括刚刚的和之前的)
	trafficStats := make([]TrafficStat, 0)
	for _, entry := range trafficHistory {
		trafficStats = append(trafficStats, TrafficStat{
			srcAddr: entry.SrcAddr,
			srcPort: entry.SrcPort,
			dstAddr: entry.DstAddr,
			dstPort: entry.DstPort,
			recvKB:  entry.RecvKB,
			sendKB:  entry.SendKB,
			totalKB: entry.TotalKB,
		})
	}

	sort.Slice(trafficStats, func(i, j int) bool {
		return trafficStats[i].totalKB > trafficStats[j].totalKB
	})

	// 转换为排序列表
	timewaitList := make([]statPair, 0, len(timewaitMap))
	for addr, count := range timewaitMap {
		timewaitList = append(timewaitList, statPair{addr, count})
	}
	sort.Slice(timewaitList, func(i, j int) bool {
		return timewaitList[i].count > timewaitList[j].count
	})

	concurrentList := make([]statPair, 0, len(concurrentMap))
	for addr, count := range concurrentMap {
		concurrentList = append(concurrentList, statPair{addr, count})
	}
	sort.Slice(concurrentList, func(i, j int) bool {
		return concurrentList[i].count > concurrentList[j].count
	})

	// 计算布局
	halfWidth := (width - 6) / 2
	leftX := 2
	rightX := leftX + halfWidth + 2

	// 上半部分高度(TOP 10 各占一半高度)
	topHeight := (height - y - 4) * 4 / 10
	if topHeight < 8 {
		topHeight = 8
	}

	// 绘制左上: TIME_WAIT TOP 10
	drawTimeWaitTopBox(screen, leftX, y, halfWidth, topHeight, timewaitList)

	// 绘制右上: 并发IP TOP 10
	drawConcurrentIPTopBox(screen, rightX, y, halfWidth, topHeight, concurrentList)

	// 绘制下半: 流量 TOP 10
	trafficY := y + topHeight + 1
	trafficHeight := height - trafficY - 2
	drawTrafficTopBox(screen, leftX, trafficY, width-4, trafficHeight, trafficStats)
}

// drawTimeWaitTopBox 绘制TIME_WAIT TOP 10框
func drawTimeWaitTopBox(screen tcell.Screen, x, y, width, height int, stats []statPair) {
	ui.DrawBox(screen, x, y, width, height, " TIME_WAIT TOP10 ", ui.ColorSecondary)

	contentY := y + 2

	// 表头
	headerStyle := tcell.StyleDefault.Foreground(ui.ColorAccent).Bold(true)
	header := fmt.Sprintf("  %-4s %-25s %s", "排名", "目标地址", "连接数")
	drawTextWithStyle(screen, x+2, contentY, header, headerStyle)
	contentY++

	// 分隔线
	ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorSecondary)
	contentY++

	// 显示TOP 10
	limit := 10
	if len(stats) < limit {
		limit = len(stats)
	}

	maxRows := height - 6
	if limit > maxRows {
		limit = maxRows
	}

	if len(stats) == 0 {
		drawText(screen, x+2, contentY, "暂无TIME_WAIT连接", ui.ColorMuted)
	} else {
		for i := 0; i < limit; i++ {
			stat := stats[i]

			// 根据排名选择颜色
			color := ui.ColorText
			if i < 3 {
				color = ui.ColorDanger
			} else if i < 5 {
				color = ui.ColorWarning
			}

			addr := stat.addr
			if len(addr) > 24 {
				addr = addr[:24]
			}

			line := fmt.Sprintf("  %-4d %-25s %d", i+1, addr, stat.count)
			drawText(screen, x+2, contentY, line, color)
			contentY++
		}
	}
}

// drawConcurrentIPTopBox 绘制并发IP TOP 10框
func drawConcurrentIPTopBox(screen tcell.Screen, x, y, width, height int, stats []statPair) {
	ui.DrawBox(screen, x, y, width, height, " 并发IP TOP10 ", ui.ColorSecondary)

	contentY := y + 2

	// 表头
	headerStyle := tcell.StyleDefault.Foreground(ui.ColorAccent).Bold(true)
	header := fmt.Sprintf("  %-4s %-25s %s", "排名", "目标地址", "连接数")
	drawTextWithStyle(screen, x+2, contentY, header, headerStyle)
	contentY++

	// 分隔线
	ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorSecondary)
	contentY++

	// 显示TOP 10
	limit := 10
	if len(stats) < limit {
		limit = len(stats)
	}

	maxRows := height - 6
	if limit > maxRows {
		limit = maxRows
	}

	if len(stats) == 0 {
		drawText(screen, x+2, contentY, "暂无并发连接", ui.ColorMuted)
	} else {
		for i := 0; i < limit; i++ {
			stat := stats[i]

			// 根据排名选择颜色
			color := ui.ColorText
			if i < 3 {
				color = ui.ColorDanger
			} else if i < 5 {
				color = ui.ColorWarning
			}

			addr := stat.addr
			if len(addr) > 24 {
				addr = addr[:24]
			}

			line := fmt.Sprintf("  %-4d %-25s %d", i+1, addr, stat.count)
			drawText(screen, x+2, contentY, line, color)
			contentY++
		}
	}
}

// drawTrafficTopBox 绘制流量 TOP 10框
func drawTrafficTopBox(screen tcell.Screen, x, y, width, height int, stats []TrafficStat) {
	ui.DrawBox(screen, x, y, width, height, " 活跃连接 TOP10 ", ui.ColorSecondary)

	contentY := y + 2

	// 表头
	headerStyle := tcell.StyleDefault.Foreground(ui.ColorAccent).Bold(true)
	header := fmt.Sprintf("  %-4s %-28s %-28s %-8s %-8s %-8s",
		"排名", "源地址:端口", "目标地址:端口", "接收KB", "发送KB", "总流量")
	drawTextWithStyle(screen, x+2, contentY, header, headerStyle)
	contentY++

	// 分隔线
	ui.DrawHorizontalLine(screen, x, contentY, width, ui.ColorSecondary)
	contentY++

	// 显示TOP 10
	limit := 10
	if len(stats) < limit {
		limit = len(stats)
	}

	maxRows := height - 6
	if limit > maxRows {
		limit = maxRows
	}

	if len(stats) == 0 {
		drawText(screen, x+2, contentY, "暂无流量数据", ui.ColorMuted)
	} else {
		for i := 0; i < limit; i++ {
			stat := stats[i]

			// 根据流量大小选择颜色
			color := ui.ColorText
			if stat.totalKB > 100 {
				color = ui.ColorDanger
			} else if stat.totalKB > 10 {
				color = ui.ColorWarning
			} else if stat.totalKB > 1 {
				color = ui.ColorAccent
			}

			// 格式化地址
			srcAddr := fmt.Sprintf("%s:%d", stat.srcAddr, stat.srcPort)
			dstAddr := fmt.Sprintf("%s:%d", stat.dstAddr, stat.dstPort)

			if len(srcAddr) > 27 {
				srcAddr = srcAddr[:27]
			}
			if len(dstAddr) > 27 {
				dstAddr = dstAddr[:27]
			}

			line := fmt.Sprintf("  %-4d %-28s %-28s %8.2f %8.2f %8.2f",
				i+1, srcAddr, dstAddr, stat.recvKB, stat.sendKB, stat.totalKB)
			drawText(screen, x+2, contentY, line, color)
			contentY++
		}
	}
}
