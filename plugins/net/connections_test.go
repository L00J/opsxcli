package net

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// === SimpleConnectionTracker 文件读取测试 ===

// TestReadTCPConnections_ValidFile 从有效文件读取TCP连接
func TestReadTCPConnections_ValidFile(t *testing.T) {
	sct := NewSimpleConnectionTracker()
	defer sct.Stop()

	// 创建临时文件模拟 /proc/net/tcp
	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345
   1: 0100A8C0:1F90 0201A8C0:0050 01 00000064:000000C8 00:00000000 00000000  1000        0 67890
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tcp")
	err := os.WriteFile(tmpFile, []byte(content), 0600)
	assert.NoError(t, err)

	sct.connections = make(map[string]*SimpleConnectionStats)
	sct.readTCPConnections(tmpFile, "TCP")

	// 应解析出两个连接
	assert.Equal(t, 2, len(sct.connections))

	// 验证第一个连接（LISTEN状态）
	key1 := "127.0.0.1:80-0.0.0.0:0-TCP"
	conn1, ok1 := sct.connections[key1]
	assert.True(t, ok1, "应包含 LISTEN 连接")
	if ok1 {
		assert.Equal(t, "LISTEN", conn1.Key.State)
		assert.Equal(t, uint16(80), conn1.Key.LocalPort)
		assert.Equal(t, "TCP", conn1.Protocol)
	}

	// 验证第二个连接（ESTABLISHED状态）
	key2 := "192.168.0.1:8080-192.168.1.2:80-TCP"
	conn2, ok2 := sct.connections[key2]
	assert.True(t, ok2, "应包含 ESTABLISHED 连接")
	if ok2 {
		assert.Equal(t, "ESTABLISHED", conn2.Key.State)
		assert.Equal(t, uint16(8080), conn2.Key.LocalPort)
		assert.Equal(t, uint16(80), conn2.Key.RemotePort)
		// 验证队列解析: tx_queue=64, rx_queue=200 (十六进制)
		assert.Equal(t, uint64(100), conn2.RxQueue)
		assert.Equal(t, uint64(200), conn2.TxQueue)
	}
}

// TestReadTCPConnections_NonexistentFile 不存在的文件不崩溃
func TestReadTCPConnections_NonexistentFile(t *testing.T) {
	sct := NewSimpleConnectionTracker()
	defer sct.Stop()

	sct.connections = make(map[string]*SimpleConnectionStats)
	// 不存在的文件不应 panic
	sct.readTCPConnections("/proc/nonexistent/tcp", "TCP")
	assert.Empty(t, sct.connections)
}

// TestReadTCPConnections_ShortLines 字段不足的行被跳过
func TestReadTCPConnections_ShortLines(t *testing.T) {
	sct := NewSimpleConnectionTracker()
	defer sct.Stop()

	content := `  sl  local_address rem_address   st
   0: 0100007F 0050
   1: ab
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tcp")
	err := os.WriteFile(tmpFile, []byte(content), 0600)
	assert.NoError(t, err)

	sct.connections = make(map[string]*SimpleConnectionStats)
	sct.readTCPConnections(tmpFile, "TCP")

	// 字段不足的行应被跳过
	assert.Empty(t, sct.connections)
}

// TestReadUDPConnections_ValidFile 从有效文件读取UDP连接
func TestReadUDPConnections_ValidFile(t *testing.T) {
	sct := NewSimpleConnectionTracker()
	defer sct.Stop()

	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:14E9 00000000:0000 07 00000000:00000000 00:00000000 00000000  1000        0 12345
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "udp")
	err := os.WriteFile(tmpFile, []byte(content), 0600)
	assert.NoError(t, err)

	sct.connections = make(map[string]*SimpleConnectionStats)
	sct.readUDPConnections(tmpFile, "UDP")

	assert.Equal(t, 1, len(sct.connections))

	// UDP 连接状态应为 UNCONN
	for _, conn := range sct.connections {
		assert.Equal(t, "UNCONN", conn.Key.State)
		assert.Equal(t, "UDP", conn.Protocol)
		assert.Equal(t, "127.0.0.1", conn.Key.LocalAddr)
		assert.Equal(t, uint16(5353), conn.Key.LocalPort)
	}
}

// TestReadUDPConnections_NonexistentFile 不存在的文件不崩溃
func TestReadUDPConnections_NonexistentFile(t *testing.T) {
	sct := NewSimpleConnectionTracker()
	defer sct.Stop()

	sct.connections = make(map[string]*SimpleConnectionStats)
	sct.readUDPConnections("/proc/nonexistent/udp", "UDP")
	assert.Empty(t, sct.connections)
}

// === updateConnections 集成测试 ===

// TestUpdateConnections_NoProcFiles 无 /proc 文件时不崩溃（macOS 环境）
func TestUpdateConnections_NoProcFiles(t *testing.T) {
	sct := NewSimpleConnectionTracker()
	defer sct.Stop()

	// macOS 上通常没有 /proc/net/tcp，updateConnections 不应 panic
	sct.updateConnections()
	// 仅验证不 panic
}

// === SimpleConnectionTracker Start 测试 ===

// TestSimpleConnectionTracker_StartAndStop 启动和停止不崩溃
func TestSimpleConnectionTracker_StartAndStop(t *testing.T) {
	sct := NewSimpleConnectionTracker()
	sct.Start()

	// 等待一小段时间让 goroutine 运行
	time.Sleep(50 * time.Millisecond)

	sct.Stop()

	// 多次 Stop 不 panic
	sct.Stop()
}

// === parseAddr 额外边界测试 ===

// TestParseAddr_SingleColon 单冒号（无效格式）
func TestParseAddr_SingleColon(t *testing.T) {
	ip, port := parseAddr("invalid-no-colon")
	assert.Equal(t, "0.0.0.0", ip)
	assert.Equal(t, uint16(0), port)
}

// TestParseAddr_MultipleColons 多个冒号
func TestParseAddr_MultipleColons(t *testing.T) {
	ip, port := parseAddr("0100007F:0050:extra")
	assert.Equal(t, "0.0.0.0", ip)
	assert.Equal(t, uint16(0), port)
}

// TestParseAddr_IPv4ZeroPort 零端口
func TestParseAddr_IPv4ZeroPort(t *testing.T) {
	ip, port := parseAddr("0100007F:0000")
	assert.Equal(t, "127.0.0.1", ip)
	assert.Equal(t, uint16(0), port)
}

// TestParseAddr_LongIPv6 长IPv6地址
func TestParseAddr_LongIPv6(t *testing.T) {
	// 超过8字符但不是标准IPv4，应返回 [IPv6]
	ip, port := parseAddr("0000000000000000000000000100007F:0050")
	assert.Equal(t, "[IPv6]", ip)
	assert.Equal(t, uint16(80), port)
}

// === parseTCPState 完整覆盖测试 ===

// TestParseTCPState_AllStates 遍历所有已知状态
func TestParseTCPState_AllStates(t *testing.T) {
	expected := map[string]string{
		"00": "",
		"01": "ESTABLISHED",
		"02": "SYN_SENT",
		"03": "SYN_RECV",
		"04": "FIN_WAIT1",
		"05": "FIN_WAIT2",
		"06": "TIME_WAIT",
		"07": "CLOSE",
		"08": "CLOSE_WAIT",
		"09": "LAST_ACK",
		"0A": "LISTEN",
		"0B": "CLOSING",
		"0C": "UNKNOWN",
		"FF": "UNKNOWN",
	}

	for hex, expectedState := range expected {
		result := parseTCPState(hex)
		assert.Equal(t, expectedState, result, "parseTCPState(%s) 应返回 %s", hex, expectedState)
	}
}

// === formatDuration 边界测试补充 ===

// TestFormatDuration_LessThanSecond 不足一秒
func TestFormatDuration_LessThanSecond(t *testing.T) {
	result := formatDuration(500 * time.Millisecond)
	assert.Equal(t, "0秒", result)
}

// TestFormatDuration_ExactMinute 整分钟
func TestFormatDuration_ExactMinute(t *testing.T) {
	result := formatDuration(5 * time.Minute)
	assert.Equal(t, "5分0秒", result)
}

// === formatNumber 边界测试补充 ===

// TestFormatNumber_BoundaryK K边界
func TestFormatNumber_BoundaryK(t *testing.T) {
	assert.Equal(t, "999", formatNumber(999))
	assert.Equal(t, "1.0K", formatNumber(1000))
	assert.Equal(t, "1.5K", formatNumber(1500))
}

// TestFormatNumber_BoundaryM M边界
func TestFormatNumber_BoundaryM(t *testing.T) {
	assert.Equal(t, "999.9K", formatNumber(999900))
	assert.Equal(t, "1.0M", formatNumber(1000000))
}

// TestFormatNumber_BoundaryG G边界
func TestFormatNumber_BoundaryG(t *testing.T) {
	assert.Equal(t, "999.9M", formatNumber(999900000))
	assert.Equal(t, "1.0G", formatNumber(1000000000))
}

// TestFormatNumber_VeryLarge 极大值
func TestFormatNumber_VeryLarge(t *testing.T) {
	result := formatNumber(100000000000) // 100G
	assert.Contains(t, result, "G")
}
