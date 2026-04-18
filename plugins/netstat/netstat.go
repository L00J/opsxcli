package netstat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Netstat 显示网络连接状态
func Netstat(listen, all, tcp, udp, numeric, programs bool) error {
	// 如果没有指定协议，默认显示TCP和UDP
	if !tcp && !udp {
		tcp = true
		udp = true
	}

	// 获取连接信息
	connections, err := net.Connections("inet")
	if err != nil {
		return err
	}

	// 过滤连接
	var filtered []net.ConnectionStat
	for _, conn := range connections {
		// 协议过滤
		connType := getConnectionType(conn.Type)
		if tcp && connType == "tcp" {
			if listen && conn.Status == "LISTEN" {
				filtered = append(filtered, conn)
			} else if all || !listen {
				filtered = append(filtered, conn)
			}
		}
		if udp && connType == "udp" {
			// UDP 是无连接协议，没有 LISTEN 状态
			// 当使用 -l 参数时，显示所有绑定到本地端口的 UDP 连接
			if listen {
				// UDP 连接只要有本地端口就算是"监听"状态
				if conn.Laddr.Port > 0 {
					filtered = append(filtered, conn)
				}
			} else if all || !listen {
				filtered = append(filtered, conn)
			}
		}
	}

	// 按本地地址排序
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Laddr.IP != filtered[j].Laddr.IP {
			return filtered[i].Laddr.IP < filtered[j].Laddr.IP
		}
		return filtered[i].Laddr.Port < filtered[j].Laddr.Port
	})

	// 打印表头
	fmt.Println("Active Internet connections (only servers)")
	fmt.Printf("%-10s %-8s %-8s %-25s %-25s %-15s", "Proto", "Recv-Q", "Send-Q", "Local Address", "Foreign Address", "State")
	if programs {
		fmt.Printf(" %s", "PID/Program name")
	}
	fmt.Println()

	// 打印连接信息
	for _, conn := range filtered {
		printConnection(conn, numeric, programs)
	}

	return nil
}

// getConnectionType 获取连接类型字符串
func getConnectionType(connType uint32) string {
	switch connType {
	case 1: // TCP
		return "tcp"
	case 2: // UDP
		return "udp"
	default:
		return "unknown"
	}
}

// printConnection 打印连接信息
func printConnection(conn net.ConnectionStat, numeric, programs bool) {
	// 获取协议名（小写）并区分 IPv6
	proto := getConnectionType(conn.Type)
	if isIPv6(conn.Laddr.IP) {
		proto = proto + "6"
	}

	localAddr := formatAddr(conn.Laddr.IP, conn.Laddr.Port, numeric)
	foreignAddr := formatAddr(conn.Raddr.IP, conn.Raddr.Port, numeric)
	state := conn.Status
	if state == "" {
		state = "-"
	}

	fmt.Printf("%-10s %-8s %-8s %-25s %-25s %-15s",
		proto, "0", "0", localAddr, foreignAddr, state)

	if programs {
		pid := conn.Pid
		progName := getProgramName(pid)
		if progName != "" {
			fmt.Printf(" %d/%s", pid, progName)
		} else if pid > 0 {
			fmt.Printf(" %d", pid)
		}
	}
	fmt.Println()
}

// formatAddr 格式化地址
func formatAddr(ip string, port uint32, numeric bool) string {
	if ip == "" {
		ip = "*"
	}
	if port == 0 {
		return fmt.Sprintf("%s:*", ip)
	}

	if numeric {
		return fmt.Sprintf("%s:%d", ip, port)
	}

	// 尝试解析服务名
	service := ""
	if !numeric {
		service = getServiceName(port)
	}

	if service != "" {
		return fmt.Sprintf("%s:%s", ip, service)
	}
	return fmt.Sprintf("%s:%d", ip, port)
}

// getServiceName 获取服务名
func getServiceName(port uint32) string {
	// 常见端口映射
	portMap := map[uint32]string{
		22:   "ssh",
		25:   "smtp",
		80:   "http",
		443:  "https",
		3306: "mysql",
		6379: "redis",
		5601: "kibana",
		111:  "rpcbind",
	}

	if name, ok := portMap[port]; ok {
		return name
	}
	return ""
}

// getProgramName 获取程序名
func getProgramName(pid int32) string {
	if pid <= 0 {
		return ""
	}

	proc, err := process.NewProcess(pid)
	if err != nil {
		return ""
	}

	name, err := proc.Name()
	if err != nil {
		return ""
	}

	return name
}

// isIPv6 检查 IP 地址是否为 IPv6
func isIPv6(ip string) bool {
	return strings.Contains(ip, ":")
}
