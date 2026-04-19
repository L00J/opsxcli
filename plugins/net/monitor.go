package net

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"

	"opsxcli/internal/ui"
)

// NetMonitor 网络监控UI(重新设计为类似iftop的流量监控)
type NetMonitor struct {
	width          int
	height         int
	currentTab     TabType
	selectedIf     int
	updateInterval time.Duration
	mu             sync.RWMutex

	// 数据缓存
	data *NetworkData

	// 控制更新
	ctx    context.Context
	cancel context.CancelFunc

	// 数据收集器
	collector *DataCollector

	// 简单连接追踪器(基于/proc,无需root)
	simpleConnTracker *SimpleConnectionTracker

	// 连接流量历史(保留60秒)
	trafficHistory map[string]*TrafficHistoryEntry

	// UI管理器引用（用于触发刷新）
	uiManager *ui.UIManager
}

// NewNetMonitor 创建网络监控实例(重新设计)
func NewNetMonitor() *NetMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	monitor := &NetMonitor{
		currentTab:     TabRealtime,
		selectedIf:     0,
		updateInterval: 2 * time.Second, // 改为2秒更新一次,减少CPU占用
		data: &NetworkData{
			Interfaces:       make([]*InterfaceInfo, 0),
			InterfaceTraffic: make(map[string]*InterfaceTraffic),
			LastStats:        make(map[string]*NetStats),
			CurrentStats:     make(map[string]*NetStats),
		},
		ctx:            ctx,
		cancel:         cancel,
		collector:      NewDataCollector(ctx, 2*time.Second),
		trafficHistory: make(map[string]*TrafficHistoryEntry),
	}

	// 延迟启动数据收集和连接追踪，在UI初始化后才开始（避免阻塞启动）
	// 将在 Run() 函数中调用

	return monitor
}

// Run 运行网络监控
func Run() error {
	fmt.Println("opsxcli 智能运维助手")
	fmt.Println("正在初始化网络监控，预计 2-3 秒...")
	fmt.Println("按 Q 或 ESC 退出")
	fmt.Println()

	monitor := NewNetMonitor()
	defer monitor.cancel()

	manager, err := ui.NewUIManager(monitor)
	if err != nil {
		return err
	}
	defer manager.Finish()

	// 保存UI管理器引用，用于触发刷新
	monitor.uiManager = manager

	// UI初始化完成后，启动数据收集和连接追踪（避免阻塞启动）
	monitor.collector.Start(monitor.updateData)

	// 启动简单连接追踪(基于/proc,不需要root权限)
	simpleTracker := NewSimpleConnectionTracker()
	simpleTracker.Start()
	monitor.simpleConnTracker = simpleTracker

	return manager.Run()
}

// Draw 绘制界面
func (n *NetMonitor) Draw(screen tcell.Screen) error {
	n.width, n.height = screen.Size()

	// 绘制标题栏
	drawHeader(screen, n.width)

	// 绘制标签栏
	drawTabs(screen, n.currentTab, n.width)

	// 根据当前标签绘制内容
	n.mu.RLock()
	data := n.data
	selectedIf := n.selectedIf
	n.mu.RUnlock()

	switch n.currentTab {
	case TabRealtime:
		// 实时流量监控(类似iftop)
		drawRealtime(screen, data, selectedIf, n.width, n.height)
	case TabConnections:
		// 连接综合仪表板(TIME_WAIT TOP + 并发IP TOP + 流量TOP)
		n.mu.RLock()
		trafficHistory := n.trafficHistory
		n.mu.RUnlock()
		drawConnectionsDashboard(screen, n.width, n.height, trafficHistory, n.updateTrafficHistory)
	case TabStatistics:
		// 流量统计(累计统计、质量指标)
		drawStatistics(screen, data, selectedIf, n.width, n.height)
	}

	// 绘制底部帮助
	drawFooter(screen, n.width, n.height, n.currentTab)

	return nil
}

// updateData 更新数据回调
func (n *NetMonitor) updateData(data *NetworkData) {
	n.mu.Lock()
	n.data = data
	n.mu.Unlock()

	// 触发UI立即刷新（非阻塞）
	if n.uiManager != nil {
		n.uiManager.TriggerRefresh()
	}
}

// HandleEvent 处理事件
func (n *NetMonitor) HandleEvent(event *tcell.EventKey) bool {
	return handleEvent(n, event)
}

// Run 实现BaseUI接口
func (n *NetMonitor) Run(ctx context.Context) error {
	return nil
}

// updateTrafficHistory 更新流量历史记录
func (n *NetMonitor) updateTrafficHistory(entries []*TrafficHistoryEntry) {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now()

	// 更新新数据
	for _, entry := range entries {
		key := fmt.Sprintf("%s:%d->%s:%d", entry.SrcAddr, entry.SrcPort, entry.DstAddr, entry.DstPort)
		n.trafficHistory[key] = entry
	}

	// 清理60秒前的旧数据
	for key, entry := range n.trafficHistory {
		if now.Sub(entry.LastUpdate) > 60*time.Second {
			delete(n.trafficHistory, key)
		}
	}
}
