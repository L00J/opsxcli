package kubernetes

import (
	"fmt"
	"strings"
	"time"
)

// CheckResult 检查结果
type CheckResult struct {
	Category   string // 检查类别
	Level      string // 严重程度: Critical, Warning, Info
	Resource   string // 资源名称
	Namespace  string // 命名空间
	Issue      string // 问题描述
	Suggestion string // 修复建议
}

// CheckCluster 执行集群健康检查
func CheckCluster(kubeconfigPath string) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	fmt.Println("🔍 开始 Kubernetes 集群健康检查...")
	fmt.Println()

	var results []CheckResult

	// 1. 检查节点状态
	nodeResults := checkNodes(client)
	results = append(results, nodeResults...)

	// 2. 检查系统组件
	systemResults := checkSystemComponents(client)
	results = append(results, systemResults...)

	// 3. 检查 Pod 异常状态
	podResults := checkPods(client)
	results = append(results, podResults...)

	// 4. 检查 Ingress 问题
	ingressResults := checkIngresses(client)
	results = append(results, ingressResults...)

	// 5. 检查 Service 配置
	serviceResults := checkServices(client)
	results = append(results, serviceResults...)

	// 6. 检查 Deployment 状态
	deploymentResults := checkDeployments(client)
	results = append(results, deploymentResults...)

	// 输出结果
	printResults(results)

	return nil
}

// checkNodes 检查节点状态
func checkNodes(client *K8sHTTPClient) []CheckResult {
	var results []CheckResult

	var nodesList struct {
		Items []struct {
			Metadata Metadata `json:"metadata"`
			Status   struct {
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
					Reason string `json:"reason"`
				} `json:"conditions"`
				Allocatable map[string]string `json:"allocatable"`
			} `json:"status"`
		} `json:"items"`
	}

	err := client.Get("/api/v1/nodes", &nodesList)
	if err != nil {
		results = append(results, CheckResult{
			Category:   "节点",
			Level:      "Critical",
			Resource:   "Nodes",
			Issue:      "无法获取节点信息",
			Suggestion: "检查 API Server 连接和权限",
		})
		return results
	}

	for _, node := range nodesList.Items {
		// 检查节点状态
		ready := false
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" {
				if cond.Status == "True" {
					ready = true
				} else {
					results = append(results, CheckResult{
						Category:   "节点",
						Level:      "Critical",
						Resource:   node.Metadata.Name,
						Issue:      fmt.Sprintf("节点状态异常: %s", cond.Reason),
						Suggestion: "检查节点网络、kubelet 服务和系统资源",
					})
				}
			}

			// 检查节点压力
			if cond.Type == "MemoryPressure" && cond.Status == "True" {
				results = append(results, CheckResult{
					Category:   "节点",
					Level:      "Warning",
					Resource:   node.Metadata.Name,
					Issue:      "节点内存压力",
					Suggestion: "清理节点内存或添加更多节点",
				})
			}

			if cond.Type == "DiskPressure" && cond.Status == "True" {
				results = append(results, CheckResult{
					Category:   "节点",
					Level:      "Warning",
					Resource:   node.Metadata.Name,
					Issue:      "节点磁盘压力",
					Suggestion: "清理磁盘空间或扩容磁盘",
				})
			}
		}

		if ready {
			// 节点正常，记录为 Info
			// results = append(results, CheckResult{
			// 	Category: "节点",
			// 	Level:    "Info",
			// 	Resource: node.Metadata.Name,
			// 	Issue:    "节点运行正常",
			// })
		}
	}

	return results
}

// checkSystemComponents 检查系统组件
func checkSystemComponents(client *K8sHTTPClient) []CheckResult {
	var results []CheckResult

	// 检查关键系统组件
	criticalComponents := map[string]string{
		"coredns":           "kube-system",
		"metrics-server":    "kube-system",
		"aws-load-balancer-controller": "kube-system",
	}

	for component, namespace := range criticalComponents {
		var deploymentsList DeploymentList
		apiPath := fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments", namespace)
		err := client.Get(apiPath, &deploymentsList)
		if err != nil {
			continue
		}

		found := false
		for _, deploy := range deploymentsList.Items {
			if strings.Contains(deploy.Metadata.Name, component) {
				found = true
				replicas := int32(0)
				if deploy.Spec.Replicas != nil {
					replicas = *deploy.Spec.Replicas
				}

				if deploy.Status.AvailableReplicas < replicas {
					results = append(results, CheckResult{
						Category:   "系统组件",
						Level:      "Critical",
						Resource:   deploy.Metadata.Name,
						Namespace:  namespace,
						Issue:      fmt.Sprintf("系统组件副本不足: %d/%d", deploy.Status.AvailableReplicas, replicas),
						Suggestion: fmt.Sprintf("检查 %s Pod 日志和事件", component),
					})
				}
			}
		}

		if !found && component != "aws-load-balancer-controller" {
			results = append(results, CheckResult{
				Category:   "系统组件",
				Level:      "Warning",
				Resource:   component,
				Namespace:  namespace,
				Issue:      "系统组件未找到",
				Suggestion: fmt.Sprintf("安装 %s 组件", component),
			})
		}
	}

	return results
}

// checkPods 检查 Pod 异常状态
func checkPods(client *K8sHTTPClient) []CheckResult {
	var results []CheckResult

	var podsList PodList
	err := client.ListAllNamespaces("pods", &podsList)
	if err != nil {
		return results
	}

	for _, pod := range podsList.Items {
		// 跳过系统命名空间的已完成 Job
		if pod.Status.Phase == "Succeeded" {
			continue
		}

		// 检查 Pod 异常状态
		if pod.Status.Phase == "Failed" {
			results = append(results, CheckResult{
				Category:   "Pod 状态",
				Level:      "Warning",
				Resource:   pod.Metadata.Name,
				Namespace:  pod.Metadata.Namespace,
				Issue:      "Pod 运行失败",
				Suggestion: fmt.Sprintf("kubectl logs %s -n %s 查看日志", pod.Metadata.Name, pod.Metadata.Namespace),
			})
		}

		if pod.Status.Phase == "Pending" {
			// 检查是否超过 5 分钟
			createTime, _ := time.Parse(time.RFC3339, pod.Metadata.CreationTimestamp)
			if time.Since(createTime) > 5*time.Minute {
				results = append(results, CheckResult{
					Category:   "Pod 状态",
					Level:      "Warning",
					Resource:   pod.Metadata.Name,
					Namespace:  pod.Metadata.Namespace,
					Issue:      "Pod 长时间处于 Pending 状态",
					Suggestion: "检查资源配额、节点资源、镜像拉取和调度限制",
				})
			}
		}

		// 检查容器重启次数
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.RestartCount > 10 {
				results = append(results, CheckResult{
					Category:   "Pod 状态",
					Level:      "Critical",
					Resource:   fmt.Sprintf("%s/%s", pod.Metadata.Name, cs.Name),
					Namespace:  pod.Metadata.Namespace,
					Issue:      fmt.Sprintf("容器重启次数过高: %d 次", cs.RestartCount),
					Suggestion: "检查应用日志、健康检查配置和资源限制",
				})
			}

			// 检查容器状态
			if !cs.Ready && cs.RestartCount > 0 {
				results = append(results, CheckResult{
					Category:   "Pod 状态",
					Level:      "Warning",
					Resource:   fmt.Sprintf("%s/%s", pod.Metadata.Name, cs.Name),
					Namespace:  pod.Metadata.Namespace,
					Issue:      "容器未就绪且有重启记录",
					Suggestion: "检查容器健康检查配置和应用启动逻辑",
				})
			}
		}
	}

	return results
}

// checkIngresses 检查 Ingress 问题
func checkIngresses(client *K8sHTTPClient) []CheckResult {
	var results []CheckResult

	var ingressesList IngressList
	err := client.ListAllNamespaces("ingresses", &ingressesList)
	if err != nil {
		return results
	}

	for _, ing := range ingressesList.Items {
		// 检查 Ingress 是否有地址
		// 注意：Ingress 的 status.loadBalancer.ingress 字段需要在结构体中定义
		// 这里简化检查，通过创建时间判断
		createTime, _ := time.Parse(time.RFC3339, ing.Metadata.CreationTimestamp)
		if time.Since(createTime) > 10*time.Minute {
			// 假设 10 分钟后仍无地址则报警（需要实际检查 status）
			// 这里简化处理
		}

		// 检查 Ingress 关联的 Service
		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				serviceName := path.Backend.Service.Name
				
				// 检查 Service 是否存在
				var svc Service
				apiPath := fmt.Sprintf("/api/v1/namespaces/%s/services/%s", ing.Metadata.Namespace, serviceName)
				err := client.Get(apiPath, &svc)
				if err != nil {
					results = append(results, CheckResult{
						Category:   "Ingress 配置",
						Level:      "Critical",
						Resource:   ing.Metadata.Name,
						Namespace:  ing.Metadata.Namespace,
						Issue:      fmt.Sprintf("Ingress 引用的 Service 不存在: %s", serviceName),
						Suggestion: fmt.Sprintf("创建 Service %s 或修改 Ingress 配置", serviceName),
					})
				}
			}
		}

		// 检查 IngressClass
		if ing.Spec.IngressClassName == "" {
			results = append(results, CheckResult{
				Category:   "Ingress 配置",
				Level:      "Warning",
				Resource:   ing.Metadata.Name,
				Namespace:  ing.Metadata.Namespace,
				Issue:      "Ingress 未指定 IngressClassName",
				Suggestion: "添加 spec.ingressClassName 字段（如: nginx）",
			})
		}
	}

	// 检查 Ingress Controller
	var podsList PodList
	client.Get("/api/v1/namespaces/ingress-nginx/pods", &podsList)
	
	controllerFound := false
	for _, pod := range podsList.Items {
		if strings.Contains(pod.Metadata.Name, "nginx-ingress") || strings.Contains(pod.Metadata.Name, "ingress-controller") {
			controllerFound = true
			if pod.Status.Phase != "Running" {
				results = append(results, CheckResult{
					Category:   "Ingress Controller",
					Level:      "Critical",
					Resource:   pod.Metadata.Name,
					Namespace:  "ingress-nginx",
					Issue:      "Ingress Controller 未运行",
					Suggestion: "检查 Ingress Controller 的 Deployment 和日志",
				})
			}
		}
	}

	if !controllerFound && len(ingressesList.Items) > 0 {
		results = append(results, CheckResult{
			Category:   "Ingress Controller",
			Level:      "Critical",
			Resource:   "ingress-controller",
			Issue:      "未找到 Ingress Controller，但存在 Ingress 资源",
			Suggestion: "安装 Ingress Controller (如 nginx-ingress)",
		})
	}

	return results
}

// checkServices 检查 Service 配置
func checkServices(client *K8sHTTPClient) []CheckResult {
	var results []CheckResult

	var servicesList ServiceList
	err := client.ListAllNamespaces("services", &servicesList)
	if err != nil {
		return results
	}

	for _, svc := range servicesList.Items {
		// 跳过 kubernetes 默认服务
		if svc.Metadata.Name == "kubernetes" && svc.Metadata.Namespace == "default" {
			continue
		}

		// 检查 Service 是否有选择器
		if len(svc.Spec.Selector) == 0 {
			// Headless service 或 ExternalName 类型可以没有 selector
			if svc.Spec.Type != "ExternalName" && svc.Spec.ClusterIP != "None" {
				results = append(results, CheckResult{
					Category:   "Service 配置",
					Level:      "Warning",
					Resource:   svc.Metadata.Name,
					Namespace:  svc.Metadata.Namespace,
					Issue:      "Service 没有选择器",
					Suggestion: "添加 selector 或确认这是 ExternalName 类型服务",
				})
			}
			continue
		}

		// 检查是否有 Pod 匹配选择器
		var podsList PodList
		apiPath := fmt.Sprintf("/api/v1/namespaces/%s/pods", svc.Metadata.Namespace)
		err := client.Get(apiPath, &podsList)
		if err != nil {
			continue
		}

		matched := false
		for _, pod := range podsList.Items {
			if matchesSelector(pod.Metadata.Labels, svc.Spec.Selector) {
				matched = true
				break
			}
		}

		if !matched {
			results = append(results, CheckResult{
				Category:   "Service 配置",
				Level:      "Critical",
				Resource:   svc.Metadata.Name,
				Namespace:  svc.Metadata.Namespace,
				Issue:      "Service 选择器未匹配到任何 Pod",
				Suggestion: "检查 Service selector 和 Pod labels 是否一致",
			})
		}
	}

	return results
}

// checkDeployments 检查 Deployment 状态
func checkDeployments(client *K8sHTTPClient) []CheckResult {
	var results []CheckResult

	var deploymentsList DeploymentList
	err := client.ListAllNamespaces("deployments", &deploymentsList)
	if err != nil {
		return results
	}

	for _, deploy := range deploymentsList.Items {
		replicas := int32(0)
		if deploy.Spec.Replicas != nil {
			replicas = *deploy.Spec.Replicas
		}

		// 检查副本数是否正常
		if replicas > 0 && deploy.Status.AvailableReplicas == 0 {
			results = append(results, CheckResult{
				Category:   "Deployment 状态",
				Level:      "Critical",
				Resource:   deploy.Metadata.Name,
				Namespace:  deploy.Metadata.Namespace,
				Issue:      "Deployment 无可用副本",
				Suggestion: "检查 Pod 日志、镜像拉取和资源配额",
			})
		} else if deploy.Status.AvailableReplicas < replicas {
			results = append(results, CheckResult{
				Category:   "Deployment 状态",
				Level:      "Warning",
				Resource:   deploy.Metadata.Name,
				Namespace:  deploy.Metadata.Namespace,
				Issue:      fmt.Sprintf("Deployment 副本不足: %d/%d", deploy.Status.AvailableReplicas, replicas),
				Suggestion: "检查 Pod 状态和事件",
			})
		}
	}

	return results
}

// matchesSelector 检查标签是否匹配选择器
func matchesSelector(labels, selector map[string]string) bool {
	if len(selector) == 0 {
		return false
	}

	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}

	return true
}

// printResults 打印检查结果
func printResults(results []CheckResult) {
	// 统计
	critical := 0
	warning := 0
	info := 0

	for _, r := range results {
		switch r.Level {
		case "Critical":
			critical++
		case "Warning":
			warning++
		case "Info":
			info++
		}
	}

	// 打印摘要
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║               集群健康检查报告                                 ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	fmt.Printf("📊 检查结果汇总: ")
	if critical > 0 {
		fmt.Printf("🔴 严重 %d  ", critical)
	}
	if warning > 0 {
		fmt.Printf("🟡 警告 %d  ", warning)
	}
	if critical == 0 && warning == 0 {
		fmt.Printf("✅ 集群健康")
	}
	fmt.Println()
	fmt.Println()

	// 按严重程度排序输出
	if critical > 0 {
		fmt.Println("🔴 严重问题 (需要立即处理)")
		fmt.Println("─────────────────────────────────────────────────────────────")
		for _, r := range results {
			if r.Level == "Critical" {
				printResult(r)
			}
		}
		fmt.Println()
	}

	if warning > 0 {
		fmt.Println("🟡 警告 (建议关注)")
		fmt.Println("─────────────────────────────────────────────────────────────")
		for _, r := range results {
			if r.Level == "Warning" {
				printResult(r)
			}
		}
		fmt.Println()
	}

	if critical == 0 && warning == 0 {
		fmt.Println("✅ 未发现问题，集群运行正常")
		fmt.Println()
	}

	// 打印建议
	if critical > 0 || warning > 0 {
		fmt.Println("💡 快速诊断命令")
		fmt.Println("─────────────────────────────────────────────────────────────")
		fmt.Println("  查看 Pod 详情:    opsxcli kubectl get po -A")
		fmt.Println("  查看 Pod 日志:    opsxcli kubectl logs <pod-name> -n <namespace>")
		fmt.Println("  查看事件:         kubectl get events -A --sort-by='.lastTimestamp'")
		fmt.Println("  查看节点资源:     kubectl top nodes")
		fmt.Println("  查看 Pod 资源:    kubectl top pods -A")
		fmt.Println()
	}
}

// printResult 打印单个检查结果
func printResult(r CheckResult) {
	icon := "•"
	switch r.Level {
	case "Critical":
		icon = "❌"
	case "Warning":
		icon = "⚠️"
	case "Info":
		icon = "ℹ️"
	}

	fmt.Printf("  %s [%s] %s", icon, r.Category, r.Resource)
	if r.Namespace != "" {
		fmt.Printf(" (命名空间: %s)", r.Namespace)
	}
	fmt.Println()
	fmt.Printf("     问题: %s\n", r.Issue)
	fmt.Printf("     建议: %s\n", r.Suggestion)
	fmt.Println()
}
