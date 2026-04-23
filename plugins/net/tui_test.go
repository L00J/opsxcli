package net

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

// === 辅助函数 ===

// newKeyEvent 创建按键事件
func newKeyEvent(ch rune) *tcell.EventKey {
	return tcell.NewEventKey(tcell.KeyRune, ch, tcell.ModNone)
}

// === Draw 方法测试 ===

// TestNetMonitor_Draw_RealtimeTab 验证实时流量标签页绘制不崩溃
func TestNetMonitor_Draw_RealtimeTab(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	// 设置测试数据
	monitor.updateData(&NetworkData{
		Interfaces: []*InterfaceInfo{
			{Name: "eth0", IsUp: true, BytesSent: 1000, BytesRecv: 2000},
		},
		InterfaceTraffic: map[string]*InterfaceTraffic{
			"eth0": {Name: "eth0", SendRate: 1024.0, RecvRate: 2048.0, TotalSent: 1000, TotalRecv: 2000},
		},
		LastStats:        make(map[string]*NetStats),
		CurrentStats:     make(map[string]*NetStats),
		TotalBytesSent:   1000,
		TotalBytesRecv:   2000,
		TotalPacketsSent: 10,
		TotalPacketsRecv: 20,
		Platform:         "darwin",
	})

	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(120, 40)

	// 绘制不应 panic
	err = monitor.Draw(screen)
	assert.NoError(t, err)
}

// TestNetMonitor_Draw_EmptyData 空数据时绘制不崩溃
func TestNetMonitor_Draw_EmptyData(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(80, 24)

	err = monitor.Draw(screen)
	assert.NoError(t, err)
}

// TestNetMonitor_Draw_ConnectionsTab 验证连接标签页绘制
func TestNetMonitor_Draw_ConnectionsTab(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()
	monitor.currentTab = TabConnections

	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(120, 40)

	err = monitor.Draw(screen)
	assert.NoError(t, err)
}

// TestNetMonitor_Draw_StatisticsTab 验证统计标签页绘制
func TestNetMonitor_Draw_StatisticsTab(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()
	monitor.currentTab = TabStatistics

	monitor.updateData(&NetworkData{
		Interfaces: []*InterfaceInfo{
			{Name: "eth0", IsUp: true, BytesSent: 50000, BytesRecv: 80000},
			{Name: "lo", IsUp: true, BytesSent: 1000, BytesRecv: 1000},
		},
		InterfaceTraffic: map[string]*InterfaceTraffic{
			"eth0": {Name: "eth0", SendRate: 5120.0, RecvRate: 10240.0, PeakSendRate: 10000.0, PeakRecvRate: 20000.0, TotalSent: 50000, TotalRecv: 80000},
			"lo":   {Name: "lo", SendRate: 0, RecvRate: 0, TotalSent: 1000, TotalRecv: 1000},
		},
		LastStats:        make(map[string]*NetStats),
		CurrentStats:     make(map[string]*NetStats),
		TotalBytesSent:   50000,
		TotalBytesRecv:   80000,
		TotalPacketsSent: 500,
		TotalPacketsRecv: 800,
		TotalErrors:      2,
		TotalDrops:       5,
		PacketLossRate:   0.38,
		ErrorRate:        0.15,
		Platform:         "linux",
	})

	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(120, 40)

	err = monitor.Draw(screen)
	assert.NoError(t, err)
}

// TestNetMonitor_Draw_AllTabs 切换所有标签页绘制不崩溃
func TestNetMonitor_Draw_AllTabs(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(100, 30)

	tabs := []TabType{TabRealtime, TabConnections, TabStatistics}
	for _, tab := range tabs {
		monitor.currentTab = tab
		err = monitor.Draw(screen)
		assert.NoError(t, err, "绘制标签页 %d 不应出错", tab)
	}
}

// TestDrawRealtime_NoInterfaces 无接口时不崩溃
func TestDrawRealtime_NoInterfaces(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(80, 24)

	data := &NetworkData{
		Interfaces:       []*InterfaceInfo{},
		InterfaceTraffic: make(map[string]*InterfaceTraffic),
	}

	// 不应 panic
	drawRealtime(screen, data, 0, 80, 24)
}

// TestDrawRealtime_WithInterfaces 有接口时绘制完整界面
func TestDrawRealtime_WithInterfaces(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(120, 40)

	data := &NetworkData{
		Interfaces: []*InterfaceInfo{
			{Name: "eth0", IsUp: true, BytesSent: 100000, BytesRecv: 200000, Addrs: []string{"192.168.1.100/24"}},
			{Name: "eth1", IsUp: false, BytesSent: 0, BytesRecv: 0},
		},
		InterfaceTraffic: map[string]*InterfaceTraffic{
			"eth0": {Name: "eth0", SendRate: 50000.0, RecvRate: 100000.0, PeakSendRate: 80000.0, PeakRecvRate: 150000.0, TotalSent: 100000, TotalRecv: 200000},
			"eth1": {Name: "eth1", SendRate: 0, RecvRate: 0, TotalSent: 0, TotalRecv: 0},
		},
		TotalBytesSent: 100000,
		TotalBytesRecv: 200000,
		PacketLossRate: 0.1,
		ErrorRate:      0.05,
	}

	drawRealtime(screen, data, 0, 120, 40)
}

// TestDrawStatistics_NoInterfaces 无接口时不崩溃
func TestDrawStatistics_NoInterfaces(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(80, 24)

	data := &NetworkData{
		Interfaces:       []*InterfaceInfo{},
		InterfaceTraffic: make(map[string]*InterfaceTraffic),
		StartTime:        time.Now().Add(-5 * time.Minute),
	}

	drawStatistics(screen, data, 0, 80, 24)
}

// TestDrawProgressBar 边界测试
func TestDrawProgressBar_Over100(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(80, 24)

	// percent > 100 不应 panic
	drawProgressBar(screen, 0, 0, 20, 150.0, tcell.ColorGreen)
}

// TestDrawProgressBar_Negative 负值不崩溃
func TestDrawProgressBar_Negative(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(80, 24)

	// 负 percent 不应 panic
	drawProgressBar(screen, 0, 0, 20, -10.0, tcell.ColorGreen)
}

// TestDrawProgressBar_ZeroWidth 零宽度不崩溃
func TestDrawProgressBar_ZeroWidth(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(80, 24)

	// 宽度为 0 不应 panic
	drawProgressBar(screen, 0, 0, 0, 50.0, tcell.ColorGreen)
}

// TestDrawHeader 不崩溃
func TestDrawHeader(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(120, 24)

	// 不应 panic
	drawHeader(screen, 120)
	drawHeader(screen, 40) // 窄屏幕
	drawHeader(screen, 20) // 极窄
}

// TestDrawTabs 不崩溃
func TestDrawTabs(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(120, 24)

	drawTabs(screen, TabRealtime, 120)
	drawTabs(screen, TabConnections, 120)
	drawTabs(screen, TabStatistics, 120)
	drawTabs(screen, TabRealtime, 20) // 窄屏
}

// TestDrawFooter 不崩溃
func TestDrawFooter(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(120, 24)

	drawFooter(screen, 120, 24, TabRealtime)
	drawFooter(screen, 40, 10, TabConnections)
}

// TestDrawTextAndStyle 文本绘制测试
func TestDrawTextAndStyle(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(80, 24)

	// 普通文本绘制
	drawText(screen, 0, 0, "Hello", tcell.ColorWhite)

	// 样式文本绘制
	drawTextWithStyle(screen, 0, 1, "World", tcell.StyleDefault.Foreground(tcell.ColorGreen))

	// 空文本
	drawText(screen, 0, 2, "", tcell.ColorWhite)

	// 超长文本（超出屏幕宽度）
	drawText(screen, 0, 3, "This is a very long text that should be clipped by the screen width boundary", tcell.ColorWhite)

	// 中文文本
	drawText(screen, 0, 4, "中文测试", tcell.ColorWhite)
}

// === draw_stats_left.go 测试（通过 drawStatistics 间接测试）===

// TestDrawStatistics_WithQualityMetrics 有质量指标时绘制
func TestDrawStatistics_WithQualityMetrics(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(120, 40)

	data := &NetworkData{
		Interfaces: []*InterfaceInfo{
			{Name: "eth0", IsUp: true, BytesSent: 100000, BytesRecv: 200000},
		},
		InterfaceTraffic: map[string]*InterfaceTraffic{
			"eth0": {Name: "eth0", SendRate: 1024.0, RecvRate: 2048.0, PeakSendRate: 5000.0, PeakRecvRate: 10000.0, TotalSent: 100000, TotalRecv: 200000},
		},
		TotalBytesSent:   100000,
		TotalBytesRecv:   200000,
		TotalPacketsSent: 1000,
		TotalPacketsRecv: 2000,
		TotalErrors:      50,
		TotalDrops:       30,
		PacketLossRate:   1.5, // 高丢包率
		ErrorRate:        2.0, // 高错误率
		StartTime:        time.Now().Add(-10 * time.Minute),
		ConnectionStats: map[string]int{
			"ESTABLISHED": 150,  // 高连接数
			"TIME_WAIT":   6000, // 高 TIME_WAIT
			"LISTEN":      10,
		},
	}

	drawStatistics(screen, data, 0, 120, 40)
}

// TestDrawStatistics_NormalQuality 正常质量指标
func TestDrawStatistics_NormalQuality(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	assert.NoError(t, err)
	screen.SetSize(120, 40)

	data := &NetworkData{
		Interfaces: []*InterfaceInfo{
			{Name: "en0", IsUp: true},
		},
		InterfaceTraffic: map[string]*InterfaceTraffic{
			"en0": {Name: "en0", SendRate: 500.0, RecvRate: 800.0, TotalSent: 10000, TotalRecv: 20000},
		},
		TotalPacketsSent: 100,
		TotalPacketsRecv: 200,
		PacketLossRate:   0.01,
		ErrorRate:        0.005,
		StartTime:        time.Now().Add(-1 * time.Hour),
		ConnectionStats: map[string]int{
			"ESTABLISHED": 50,
			"TIME_WAIT":   100,
			"LISTEN":      5,
		},
	}

	drawStatistics(screen, data, 0, 120, 40)
}
