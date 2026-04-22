package cmd

import (
	"opsxcli/plugins/docker"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("docker", "Docker", "镜像与容器管理工具集", NewDockerCmd)
}

// NewDockerCmd 创建 Docker 命令
func NewDockerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Docker 镜像与容器管理工具集",
		Long: `Docker 工具集提供以下功能：

镜像管理：
  - pull:    拉取 Docker 镜像（支持多源极速、并发下载、断点续传）

容器管理：
  - ps:      列出容器
  - inspect: 查看容器详细信息
  - logs:    查看容器日志
  - start:   启动容器
  - stop:    停止容器
  - restart: 重启容器
  - rm:      删除容器

使用 'opsxcli docker <command> --help' 查看具体命令的帮助信息。`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// 添加子命令
	cmd.AddCommand(NewDockerPullCmd())
	cmd.AddCommand(NewDockerPSCmd())
	cmd.AddCommand(NewDockerInspectCmd())
	cmd.AddCommand(NewDockerLogsCmd())
	cmd.AddCommand(NewDockerStartCmd())
	cmd.AddCommand(NewDockerStopCmd())
	cmd.AddCommand(NewDockerRestartCmd())
	cmd.AddCommand(NewDockerRMCmd())

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

// NewDockerPSCmd 创建 docker ps 命令
func NewDockerPSCmd() *cobra.Command {
	var (
		all     bool
		last    int
		filter  string
		noTrunc bool
		quiet   bool
	)

	cmd := &cobra.Command{
		Use:   "ps",
		Short: "列出 Docker 容器",
		Long: `列出 Docker 容器

示例:
  # 列出运行中的容器
  opsxcli docker ps

  # 列出所有容器（包括已停止的）
  opsxcli docker ps -a

  # 显示最近 5 个容器
  opsxcli docker ps -n 5

  # 按名称过滤
  opsxcli docker ps --filter name=nginx

  # 只显示容器 ID
  opsxcli docker ps -q`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &docker.PSOptions{
				All:     all,
				Last:    last,
				Filter:  filter,
				NoTrunc: noTrunc,
				Quiet:   quiet,
			}
			return docker.PS(opts)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "显示所有容器（包括已停止的）")
	cmd.Flags().IntVarP(&last, "last", "n", 0, "显示最近创建的 N 个容器")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "过滤条件 (name=xxx, status=running 等)")
	cmd.Flags().BoolVar(&noTrunc, "no-trunc", false, "不截断容器 ID")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "只显示容器 ID")

	return cmd
}

// NewDockerInspectCmd 创建 docker inspect 命令
func NewDockerInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect <container>",
		Short: "查看容器详细信息",
		Long: `查看容器的详细信息，包括网络配置、挂载点、环境变量等

示例:
  opsxcli docker inspect my-container
  opsxcli docker inspect abc123def456`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := docker.Inspect(args[0])
			if err != nil {
				return err
			}
			docker.PrintInspect(result)
			return nil
		},
	}

	return cmd
}

// NewDockerLogsCmd 创建 docker logs 命令
func NewDockerLogsCmd() *cobra.Command {
	var (
		tail   int
		follow bool
		since  string
	)

	cmd := &cobra.Command{
		Use:   "logs <container>",
		Short: "查看容器日志",
		Long: `获取容器的标准输出日志

示例:
  # 查看最近 100 行日志
  opsxcli docker logs my-container --tail 100

  # 实时跟踪日志
  opsxcli docker logs my-container -f

  # 查看指定时间之后的日志
  opsxcli docker logs my-container --since 1h`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return docker.Logs(args[0], tail, follow, since)
		},
	}

	cmd.Flags().IntVar(&tail, "tail", 0, "显示最后 N 行日志 (0=全部)")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "实时跟踪日志输出")
	cmd.Flags().StringVar(&since, "since", "", "显示指定时间之后的日志 (如 1h, 30m, 2024-01-01)")

	return cmd
}

// NewDockerStopCmd 创建 docker stop 命令
func NewDockerStopCmd() *cobra.Command {
	var timeout int

	cmd := &cobra.Command{
		Use:   "stop <container> [container...]",
		Short: "停止一个或多个容器",
		Long: `停止运行中的容器

示例:
  opsxcli docker stop my-container
  opsxcli docker stop container1 container2
  opsxcli docker stop --timeout 30 my-container`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &docker.ContainerActionOptions{
				Containers: args,
				Timeout:    timeout,
			}
			return docker.Stop(opts)
		},
	}

	cmd.Flags().IntVarP(&timeout, "timeout", "t", 0, "停止超时秒数 (默认 10)")

	return cmd
}

// NewDockerStartCmd 创建 docker start 命令
func NewDockerStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <container> [container...]",
		Short: "启动一个或多个已停止的容器",
		Long: `启动已停止的容器

示例:
  opsxcli docker start my-container
  opsxcli docker start container1 container2`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &docker.ContainerActionOptions{
				Containers: args,
			}
			return docker.Start(opts)
		},
	}

	return cmd
}

// NewDockerRestartCmd 创建 docker restart 命令
func NewDockerRestartCmd() *cobra.Command {
	var timeout int

	cmd := &cobra.Command{
		Use:   "restart <container> [container...]",
		Short: "重启一个或多个容器",
		Long: `重启容器（先停止再启动）

示例:
  opsxcli docker restart my-container
  opsxcli docker restart container1 container2
  opsxcli docker restart --timeout 30 my-container`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &docker.ContainerActionOptions{
				Containers: args,
				Timeout:    timeout,
			}
			return docker.Restart(opts)
		},
	}

	cmd.Flags().IntVarP(&timeout, "timeout", "t", 0, "重启超时秒数")

	return cmd
}

// NewDockerRMCmd 创建 docker rm 命令
func NewDockerRMCmd() *cobra.Command {
	var (
		force   bool
		volumes bool
	)

	cmd := &cobra.Command{
		Use:   "rm <container> [container...]",
		Short: "删除一个或多个容器",
		Long: `删除已停止的容器

示例:
  opsxcli docker rm my-container
  opsxcli docker rm container1 container2
  opsxcli docker rm -f my-container    # 强制删除运行中的容器
  opsxcli docker rm -v my-container    # 同时删除关联的匿名卷`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &docker.ContainerActionOptions{
				Containers: args,
				Force:      force,
				Volumes:    volumes,
			}
			return docker.RM(opts)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "强制删除运行中的容器")
	cmd.Flags().BoolVarP(&volumes, "volumes", "v", false, "同时删除关联的匿名卷")

	return cmd
}
