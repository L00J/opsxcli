package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// ExportYAMLResources 导出 Deployment/Service/Ingress 为 YAML 文件
func ExportYAMLResources(kubeconfigPath string) error {
	// 创建 HTTP 客户端
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	fmt.Println("Kubernetes API 连接成功")

	// 获取所有资源
	var deploymentsList DeploymentList
	if err := client.ListAllNamespaces("deployments", &deploymentsList); err != nil {
		return fmt.Errorf("获取Deployment列表失败: %v", err)
	}

	var servicesList ServiceList
	if err := client.ListAllNamespaces("services", &servicesList); err != nil {
		return fmt.Errorf("获取Service列表失败: %v", err)
	}

	var ingressesList IngressList
	if err := client.ListAllNamespaces("ingresses", &ingressesList); err != nil {
		return fmt.Errorf("获取Ingress列表失败: %v", err)
	}

	fmt.Printf("找到 %d 个 Deployment, %d 个 Service, %d 个 Ingress\n",
		len(deploymentsList.Items), len(servicesList.Items), len(ingressesList.Items))

	// 创建输出目录
	deploymentsDir := filepath.Join(".", "deployments")
	ingressDir := filepath.Join(".", "ingress")
	if err := os.MkdirAll(deploymentsDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}
	if err := os.MkdirAll(ingressDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 按命名空间组织资源
	namespaceMap := make(map[string]struct {
		deployments []Deployment
		services    []Service
		ingresses   []Ingress
	})

	for _, dep := range deploymentsList.Items {
		ns := namespaceMap[dep.Metadata.Namespace]
		ns.deployments = append(ns.deployments, dep)
		namespaceMap[dep.Metadata.Namespace] = ns
	}

	for _, svc := range servicesList.Items {
		ns := namespaceMap[svc.Metadata.Namespace]
		ns.services = append(ns.services, svc)
		namespaceMap[svc.Metadata.Namespace] = ns
	}

	for _, ing := range ingressesList.Items {
		ns := namespaceMap[ing.Metadata.Namespace]
		ns.ingresses = append(ns.ingresses, ing)
		namespaceMap[ing.Metadata.Namespace] = ns
	}

	exportedCount := 0

	// 导出 Deployment + Service
	for namespace, resources := range namespaceMap {
		nsDir := filepath.Join(deploymentsDir, namespace)
		if err := os.MkdirAll(nsDir, 0755); err != nil {
			continue
		}

		exportedServices := make(map[string]bool)

		// 导出每个 Deployment 及其关联的 Service
		for _, dep := range resources.deployments {
			// 设置 apiVersion 和 kind
			dep.APIVersion = "apps/v1"
			dep.Kind = "Deployment"

			// 清理运行时字段
			cleanDeployment(&dep)

			var yamlResources []interface{}
			yamlResources = append(yamlResources, dep)

			// 查找关联的 Service
			podLabels := dep.Spec.Template.Metadata.Labels
			for _, svc := range resources.services {
				if exportedServices[svc.Metadata.Name] {
					continue
				}

				// 通过 selector 匹配
				if matchesSelector(podLabels, svc.Spec.Selector) {
					svc.APIVersion = "v1"
					svc.Kind = "Service"
					cleanService(&svc)
					yamlResources = append(yamlResources, svc)
					exportedServices[svc.Metadata.Name] = true
				} else if svc.Metadata.Name == dep.Metadata.Name {
					// 名称完全匹配
					svc.APIVersion = "v1"
					svc.Kind = "Service"
					cleanService(&svc)
					yamlResources = append(yamlResources, svc)
					exportedServices[svc.Metadata.Name] = true
				}
			}

			// 写入文件
			filename := filepath.Join(nsDir, fmt.Sprintf("%s.yaml", dep.Metadata.Name))
			if err := writeYAML(filename, yamlResources); err != nil {
				fmt.Printf("⊗ 导出 %s 失败: %v\n", filename, err)
			} else {
				fmt.Printf("✓ 导出 %d 个资源到 %s\n", len(yamlResources), filename)
				exportedCount += len(yamlResources)
			}
		}

		// 导出独立的 Service
		for _, svc := range resources.services {
			if !exportedServices[svc.Metadata.Name] {
				svc.APIVersion = "v1"
				svc.Kind = "Service"
				cleanService(&svc)
				filename := filepath.Join(nsDir, fmt.Sprintf("%s.yaml", svc.Metadata.Name))
				if err := writeYAML(filename, []interface{}{svc}); err != nil {
					fmt.Printf("⊗ 导出 %s 失败: %v\n", filename, err)
				} else {
					fmt.Printf("✓ 导出独立Service到 %s\n", filename)
					exportedCount++
				}
			}
		}
	}

	// 导出 Ingress
	for namespace, resources := range namespaceMap {
		if len(resources.ingresses) == 0 {
			continue
		}

		nsDir := filepath.Join(ingressDir, namespace)
		if err := os.MkdirAll(nsDir, 0755); err != nil {
			continue
		}

		for _, ing := range resources.ingresses {
			ing.APIVersion = "networking.k8s.io/v1"
			ing.Kind = "Ingress"
			cleanIngress(&ing)
			filename := filepath.Join(nsDir, fmt.Sprintf("%s.yaml", ing.Metadata.Name))
			if err := writeYAML(filename, []interface{}{ing}); err != nil {
				fmt.Printf("⊗ 导出 %s 失败: %v\n", filename, err)
			} else {
				fmt.Printf("✓ 导出Ingress到 %s\n", filename)
				exportedCount++
			}
		}
	}

	fmt.Printf("\n✓ 导出完成! 共导出 %d 个资源\n", exportedCount)
	fmt.Printf("  - Deployment+Service: %s/\n", deploymentsDir)
	fmt.Printf("  - Ingress: %s/\n", ingressDir)

	return nil
}

// writeYAML 写入 YAML 文件
func writeYAML(filename string, resources []interface{}) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	for i, resource := range resources {
		if i > 0 {
			f.WriteString("---\n")
		}

		data, err := yaml.Marshal(resource)
		if err != nil {
			return err
		}

		if _, err := f.Write(data); err != nil {
			return err
		}
	}

	return nil
}

// cleanDeployment 清理 Deployment 的运行时字段
func cleanDeployment(dep *Deployment) {
	// 清理 metadata
	dep.Metadata.CreationTimestamp = ""
	dep.Metadata.Annotations = cleanAnnotations(dep.Metadata.Annotations)

	// 清理 template.metadata - 移除空字段
	dep.Spec.Template.Metadata.Name = ""
	dep.Spec.Template.Metadata.Namespace = ""
	dep.Spec.Template.Metadata.CreationTimestamp = ""
	// 清理 template annotations 中的运行时字段
	dep.Spec.Template.Metadata.Annotations = cleanAnnotations(dep.Spec.Template.Metadata.Annotations)

	// 清理 spec.template.spec.nodeName
	dep.Spec.Template.Spec.NodeName = ""

	// 移除 status
	dep.Status = nil
}

// cleanService 清理 Service 的运行时字段
func cleanService(svc *Service) {
	// 清理 metadata
	svc.Metadata.CreationTimestamp = ""
	svc.Metadata.Annotations = cleanAnnotations(svc.Metadata.Annotations)

	// 清理 clusterIP (动态分配的)
	svc.Spec.ClusterIP = ""
}

// cleanIngress 清理 Ingress 的运行时字段
func cleanIngress(ing *Ingress) {
	// 清理 metadata
	ing.Metadata.CreationTimestamp = ""
	ing.Metadata.Annotations = cleanAnnotations(ing.Metadata.Annotations)
}

// cleanAnnotations 清理不需要的 annotations
func cleanAnnotations(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}

	cleaned := make(map[string]string)
	for k, v := range annotations {
		// 跳过运行时 annotations
		if k == "kubectl.kubernetes.io/last-applied-configuration" ||
			k == "deployment.kubernetes.io/revision" {
			continue
		}
		cleaned[k] = v
	}

	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}
