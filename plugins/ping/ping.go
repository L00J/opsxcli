// Package ping 实现网络连通性测试功能，支持 ICMP/UDP/TCP 三种探测模式
package ping

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"opsxcli/internal/logger"
)

const (
	DefaultInterval = 1 * time.Second
	DefaultTimeout  = 3 * time.Second
	DefaultCount    = 4
	DefaultPayload  = 56
	ProtocolICMP    = 1
	ProtocolICMPv6  = 58
)

// PingMode 表示 ping 使用的协议模式
type PingMode string

const (
	ModeICMP PingMode = "icmp" // 原生 ICMP（需 root）
	ModeUDP  PingMode = "udp"  // UDP 回退（非 root）
	ModeTCP  PingMode = "tcp"  // TCP 模式（兼容旧版）
)

// PingConfig ping 配置参数
type PingConfig struct {
	Host        string        // 目标主机
	Count       int           // 发送次数（-1 表示持续）
	Interval    time.Duration // 发送间隔
	Timeout     time.Duration // 单次超时
	Mode        PingMode      // 协议模式（空=自动选择）
	IPVersion   string        // "4" 或 "6"
	PayloadSize int           // 负载大小（字节）
}

// PingResult ping 结果
type PingResult struct {
	Host     string
	IP       string
	Sent     int
	Received int
	LossPct  float64
	MinRTT   time.Duration
	MaxRTT   time.Duration
	AvgRTT   time.Duration
	Jitter   time.Duration
	StdDev   time.Duration
	Mode     PingMode
	RTTs     []time.Duration
}

// NewPingConfig 创建默认 PingConfig
func NewPingConfig(host string) *PingConfig {
	return &PingConfig{
		Host:        host,
		Count:       DefaultCount,
		Interval:    DefaultInterval,
		Timeout:     DefaultTimeout,
		Mode:        "", // 自动选择
		IPVersion:   "4",
		PayloadSize: DefaultPayload,
	}
}

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

// resolveIP 解析主机名，支持 IPv4/IPv6 选择
func resolveIP(host string, ipVersion string) (net.IP, error) {
	// 如果已经是 IP 地址，直接解析
	if ip := net.ParseIP(host); ip != nil {
		if ipVersion == "6" && ip.To4() != nil {
			return nil, fmt.Errorf("期望IPv6但得到IPv4地址")
		}
		return ip, nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("无法解析主机: %v", err)
	}

	for _, candidate := range ips {
		if ipVersion == "6" {
			if candidate.To16() != nil && candidate.To4() == nil {
				return candidate, nil
			}
		} else {
			if candidate.To4() != nil {
				return candidate, nil
			}
		}
	}

	if ipVersion == "6" {
		return nil, fmt.Errorf("无法找到IPv6地址")
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

// calcJitter 计算抖动（相邻 RTT 差值的平均）
func calcJitter(times []time.Duration) time.Duration {
	if len(times) < 2 {
		return 0
	}
	var totalDiff time.Duration
	for i := 1; i < len(times); i++ {
		diff := times[i] - times[i-1]
		if diff < 0 {
			diff = -diff
		}
		totalDiff += diff
	}
	return totalDiff / time.Duration(len(times)-1)
}

// calcStdDev 计算标准差
func calcStdDev(times []time.Duration) time.Duration {
	if len(times) < 2 {
		return 0
	}
	// 计算平均值
	var total time.Duration
	for _, d := range times {
		total += d
	}
	avg := total / time.Duration(len(times))

	// 计算方差
	var sumSq float64
	for _, d := range times {
		diff := float64(d-avg) / float64(time.Millisecond)
		sumSq += diff * diff
	}
	variance := sumSq / float64(len(times)-1) // 样本标准差

	return time.Duration(math.Sqrt(variance) * float64(time.Millisecond))
}

// toMillis 将 time.Duration 转换为毫秒（float64）
func toMillis(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1e6
}

// formatResultLine 格式化单次 ping 结果行（TCP 模式兼容）
func formatResultLine(ip string, seq int, elapsed time.Duration) string {
	return fmt.Sprintf("%d bytes from %s: seq=%d time=%.3f ms",
		64, ip, seq, toMillis(elapsed))
}

// formatResultLineICMP 格式化 ICMP ping 结果行
func formatResultLineICMP(ip string, seq int, ttl int, elapsed time.Duration) string {
	return fmt.Sprintf("%d bytes from %s: icmp_seq=%d ttl=%d time=%.3f ms",
		64, ip, seq, ttl, toMillis(elapsed))
}

// formatStatsSummary 格式化统计摘要（旧版兼容）
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

// formatStatsSummaryEnhanced 格式化增强统计摘要
func formatStatsSummaryEnhanced(r *PingResult) string {
	loss := calcPacketLoss(r.Sent, r.Received)

	result := fmt.Sprintf("\n--- %s (%s) ping statistics ---\n", r.Host, r.IP)
	result += fmt.Sprintf("%d packets transmitted, %d received, %.1f%% packet loss\n",
		r.Sent, r.Received, loss)
	if r.Received > 0 {
		result += fmt.Sprintf("rtt min/avg/max = %.3f/%.3f/%.3f ms  jitter = %.3f ms  stddev = %.3f ms\n",
			toMillis(r.MinRTT), toMillis(r.AvgRTT), toMillis(r.MaxRTT),
			toMillis(r.Jitter), toMillis(r.StdDev))
	}
	result += fmt.Sprintf("mode: [%s]\n", r.Mode)
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

// selectPingMode 选择 ping 模式，空值自动选择
func selectPingMode(mode PingMode) PingMode {
	if mode != "" {
		return mode
	}
	// 自动选择：优先 ICMP，需要 root 权限
	if os.Getuid() == 0 {
		return ModeICMP
	}
	// 非 root：优先 UDP 回退
	return ModeUDP
}

// calcChecksum 计算 ICMP 校验和
func calcChecksum(data []byte) uint16 {
	// 确保偶数长度
	if len(data)%2 != 0 {
		data = append(data, 0)
	}

	var sum uint32
	for i := 0; i < len(data); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}

	// 折叠进位
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum)
}

// buildICMPEchoRequest 构建 ICMP Echo Request 报文
func buildICMPEchoRequest(seq, payloadSize int) []byte {
	// ICMP Echo Request: Type(1) + Code(1) + Checksum(2) + Identifier(2) + Sequence(2) + Payload
	totalSize := 8 + payloadSize
	data := make([]byte, totalSize)

	// Type = 8 (Echo Request)
	data[0] = 8
	// Code = 0
	data[1] = 0
	// Checksum = 0 (先填0，后面计算)
	data[2] = 0
	data[3] = 0
	// Identifier = 进程ID的低16位
	pid := os.Getpid() & 0xffff
	data[4] = byte(pid >> 8)
	data[5] = byte(pid & 0xff)
	// Sequence number
	data[6] = byte(seq >> 8)
	data[7] = byte(seq & 0xff)

	// 填充 payload（使用递增模式便于验证）
	for i := 0; i < payloadSize; i++ {
		data[8+i] = byte(i & 0xff)
	}

	// 计算并填充校验和
	checksum := calcChecksum(data)
	data[2] = byte(checksum >> 8)
	data[3] = byte(checksum & 0xff)

	return data
}

// RunPing 执行增强 ping（ICMP/UDP/TCP 自动选择）
func RunPing(cfg *PingConfig) (*PingResult, error) {
	// 规范化参数
	cfg.Interval, cfg.Timeout = normalizeDurations(cfg.Interval, cfg.Timeout)
	if cfg.Count == 0 {
		cfg.Count = DefaultCount
	}
	if cfg.PayloadSize <= 0 {
		cfg.PayloadSize = DefaultPayload
	}

	// 解析目标地址
	ip, err := resolveIP(cfg.Host, cfg.IPVersion)
	if err != nil {
		logger.Error("Ping无法解析主机 %s: %v", cfg.Host, err)
		return nil, err
	}

	// 选择模式
	mode := selectPingMode(cfg.Mode)
	cfg.Mode = mode

	result := &PingResult{
		Host: cfg.Host,
		IP:   ip.String(),
		Mode: mode,
	}

	fmt.Printf("PING %s (%s): %d data bytes, mode=[%s]\n",
		cfg.Host, ip.String(), cfg.PayloadSize, mode)

	switch mode {
	case ModeICMP:
		err = runICMPListen(cfg, ip, result)
	case ModeUDP:
		err = runUDPListen(cfg, ip, result)
	case ModeTCP:
		err = runTCPPing(cfg, ip, result)
	default:
		err = runUDPListen(cfg, ip, result)
	}

	if err != nil && result.Sent == 0 {
		return nil, err
	}

	// 计算统计
	result.LossPct = calcPacketLoss(result.Sent, result.Received)
	if len(result.RTTs) > 0 {
		stats := calcTimeStats(result.RTTs)
		result.MinRTT = stats.Min
		result.MaxRTT = stats.Max
		result.AvgRTT = stats.Avg
		result.Jitter = calcJitter(result.RTTs)
		result.StdDev = calcStdDev(result.RTTs)
	}

	fmt.Print(formatStatsSummaryEnhanced(result))
	return result, nil
}

// runICMPListen 使用原生 ICMP socket 执行 ping（需 root）
func runICMPListen(cfg *PingConfig, ip net.IP, result *PingResult) error {
	// 监听 ICMP 回复
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		logger.Warning("ICMP socket创建失败（需root权限），回退到UDP: %v", err)
		// 回退到 UDP
		result.Mode = ModeUDP
		cfg.Mode = ModeUDP
		return runUDPListen(cfg, ip, result)
	}
	defer conn.Close()

	dst := &net.IPAddr{IP: ip}
	sendICMP := func(seq int) error {
		msg := &icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  seq,
				Data: makePayload(cfg.PayloadSize),
			},
		}
		data, err := msg.Marshal(nil)
		if err != nil {
			return fmt.Errorf("构建ICMP包失败: %w", err)
		}
		_, err = conn.WriteTo(data, dst)
		return err
	}

	return runPingLoop(cfg, result, conn, sendICMP, parseICMPReply)
}

// runUDPListen 使用 UDP socket 执行 ping（非 root 回退方案）
func runUDPListen(cfg *PingConfig, ip net.IP, result *PingResult) error {
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		logger.Warning("UDP ICMP socket创建失败，回退到TCP: %v", err)
		result.Mode = ModeTCP
		cfg.Mode = ModeTCP
		return runTCPPing(cfg, ip, result)
	}
	defer conn.Close()

	dst := &net.UDPAddr{IP: ip, Port: 0}
	sendUDP := func(seq int) error {
		msg := &icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  seq,
				Data: makePayload(cfg.PayloadSize),
			},
		}
		data, err := msg.Marshal(nil)
		if err != nil {
			return fmt.Errorf("构建UDP ICMP包失败: %w", err)
		}
		_, err = conn.WriteTo(data, dst)
		return err
	}

	return runPingLoop(cfg, result, conn, sendUDP, parseUDPReply)
}

// runTCPPing 使用 TCP 连接测试连通性（旧版兼容）
func runTCPPing(cfg *PingConfig, ip net.IP, result *PingResult) error {
	for seq := 1; cfg.Count < 0 || seq <= cfg.Count; seq++ {
		result.Sent++
		start := time.Now()

		// 尝试常见端口
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip.String(), "80"), cfg.Timeout)
		if err != nil {
			conn, err = net.DialTimeout("tcp", net.JoinHostPort(ip.String(), "22"), cfg.Timeout)
		}
		if err != nil {
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

// sendFunc 发送探测包的函数类型
type sendFunc func(seq int) error

// parseFunc 解析回复包的函数类型
type parseFunc func(conn *icmp.PacketConn, timeout time.Duration, expectedID int) (seq int, ttl int, err error)

// runPingLoop 通用 ping 循环
func runPingLoop(cfg *PingConfig, result *PingResult, conn *icmp.PacketConn, send sendFunc, parse parseFunc) error {
	expectedID := os.Getpid() & 0xffff

	for seq := 1; cfg.Count < 0 || seq <= cfg.Count; seq++ {
		result.Sent++
		start := time.Now()

		// 发送探测包
		if err := send(seq); err != nil {
			logger.Warning("发送ICMP包失败 seq=%d: %v", seq, err)
			fmt.Printf("请求超时: seq=%d\n", seq)
			if cfg.Count > 0 && seq >= cfg.Count {
				break
			}
			time.Sleep(cfg.Interval)
			continue
		}

		// 等待回复
		remaining := cfg.Timeout - time.Since(start)
		if remaining <= 0 {
			fmt.Printf("请求超时: seq=%d\n", seq)
			if cfg.Count > 0 && seq >= cfg.Count {
				break
			}
			time.Sleep(cfg.Interval)
			continue
		}

		conn.SetReadDeadline(time.Now().Add(remaining))
		replySeq, ttl, err := parse(conn, remaining, expectedID)
		if err != nil {
			fmt.Printf("请求超时: seq=%d\n", seq)
			if cfg.Count > 0 && seq >= cfg.Count {
				break
			}
			time.Sleep(cfg.Interval)
			continue
		}

		elapsed := time.Since(start)
		result.Received++
		result.RTTs = append(result.RTTs, elapsed)
		fmt.Println(formatResultLineICMP(result.IP, replySeq, ttl, elapsed))

		if cfg.Count > 0 && seq >= cfg.Count {
			break
		}
		time.Sleep(cfg.Interval)
	}
	return nil
}

// parseICMPReply 解析 ICMP 回复（原始 socket）
func parseICMPReply(conn *icmp.PacketConn, timeout time.Duration, expectedID int) (seq int, ttl int, err error) {
	buf := make([]byte, 1500)
	n, _, readErr := conn.ReadFrom(buf)
	if readErr != nil {
		return 0, 0, fmt.Errorf("读取ICMP回复失败: %w", readErr)
	}

	msg, parseErr := icmp.ParseMessage(ProtocolICMP, buf[:n])
	if parseErr != nil {
		return 0, 0, fmt.Errorf("解析ICMP消息失败: %w", parseErr)
	}

	if msg.Type == ipv4.ICMPTypeEchoReply {
		echo, ok := msg.Body.(*icmp.Echo)
		if !ok {
			return 0, 0, fmt.Errorf("无效的Echo Reply")
		}
		// 只处理我们自己的回复
		if echo.ID == expectedID {
			return echo.Seq, 0, nil
		}
	}

	return 0, 0, fmt.Errorf("非Echo Reply消息: type=%d", msg.Type)
}

// parseUDPReply 解析 UDP ICMP 回复
func parseUDPReply(conn *icmp.PacketConn, timeout time.Duration, expectedID int) (seq int, ttl int, err error) {
	buf := make([]byte, 1500)
	n, _, readErr := conn.ReadFrom(buf)
	if readErr != nil {
		return 0, 0, fmt.Errorf("读取UDP ICMP回复失败: %w", readErr)
	}

	// UDP 模式下，前 20 字节是 IP 头
	// icmp.ParseMessage 需要偏移 IP 头之后
	msg, parseErr := icmp.ParseMessage(ProtocolICMP, buf[:n])
	if parseErr != nil {
		return 0, 0, fmt.Errorf("解析UDP ICMP消息失败: %w", parseErr)
	}

	if msg.Type == ipv4.ICMPTypeEchoReply {
		echo, ok := msg.Body.(*icmp.Echo)
		if !ok {
			return 0, 0, fmt.Errorf("无效的Echo Reply")
		}
		if echo.ID == expectedID {
			return echo.Seq, 0, nil
		}
	}

	return 0, 0, fmt.Errorf("非Echo Reply消息: type=%d", msg.Type)
}

// makePayload 生成指定大小的 payload
func makePayload(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i & 0xff)
	}
	return data
}

// Ping 执行ping操作（旧版兼容入口）
func Ping(host string, count int, interval, timeout time.Duration) error {
	cfg := NewPingConfig(host)
	cfg.Count = count
	cfg.Interval = interval
	cfg.Timeout = timeout
	cfg.Mode = ModeTCP // 旧版默认 TCP 模式

	_, err := RunPing(cfg)
	return err
}
