package kubernetes

import (
	"fmt"
	"strconv"
	"strings"
)

// PodMetrics Pod 资源使用指标
type PodMetrics struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Containers []struct {
		Name  string `json:"name"`
		Usage struct {
			CPU    string `json:"cpu"`
			Memory string `json:"memory"`
		} `json:"usage"`
	} `json:"containers"`
}

// PodMetricsList Pod 指标列表
type PodMetricsList struct {
	Items []PodMetrics `json:"items"`
}

// NodeMetrics Node 资源使用指标
type NodeMetrics struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Usage struct {
		CPU    string `json:"cpu"`
		Memory string `json:"memory"`
	} `json:"usage"`
}

// NodeMetricsList Node 指标列表
type NodeMetricsList struct {
	Items []NodeMetrics `json:"items"`
}

// TopPods 获取 Pod 资源使用情况
func TopPods(kubeconfigPath, namespace string, allNamespaces bool) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	var metricsList PodMetricsList
	var apiPath string

	if allNamespaces {
		// 获取所有命名空间的 Pod 指标
		apiPath = "/apis/metrics.k8s.io/v1beta1/pods"
	} else {
		if namespace == "" {
			namespace = "default"
		}
		apiPath = fmt.Sprintf("/apis/metrics.k8s.io/v1beta1/namespaces/%s/pods", namespace)
	}

	err = client.Get(apiPath, &metricsList)
	if err != nil {
		return fmt.Errorf("获取 Metrics 失败: %v (请确保已安装 Metrics Server)", err)
	}

	// 打印表头
	if allNamespaces {
		fmt.Printf("%-30s %-50s %-15s %-15s\n", "NAMESPACE", "NAME", "CPU(cores)", "MEMORY(bytes)")
	} else {
		fmt.Printf("%-50s %-15s %-15s\n", "NAME", "CPU(cores)", "MEMORY(bytes)")
	}

	// 打印数据
	for _, pod := range metricsList.Items {
		// 计算总 CPU 和 Memory
		totalCPU := int64(0)
		totalMemory := int64(0)

		for _, container := range pod.Containers {
			cpu := parseCPU(container.Usage.CPU)
			memory := parseMemory(container.Usage.Memory)
			totalCPU += cpu
			totalMemory += memory
		}

		cpuStr := formatCPU(totalCPU)
		memoryStr := formatMemory(totalMemory)

		if allNamespaces {
			fmt.Printf("%-30s %-50s %-15s %-15s\n",
				pod.Metadata.Namespace, pod.Metadata.Name, cpuStr, memoryStr)
		} else {
			fmt.Printf("%-50s %-15s %-15s\n",
				pod.Metadata.Name, cpuStr, memoryStr)
		}
	}

	return nil
}

// TopNodes 获取 Node 资源使用情况
func TopNodes(kubeconfigPath string) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	var metricsList NodeMetricsList
	apiPath := "/apis/metrics.k8s.io/v1beta1/nodes"

	err = client.Get(apiPath, &metricsList)
	if err != nil {
		return fmt.Errorf("获取 Metrics 失败: %v (请确保已安装 Metrics Server)", err)
	}

	// 打印表头
	fmt.Printf("%-50s %-15s %-15s\n", "NAME", "CPU(cores)", "MEMORY(bytes)")

	// 打印数据
	for _, node := range metricsList.Items {
		cpu := parseCPU(node.Usage.CPU)
		memory := parseMemory(node.Usage.Memory)

		cpuStr := formatCPU(cpu)
		memoryStr := formatMemory(memory)

		fmt.Printf("%-50s %-15s %-15s\n",
			node.Metadata.Name, cpuStr, memoryStr)
	}

	return nil
}

// parseCPU 解析 CPU 使用量 (例如: "100n" -> 100 纳核)
func parseCPU(cpu string) int64 {
	cpu = strings.TrimSpace(cpu)
	if cpu == "" {
		return 0
	}

	// 处理纳核 (n)
	if strings.HasSuffix(cpu, "n") {
		value, _ := strconv.ParseInt(strings.TrimSuffix(cpu, "n"), 10, 64)
		return value
	}

	// 处理毫核 (m)
	if strings.HasSuffix(cpu, "m") {
		value, _ := strconv.ParseInt(strings.TrimSuffix(cpu, "m"), 10, 64)
		return value * 1000000 // 转换为纳核
	}

	// 处理核数
	value, _ := strconv.ParseFloat(cpu, 64)
	return int64(value * 1000000000) // 转换为纳核
}

// parseMemory 解析内存使用量 (例如: "100Mi" -> 字节数)
func parseMemory(memory string) int64 {
	memory = strings.TrimSpace(memory)
	if memory == "" {
		return 0
	}

	// 处理 Ki
	if strings.HasSuffix(memory, "Ki") {
		value, _ := strconv.ParseInt(strings.TrimSuffix(memory, "Ki"), 10, 64)
		return value * 1024
	}

	// 处理 Mi
	if strings.HasSuffix(memory, "Mi") {
		value, _ := strconv.ParseInt(strings.TrimSuffix(memory, "Mi"), 10, 64)
		return value * 1024 * 1024
	}

	// 处理 Gi
	if strings.HasSuffix(memory, "Gi") {
		value, _ := strconv.ParseInt(strings.TrimSuffix(memory, "Gi"), 10, 64)
		return value * 1024 * 1024 * 1024
	}

	// 处理字节
	value, _ := strconv.ParseInt(memory, 10, 64)
	return value
}

// formatCPU 格式化 CPU 显示 (纳核 -> 毫核)
func formatCPU(nanocores int64) string {
	millicores := nanocores / 1000000
	return fmt.Sprintf("%dm", millicores)
}

// formatMemory 格式化内存显示 (字节 -> Mi)
func formatMemory(bytes int64) string {
	mi := bytes / (1024 * 1024)
	return fmt.Sprintf("%dMi", mi)
}
