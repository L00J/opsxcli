package cloudhost

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// HostEntry 主机条目（hostname + IP）
type HostEntry struct {
	Hostname string
	IP       string
}

// ConsulServiceRegistration Consul 服务注册数据
type ConsulServiceRegistration struct {
	ID      string              `json:"id"`
	Name    string              `json:"name"`
	Address string              `json:"address"`
	Port    int                 `json:"port"`
	Checks  []ConsulServiceCheck `json:"checks"`
}

// ConsulServiceCheck Consul 健康检查配置
type ConsulServiceCheck struct {
	HTTP     string `json:"http"`
	Interval string `json:"interval"`
}

// ConsulHealthCheckResult Consul 健康检查结果
type ConsulHealthCheckResult struct {
	ServiceID string `json:"ServiceID"`
}

// CloudHostRegistry 云主机 Consul 注册器
type CloudHostRegistry struct {
	consulURL    string
	hosts        []HostEntry
	appPort      int
	nodeExpPort  int
	metricsPath  string
	httpClient   *http.Client
}

// NewCloudHostRegistry 创建云主机注册器
func NewCloudHostRegistry(consulURL string, hosts []HostEntry, appPort, nodeExpPort int, metricsPath string) *CloudHostRegistry {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return &CloudHostRegistry{
		consulURL:   strings.TrimSuffix(consulURL, "/"),
		hosts:       hosts,
		appPort:     appPort,
		nodeExpPort: nodeExpPort,
		metricsPath: metricsPath,
		httpClient:  httpClient,
	}
}

// checkEndpoint 检查 HTTP 端点是否可用
func (r *CloudHostRegistry) checkEndpoint(url, expectedText string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	if expectedText == "" {
		return true
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	return strings.Contains(string(body), expectedText)
}

// registerService 注册单个服务到 Consul
func (r *CloudHostRegistry) registerService(serviceID, serviceName, address string, port int, checkURL string) error {
	registration := ConsulServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Address: address,
		Port:    port,
		Checks: []ConsulServiceCheck{
			{
				HTTP:     checkURL,
				Interval: "5s",
			},
		},
	}

	jsonData, err := json.Marshal(registration)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/v1/agent/service/register", r.consulURL)
	req, err := http.NewRequest("PUT", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RegisterAppServices 注册应用服务（检查 metrics 端点）
func (r *CloudHostRegistry) RegisterAppServices() {
	fmt.Println("开始注册应用服务...")

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	skipCount := 0

	for _, host := range r.hosts {
		wg.Add(1)
		go func(h HostEntry) {
			defer wg.Done()

			promURL := fmt.Sprintf("http://%s:%d%s", h.IP, r.appPort, r.metricsPath)
			if r.checkEndpoint(promURL, "system_cpu_usage") {
				serviceID := fmt.Sprintf("%s-app", h.Hostname)
				if err := r.registerService(serviceID, "application", h.IP, r.appPort, promURL); err != nil {
					fmt.Printf("  ✗ 注册失败 %s (%s): %v\n", h.Hostname, h.IP, err)
				} else {
					mu.Lock()
					successCount++
					mu.Unlock()
					fmt.Printf("  ✓ 服务注册成功: application (id=%s, address=%s:%d)\n", serviceID, h.IP, r.appPort)
				}
			} else {
				mu.Lock()
				skipCount++
				mu.Unlock()
				fmt.Printf("  ⚠ 跳过 %s (%s): %s 不可用或不包含 system_cpu_usage\n", h.Hostname, h.IP, promURL)
			}
		}(host)
	}

	wg.Wait()
	fmt.Printf("\n应用服务注册完成：成功 %d，跳过 %d，共 %d 台主机\n", successCount, skipCount, len(r.hosts))
}

// RegisterNodeExporters 注册 node-exporter 服务
func (r *CloudHostRegistry) RegisterNodeExporters() {
	fmt.Println("\n开始注册 node-exporter 服务...")

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	skipCount := 0

	for _, host := range r.hosts {
		wg.Add(1)
		go func(h HostEntry) {
			defer wg.Done()

			metricsURL := fmt.Sprintf("http://%s:%d/metrics", h.IP, r.nodeExpPort)
			if r.checkEndpoint(metricsURL, "") {
				serviceID := fmt.Sprintf("%s-node-exporter", h.Hostname)
				if err := r.registerService(serviceID, "node-exporter", h.IP, r.nodeExpPort, metricsURL); err != nil {
					fmt.Printf("  ✗ 注册失败 %s (%s): %v\n", h.Hostname, h.IP, err)
				} else {
					mu.Lock()
					successCount++
					mu.Unlock()
					fmt.Printf("  ✓ 服务注册成功: node-exporter (id=%s, address=%s:%d)\n", serviceID, h.IP, r.nodeExpPort)
				}
			} else {
				mu.Lock()
				skipCount++
				mu.Unlock()
				fmt.Printf("  ⚠ 跳过 %s (%s): %s 不可用\n", h.Hostname, h.IP, metricsURL)
			}
		}(host)
	}

	wg.Wait()
	fmt.Printf("\nnode-exporter 注册完成：成功 %d，跳过 %d，共 %d 台主机\n", successCount, skipCount, len(r.hosts))
}

// CleanFailedInstances 清理 Consul 中状态为 critical 的实例
func (r *CloudHostRegistry) CleanFailedInstances() error {
	fmt.Println("\n开始清理失效实例...")

	// 等待 Consul 完成健康检查
	time.Sleep(3 * time.Second)

	url := fmt.Sprintf("%s/v1/health/state/critical", r.consulURL)
	resp, err := r.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("获取 critical 服务列表失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("获取 critical 服务列表失败: HTTP %d", resp.StatusCode)
	}

	var instances []ConsulHealthCheckResult
	if err := json.NewDecoder(resp.Body).Decode(&instances); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if len(instances) == 0 {
		fmt.Println("✓ 没有发现 critical 服务实例")
		return nil
	}

	fmt.Printf("找到 %d 个 critical 实例\n", len(instances))

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0

	for _, inst := range instances {
		if inst.ServiceID == "" {
			continue
		}
		wg.Add(1)
		go func(serviceID string) {
			defer wg.Done()

			deregURL := fmt.Sprintf("%s/v1/agent/service/deregister/%s", r.consulURL, serviceID)
			req, err := http.NewRequest("PUT", deregURL, nil)
			if err != nil {
				return
			}

			resp, err := r.httpClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				mu.Lock()
				successCount++
				mu.Unlock()
				fmt.Printf("  ✓ 已删除失效实例: %s\n", serviceID)
			} else {
				fmt.Printf("  ✗ 删除失败 %s: HTTP %d\n", serviceID, resp.StatusCode)
			}
		}(inst.ServiceID)
	}

	wg.Wait()
	fmt.Printf("\n清理完成：成功移除 %d/%d 个失效实例\n", successCount, len(instances))
	return nil
}

// ParseHosts 从命令行参数解析主机列表
// 格式: hostname:ip 或 hostname,ip
func ParseHosts(hostArgs []string) []HostEntry {
	var hosts []HostEntry
	for _, item := range hostArgs {
		var parts []string
		if strings.Contains(item, ":") {
			parts = strings.SplitN(item, ":", 2)
		} else if strings.Contains(item, ",") {
			parts = strings.SplitN(item, ",", 2)
		} else {
			fmt.Printf("  ✗ 无效的主机格式: %s，应为 hostname:ip 或 hostname,ip\n", item)
			continue
		}
		if len(parts) == 2 {
			hosts = append(hosts, HostEntry{
				Hostname: strings.TrimSpace(parts[0]),
				IP:       strings.TrimSpace(parts[1]),
			})
		}
	}
	return hosts
}

// LoadHostsFromFile 从文件加载主机列表
// 每行格式: hostname:ip 或 hostname,ip（支持 # 注释）
func LoadHostsFromFile(filePath string) ([]HostEntry, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	var hosts []HostEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var parts []string
		if strings.Contains(line, ":") {
			parts = strings.SplitN(line, ":", 2)
		} else if strings.Contains(line, ",") {
			parts = strings.SplitN(line, ",", 2)
		} else {
			fmt.Printf("  ✗ 文件行格式错误: %s\n", line)
			continue
		}
		if len(parts) == 2 {
			hosts = append(hosts, HostEntry{
				Hostname: strings.TrimSpace(parts[0]),
				IP:       strings.TrimSpace(parts[1]),
			})
		}
	}
	return hosts, nil
}
