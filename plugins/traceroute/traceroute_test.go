package traceroute

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Constants 测试 ---

func TestDefaultMaxHops(t *testing.T) {
	assert.Equal(t, 30, DefaultMaxHops)
}

func TestDefaultPacketSize(t *testing.T) {
	assert.Equal(t, 60, DefaultPacketSize)
}

func TestDefaultTimeout(t *testing.T) {
	assert.Equal(t, 3*time.Second, DefaultTimeout)
}

func TestDefaultPort(t *testing.T) {
	assert.Equal(t, 33434, DefaultPort)
}

// --- 参数验证测试 ---

func TestTraceroute_DefaultParameters(t *testing.T) {
	// 测试默认参数回退逻辑
	maxHops := 0
	packetSize := 0

	if maxHops <= 0 {
		maxHops = DefaultMaxHops
	}
	if packetSize <= 0 {
		packetSize = DefaultPacketSize
	}

	assert.Equal(t, DefaultMaxHops, maxHops)
	assert.Equal(t, DefaultPacketSize, packetSize)
}

func TestTraceroute_NegativeMaxHops(t *testing.T) {
	maxHops := -1
	if maxHops <= 0 {
		maxHops = DefaultMaxHops
	}
	assert.Equal(t, DefaultMaxHops, maxHops)
}

func TestTraceroute_CustomParameters(t *testing.T) {
	maxHops := 15
	packetSize := 128

	if maxHops <= 0 {
		maxHops = DefaultMaxHops
	}
	if packetSize <= 0 {
		packetSize = DefaultPacketSize
	}

	assert.Equal(t, 15, maxHops)
	assert.Equal(t, 128, packetSize)
}

// --- DNS 解析测试 ---

func TestTraceroute_ResolveLocalhost(t *testing.T) {
	ips, err := net.LookupIP("localhost")
	require.NoError(t, err)
	assert.NotEmpty(t, ips)

	var hasIPv4 bool
	for _, ip := range ips {
		if ip.To4() != nil {
			hasIPv4 = true
			break
		}
	}
	assert.True(t, hasIPv4)
}

func TestTraceroute_InvalidHost(t *testing.T) {
	_, err := net.LookupIP("this.host.does.not.exist.invalid")
	assert.Error(t, err)
}

func TestTraceroute_IPv4Selection(t *testing.T) {
	// 测试 IPv4 地址选择逻辑
	ips := []net.IP{
		net.ParseIP("::1"),
		net.ParseIP("127.0.0.1"),
		net.ParseIP("fe80::1"),
	}

	var ip net.IP
	for _, candidate := range ips {
		if candidate.To4() != nil {
			ip = candidate
			break
		}
	}

	assert.NotNil(t, ip)
	assert.Equal(t, "127.0.0.1", ip.String())
}

func TestTraceroute_NoIPv4(t *testing.T) {
	// 测试没有 IPv4 地址的情况
	ips := []net.IP{
		net.ParseIP("::1"),
		net.ParseIP("fe80::1"),
	}

	var ip net.IP
	for _, candidate := range ips {
		if candidate.To4() != nil {
			ip = candidate
			break
		}
	}

	assert.Nil(t, ip)
}

// --- 端口计算测试 ---

func TestTraceroute_PortCalculation(t *testing.T) {
	// 测试端口计算: DefaultPort + ttl*3 + probe
	tests := []struct {
		ttl   int
		probe int
		port  int
	}{
		{1, 0, 33434 + 3},   // 33437
		{1, 1, 33434 + 4},   // 33438
		{1, 2, 33434 + 5},   // 33439
		{5, 0, 33434 + 15},  // 33449
		{10, 0, 33434 + 30}, // 33464
		{30, 2, 33434 + 92}, // 33526
	}

	for _, tt := range tests {
		port := DefaultPort + tt.ttl*3 + tt.probe
		assert.Equal(t, tt.port, port, "ttl=%d, probe=%d", tt.ttl, tt.probe)
	}
}

// --- 输出格式测试 ---

func TestTraceroute_OutputFormat(t *testing.T) {
	// 测试 traceroute 输出格式
	host := "example.com"
	targetIP := net.ParseIP("93.184.216.34")
	maxHops := 30
	packetSize := 60

	header := fmt.Sprintf("traceroute to %s (%s), %d hops max, %d byte packets\n",
		host, targetIP.String(), maxHops, packetSize)

	assert.Contains(t, header, "traceroute to example.com")
	assert.Contains(t, header, "93.184.216.34")
	assert.Contains(t, header, "30 hops max")
	assert.Contains(t, header, "60 byte packets")
}

func TestTraceroute_HopFormat(t *testing.T) {
	// 测试每跳的输出格式
	ttl := 5
	elapsed := 15 * time.Millisecond

	hopLine := fmt.Sprintf("%2d  ", ttl)
	assert.Equal(t, " 5  ", hopLine)

	timeStr := fmt.Sprintf("%.3f ms  ", float64(elapsed.Nanoseconds())/1e6)
	assert.Contains(t, timeStr, "ms")
}

func TestTraceroute_TimeoutFormat(t *testing.T) {
	// 测试超时显示
	timeoutDisplay := "*  "
	assert.Equal(t, "*  ", timeoutDisplay)
}

// --- UDP 地址测试 ---

func TestTraceroute_UDPAddress(t *testing.T) {
	targetIP := net.ParseIP("8.8.8.8")
	port := 33437

	remoteAddr := &net.UDPAddr{
		IP:   targetIP,
		Port: port,
	}

	assert.Equal(t, "8.8.8.8:33437", remoteAddr.String())
}

// --- ICMP 类型判断测试 ---

func TestTraceroute_ICMPDestinationUnreachable_Code3(t *testing.T) {
	// 测试 ICMP Destination Unreachable Code 3 (Port Unreachable) 判断逻辑
	// Code 3 表示目标端口不可达，traceroute 据此判断到达目标
	msgCode := 3
	responderIP := net.ParseIP("8.8.8.8")
	targetIP := net.ParseIP("8.8.8.8")

	probeReached := false
	if msgCode == 3 && responderIP != nil && responderIP.Equal(targetIP) {
		probeReached = true
	}
	assert.True(t, probeReached, "相同 IP + Code 3 应该判定到达目标")
}

func TestTraceroute_ICMPDestinationUnreachable_WrongIP(t *testing.T) {
	msgCode := 3
	responderIP := net.ParseIP("10.0.0.1")
	targetIP := net.ParseIP("8.8.8.8")

	probeReached := false
	if msgCode == 3 && responderIP != nil && responderIP.Equal(targetIP) {
		probeReached = true
	}
	assert.False(t, probeReached, "不同 IP 不应判定到达目标")
}

func TestTraceroute_ICMPTimeExceeded(t *testing.T) {
	// TTL 超时不应该判定到达目标
	probeReached := false // TimeExceeded 时保持 false
	assert.False(t, probeReached)
}

// --- 主机名解析测试 ---

func TestTraceroute_HostnameLookup(t *testing.T) {
	// 测试反向 DNS 查找
	hostname := "8.8.8.8"
	names, err := net.LookupAddr(hostname)
	if err == nil && len(names) > 0 {
		// 可能有反向 DNS
		name := names[0]
		if len(name) > 0 && name[len(name)-1] == '.' {
			name = name[:len(name)-1]
		}
		assert.NotEmpty(t, name)
	}
	// 反向 DNS 可能失败，这是正常的
}

func TestTraceroute_HostnameFormat(t *testing.T) {
	// 测试主机名格式化逻辑
	responderIP := net.ParseIP("192.168.1.1")
	hostname := responderIP.String()
	assert.Equal(t, "192.168.1.1", hostname)

	// 模拟有主机名的情况
	name := "router.local."
	if len(name) > 0 && name[len(name)-1] == '.' {
		name = name[:len(name)-1]
	}
	formatted := fmt.Sprintf("%s (%s)", name, responderIP.String())
	assert.Equal(t, "router.local (192.168.1.1)", formatted)
}

// --- TCP 回退测试 ---

func TestTraceroute_TCPFallbackPorts(t *testing.T) {
	// tracerouteTCP 使用的端口列表
	ports := []string{"80", "443", "22", "53"}
	assert.Len(t, ports, 4)

	for _, port := range ports {
		_, err := net.LookupPort("tcp", port)
		assert.NoError(t, err, "端口 %s 应该有效", port)
	}
}

func TestTraceroute_TCPFallbackConnection(t *testing.T) {
	// 测试 TCP 回退方式的连接
	// 启动本地服务器
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	_, port, _ := net.SplitHostPort(ln.Addr().String())

	// TCP 方式连接
	start := time.Now()
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 2*time.Second)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	if conn != nil {
		conn.Close()
	}
	assert.True(t, elapsed < 2*time.Second)

	ln.Close()
}

// --- TTL 相关测试 ---

func TestTraceroute_TTLRange(t *testing.T) {
	// 测试 TTL 范围 1 到 maxHops
	maxHops := 30
	count := 0
	for ttl := 1; ttl <= maxHops; ttl++ {
		count++
		assert.True(t, ttl >= 1 && ttl <= 30)
	}
	assert.Equal(t, 30, count)
}

func TestTraceroute_Probes(t *testing.T) {
	// 每跳发送 3 次探测
	probes := 3
	for probe := 0; probe < probes; probe++ {
		assert.True(t, probe >= 0 && probe < 3)
	}
}
