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

// resolveIPv4 解析主机名并返回第一个 IPv4 地址
func resolveIPv4(host string) (net.IP, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("无法解析主机: %v", err)
	}

	for _, candidate := range ips {
		if candidate.To4() != nil {
			return candidate, nil
		}
	}

	return nil, fmt.Errorf("无法找到IPv4地址")
}

// calcPacketLoss 计算丢包率百分比
func calcPacketLoss(sent, received int) float64 {
	if sent == 0 {
		return 0.0
	}
	return float64(sent-received) / float64(sent) * 100
}

// timeStats 时间统计结果
type timeStats struct {
	Min time.Duration
	Max time.Duration
	Avg time.Duration
}

// calcTimeStats 从一组时间中计算 min/max/avg
func calcTimeStats(times []time.Duration) timeStats {
	if len(times) == 0 {
		return timeStats{}
	}

	minTime := times[0]
	maxTime := times[0]
	total := time.Duration(0)

	for _, d := range times {
		if d < minTime {
			minTime = d
		}
		if d > maxTime {
			maxTime = d
		}
		total += d
	}

	return timeStats{
		Min: minTime,
		Max: maxTime,
		Avg: total / time.Duration(len(times)),
	}
}

// toMillis 将 time.Duration 转换为毫秒（float64）
func toMillis(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1e6
}

// formatResultLine 格式化单次 ping 结果行
func formatResultLine(ip string, seq int, elapsed time.Duration) string {
	return fmt.Sprintf("%d bytes from %s: seq=%d time=%.3f ms",
		64, ip, seq, toMillis(elapsed))
}

// formatStatsSummary 格式化统计摘要
func formatStatsSummary(host string, sent, received int, minTime, avgTime, maxTime time.Duration) string {
	loss := calcPacketLoss(sent, received)

	result := fmt.Sprintf("\n--- %s ping statistics ---\n", host)
	result += fmt.Sprintf("%d packets transmitted, %d received, %.1f%% packet loss\n",
		sent, received, loss)
	if received > 0 {
		result += fmt.Sprintf("rtt min/avg/max = %.3f/%.3f/%.3f ms\n",
			toMillis(minTime), toMillis(avgTime), toMillis(maxTime))
	}
	return result
}

// normalizeDurations 将零值的 interval 和 timeout 替换为默认值
func normalizeDurations(interval, timeout time.Duration) (time.Duration, time.Duration) {
	if interval == 0 {
		interval = DefaultInterval
	}
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return interval, timeout
}

// Ping 执行ping操作
func Ping(host string, count int, interval, timeout time.Duration) error {
	interval, timeout = normalizeDurations(interval, timeout)

	// 解析主机地址
	ip, err := resolveIPv4(host)
	if err != nil {
		logger.Error("Ping无法解析主机 %s: %v", host, err)
		return err
	}

	fmt.Printf("PING %s (%s):\n", host, ip.String())

	sent := 0
	received := 0
	var elapsedTimes []time.Duration

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
		elapsedTimes = append(elapsedTimes, elapsed)

		fmt.Println(formatResultLine(ip.String(), sent, elapsed))

		if count > 0 && sent >= count {
			break
		}

		time.Sleep(interval)
	}

	// 统计信息
	stats := calcTimeStats(elapsedTimes)
	fmt.Print(formatStatsSummary(host, sent, received, stats.Min, stats.Avg, stats.Max))

	return nil
}
