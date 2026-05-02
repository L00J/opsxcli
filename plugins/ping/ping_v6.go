// Package ping 实现网络连通性测试功能，支持 IPv6 ICMPv6 探测
package ping

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"

	"opsxcli/internal/logger"
)

// isIPv6 判断 IP 地址是否为 IPv6（非 IPv4 映射）
func isIPv6(ip net.IP) bool {
	return ip.To16() != nil && ip.To4() == nil
}

// resolveIPv6 解析主机名并返回第一个 IPv6 地址
func resolveIPv6(host string) (net.IP, error) {
	// 如果已经是 IP 地址，直接判断
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() != nil {
			return nil, fmt.Errorf("期望IPv6但得到IPv4地址")
		}
		return ip, nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("无法解析主机: %v", err)
	}

	for _, candidate := range ips {
		if candidate.To16() != nil && candidate.To4() == nil {
			return candidate, nil
		}
	}

	return nil, fmt.Errorf("无法找到IPv6地址")
}

// buildICMPv6EchoRequest 构建 ICMPv6 Echo Request 报文
// ICMPv6 Echo Request: Type(128) + Code(0) + Checksum(2) + Identifier(2) + Sequence(2) + Payload
// 注意：ICMPv6 校验和由内核自动计算（与 ICMPv4 不同），无需手动填充
func buildICMPv6EchoRequest(seq, payloadSize int) *icmp.Message {
	return &icmp.Message{
		Type: ipv6.ICMPTypeEchoRequest,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  seq,
			Data: makePayload(payloadSize),
		},
	}
}

// buildICMPv6EchoRequestBytes 构建 ICMPv6 Echo Request 报文字节（用于纯函数测试）
func buildICMPv6EchoRequestBytes(seq, payloadSize int) ([]byte, error) {
	msg := buildICMPv6EchoRequest(seq, payloadSize)
	data, err := msg.Marshal(icmp.IPv6PseudoHeader(net.ParseIP("::1"), net.ParseIP("::1")))
	if err != nil {
		return nil, fmt.Errorf("构建ICMPv6包失败: %w", err)
	}
	return data, nil
}

// runICMPv6Listen 使用原生 ICMPv6 socket 执行 ping（需 root）
func runICMPv6Listen(cfg *PingConfig, ip net.IP, result *PingResult) error {
	conn, err := icmp.ListenPacket("ip6:ipv6-icmp", "::")
	if err != nil {
		logger.Warning("ICMPv6 socket创建失败（需root权限），回退到UDPv6: %v", err)
		// 回退到 UDPv6
		result.Mode = ModeUDP
		cfg.Mode = ModeUDP
		return runUDPv6Listen(cfg, ip, result)
	}
	defer conn.Close()

	dst := &net.IPAddr{IP: ip}
	sendICMPv6 := func(seq int) error {
		msg := buildICMPv6EchoRequest(seq, cfg.PayloadSize)
		data, marshalErr := msg.Marshal(nil)
		if marshalErr != nil {
			return fmt.Errorf("构建ICMPv6包失败: %w", marshalErr)
		}
		_, writeErr := conn.WriteTo(data, dst)
		return writeErr
	}

	return runPingLoop(cfg, result, conn, sendICMPv6, parseICMPv6Reply)
}

// runUDPv6Listen 使用 UDPv6 socket 执行 ping（非 root 回退方案）
func runUDPv6Listen(cfg *PingConfig, ip net.IP, result *PingResult) error {
	conn, err := icmp.ListenPacket("udp6", "::")
	if err != nil {
		logger.Warning("UDPv6 ICMP socket创建失败，回退到TCPv6: %v", err)
		result.Mode = ModeTCP
		cfg.Mode = ModeTCP
		return runTCPv6Ping(cfg, ip, result)
	}
	defer conn.Close()

	dst := &net.UDPAddr{IP: ip, Port: 0}
	sendUDPv6 := func(seq int) error {
		msg := buildICMPv6EchoRequest(seq, cfg.PayloadSize)
		data, marshalErr := msg.Marshal(icmp.IPv6PseudoHeader(net.ParseIP("::"), ip))
		if marshalErr != nil {
			return fmt.Errorf("构建UDPv6 ICMP包失败: %w", marshalErr)
		}
		_, writeErr := conn.WriteTo(data, dst)
		return writeErr
	}

	return runPingLoop(cfg, result, conn, sendUDPv6, parseUDPv6Reply)
}

// runTCPv6Ping 使用 TCPv6 连接测试连通性
func runTCPv6Ping(cfg *PingConfig, ip net.IP, result *PingResult) error {
	for seq := 1; cfg.Count < 0 || seq <= cfg.Count; seq++ {
		result.Sent++
		start := time.Now()

		// 尝试常见端口
		conn, dialErr := net.DialTimeout("tcp6", net.JoinHostPort(ip.String(), "80"), cfg.Timeout)
		if dialErr != nil {
			conn, dialErr = net.DialTimeout("tcp6", net.JoinHostPort(ip.String(), "22"), cfg.Timeout)
		}
		if dialErr != nil {
			fmt.Printf("请求超时: seq=%d\n", seq)
			if cfg.Count > 0 && seq >= cfg.Count {
				break
			}
			time.Sleep(cfg.Interval)
			continue
		}
		conn.Close()

		elapsed := time.Since(start)
		result.Received++
		result.RTTs = append(result.RTTs, elapsed)
		fmt.Println(formatResultLine(ip.String(), seq, elapsed))

		if cfg.Count > 0 && seq >= cfg.Count {
			break
		}
		time.Sleep(cfg.Interval)
	}
	return nil
}

// parseICMPv6Reply 解析 ICMPv6 回复（原始 socket）
func parseICMPv6Reply(conn *icmp.PacketConn, timeout time.Duration, expectedID int) (seq int, ttl int, err error) {
	buf := make([]byte, 1500)
	n, _, readErr := conn.ReadFrom(buf)
	if readErr != nil {
		return 0, 0, fmt.Errorf("读取ICMPv6回复失败: %w", readErr)
	}

	msg, parseErr := icmp.ParseMessage(ProtocolICMPv6, buf[:n])
	if parseErr != nil {
		return 0, 0, fmt.Errorf("解析ICMPv6消息失败: %w", parseErr)
	}

	if msg.Type == ipv6.ICMPTypeEchoReply {
		echo, ok := msg.Body.(*icmp.Echo)
		if !ok {
			return 0, 0, fmt.Errorf("无效的ICMPv6 Echo Reply")
		}
		// 只处理我们自己的回复
		if echo.ID == expectedID {
			return echo.Seq, 0, nil
		}
	}

	return 0, 0, fmt.Errorf("非ICMPv6 Echo Reply消息: type=%d", msg.Type)
}

// parseUDPv6Reply 解析 UDPv6 ICMP 回复
func parseUDPv6Reply(conn *icmp.PacketConn, timeout time.Duration, expectedID int) (seq int, ttl int, err error) {
	buf := make([]byte, 1500)
	n, _, readErr := conn.ReadFrom(buf)
	if readErr != nil {
		return 0, 0, fmt.Errorf("读取UDPv6 ICMP回复失败: %w", readErr)
	}

	msg, parseErr := icmp.ParseMessage(ProtocolICMPv6, buf[:n])
	if parseErr != nil {
		return 0, 0, fmt.Errorf("解析UDPv6 ICMP消息失败: %w", parseErr)
	}

	if msg.Type == ipv6.ICMPTypeEchoReply {
		echo, ok := msg.Body.(*icmp.Echo)
		if !ok {
			return 0, 0, fmt.Errorf("无效的ICMPv6 Echo Reply")
		}
		if echo.ID == expectedID {
			return echo.Seq, 0, nil
		}
	}

	return 0, 0, fmt.Errorf("非ICMPv6 Echo Reply消息: type=%d", msg.Type)
}
