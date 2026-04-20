package cmd

import (
	"fmt"
	"opsxcli/plugins/cloudhost"
	"opsxcli/plugins/kubernetes"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("consul", "Kubernetes", "Consul服务注册和管理工具", NewConsulCmd)
}

// NewConsulCmd 创建Consul命令
func NewConsulCmd() *cobra.Command {
	var (
		// 通用参数
		service     string
		metricsPath string
		clean       bool

		// K8s 模式参数
		kubeconfig  string
		clearCache  bool

		// 云主机模式参数
		hosts           []string
		hostsFile       string
		appPort         int
		nodeExpPort     int
		skipNodeExporter bool
	)

	cmd := &cobra.Command{
		Use:   "consul [flags]",
		Short: "Consul服务注册和管理工具",
		Long: `Consul工具支持以下功能：

Kubernetes 模式（默认）:
  - 从Kubernetes集群中自动发现服务并注册到Consul
  - 自动跳过本地健康检查（由Consul负责健康检查）

云主机模式（使用 --hosts 或 --hosts-file）:
  - 从主机列表注册应用和 node-exporter 到Consul
  - 支持 hostname:ip 或 hostname,ip 格式
  - 自动检查端点可用性

通用功能:
  - 智能清理失效实例（--clean）

示例:
  # K8s 模式 — 从集群注册服务到Consul
  opsxcli consul -s https://consul.example.com:8500 -m /actuator/prometheus

  # 云主机模式 — 指定主机列表
  opsxcli consul -s http://consul.example.com:8500 --hosts web1:192.168.1.10 web2:192.168.1.11

  # 云主机模式 — 从文件读取主机列表
  opsxcli consul -s http://consul.example.com:8500 --hosts-file /srv/hosts.txt --app-port 9999

  # 仅清理失效实例（两种模式通用）
  opsxcli consul -s https://consul.example.com:8500 --clean`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			if service == "" {
				return cmd.Help()
			}

			// 判断运行模式：
			// 1. 有 --hosts 或 --hosts-file → 云主机模式
			// 2. 有 --kubeconfig 或 -k → K8s 模式
			// 3. 仅 --clean 且无主机/K8s参数 → 通用清理模式（只需 Consul URL）
			// 4. 默认 → K8s 模式
			isCloudHostMode := len(hosts) > 0 || hostsFile != ""
			hasKubeconfig := cmd.Flags().Changed("kubeconfig")

			if isCloudHostMode {
				return runCloudHostMode(service, metricsPath, clean, hosts, hostsFile, appPort, nodeExpPort, skipNodeExporter)
			}

			if clean && !hasKubeconfig {
				// 仅清理，不需要 K8s 也不需要主机列表
				return runCleanMode(service)
			}

			// K8s 模式
			return runK8sMode(kubeconfig, service, metricsPath, clean, clearCache)
		},
	}

	// 通用参数
	cmd.Flags().StringVarP(&service, "service", "s", "", "Consul服务地址（如：https://consul.example.com:8500）")
	cmd.Flags().StringVarP(&metricsPath, "metrics", "m", "/actuator/prometheus", "应用指标URL路径")
	cmd.Flags().BoolVar(&clean, "clean", false, "仅清理失效实例")

	// K8s 模式参数
	cmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "kubeconfig路径（默认：~/.kube/config）")
	cmd.Flags().BoolVar(&clearCache, "clear-cache", false, "清除本地缓存（仅K8s模式）")

	// 云主机模式参数
	cmd.Flags().StringSliceVar(&hosts, "hosts", nil, "主机列表，格式 hostname:ip 或 hostname,ip（多个用逗号分隔或多次指定）")
	cmd.Flags().StringVar(&hostsFile, "hosts-file", "", "主机列表文件路径（每行 hostname:ip 或 hostname,ip）")
	cmd.Flags().IntVar(&appPort, "app-port", 8080, "应用指标端口（云主机模式，默认8080）")
	cmd.Flags().IntVar(&nodeExpPort, "node-exp-port", 9100, "node-exporter 端口（云主机模式，默认9100）")
	cmd.Flags().BoolVar(&skipNodeExporter, "skip-node-exporter", false, "跳过 node-exporter 注册")

	// 标记必需参数
	cmd.MarkFlagRequired("service")

	return cmd
}

// runK8sMode K8s 模式执行
func runK8sMode(kubeconfig, service, metricsPath string, clean, clearCache bool) error {
	monitor := kubernetes.NewKubernetesMonitor(kubeconfig, service, metricsPath, true)

	if clearCache {
		monitor.ClearCache()
	}

	if clean {
		return monitor.CleanFailedInstances()
	}

	return monitor.UpdateServices()
}

// runCleanMode 通用清理模式（只需 Consul URL，不依赖 K8s 或主机列表）
func runCleanMode(consulURL string) error {
	registry := cloudhost.NewCloudHostRegistry(consulURL, nil, 0, 0, "")
	return registry.CleanFailedInstances()
}

// runCloudHostMode 云主机模式执行
func runCloudHostMode(consulURL, metricsPath string, clean bool, hostArgs []string, hostsFile string, appPort, nodeExpPort int, skipNodeExporter bool) error {
	// 仅清理模式（不需要主机列表）
	if clean && len(hostArgs) == 0 && hostsFile == "" {
		registry := cloudhost.NewCloudHostRegistry(consulURL, nil, appPort, nodeExpPort, metricsPath)
		return registry.CleanFailedInstances()
	}

	// 解析主机列表
	var hostEntries []cloudhost.HostEntry

	if hostsFile != "" {
		loaded, err := cloudhost.LoadHostsFromFile(hostsFile)
		if err != nil {
			return fmt.Errorf("加载主机文件失败: %v", err)
		}
		hostEntries = loaded
	}

	if len(hostArgs) > 0 {
		hostEntries = append(hostEntries, cloudhost.ParseHosts(hostArgs)...)
	}

	if len(hostEntries) == 0 {
		return fmt.Errorf("未提供任何有效的主机，请使用 --hosts 或 --hosts-file 指定")
	}

	fmt.Printf("待处理主机数量: %d\n", len(hostEntries))
	for _, h := range hostEntries {
		fmt.Printf("  - %s → %s\n", h.Hostname, h.IP)
	}

	registry := cloudhost.NewCloudHostRegistry(consulURL, hostEntries, appPort, nodeExpPort, metricsPath)

	// 注册应用服务
	registry.RegisterAppServices()

	// 注册 node-exporter
	if !skipNodeExporter {
		registry.RegisterNodeExporters()
	}

	// 清理失效实例
	return registry.CleanFailedInstances()
}
