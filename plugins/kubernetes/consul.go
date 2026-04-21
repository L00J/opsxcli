package kubernetes

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// KubernetesMonitor Kubernetes服务监控器
type KubernetesMonitor struct {
	consulURL   string
	metricsPath string
	cacheDir    string
	skipCheck   bool
	client      *K8sHTTPClient
	httpClient  *http.Client
}

// ServiceRegistration Consul服务注册数据
type ServiceRegistration struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Port    int               `json:"port"`
	Checks  []ServiceCheck    `json:"checks"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// PodIPCache Pod IP缓存数据
type PodIPCache struct {
	PodIPs map[string]string `json:"pod_ips"` // podName -> podIP
}

// ServiceCheck Consul健康检查配置
type ServiceCheck struct {
	HTTP     string `json:"http"`
	Interval string `json:"interval"`
}

// HealthCheckResult Consul健康检查结果
type HealthCheckResult struct {
	ServiceID string `json:"ServiceID"`
}

// NewKubernetesMonitor 创建新的监控器
func NewKubernetesMonitor(kubeconfigPath, consulURL, metricsPath string, skipLocalCheck bool, insecureSkipVerify bool) *KubernetesMonitor {
	// 创建 HTTP 客户端
	k8sClient, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	// 测试连接 - 获取一个命名空间来验证连接
	var nsList struct {
		Items []struct {
			Metadata Metadata `json:"metadata"`
		} `json:"items"`
	}
	if err := k8sClient.Get("/api/v1/namespaces?limit=1", &nsList); err != nil {
		fmt.Printf("Kubernetes API 连接失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Kubernetes API 连接成功")

	// 创建缓存目录
	cacheDir := filepath.Join(os.TempDir(), "k8s_monitor_cache")
	os.MkdirAll(cacheDir, 0755)

	// 创建HTTP客户端（根据参数决定是否跳过TLS验证，10秒超时）
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		},
	}

	return &KubernetesMonitor{
		consulURL:   strings.TrimSuffix(consulURL, "/"),
		metricsPath: metricsPath,
		cacheDir:    cacheDir,
		skipCheck:   skipLocalCheck,
		client:      k8sClient,
		httpClient:  httpClient,
	}
}

// ClearCache 清除本地缓存
func (m *KubernetesMonitor) ClearCache() {
	fmt.Println("正在清除缓存...")
	err := os.RemoveAll(m.cacheDir)
	if err != nil {
		fmt.Printf("清除缓存失败: %v\n", err)
		return
	}
	os.MkdirAll(m.cacheDir, 0755)
	fmt.Println("✓ 缓存已清除")
}

// getServiceHash 生成服务唯一标识哈希
func (m *KubernetesMonitor) getServiceHash(namespace, name string) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s/%s", namespace, name)))
	return fmt.Sprintf("%x", hash)
}

// loadCache 加载缓存
func (m *KubernetesMonitor) loadCache(cacheKey string) (map[string]interface{}, error) {
	cacheFile := filepath.Join(m.cacheDir, cacheKey+".json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var cache map[string]interface{}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return cache, nil
}

// saveCache 保存缓存
func (m *KubernetesMonitor) saveCache(cacheKey string, data interface{}, ttl int) error {
	cacheFile := filepath.Join(m.cacheDir, cacheKey+".json")
	cache := map[string]interface{}{
		"expire": time.Now().Add(time.Duration(ttl) * time.Second).Unix(),
		"data":   data,
	}

	jsonData, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, jsonData, 0644)
}

// isCacheValid 检查缓存有效性
func (m *KubernetesMonitor) isCacheValid(cacheKey string) bool {
	cache, err := m.loadCache(cacheKey)
	if err != nil {
		return false
	}

	expire, ok := cache["expire"].(float64)
	if !ok {
		return false
	}

	return time.Now().Unix() < int64(expire)
}

// checkPrometheusEndpoint 检查Prometheus端点
func (m *KubernetesMonitor) checkPrometheusEndpoint(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	bodyStr := string(body)
	// 检查是否包含Prometheus格式的指标（更宽松的检查）
	// 包含 HELP 或 TYPE 注释，或者包含常见的指标
	return strings.Contains(bodyStr, "# HELP") ||
		strings.Contains(bodyStr, "# TYPE") ||
		strings.Contains(bodyStr, "system_cpu_usage") ||
		strings.Contains(bodyStr, "jvm_") ||
		strings.Contains(bodyStr, "process_")
}

// getPodIPsForService 获取Service背后所有Pod的IP
func (m *KubernetesMonitor) getPodIPsForService(namespace, serviceName string) (map[string]string, error) {
	// 获取Service
	var svc Service
	if err := m.client.Get(fmt.Sprintf("/api/v1/namespaces/%s/services/%s", namespace, serviceName), &svc); err != nil {
		return nil, err
	}

	if len(svc.Spec.Selector) == 0 {
		return map[string]string{}, nil // 无selector的Service
	}

	// 构建 label selector
	var selectors []string
	for k, v := range svc.Spec.Selector {
		selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
	}
	labelSelector := strings.Join(selectors, ",")

	// 使用selector查询Pods
	var podsList PodList
	apiPath := fmt.Sprintf("/api/v1/namespaces/%s/pods?labelSelector=%s", namespace, labelSelector)
	if err := m.client.Get(apiPath, &podsList); err != nil {
		return nil, err
	}

	// 收集Running状态的Pod IP
	podIPs := make(map[string]string)
	for _, pod := range podsList.Items {
		// 只收集 Running 状态且有 IP 的 Pod
		if pod.Status.Phase == "Running" && pod.Status.PodIP != "" {
			podIPs[pod.Metadata.Name] = pod.Status.PodIP
		}
	}

	return podIPs, nil
}

// processService 处理单个服务，返回需要注册的Pod列表（全量注册）
func (m *KubernetesMonitor) processService(namespace, name string, port int32) []*ServiceRegistration {
	// 获取当前Pod IP列表
	currentPodIPs, err := m.getPodIPsForService(namespace, name)
	if err != nil {
		fmt.Printf("⊗ [%s/%s] 获取Pod列表失败: %v\n", namespace, name, err)
		return nil
	}

	if len(currentPodIPs) == 0 {
		return nil
	}

	// 显示服务信息
	fmt.Printf("✓ [%s/%s] 发现 %d 个Pod\n", namespace, name, len(currentPodIPs))

	// 构建需要注册的Pod列表（全量）
	var registrations []*ServiceRegistration
	for podName, podIP := range currentPodIPs {
		prometheusURL := fmt.Sprintf("http://%s:%d%s", podIP, port, m.metricsPath)

		// 如果不跳过本地检查，验证Prometheus端点
		if !m.skipCheck && !m.checkPrometheusEndpoint(prometheusURL) {
			continue
		}

		registration := &ServiceRegistration{
			ID:      podName,
			Name:    "application",
			Address: podIP,
			Port:    int(port),
			Checks: []ServiceCheck{
				{
					HTTP:     prometheusURL,
					Interval: "5s",
				},
			},
			Meta: map[string]string{
				"namespace": namespace,
				"service":   name,
				"pod":       podName,
			},
		}
		registrations = append(registrations, registration)
	}

	return registrations
}

// registerService 注册单个服务到Consul
func (m *KubernetesMonitor) registerService(registration *ServiceRegistration) error {
	url := fmt.Sprintf("%s/v1/agent/service/register", m.consulURL)

	jsonData, err := json.Marshal(registration)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("注册失败[%d]: %s", resp.StatusCode, string(body))
	}

	return nil
}

// deregisterService 从Consul注销服务
func (m *KubernetesMonitor) deregisterService(serviceID string) {
	url := fmt.Sprintf("%s/v1/agent/service/deregister/%s", m.consulURL, serviceID)
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("✓ 已注销 [%s]\n", serviceID)
	}
}

// UpdateServices 增量更新服务
func (m *KubernetesMonitor) UpdateServices() error {
	fmt.Println("开始扫描Kubernetes服务...")

	// 获取所有服务
	var servicesList ServiceList
	if err := m.client.ListAllNamespaces("services", &servicesList); err != nil {
		return fmt.Errorf("获取服务列表失败: %v", err)
	}

	fmt.Printf("找到 %d 个服务，开始处理...\n", len(servicesList.Items))

	// 并行处理服务
	var wg sync.WaitGroup
	registrations := make(chan *ServiceRegistration, len(servicesList.Items)*10) // 增加容量以容纳多个Pod

	for _, svc := range servicesList.Items {
		if len(svc.Spec.Ports) == 0 {
			continue
		}

		wg.Add(1)
		go func(namespace, name string, port int32) {
			defer wg.Done()
			if regs := m.processService(namespace, name, port); regs != nil {
				for _, reg := range regs {
					registrations <- reg
				}
			}
		}(svc.Metadata.Namespace, svc.Metadata.Name, svc.Spec.Ports[0].Port)
	}

	// 等待所有处理完成
	go func() {
		wg.Wait()
		close(registrations)
	}()

	// 收集并注册服务
	successCount := 0
	totalCount := 0
	for reg := range registrations {
		totalCount++
		if err := m.registerService(reg); err != nil {
			fmt.Printf("✗ 注册失败 [%s]: %v\n", reg.ID, err)
		} else {
			successCount++
			fmt.Printf("✓ 注册成功 [%s] %s:%d\n", reg.ID, reg.Address, reg.Port)
		}
	}

	fmt.Printf("\n注册完成：成功 %d/%d\n", successCount, totalCount)

	// 等待5秒让Consul完成健康检查
	if totalCount > 0 {
		fmt.Println("\n等待5秒让Consul完成健康检查...")
		time.Sleep(5 * time.Second)

		// 自动清理失效实例
		fmt.Println("\n开始自动清理失效实例...")
		return m.CleanFailedInstances()
	}

	return nil
}

// CleanFailedInstances 清理失效实例
func (m *KubernetesMonitor) CleanFailedInstances() error {
	fmt.Println("开始清理失效实例...")

	url := fmt.Sprintf("%s/v1/health/state/critical", m.consulURL)
	resp, err := m.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("获取失效实例失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("获取失效实例失败[%d]", resp.StatusCode)
	}

	var criticalServices []HealthCheckResult
	if err := json.NewDecoder(resp.Body).Decode(&criticalServices); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if len(criticalServices) == 0 {
		fmt.Println("没有需要清理的失效实例")
		return nil
	}

	fmt.Printf("找到 %d 个失效实例\n", len(criticalServices))

	// 并行注销
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for _, instance := range criticalServices {
		wg.Add(1)
		go func(serviceID string) {
			defer wg.Done()

			deregisterURL := fmt.Sprintf("%s/v1/agent/service/deregister/%s", m.consulURL, serviceID)
			req, err := http.NewRequest("PUT", deregisterURL, nil)
			if err != nil {
				return
			}

			resp, err := m.httpClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				mu.Lock()
				successCount++
				mu.Unlock()
				fmt.Printf("✓ 已移除 [%s]\n", serviceID)
			}
		}(instance.ServiceID)
	}

	wg.Wait()

	fmt.Printf("\n清理完成：成功移除 %d/%d 个失效实例\n", successCount, len(criticalServices))
	return nil
}
