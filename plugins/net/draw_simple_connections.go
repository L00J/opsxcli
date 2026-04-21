package net

import (
	"fmt"
	"sort"

	"opsxcli/internal/ui"
	"opsxcli/plugins/netstat"

	"github.com/gdamore/tcell/v2"
)

// drawSimpleConnections 绘制连接追踪界面(使用ss模块，高性能)
func drawSimpleConnections(screen tcell.Screen, tracker *SimpleConnectionTracker, connections []netstat.SsConnection, width, height int) {
	y := 4

	// 绘制边框
	ui.DrawBox(screen, 2, y, width-4, height-y-2, " 活跃连接 (基于ss模块) ", ui.ColorSecondary)

	contentY := y + 2

	// 使用缓存的连接数据
	activeConnections := getActiveConnectionsFromSS(connections)

	if len(activeConnections) == 0 {
		drawText(screen, 4, contentY, "暂无活跃连接...", ui.ColorMuted)
		return
	}

	// 表头
	headerStyle := tcell.StyleDefault.
		Foreground(ui.ColorAccent).
		Bold(true)

	header := fmt.Sprintf("  %-6s %-35s %-35s %-12s %-8s %-8s",
		"协议", "本地地址:端口", "远程地址:端口", "状态", "接收队列", "发送队列")
	drawTextWithStyle(screen, 4, contentY, header, headerStyle)

	// 分隔线
	ui.DrawHorizontalLine(screen, 2, contentY+1, width-4, ui.ColorSecondary)
	contentY += 2

	// 显示连接列表
	maxRows := height - contentY - 2
	displayCount := len(activeConnections)
	if displayCount > maxRows {
		displayCount = maxRows
	}

	for i := 0; i < displayCount; i++ {
		conn := activeConnections[i]

		// 格式化本地地址
		localAddr := fmt.Sprintf("%s:%d", conn.LocalAddr, conn.LocalPort)
		if len(localAddr) > 34 {
			localAddr = localAddr[:34]
		}

		// 格式化远程地址
		remoteAddr := fmt.Sprintf("%s:%d", conn.ForeignAddr, conn.ForeignPort)
		if len(remoteAddr) > 34 {
			remoteAddr = remoteAddr[:34]
		}

		// 根据状态选择颜色
		color := ui.ColorText
		if conn.State == "ESTABLISHED" {
			color = ui.ColorSuccess
		} else if conn.State == "TIME_WAIT" || conn.State == "CLOSE_WAIT" {
			color = ui.ColorWarning
		} else if conn.State == "LISTEN" {
			color = ui.ColorInfo
		}

		// 显示连接
		line := fmt.Sprintf("  %-6s %-35s %-35s %-12s %8d %8d",
			conn.Proto,
			localAddr,
			remoteAddr,
			conn.State,
			conn.RecvQ,
			conn.SendQ)

		drawText(screen, 4, contentY, line, color)
		contentY++
	}

	// 底部统计
	statsY := height - 2
	ui.DrawDoubleHorizontalLine(screen, 3, statsY-1, width-6, ui.ColorSecondary)

	stats := fmt.Sprintf(" 活跃连接: %d | 提示: 使用ss模块高性能读取 /proc/net (无需root权限)",
		len(activeConnections))
	drawText(screen, 4, statsY, stats, ui.ColorInfo)
}

// getActiveConnectionsFromSS 从缓存的连接数据中过滤活跃连接
func getActiveConnectionsFromSS(allConnections []netstat.SsConnection) []netstat.SsConnection {
	// 过滤掉LISTEN状态的连接，只显示活跃连接
	activeConnections := make([]netstat.SsConnection, 0)
	for _, conn := range allConnections {
		if conn.State != "LISTEN" {
			activeConnections = append(activeConnections, conn)
		}
	}

	// 按队列大小排序（显示流量最大的连接）
	sort.Slice(activeConnections, func(i, j int) bool {
		totalI := activeConnections[i].RecvQ + activeConnections[i].SendQ
		totalJ := activeConnections[j].RecvQ + activeConnections[j].SendQ
		return totalI > totalJ
	})

	return activeConnections
}
