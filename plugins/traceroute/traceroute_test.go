package traceroute

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/ipv4"
)

// --- Constants 测试 ---

func TestConstants_DefaultValues(t *testing.T) {
	assert.Equal(t, 30, DefaultMaxHops)
	assert.Equal(t, 60, DefaultPacketSize)
	assert.Equal(t, 3*time.Second, DefaultTimeout)
	assert.Equal(t, 33434, DefaultPort)
}

// --- selectIPv4 ---

func TestSelectIPv4_WithIPv4(t *testing.T) {
	ips := []net.IP{
		net.ParseIP("::1"),
		net.ParseIP("127.0.0.1"),
		net.ParseIP("fe80::1"),
	}
	ip := selectIPv4(ips)
	require.NotNil(t, ip)
	assert.Equal(t, "127.0.0.1", ip.String())
}

func TestSelectIPv4_NoIPv4(t *testing.T) {
	ips := []net.IP{
		net.ParseIP("::1"),
		net.ParseIP("fe80::1"),
	}
	ip := selectIPv4(ips)
	assert.Nil(t, ip)
}

func TestSelectIPv4_Empty(t *testing.T) {
	ip := selectIPv4(nil)
	assert.Nil(t, ip)
}

func TestSelectIPv4_FirstIPv4(t *testing.T) {
	ips := []net.IP{
		net.ParseIP("10.0.0.1"),
		net.ParseIP("192.168.1.1"),
	}
	ip := selectIPv4(ips)
	require.NotNil(t, ip)
	assert.Equal(t, "10.0.0.1", ip.String()) // returns first IPv4
}

// --- calculatePort ---

func TestCalculatePort_Basic(t *testing.T) {
	assert.Equal(t, 33434+3, calculatePort(1, 0))
	assert.Equal(t, 33434+4, calculatePort(1, 1))
	assert.Equal(t, 33434+5, calculatePort(1, 2))
}

func TestCalculatePort_LargeTTL(t *testing.T) {
	assert.Equal(t, 33434+90, calculatePort(30, 0))
	assert.Equal(t, 33434+92, calculatePort(30, 2))
}

func TestCalculatePort_Zero(t *testing.T) {
	assert.Equal(t, 33434, calculatePort(0, 0))
}

// --- normalizeMaxHops ---

func TestNormalizeMaxHops_Valid(t *testing.T) {
	assert.Equal(t, 15, normalizeMaxHops(15))
	assert.Equal(t, 30, normalizeMaxHops(30))
}

func TestNormalizeMaxHops_Zero(t *testing.T) {
	assert.Equal(t, DefaultMaxHops, normalizeMaxHops(0))
}

func TestNormalizeMaxHops_Negative(t *testing.T) {
	assert.Equal(t, DefaultMaxHops, normalizeMaxHops(-5))
}

// --- normalizePacketSize ---

func TestNormalizePacketSize_Valid(t *testing.T) {
	assert.Equal(t, 128, normalizePacketSize(128))
	assert.Equal(t, 60, normalizePacketSize(60))
}

func TestNormalizePacketSize_Zero(t *testing.T) {
	assert.Equal(t, DefaultPacketSize, normalizePacketSize(0))
}

func TestNormalizePacketSize_Negative(t *testing.T) {
	assert.Equal(t, DefaultPacketSize, normalizePacketSize(-100))
}

// --- stripTrailingDot ---

func TestStripTrailingDot_WithDot(t *testing.T) {
	assert.Equal(t, "example.com", stripTrailingDot("example.com."))
}

func TestStripTrailingDot_WithoutDot(t *testing.T) {
	assert.Equal(t, "example.com", stripTrailingDot("example.com"))
}

func TestStripTrailingDot_Empty(t *testing.T) {
	assert.Equal(t, "", stripTrailingDot(""))
}

func TestStripTrailingDot_OnlyDot(t *testing.T) {
	assert.Equal(t, "", stripTrailingDot("."))
}

// --- formatElapsedTime ---

func TestFormatElapsedTime_Milliseconds(t *testing.T) {
	result := formatElapsedTime(15 * time.Millisecond)
	assert.Contains(t, result, "15.000 ms")
}

func TestFormatElapsedTime_Zero(t *testing.T) {
	result := formatElapsedTime(0)
	assert.Contains(t, result, "0.000 ms")
}

func TestFormatElapsedTime_SubMs(t *testing.T) {
	result := formatElapsedTime(500 * time.Microsecond)
	assert.Contains(t, result, "0.500 ms")
}

// --- formatTracerouteHeader ---

func TestFormatTracerouteHeader_Normal(t *testing.T) {
	result := formatTracerouteHeader("example.com", net.ParseIP("93.184.216.34"), 30, 60)
	assert.Contains(t, result, "traceroute to example.com")
	assert.Contains(t, result, "93.184.216.34")
	assert.Contains(t, result, "30 hops max")
	assert.Contains(t, result, "60 byte packets")
}

// --- formatHopPrefix ---

func TestFormatHopPrefix_Single(t *testing.T) {
	assert.Equal(t, " 1  ", formatHopPrefix(1))
}

func TestFormatHopPrefix_Double(t *testing.T) {
	assert.Equal(t, "10  ", formatHopPrefix(10))
}

func TestFormatHopPrefix_SingleLarge(t *testing.T) {
	assert.Equal(t, "30  ", formatHopPrefix(30))
}

// --- buildUDPAddr ---

func TestBuildUDPAddr_Normal(t *testing.T) {
	addr := buildUDPAddr(net.ParseIP("8.8.8.8"), 33437)
	require.NotNil(t, addr)
	assert.Equal(t, "8.8.8.8:33437", addr.String())
}

// --- formatHostDisplay ---

func TestFormatHostDisplay_WithLoopback(t *testing.T) {
	// 127.0.0.1 likely has a reverse DNS entry on most systems, but even if not,
	// it should at least return the IP string.
	ip := net.ParseIP("127.0.0.1")
	result := formatHostDisplay(ip)
	if result == "" {
		t.Error("formatHostDisplay should return non-empty string")
	}
	// Should always contain the IP
	if !strings.Contains(result, "127.0.0.1") {
		t.Errorf("formatHostDisplay should contain IP, got %q", result)
	}
}

func TestFormatHostDisplay_LocalAddress(t *testing.T) {
	// A local address that typically has no reverse DNS
	ip := net.ParseIP("192.168.255.254")
	result := formatHostDisplay(ip)
	if result == "" {
		t.Error("formatHostDisplay should return non-empty string")
	}
	// Without reverse DNS, should just be the IP
	if !strings.Contains(result, "192.168.255.254") {
		t.Errorf("formatHostDisplay should contain IP, got %q", result)
	}
}

func TestFormatHostDisplay_Format(t *testing.T) {
	// If there's a reverse DNS entry, format is "hostname (ip)"
	// If not, format is just the IP
	ip := net.ParseIP("8.8.8.8")
	result := formatHostDisplay(ip)
	if result == "" {
		t.Error("formatHostDisplay should return non-empty string")
	}
	// Must contain the IP regardless
	if !strings.Contains(result, "8.8.8.8") {
		t.Errorf("formatHostDisplay should contain IP, got %q", result)
	}
}

// --- isDestinationReached ---

func TestIsDestinationReached_TimeExceeded(t *testing.T) {
	result := isDestinationReached(ipv4.ICMPTypeTimeExceeded, 0, nil, nil)
	assert.False(t, result)
}

func TestIsDestinationReached_DestUnreachableMatch(t *testing.T) {
	ip := net.ParseIP("8.8.8.8")
	result := isDestinationReached(ipv4.ICMPTypeDestinationUnreachable, 3, ip, ip)
	assert.True(t, result)
}

func TestIsDestinationReached_DestUnreachableWrongIP(t *testing.T) {
	responder := net.ParseIP("10.0.0.1")
	target := net.ParseIP("8.8.8.8")
	result := isDestinationReached(ipv4.ICMPTypeDestinationUnreachable, 3, responder, target)
	assert.False(t, result)
}

func TestIsDestinationReached_DestUnreachableWrongCode(t *testing.T) {
	ip := net.ParseIP("8.8.8.8")
	result := isDestinationReached(ipv4.ICMPTypeDestinationUnreachable, 1, ip, ip)
	assert.False(t, result)
}

func TestIsDestinationReached_NilResponder(t *testing.T) {
	result := isDestinationReached(ipv4.ICMPTypeDestinationUnreachable, 3, nil, net.ParseIP("8.8.8.8"))
	assert.False(t, result)
}

func TestIsDestinationReached_UnknownType(t *testing.T) {
	result := isDestinationReached(ipv4.ICMPType(0), 0, nil, nil)
	assert.False(t, result)
}
