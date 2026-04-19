package ui

import (
	"context"
	"time"

	"github.com/gdamore/tcell/v2"
)

// BaseUI 基础UI接口
type BaseUI interface {
	// Run 运行UI
	Run(ctx context.Context) error

	// Draw 绘制界面
	Draw(screen tcell.Screen) error

	// HandleEvent 处理事件
	HandleEvent(event *tcell.EventKey) bool
}

// UIManager UI管理器
type UIManager struct {
	screen      tcell.Screen
	ui          BaseUI
	ctx         context.Context
	cancel      context.CancelFunc
	refreshChan chan struct{} // 用于触发立即刷新
}

// NewUIManager 创建UI管理器
func NewUIManager(ui BaseUI) (*UIManager, error) {
	// 初始化tcell screen
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	if err := screen.Init(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &UIManager{
		screen:      screen,
		ui:          ui,
		ctx:         ctx,
		cancel:      cancel,
		refreshChan: make(chan struct{}, 1), // 缓冲通道，避免阻塞
	}, nil
}

// Run 运行UI
func (m *UIManager) Run() error {
	defer m.Finish()

	// 设置默认样式
	defStyle := tcell.StyleDefault.
		Background(ColorBackground).
		Foreground(ColorText)
	m.screen.SetStyle(defStyle)
	m.screen.Clear()

	// 启动事件循环
	eventChan := make(chan tcell.Event)
	go func() {
		for {
			event := m.screen.PollEvent()
			if event == nil {
				return
			}
			select {
			case eventChan <- event:
			case <-m.ctx.Done():
				return
			}
		}
	}()

	// 启动刷新循环
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return nil

		case event := <-eventChan:
			switch ev := event.(type) {
			case *tcell.EventKey:
				// 处理退出
				if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
					return nil
				}
				// 处理其他按键
				// HandleEvent 返回 false 表示需要退出，true 表示继续
				if !m.ui.HandleEvent(ev) {
					return nil // 退出
				}
				m.refresh()

			case *tcell.EventResize:
				m.refresh()
			}

		case <-ticker.C:
			m.refresh()

		case <-m.refreshChan:
			// 数据更新时立即刷新
			m.refresh()
		}
	}
}

// refresh 刷新界面
func (m *UIManager) refresh() {
	m.screen.Clear()
	if err := m.ui.Draw(m.screen); err != nil {
		return
	}
	m.screen.Show()
}

// Stop 停止UI
func (m *UIManager) Stop() {
	m.cancel()
}

// TriggerRefresh 触发立即刷新（非阻塞）
func (m *UIManager) TriggerRefresh() {
	select {
	case m.refreshChan <- struct{}{}:
	default:
		// 通道已满，跳过（避免阻塞）
	}
}

// Finish 完成并清理
func (m *UIManager) Finish() {
	m.screen.Fini()
}
