package kubernetes

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// 系统命名空间列表
var systemNamespaces = map[string]bool{
	"ingress-nginx":        true,
	"kube-system":          true,
	"kuboard":              true,
	"prometheus":           true,
	"devops":               true,
	"monitor":              true,
	"logging":              true,
	"istio-system":         true,
	"istio-egressgateway":  true,
	"istio-ingressgateway": true,
}

// DeploymentResource Deployment资源信息
type DeploymentResource struct {
	Namespace           string
	DeploymentName      string
	Replicas            int32
	AvailableReplicas   int32
	CapacityProvisioned string
	CPURequest          float64
	CPULimit            float64
	MemoryRequest       float64
	MemoryLimit         float64
}

// extractDeploymentName 从 Pod 名称中提取 Deployment 名称
func extractDeploymentName(podName string) string {
	if podName == "" || !strings.Contains(podName, "-") {
		return podName
	}

	parts := strings.Split(podName, "-")

	if len(parts) >= 3 {
		lastPart := parts[len(parts)-1]
		secondLast := parts[len(parts)-2]

		// 随机后缀通常是5个字符
		isRandomSuffix := len(lastPart) == 5 && isAlphaNumeric(lastPart)

		// 模板哈希通常是8-10个字符
		isTemplateHash := len(secondLast) >= 8 && len(secondLast) <= 10 &&
			isAlphaNumeric(secondLast) && !hasUpperCase(secondLast)

		if isRandomSuffix && isTemplateHash {
			return strings.Join(parts[:len(parts)-2], "-")
		}
	}

	if len(parts) >= 2 && isAlphaNumeric(parts[len(parts)-1]) && len(parts[len(parts)-1]) >= 3 {
		return strings.Join(parts[:len(parts)-1], "-")
	}

	return podName
}

// isAlphaNumeric 检查字符串是否为字母数字
func isAlphaNumeric(s string) bool {
	matched, _ := regexp.MatchString("^[a-zA-Z0-9]+$", s)
	return matched
}

// hasUpperCase 检查字符串是否包含大写字母
func hasUpperCase(s string) bool {
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			return true
		}
	}
	return false
}

// parseCPUValue 解析 CPU 值并转换为核数
func parseCPUValue(cpuStr string) float64 {
	if cpuStr == "" {
		return 0.0
	}

	if strings.HasSuffix(cpuStr, "m") {
		val, _ := strconv.ParseFloat(cpuStr[:len(cpuStr)-1], 64)
		return val / 1000.0
	}

	val, _ := strconv.ParseFloat(cpuStr, 64)
	return val
}

// parseMemoryValue 解析内存值并转换为 MB
func parseMemoryValue(memoryStr string) float64 {
	if memoryStr == "" {
		return 0.0
	}

	memoryStr = strings.ToUpper(memoryStr)

	if strings.HasSuffix(memoryStr, "KI") {
		val, _ := strconv.ParseFloat(memoryStr[:len(memoryStr)-2], 64)
		return val / 1024.0
	} else if strings.HasSuffix(memoryStr, "MI") {
		val, _ := strconv.ParseFloat(memoryStr[:len(memoryStr)-2], 64)
		return val
	} else if strings.HasSuffix(memoryStr, "GI") {
		val, _ := strconv.ParseFloat(memoryStr[:len(memoryStr)-2], 64)
		return val * 1024.0
	} else if strings.HasSuffix(memoryStr, "TI") {
		val, _ := strconv.ParseFloat(memoryStr[:len(memoryStr)-2], 64)
		return val * 1024.0 * 1024.0
	} else if strings.HasSuffix(memoryStr, "K") {
		val, _ := strconv.ParseFloat(memoryStr[:len(memoryStr)-1], 64)
		return val / 1000.0
	} else if strings.HasSuffix(memoryStr, "M") {
		val, _ := strconv.ParseFloat(memoryStr[:len(memoryStr)-1], 64)
		return val
	} else if strings.HasSuffix(memoryStr, "G") {
		val, _ := strconv.ParseFloat(memoryStr[:len(memoryStr)-1], 64)
		return val * 1000.0
	} else if strings.HasSuffix(memoryStr, "T") {
		val, _ := strconv.ParseFloat(memoryStr[:len(memoryStr)-1], 64)
		return val * 1000.0 * 1000.0
	}

	// 默认假设是字节
	val, _ := strconv.ParseFloat(memoryStr, 64)
	return val / (1024 * 1024)
}

// parseCapacityProvisioned 解析CapacityProvisioned注解
func parseCapacityProvisioned(capacityStr string) (float64, float64) {
	if capacityStr == "" || capacityStr == "N/A" {
		return 0.0, 0.0
	}

	cpuCores := 0.0
	memoryMB := 0.0

	// 匹配CPU: 数字 + vCPU
	cpuRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)vCPU`)
	if matches := cpuRegex.FindStringSubmatch(capacityStr); len(matches) > 1 {
		cpuCores, _ = strconv.ParseFloat(matches[1], 64)
	}

	// 匹配内存: 数字 + GB
	memRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)GB`)
	if matches := memRegex.FindStringSubmatch(capacityStr); len(matches) > 1 {
		memGB, _ := strconv.ParseFloat(matches[1], 64)
		memoryMB = memGB * 1024.0
	}

	return cpuCores, memoryMB
}

// formatCPUValue 格式化 CPU 值
func formatCPUValue(cpuCores float64) string {
	if cpuCores == 0 {
		return "0m"
	} else if cpuCores < 1 {
		millis := cpuCores * 1000
		if millis == float64(int(millis)) {
			return fmt.Sprintf("%dm", int(millis))
		}
		return fmt.Sprintf("%.1fm", millis)
	}
	return fmt.Sprintf("%.2f cores", cpuCores)
}

// formatMemoryValue 格式化内存值
func formatMemoryValue(memoryMB float64) string {
	if memoryMB == 0 {
		return "0 Mi"
	} else if memoryMB < 1024 {
		if memoryMB == float64(int(memoryMB)) {
			return fmt.Sprintf("%d Mi", int(memoryMB))
		}
		return fmt.Sprintf("%.1f Mi", memoryMB)
	}
	return fmt.Sprintf("%.2f Gi", memoryMB/1024)
}

// ExportResourceInventory 导出资源清单到 Excel
func ExportResourceInventory(kubeconfigPath string) error {
	// 创建 HTTP 客户端
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	fmt.Println("Kubernetes API 连接成功")

	// 获取所有 Deployment
	deployments := make(map[string]*DeploymentResource)
	var deploymentsList DeploymentList
	if err := client.ListAllNamespaces("deployments", &deploymentsList); err != nil {
		return fmt.Errorf("获取Deployment列表失败: %v", err)
	}

	for _, dep := range deploymentsList.Items {
		key := fmt.Sprintf("%s/%s", dep.Metadata.Namespace, dep.Metadata.Name)
		replicas := int32(0)
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}
		deployments[key] = &DeploymentResource{
			Namespace:         dep.Metadata.Namespace,
			DeploymentName:    dep.Metadata.Name,
			Replicas:          replicas,
			AvailableReplicas: dep.Status.AvailableReplicas,
		}
	}

	// 获取所有 Pod
	var podsList PodList
	if err := client.ListAllNamespaces("pods", &podsList); err != nil {
		return fmt.Errorf("获取Pod列表失败: %v", err)
	}

	fmt.Printf("找到 %d 个 Pod\n", len(podsList.Items))

	// 聚合数据
	deploymentData := make(map[string]*DeploymentResource)
	podCount := make(map[string]int)

	for _, pod := range podsList.Items {
		namespace := pod.Metadata.Namespace
		podName := pod.Metadata.Name

		deploymentName := extractDeploymentName(podName)
		key := fmt.Sprintf("%s/%s", namespace, deploymentName)

		if deploymentData[key] == nil {
			deploymentData[key] = &DeploymentResource{
				Namespace:      namespace,
				DeploymentName: deploymentName,
			}
		}

		data := deploymentData[key]

		// 获取注解
		capacityProvisioned := "N/A"
		if pod.Metadata.Annotations != nil {
			if cap, ok := pod.Metadata.Annotations["CapacityProvisioned"]; ok {
				capacityProvisioned = cap
			}
		}
		data.CapacityProvisioned = capacityProvisioned

		// 只记录第一个Pod的资源配置
		if podCount[key] == 0 {
			var cpuReq, cpuLim, memReq, memLim float64

			for _, container := range pod.Spec.Containers {
				if container.Resources.Requests != nil {
					if cpu, ok := container.Resources.Requests["cpu"]; ok {
						cpuReq += parseCPUValue(cpu)
					}
					if mem, ok := container.Resources.Requests["memory"]; ok {
						memReq += parseMemoryValue(mem)
					}
				}

				if container.Resources.Limits != nil {
					if cpu, ok := container.Resources.Limits["cpu"]; ok {
						cpuLim += parseCPUValue(cpu)
					}
					if mem, ok := container.Resources.Limits["memory"]; ok {
						memLim += parseMemoryValue(mem)
					}
				}
			}

			// 如果没有资源配置但有注解,从注解中解析
			if cpuReq == 0 && memReq == 0 && cpuLim == 0 && memLim == 0 && capacityProvisioned != "N/A" {
				annCPU, annMem := parseCapacityProvisioned(capacityProvisioned)
				if annCPU > 0 || annMem > 0 {
					cpuReq = annCPU
					cpuLim = annCPU
					memReq = annMem
					memLim = annMem
				}
			}

			data.CPURequest = cpuReq
			data.CPULimit = cpuLim
			data.MemoryRequest = memReq
			data.MemoryLimit = memLim
		}

		podCount[key]++

		// 从已知的Deployment中获取副本数
		if depInfo, ok := deployments[key]; ok {
			data.Replicas = depInfo.Replicas
			data.AvailableReplicas = depInfo.AvailableReplicas
		}
	}

	// 分为业务和系统命名空间
	var businessData, systemData []*DeploymentResource
	for _, data := range deploymentData {
		if systemNamespaces[data.Namespace] {
			systemData = append(systemData, data)
		} else {
			businessData = append(businessData, data)
		}
	}

	// 排序
	sort.Slice(businessData, func(i, j int) bool {
		if businessData[i].Namespace != businessData[j].Namespace {
			return businessData[i].Namespace < businessData[j].Namespace
		}
		return businessData[i].DeploymentName < businessData[j].DeploymentName
	})

	sort.Slice(systemData, func(i, j int) bool {
		if systemData[i].Namespace != systemData[j].Namespace {
			return systemData[i].Namespace < systemData[j].Namespace
		}
		return systemData[i].DeploymentName < systemData[j].DeploymentName
	})

	// 创建CSV文件
	filename := "deployments_resources.csv"
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建CSV文件失败: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{"Namespace", "Deployment Name", "Replicas", "Available Replicas",
		"Capacity Provisioned", "CPU Request", "CPU Limit", "Memory Request", "Memory Limit"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("写入表头失败: %v", err)
	}

	// 写入数据
	allData := append(businessData, systemData...)
	for _, data := range allData {
		row := []string{
			data.Namespace,
			data.DeploymentName,
			fmt.Sprintf("%d", data.Replicas),
			fmt.Sprintf("%d", data.AvailableReplicas),
			data.CapacityProvisioned,
			formatCPUValue(data.CPURequest),
			formatCPUValue(data.CPULimit),
			formatMemoryValue(data.MemoryRequest),
			formatMemoryValue(data.MemoryLimit),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("写入数据行失败: %v", err)
		}
	}

	fmt.Printf("\nCSV文件已保存为 '%s'\n", filename)
	fmt.Printf("共导出 %d 个 Deployment 的资源信息\n", len(allData))

	return nil
}
