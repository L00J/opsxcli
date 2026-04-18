package busybox

import (
	"fmt"
	"net"
	"os"
	"strings"

	psnet "github.com/shirou/gopsutil/v3/net"
)

// ShowInterfaces 显示网络接口信息（类似 ifconfig）
func ShowInterfaces() error {
	interfaces, err := psnet.Interfaces()
	if err != nil {
		return fmt.Errorf("获取网络接口失败: %v", err)
	}

	// 获取网络 I/O 统计信息
	ioCounters, _ := psnet.IOCounters(true)
	ioMap := make(map[string]psnet.IOCountersStat)
	for _, io := range ioCounters {
		ioMap[io.Name] = io
	}

	for _, iface := range interfaces {
		// 获取接口标志字符串和数值
		flagsStr := getInterfaceFlagsString(iface.Flags)
		flagsNum := getInterfaceFlagsNum(iface.Flags)

		// 第一行：接口名、flags、mtu
		fmt.Printf("%s: flags=%d<%s>  mtu %d\n", iface.Name, flagsNum, flagsStr, iface.MTU)

		// 显示 IP 地址（inet 在 ether 之前）
		for _, addr := range iface.Addrs {
			printIfconfigAddr(addr, iface.Name)
		}

		// 显示 MAC 地址和 txqueuelen
		if len(iface.HardwareAddr) > 0 {
			qlen := getInterfaceQlen(iface.Name)
			ifaceType := getInterfaceType(iface.Name)
			fmt.Printf("        ether %s  txqueuelen %d  (%s)\n", iface.HardwareAddr, qlen, ifaceType)
		} else {
			// loopback 接口
			qlen := getInterfaceQlen(iface.Name)
			fmt.Printf("        loop  txqueuelen %d  (Local Loopback)\n", qlen)
		}

		// 显示 RX/TX 统计信息
		if io, ok := ioMap[iface.Name]; ok {
			printIOStats("RX", io.PacketsRecv, io.BytesRecv, io.Errin, io.Dropin)
			printIOStats("TX", io.PacketsSent, io.BytesSent, io.Errout, io.Dropout)
		}

		fmt.Println()
	}

	return nil
}

// ShowRoutes 显示路由表（类似 route -n 或 ip route）
func ShowRoutes() error {
	fmt.Println("Kernel IP routing table")
	fmt.Printf("%-20s %-20s %-15s %-8s %-6s %-6s %-6s %s\n",
		"Destination", "Gateway", "Genmask", "Flags", "Metric", "Ref", "Use", "Iface")

	// 尝试读取 /proc/net/route
	if err := showRoutesFromProc(); err == nil {
		return nil
	}

	// 如果读取 proc 失败，显示提示信息
	fmt.Println("(无法读取路由表信息)")

	return nil
}

// ShowIPAddr 显示 IP 地址信息（类似 ip addr show）
func ShowIPAddr() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("获取网络接口失败: %v", err)
	}

	for idx, iface := range interfaces {
		// 获取接口的额外信息
		qdisc := getInterfaceQdisc(iface.Name)
		state := getInterfaceState(iface.Name)
		qlen := getInterfaceQlen(iface.Name)

		// 第一行：接口索引、名称、标志、mtu、qdisc、state、group、qlen
		fmt.Printf("%d: %s: <%s> mtu %d qdisc %s state %s group default",
			idx+1, iface.Name, getEnhancedFlagsString(iface.Flags), iface.MTU, qdisc, state)

		// 只有非 loopback 接口才显示 qlen
		if iface.Flags&net.FlagLoopback == 0 && qlen > 0 {
			fmt.Printf(" qlen %d", qlen)
		}
		fmt.Println()

		// 第二行：link 信息
		if iface.Flags&net.FlagLoopback != 0 {
			// loopback 接口
			loAddr := "00:00:00:00:00:00"
			if len(iface.HardwareAddr) > 0 {
				loAddr = iface.HardwareAddr.String()
			}
			fmt.Printf("    link/loopback %s brd %s\n", loAddr, loAddr)
		} else if len(iface.HardwareAddr) > 0 {
			// 以太网接口
			fmt.Printf("    link/ether %s brd ff:ff:ff:ff:ff:ff\n", iface.HardwareAddr)
		}

		// IP 地址信息
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				printIPAddress(addr, iface.Name)
			}
		}
	}

	return nil
}

// getInterfaceFlagsString 将 gopsutil 的 flags 数组转换为字符串
func getInterfaceFlagsString(flags []string) string {
	if len(flags) == 0 {
		return ""
	}
	// 转换为大写
	upperFlags := make([]string, len(flags))
	for i, f := range flags {
		upperFlags[i] = strings.ToUpper(f)
	}
	return strings.Join(upperFlags, ",")
}

// cidrToNetmask 将 CIDR 前缀长度转换为子网掩码
func cidrToNetmask(cidr string) string {
	var prefixLen int
	fmt.Sscanf(cidr, "%d", &prefixLen)

	if prefixLen == 0 || prefixLen > 32 {
		return "255.255.255.255"
	}

	mask := net.CIDRMask(prefixLen, 32)
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

// getFlagsString 将接口标志转换为字符串
func getFlagsString(flags net.Flags) string {
	var flagStrs []string

	if flags&net.FlagUp != 0 {
		flagStrs = append(flagStrs, "UP")
	}
	if flags&net.FlagBroadcast != 0 {
		flagStrs = append(flagStrs, "BROADCAST")
	}
	if flags&net.FlagLoopback != 0 {
		flagStrs = append(flagStrs, "LOOPBACK")
	}
	if flags&net.FlagPointToPoint != 0 {
		flagStrs = append(flagStrs, "POINTOPOINT")
	}
	if flags&net.FlagMulticast != 0 {
		flagStrs = append(flagStrs, "MULTICAST")
	}

	return strings.Join(flagStrs, ",")
}

// getEnhancedFlagsString 将接口标志转换为字符串（增强版，包含 LOWER_UP）
func getEnhancedFlagsString(flags net.Flags) string {
	var flagStrs []string

	if flags&net.FlagLoopback != 0 {
		flagStrs = append(flagStrs, "LOOPBACK")
	}
	if flags&net.FlagBroadcast != 0 {
		flagStrs = append(flagStrs, "BROADCAST")
	}
	if flags&net.FlagUp != 0 {
		flagStrs = append(flagStrs, "UP")
		// 如果接口是 UP 状态，通常也是 LOWER_UP
		flagStrs = append(flagStrs, "LOWER_UP")
	}
	if flags&net.FlagPointToPoint != 0 {
		flagStrs = append(flagStrs, "POINTOPOINT")
	}
	if flags&net.FlagMulticast != 0 {
		flagStrs = append(flagStrs, "MULTICAST")
	}

	return strings.Join(flagStrs, ",")
}

// showRoutesFromProc 从 /proc/net/route 读取路由信息
func showRoutesFromProc() error {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // 跳过表头和空行
		}

		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		iface := fields[0]
		dest := hexToIP(fields[1])
		gateway := hexToIP(fields[2])
		flags := fields[3]
		mask := hexToIP(fields[7])

		fmt.Printf("%-20s %-20s %-15s %-8s %-6s %-6s %-6s %s\n",
			dest, gateway, mask, flags, "0", "0", "0", iface)
	}

	return nil
}

// hexToIP 将十六进制字符串转换为 IP 地址
func hexToIP(hex string) string {
	var ip [4]byte
	fmt.Sscanf(hex, "%02X%02X%02X%02X", &ip[3], &ip[2], &ip[1], &ip[0])
	return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
}

// getInterfaceQdisc 获取接口的队列规则
func getInterfaceQdisc(ifaceName string) string {
	data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/qdisc", ifaceName))
	if err != nil {
		return "noqueue"
	}
	qdisc := strings.TrimSpace(string(data))
	if qdisc == "" {
		return "noqueue"
	}
	return qdisc
}

// getInterfaceState 获取接口状态
func getInterfaceState(ifaceName string) string {
	data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/operstate", ifaceName))
	if err != nil {
		return "UNKNOWN"
	}
	state := strings.TrimSpace(strings.ToUpper(string(data)))
	if state == "" {
		return "UNKNOWN"
	}
	return state
}

// getInterfaceQlen 获取接口队列长度
func getInterfaceQlen(ifaceName string) int {
	data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/tx_queue_len", ifaceName))
	if err != nil {
		return 0
	}
	var qlen int
	fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &qlen)
	return qlen
}

// printIPAddress 打印 IP 地址信息（包含 scope、broadcast 等）
func printIPAddress(addr net.Addr, ifaceName string) {
	ipNet, ok := addr.(*net.IPNet)
	if !ok {
		return
	}

	ip := ipNet.IP
	isIPv4 := ip.To4() != nil

	if isIPv4 {
		// IPv4 地址
		scope := getIPv4Scope(ip, ifaceName)
		fmt.Printf("    inet %s", addr.String())

		// 添加 broadcast 地址
		if scope != "host" {
			brd := getBroadcastAddr(ipNet)
			fmt.Printf(" brd %s", brd)
		}

		fmt.Printf(" scope %s %s\n", scope, ifaceName)
		fmt.Printf("       valid_lft forever preferred_lft forever\n")
	} else {
		// IPv6 地址
		scope := getIPv6Scope(ip)
		fmt.Printf("    inet6 %s scope %s", addr.String(), scope)

		// 对于非 link-local 地址，显示接口名
		if scope != "link" {
			fmt.Printf(" %s", ifaceName)
		}
		fmt.Println()
		fmt.Printf("       valid_lft forever preferred_lft forever\n")
	}
}

// getIPv4Scope 获取 IPv4 地址的 scope
func getIPv4Scope(ip net.IP, ifaceName string) string {
	if ip.IsLoopback() {
		return "host"
	}
	if ip.IsLinkLocalUnicast() {
		return "link"
	}
	return "global"
}

// getIPv6Scope 获取 IPv6 地址的 scope
func getIPv6Scope(ip net.IP) string {
	if ip.IsLoopback() {
		return "host"
	}
	if ip.IsLinkLocalUnicast() {
		return "link"
	}
	return "global"
}

// getBroadcastAddr 计算广播地址
func getBroadcastAddr(ipNet *net.IPNet) string {
	ip := ipNet.IP.To4()
	if ip == nil {
		return ""
	}

	mask := ipNet.Mask
	broadcast := make(net.IP, len(ip))
	for i := range ip {
		broadcast[i] = ip[i] | ^mask[i]
	}

	return broadcast.String()
}

// getInterfaceFlagsNum 获取接口标志的数值
func getInterfaceFlagsNum(flags []string) int {
	flagNum := 0
	for _, flag := range flags {
		switch strings.ToUpper(flag) {
		case "UP":
			flagNum |= 0x1
		case "BROADCAST":
			flagNum |= 0x2
		case "LOOPBACK":
			flagNum |= 0x8
		case "POINTOPOINT":
			flagNum |= 0x10
		case "RUNNING":
			flagNum |= 0x40
		case "MULTICAST":
			flagNum |= 0x1000
		}
	}
	return flagNum
}

// getInterfaceType 获取接口类型
func getInterfaceType(ifaceName string) string {
	// 根据接口名称判断类型
	if strings.HasPrefix(ifaceName, "lo") {
		return "Local Loopback"
	}
	if strings.HasPrefix(ifaceName, "eth") || strings.HasPrefix(ifaceName, "enp") || strings.HasPrefix(ifaceName, "ens") {
		return "Ethernet"
	}
	if strings.HasPrefix(ifaceName, "wlan") || strings.HasPrefix(ifaceName, "wlp") {
		return "Wireless"
	}
	if strings.HasPrefix(ifaceName, "docker") || strings.HasPrefix(ifaceName, "br-") {
		return "Bridge"
	}
	return "Unknown"
}

// printIfconfigAddr 打印 ifconfig 格式的地址信息
func printIfconfigAddr(addr psnet.InterfaceAddr, ifaceName string) {
	parts := strings.Split(addr.Addr, "/")
	if len(parts) == 0 {
		return
	}

	ip := parts[0]
	prefixLen := ""
	if len(parts) > 1 {
		prefixLen = parts[1]
	}

	// 判断 IPv4 还是 IPv6
	if strings.Contains(ip, ":") {
		// IPv6 地址
		scopeID := getIPv6ScopeID(ip)
		fmt.Printf("        inet6 %s  prefixlen %s  scopeid %s\n", ip, prefixLen, scopeID)
	} else {
		// IPv4 地址
		netmask := cidrToNetmask(prefixLen)
		fmt.Printf("        inet %s  netmask %s", ip, netmask)

		// 计算并显示广播地址（非 loopback）
		if !strings.HasPrefix(ifaceName, "lo") && prefixLen != "" {
			ipNet := &net.IPNet{
				IP:   net.ParseIP(ip),
				Mask: net.CIDRMask(parsePrefixLen(prefixLen), 32),
			}
			brd := getBroadcastAddr(ipNet)
			if brd != "" {
				fmt.Printf("  broadcast %s", brd)
			}
		}
		fmt.Println()
	}
}

// printIOStats 打印 RX/TX 统计信息
func printIOStats(direction string, packets, bytes, errors, dropped uint64) {
	// 格式化字节数为人类可读格式
	bytesStr := formatBytesNetwork(bytes)
	fmt.Printf("        %s packets %d  bytes %d (%s)\n", direction, packets, bytes, bytesStr)
	fmt.Printf("        %s errors %d  dropped %d overruns 0  ", direction, errors, dropped)
	if direction == "RX" {
		fmt.Printf("frame 0\n")
	} else {
		fmt.Printf("carrier 0  collisions 0\n")
	}
}

// getIPv6ScopeID 获取 IPv6 地址的 scope ID
func getIPv6ScopeID(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "0x0"
	}

	if ip.IsLoopback() {
		return "0x10<host>"
	}
	if ip.IsLinkLocalUnicast() {
		return "0x20<link>"
	}
	return "0x0<global>"
}

// parsePrefixLen 解析前缀长度
func parsePrefixLen(prefixStr string) int {
	var prefix int
	fmt.Sscanf(prefixStr, "%d", &prefix)
	if prefix <= 0 || prefix > 128 {
		return 24 // 默认值
	}
	return prefix
}

// formatBytesNetwork 格式化字节数为人类可读格式（用于网络统计）
func formatBytesNetwork(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}
