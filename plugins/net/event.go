package net

import (
	"github.com/gdamore/tcell/v2"
)

// handleEvent 处理事件
func handleEvent(n *NetMonitor, event *tcell.EventKey) bool {
	// 立即响应退出事件
	if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyCtrlC {
		n.cancel()
		return false
	}

	switch event.Key() {
	case tcell.KeyLeft, tcell.KeyTab:
		// 切换到上一个标签(3个标签：流量、连接、统计)
		switch n.currentTab {
		case TabRealtime:
			n.currentTab = TabStatistics
		case TabConnections:
			n.currentTab = TabRealtime
		case TabStatistics:
			n.currentTab = TabConnections
		}
		n.selectedIf = 0
		// 立即触发重绘，提升流畅度
		if n.uiManager != nil {
			n.uiManager.TriggerRefresh()
		}
		return true

	case tcell.KeyRight:
		// 切换到下一个标签(3个标签：流量、连接、统计)
		switch n.currentTab {
		case TabRealtime:
			n.currentTab = TabConnections
		case TabConnections:
			n.currentTab = TabStatistics
		case TabStatistics:
			n.currentTab = TabRealtime
		}
		n.selectedIf = 0
		// 立即触发重绘，提升流畅度
		if n.uiManager != nil {
			n.uiManager.TriggerRefresh()
		}
		return true

	case tcell.KeyUp:
		if n.selectedIf > 0 {
			n.selectedIf--
		}
		return true

	case tcell.KeyDown:
		n.mu.RLock()
		maxItems := len(n.data.Interfaces)
		n.mu.RUnlock()
		if n.selectedIf < maxItems-1 && maxItems > 0 {
			n.selectedIf++
		}
		return true

	case tcell.KeyRune:
		switch event.Rune() {
		case 'q', 'Q':
			n.cancel()
			return false
		case 'r', 'R':
			// 刷新数据
			return true
		}
	}

	return true
}
