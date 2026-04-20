package netstat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// === getConnectionType 测试 ===

func TestGetConnectionType(t *testing.T) {
	tests := []struct {
		name     string
		connType uint32
		want     string
	}{
		{"TCP类型", 1, "tcp"},
		{"UDP类型", 2, "udp"},
		{"未知类型", 99, "unknown"},
		{"零值", 0, "unknown"},
		{"TCP6仍返回tcp", 1, "tcp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getConnectionType(tt.connType)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === isIPv6 测试 ===

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"IPv4地址", "192.168.1.1", false},
		{"IPv4回环", "127.0.0.1", false},
		{"空字符串", "", false},
		{"IPv6地址", "fe80::1", true},
		{"IPv6回环", "::1", true},
		{"IPv6完整", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"IPv6缩写", "::", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIPv6(tt.ip)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === formatAddr 测试 ===

func TestFormatAddr(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		port    uint32
		numeric bool
		want    string
	}{
		{"空IP零端口", "", 0, false, "*:*"},
		{"空IP有端口", "", 8080, false, "*:8080"},
		{"IPv4_numeric模式", "192.168.1.1", 8080, true, "192.168.1.1:8080"},
		{"IPv4_零端口", "192.168.1.1", 0, false, "192.168.1.1:*"},
		{"IPv4_知名端口_非numeric", "10.0.0.1", 80, false, "10.0.0.1:http"},
		{"IPv4_知名端口_ssh", "10.0.0.1", 22, false, "10.0.0.1:ssh"},
		{"IPv4_知名端口_https", "10.0.0.1", 443, false, "10.0.0.1:https"},
		{"IPv4_知名端口_mysql", "10.0.0.1", 3306, false, "10.0.0.1:mysql"},
		{"IPv4_知名端口_redis", "10.0.0.1", 6379, false, "10.0.0.1:redis"},
		{"IPv4_非知名端口_显示数字", "10.0.0.1", 9999, false, "10.0.0.1:9999"},
		{"IPv4_numeric模式_忽略服务名", "10.0.0.1", 80, true, "10.0.0.1:80"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAddr(tt.ip, tt.port, tt.numeric)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === getServiceName 测试 ===

func TestGetServiceName(t *testing.T) {
	tests := []struct {
		name string
		port uint32
		want string
	}{
		{"SSH端口", 22, "ssh"},
		{"SMTP端口", 25, "smtp"},
		{"HTTP端口", 80, "http"},
		{"HTTPS端口", 443, "https"},
		{"MySQL端口", 3306, "mysql"},
		{"Redis端口", 6379, "redis"},
		{"Kibana端口", 5601, "kibana"},
		{"rpcbind端口", 111, "rpcbind"},
		{"未知端口", 12345, ""},
		{"零端口", 0, ""},
		{"高端口", 65535, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getServiceName(tt.port)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === parseHexAddr 测试 (来自ss.go) ===

func TestParseHexAddr(t *testing.T) {
	tests := []struct {
		name      string
		addrPort  string
		wantIP    string
		wantPort  uint32
	}{
		{
			"IPv4回环地址_端口80",
			"0100007F:0050",
			"127.0.0.1",
			80,
		},
		{
			"IPv4_0.0.0.0_端口0",
			"00000000:0000",
			"0.0.0.0",
			0,
		},
		{
			"IPv4_192.168.1.1_端口8080",
			"0101A8C0:1F90",
			"192.168.1.1",
			8080,
		},
		{
			"IPv4_10.0.0.1_端口22",
			"0100000A:0016",
			"10.0.0.1",
			22,
		},
		{
			"IPv4_全0_端口9999",
			"00000000:270F",
			"0.0.0.0",
			9999,
		},
		{
			"IPv6地址_回环",
			"00000000000000000000000001000000:0050",
			"[::1]",
			80,
		},
		{
			"无效格式_无冒号",
			"invalid",
			"0.0.0.0",
			0,
		},
		{
			"空字符串",
			"",
			"0.0.0.0",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIP, gotPort := parseHexAddr(tt.addrPort)
			assert.Equal(t, tt.wantIP, gotIP)
			assert.Equal(t, tt.wantPort, gotPort)
		})
	}
}

// === parseTCPState 测试 ===

func TestParseTCPState(t *testing.T) {
	tests := []struct {
		name     string
		stateHex string
		want     string
	}{
		{"ESTABLISHED", "01", "ESTABLISHED"},
		{"SYN_SENT", "02", "SYN_SENT"},
		{"SYN_RECV", "03", "SYN_RECV"},
		{"FIN_WAIT1", "04", "FIN_WAIT1"},
		{"FIN_WAIT2", "05", "FIN_WAIT2"},
		{"TIME_WAIT", "06", "TIME_WAIT"},
		{"CLOSE", "07", "CLOSE"},
		{"CLOSE_WAIT", "08", "CLOSE_WAIT"},
		{"LAST_ACK", "09", "LAST_ACK"},
		{"LISTEN", "0A", "LISTEN"},
		{"CLOSING", "0B", "CLOSING"},
		{"未知状态_0C", "0C", "UNKNOWN"},
		{"无效十六进制", "ZZ", "UNKNOWN"},
		{"空字符串", "", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTCPState(tt.stateHex)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === SsConnection 结构体测试 ===

func TestSsConnection_Fields(t *testing.T) {
	conn := SsConnection{
		Proto:       "tcp",
		RecvQ:       0,
		SendQ:       0,
		LocalAddr:   "127.0.0.1",
		LocalPort:   8080,
		ForeignAddr: "192.168.1.1",
		ForeignPort: 54321,
		State:       "ESTABLISHED",
		PID:         1234,
		ProcessName: "nginx",
	}

	assert.Equal(t, "tcp", conn.Proto)
	assert.Equal(t, "127.0.0.1", conn.LocalAddr)
	assert.Equal(t, uint32(8080), conn.LocalPort)
	assert.Equal(t, "192.168.1.1", conn.ForeignAddr)
	assert.Equal(t, uint32(54321), conn.ForeignPort)
	assert.Equal(t, "ESTABLISHED", conn.State)
	assert.Equal(t, int32(1234), conn.PID)
	assert.Equal(t, "nginx", conn.ProcessName)
}

// === 集成测试: parseHexAddr + parseTCPState 联合解析 ===

func TestParseProcNetTCPLine(t *testing.T) {
	// 模拟 /proc/net/tcp 中的一行数据
	// 格式: sl  local_address rem_address   st tx_queue rx_queue ...
	// 示例: 0: 0100007F:0050 00000000:0000 0A ...
	localIP, localPort := parseHexAddr("0100007F:0050")
	remoteIP, remotePort := parseHexAddr("00000000:0000")
	state := parseTCPState("0A")

	assert.Equal(t, "127.0.0.1", localIP)
	assert.Equal(t, uint32(80), localPort)
	assert.Equal(t, "0.0.0.0", remoteIP)
	assert.Equal(t, uint32(0), remotePort)
	assert.Equal(t, "LISTEN", state)
}

func TestParseProcNetTCPEstablished(t *testing.T) {
	// 已建立的连接: 192.168.1.100:54321 -> 10.0.0.1:80
	localIP, localPort := parseHexAddr("6401A8C0:D431")
	remoteIP, remotePort := parseHexAddr("0100000A:0050")
	state := parseTCPState("01")

	assert.Equal(t, "192.168.1.100", localIP)
	assert.Equal(t, uint32(54321), localPort)
	assert.Equal(t, "10.0.0.1", remoteIP)
	assert.Equal(t, uint32(80), remotePort)
	assert.Equal(t, "ESTABLISHED", state)
}

func TestParseProcNetTCPTimewait(t *testing.T) {
	// TIME_WAIT 连接
	state := parseTCPState("06")
	assert.Equal(t, "TIME_WAIT", state)
}

// === 统计相关辅助函数测试 ===

func TestSsConnectionSlice_SortByLocalAddr(t *testing.T) {
	connections := []SsConnection{
		{LocalAddr: "192.168.1.2", LocalPort: 80},
		{LocalAddr: "192.168.1.1", LocalPort: 443},
		{LocalAddr: "192.168.1.1", LocalPort: 80},
	}

	// 模拟排序逻辑（与ss.go中相同）
	// 按本地地址排序，地址相同按端口排序
	for i := 0; i < len(connections)-1; i++ {
		for j := i + 1; j < len(connections); j++ {
			if connections[i].LocalAddr > connections[j].LocalAddr ||
				(connections[i].LocalAddr == connections[j].LocalAddr &&
					connections[i].LocalPort > connections[j].LocalPort) {
				connections[i], connections[j] = connections[j], connections[i]
			}
		}
	}

	assert.Equal(t, "192.168.1.1", connections[0].LocalAddr)
	assert.Equal(t, uint32(80), connections[0].LocalPort)
	assert.Equal(t, "192.168.1.1", connections[1].LocalAddr)
	assert.Equal(t, uint32(443), connections[1].LocalPort)
	assert.Equal(t, "192.168.1.2", connections[2].LocalAddr)
}
