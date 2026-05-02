package traceroute

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/ipv4"
)

// selectIPv4 从 IP 列表中选择第一个 IPv4 地址
func selectIPv4(ips []net.IP) net.IP {
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip
		}
	}
	return nil
}

// calculatePort 计算 traceroute 探测使用的 UDP 端口号
func calculatePort(ttl, probe int) int {
	return DefaultPort + ttl*3 + probe
}

// normalizeMaxHops 规范化 maxHops 参数，无效值使用默认值
func normalizeMaxHops(maxHops int) int {
	if maxHops <= 0 {
		return DefaultMaxHops
	}
	return maxHops
}

// normalizePacketSize 规范化 packetSize 参数，无效值使用默认值
func normalizePacketSize(packetSize int) int {
	if packetSize <= 0 {
		return DefaultPacketSize
	}
	return packetSize
}

// stripTrailingDot 移除字符串末尾的点号
func stripTrailingDot(s string) string {
	if len(s) > 0 && s[len(s)-1] == '.' {
		return s[:len(s)-1]
	}
	return s
}

// formatHostDisplay 格式化主机显示信息
// 如果有反向 DNS 记录，显示为 "hostname (ip)"，否则只显示 IP
func formatHostDisplay(ip net.IP) string {
	hostname := ip.String()
	if names, err := net.LookupAddr(ip.String()); err == nil && len(names) > 0 {
		name := stripTrailingDot(names[0])
		hostname = fmt.Sprintf("%s (%s)", name, ip.String())
	}
	return hostname
}

// isDestinationReached 判断 ICMP 响应是否表示已到达目标
// msgCode: ICMP 消息代码
// msgType: ICMP 消息类型 (使用 protocol number)
// responderIP: 响应者 IP
// targetIP: 目标 IP
func isDestinationReached(msgType ipv4.ICMPType, msgCode int, responderIP, targetIP net.IP) bool {
	switch msgType {
	case ipv4.ICMPTypeTimeExceeded:
		return false
	case ipv4.ICMPTypeDestinationUnreachable:
		if msgCode == 3 && responderIP != nil && responderIP.Equal(targetIP) {
			return true
		}
		return false
	default:
		return false
	}
}

// formatElapsedTime 将时间Duration格式化为毫秒字符串
func formatElapsedTime(d time.Duration) string {
	return fmt.Sprintf("%.3f ms", float64(d.Nanoseconds())/1e6)
}

// formatTracerouteHeader 生成 traceroute 的头部输出行
func formatTracerouteHeader(host string, targetIP net.IP, maxHops, packetSize int) string {
	return fmt.Sprintf("traceroute to %s (%s), %d hops max, %d byte packets\n",
		host, targetIP.String(), maxHops, packetSize)
}

// formatHopPrefix 格式化跳数前缀
func formatHopPrefix(ttl int) string {
	return fmt.Sprintf("%2d  ", ttl)
}

// buildUDPAddr 构建 UDP 地址
func buildUDPAddr(ip net.IP, port int) *net.UDPAddr {
	return &net.UDPAddr{
		IP:   ip,
		Port: port,
	}
}

// hopResult 表示单次跳探测的结果
type hopResult struct {
	TTL               int
	Times             []time.Duration
	ResponderIP       net.IP
	ReachedDestination bool
}

// formatHopResult 格式化单跳探测的输出行
// 返回格式化后的字符串（不包含 fmt.Printf 的副作用）
func formatHopResult(result hopResult) string {
	line := formatHopPrefix(result.TTL)

	// 格式化探测时间
	for _, t := range result.Times {
		if t < 0 {
			line += "*  "
		} else {
			line += formatElapsedTime(t) + "  "
		}
	}

	// 格式化响应者信息
	if result.ResponderIP != nil {
		hostname := formatHostDisplay(result.ResponderIP)
		line += " " + hostname
	}

	return line
}

// validateTracerouteParams 验证并规范化 traceroute 参数
// 返回规范化后的 maxHops 和 packetSize
func validateTracerouteParams(maxHops, packetSize int) (int, int) {
	return normalizeMaxHops(maxHops), normalizePacketSize(packetSize)
}

// formatProbeResult 格式化单次探测结果
// elapsed < 0 表示超时或错误
func formatProbeResult(elapsed time.Duration) string {
	if elapsed < 0 {
		return "*  "
	}
	return formatElapsedTime(elapsed) + "  "
}

// checkICMPReachDestination 检查 ICMP 消息是否表示到达目标
// 封装了完整的判断逻辑，包括消息类型和代码
func checkICMPReachDestination(msgType ipv4.ICMPType, msgCode int, responderIP, targetIP net.IP) bool {
	return isDestinationReached(msgType, msgCode, responderIP, targetIP)
}
