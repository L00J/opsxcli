package ping

import (
	"fmt"
	"net"
	"time"

	"opsxcli/internal/logger"
)

const (
	DefaultInterval = 1 * time.Second
	DefaultTimeout  = 3 * time.Second
)

// Ping 执行ping操作
func Ping(host string, count int, interval, timeout time.Duration) error {
	if interval == 0 {
		interval = DefaultInterval
	}
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	// 解析主机地址
	ips, err := net.LookupIP(host)
	if err != nil {
		logger.Error("Ping无法解析主机 %s: %v", host, err)
		return fmt.Errorf("无法解析主机: %v", err)
	}

	var ip net.IP
	for _, candidate := range ips {
		if candidate.To4() != nil {
			ip = candidate
			break
		}
	}

	if ip == nil {
		logger.Error("Ping无法找到 %s 的IPv4地址", host)
		return fmt.Errorf("无法找到IPv4地址")
	}

	fmt.Printf("PING %s (%s):\n", host, ip.String())

	sent := 0
	received := 0
	minTime := time.Duration(0)
	maxTime := time.Duration(0)
	totalTime := time.Duration(0)

	for {
		sent++
		start := time.Now()

		// 使用TCP连接测试连通性（不需要root权限）
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip.String(), "80"), timeout)
		if err != nil {
			// 尝试其他常见端口
			conn, err = net.DialTimeout("tcp", net.JoinHostPort(ip.String(), "22"), timeout)
		}
		if err != nil {
			fmt.Printf("请求超时\n")
			if count > 0 && sent >= count {
				break
			}
			time.Sleep(interval)
			continue
		}
		conn.Close()

		elapsed := time.Since(start)
		received++

		if minTime == 0 || elapsed < minTime {
			minTime = elapsed
		}
		if elapsed > maxTime {
			maxTime = elapsed
		}
		totalTime += elapsed

		fmt.Printf("%d bytes from %s: seq=%d time=%.3f ms\n",
			64, ip.String(), sent, float64(elapsed.Nanoseconds())/1e6)

		if count > 0 && sent >= count {
			break
		}

		time.Sleep(interval)
	}

	// 统计信息
	loss := float64(sent-received) / float64(sent) * 100
	avgTime := totalTime / time.Duration(received)

	fmt.Printf("\n--- %s ping statistics ---\n", host)
	fmt.Printf("%d packets transmitted, %d received, %.1f%% packet loss\n",
		sent, received, loss)
	if received > 0 {
		fmt.Printf("rtt min/avg/max = %.3f/%.3f/%.3f ms\n",
			float64(minTime.Nanoseconds())/1e6,
			float64(avgTime.Nanoseconds())/1e6,
			float64(maxTime.Nanoseconds())/1e6)
	}

	return nil
}
