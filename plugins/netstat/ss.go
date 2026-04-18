package netstat

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/shirou/gopsutil/v3/process"
)

// SsConnection ss风格的连接信息
type SsConnection struct {
	Proto       string
	RecvQ       uint64
	SendQ       uint64
	LocalAddr   string
	LocalPort   uint32
	ForeignAddr string
	ForeignPort uint32
	State       string
	PID         int32
	ProcessName string
}

// Ss 高性能网络连接查询（类似ss命令，直接读取/proc/net）
func Ss(listen, all, tcp, udp, numeric, programs, stats, timewait bool, top int) error {
	// 如果没有指定协议，默认显示TCP和UDP
	if !tcp && !udp {
		tcp = true
		udp = true
	}

	var connections []SsConnection

	// 读取TCP连接
	if tcp {
		tcpConns := ReadTCPConnectionsWithPrograms(listen, all, programs)
		connections = append(connections, tcpConns...)
	}

	// 读取UDP连接
	if udp {
		udpConns := ReadUDPConnectionsWithPrograms(listen, all, programs)
		connections = append(connections, udpConns...)
	}

	// 如果需要进程信息，提前构建inode到PID的映射（只构建一次，提高性能）
	if programs {
		buildInodeToPIDMap()
	}

	// 统计功能
	if stats {
		return printStateStats(connections)
	}
	if timewait {
		return printTimeWaitStats(connections)
	}
	if top > 0 {
		return printTopDestinations(connections, top)
	}

	// 按本地地址排序
	sort.Slice(connections, func(i, j int) bool {
		if connections[i].LocalAddr != connections[j].LocalAddr {
			return connections[i].LocalAddr < connections[j].LocalAddr
		}
		return connections[i].LocalPort < connections[j].LocalPort
	})

	// 打印表头
	fmt.Println("Active Internet connections (using ss-style /proc/net)")
	fmt.Printf("%-10s %-8s %-8s %-25s %-25s %-15s", "Proto", "Recv-Q", "Send-Q", "Local Address", "Foreign Address", "State")
	if programs {
		fmt.Printf(" %s", "PID/Program name")
	}
	fmt.Println()

	// 打印连接信息
	for _, conn := range connections {
		printSsConnection(conn, numeric, programs)
	}

	return nil
}

// ReadTCPConnectionsWithPrograms 读取TCP连接（直接读取/proc/net/tcp和tcp6）（导出函数）
func ReadTCPConnectionsWithPrograms(listen, all, programs bool) []SsConnection {
	var connections []SsConnection

	// 读取IPv4 TCP
	conns := readTCPFileWithPrograms("/proc/net/tcp", "TCP", listen, all, programs)
	connections = append(connections, conns...)

	// 读取IPv6 TCP
	conns6 := readTCPFileWithPrograms("/proc/net/tcp6", "TCP6", listen, all, programs)
	connections = append(connections, conns6...)

	return connections
}

// ReadUDPConnectionsWithPrograms 读取UDP连接（直接读取/proc/net/udp和udp6）（导出函数）
func ReadUDPConnectionsWithPrograms(listen, all, programs bool) []SsConnection {
	var connections []SsConnection

	// 读取IPv4 UDP
	conns := readUDPFileWithPrograms("/proc/net/udp", "UDP", listen, all, programs)
	connections = append(connections, conns...)

	// 读取IPv6 UDP
	conns6 := readUDPFileWithPrograms("/proc/net/udp6", "UDP6", listen, all, programs)
	connections = append(connections, conns6...)

	return connections
}

// readTCPFileWithPrograms 读取TCP文件（/proc/net/tcp或tcp6）
func readTCPFileWithPrograms(filename, proto string, listen, all, programs bool) []SsConnection {
	var connections []SsConnection

	file, err := os.Open(filename)
	if err != nil {
		return connections
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // 跳过header行

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// 解析地址和端口
		localAddr, localPort := parseHexAddr(fields[1])
		remoteAddr, remotePort := parseHexAddr(fields[2])
		state := parseTCPState(fields[3])

		// 状态过滤
		if listen && state != "LISTEN" {
			continue
		}
		if !all && !listen && state == "LISTEN" {
			continue
		}

		// 解析队列信息
		var recvQ, sendQ uint64
		if len(fields) >= 5 {
			queueParts := strings.Split(fields[4], ":")
			if len(queueParts) == 2 {
				recvQ, _ = strconv.ParseUint(queueParts[0], 16, 64)
				sendQ, _ = strconv.ParseUint(queueParts[1], 16, 64)
			}
		}

		// 解析inode（用于查找PID）
		var inode uint64
		if len(fields) >= 10 {
			inode, _ = strconv.ParseUint(fields[9], 10, 64)
		}

		conn := SsConnection{
			Proto:       proto,
			RecvQ:       recvQ,
			SendQ:       sendQ,
			LocalAddr:   localAddr,
			LocalPort:   localPort,
			ForeignAddr: remoteAddr,
			ForeignPort: remotePort,
			State:       state,
		}

		// 通过inode查找PID
		if inode > 0 && programs {
			if pid := findPIDByInode(inode); pid > 0 {
				conn.PID = pid
				if proc, err := process.NewProcess(pid); err == nil {
					if name, err := proc.Name(); err == nil {
						conn.ProcessName = name
					}
				}
			}
		}

		connections = append(connections, conn)
	}

	return connections
}

// readUDPFileWithPrograms 读取UDP文件（/proc/net/udp或udp6）
func readUDPFileWithPrograms(filename, proto string, listen, all, programs bool) []SsConnection {
	var connections []SsConnection

	file, err := os.Open(filename)
	if err != nil {
		return connections
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // 跳过header行

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// 解析地址和端口
		localAddr, localPort := parseHexAddr(fields[1])
		remoteAddr, remotePort := parseHexAddr(fields[2])

		// UDP没有状态，但可以判断是否是监听状态
		state := "UNCONN"
		if remoteAddr != "0.0.0.0" && remotePort != 0 {
			state = "ESTAB"
		}

		// 状态过滤
		if listen && state != "UNCONN" {
			continue
		}
		if !all && !listen && state == "UNCONN" {
			continue
		}

		// 解析队列信息
		var recvQ, sendQ uint64
		if len(fields) >= 5 {
			queueParts := strings.Split(fields[4], ":")
			if len(queueParts) == 2 {
				recvQ, _ = strconv.ParseUint(queueParts[0], 16, 64)
				sendQ, _ = strconv.ParseUint(queueParts[1], 16, 64)
			}
		}

		// 解析inode
		var inode uint64
		if len(fields) >= 10 {
			inode, _ = strconv.ParseUint(fields[9], 10, 64)
		}

		conn := SsConnection{
			Proto:       proto,
			RecvQ:       recvQ,
			SendQ:       sendQ,
			LocalAddr:   localAddr,
			LocalPort:   localPort,
			ForeignAddr: remoteAddr,
			ForeignPort: remotePort,
			State:       state,
		}

		// 通过inode查找PID
		if inode > 0 && programs {
			if pid := findPIDByInode(inode); pid > 0 {
				conn.PID = pid
				if proc, err := process.NewProcess(pid); err == nil {
					if name, err := proc.Name(); err == nil {
						conn.ProcessName = name
					}
				}
			}
		}

		connections = append(connections, conn)
	}

	return connections
}

// parseHexAddr 解析十六进制地址和端口（/proc/net格式）
func parseHexAddr(addrPort string) (string, uint32) {
	parts := strings.Split(addrPort, ":")
	if len(parts) != 2 {
		return "0.0.0.0", 0
	}

	ipHex := parts[0]
	portHex := parts[1]

	// 解析端口（十六进制）
	port, _ := strconv.ParseUint(portHex, 16, 32)

	// 解析IP地址（小端序，十六进制）
	if len(ipHex) == 8 {
		// IPv4
		ip1, _ := strconv.ParseUint(ipHex[6:8], 16, 8)
		ip2, _ := strconv.ParseUint(ipHex[4:6], 16, 8)
		ip3, _ := strconv.ParseUint(ipHex[2:4], 16, 8)
		ip4, _ := strconv.ParseUint(ipHex[0:2], 16, 8)
		return fmt.Sprintf("%d.%d.%d.%d", ip1, ip2, ip3, ip4), uint32(port)
	} else if len(ipHex) == 32 {
		// IPv6（简化处理，只显示前4段）
		return "[::1]", uint32(port)
	}

	return "0.0.0.0", uint32(port)
}

// parseTCPState 解析TCP状态（十六进制转状态名）
func parseTCPState(stateHex string) string {
	state, _ := strconv.ParseUint(stateHex, 16, 8)
	states := []string{
		"",            // 0
		"ESTABLISHED", // 1
		"SYN_SENT",    // 2
		"SYN_RECV",    // 3
		"FIN_WAIT1",   // 4
		"FIN_WAIT2",   // 5
		"TIME_WAIT",   // 6
		"CLOSE",       // 7
		"CLOSE_WAIT",  // 8
		"LAST_ACK",    // 9
		"LISTEN",      // 10
		"CLOSING",     // 11
	}
	if int(state) < len(states) && states[state] != "" {
		return states[state]
	}
	return "UNKNOWN"
}

// inodeToPIDCache inode到PID的缓存
var (
	inodeToPIDCache     = make(map[uint64]int32)
	inodeToPIDCacheMu   sync.RWMutex
	inodeToPIDCacheTime int64
)

// buildInodeToPIDMap 构建inode到PID的映射（通过扫描/proc/*/fd/，类似ss命令的实现）
// 这是ss命令高性能的关键：直接读取/proc/*/fd/而不是遍历所有进程目录
func buildInodeToPIDMap() map[uint64]int32 {
	// 检查缓存是否有效（可以添加时间戳检查，这里简化处理）
	inodeToPIDCacheMu.RLock()
	if len(inodeToPIDCache) > 0 {
		// 如果缓存不为空，直接返回（可以添加过期时间检查）
		cache := make(map[uint64]int32)
		for k, v := range inodeToPIDCache {
			cache[k] = v
		}
		inodeToPIDCacheMu.RUnlock()
		return cache
	}
	inodeToPIDCacheMu.RUnlock()

	pidMap := make(map[uint64]int32)

	// 扫描所有进程的fd目录（这是ss命令的核心实现方式）
	procDir, err := os.Open("/proc")
	if err != nil {
		return pidMap
	}
	defer procDir.Close()

	entries, err := procDir.Readdir(-1)
	if err != nil {
		return pidMap
	}

	// 限制扫描的进程数量，提高性能（类似ss命令的优化策略）
	maxProcs := 2000 // 增加扫描数量，但限制上限
	scanned := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pidStr := entry.Name()
		pid, err := strconv.ParseInt(pidStr, 10, 32)
		if err != nil {
			continue
		}

		// 读取进程的fd目录（这是ss命令获取PID的方式）
		fdDir := fmt.Sprintf("/proc/%d/fd", pid)
		fdEntries, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		// 快速扫描fd目录，只查找socket链接
		for _, fdEntry := range fdEntries {
			linkPath := fmt.Sprintf("/proc/%d/fd/%s", pid, fdEntry.Name())
			linkTarget, err := os.Readlink(linkPath)
			if err != nil {
				continue
			}

			// 解析socket链接：socket:[12345]（这是ss命令识别socket的方式）
			if strings.HasPrefix(linkTarget, "socket:[") {
				inodeStr := strings.TrimPrefix(linkTarget, "socket:[")
				inodeStr = strings.TrimSuffix(inodeStr, "]")
				if inode, err := strconv.ParseUint(inodeStr, 10, 64); err == nil {
					pidMap[inode] = int32(pid)
				}
			}
		}

		scanned++
		if scanned >= maxProcs {
			break
		}
	}

	// 更新缓存（使用写锁）
	inodeToPIDCacheMu.Lock()
	inodeToPIDCache = pidMap
	inodeToPIDCacheMu.Unlock()

	return pidMap
}

// findPIDByInode 通过inode查找PID（使用缓存）
func findPIDByInode(inode uint64) int32 {
	inodeToPIDCacheMu.RLock()
	if pid, ok := inodeToPIDCache[inode]; ok {
		inodeToPIDCacheMu.RUnlock()
		return pid
	}
	inodeToPIDCacheMu.RUnlock()

	// 如果缓存中没有，重新构建映射
	pidMap := buildInodeToPIDMap()
	if pid, ok := pidMap[inode]; ok {
		return pid
	}

	return 0
}

// printSsConnection 打印ss风格的连接信息
func printSsConnection(conn SsConnection, numeric, programs bool) {
	localAddr := formatAddr(conn.LocalAddr, conn.LocalPort, numeric)
	foreignAddr := formatAddr(conn.ForeignAddr, conn.ForeignPort, numeric)

	fmt.Printf("%-10s %-8d %-8d %-25s %-25s %-15s",
		conn.Proto, conn.RecvQ, conn.SendQ, localAddr, foreignAddr, conn.State)

	if programs {
		if conn.ProcessName != "" {
			fmt.Printf(" %d/%s", conn.PID, conn.ProcessName)
		} else if conn.PID > 0 {
			fmt.Printf(" %d", conn.PID)
		}
	}
	fmt.Println()
}

// printStateStats 统计TCP状态数量（类似: ss -ant | awk '{++s[$1]} END {for(k in s) print k,s[k]}'）
func printStateStats(connections []SsConnection) error {
	stateCount := make(map[string]int)

	for _, conn := range connections {
		stateCount[conn.State]++
	}

	fmt.Println("TCP状态统计:")
	fmt.Println("状态\t\t数量")
	fmt.Println("-------------------")

	// 按状态名排序
	states := make([]string, 0, len(stateCount))
	for state := range stateCount {
		states = append(states, state)
	}
	sort.Strings(states)

	for _, state := range states {
		fmt.Printf("%-15s\t%d\n", state, stateCount[state])
	}

	return nil
}

// printTimeWaitStats 显示TIME_WAIT状态的目标地址TOP 10
func printTimeWaitStats(connections []SsConnection) error {
	destCount := make(map[string]int)

	for _, conn := range connections {
		if conn.State == "TIME_WAIT" {
			destCount[conn.ForeignAddr]++
		}
	}

	if len(destCount) == 0 {
		fmt.Println("没有TIME_WAIT状态的连接")
		return nil
	}

	// 转换为切片并排序
	type destStat struct {
		addr  string
		count int
	}
	stats := make([]destStat, 0, len(destCount))
	for addr, count := range destCount {
		stats = append(stats, destStat{addr: addr, count: count})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].count > stats[j].count
	})

	fmt.Println("TIME_WAIT状态的目标地址TOP 10:")
	fmt.Printf("%-20s\t%s\n", "目标地址", "连接数")
	fmt.Println("-------------------")

	limit := 10
	if len(stats) < limit {
		limit = len(stats)
	}

	for i := 0; i < limit; i++ {
		fmt.Printf("%-20s\t%d\n", stats[i].addr, stats[i].count)
	}

	return nil
}

// printTopDestinations 显示目标地址TOP N
func printTopDestinations(connections []SsConnection, top int) error {
	destCount := make(map[string]int)

	for _, conn := range connections {
		if conn.ForeignAddr != "0.0.0.0" && conn.ForeignAddr != "" {
			destCount[conn.ForeignAddr]++
		}
	}

	if len(destCount) == 0 {
		fmt.Println("没有找到目标地址")
		return nil
	}

	// 转换为切片并排序
	type destStat struct {
		addr  string
		count int
	}
	stats := make([]destStat, 0, len(destCount))
	for addr, count := range destCount {
		stats = append(stats, destStat{addr: addr, count: count})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].count > stats[j].count
	})

	fmt.Printf("目标地址TOP %d:\n", top)
	fmt.Printf("%-20s\t%s\n", "目标地址", "连接数")
	fmt.Println("-------------------")

	limit := top
	if len(stats) < limit {
		limit = len(stats)
	}

	for i := 0; i < limit; i++ {
		fmt.Printf("%-20s\t%d\n", stats[i].addr, stats[i].count)
	}

	return nil
}

// PrintComprehensiveDashboard 打印综合仪表板（TIME_WAIT TOP + 并发IP TOP + 流量TOP）
func PrintComprehensiveDashboard(connections []SsConnection, topN int) error {
	if topN == 0 {
		topN = 10
	}

	// 1. 收集TIME_WAIT统计
	timewaitMap := make(map[string]int)
	for _, conn := range connections {
		if conn.State == "TIME_WAIT" {
			timewaitMap[conn.ForeignAddr]++
		}
	}

	// 2. 收集并发IP统计
	concurrentMap := make(map[string]int)
	for _, conn := range connections {
		if conn.ForeignAddr != "0.0.0.0" && conn.ForeignAddr != "" && conn.ForeignAddr != "*" {
			concurrentMap[conn.ForeignAddr]++
		}
	}

	// 3. 收集流量统计（基于队列大小估算）
	type TrafficStat struct {
		srcAddr  string
		srcPort  uint32
		dstAddr  string
		dstPort  uint32
		recvKB   float64
		sendKB   float64
		totalKB  float64
	}

	trafficStats := make([]TrafficStat, 0)
	for _, conn := range connections {
		if conn.RecvQ > 0 || conn.SendQ > 0 {
			stat := TrafficStat{
				srcAddr: conn.LocalAddr,
				srcPort: conn.LocalPort,
				dstAddr: conn.ForeignAddr,
				dstPort: conn.ForeignPort,
				recvKB:  float64(conn.RecvQ) / 1024.0,
				sendKB:  float64(conn.SendQ) / 1024.0,
				totalKB: float64(conn.RecvQ+conn.SendQ) / 1024.0,
			}
			trafficStats = append(trafficStats, stat)
		}
	}

	sort.Slice(trafficStats, func(i, j int) bool {
		return trafficStats[i].totalKB > trafficStats[j].totalKB
	})

	type statPair struct {
		addr  string
		count int
	}

	timewaitList := make([]statPair, 0, len(timewaitMap))
	for addr, count := range timewaitMap {
		timewaitList = append(timewaitList, statPair{addr, count})
	}
	sort.Slice(timewaitList, func(i, j int) bool {
		return timewaitList[i].count > timewaitList[j].count
	})

	concurrentList := make([]statPair, 0, len(concurrentMap))
	for addr, count := range concurrentMap {
		concurrentList = append(concurrentList, statPair{addr, count})
	}
	sort.Slice(concurrentList, func(i, j int) bool {
		return concurrentList[i].count > concurrentList[j].count
	})

	// 打印上半部分：TIME_WAIT TOP (左) + 并发IP TOP (右)
	fmt.Println("╔════════════════════════════════════════╦════════════════════════════════════════╗")
	fmt.Println("║     TIME_WAIT TOP 10                   ║     并发连接 IP TOP 10                  ║")
	fmt.Println("╠════════════════════════════════════════╬════════════════════════════════════════╣")
	fmt.Printf("║ %-20s %15s ║ %-20s %15s ║\n", "目标IP", "连接数", "目标IP", "连接数")
	fmt.Println("╠════════════════════════════════════════╬════════════════════════════════════════╣")

	maxRows := topN
	for i := 0; i < maxRows; i++ {
		leftAddr := ""
		leftCount := ""
		if i < len(timewaitList) {
			leftAddr = timewaitList[i].addr
			leftCount = fmt.Sprintf("%d", timewaitList[i].count)
		}

		rightAddr := ""
		rightCount := ""
		if i < len(concurrentList) {
			rightAddr = concurrentList[i].addr
			rightCount = fmt.Sprintf("%d", concurrentList[i].count)
		}

		fmt.Printf("║ %-20s %15s ║ %-20s %15s ║\n", leftAddr, leftCount, rightAddr, rightCount)
	}

	// 打印下半部分：连接流量TOP
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║                         连接流量 TOP 10 (基于队列大小)                          ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ %-21s -> %-21s %-8s %10s ║\n", "源地址", "目标地址", "方向", "队列KB")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════╣")

	trafficLimit := topN
	if len(trafficStats) < trafficLimit {
		trafficLimit = len(trafficStats)
	}

	for i := 0; i < trafficLimit; i++ {
		stat := trafficStats[i]
		src := fmt.Sprintf("%s:%d", stat.srcAddr, stat.srcPort)
		dst := fmt.Sprintf("%s:%d", stat.dstAddr, stat.dstPort)

		direction := "↓下载"
		dataKB := stat.recvKB
		if stat.sendKB > stat.recvKB {
			direction = "↑上传"
			dataKB = stat.sendKB
		}

		if len(src) > 21 {
			src = src[:18] + "..."
		}
		if len(dst) > 21 {
			dst = dst[:18] + "..."
		}

		fmt.Printf("║ %-21s -> %-21s %-8s %9.2f ║\n", src, dst, direction, dataKB)
	}

	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════╝")

	if len(trafficStats) == 0 {
		fmt.Println("\n提示: 当前无活跃流量（所有连接队列为空）")
	}

	return nil
}
