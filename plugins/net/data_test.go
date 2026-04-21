package net

import (
	"context"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/stretchr/testify/assert"

	"opsxcli/plugins/netstat"
)

// --- getActiveConnectionsFromSS ---

func TestGetActiveConnectionsFromSS_Empty(t *testing.T) {
	result := getActiveConnectionsFromSS(nil)
	assert.Empty(t, result)
}

func TestGetActiveConnectionsFromSS_EmptySlice(t *testing.T) {
	conns := []netstat.SsConnection{}
	result := getActiveConnectionsFromSS(conns)
	assert.Empty(t, result)
}

func TestGetActiveConnectionsFromSS_AllListen(t *testing.T) {
	conns := []netstat.SsConnection{
		{State: "LISTEN", LocalAddr: "0.0.0.0", LocalPort: 80},
		{State: "LISTEN", LocalAddr: "0.0.0.0", LocalPort: 443},
	}
	result := getActiveConnectionsFromSS(conns)
	assert.Empty(t, result)
}

func TestGetActiveConnectionsFromSS_AllActive(t *testing.T) {
	conns := []netstat.SsConnection{
		{State: "ESTABLISHED", LocalAddr: "192.168.1.1", LocalPort: 80, RecvQ: 100, SendQ: 200},
		{State: "TIME_WAIT", LocalAddr: "192.168.1.2", LocalPort: 443, RecvQ: 50, SendQ: 50},
	}
	result := getActiveConnectionsFromSS(conns)
	assert.Len(t, result, 2)
}

func TestGetActiveConnectionsFromSS_Mixed(t *testing.T) {
	conns := []netstat.SsConnection{
		{State: "LISTEN", LocalAddr: "0.0.0.0", LocalPort: 80},
		{State: "ESTABLISHED", LocalAddr: "192.168.1.1", LocalPort: 8080, RecvQ: 100, SendQ: 200},
		{State: "LISTEN", LocalAddr: "0.0.0.0", LocalPort: 443},
		{State: "TIME_WAIT", LocalAddr: "10.0.0.1", LocalPort: 3000, RecvQ: 0, SendQ: 0},
	}
	result := getActiveConnectionsFromSS(conns)
	assert.Len(t, result, 2)
	// ESTABLISHED (100+200=300) should be first, TIME_WAIT (0+0=0) second
	assert.Equal(t, "ESTABLISHED", result[0].State)
	assert.Equal(t, "TIME_WAIT", result[1].State)
}

func TestGetActiveConnectionsFromSS_SortedByQueueSize(t *testing.T) {
	conns := []netstat.SsConnection{
		{State: "ESTABLISHED", LocalAddr: "a", RecvQ: 10, SendQ: 20},  // total 30
		{State: "ESTABLISHED", LocalAddr: "b", RecvQ: 500, SendQ: 500}, // total 1000
		{State: "ESTABLISHED", LocalAddr: "c", RecvQ: 100, SendQ: 100}, // total 200
	}
	result := getActiveConnectionsFromSS(conns)
	assert.Len(t, result, 3)
	// Should be sorted descending by RecvQ + SendQ
	assert.Equal(t, "b", result[0].LocalAddr)
	assert.Equal(t, "c", result[1].LocalAddr)
	assert.Equal(t, "a", result[2].LocalAddr)
}

func TestGetActiveConnectionsFromSS_VariousStates(t *testing.T) {
	conns := []netstat.SsConnection{
		{State: "LISTEN", LocalAddr: "0.0.0.0", LocalPort: 80},
		{State: "ESTABLISHED", LocalAddr: "a", RecvQ: 1, SendQ: 1},
		{State: "TIME_WAIT", LocalAddr: "b", RecvQ: 2, SendQ: 2},
		{State: "CLOSE_WAIT", LocalAddr: "c", RecvQ: 3, SendQ: 3},
		{State: "SYN_SENT", LocalAddr: "d", RecvQ: 4, SendQ: 4},
		{State: "FIN_WAIT1", LocalAddr: "e", RecvQ: 5, SendQ: 5},
		{State: "FIN_WAIT2", LocalAddr: "f", RecvQ: 6, SendQ: 6},
		{State: "CLOSING", LocalAddr: "g", RecvQ: 7, SendQ: 7},
		{State: "LAST_ACK", LocalAddr: "h", RecvQ: 8, SendQ: 8},
	}
	result := getActiveConnectionsFromSS(conns)
	// Only LISTEN should be filtered out
	assert.Len(t, result, 8)
	// First should be LAST_ACK (total 16), then CLOSING (14), etc.
	assert.Equal(t, "h", result[0].LocalAddr)
	assert.Equal(t, "g", result[1].LocalAddr)
}

// --- calculateTraffic additional tests ---

func TestCalculateTraffic_WithHistoryRates(t *testing.T) {
	ctx := context.Background()
	dc := NewDataCollector(ctx, 2*time.Second)

	// Set up history: 1 second ago, 1000 bytes sent, 2000 bytes recv
	dc.lastStats["eth0"] = &NetStats{
		BytesSent:   1000,
		BytesRecv:   2000,
		PacketsSent: 10,
		PacketsRecv: 20,
		Timestamp:   time.Now().Add(-1 * time.Second),
	}

	stat := &net.IOCountersStat{
		Name:        "eth0",
		BytesSent:   2000, // +1000 in 1 sec
		BytesRecv:   4000, // +2000 in 1 sec
		PacketsSent: 20,
		PacketsRecv: 40,
	}

	traffic := dc.calculateTraffic("eth0", stat)
	assert.Equal(t, uint64(2000), traffic.TotalSent)
	assert.Equal(t, uint64(4000), traffic.TotalRecv)
	// Rates should be approximately 1000 and 2000 bytes/sec
	assert.InDelta(t, 1000, traffic.SendRate, 200)
	assert.InDelta(t, 2000, traffic.RecvRate, 400)
}

func TestCalculateTraffic_PeakTracking(t *testing.T) {
	ctx := context.Background()
	dc := NewDataCollector(ctx, 2*time.Second)

	// First measurement: set initial peak
	dc.lastStats["eth0"] = &NetStats{
		BytesSent: 0,
		BytesRecv: 0,
		Timestamp: time.Now().Add(-1 * time.Second),
	}
	dc.peakStats["eth0"] = &PeakStats{
		PeakSendRate: 500,
		PeakRecvRate: 600,
	}

	stat := &net.IOCountersStat{
		Name:      "eth0",
		BytesSent: 1000, // rate = 1000/sec
		BytesRecv: 800,  // rate = 800/sec
	}

	traffic := dc.calculateTraffic("eth0", stat)
	// SendRate (~1000) > PeakSendRate (500), so PeakSendRate should be updated
	assert.GreaterOrEqual(t, traffic.PeakSendRate, 900.0)
	// RecvRate (~800) > PeakRecvRate (600), so PeakRecvRate should be updated
	assert.GreaterOrEqual(t, traffic.PeakRecvRate, 700.0)
}

func TestCalculateTraffic_NegativeRate(t *testing.T) {
	ctx := context.Background()
	dc := NewDataCollector(ctx, 2*time.Second)

	// Simulate counter reset: current bytes < last bytes
	dc.lastStats["eth0"] = &NetStats{
		BytesSent: 10000,
		BytesRecv: 20000,
		Timestamp: time.Now().Add(-1 * time.Second),
	}

	stat := &net.IOCountersStat{
		Name:      "eth0",
		BytesSent: 5000, // less than last
		BytesRecv: 3000, // less than last
	}

	traffic := dc.calculateTraffic("eth0", stat)
	// Negative rates should be clamped to 0
	assert.GreaterOrEqual(t, traffic.SendRate, float64(0))
	assert.GreaterOrEqual(t, traffic.RecvRate, float64(0))
}

func TestCalculateTraffic_StaleHistory(t *testing.T) {
	ctx := context.Background()
	dc := NewDataCollector(ctx, 2*time.Second)

	// Very old history (>10s) should be ignored
	dc.lastStats["eth0"] = &NetStats{
		BytesSent: 0,
		BytesRecv: 0,
		Timestamp: time.Now().Add(-30 * time.Second),
	}

	stat := &net.IOCountersStat{
		Name:      "eth0",
		BytesSent: 10000,
		BytesRecv: 20000,
	}

	traffic := dc.calculateTraffic("eth0", stat)
	// DeltaTime > 10s, rates should be 0
	assert.Equal(t, float64(0), traffic.SendRate)
	assert.Equal(t, float64(0), traffic.RecvRate)
}

// --- NewNetMonitor ---

func TestNewNetMonitor(t *testing.T) {
	monitor := NewNetMonitor()
	assert.NotNil(t, monitor)
	assert.Equal(t, TabRealtime, monitor.currentTab)
	assert.Equal(t, 0, monitor.selectedIf)
	assert.Equal(t, 2*time.Second, monitor.updateInterval)
	assert.NotNil(t, monitor.data)
	assert.NotNil(t, monitor.data.Interfaces)
	assert.NotNil(t, monitor.data.InterfaceTraffic)
	assert.NotNil(t, monitor.data.LastStats)
	assert.NotNil(t, monitor.data.CurrentStats)
	assert.NotNil(t, monitor.trafficHistory)
	assert.NotNil(t, monitor.ctx)
	assert.NotNil(t, monitor.collector)

	monitor.cancel() // cleanup
}

func TestNewNetMonitor_InitialState(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	assert.Equal(t, TabRealtime, monitor.currentTab)
	assert.Equal(t, 0, monitor.selectedIf)
	assert.NotNil(t, monitor.data)
	assert.Empty(t, monitor.data.Interfaces)
}

// --- handleEvent ---

func TestHandleEvent_Escape(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	screen := tcell.NewSimulationScreen("")
	screen.Init()
	evt := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)

	result := handleEvent(monitor, evt)
	assert.False(t, result) // should return false to exit
}

func TestHandleEvent_CtrlC(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	screen := tcell.NewSimulationScreen("")
	screen.Init()
	evt := tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)

	result := handleEvent(monitor, evt)
	assert.False(t, result)
}

func TestHandleEvent_TabKey(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	screen := tcell.NewSimulationScreen("")
	screen.Init()

	// Tab (same as Left in this code)
	evt := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	result := handleEvent(monitor, evt)
	assert.True(t, result)
	// TabRealtime -> TabStatistics
	assert.Equal(t, TabStatistics, monitor.currentTab)
}

func TestHandleEvent_RightKey(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	screen := tcell.NewSimulationScreen("")
	screen.Init()

	evt := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	result := handleEvent(monitor, evt)
	assert.True(t, result)
	assert.Equal(t, TabConnections, monitor.currentTab)

	// Press right again
	result = handleEvent(monitor, evt)
	assert.True(t, result)
	assert.Equal(t, TabStatistics, monitor.currentTab)

	// Press right again (wrap around)
	result = handleEvent(monitor, evt)
	assert.True(t, result)
	assert.Equal(t, TabRealtime, monitor.currentTab)
}

func TestHandleEvent_LeftKey(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	screen := tcell.NewSimulationScreen("")
	screen.Init()

	evt := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	result := handleEvent(monitor, evt)
	assert.True(t, result)
	// TabRealtime -> TabStatistics (wraps left)
	assert.Equal(t, TabStatistics, monitor.currentTab)

	// Press left again
	result = handleEvent(monitor, evt)
	assert.True(t, result)
	assert.Equal(t, TabConnections, monitor.currentTab)
}

func TestHandleEvent_UpAndDown(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	screen := tcell.NewSimulationScreen("")
	screen.Init()

	// Up from 0 should stay at 0
	evt := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	result := handleEvent(monitor, evt)
	assert.True(t, result)
	assert.Equal(t, 0, monitor.selectedIf)

	// Add some interfaces
	monitor.mu.Lock()
	monitor.data.Interfaces = []*InterfaceInfo{
		{Name: "eth0"},
		{Name: "eth1"},
		{Name: "eth2"},
	}
	monitor.mu.Unlock()

	// Down should increase
	evtDown := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	handleEvent(monitor, evtDown)
	assert.Equal(t, 1, monitor.selectedIf)
	handleEvent(monitor, evtDown)
	assert.Equal(t, 2, monitor.selectedIf)
	// Down at max should stay
	handleEvent(monitor, evtDown)
	assert.Equal(t, 2, monitor.selectedIf)

	// Up should decrease
	handleEvent(monitor, evt)
	assert.Equal(t, 1, monitor.selectedIf)
}

func TestHandleEvent_QKey(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	screen := tcell.NewSimulationScreen("")
	screen.Init()

	evt := tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)
	result := handleEvent(monitor, evt)
	assert.False(t, result) // should exit
}

func TestHandleEvent_RKey(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	screen := tcell.NewSimulationScreen("")
	screen.Init()

	evt := tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModNone)
	result := handleEvent(monitor, evt)
	assert.True(t, result) // should refresh, not exit
}

// --- NetworkData ---

func TestNetworkData(t *testing.T) {
	data := &NetworkData{
		Interfaces:       []*InterfaceInfo{{Name: "eth0"}},
		InterfaceTraffic: map[string]*InterfaceTraffic{"eth0": {}},
		LastStats:        map[string]*NetStats{},
		CurrentStats:     map[string]*NetStats{},
		TotalBytesSent:   1000,
		TotalBytesRecv:   2000,
		TotalPacketsSent: 100,
		TotalPacketsRecv: 200,
		TotalErrors:      5,
		TotalDrops:       10,
		PacketLossRate:   3.33,
		ErrorRate:        1.67,
		Platform:         "darwin",
	}
	assert.Len(t, data.Interfaces, 1)
	assert.Equal(t, uint64(1000), data.TotalBytesSent)
	assert.Equal(t, uint64(10), data.TotalDrops)
	assert.InDelta(t, 3.33, data.PacketLossRate, 0.01)
	assert.Equal(t, "darwin", data.Platform)
}
