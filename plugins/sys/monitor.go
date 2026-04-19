package sys

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"

	"opsxcli/internal/ui"
)

// SysMonitor 系统监控UI
type SysMonitor struct {
	width          int
	height         int
	currentTab     TabType
	selected       int
	updateInterval time.Duration
	mu             sync.RWMutex // 保护数据并发访问

	// 数据缓存
	data *SystemData

	// 控制更新
	ctx    context.Context
	cancel context.CancelFunc

	// 数据收集器
	collector *DataCollector

	// UI管理器引用（用于触发刷新）
	uiManager *ui.UIManager
}

// NewSysMonitor 创建系统监控实例
func NewSysMonitor() *SysMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	monitor := &SysMonitor{
		currentTab:     TabOverview,
		selected:       0,
		updateInterval: 2 * time.Second,                          // 每2秒更新一次（类似iftop/htop）
		data:           &SystemData{Processes: []*ProcessInfo{}}, // 初始化为空列表，避免 nil
		ctx:            ctx,
		cancel:         cancel,
		collector:      NewDataCollector(ctx, 2*time.Second),
	}

	// 延迟启动数据收集，在UI初始化后才开始收集（避免阻塞启动）
	// 将在 Run() 函数中调用 monitor.collector.Start()

	return monitor
}

// Run 运行系统监控
func Run() error {
	fmt.Println("opsxcli 智能运维助手")
	fmt.Println("正在初始化系统监控，预计 2-3 秒...")
	fmt.Println("按 Q 或 ESC 退出")
	fmt.Println()

	monitor := NewSysMonitor()
	defer monitor.cancel() // 确保清理资源

	manager, err := ui.NewUIManager(monitor)
	if err != nil {
		return err
	}
	defer manager.Finish()

	// 保存UI管理器引用，用于触发刷新
	monitor.uiManager = manager

	// UI初始化完成后，启动数据收集（避免阻塞启动）
	monitor.collector.Start(monitor.updateData)

	return manager.Run()
}

// Draw 绘制界面
func (s *SysMonitor) Draw(screen tcell.Screen) error {
	s.width, s.height = screen.Size()

	// 绘制标题栏
	drawHeader(screen, s.width)

	// 绘制标签栏
	drawTabs(screen, s.currentTab, s.width)

	// 根据当前标签绘制内容（数据已在后台异步更新）
	s.mu.RLock()
	data := s.data
	s.mu.RUnlock()

	switch s.currentTab {
	case TabOverview:
		drawOverview(screen, data, s.width, s.height)
	case TabProcesses:
		drawProcesses(screen, data, s.selected, s.width, s.height)
	case TabCPU:
		drawCPU(screen, data, s.width, s.height)
	case TabMemory:
		drawMemory(screen, data, s.width, s.height)
	case TabDisk:
		drawDisk(screen, data, s.width, s.height)
		// TabNetwork 已移除，使用 opsxcli net 查看详细网络信息
	}

	// 绘制底部帮助
	drawFooter(screen, s.width, s.height)

	return nil
}

// updateData 更新数据回调
func (s *SysMonitor) updateData(data *SystemData) {
	s.mu.Lock()
	s.data = data
	s.mu.Unlock()

	// 触发UI立即刷新（非阻塞）
	if s.uiManager != nil {
		s.uiManager.TriggerRefresh()
	}
}

// HandleEvent 处理事件
func (s *SysMonitor) HandleEvent(event *tcell.EventKey) bool {
	return handleEvent(s, event)
}

// Run 实现BaseUI接口
func (s *SysMonitor) Run(ctx context.Context) error {
	return nil
}
