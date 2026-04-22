package builtin

import (
	"net"
	"testing"
)

// TestCidrToNetmask 测试 CIDR 前缀长度转子网掩码
func TestCidrToNetmask(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected string
	}{
		{"24位前缀", "24", "255.255.255.0"},
		{"16位前缀", "16", "255.255.0.0"},
		{"8位前缀", "8", "255.0.0.0"},
		{"32位前缀(全1)", "32", "255.255.255.255"},
		{"0位前缀(边界)", "0", "255.255.255.255"},
		{"超过32(边界)", "33", "255.255.255.255"},
		{"1位前缀", "1", "128.0.0.0"},
		{"20位前缀", "20", "255.255.240.0"},
		{"空字符串", "0", "255.255.255.255"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cidrToNetmask(tt.cidr)
			if result != tt.expected {
				t.Errorf("cidrToNetmask(%q) = %q, 期望 %q", tt.cidr, result, tt.expected)
			}
		})
	}
}

// TestGetInterfaceFlagsString 测试 flags 数组转大写字符串
func TestGetInterfaceFlagsString(t *testing.T) {
	tests := []struct {
		name     string
		flags    []string
		expected string
	}{
		{"正常flags", []string{"up", "broadcast"}, "UP,BROADCAST"},
		{"单个flag", []string{"up"}, "UP"},
		{"空数组", []string{}, ""},
		{"nil输入", nil, ""},
		{"多个flags", []string{"up", "broadcast", "running", "multicast"}, "UP,BROADCAST,RUNNING,MULTICAST"},
		{"已大写flags", []string{"UP", "BROADCAST"}, "UP,BROADCAST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInterfaceFlagsString(tt.flags)
			if result != tt.expected {
				t.Errorf("getInterfaceFlagsString(%v) = %q, 期望 %q", tt.flags, result, tt.expected)
			}
		})
	}
}

// TestGetFlagsString 测试 net.Flags 转字符串
func TestGetFlagsString(t *testing.T) {
	tests := []struct {
		name     string
		flags    net.Flags
		expected string
	}{
		{"UP和Broadcast", net.FlagUp | net.FlagBroadcast, "UP,BROADCAST"},
		{"仅UP", net.FlagUp, "UP"},
		{"零值", 0, ""},
		{"全部flags", net.FlagUp | net.FlagBroadcast | net.FlagLoopback | net.FlagPointToPoint | net.FlagMulticast,
			"UP,BROADCAST,LOOPBACK,POINTOPOINT,MULTICAST"},
		{"Loopback和Up", net.FlagUp | net.FlagLoopback, "UP,LOOPBACK"},
		{"仅Multicast", net.FlagMulticast, "MULTICAST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFlagsString(tt.flags)
			if result != tt.expected {
				t.Errorf("getFlagsString(%v) = %q, 期望 %q", tt.flags, result, tt.expected)
			}
		})
	}
}

// TestGetEnhancedFlagsString 测试增强版 flags（UP 时加 LOWER_UP）
func TestGetEnhancedFlagsString(t *testing.T) {
	tests := []struct {
		name     string
		flags    net.Flags
		expected string
	}{
		{"UP状态加LOWER_UP", net.FlagUp | net.FlagBroadcast, "BROADCAST,UP,LOWER_UP"},
		{"仅UP", net.FlagUp, "UP,LOWER_UP"},
		{"非UP无LOWER_UP", net.FlagBroadcast, "BROADCAST"},
		{"零值", 0, ""},
		{"Loopback接口", net.FlagUp | net.FlagLoopback, "LOOPBACK,UP,LOWER_UP"},
		{"完整flags", net.FlagUp | net.FlagBroadcast | net.FlagLoopback | net.FlagPointToPoint | net.FlagMulticast,
			"LOOPBACK,BROADCAST,UP,LOWER_UP,POINTOPOINT,MULTICAST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnhancedFlagsString(tt.flags)
			if result != tt.expected {
				t.Errorf("getEnhancedFlagsString(%v) = %q, 期望 %q", tt.flags, result, tt.expected)
			}
		})
	}
}

// TestHexToIP 测试十六进制转 IP
func TestHexToIP(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		expected string
	}{
		{"127.0.0.1", "0100007F", "127.0.0.1"},
		{"0.0.0.0", "00000000", "0.0.0.0"},
		{"192.168.1.1", "0101A8C0", "192.168.1.1"},
		{"255.255.255.255", "FFFFFFFF", "255.255.255.255"},
		{"10.0.0.1", "0100000A", "10.0.0.1"},
		{"默认路由", "00000000", "0.0.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hexToIP(tt.hex)
			if result != tt.expected {
				t.Errorf("hexToIP(%q) = %q, 期望 %q", tt.hex, result, tt.expected)
			}
		})
	}
}

// TestGetIPv4Scope 测试 IPv4 地址 scope 判断
func TestGetIPv4Scope(t *testing.T) {
	tests := []struct {
		name      string
		ip        net.IP
		ifaceName string
		expected  string
	}{
		{"回环地址", net.ParseIP("127.0.0.1"), "lo", "host"},
		{"链路本地地址", net.ParseIP("169.254.1.1"), "eth0", "link"},
		{"全局地址", net.ParseIP("192.168.1.1"), "eth0", "global"},
		{"全局地址10段", net.ParseIP("10.0.0.1"), "eth0", "global"},
		{"全局地址172段", net.ParseIP("172.16.0.1"), "eth0", "global"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIPv4Scope(tt.ip, tt.ifaceName)
			if result != tt.expected {
				t.Errorf("getIPv4Scope(%v, %q) = %q, 期望 %q", tt.ip, tt.ifaceName, result, tt.expected)
			}
		})
	}
}

// TestGetIPv6Scope 测试 IPv6 地址 scope 判断
func TestGetIPv6Scope(t *testing.T) {
	tests := []struct {
		name     string
		ip       net.IP
		expected string
	}{
		{"回环地址::1", net.ParseIP("::1"), "host"},
		{"链路本地fe80", net.ParseIP("fe80::1"), "link"},
		{"全局地址", net.ParseIP("2001:db8::1"), "global"},
		{"全局地址fd00", net.ParseIP("fd00::1"), "global"},
		{"零地址", net.ParseIP("::"), "global"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIPv6Scope(tt.ip)
			if result != tt.expected {
				t.Errorf("getIPv6Scope(%v) = %q, 期望 %q", tt.ip, result, tt.expected)
			}
		})
	}
}

// TestParsePrefixLen 测试前缀长度解析
func TestParsePrefixLen(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"正常值64", "64", 64},
		{"正常值24", "24", 24},
		{"正常值32", "32", 32},
		{"零值返回默认24", "0", 24},
		{"负值返回默认24", "-1", 24},
		{"超过128返回默认24", "129", 24},
		{"空字符串返回默认24", "", 24},
		{"最大有效值128", "128", 128},
		{"正常值16", "16", 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePrefixLen(tt.input)
			if result != tt.expected {
				t.Errorf("parsePrefixLen(%q) = %d, 期望 %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFormatBytesNetwork 测试字节数格式化
func TestFormatBytesNetwork(t *testing.T) {
	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{"零字节", 0, "0 B"},
		{"小于1KiB", 512, "512 B"},
		{"正好1KiB", 1024, "1.0 KiB"},
		{"1MiB", 1048576, "1.0 MiB"},
		{"1GiB", 1073741824, "1.0 GiB"},
		{"1TiB", 1099511627776, "1.0 TiB"},
		{"1.5KiB", 1536, "1.5 KiB"},
		{"大数值", 107374182400, "100.0 GiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytesNetwork(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytesNetwork(%d) = %q, 期望 %q", tt.bytes, result, tt.expected)
			}
		})
	}
}
