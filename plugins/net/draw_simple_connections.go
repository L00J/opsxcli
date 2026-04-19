package net

import (
	"fmt"
	"sort"

	"opsxcli/internal/ui"
	"opsxcli/plugins/netstat"

	"github.com/gdamore/tcell/v2"
)

// drawSimpleConnections 绘制连接追踪界面(使用ss模块，高性能)
func drawSimpleConnections(screen tcell.Screen, tracker *SimpleConnectionTracker, width, height int) {
	y := 4

	// 绘制边框
	ui.DrawBox(screen, 2, y, width-4, height-y-2, " 活跃连接 (基于ss模块) ", ui.ColorSecondary)

	contentY := y + 2

	// 使用ss模块获取连接（显示所有TCP和UDP连接，不包括LISTEN状态）
	connections := getActiveConnectionsFromSS()

	if len(connections) == 0 {
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
	displayCount := len(connections)
	if displayCount > maxRows {
		displayCount = maxRows
	}

	for i := 0; i < displayCount; i++ {
		conn := connections[i]

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
		len(connections))
	drawText(screen, 4, statsY, stats, ui.ColorInfo)
}

// getActiveConnectionsFromSS 使用ss模块获取活跃连接
func getActiveConnectionsFromSS() []netstat.SsConnection {
	// 读取TCP和UDP连接，排除LISTEN状态（只显示活跃连接）
	// listen=false, all=true 表示显示所有非LISTEN状态的连接
	tcpConns := netstat.ReadTCPConnectionsWithPrograms(false, true, false)
	udpConns := netstat.ReadUDPConnectionsWithPrograms(false, true, false)

	// 合并连接
	connections := make([]netstat.SsConnection, 0, len(tcpConns)+len(udpConns))
	connections = append(connections, tcpConns...)
	connections = append(connections, udpConns...)

	// 过滤掉LISTEN状态的连接，只显示活跃连接
	activeConnections := make([]netstat.SsConnection, 0)
	for _, conn := range connections {
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
