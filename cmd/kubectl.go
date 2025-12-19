package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"opsxcli/plugins/kubernetes"
)

var (
	kubectlKubeconfig   string
	kubectlNamespace    string
	kubectlAllNs        bool
	kubectlFollow       bool
	kubectlTail         int
	kubectlOutputFormat string
	kubectlReplicas     int
)

// NewKubectlCmd 创建 kubectl 命令
func NewKubectlCmd() *cobra.Command {
	kubectlCmd := &cobra.Command{
		Use:   "kubectl",
		Short: "Kubernetes 命令行工具",
		Long:  `轻量级 kubectl 工具，无需安装 kubectl 即可管理 Kubernetes 资源`,
	}

	// kubectl get 命令
	kubectlGetCmd := &cobra.Command{
		Use:   "get [资源类型]",
		Short: "获取资源列表",
		Long: `获取 Kubernetes 资源列表

支持的资源类型:
  pods, po           - Pods
  deployments, deploy - Deployments
  services, svc      - Services
  ingresses, ing     - Ingresses
  nodes              - Nodes
  namespaces, ns     - Namespaces

示例:
  opsxcli kubectl get pods
  opsxcli kubectl get po -n kube-system
  opsxcli kubectl get deploy --all-namespaces
  opsxcli kubectl get svc -A`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			resourceType := args[0]
			err := kubernetes.GetResources(kubectlKubeconfig, resourceType, kubectlNamespace, kubectlAllNs, kubectlOutputFormat)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
		},
	}

	kubectlGetCmd.Flags().StringVarP(&kubectlKubeconfig, "kubeconfig", "k", "", "kubeconfig 文件路径（默认：~/.kube/config）")
	kubectlGetCmd.Flags().StringVarP(&kubectlNamespace, "namespace", "n", "", "命名空间")
	kubectlGetCmd.Flags().BoolVarP(&kubectlAllNs, "all-namespaces", "A", false, "所有命名空间")
	kubectlGetCmd.Flags().StringVarP(&kubectlOutputFormat, "output", "o", "", "输出格式 (wide)")

	// kubectl describe 命令
	kubectlDescribeCmd := &cobra.Command{
		Use:   "describe [资源类型] [名称]",
		Short: "查看资源详情",
		Long: `查看 Kubernetes 资源的详细信息

支持的资源类型:
  pod, po           - Pod
  deployment, deploy - Deployment
  service, svc      - Service

示例:
  opsxcli kubectl describe pod my-pod
  opsxcli kubectl describe deploy my-deployment -n default
  opsxcli kubectl describe svc my-service`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			resourceType := args[0]
			name := args[1]
			err := kubernetes.DescribeResource(kubectlKubeconfig, resourceType, name, kubectlNamespace)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
		},
	}

	kubectlDescribeCmd.Flags().StringVarP(&kubectlKubeconfig, "kubeconfig", "k", "", "kubeconfig 文件路径（默认：~/.kube/config）")
	kubectlDescribeCmd.Flags().StringVarP(&kubectlNamespace, "namespace", "n", "", "命名空间（默认：default）")

	// kubectl logs 命令
	kubectlLogsCmd := &cobra.Command{
		Use:   "logs [pod名称]",
		Short: "查看 Pod 日志",
		Long: `查看 Kubernetes Pod 的日志

示例:
  opsxcli kubectl logs my-pod
  opsxcli kubectl logs my-pod -n kube-system
  opsxcli kubectl logs my-pod --tail 100
  opsxcli kubectl logs my-pod -f`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			podName := args[0]
			err := kubernetes.GetLogs(kubectlKubeconfig, podName, kubectlNamespace, kubectlFollow, kubectlTail)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
		},
	}

	kubectlLogsCmd.Flags().StringVarP(&kubectlKubeconfig, "kubeconfig", "k", "", "kubeconfig 文件路径（默认：~/.kube/config）")
	kubectlLogsCmd.Flags().StringVarP(&kubectlNamespace, "namespace", "n", "", "命名空间（默认：default）")
	kubectlLogsCmd.Flags().BoolVarP(&kubectlFollow, "follow", "f", false, "持续输出日志")
	kubectlLogsCmd.Flags().IntVar(&kubectlTail, "tail", 0, "显示最后 N 行日志")

	// kubectl delete 命令
	kubectlDeleteCmd := &cobra.Command{
		Use:   "delete [资源类型] [名称]",
		Short: "删除资源",
		Long: `删除 Kubernetes 资源

支持的资源类型:
  pod, po           - Pod
  deployment, deploy - Deployment
  service, svc      - Service
  ingress, ing      - Ingress

示例:
  opsxcli kubectl delete pod my-pod -n default
  opsxcli kubectl delete deploy my-deployment
  opsxcli kubectl delete svc my-service`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			resourceType := args[0]
			name := args[1]
			err := kubernetes.DeleteResource(kubectlKubeconfig, resourceType, name, kubectlNamespace)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
		},
	}

	kubectlDeleteCmd.Flags().StringVarP(&kubectlKubeconfig, "kubeconfig", "k", "", "kubeconfig 文件路径（默认：~/.kube/config）")
	kubectlDeleteCmd.Flags().StringVarP(&kubectlNamespace, "namespace", "n", "", "命名空间（默认：default）")

	// kubectl scale 命令
	kubectlScaleCmd := &cobra.Command{
		Use:   "scale [资源类型] [名称] --replicas=N",
		Short: "扩缩容资源",
		Long: `设置 Deployment 的副本数

示例:
  opsxcli kubectl scale deployment my-deployment --replicas=3
  opsxcli kubectl scale deploy my-deployment --replicas=5 -n default`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			resourceType := args[0]
			name := args[1]

			if resourceType != "deployment" && resourceType != "deploy" {
				fmt.Fprintf(os.Stderr, "错误: scale 只支持 deployment\n")
				os.Exit(1)
			}

			if kubectlReplicas < 0 {
				fmt.Fprintf(os.Stderr, "错误: --replicas 必须指定\n")
				os.Exit(1)
			}

			err := kubernetes.ScaleDeployment(kubectlKubeconfig, name, kubectlNamespace, int32(kubectlReplicas))
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
		},
	}

	kubectlScaleCmd.Flags().StringVarP(&kubectlKubeconfig, "kubeconfig", "k", "", "kubeconfig 文件路径（默认：~/.kube/config）")
	kubectlScaleCmd.Flags().StringVarP(&kubectlNamespace, "namespace", "n", "", "命名空间（默认：default）")
	kubectlScaleCmd.Flags().IntVar(&kubectlReplicas, "replicas", -1, "副本数")
	kubectlScaleCmd.MarkFlagRequired("replicas")

	// kubectl rollout 命令
	kubectlRolloutCmd := &cobra.Command{
		Use:   "rollout",
		Short: "管理发布",
		Long:  `管理 Deployment 的发布`,
	}

	kubectlRolloutStatusCmd := &cobra.Command{
		Use:   "status [资源类型] [名称]",
		Short: "查看发布状态",
		Long: `查看 Deployment 的发布状态

示例:
  opsxcli kubectl rollout status deployment my-deployment
  opsxcli kubectl rollout status deploy my-deployment -n default`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			resourceType := args[0]
			name := args[1]
			err := kubernetes.RolloutStatus(kubectlKubeconfig, resourceType, name, kubectlNamespace)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
		},
	}

	kubectlRolloutStatusCmd.Flags().StringVarP(&kubectlKubeconfig, "kubeconfig", "k", "", "kubeconfig 文件路径（默认：~/.kube/config）")
	kubectlRolloutStatusCmd.Flags().StringVarP(&kubectlNamespace, "namespace", "n", "", "命名空间（默认：default）")

	kubectlRolloutCmd.AddCommand(kubectlRolloutStatusCmd)

	// kubectl apply 命令
	kubectlApplyCmd := &cobra.Command{
		Use:   "apply -f [文件]",
		Short: "应用配置",
		Long: `从文件应用配置到资源

示例:
  opsxcli kubectl apply -f deployment.yaml
  opsxcli kubectl apply -f service.json`,
		Run: func(cmd *cobra.Command, args []string) {
			filename, _ := cmd.Flags().GetString("filename")
			if filename == "" {
				fmt.Fprintf(os.Stderr, "错误: 必须指定 -f 文件\n")
				os.Exit(1)
			}

			err := kubernetes.ApplyYAML(kubectlKubeconfig, filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
		},
	}

	kubectlApplyCmd.Flags().StringVarP(&kubectlKubeconfig, "kubeconfig", "k", "", "kubeconfig 文件路径（默认：~/.kube/config）")
	kubectlApplyCmd.Flags().StringP("filename", "f", "", "YAML/JSON 文件路径")
	kubectlApplyCmd.MarkFlagRequired("filename")

	// kubectl exec 命令
	var execContainer string
	var execStdin bool
	var execTTY bool
	kubectlExecCmd := &cobra.Command{
		Use:   "exec [pod名称] -- [命令...]",
		Short: "在容器中执行命令",
		Long: `在 Pod 的容器中执行命令

示例:
  opsxcli kubectl exec my-pod -- ls /
  opsxcli kubectl exec my-pod -c container-name -- sh
  opsxcli kubectl exec my-pod -it -- /bin/bash`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			podName := args[0]

			// cobra 会自动处理 "--"，剩余的参数在 args[1:] 中
			command := []string{}
			if cmd.ArgsLenAtDash() > 0 {
				// 使用 ArgsLenAtDash() 获取 -- 后的参数
				command = args[cmd.ArgsLenAtDash():]
			} else if len(args) > 1 {
				// 如果没有 --，直接使用剩余参数
				command = args[1:]
			}

			if len(command) == 0 {
				fmt.Fprintf(os.Stderr, "错误: 必须指定执行的命令 (使用 -- 分隔)\n")
				os.Exit(1)
			}

			err := kubernetes.ExecPod(kubectlKubeconfig, podName, kubectlNamespace, execContainer, command, execStdin, execTTY)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
		},
	}

	kubectlExecCmd.Flags().StringVarP(&kubectlKubeconfig, "kubeconfig", "k", "", "kubeconfig 文件路径（默认：~/.kube/config）")
	kubectlExecCmd.Flags().StringVarP(&kubectlNamespace, "namespace", "n", "", "命名空间（默认：default）")
	kubectlExecCmd.Flags().StringVarP(&execContainer, "container", "c", "", "容器名称")
	kubectlExecCmd.Flags().BoolVarP(&execStdin, "stdin", "i", false, "传递 stdin 到容器")
	kubectlExecCmd.Flags().BoolVarP(&execTTY, "tty", "t", false, "分配 TTY")


	// 添加子命令
	kubectlCmd.AddCommand(kubectlGetCmd)
	kubectlCmd.AddCommand(kubectlDescribeCmd)
	kubectlCmd.AddCommand(kubectlLogsCmd)
	kubectlCmd.AddCommand(kubectlExecCmd)
	kubectlCmd.AddCommand(kubectlDeleteCmd)
	kubectlCmd.AddCommand(kubectlScaleCmd)
	kubectlCmd.AddCommand(kubectlRolloutCmd)
	kubectlCmd.AddCommand(kubectlApplyCmd)

	return kubectlCmd
}
