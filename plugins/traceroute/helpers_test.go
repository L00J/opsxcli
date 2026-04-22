package traceroute

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/ipv4"
)

// --- selectIPv4 table-driven tests ---

func TestSelectIPv4Table(t *testing.T) {
	tests := []struct {
		name     string
		ips      []net.IP
		wantNil  bool
		wantIPv4 string // expected IPv4 string if not nil
	}{
		{
			name:     "从混合列表中选第一个IPv4",
			ips:      []net.IP{net.ParseIP("::1"), net.ParseIP("192.168.1.1"), net.ParseIP("fe80::1")},
			wantNil:  false,
			wantIPv4: "192.168.1.1",
		},
		{
			name:    "只有IPv6返回nil",
			ips:     []net.IP{net.ParseIP("::1"), net.ParseIP("fe80::1")},
			wantNil: true,
		},
		{
			name:    "空列表返回nil",
			ips:     nil,
			wantNil: true,
		},
		{
			name:     "多个IPv4返回第一个",
			ips:      []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("172.16.0.1"), net.ParseIP("192.168.1.1")},
			wantNil:  false,
			wantIPv4: "10.0.0.1",
		},
		{
			name:     "空列表返回nil",
			ips:     nil,
			wantNil: true,
		},
		{
			name:     "全是IPv4返回第一个",
			ips:      []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("172.16.0.1")},
			wantNil:  false,
			wantIPv4: "10.0.0.1",
		},
		{
			name:     "单个IPv4在首位",
			ips:      []net.IP{net.ParseIP("127.0.0.1")},
			wantNil:  false,
			wantIPv4: "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectIPv4(tt.ips)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.wantIPv4, got.String())
			}
		})
	}
}

// --- calculatePort table-driven tests ---

func TestCalculatePortTable(t *testing.T) {
	tests := []struct {
		name string
		ttl  int
		probe int
		want  int
	}{
		{name: "ttl=1 probe=0", ttl: 1, probe: 0, want: 33434 + 3},
		{name: "ttl=1 probe=1", ttl: 1, probe: 1, want: 33434 + 4},
		{name: "ttl=1 probe=2", ttl: 1, probe: 2, want: 33434 + 5},
		{name: "ttl=5 probe=2", ttl: 5, probe: 2, want: 33434 + 17},
		{name: "ttl=0 probe=0 基础端口", ttl: 0, probe: 0, want: 33434},
		{name: "ttl=30 probe=0 最大跳数", ttl: 30, probe: 0, want: 33434 + 90},
		{name: "ttl=30 probe=2 最大跳数最大探测", ttl: 30, probe: 2, want: 33434 + 92},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, calculatePort(tt.ttl, tt.probe))
		})
	}
}

// --- normalizeMaxHops table-driven tests ---

func TestNormalizeMaxHopsTable(t *testing.T) {
	tests := []struct {
		name    string
		maxHops int
		want    int
	}{
		{name: "正数原值返回", maxHops: 15, want: 15},
		{name: "零返回默认值", maxHops: 0, want: DefaultMaxHops},
		{name: "负数返回默认值", maxHops: -1, want: DefaultMaxHops},
		{name: "大值原值返回", maxHops: 64, want: 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeMaxHops(tt.maxHops))
		})
	}
}

// --- normalizePacketSize table-driven tests ---

func TestNormalizePacketSizeTable(t *testing.T) {
	tests := []struct {
		name       string
		packetSize int
		want       int
	}{
		{name: "正数原值返回", packetSize: 128, want: 128},
		{name: "零返回默认值", packetSize: 0, want: DefaultPacketSize},
		{name: "负数返回默认值", packetSize: -100, want: DefaultPacketSize},
		{name: "等于默认值原值返回", packetSize: 60, want: 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizePacketSize(tt.packetSize))
		})
	}
}

// --- stripTrailingDot table-driven tests ---

func TestStripTrailingDotTable(t *testing.T) {
	tests := []struct {
		name string
		input string
		want  string
	}{
		{name: "带末尾点号", input: "example.com.", want: "example.com"},
		{name: "不带末尾点号", input: "example.com", want: "example.com"},
		{name: "空字符串", input: "", want: ""},
		{name: "仅一个点号", input: ".", want: ""},
		{name: "多个点号移除最后一个", input: "a.b.c.", want: "a.b.c"},
		{name: "中间有点号末尾无点号", input: "host.example.com", want: "host.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stripTrailingDot(tt.input))
		})
	}
}

// --- formatHostDisplay table-driven tests ---
// 注意：formatHostDisplay 会做 DNS 查询，只测试无反查记录的 IP（回环地址等）

func TestFormatHostDisplayTable(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want string // 至少包含 IP 字符串
	}{
		{name: "回环地址显示IP", ip: net.ParseIP("127.0.0.1"), want: "127.0.0.1"},
		{name: "私有地址显示IP", ip: net.ParseIP("192.168.1.1"), want: "192.168.1.1"},
		{name: "10网段显示IP", ip: net.ParseIP("10.0.0.1"), want: "10.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatHostDisplay(tt.ip)
			// 因为 DNS 查询结果不可控，只验证结果至少包含 IP 字符串
			assert.Contains(t, result, tt.want)
		})
	}
}

// --- isDestinationReached table-driven tests ---

func TestIsDestinationReachedTable(t *testing.T) {
	ip8 := net.ParseIP("8.8.8.8")
	ip10 := net.ParseIP("10.0.0.1")

	tests := []struct {
		name       string
		msgType    ipv4.ICMPType
		msgCode    int
		responderIP net.IP
		targetIP   net.IP
		want       bool
	}{
		{
			name:       "TimeExceeded返回false",
			msgType:    ipv4.ICMPTypeTimeExceeded,
			msgCode:    0,
			responderIP: ip8,
			targetIP:   ip8,
			want:       false,
		},
		{
			name:       "DestUnreachable匹配IP和code=3返回true",
			msgType:    ipv4.ICMPTypeDestinationUnreachable,
			msgCode:    3,
			responderIP: ip8,
			targetIP:   ip8,
			want:       true,
		},
		{
			name:       "DestUnreachable IP不匹配返回false",
			msgType:    ipv4.ICMPTypeDestinationUnreachable,
			msgCode:    3,
			responderIP: ip10,
			targetIP:   ip8,
			want:       false,
		},
		{
			name:       "DestUnreachable code不匹配返回false",
			msgType:    ipv4.ICMPTypeDestinationUnreachable,
			msgCode:    1,
			responderIP: ip8,
			targetIP:   ip8,
			want:       false,
		},
		{
			name:       "DestUnreachable responderIP为nil返回false",
			msgType:    ipv4.ICMPTypeDestinationUnreachable,
			msgCode:    3,
			responderIP: nil,
			targetIP:   ip8,
			want:       false,
		},
		{
			name:       "未知ICMP类型返回false",
			msgType:    ipv4.ICMPType(0),
			msgCode:    0,
			responderIP: ip8,
			targetIP:   ip8,
			want:       false,
		},
		{
			name:       "EchoReply类型返回false",
			msgType:    ipv4.ICMPTypeEchoReply,
			msgCode:    0,
			responderIP: ip8,
			targetIP:   ip8,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isDestinationReached(tt.msgType, tt.msgCode, tt.responderIP, tt.targetIP))
		})
	}
}

// --- formatElapsedTime table-driven tests ---

func TestFormatElapsedTimeTable(t *testing.T) {
	tests := []struct {
		name   string
		input  time.Duration
		want   string
	}{
		{name: "零", input: 0, want: "0.000 ms"},
		{name: "15毫秒", input: 15 * time.Millisecond, want: "15.000 ms"},
		{name: "500微秒", input: 500 * time.Microsecond, want: "0.500 ms"},
		{name: "1秒", input: 1 * time.Second, want: "1000.000 ms"},
		{name: "1毫秒", input: 1 * time.Millisecond, want: "1.000 ms"},
		{name: "123微秒", input: 123 * time.Microsecond, want: "0.123 ms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatElapsedTime(tt.input))
		})
	}
}

// --- formatTracerouteHeader table-driven tests ---

func TestFormatTracerouteHeaderTable(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		targetIP   net.IP
		maxHops    int
		packetSize int
		want       string
	}{
		{
			name:       "标准格式",
			host:       "example.com",
			targetIP:   net.ParseIP("93.184.216.34"),
			maxHops:    30,
			packetSize: 60,
			want:       "traceroute to example.com (93.184.216.34), 30 hops max, 60 byte packets\n",
		},
		{
			name:       "自定义跳数和包大小",
			host:       "google.com",
			targetIP:   net.ParseIP("8.8.8.8"),
			maxHops:    20,
			packetSize: 128,
			want:       "traceroute to google.com (8.8.8.8), 20 hops max, 128 byte packets\n",
		},
		{
			name:       "IP地址作为主机名",
			host:       "10.0.0.1",
			targetIP:   net.ParseIP("10.0.0.1"),
			maxHops:    15,
			packetSize: 40,
			want:       "traceroute to 10.0.0.1 (10.0.0.1), 15 hops max, 40 byte packets\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatTracerouteHeader(tt.host, tt.targetIP, tt.maxHops, tt.packetSize))
		})
	}
}

// --- formatHopPrefix table-driven tests ---

func TestFormatHopPrefixTable(t *testing.T) {
	tests := []struct {
		name string
		ttl  int
		want string
	}{
		{name: "ttl=1 单数字", ttl: 1, want: " 1  "},
		{name: "ttl=9 单数字", ttl: 9, want: " 9  "},
		{name: "ttl=10 双数字", ttl: 10, want: "10  "},
		{name: "ttl=30 双数字", ttl: 30, want: "30  "},
		{name: "ttl=99 双数字", ttl: 99, want: "99  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatHopPrefix(tt.ttl))
		})
	}
}

// --- buildUDPAddr table-driven tests ---

func TestBuildUDPAddrTable(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		port int
		want string // expected addr.String()
	}{
		{name: "标准地址", ip: net.ParseIP("8.8.8.8"), port: 33437, want: "8.8.8.8:33437"},
		{name: "回环地址", ip: net.ParseIP("127.0.0.1"), port: 0, want: "127.0.0.1:0"},
		{name: "私有地址常用端口", ip: net.ParseIP("192.168.1.1"), port: 53, want: "192.168.1.1:53"},
		{name: "10网段", ip: net.ParseIP("10.0.0.1"), port: 33434, want: "10.0.0.1:33434"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := buildUDPAddr(tt.ip, tt.port)
			require.NotNil(t, addr)
			assert.Equal(t, tt.want, addr.String())
		})
	}
}
