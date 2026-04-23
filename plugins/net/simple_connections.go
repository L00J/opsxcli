package net

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SimpleConnectionKey 简化的连接标识
type SimpleConnectionKey struct {
	LocalAddr  string
	LocalPort  uint16
	RemoteAddr string
	RemotePort uint16
	State      string
}

// SimpleConnectionStats 简化的连接统计
type SimpleConnectionStats struct {
	Key      SimpleConnectionKey
	RxQueue  uint64
	TxQueue  uint64
	LastSeen time.Time
	Protocol string // TCP or UDP
}

// SimpleConnectionTracker 基于/proc的简单连接追踪器
type SimpleConnectionTracker struct {
	ctx         context.Context
	cancel      context.CancelFunc
	connections map[string]*SimpleConnectionStats
	mu          sync.RWMutex
}

// NewSimpleConnectionTracker 创建简单连接追踪器
func NewSimpleConnectionTracker() *SimpleConnectionTracker {
	ctx, cancel := context.WithCancel(context.Background())
	return &SimpleConnectionTracker{
		ctx:         ctx,
		cancel:      cancel,
		connections: make(map[string]*SimpleConnectionStats),
	}
}

// Start 启动连接追踪
func (sct *SimpleConnectionTracker) Start() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-sct.ctx.Done():
				return
			case <-ticker.C:
				sct.updateConnections()
			}
		}
	}()
}

// updateConnections 更新连接列表
func (sct *SimpleConnectionTracker) updateConnections() {
	sct.mu.Lock()
	defer sct.mu.Unlock()

	// 清空旧连接
	sct.connections = make(map[string]*SimpleConnectionStats)

	// 读取TCP连接
	sct.readTCPConnections("/proc/net/tcp", "TCP")
	sct.readTCPConnections("/proc/net/tcp6", "TCP6")

	// 读取UDP连接
	sct.readUDPConnections("/proc/net/udp", "UDP")
	sct.readUDPConnections("/proc/net/udp6", "UDP6")
}

// readTCPConnections 读取TCP连接
func (sct *SimpleConnectionTracker) readTCPConnections(filename, proto string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // 跳过header

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		localAddr, localPort := parseAddr(fields[1])
		remoteAddr, remotePort := parseAddr(fields[2])
		state := parseTCPState(fields[3])

		var rxQueue, txQueue uint64
		if len(fields) >= 5 {
			queueParts := strings.Split(fields[4], ":")
			if len(queueParts) == 2 {
				rxQueue, _ = strconv.ParseUint(queueParts[0], 16, 64)
				txQueue, _ = strconv.ParseUint(queueParts[1], 16, 64)
			}
		}

		key := fmt.Sprintf("%s:%d-%s:%d-%s", localAddr, localPort, remoteAddr, remotePort, proto)
		sct.connections[key] = &SimpleConnectionStats{
			Key: SimpleConnectionKey{
				LocalAddr:  localAddr,
				LocalPort:  localPort,
				RemoteAddr: remoteAddr,
				RemotePort: remotePort,
				State:      state,
			},
			RxQueue:  rxQueue,
			TxQueue:  txQueue,
			LastSeen: time.Now(),
			Protocol: proto,
		}
	}
}

// readUDPConnections 读取UDP连接
func (sct *SimpleConnectionTracker) readUDPConnections(filename, proto string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // 跳过header

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		localAddr, localPort := parseAddr(fields[1])
		remoteAddr, remotePort := parseAddr(fields[2])

		var rxQueue, txQueue uint64
		if len(fields) >= 5 {
			queueParts := strings.Split(fields[4], ":")
			if len(queueParts) == 2 {
				rxQueue, _ = strconv.ParseUint(queueParts[0], 16, 64)
				txQueue, _ = strconv.ParseUint(queueParts[1], 16, 64)
			}
		}

		key := fmt.Sprintf("%s:%d-%s:%d-%s", localAddr, localPort, remoteAddr, remotePort, proto)
		sct.connections[key] = &SimpleConnectionStats{
			Key: SimpleConnectionKey{
				LocalAddr:  localAddr,
				LocalPort:  localPort,
				RemoteAddr: remoteAddr,
				RemotePort: remotePort,
				State:      "UNCONN",
			},
			RxQueue:  rxQueue,
			TxQueue:  txQueue,
			LastSeen: time.Now(),
			Protocol: proto,
		}
	}
}

// parseAddr 解析地址和端口
func parseAddr(addrPort string) (string, uint16) {
	parts := strings.Split(addrPort, ":")
	if len(parts) != 2 {
		return "0.0.0.0", 0
	}

	// 解析十六进制IP地址
	ipHex := parts[0]
	portHex := parts[1]

	// 解析端口
	port, _ := strconv.ParseUint(portHex, 16, 16)

	// 解析IP (小端序)
	if len(ipHex) == 8 {
		// IPv4
		ip1, _ := strconv.ParseUint(ipHex[6:8], 16, 8)
		ip2, _ := strconv.ParseUint(ipHex[4:6], 16, 8)
		ip3, _ := strconv.ParseUint(ipHex[2:4], 16, 8)
		ip4, _ := strconv.ParseUint(ipHex[0:2], 16, 8)
		return fmt.Sprintf("%d.%d.%d.%d", ip1, ip2, ip3, ip4), uint16(port)
	}

	// IPv6 (简化处理)
	return "[IPv6]", uint16(port)
}

// parseTCPState 解析TCP状态
func parseTCPState(stateHex string) string {
	state, _ := strconv.ParseUint(stateHex, 16, 8)
	states := []string{"", "ESTABLISHED", "SYN_SENT", "SYN_RECV", "FIN_WAIT1",
		"FIN_WAIT2", "TIME_WAIT", "CLOSE", "CLOSE_WAIT", "LAST_ACK", "LISTEN", "CLOSING"}
	if int(state) < len(states) {
		return states[state]
	}
	return "UNKNOWN"
}

// GetTopConnections 获取TOP连接
func (sct *SimpleConnectionTracker) GetTopConnections(limit int) []*SimpleConnectionStats {
	sct.mu.RLock()
	defer sct.mu.RUnlock()

	conns := make([]*SimpleConnectionStats, 0, len(sct.connections))
	for _, conn := range sct.connections {
		// 只显示已建立的连接或有数据的连接
		if conn.Key.State == "ESTABLISHED" || conn.RxQueue > 0 || conn.TxQueue > 0 {
			connCopy := &SimpleConnectionStats{
				Key:      conn.Key,
				RxQueue:  conn.RxQueue,
				TxQueue:  conn.TxQueue,
				LastSeen: conn.LastSeen,
				Protocol: conn.Protocol,
			}
			conns = append(conns, connCopy)
		}
	}

	// 按队列大小排序
	sort.Slice(conns, func(i, j int) bool {
		totalI := conns[i].RxQueue + conns[i].TxQueue
		totalJ := conns[j].RxQueue + conns[j].TxQueue
		return totalI > totalJ
	})

	if limit > 0 && len(conns) > limit {
		conns = conns[:limit]
	}

	return conns
}

// Stop 停止追踪
func (sct *SimpleConnectionTracker) Stop() {
	sct.cancel()
}
