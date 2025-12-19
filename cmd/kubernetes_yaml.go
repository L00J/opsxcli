package cmd

import (
	"opsxcli/plugins/kubernetes"

	"github.com/spf13/cobra"
)

// NewKubernetesYamlCmd 创建 Kubernetes YAML 导出命令
func NewKubernetesYamlCmd() *cobra.Command {
	var kubeconfig string

	cmd := &cobra.Command{
		Use:   "yaml",
		Short: "导出 Deployment/Service/Ingress 为 YAML 文件",
		Long: `导出 Kubernetes 集群中的资源为 YAML 文件:
  - Deployment + 关联的 Service
  - 独立的 Service
  - Ingress 资源

自动清理不必要的字段，按命名空间组织文件结构`,
		Example: `  # 使用默认 kubeconfig
  opsxcli kubernetes yaml

  # 指定 kubeconfig 路径
  opsxcli kubernetes yaml -k /path/to/kubeconfig`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return kubernetes.ExportYAMLResources(kubeconfig)
		},
	}

	cmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "kubeconfig路径（默认：~/.kube/config）")

	return cmd
}
