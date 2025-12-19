package cmd

import (
	"opsxcli/plugins/kubernetes"

	"github.com/spf13/cobra"
)

// NewKubernetesResourceCmd 创建 Kubernetes 资源清单命令
func NewKubernetesResourceCmd() *cobra.Command {
	var kubeconfig string

	cmd := &cobra.Command{
		Use:   "resource",
		Short: "统计集群资源配置,生成Excel报告",
		Long: `统计 Kubernetes 集群中所有 Deployment 的资源配置信息:
  - CPU Request/Limit
  - Memory Request/Limit
  - 副本数 (期望/可用)
  - CapacityProvisioned 注解信息

生成详细的 Excel 报告 (deployments_resources.xlsx)`,
		Example: `  # 使用默认 kubeconfig
  opsxcli kubernetes resource

  # 指定 kubeconfig 路径
  opsxcli kubernetes resource -k /path/to/kubeconfig`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return kubernetes.ExportResourceInventory(kubeconfig)
		},
	}

	cmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "kubeconfig路径（默认：~/.kube/config）")

	return cmd
}
