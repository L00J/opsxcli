package ping

import (
	"net"
	"testing"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"

	"github.com/stretchr/testify/assert"
)

// =============================================
// IPv6 ICMPv6 Ping 测试
// =============================================

// --- isIPv6 ---

func TestIsIPv6_PureIPv6(t *testing.T) {
	// 纯 IPv6 地址
	ip := net.ParseIP("2001:db8::1")
	assert.True(t, isIPv6(ip))
}

func TestIsIPv6_Loopback(t *testing.T) {
	// IPv6 回环地址
	ip := net.ParseIP("::1")
	assert.True(t, isIPv6(ip))
}

func TestIsIPv6_IPv4Mapped(t *testing.T) {
	// IPv4 映射的 IPv6 地址（::ffff:192.168.1.1）应该返回 false
	ip := net.ParseIP("::ffff:192.168.1.1")
	assert.False(t, isIPv6(ip))
}

func TestIsIPv6_IPv4(t *testing.T) {
	// 纯 IPv4 地址应该返回 false
	ip := net.ParseIP("192.168.1.1")
	assert.False(t, isIPv6(ip))
}

func TestIsIPv6_IPv4Localhost(t *testing.T) {
	ip := net.ParseIP("127.0.0.1")
	assert.False(t, isIPv6(ip))
}

func TestIsIPv6_IPv6LinkLocal(t *testing.T) {
	// IPv6 链路本地地址
	ip := net.ParseIP("fe80::1")
	assert.True(t, isIPv6(ip))
}

func TestIsIPv6_IPv6AllNodes(t *testing.T) {
	// IPv6 所有节点多播地址
	ip := net.ParseIP("ff02::1")
	assert.True(t, isIPv6(ip))
}

// --- resolveIPv6 ---

func TestResolveIPv6_Loopback(t *testing.T) {
	ip, err := resolveIPv6("::1")
	assert.NoError(t, err)
	assert.NotNil(t, ip)
	assert.True(t, isIPv6(ip))
}

func TestResolveIPv6_IPv4Address(t *testing.T) {
	// 传入 IPv4 地址应该返回错误
	_, err := resolveIPv6("127.0.0.1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IPv4")
}

func TestResolveIPv6_IPv6Address(t *testing.T) {
	ip, err := resolveIPv6("2001:db8::1")
	assert.NoError(t, err)
	assert.Equal(t, "2001:db8::1", ip.String())
}

func TestResolveIPv6_InvalidHost(t *testing.T) {
	_, err := resolveIPv6("this.host.does.not.exist.invalid.tld")
	assert.Error(t, err)
}

// --- buildICMPv6EchoRequest ---

func TestBuildICMPv6EchoRequest_Type(t *testing.T) {
	msg := buildICMPv6EchoRequest(1, 56)
	assert.NotNil(t, msg)
	// ICMPv6 Echo Request Type = 128
	assert.Equal(t, ipv6.ICMPTypeEchoRequest, msg.Type)
	assert.Equal(t, 0, msg.Code)
}

func TestBuildICMPv6EchoRequest_EchoBody(t *testing.T) {
	msg := buildICMPv6EchoRequest(5, 32)
	echo, ok := msg.Body.(*icmp.Echo)
	assert.True(t, ok, "Body 应该是 *icmp.Echo 类型")
	assert.NotNil(t, echo)
}

func TestBuildICMPv6EchoRequest_EchoBodyDirect(t *testing.T) {
	msg := buildICMPv6EchoRequest(5, 32)
	// 直接使用接口断言获取 Echo body
	body := msg.Body
	assert.NotNil(t, body)
}

func TestBuildICMPv6EchoRequest_DifferentSeq(t *testing.T) {
	msg1 := buildICMPv6EchoRequest(1, 8)
	msg2 := buildICMPv6EchoRequest(2, 8)
	// 不同的 seq 号
	assert.NotNil(t, msg1)
	assert.NotNil(t, msg2)
}

func TestBuildICMPv6EchoRequest_PayloadSize(t *testing.T) {
	for _, size := range []int{0, 8, 24, 56, 128} {
		msg := buildICMPv6EchoRequest(1, size)
		assert.NotNil(t, msg, "payload size=%d", size)
	}
}

// --- buildICMPv6EchoRequestBytes ---

func TestBuildICMPv6EchoRequestBytes_Normal(t *testing.T) {
	data, err := buildICMPv6EchoRequestBytes(1, 56)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	// ICMPv6 Echo Request 最小 8 字节 + payload
	assert.GreaterOrEqual(t, len(data), 8)
}

func TestBuildICMPv6EchoRequestBytes_Type128(t *testing.T) {
	data, err := buildICMPv6EchoRequestBytes(1, 8)
	assert.NoError(t, err)
	// Type = 128 (Echo Request)
	assert.Equal(t, uint8(128), data[0])
	// Code = 0
	assert.Equal(t, uint8(0), data[1])
}

func TestBuildICMPv6EchoRequestBytes_DifferentPayloadSizes(t *testing.T) {
	for _, size := range []int{0, 8, 56, 128} {
		data, err := buildICMPv6EchoRequestBytes(1, size)
		assert.NoError(t, err, "payload size=%d", size)
		assert.GreaterOrEqual(t, len(data), 8, "payload size=%d", size)
	}
}

// --- IPv4/IPv6 自动检测逻辑 ---

func TestAutoDetect_IPv4Address(t *testing.T) {
	// 通过 resolveIP 获取 IPv4 地址后检测
	ip := net.ParseIP("192.168.1.1")
	assert.False(t, isIPv6(ip))
}

func TestAutoDetect_IPv6Address(t *testing.T) {
	// 通过 resolveIP 获取 IPv6 地址后检测
	ip := net.ParseIP("2001:db8::1")
	assert.True(t, isIPv6(ip))
}

func TestAutoDetect_LocalhostIPv4(t *testing.T) {
	ip := net.ParseIP("127.0.0.1")
	assert.False(t, isIPv6(ip))
}

func TestAutoDetect_LocalhostIPv6(t *testing.T) {
	ip := net.ParseIP("::1")
	assert.True(t, isIPv6(ip))
}

func TestAutoDetect_IPv4MappedAsIPv4(t *testing.T) {
	// ::ffff:10.0.0.1 是 IPv4 映射地址，应视为 IPv4
	ip := net.ParseIP("::ffff:10.0.0.1")
	assert.False(t, isIPv6(ip))
}

// --- resolveIP IPv6 测试 ---

func TestResolveIP_IPv6Loopback(t *testing.T) {
	ip, err := resolveIP("::1", "6")
	assert.NoError(t, err)
	assert.True(t, isIPv6(ip))
}

func TestResolveIP_IPv6FullAddress(t *testing.T) {
	ip, err := resolveIP("2001:db8::1", "6")
	assert.NoError(t, err)
	assert.Equal(t, "2001:db8::1", ip.String())
}

func TestResolveIP_IPv6WantedButGotIPv4(t *testing.T) {
	// 期望 IPv6 但传入 IPv4 地址
	_, err := resolveIP("192.168.1.1", "6")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IPv4")
}

func TestResolveIP_IPv4WantedButGotIPv6(t *testing.T) {
	// resolveIP("::1", "4") 直接 IP 不做版本冲突检查，直接返回
	// 因为 resolveIP 中只检查 ipVersion=="6" && To4()!=nil
	// IPv4 模式下传入 IPv6 地址会直接返回（后续 isIPv6 检测会路由到 IPv6 路径）
	ip, err := resolveIP("::1", "4")
	assert.NoError(t, err)
	assert.True(t, isIPv6(ip))
}

// --- PingConfig IPv6 ---

func TestPingConfig_IPv6Mode(t *testing.T) {
	cfg := NewPingConfig("::1")
	cfg.IPVersion = "6"
	assert.Equal(t, "6", cfg.IPVersion)
}

func TestPingConfig_IPv6WithModeICMP(t *testing.T) {
	cfg := NewPingConfig("2001:db8::1")
	cfg.IPVersion = "6"
	cfg.Mode = ModeICMP
	assert.Equal(t, ModeICMP, cfg.Mode)
	assert.Equal(t, "6", cfg.IPVersion)
}

func TestPingConfig_IPv6WithModeUDP(t *testing.T) {
	cfg := NewPingConfig("2001:db8::1")
	cfg.IPVersion = "6"
	cfg.Mode = ModeUDP
	assert.Equal(t, ModeUDP, cfg.Mode)
}

func TestPingConfig_IPv6WithModeTCP(t *testing.T) {
	cfg := NewPingConfig("2001:db8::1")
	cfg.IPVersion = "6"
	cfg.Mode = ModeTCP
	assert.Equal(t, ModeTCP, cfg.Mode)
}

// --- ProtocolICMPv6 常量 ---

func TestProtocolICMPv6_Value(t *testing.T) {
	assert.Equal(t, 58, ProtocolICMPv6)
}

// --- selectPingMode（IPv6 不影响模式选择） ---

func TestSelectPingMode_ICMPv6(t *testing.T) {
	// IPv6 模式下模式选择逻辑与 IPv4 一致
	mode := selectPingMode(ModeICMP)
	assert.Equal(t, ModeICMP, mode)
}

func TestSelectPingMode_UDPv6(t *testing.T) {
	mode := selectPingMode(ModeUDP)
	assert.Equal(t, ModeUDP, mode)
}

func TestSelectPingMode_TCPv6(t *testing.T) {
	mode := selectPingMode(ModeTCP)
	assert.Equal(t, ModeTCP, mode)
}
