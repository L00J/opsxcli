package net

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// === monitor.go 测试 ===

// TestNetMonitor_UpdateData 验证 updateData 回调更新数据
func TestNetMonitor_UpdateData(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	newData := &NetworkData{
		Interfaces: []*InterfaceInfo{
			{Name: "eth0", IsUp: true, BytesSent: 1000},
		},
		InterfaceTraffic: map[string]*InterfaceTraffic{
			"eth0": {Name: "eth0", SendRate: 512.0, RecvRate: 1024.0},
		},
		TotalBytesSent:   1000,
		TotalBytesRecv:   2000,
		TotalPacketsSent: 10,
		TotalPacketsRecv: 20,
		PacketLossRate:   0.5,
		ErrorRate:        0.1,
		Platform:         "darwin",
	}

	monitor.updateData(newData)

	monitor.mu.RLock()
	data := monitor.data
	monitor.mu.RUnlock()

	assert.Equal(t, "eth0", data.Interfaces[0].Name)
	assert.Equal(t, uint64(1000), data.TotalBytesSent)
	assert.Equal(t, uint64(2000), data.TotalBytesRecv)
	assert.InDelta(t, 0.5, data.PacketLossRate, 0.01)
}

// TestNetMonitor_UpdateTrafficHistory 验证流量历史更新和清理
func TestNetMonitor_UpdateTrafficHistory(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	entries := []*TrafficHistoryEntry{
		{
			SrcAddr: "192.168.1.1", SrcPort: 12345,
			DstAddr: "10.0.0.1", DstPort: 80,
			RecvKB: 100.0, SendKB: 50.0, TotalKB: 150.0,
			LastUpdate: time.Now(),
		},
		{
			SrcAddr: "192.168.1.2", SrcPort: 54321,
			DstAddr: "10.0.0.2", DstPort: 443,
			RecvKB: 200.0, SendKB: 100.0, TotalKB: 300.0,
			LastUpdate: time.Now(),
		},
	}

	monitor.updateTrafficHistory(entries)

	monitor.mu.RLock()
	history := monitor.trafficHistory
	monitor.mu.RUnlock()

	assert.Len(t, history, 2)

	// 验证 key 格式
	_, ok1 := history["192.168.1.1:12345->10.0.0.1:80"]
	assert.True(t, ok1, "应包含 192.168.1.1:12345->10.0.0.1:80 条目")

	_, ok2 := history["192.168.1.2:54321->10.0.0.2:443"]
	assert.True(t, ok2, "应包含 192.168.1.2:54321->10.0.0.2:443 条目")
}

// TestNetMonitor_UpdateTrafficHistory_Cleanup 验证60秒前的旧数据被清理
func TestNetMonitor_UpdateTrafficHistory_Cleanup(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	// 先添加一个旧条目（超过60秒）
	monitor.mu.Lock()
	monitor.trafficHistory["old:1234->remote:80"] = &TrafficHistoryEntry{
		SrcAddr:    "old",
		SrcPort:    1234,
		DstAddr:    "remote",
		DstPort:    80,
		RecvKB:     10.0,
		SendKB:     5.0,
		TotalKB:    15.0,
		LastUpdate: time.Now().Add(-120 * time.Second), // 120秒前，应被清理
	}
	monitor.mu.Unlock()

	// 添加新条目
	entries := []*TrafficHistoryEntry{
		{
			SrcAddr: "new", SrcPort: 5678,
			DstAddr: "target", DstPort: 443,
			RecvKB: 50.0, SendKB: 25.0, TotalKB: 75.0,
			LastUpdate: time.Now(),
		},
	}

	monitor.updateTrafficHistory(entries)

	monitor.mu.RLock()
	history := monitor.trafficHistory
	monitor.mu.RUnlock()

	// 旧条目应被清理，只保留新条目
	assert.Len(t, history, 1)
	_, hasNew := history["new:5678->target:443"]
	assert.True(t, hasNew, "应保留新条目")
}

// TestNetMonitor_UpdateTrafficHistory_Empty 空条目不崩溃
func TestNetMonitor_UpdateTrafficHistory_Empty(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	// 空切片不应 panic
	monitor.updateTrafficHistory(nil)
	monitor.updateTrafficHistory([]*TrafficHistoryEntry{})

	monitor.mu.RLock()
	history := monitor.trafficHistory
	monitor.mu.RUnlock()

	assert.Empty(t, history)
}

// TestNetMonitor_UpdateTrafficHistory_Overwrite 相同 key 的条目被覆盖
func TestNetMonitor_UpdateTrafficHistory_Overwrite(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	entry1 := &TrafficHistoryEntry{
		SrcAddr: "192.168.1.1", SrcPort: 80,
		DstAddr: "10.0.0.1", DstPort: 80,
		RecvKB: 100.0, SendKB: 50.0, TotalKB: 150.0,
		LastUpdate: time.Now(),
	}

	monitor.updateTrafficHistory([]*TrafficHistoryEntry{entry1})

	entry2 := &TrafficHistoryEntry{
		SrcAddr: "192.168.1.1", SrcPort: 80,
		DstAddr: "10.0.0.1", DstPort: 80,
		RecvKB: 200.0, SendKB: 100.0, TotalKB: 300.0,
		LastUpdate: time.Now(),
	}

	monitor.updateTrafficHistory([]*TrafficHistoryEntry{entry2})

	monitor.mu.RLock()
	history := monitor.trafficHistory
	monitor.mu.RUnlock()

	assert.Len(t, history, 1)
	assert.Equal(t, float64(300.0), history["192.168.1.1:80->10.0.0.1:80"].TotalKB)
}

// TestNetMonitor_HandleEvent_Delegates 验证 HandleEvent 委托给 handleEvent
func TestNetMonitor_HandleEvent_Delegates(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	// 测试 Q 键退出
	result := monitor.HandleEvent(newKeyEvent('q'))
	assert.False(t, result)
}

// TestNetMonitor_SelectedIfBounds 验证 selectedIf 不会越界
func TestNetMonitor_SelectedIfBounds(t *testing.T) {
	monitor := NewNetMonitor()
	defer monitor.cancel()

	// 初始值应为 0
	assert.Equal(t, 0, monitor.selectedIf)
}
