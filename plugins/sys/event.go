package sys

import (
	"github.com/gdamore/tcell/v2"
)

// handleEvent 处理事件
func handleEvent(s *SysMonitor, event *tcell.EventKey) bool {
	// 立即响应退出事件，不等待任何操作
	if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyCtrlC {
		s.cancel() // 停止数据更新循环
		return false
	}

	switch event.Key() {
	case tcell.KeyLeft:
		// 切换到上一个标签
		if s.currentTab > 0 {
			s.currentTab--
			s.selected = 0
		} else {
			s.currentTab = TabDisk // 最后一个标签
		}
		// 立即触发重绘，提升流畅度
		if s.uiManager != nil {
			s.uiManager.TriggerRefresh()
		}
		return true

	case tcell.KeyRight:
		// 切换到下一个标签
		if int(s.currentTab) < int(TabDisk) {
			s.currentTab++
			s.selected = 0
		} else {
			s.currentTab = TabOverview
		}
		// 立即触发重绘，提升流畅度
		if s.uiManager != nil {
			s.uiManager.TriggerRefresh()
		}
		return true

	case tcell.KeyUp:
		if s.selected > 0 {
			s.selected--
		}
		return true

	case tcell.KeyDown:
		s.mu.RLock()
		maxItems := 0
		switch s.currentTab {
		case TabProcesses:
			maxItems = len(s.data.Processes)
		}
		s.mu.RUnlock()
		if s.selected < maxItems-1 {
			s.selected++
		}
		return true

	case tcell.KeyRune:
		switch event.Rune() {
		case 'q', 'Q':
			s.cancel() // 停止数据更新循环
			return false
		case 'r', 'R':
			// 触发立即更新（异步）
			go func() {
				data := s.collector.collect()
				s.updateData(data)
			}()
			return true
		case 'c', 'C':
			// 按 CPU 排序
			if s.currentTab == TabProcesses {
				SetSortType(SortByCPU)
				return true
			}
		case 'm', 'M':
			// 按内存排序
			if s.currentTab == TabProcesses {
				SetSortType(SortByMemory)
				return true
			}
		case 'd', 'D':
			// 按磁盘 I/O 排序
			if s.currentTab == TabProcesses {
				SetSortType(SortByDiskIO)
				return true
			}
		case 't', 'T':
			// 按 CPU 时间排序
			if s.currentTab == TabProcesses {
				SetSortType(SortByCPUTime)
				return true
			}
		}
	}

	return true
}
