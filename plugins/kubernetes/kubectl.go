package kubernetes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GetResources 获取资源列表
func GetResources(kubeconfigPath, resourceType, namespace string, allNamespaces bool, outputFormat string) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	// 资源类型别名映射
	resourceMap := map[string]string{
		"po":          "pods",
		"pod":         "pods",
		"pods":        "pods",
		"deploy":      "deployments",
		"deployment":  "deployments",
		"deployments": "deployments",
		"svc":         "services",
		"service":     "services",
		"services":    "services",
		"ing":         "ingresses",
		"ingress":     "ingresses",
		"ingresses":   "ingresses",
		"node":        "nodes",
		"nodes":       "nodes",
		"ns":          "namespaces",
		"namespace":   "namespaces",
		"namespaces":  "namespaces",
	}

	fullType := resourceMap[resourceType]
	if fullType == "" {
		return fmt.Errorf("不支持的资源类型: %s", resourceType)
	}

	wide := (outputFormat == "wide")

	switch fullType {
	case "pods":
		return getPods(client, namespace, allNamespaces, wide)
	case "deployments":
		return getDeployments(client, namespace, allNamespaces)
	case "services":
		return getServices(client, namespace, allNamespaces)
	case "ingresses":
		return getIngresses(client, namespace, allNamespaces)
	case "nodes":
		return getNodes(client)
	case "namespaces":
		return getNamespaces(client)
	default:
		return fmt.Errorf("暂不支持的资源类型: %s", fullType)
	}
}

// getPods 获取 Pods 列表
func getPods(client *K8sHTTPClient, namespace string, allNamespaces bool, wide bool) error {
	var podsList PodList
	var err error

	if allNamespaces {
		err = client.ListAllNamespaces("pods", &podsList)
	} else {
		if namespace == "" {
			namespace = "default"
		}
		apiPath := fmt.Sprintf("/api/v1/namespaces/%s/pods", namespace)
		err = client.Get(apiPath, &podsList)
	}

	if err != nil {
		return err
	}

	// 打印表头
	if wide {
		if allNamespaces {
			fmt.Printf("%-15s %-50s %-10s %-15s %-10s %-10s %-20s %-50s %-20s %-20s\n",
				"NAMESPACE", "NAME", "READY", "STATUS", "RESTARTS", "AGE", "IP", "NODE", "NOMINATED NODE", "READINESS GATES")
		} else {
			fmt.Printf("%-50s %-10s %-15s %-10s %-10s %-20s %-50s %-20s %-20s\n",
				"NAME", "READY", "STATUS", "RESTARTS", "AGE", "IP", "NODE", "NOMINATED NODE", "READINESS GATES")
		}
	} else {
		if allNamespaces {
			fmt.Printf("%-30s %-50s %-10s %-15s %-10s %-10s\n",
				"NAMESPACE", "NAME", "READY", "STATUS", "RESTARTS", "AGE")
		} else {
			fmt.Printf("%-50s %-10s %-15s %-10s %-10s\n",
				"NAME", "READY", "STATUS", "RESTARTS", "AGE")
		}
	}

	// 打印数据
	for _, pod := range podsList.Items {
		// 计算 READY 状态
		readyCount := 0
		totalContainers := len(pod.Spec.Containers)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyCount++
			}
		}
		ready := fmt.Sprintf("%d/%d", readyCount, totalContainers)

		// 获取 STATUS
		status := pod.Status.Phase
		if status == "" {
			status = "Unknown"
		}

		// 计算 RESTARTS
		restarts := int32(0)
		for _, cs := range pod.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}

		// 计算 AGE
		age := formatAge(pod.Metadata.CreationTimestamp)

		if wide {
			// Wide 输出格式
			podIP := pod.Status.PodIP
			if podIP == "" {
				podIP = "<none>"
			}
			node := pod.Spec.NodeName
			if node == "" {
				node = "<none>"
			}
			nominated := pod.Status.NominatedNodeName
			if nominated == "" {
				nominated = "<none>"
			}

			if allNamespaces {
				fmt.Printf("%-15s %-50s %-10s %-15s %-10d %-10s %-20s %-50s %-20s %-20s\n",
					pod.Metadata.Namespace, pod.Metadata.Name, ready, status, restarts, age,
					podIP, node, nominated, "<none>")
			} else {
				fmt.Printf("%-50s %-10s %-15s %-10d %-10s %-20s %-50s %-20s %-20s\n",
					pod.Metadata.Name, ready, status, restarts, age,
					podIP, node, nominated, "<none>")
			}
		} else {
			// 标准输出格式
			if allNamespaces {
				fmt.Printf("%-30s %-50s %-10s %-15s %-10d %-10s\n",
					pod.Metadata.Namespace, pod.Metadata.Name, ready, status, restarts, age)
			} else {
				fmt.Printf("%-50s %-10s %-15s %-10d %-10s\n",
					pod.Metadata.Name, ready, status, restarts, age)
			}
		}
	}

	return nil
}

// getDeployments 获取 Deployments 列表
func getDeployments(client *K8sHTTPClient, namespace string, allNamespaces bool) error {
	var deploymentsList DeploymentList
	var err error

	if allNamespaces {
		err = client.ListAllNamespaces("deployments", &deploymentsList)
	} else {
		if namespace == "" {
			namespace = "default"
		}
		apiPath := fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments", namespace)
		err = client.Get(apiPath, &deploymentsList)
	}

	if err != nil {
		return err
	}

	// 打印表头
	if allNamespaces {
		fmt.Printf("%-30s %-50s %-10s %-10s %-10s %-10s\n",
			"NAMESPACE", "NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE")
	} else {
		fmt.Printf("%-50s %-10s %-10s %-10s %-10s\n",
			"NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE")
	}

	// 打印数据
	for _, deploy := range deploymentsList.Items {
		replicas := int32(0)
		if deploy.Spec.Replicas != nil {
			replicas = *deploy.Spec.Replicas
		}
		ready := fmt.Sprintf("%d/%d", deploy.Status.AvailableReplicas, replicas)
		age := formatAge(deploy.Metadata.CreationTimestamp)

		if allNamespaces {
			fmt.Printf("%-30s %-50s %-10s %-10d %-10d %-10s\n",
				deploy.Metadata.Namespace, deploy.Metadata.Name, ready,
				replicas, deploy.Status.AvailableReplicas, age)
		} else {
			fmt.Printf("%-50s %-10s %-10d %-10d %-10s\n",
				deploy.Metadata.Name, ready, replicas,
				deploy.Status.AvailableReplicas, age)
		}
	}

	return nil
}

// getServices 获取 Services 列表
func getServices(client *K8sHTTPClient, namespace string, allNamespaces bool) error {
	var servicesList ServiceList
	var err error

	if allNamespaces {
		err = client.ListAllNamespaces("services", &servicesList)
	} else {
		if namespace == "" {
			namespace = "default"
		}
		apiPath := fmt.Sprintf("/api/v1/namespaces/%s/services", namespace)
		err = client.Get(apiPath, &servicesList)
	}

	if err != nil {
		return err
	}

	// 打印表头
	if allNamespaces {
		fmt.Printf("%-30s %-50s %-15s %-20s %-30s %-10s\n",
			"NAMESPACE", "NAME", "TYPE", "CLUSTER-IP", "PORTS", "AGE")
	} else {
		fmt.Printf("%-50s %-15s %-20s %-30s %-10s\n",
			"NAME", "TYPE", "CLUSTER-IP", "PORTS", "AGE")
	}

	// 打印数据
	for _, svc := range servicesList.Items {
		// 构建端口列表（带协议）
		ports := []string{}
		for _, port := range svc.Spec.Ports {
			protocol := port.Protocol
			if protocol == "" {
				protocol = "TCP"
			}
			ports = append(ports, fmt.Sprintf("%d/%s", port.Port, protocol))
		}
		portsStr := strings.Join(ports, ",")

		// 获取 Type
		svcType := svc.Spec.Type
		if svcType == "" {
			svcType = "ClusterIP"
		}

		// 获取 ClusterIP
		clusterIP := svc.Spec.ClusterIP
		if clusterIP == "" || clusterIP == "None" {
			clusterIP = "<none>"
		}

		// 计算 AGE
		age := formatAge(svc.Metadata.CreationTimestamp)

		if allNamespaces {
			fmt.Printf("%-30s %-50s %-15s %-20s %-30s %-10s\n",
				svc.Metadata.Namespace, svc.Metadata.Name, svcType, clusterIP, portsStr, age)
		} else {
			fmt.Printf("%-50s %-15s %-20s %-30s %-10s\n",
				svc.Metadata.Name, svcType, clusterIP, portsStr, age)
		}
	}

	return nil
}

// getIngresses 获取 Ingresses 列表
func getIngresses(client *K8sHTTPClient, namespace string, allNamespaces bool) error {
	var ingressesList IngressList
	var err error

	if allNamespaces {
		err = client.ListAllNamespaces("ingresses", &ingressesList)
	} else {
		if namespace == "" {
			namespace = "default"
		}
		apiPath := fmt.Sprintf("/apis/networking.k8s.io/v1/namespaces/%s/ingresses", namespace)
		err = client.Get(apiPath, &ingressesList)
	}

	if err != nil {
		return err
	}

	// 打印表头
	if allNamespaces {
		fmt.Printf("%-30s %-50s %-40s %-20s %-10s %-10s\n",
			"NAMESPACE", "NAME", "HOSTS", "ADDRESS", "PORTS", "AGE")
	} else {
		fmt.Printf("%-50s %-40s %-20s %-10s %-10s\n",
			"NAME", "HOSTS", "ADDRESS", "PORTS", "AGE")
	}

	// 打印数据
	for _, ing := range ingressesList.Items {
		hosts := []string{}
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		hostsStr := strings.Join(hosts, ",")
		if hostsStr == "" {
			hostsStr = "*"
		}

		age := formatAge(ing.Metadata.CreationTimestamp)

		if allNamespaces {
			fmt.Printf("%-30s %-50s %-40s %-20s %-10s %-10s\n",
				ing.Metadata.Namespace, ing.Metadata.Name, hostsStr, "-", "80", age)
		} else {
			fmt.Printf("%-50s %-40s %-20s %-10s %-10s\n",
				ing.Metadata.Name, hostsStr, "-", "80", age)
		}
	}

	return nil
}

// getNodes 获取 Nodes 列表
func getNodes(client *K8sHTTPClient) error {
	var nodesList struct {
		Items []struct {
			Metadata Metadata `json:"metadata"`
			Status   struct {
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
			} `json:"status"`
		} `json:"items"`
	}

	err := client.Get("/api/v1/nodes", &nodesList)
	if err != nil {
		return err
	}

	// 打印表头
	fmt.Printf("%-50s %-15s %-20s %-10s %-15s\n",
		"NAME", "STATUS", "ROLES", "AGE", "VERSION")

	// 打印数据
	for _, node := range nodesList.Items {
		status := "Unknown"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				status = "Ready"
				break
			}
		}

		role := "<none>"
		if node.Metadata.Labels != nil {
			if _, ok := node.Metadata.Labels["node-role.kubernetes.io/master"]; ok {
				role = "master"
			} else if _, ok := node.Metadata.Labels["node-role.kubernetes.io/control-plane"]; ok {
				role = "control-plane"
			}
		}

		fmt.Printf("%-50s %-15s %-20s %-10s %-15s\n",
			node.Metadata.Name, status, role, "Unknown", "-")
	}

	return nil
}

// getNamespaces 获取 Namespaces 列表
func getNamespaces(client *K8sHTTPClient) error {
	var nsList struct {
		Items []struct {
			Metadata Metadata `json:"metadata"`
			Status   struct {
				Phase string `json:"phase"`
			} `json:"status"`
		} `json:"items"`
	}

	err := client.Get("/api/v1/namespaces", &nsList)
	if err != nil {
		return err
	}

	// 打印表头
	fmt.Printf("%-50s %-15s %-10s\n", "NAME", "STATUS", "AGE")

	// 打印数据
	for _, ns := range nsList.Items {
		status := ns.Status.Phase
		if status == "" {
			status = "Active"
		}
		age := formatAge(ns.Metadata.CreationTimestamp)
		fmt.Printf("%-50s %-15s %-10s\n", ns.Metadata.Name, status, age)
	}

	return nil
}

// GetLogs 获取 Pod 日志
func GetLogs(kubeconfigPath, podName, namespace string, follow bool, tail int) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	// 构建日志 API 路径
	apiPath := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", namespace, podName)
	if tail > 0 {
		apiPath += fmt.Sprintf("?tailLines=%d", tail)
	}
	if follow {
		apiPath += "&follow=true"
	}

	// 获取日志（支持流式）
	url := client.baseURL + apiPath
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// 添加认证
	if client.token != "" {
		req.Header.Set("Authorization", "Bearer "+client.token)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("获取日志失败 HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 打印日志
	data := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(data)
		if n > 0 {
			fmt.Print(string(data[:n]))
		}
		if err != nil {
			break
		}
	}

	return nil
}

// DescribeResource 描述资源详情
func DescribeResource(kubeconfigPath, resourceType, name, namespace string) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	// 资源类型映射
	resourceMap := map[string]string{
		"po":         "pod",
		"pod":        "pod",
		"deploy":     "deployment",
		"deployment": "deployment",
		"svc":        "service",
		"service":    "service",
	}

	fullType := resourceMap[resourceType]
	if fullType == "" {
		return fmt.Errorf("不支持的资源类型: %s", resourceType)
	}

	switch fullType {
	case "pod":
		return describePod(client, name, namespace)
	case "deployment":
		return describeDeployment(client, name, namespace)
	case "service":
		return describeService(client, name, namespace)
	default:
		return fmt.Errorf("暂不支持的资源类型: %s", fullType)
	}
}

// describePod 描述 Pod
func describePod(client *K8sHTTPClient, name, namespace string) error {
	var pod Pod
	apiPath := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", namespace, name)
	err := client.Get(apiPath, &pod)
	if err != nil {
		return err
	}

	fmt.Printf("Name:         %s\n", pod.Metadata.Name)
	fmt.Printf("Namespace:    %s\n", pod.Metadata.Namespace)
	fmt.Printf("Labels:       ")
	if len(pod.Metadata.Labels) > 0 {
		for k, v := range pod.Metadata.Labels {
			fmt.Printf("%s=%s ", k, v)
		}
	}
	fmt.Println()
	fmt.Printf("\nContainers:\n")
	for _, container := range pod.Spec.Containers {
		fmt.Printf("  %s:\n", container.Name)
		if container.Resources.Requests != nil {
			fmt.Printf("    Requests:\n")
			for k, v := range container.Resources.Requests {
				fmt.Printf("      %s: %s\n", k, v)
			}
		}
		if container.Resources.Limits != nil {
			fmt.Printf("    Limits:\n")
			for k, v := range container.Resources.Limits {
				fmt.Printf("      %s: %s\n", k, v)
			}
		}
	}

	return nil
}

// describeDeployment 描述 Deployment
func describeDeployment(client *K8sHTTPClient, name, namespace string) error {
	var deploy Deployment
	apiPath := fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments/%s", namespace, name)
	err := client.Get(apiPath, &deploy)
	if err != nil {
		return err
	}

	replicas := int32(0)
	if deploy.Spec.Replicas != nil {
		replicas = *deploy.Spec.Replicas
	}

	fmt.Printf("Name:         %s\n", deploy.Metadata.Name)
	fmt.Printf("Namespace:    %s\n", deploy.Metadata.Namespace)
	fmt.Printf("Replicas:     %d desired | %d available\n", replicas, deploy.Status.AvailableReplicas)
	fmt.Printf("Labels:       ")
	if len(deploy.Metadata.Labels) > 0 {
		for k, v := range deploy.Metadata.Labels {
			fmt.Printf("%s=%s ", k, v)
		}
	}
	fmt.Println()

	return nil
}

// describeService 描述 Service
func describeService(client *K8sHTTPClient, name, namespace string) error {
	var svc Service
	apiPath := fmt.Sprintf("/api/v1/namespaces/%s/services/%s", namespace, name)
	err := client.Get(apiPath, &svc)
	if err != nil {
		return err
	}

	fmt.Printf("Name:         %s\n", svc.Metadata.Name)
	fmt.Printf("Namespace:    %s\n", svc.Metadata.Namespace)
	fmt.Printf("Selector:     ")
	if len(svc.Spec.Selector) > 0 {
		for k, v := range svc.Spec.Selector {
			fmt.Printf("%s=%s ", k, v)
		}
	}
	fmt.Println()
	fmt.Printf("Ports:\n")
	for _, port := range svc.Spec.Ports {
		fmt.Printf("  %s\t%d\n", port.Name, port.Port)
	}

	return nil
}

// formatAge 格式化时间差
func formatAge(timestamp string) string {
	if timestamp == "" {
		return "Unknown"
	}

	// 解析 ISO8601 格式的时间戳
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return "Unknown"
	}

	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	return fmt.Sprintf("%dd", int(duration.Hours()/24))
}

// DeleteResource 删除资源
func DeleteResource(kubeconfigPath, resourceType, name, namespace string) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	// 资源类型映射
	resourceMap := map[string]struct {
		apiPath string
		api     string
	}{
		"pod":        {"/api/v1/namespaces/%s/pods/%s", "v1"},
		"po":         {"/api/v1/namespaces/%s/pods/%s", "v1"},
		"deployment": {"/apis/apps/v1/namespaces/%s/deployments/%s", "apps/v1"},
		"deploy":     {"/apis/apps/v1/namespaces/%s/deployments/%s", "apps/v1"},
		"service":    {"/api/v1/namespaces/%s/services/%s", "v1"},
		"svc":        {"/api/v1/namespaces/%s/services/%s", "v1"},
		"ingress":    {"/apis/networking.k8s.io/v1/namespaces/%s/ingresses/%s", "networking.k8s.io/v1"},
		"ing":        {"/apis/networking.k8s.io/v1/namespaces/%s/ingresses/%s", "networking.k8s.io/v1"},
	}

	resource, ok := resourceMap[resourceType]
	if !ok {
		return fmt.Errorf("不支持删除的资源类型: %s", resourceType)
	}

	apiPath := fmt.Sprintf(resource.apiPath, namespace, name)
	url := client.baseURL + apiPath

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	if client.token != "" {
		req.Header.Set("Authorization", "Bearer "+client.token)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("删除失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("删除失败 %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("%s \"%s\" deleted\n", resourceType, name)
	return nil
}

// ScaleDeployment 扩缩容 Deployment
func ScaleDeployment(kubeconfigPath, name, namespace string, replicas int32) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	// PATCH 请求修改 replicas
	apiPath := fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments/%s", namespace, name)
	url := client.baseURL + apiPath

	patchData := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": replicas,
		},
	}

	jsonData, err := json.Marshal(patchData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/merge-patch+json")
	if client.token != "" {
		req.Header.Set("Authorization", "Bearer "+client.token)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("扩缩容失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("扩缩容失败 %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("deployment.apps/%s scaled\n", name)
	return nil
}

// RolloutStatus 查看发布状态
func RolloutStatus(kubeconfigPath, resourceType, name, namespace string) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	// 只支持 deployment
	if resourceType != "deployment" && resourceType != "deploy" {
		return fmt.Errorf("rollout status 只支持 deployment")
	}

	var deploy Deployment
	apiPath := fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments/%s", namespace, name)
	err = client.Get(apiPath, &deploy)
	if err != nil {
		return err
	}

	replicas := int32(0)
	if deploy.Spec.Replicas != nil {
		replicas = *deploy.Spec.Replicas
	}

	available := deploy.Status.AvailableReplicas

	if available == replicas && replicas > 0 {
		fmt.Printf("deployment \"%s\" successfully rolled out\n", name)
	} else {
		fmt.Printf("Waiting for deployment \"%s\" rollout to finish: %d out of %d new replicas are available...\n",
			name, available, replicas)
	}

	return nil
}

// ApplyYAML 应用 YAML 配置
func ApplyYAML(kubeconfigPath, filename string) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	// 读取 YAML 文件
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	// 解析 YAML（简化版，只处理单个资源）
	var resource map[string]interface{}
	if err := json.Unmarshal(data, &resource); err != nil {
		// 尝试 YAML 解析
		return fmt.Errorf("解析文件失败: %v (暂只支持 JSON 格式)", err)
	}

	// 获取资源类型和名称
	kind, _ := resource["kind"].(string)
	metadata, _ := resource["metadata"].(map[string]interface{})
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)

	if namespace == "" {
		namespace = "default"
	}

	// 构建 API 路径
	var apiPath string
	var method string

	switch kind {
	case "Deployment":
		apiPath = fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments/%s", namespace, name)
		method = "PUT"
	case "Service":
		apiPath = fmt.Sprintf("/api/v1/namespaces/%s/services/%s", namespace, name)
		method = "PUT"
	default:
		return fmt.Errorf("暂不支持的资源类型: %s", kind)
	}

	url := client.baseURL + apiPath

	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if client.token != "" {
		req.Header.Set("Authorization", "Bearer "+client.token)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("应用失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// 资源不存在，使用 POST 创建
		createPath := strings.TrimSuffix(apiPath, "/"+name)
		req, _ = http.NewRequest("POST", client.baseURL+createPath, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		if client.token != "" {
			req.Header.Set("Authorization", "Bearer "+client.token)
		}

		resp, err = client.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("创建失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("创建失败 %d: %s", resp.StatusCode, string(body))
		}
		fmt.Printf("%s/%s created\n", strings.ToLower(kind), name)
	} else if resp.StatusCode == http.StatusOK {
		fmt.Printf("%s/%s configured\n", strings.ToLower(kind), name)
	} else {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("应用失败 %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
