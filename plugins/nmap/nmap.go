package nmap

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"opsxcli/internal/logger"
)

const (
	DefaultTimeout = 2 * time.Second
)

// Scan 执行端口扫描
func Scan(host, ports string, timeout time.Duration, verbose bool) error {
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	// 解析目标地址
	ips, err := net.LookupIP(host)
	if err != nil {
		logger.Error("无法解析主机 %s: %v", host, err)
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
		logger.Error("无法找到 %s 的IPv4地址", host)
		return fmt.Errorf("无法找到IPv4地址")
	}

	// 解析端口范围
	portList, err := parsePorts(ports)
	if err != nil {
		logger.Error("无效的端口范围 %s: %v", ports, err)
		return fmt.Errorf("无效的端口范围: %v", err)
	}

	fmt.Printf("Starting Nmap scan for %s (%s)\n", host, targetIP.String())
	fmt.Printf("Scanning %d ports...\n\n", len(portList))

	// 并发扫描端口
	var wg sync.WaitGroup
	var mu sync.Mutex
	var openPorts []int

	// 限制并发数
	semaphore := make(chan struct{}, 100)

	for _, port := range portList {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量

		go func(p int) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量

			address := net.JoinHostPort(targetIP.String(), strconv.Itoa(p))
			conn, err := net.DialTimeout("tcp", address, timeout)
			if err == nil {
				conn.Close()
				mu.Lock()
				openPorts = append(openPorts, p)
				mu.Unlock()

				if verbose {
					service := guessService(p)
					fmt.Printf("Port %d/tcp open  %s\n", p, service)
				}
			} else if verbose {
				fmt.Printf("Port %d/tcp closed\n", p)
			}
		}(port)
	}

	wg.Wait()

	// 显示结果
	fmt.Printf("\nNmap scan report for %s (%s)\n", host, targetIP.String())
	fmt.Printf("Host is up (scanned in %.2fs).\n", float64(len(portList))*float64(timeout)/float64(time.Second))

	if len(openPorts) > 0 {
		fmt.Printf("\nPORT     STATE    SERVICE\n")
		for _, port := range openPorts {
			service := guessService(port)
			fmt.Printf("%-8d open     %s\n", port, service)
		}
	} else {
		fmt.Printf("\nNo open ports found.\n")
	}

	return nil
}

// parsePorts 解析端口范围字符串
// 支持格式：80,443,8080 或 1-1000 或 80,443,1-1000
func parsePorts(ports string) ([]int, error) {
	var result []int
	parts := strings.Split(ports, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 检查是否是范围
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("无效的端口范围: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("无效的起始端口: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("无效的结束端口: %s", rangeParts[1])
			}

			if start > end {
				return nil, fmt.Errorf("起始端口不能大于结束端口: %s", part)
			}

			if start < 1 || end > 65535 {
				return nil, fmt.Errorf("端口范围必须在 1-65535 之间: %s", part)
			}

			for p := start; p <= end; p++ {
				result = append(result, p)
			}
		} else {
			// 单个端口
			port, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("无效的端口号: %s", part)
			}

			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("端口号必须在 1-65535 之间: %d", port)
			}

			result = append(result, port)
		}
	}

	return result, nil
}

// guessService 根据端口号猜测服务名称
func guessService(port int) string {
	services := map[int]string{
		21:   "ftp",
		22:   "ssh",
		23:   "telnet",
		25:   "smtp",
		53:   "domain",
		80:   "http",
		110:  "pop3",
		143:  "imap",
		443:  "https",
		3306: "mysql",
		5432: "postgresql",
		6379: "redis",
		8080: "http-proxy",
		8443: "https-alt",
		9200: "elasticsearch",
	}

	if service, ok := services[port]; ok {
		return service
	}
	return "unknown"
}
