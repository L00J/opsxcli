package netstat

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// === readTCPFileWithPrograms 测试（使用临时文件模拟 /proc/net/tcp）===

func TestReadTCPFileWithPrograms_FileNotFound(t *testing.T) {
	conns := readTCPFileWithPrograms("/proc/net/tcp_nonexistent", "TCP", false, false, false)
	assert.Empty(t, conns)
}

func TestReadTCPFileWithPrograms_ValidFile(t *testing.T) {
	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345 1 0000000000000000 100 0 0 10 0
   1: 0101A8C0:1F90 0100000A:0050 01 00000000:00000000 00:00000000 00000000  1000        0 67890 1 0000000000000000 20 0 0 10 -1
`
	tmpFile, err := os.CreateTemp("", "proc_net_tcp_*")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// all=true 显示所有连接
	conns := readTCPFileWithPrograms(tmpFile.Name(), "TCP", false, true, false)
	assert.Len(t, conns, 2)

	// 第一个是 LISTEN 状态
	assert.Equal(t, "TCP", conns[0].Proto)
	assert.Equal(t, "127.0.0.1", conns[0].LocalAddr)
	assert.Equal(t, uint32(80), conns[0].LocalPort)
	assert.Equal(t, "0.0.0.0", conns[0].ForeignAddr)
	assert.Equal(t, uint32(0), conns[0].ForeignPort)
	assert.Equal(t, "LISTEN", conns[0].State)

	// 第二个是 ESTABLISHED 状态
	assert.Equal(t, "TCP", conns[1].Proto)
	assert.Equal(t, "192.168.1.1", conns[1].LocalAddr)
	assert.Equal(t, uint32(8080), conns[1].LocalPort)
	assert.Equal(t, "10.0.0.1", conns[1].ForeignAddr)
	assert.Equal(t, uint32(80), conns[1].ForeignPort)
	assert.Equal(t, "ESTABLISHED", conns[1].State)
}

func TestReadTCPFileWithPrograms_ListenFilter(t *testing.T) {
	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345 1 0000000000000000 100 0 0 10 0
   1: 0101A8C0:1F90 0100000A:0050 01 00000000:00000000 00:00000000 00000000  1000        0 67890 1 0000000000000000 20 0 0 10 -1
`
	tmpFile, err := os.CreateTemp("", "proc_net_tcp_*")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// listen=true 只看 LISTEN 状态
	conns := readTCPFileWithPrograms(tmpFile.Name(), "TCP", true, false, false)
	assert.Len(t, conns, 1)
	assert.Equal(t, "LISTEN", conns[0].State)
}

func TestReadTCPFileWithPrograms_DefaultFilter(t *testing.T) {
	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345 1 0000000000000000 100 0 0 10 0
   1: 0101A8C0:1F90 0100000A:0050 01 00000000:00000000 00:00000000 00000000  1000        0 67890 1 0000000000000000 20 0 0 10 -1
`
	tmpFile, err := os.CreateTemp("", "proc_net_tcp_*")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// 默认模式（listen=false, all=false）→ 过滤掉 LISTEN
	conns := readTCPFileWithPrograms(tmpFile.Name(), "TCP", false, false, false)
	assert.Len(t, conns, 1)
	assert.Equal(t, "ESTABLISHED", conns[0].State)
}

func TestReadTCPFileWithPrograms_ShortLines(t *testing.T) {
	content := `  sl  local_address rem_address   st
   0: 0100007F 0050
   1: a b
`
	tmpFile, err := os.CreateTemp("", "proc_net_tcp_*")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	conns := readTCPFileWithPrograms(tmpFile.Name(), "TCP", false, true, false)
	assert.Empty(t, conns)
}

func TestReadTCPFileWithPrograms_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "proc_net_tcp_*")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := "  sl  local_address rem_address   st tx_queue rx_queue\n"
	err = os.WriteFile(tmpFile.Name(), []byte(content), 0644)
	assert.NoError(t, err)

	conns := readTCPFileWithPrograms(tmpFile.Name(), "TCP", false, true, false)
	assert.Empty(t, conns)
}

func TestReadTCPFileWithPrograms_QueueInfo(t *testing.T) {
	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0050 00000000:0000 0A 100:200 00:00000000 00000000     0        0 12345 1 0000000000000000 100 0 0 10 0
`
	tmpFile, err := os.CreateTemp("", "proc_net_tcp_*")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	conns := readTCPFileWithPrograms(tmpFile.Name(), "TCP", false, true, false)
	assert.Len(t, conns, 1)
	// 0x100 = 256 (RecvQ), 0x200 = 512 (SendQ)
	assert.Equal(t, uint64(256), conns[0].RecvQ)
	assert.Equal(t, uint64(512), conns[0].SendQ)
}

// === readUDPFileWithPrograms 测试 ===

func TestReadUDPFileWithPrograms_FileNotFound(t *testing.T) {
	conns := readUDPFileWithPrograms("/proc/net/udp_nonexistent", "UDP", false, false, false)
	assert.Empty(t, conns)
}

func TestReadUDPFileWithPrograms_ValidFile(t *testing.T) {
	// 0x14E9 = 5353, 0x0035 = 53
	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:14E9 00000000:0000 07 00000000:00000000 00:00000000 00000000  1000        0 12345 1 0000000000000000 100 0 0 10 0
   1: 00000000:14E9 0100000A:0035 07 00000000:00000000 00:00000000 00000000  1000        0 67890 1 0000000000000000 20 0 0 10 -1
`
	tmpFile, err := os.CreateTemp("", "proc_net_udp_*")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// all=true 显示所有
	conns := readUDPFileWithPrograms(tmpFile.Name(), "UDP", false, true, false)
	assert.Len(t, conns, 2)

	// 第一个：127.0.0.1:5353 → 远程 0.0.0.0:0 → UNCONN
	assert.Equal(t, "UDP", conns[0].Proto)
	assert.Equal(t, "127.0.0.1", conns[0].LocalAddr)
	assert.Equal(t, uint32(5353), conns[0].LocalPort)
	assert.Equal(t, "0.0.0.0", conns[0].ForeignAddr)
	assert.Equal(t, uint32(0), conns[0].ForeignPort)
	assert.Equal(t, "UNCONN", conns[0].State)

	// 第二个：0.0.0.0:5353 → 远程 10.0.0.1:53 → ESTAB
	assert.Equal(t, "0.0.0.0", conns[1].LocalAddr)
	assert.Equal(t, "10.0.0.1", conns[1].ForeignAddr)
	assert.Equal(t, uint32(53), conns[1].ForeignPort)
	assert.Equal(t, "ESTAB", conns[1].State)
}

func TestReadUDPFileWithPrograms_ListenFilter(t *testing.T) {
	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:14E9 00000000:0000 07 00000000:00000000 00:00000000 00000000  1000        0 12345 1 0000000000000000 100 0 0 10 0
   1: 00000000:14E9 0100000A:0035 07 00000000:00000000 00:00000000 00000000  1000        0 67890 1 0000000000000000 20 0 0 10 -1
`
	tmpFile, err := os.CreateTemp("", "proc_net_udp_*")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// listen=true 只看 UNCONN
	conns := readUDPFileWithPrograms(tmpFile.Name(), "UDP", true, false, false)
	assert.Len(t, conns, 1)
	assert.Equal(t, "UNCONN", conns[0].State)
}

func TestReadUDPFileWithPrograms_DefaultFilter(t *testing.T) {
	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:14E9 00000000:0000 07 00000000:00000000 00:00000000 00000000  1000        0 12345 1 0000000000000000 100 0 0 10 0
   1: 00000000:14E9 0100000A:0035 07 00000000:00000000 00:00000000 00000000  1000        0 67890 1 0000000000000000 20 0 0 10 -1
`
	tmpFile, err := os.CreateTemp("", "proc_net_udp_*")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	// 默认模式过滤掉 UNCONN
	conns := readUDPFileWithPrograms(tmpFile.Name(), "UDP", false, false, false)
	assert.Len(t, conns, 1)
	assert.Equal(t, "ESTAB", conns[0].State)
}

func TestReadUDPFileWithPrograms_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "proc_net_udp_*")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := "  sl  local_address rem_address   st tx_queue rx_queue\n"
	err = os.WriteFile(tmpFile.Name(), []byte(content), 0644)
	assert.NoError(t, err)

	conns := readUDPFileWithPrograms(tmpFile.Name(), "UDP", false, true, false)
	assert.Empty(t, conns)
}

// === printStateStats 测试 ===

func TestPrintStateStats(t *testing.T) {
	conns := []SsConnection{
		{State: "ESTABLISHED"},
		{State: "ESTABLISHED"},
		{State: "TIME_WAIT"},
		{State: "LISTEN"},
		{State: "LISTEN"},
		{State: "LISTEN"},
	}

	buf := captureOutput(func() {
		err := printStateStats(conns)
		assert.NoError(t, err)
	})

	assert.Contains(t, buf, "ESTABLISHED")
	assert.Contains(t, buf, "TIME_WAIT")
	assert.Contains(t, buf, "LISTEN")
	assert.Contains(t, buf, "TCP状态统计")
}

func TestPrintStateStats_Empty(t *testing.T) {
	buf := captureOutput(func() {
		err := printStateStats([]SsConnection{})
		assert.NoError(t, err)
	})

	assert.Contains(t, buf, "TCP状态统计")
}

// === printTimeWaitStats 测试 ===

func TestPrintTimeWaitStats_NoTimeWait(t *testing.T) {
	conns := []SsConnection{
		{State: "ESTABLISHED", ForeignAddr: "10.0.0.1"},
		{State: "LISTEN", ForeignAddr: "0.0.0.0"},
	}

	buf := captureOutput(func() {
		err := printTimeWaitStats(conns)
		assert.NoError(t, err)
	})

	assert.Contains(t, buf, "没有TIME_WAIT")
}

func TestPrintTimeWaitStats_WithTimeWait(t *testing.T) {
	conns := []SsConnection{
		{State: "TIME_WAIT", ForeignAddr: "10.0.0.1"},
		{State: "TIME_WAIT", ForeignAddr: "10.0.0.1"},
		{State: "TIME_WAIT", ForeignAddr: "192.168.1.1"},
	}

	buf := captureOutput(func() {
		err := printTimeWaitStats(conns)
		assert.NoError(t, err)
	})

	assert.Contains(t, buf, "10.0.0.1")
	assert.Contains(t, buf, "192.168.1.1")
	assert.Contains(t, buf, "TIME_WAIT")
}

// === printTopDestinations 测试 ===

func TestPrintTopDestinations_NoDestinations(t *testing.T) {
	conns := []SsConnection{
		{ForeignAddr: "0.0.0.0"},
		{ForeignAddr: ""},
	}

	buf := captureOutput(func() {
		err := printTopDestinations(conns, 10)
		assert.NoError(t, err)
	})

	assert.Contains(t, buf, "没有找到目标地址")
}

func TestPrintTopDestinations_WithDestinations(t *testing.T) {
	conns := []SsConnection{
		{ForeignAddr: "10.0.0.1"},
		{ForeignAddr: "10.0.0.1"},
		{ForeignAddr: "10.0.0.1"},
		{ForeignAddr: "192.168.1.1"},
	}

	buf := captureOutput(func() {
		err := printTopDestinations(conns, 5)
		assert.NoError(t, err)
	})

	assert.Contains(t, buf, "10.0.0.1")
	assert.Contains(t, buf, "192.168.1.1")
	assert.Contains(t, buf, "目标地址TOP 5")
}

// === PrintComprehensiveDashboard 测试 ===

func TestPrintComprehensiveDashboard_Empty(t *testing.T) {
	buf := captureOutput(func() {
		err := PrintComprehensiveDashboard([]SsConnection{}, 5)
		assert.NoError(t, err)
	})

	assert.Contains(t, buf, "TIME_WAIT TOP")
	assert.Contains(t, buf, "无活跃流量")
}

func TestPrintComprehensiveDashboard_WithData(t *testing.T) {
	conns := []SsConnection{
		{State: "TIME_WAIT", ForeignAddr: "10.0.0.1", LocalAddr: "192.168.1.1", LocalPort: 54321, ForeignPort: 80},
		{State: "TIME_WAIT", ForeignAddr: "10.0.0.2", LocalAddr: "192.168.1.1", LocalPort: 54322, ForeignPort: 443},
		{State: "ESTABLISHED", ForeignAddr: "10.0.0.1", LocalAddr: "192.168.1.2", LocalPort: 12345, ForeignPort: 3306, RecvQ: 4096, SendQ: 2048},
	}

	buf := captureOutput(func() {
		err := PrintComprehensiveDashboard(conns, 10)
		assert.NoError(t, err)
	})

	assert.Contains(t, buf, "10.0.0.1")
	assert.Contains(t, buf, "10.0.0.2")
}

func TestPrintComprehensiveDashboard_DefaultTopN(t *testing.T) {
	buf := captureOutput(func() {
		err := PrintComprehensiveDashboard([]SsConnection{}, 0)
		assert.NoError(t, err)
	})

	assert.Contains(t, buf, "TIME_WAIT TOP")
	// topN=0 时默认 10
	assert.Contains(t, buf, "TOP 10")
}

func TestPrintComprehensiveDashboard_WithTraffic(t *testing.T) {
	conns := []SsConnection{
		{State: "ESTABLISHED", LocalAddr: "192.168.1.1", LocalPort: 12345, ForeignAddr: "10.0.0.1", ForeignPort: 80, RecvQ: 102400, SendQ: 51200},
	}

	buf := captureOutput(func() {
		err := PrintComprehensiveDashboard(conns, 5)
		assert.NoError(t, err)
	})

	assert.Contains(t, buf, "连接流量 TOP")
}

// === ReadTCPConnections/ReadUDPConnections 在 macOS 上 ===

func TestReadTCPConnections_ReturnsSlice(t *testing.T) {
	// macOS 无 /proc/net/tcp，应返回空切片
	conns := ReadTCPConnectionsWithPrograms(false, true, false)
	assert.IsType(t, []SsConnection{}, conns)
}

func TestReadUDPConnections_ReturnsSlice(t *testing.T) {
	conns := ReadUDPConnectionsWithPrograms(false, true, false)
	assert.IsType(t, []SsConnection{}, conns)
}

// === getProgramName 边界条件 ===

func TestGetProgramName_InvalidPID(t *testing.T) {
	assert.Equal(t, "", getProgramName(0))
	assert.Equal(t, "", getProgramName(-1))
	assert.Equal(t, "", getProgramName(-100))
}

// === captureOutput 辅助函数 ===

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}
