package traceroute

import (
	"fmt"
	"net"
	"os/exec"
	"time"

	"opsxcli/internal/logger"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	DefaultMaxHops    = 30
	DefaultPacketSize = 60
	DefaultTimeout    = 3 * time.Second
	DefaultPort       = 33434 // traceroute 默认使用的 UDP 端口
)

// Traceroute 执行路由追踪
func Traceroute(host string, maxHops, packetSize int) error {
	if maxHops <= 0 {
		maxHops = DefaultMaxHops
	}
	if packetSize <= 0 {
		packetSize = DefaultPacketSize
	}

	// 解析目标地址
	ips, err := net.LookupIP(host)
	if err != nil {
		logger.Error("Traceroute无法解析主机 %s: %v", host, err)
		return fmt.Errorf("无法解析主机: %v", err)
	}

	var targetIP net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			targetIP = ip
			break
		}
	}

	if targetIP == nil {
		logger.Error("Traceroute无法找到 %s 的IPv4地址", host)
		return fmt.Errorf("无法找到IPv4地址")
	}

	fmt.Printf("traceroute to %s (%s), %d hops max, %d byte packets\n",
		host, targetIP.String(), maxHops, packetSize)

	// 创建接收 ICMP 消息的 socket
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		// 如果没有权限创建原始套接字，使用 UDP 方式（不需要 root）
		return tracerouteUDP(host, targetIP, maxHops, packetSize)
	}
	defer conn.Close()

	// 创建发送 UDP 包的 socket（用于 traceroute）
	udpConn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return fmt.Errorf("创建UDP socket失败: %v", err)
	}
	defer udpConn.Close()

	// 创建 IPv4 包连接以设置 TTL
	pc := ipv4.NewPacketConn(udpConn)

	// 逐跳追踪
	for ttl := 1; ttl <= maxHops; ttl++ {
		// 设置 TTL
		if err := pc.SetTTL(ttl); err != nil {
			logger.Error("设置TTL失败: %v", err)
			continue
		}

		fmt.Printf("%2d  ", ttl)

		var times []time.Duration
		var responderIP net.IP
		var reachedDestination bool

		// 发送3次探测
		for probe := 0; probe < 3; probe++ {
			// 重置 reachedDestination（每次探测独立判断）
			probeReached := false
			// 使用递增的端口号，便于识别响应
			port := DefaultPort + ttl*3 + probe
			remoteAddr := &net.UDPAddr{
				IP:   targetIP,
				Port: port,
			}

			// 发送 UDP 包
			start := time.Now()
			_, err := udpConn.WriteTo([]byte(""), remoteAddr)
			if err != nil {
				times = append(times, -1)
				fmt.Printf("*  ")
				continue
			}

			// 接收 ICMP 响应（设置超时）
			conn.SetReadDeadline(time.Now().Add(DefaultTimeout))
			reply := make([]byte, 1500)
			n, peer, err := conn.ReadFrom(reply)
			if err != nil {
				// 超时或错误
				times = append(times, -1)
				fmt.Printf("*  ")
				continue
			}

			elapsed := time.Since(start)
			times = append(times, elapsed)

			// 解析 ICMP 消息
			msg, err := icmp.ParseMessage(1, reply[:n])
			if err != nil {
				// 解析失败，但已经收到响应，记录 IP
				if peerAddr, ok := peer.(*net.IPAddr); ok {
					responderIP = peerAddr.IP
				}
				fmt.Printf("%.3f ms  ", float64(elapsed.Nanoseconds())/1e6)
				continue
			}

			// 获取响应者 IP
			if peerAddr, ok := peer.(*net.IPAddr); ok {
				responderIP = peerAddr.IP
			}

			// 检查 ICMP 消息类型，判断是否到达目标
			// ICMP Time Exceeded (11) = 中间路由器返回，继续追踪
			// ICMP Destination Unreachable (3) with Code 3 (Port Unreachable) = 目标返回，到达目标
			switch msg.Type {
			case ipv4.ICMPTypeTimeExceeded:
				// TTL 超时，继续追踪（即使 IP 是目标 IP，也可能是目标返回的 TTL 超时，需要继续）
				probeReached = false
			case ipv4.ICMPTypeDestinationUnreachable:
				// Code 在 Message 结构中，Code 3 = Port Unreachable
				// 如果收到目标 IP 的 Port Unreachable，说明到达了目标
				if msg.Code == 3 && responderIP != nil && responderIP.Equal(targetIP) {
					probeReached = true
				} else {
					// 其他类型的 Destination Unreachable 或不是目标 IP，继续追踪
					probeReached = false
				}
			default:
				// 其他类型，继续追踪
				probeReached = false
			}

			// 如果这次探测到达了目标，标记
			if probeReached {
				reachedDestination = true
			}

			// 显示时间
			fmt.Printf("%.3f ms  ", float64(elapsed.Nanoseconds())/1e6)
		}

		// 显示响应者信息
		if responderIP != nil {
			// 尝试获取主机名
			hostname := responderIP.String()
			if names, err := net.LookupAddr(responderIP.String()); err == nil && len(names) > 0 {
				// 移除末尾的点
				name := names[0]
				if len(name) > 0 && name[len(name)-1] == '.' {
					name = name[:len(name)-1]
				}
				hostname = fmt.Sprintf("%s (%s)", name, responderIP.String())
			}
			fmt.Printf(" %s\n", hostname)

			// 如果真正到达目标（收到 Port Unreachable），退出
			if reachedDestination {
				return nil
			}
		} else {
			fmt.Printf("\n")
		}
	}

	return nil
}

// tracerouteUDP 使用 UDP 方式实现 traceroute（不需要 root 权限，但功能受限）
func tracerouteUDP(host string, targetIP net.IP, maxHops, packetSize int) error {
	// 创建 UDP socket
	conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return fmt.Errorf("创建UDP socket失败: %v", err)
	}
	defer conn.Close()

	// 创建 ICMP 接收 socket（尝试，如果失败则使用简化版本）
	icmpConn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		// 如果没有权限，使用 TCP 连接方式（简化版，功能受限）
		return tracerouteTCP(host, targetIP, maxHops)
	}
	defer icmpConn.Close()

	pc := ipv4.NewPacketConn(conn)

	// 逐跳追踪
	for ttl := 1; ttl <= maxHops; ttl++ {
		// 设置 TTL
		if err := pc.SetTTL(ttl); err != nil {
			logger.Error("设置TTL失败: %v", err)
			continue
		}

		fmt.Printf("%2d  ", ttl)

		var times []time.Duration
		var responderIP net.IP
		var reachedDestination bool

		// 发送3次探测
		for probe := 0; probe < 3; probe++ {
			// 重置 reachedDestination（每次探测独立判断）
			probeReached := false
			port := DefaultPort + ttl*3 + probe
			remoteAddr := &net.UDPAddr{
				IP:   targetIP,
				Port: port,
			}

			start := time.Now()
			_, err := conn.WriteTo([]byte(""), remoteAddr)
			if err != nil {
				times = append(times, -1)
				fmt.Printf("*  ")
				continue
			}

			// 设置超时
			if err := icmpConn.SetReadDeadline(time.Now().Add(DefaultTimeout)); err != nil {
				times = append(times, -1)
				fmt.Printf("*  ")
				continue
			}

			// 接收 ICMP 响应
			reply := make([]byte, 1500)
			n, peer, err := icmpConn.ReadFrom(reply)
			if err != nil {
				times = append(times, -1)
				fmt.Printf("*  ")
				continue
			}

			elapsed := time.Since(start)
			times = append(times, elapsed)

			// 解析 ICMP 消息
			msg, err := icmp.ParseMessage(1, reply[:n])
			if err != nil {
				// 解析失败，但已经收到响应，记录 IP
				if peerAddr, ok := peer.(*net.IPAddr); ok {
					responderIP = peerAddr.IP
				}
				fmt.Printf("%.3f ms  ", float64(elapsed.Nanoseconds())/1e6)
				continue
			}

			// 获取响应者 IP
			if peerAddr, ok := peer.(*net.IPAddr); ok {
				responderIP = peerAddr.IP
			}

			// 检查 ICMP 消息类型，判断是否到达目标
			switch msg.Type {
			case ipv4.ICMPTypeTimeExceeded:
				// TTL 超时，继续追踪（即使 IP 是目标 IP，也可能是目标返回的 TTL 超时，需要继续）
				probeReached = false
			case ipv4.ICMPTypeDestinationUnreachable:
				// Code 在 Message 结构中，Code 3 = Port Unreachable
				// 如果收到目标 IP 的 Port Unreachable，说明到达了目标
				if msg.Code == 3 && responderIP != nil && responderIP.Equal(targetIP) {
					probeReached = true
				} else {
					// 其他类型的 Destination Unreachable 或不是目标 IP，继续追踪
					probeReached = false
				}
			default:
				// 其他类型，继续追踪
				probeReached = false
			}

			// 如果这次探测到达了目标，标记
			if probeReached {
				reachedDestination = true
			}

			fmt.Printf("%.3f ms  ", float64(elapsed.Nanoseconds())/1e6)
		}

		// 显示响应者信息
		if responderIP != nil {
			hostname := responderIP.String()
			if names, err := net.LookupAddr(responderIP.String()); err == nil && len(names) > 0 {
				name := names[0]
				if len(name) > 0 && name[len(name)-1] == '.' {
					name = name[:len(name)-1]
				}
				hostname = fmt.Sprintf("%s (%s)", name, responderIP.String())
			}
			fmt.Printf(" %s\n", hostname)

			// 如果真正到达目标（收到 Port Unreachable），退出
			if reachedDestination {
				return nil
			}
		} else {
			fmt.Printf("\n")
		}
	}

	return nil
}

// tracerouteTCP 使用 TCP 连接方式（简化版，功能受限，不需要 root 权限）
// 注意：这不是真正的 traceroute，只是通过尝试连接不同端口来模拟
func tracerouteTCP(host string, targetIP net.IP, maxHops int) error {
	// 使用系统 traceroute 命令（如果可用）
	if path, err := exec.LookPath("traceroute"); err == nil {
		logger.Error("无法创建原始套接字，请使用系统 traceroute 命令: %s %s", path, host)
		return fmt.Errorf("需要 root 权限或使用系统 traceroute 命令")
	}

	// 简化实现：尝试通过 TCP 连接来模拟（不准确，但可用）
	// 注意：这不是真正的 traceroute，只是显示连接延迟
	fmt.Printf("警告: 使用简化实现，结果可能不准确\n")
	fmt.Printf("建议: 使用系统 traceroute 命令或使用 root 权限运行\n\n")

	for hop := 1; hop <= maxHops; hop++ {
		fmt.Printf("%2d  ", hop)

		var times []time.Duration
		var reached bool

		// 尝试多个端口
		ports := []string{"80", "443", "22", "53"}
		for i := 0; i < 3 && i < len(ports); i++ {
			start := time.Now()
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(targetIP.String(), ports[i]), DefaultTimeout)
			if err != nil {
				times = append(times, -1)
				fmt.Printf("*  ")
				continue
			}
			conn.Close()
			elapsed := time.Since(start)
			times = append(times, elapsed)
			reached = true
			fmt.Printf("%.3f ms  ", float64(elapsed.Nanoseconds())/1e6)
		}

		// 填充到3次
		for len(times) < 3 {
			fmt.Printf("*  ")
			times = append(times, -1)
		}

		// 显示目标信息
		hostname := targetIP.String()
		if names, err := net.LookupAddr(targetIP.String()); err == nil && len(names) > 0 {
			name := names[0]
			if len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}
			hostname = fmt.Sprintf("%s (%s)", name, targetIP.String())
		}
		fmt.Printf(" %s\n", hostname)

		// 如果连接成功，退出（简化实现只显示一跳）
		if reached {
			break
		}
	}

	return nil
}
