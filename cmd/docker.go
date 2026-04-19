package cmd

import (
	"opsxcli/plugins/docker"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("docker", "Docker", "镜像管理", NewDockerCmd)
}

// NewDockerCmd 创建 Docker 命令
func NewDockerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Docker 镜像管理工具集",
		Long: `Docker 工具集提供以下功能：
  - pull: 拉取 Docker 镜像（支持多源极速、并发下载、断点续传）
  - push: 推送 Docker 镜像到仓库 (TODO)
  - images: 列出本地镜像 (TODO)
  - rmi: 删除本地镜像 (TODO)

使用 'opsxcli docker <command> --help' 查看具体命令的帮助信息。`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// 添加子命令
	cmd.AddCommand(NewDockerPullCmd())

	return cmd
}

// NewDockerPullCmd 创建 docker pull 命令
func NewDockerPullCmd() *cobra.Command {
	var (
		registries  []string // 镜像源列表
		concurrency int      // 并发数
	)

	cmd := &cobra.Command{
		Use:   "pull <image> [image...]",
		Short: "拉取 Docker 镜像（自动加速、多源并发、断点续传）",
		Long: `拉取一个或多个 Docker 镜像，自动选择最快的镜像源极速下载

功能特性：
  ✓ 自动测速 - 智能选择最快的镜像源
  ✓ 多镜像并发 - 同时拉取多个镜像，提高效率
  ✓ 多源极速 - 镜像源故障自动切换
  ✓ 断点续传 - 网络中断后可继续下载
  ✓ 健康监控 - 动态调整镜像源优先级

示例:
  # 拉取单个镜像（自动加速）
  opsxcli docker pull nginx:latest

  # 拉取多个镜像（并发）
  opsxcli docker pull nginx:latest redis:alpine mysql:8.0

  # 使用自定义镜像源
  opsxcli docker pull nginx:latest -r docker.1ms.run -r dockerproxy.com

  # 设置并发数
  opsxcli docker pull nginx redis mysql -c 5`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			images := args

			// 配置下载选项
			opts := &docker.PullOptions{
				Images:      images,
				Registries:  registries,
				Concurrency: concurrency,
			}

			return docker.Pull(opts)
		},
	}

	// 添加 flags
	cmd.Flags().StringSliceVarP(&registries, "registry", "r", []string{}, "自定义镜像源（可指定多个，留空使用默认源）")
	cmd.Flags().IntVarP(&concurrency, "concurrency", "c", 5, "并发下载数量")

	return cmd
}
