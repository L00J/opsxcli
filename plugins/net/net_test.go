package net

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- formatBytes ---

func TestFormatBytes_Zero(t *testing.T) {
	assert.Equal(t, "0B", formatBytes(0))
}

func TestFormatBytes_Bytes(t *testing.T) {
	assert.Equal(t, "512B", formatBytes(512))
}

func TestFormatBytes_KB(t *testing.T) {
	assert.Equal(t, "1.0KB", formatBytes(1024))
}

func TestFormatBytes_MB(t *testing.T) {
	assert.Equal(t, "1.5MB", formatBytes(1.5*1024*1024))
}

func TestFormatBytes_GB(t *testing.T) {
	assert.Equal(t, "2.0GB", formatBytes(2.0*1024*1024*1024))
}

// --- truncate ---

func TestTruncate_Short(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
}

func TestTruncate_Exact(t *testing.T) {
	assert.Equal(t, "12345", truncate("12345", 5))
}

func TestTruncate_Long(t *testing.T) {
	assert.Equal(t, "12...", truncate("1234567890", 5))
}

func TestTruncate_Empty(t *testing.T) {
	assert.Equal(t, "", truncate("", 5))
}

// --- parseAddr ---

func TestParseAddr_IPv4Loopback(t *testing.T) {
	// 0100007F:0050 = 127.0.0.1:80
	ip, port := parseAddr("0100007F:0050")
	assert.Equal(t, "127.0.0.1", ip)
	assert.Equal(t, uint16(80), port)
}

func TestParseAddr_IPv4Any(t *testing.T) {
	// 00000000:1F90 = 0.0.0.0:8080
	ip, port := parseAddr("00000000:1F90")
	assert.Equal(t, "0.0.0.0", ip)
	assert.Equal(t, uint16(8080), port)
}

func TestParseAddr_IPv4Normal(t *testing.T) {
	// 0100A8C0:0050 = 192.168.0.1:80
	ip, port := parseAddr("0100A8C0:0050")
	assert.Equal(t, "192.168.0.1", ip)
	assert.Equal(t, uint16(80), port)
}

func TestParseAddr_IPv6(t *testing.T) {
	// IPv6 addresses return "[IPv6]"
	ip, port := parseAddr("0000000000000000FFFF00000100A8C0:0050")
	assert.Equal(t, "[IPv6]", ip)
	assert.Equal(t, uint16(80), port)
}

func TestParseAddr_Invalid(t *testing.T) {
	ip, port := parseAddr("invalid")
	assert.Equal(t, "0.0.0.0", ip)
	assert.Equal(t, uint16(0), port)
}

func TestParseAddr_Empty(t *testing.T) {
	ip, port := parseAddr("")
	assert.Equal(t, "0.0.0.0", ip)
	assert.Equal(t, uint16(0), port)
}

func TestParseAddr_SpecificIP(t *testing.T) {
	// 0201A8C0 = 192.168.1.2 (little-endian)
	// C0=192, A8=168, 01=1, 02=2
	ip, port := parseAddr("0201A8C0:1BB")
	assert.Equal(t, "192.168.1.2", ip)
	assert.Equal(t, uint16(443), port)
}

// --- parseTCPState ---

func TestParseTCPState_Established(t *testing.T) {
	assert.Equal(t, "ESTABLISHED", parseTCPState("01"))
}

func TestParseTCPState_TimeWait(t *testing.T) {
	assert.Equal(t, "TIME_WAIT", parseTCPState("06"))
}

func TestParseTCPState_Listen(t *testing.T) {
	assert.Equal(t, "LISTEN", parseTCPState("0A"))
}

func TestParseTCPState_Close(t *testing.T) {
	assert.Equal(t, "CLOSE", parseTCPState("07"))
}

func TestParseTCPState_CloseWait(t *testing.T) {
	assert.Equal(t, "CLOSE_WAIT", parseTCPState("08"))
}

func TestParseTCPState_Unknown(t *testing.T) {
	assert.Equal(t, "UNKNOWN", parseTCPState("FF"))
}

func TestParseTCPState_Empty(t *testing.T) {
	assert.Equal(t, "", parseTCPState("00"))
}

func TestParseTCPState_SynSent(t *testing.T) {
	assert.Equal(t, "SYN_SENT", parseTCPState("02"))
}

func TestParseTCPState_SynRecv(t *testing.T) {
	assert.Equal(t, "SYN_RECV", parseTCPState("03"))
}

func TestParseTCPState_FinWait1(t *testing.T) {
	assert.Equal(t, "FIN_WAIT1", parseTCPState("04"))
}

func TestParseTCPState_FinWait2(t *testing.T) {
	assert.Equal(t, "FIN_WAIT2", parseTCPState("05"))
}

func TestParseTCPState_LastAck(t *testing.T) {
	assert.Equal(t, "LAST_ACK", parseTCPState("09"))
}

func TestParseTCPState_Closing(t *testing.T) {
	assert.Equal(t, "CLOSING", parseTCPState("0B"))
}

// --- SimpleConnectionTracker ---

func TestNewSimpleConnectionTracker(t *testing.T) {
	sct := NewSimpleConnectionTracker()
	assert.NotNil(t, sct)
	assert.NotNil(t, sct.connections)
	sct.Stop()
}

func TestSimpleConnectionTracker_GetTopConnectionsEmpty(t *testing.T) {
	sct := NewSimpleConnectionTracker()
	defer sct.Stop()
	conns := sct.GetTopConnections(10)
	assert.Empty(t, conns)
}

func TestSimpleConnectionTracker_GetTopConnectionsLimit(t *testing.T) {
	sct := NewSimpleConnectionTracker()
	defer sct.Stop()
	// Manually add connections to test sorting
	sct.mu.Lock()
	sct.connections["conn1"] = &SimpleConnectionStats{
		Key:      SimpleConnectionKey{LocalAddr: "127.0.0.1", LocalPort: 80, State: "ESTABLISHED"},
		RxQueue:  100,
		TxQueue:  200,
		Protocol: "TCP",
		LastSeen: time.Now(),
	}
	sct.connections["conn2"] = &SimpleConnectionStats{
		Key:      SimpleConnectionKey{LocalAddr: "127.0.0.1", LocalPort: 443, State: "ESTABLISHED"},
		RxQueue:  500,
		TxQueue:  500,
		Protocol: "TCP",
		LastSeen: time.Now(),
	}
	sct.connections["conn3"] = &SimpleConnectionStats{
		Key:      SimpleConnectionKey{LocalAddr: "127.0.0.1", LocalPort: 3306, State: "LISTEN"},
		RxQueue:  0,
		TxQueue:  0,
		Protocol: "TCP",
		LastSeen: time.Now(),
	}
	sct.mu.Unlock()

	conns := sct.GetTopConnections(2)
	assert.Len(t, conns, 2)
	// conn2 should be first (1000 total > 300 total)
	assert.Equal(t, uint16(443), conns[0].Key.LocalPort)
}

func TestSimpleConnectionTracker_GetTopConnectionsFilteredListen(t *testing.T) {
	sct := NewSimpleConnectionTracker()
	defer sct.Stop()
	sct.mu.Lock()
	sct.connections["conn1"] = &SimpleConnectionStats{
		Key:      SimpleConnectionKey{LocalAddr: "127.0.0.1", LocalPort: 80, State: "LISTEN"},
		RxQueue:  0,
		TxQueue:  0,
		Protocol: "TCP",
		LastSeen: time.Now(),
	}
	sct.mu.Unlock()

	conns := sct.GetTopConnections(10)
	// LISTEN with zero queues should be filtered out
	assert.Empty(t, conns)
}

// --- TabType constants ---

func TestTabTypeConstants(t *testing.T) {
	assert.Equal(t, TabType(0), TabRealtime)
	assert.Equal(t, TabType(1), TabConnections)
	assert.Equal(t, TabType(2), TabStatistics)
}

// --- Type construction ---

func TestInterfaceInfo(t *testing.T) {
	info := &InterfaceInfo{
		Name:     "eth0",
		MTU:      1500,
		IsUp:     true,
		BytesSent: 1000,
		BytesRecv: 2000,
	}
	assert.Equal(t, "eth0", info.Name)
	assert.True(t, info.IsUp)
	assert.Equal(t, uint64(1000), info.BytesSent)
}

func TestNetStats(t *testing.T) {
	stats := &NetStats{
		BytesSent:   100,
		BytesRecv:   200,
		PacketsSent: 10,
		PacketsRecv: 20,
		Timestamp:   time.Now(),
	}
	assert.Equal(t, uint64(100), stats.BytesSent)
	assert.Equal(t, uint64(20), stats.PacketsRecv)
}

func TestInterfaceTraffic(t *testing.T) {
	traffic := &InterfaceTraffic{
		Name:       "eth0",
		SendRate:   1024.5,
		RecvRate:   2048.0,
		PeakSendRate: 5000.0,
		PeakRecvRate: 8000.0,
	}
	assert.Equal(t, "eth0", traffic.Name)
	assert.Equal(t, 1024.5, traffic.SendRate)
	assert.Equal(t, 5000.0, traffic.PeakSendRate)
}
