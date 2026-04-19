package cmd

import (
	"opsxcli/plugins/kubernetes"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("consul", "Kubernetes", "Consul服务发现", NewConsulCmd)
}

// NewConsulCmd 创建Consul命令
func NewConsulCmd() *cobra.Command {
	var (
		kubeconfig  string
		service     string
		metricsPath string
		clean       bool
		clearCache  bool
	)

	cmd := &cobra.Command{
		Use:   "consul [flags]",
		Short: "Consul服务注册和管理工具",
		Long: `Consul工具支持以下功能：
  - 从Kubernetes集群中自动发现服务并注册到Consul
  - 自动跳过本地健康检查（由Consul负责健康检查）
  - 批量注册服务
  - 智能清理失效实例

示例:
  # 从K8s集群注册服务到Consul
  opsxcli consul -s https://consul.example.com:8500 -m /actuator/prometheus

  # 指定kubeconfig路径
  opsxcli consul -k /path/to/kubeconfig -s https://consul.example.com:8500 -m /metrics

  # 清理失效实例
  opsxcli consul -s https://consul.example.com:8500 --clean

  # 清除缓存后重新扫描
  opsxcli consul -s https://consul.example.com:8500 -m /actuator/prometheus --clear-cache`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			if service == "" {
				return cmd.Help()
			}

			// 默认跳过本地检查
			monitor := kubernetes.NewKubernetesMonitor(kubeconfig, service, metricsPath, true)

			// 清除缓存
			if clearCache {
				monitor.ClearCache()
			}

			if clean {
				return monitor.CleanFailedInstances()
			}

			return monitor.UpdateServices()
		},
	}

	// 添加参数
	cmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "kubeconfig路径（默认：~/.kube/config）")
	cmd.Flags().StringVarP(&service, "service", "s", "", "Consul服务地址（如：https://consul.example.com:8500）")
	cmd.Flags().StringVarP(&metricsPath, "metrics", "m", "/actuator/prometheus", "指标URL路径")
	cmd.Flags().BoolVar(&clean, "clean", false, "清理失效实例")
	cmd.Flags().BoolVar(&clearCache, "clear-cache", false, "清除本地缓存")

	// 标记必需参数
	cmd.MarkFlagRequired("service")

	return cmd
}
