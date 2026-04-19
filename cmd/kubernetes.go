package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"opsxcli/plugins/kubernetes"
)

func init() {
	RegisterCommand("kubernetes", "Kubernetes", "Kubernetes集群管理", NewKubernetesCmd)
}

// NewKubernetesCmd 创建Kubernetes命令
func NewKubernetesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubernetes",
		Short: "Kubernetes 集群管理工具集",
		Long: `Kubernetes 工具集提供以下功能：
  - check: 集群健康检查，快速定位故障
  - resource: 统计集群资源配置，生成 Excel 报告
  - yaml: 导出 Deployment/Service/Ingress 为 YAML 文件
  - consul: Consul 服务注册和管理 (已作为独立命令)

使用 'opsxcli kubernetes <command> --help' 查看具体命令的帮助信息。`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// 添加子命令
	cmd.AddCommand(NewKubernetesCheckCmd())
	cmd.AddCommand(NewKubernetesResourceCmd())
	cmd.AddCommand(NewKubernetesYamlCmd())

	return cmd
}

// NewKubernetesCheckCmd 创建健康检查命令
func NewKubernetesCheckCmd() *cobra.Command {
	var kubeconfigPath string

	cmd := &cobra.Command{
		Use:   "check",
		Short: "集群健康检查",
		Long: `执行 Kubernetes 集群健康检查，快速定位常见故障：

检查项目包括：
  ✓ 节点状态（NotReady、资源压力）
  ✓ 系统组件健康（CoreDNS、metrics-server、ingress-controller）
  ✓ Pod 异常状态（CrashLoopBackOff、ImagePullBackOff、Pending）
  ✓ Ingress 配置问题（缺少地址、关联 Service 不存在）
  ✓ Service 选择器匹配问题
  ✓ Deployment 副本不足

示例:
  opsxcli kubernetes check
  opsxcli kubernetes check -k /path/to/kubeconfig`,
		Run: func(cmd *cobra.Command, args []string) {
			err := kubernetes.CheckCluster(kubeconfigPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVarP(&kubeconfigPath, "kubeconfig", "k", "", "kubeconfig 文件路径（默认：~/.kube/config）")

	return cmd
}
