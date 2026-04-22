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
  - images:  列出镜像
  - rmi:     删除镜像
  - tag:     为镜像打标签
  - push:    推送镜像
  - inspect: 查看镜像详细信息
  - search:  搜索 Docker Hub 镜像
  - save:    导出镜像为 tar 文件
  - load:    从 tar 文件导入镜像
  - history: 查看镜像构建历史
  - prune:   清理未使用的镜像

容器管理：
  - ps:      列出容器
  - inspect: 查看容器详细信息
  - logs:    查看容器日志
  - start:   启动容器
  - stop:    停止容器
  - restart: 重启容器
  - rm:      删除容器

监控与系统：
  - stats:     查看容器资源使用统计
  - top:       查看容器内进程
  - events:    监听 Docker 事件
  - system-df: 查看 Docker 磁盘使用

使用 'opsxcli docker <command> --help' 查看具体命令的帮助信息。`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// 添加子命令 - 镜像拉取
	cmd.AddCommand(NewDockerPullCmd())

	// 添加子命令 - 镜像管理
	cmd.AddCommand(NewDockerImagesCmd())
	cmd.AddCommand(NewDockerRMICmd())
	cmd.AddCommand(NewDockerTagCmd())
	cmd.AddCommand(NewDockerPushCmd())
	cmd.AddCommand(NewDockerInspectImageCmd())
	cmd.AddCommand(NewDockerSearchCmd())
	cmd.AddCommand(NewDockerSaveCmd())
	cmd.AddCommand(NewDockerLoadCmd())
	cmd.AddCommand(NewDockerHistoryCmd())
	cmd.AddCommand(NewDockerPruneCmd())

	// 添加子命令 - 容器管理
	cmd.AddCommand(NewDockerPSCmd())
	cmd.AddCommand(NewDockerInspectCmd())
	cmd.AddCommand(NewDockerLogsCmd())
	cmd.AddCommand(NewDockerStartCmd())
	cmd.AddCommand(NewDockerStopCmd())
	cmd.AddCommand(NewDockerRestartCmd())
	cmd.AddCommand(NewDockerRMCmd())

	// 添加子命令 - 监控与系统
	cmd.AddCommand(NewDockerStatsCmd())
	cmd.AddCommand(NewDockerTopCmd())
	cmd.AddCommand(NewDockerEventsCmd())
	cmd.AddCommand(NewDockerSystemDFCmd())

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

// --- 镜像管理命令 ---

// NewDockerImagesCmd 创建 docker images 命令
func NewDockerImagesCmd() *cobra.Command {
	var (
		all     bool
		filter  string
		quiet   bool
		noTrunc bool
	)

	cmd := &cobra.Command{
		Use:   "images",
		Short: "列出 Docker 镜像",
		Long: `列出本地 Docker 镜像

示例:
  # 列出所有镜像
  opsxcli docker images

  # 包括中间层镜像
  opsxcli docker images --all

  # 只显示镜像 ID
  opsxcli docker images -q

  # 按条件过滤
  opsxcli docker images --filter dangling=true`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &docker.ImagesOptions{
				All:     all,
				Filters: filter,
				Quiet:   quiet,
				NoTrunc: noTrunc,
			}
			return docker.Images(opts)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "显示所有镜像（包括中间层）")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "过滤条件 (dangling=true, reference=nginx 等)")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "只显示镜像 ID")
	cmd.Flags().BoolVar(&noTrunc, "no-trunc", false, "不截断镜像 ID")

	return cmd
}

// NewDockerRMICmd 创建 docker rmi 命令
func NewDockerRMICmd() *cobra.Command {
	var (
		force   bool
		noPrune bool
	)

	cmd := &cobra.Command{
		Use:   "rmi <image> [image...]",
		Short: "删除一个或多个镜像",
		Long: `删除本地 Docker 镜像

示例:
  opsxcli docker rmi nginx:latest
  opsxcli docker rmi nginx:latest redis:alpine
  opsxcli docker rmi -f nginx:latest    # 强制删除`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &docker.RMIOptions{
				Images:  args,
				Force:   force,
				NoPrune: noPrune,
			}
			return docker.RMI(opts)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "强制删除")
	cmd.Flags().BoolVar(&noPrune, "no-prune", false, "不删除未标记的父镜像")

	return cmd
}

// NewDockerTagCmd 创建 docker tag 命令
func NewDockerTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag <source> <target>",
		Short: "为镜像打标签",
		Long: `为 Docker 镜像创建标签

示例:
  opsxcli docker tag nginx:latest myregistry/nginx:latest
  opsxcli docker tag abc123 myapp:v1.0`,
		Args:          cobra.ExactArgs(2),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &docker.TagOptions{
				Source: args[0],
				Target: args[1],
			}
			return docker.Tag(opts)
		},
	}

	return cmd
}

// NewDockerPushCmd 创建 docker push 命令
func NewDockerPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <image>",
		Short: "推送镜像到仓库",
		Long: `推送 Docker 镜像到远程仓库

示例:
  opsxcli docker push myregistry/nginx:latest
  opsxcli docker push myrepo/myapp:v1.0`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &docker.PushImageOptions{
				Image: args[0],
			}
			return docker.PushImage(opts)
		},
	}

	return cmd
}

// NewDockerInspectImageCmd 创建 docker inspect-image 命令
func NewDockerInspectImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect-image <image>",
		Short: "查看镜像详细信息",
		Long: `查看 Docker 镜像的详细信息，包括层、配置、环境变量等

示例:
  opsxcli docker inspect-image nginx:latest
  opsxcli docker inspect-image abc123def456`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := docker.InspectImage(args[0])
			if err != nil {
				return err
			}
			docker.PrintImageInspect(result)
			return nil
		},
	}

	return cmd
}

// NewDockerSearchCmd 创建 docker search 命令
func NewDockerSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search <term>",
		Short: "搜索 Docker Hub 镜像",
		Long: `在 Docker Hub 搜索镜像

示例:
  opsxcli docker search nginx
  opsxcli docker search python --limit 5`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return docker.SearchImages(args[0], limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 25, "返回结果数量上限")

	return cmd
}

// NewDockerSaveCmd 创建 docker save 命令
func NewDockerSaveCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "save <image>",
		Short: "导出镜像为 tar 文件",
		Long: `将 Docker 镜像导出为 tar 文件

示例:
  opsxcli docker save nginx:latest -o nginx.tar
  opsxcli docker save nginx:latest > nginx.tar`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return docker.SaveImage(args[0], output)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "输出文件路径")

	return cmd
}

// NewDockerLoadCmd 创建 docker load 命令
func NewDockerLoadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load <tar-file>",
		Short: "从 tar 文件导入镜像",
		Long: `从 tar 文件导入 Docker 镜像

示例:
  opsxcli docker load nginx.tar`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return docker.LoadImage(args[0])
		},
	}

	return cmd
}

// NewDockerHistoryCmd 创建 docker history 命令
func NewDockerHistoryCmd() *cobra.Command {
	var (
		noTrunc bool
		quiet   bool
	)

	cmd := &cobra.Command{
		Use:   "history <image>",
		Short: "查看镜像构建历史",
		Long: `查看 Docker 镜像的构建历史（各层信息）

示例:
  opsxcli docker history nginx:latest
  opsxcli docker history nginx:latest --no-trunc
  opsxcli docker history nginx:latest -q`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return docker.HistoryImages(args[0], noTrunc, quiet)
		},
	}

	cmd.Flags().BoolVar(&noTrunc, "no-trunc", false, "不截断输出")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "只显示镜像 ID")

	return cmd
}

// NewDockerPruneCmd 创建 docker prune 命令
func NewDockerPruneCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "清理未使用的镜像",
		Long: `清理未使用的 Docker 镜像，释放磁盘空间

示例:
  # 清理悬空镜像（无标签的）
  opsxcli docker prune

  # 清理所有未使用的镜像
  opsxcli docker prune --all`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return docker.PruneImages(all)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "清理所有未使用的镜像（不仅仅是悬空的）")

	return cmd
}

// --- 监控与系统命令 ---

// NewDockerStatsCmd 创建 docker stats 命令
func NewDockerStatsCmd() *cobra.Command {
	var (
		noStream bool
		noTrunc  bool
	)

	cmd := &cobra.Command{
		Use:   "stats [container...]",
		Short: "查看容器资源使用统计",
		Long: `显示容器的实时资源使用统计（CPU、内存、网络、磁盘 I/O）

示例:
  # 查看所有运行中容器的统计
  opsxcli docker stats

  # 查看指定容器的统计
  opsxcli docker stats nginx redis

  # 只显示一次（不实时刷新）
  opsxcli docker stats --no-stream`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &docker.StatsOptions{
				Containers: args,
				NoStream:   noStream,
				NoTrunc:    noTrunc,
			}
			return docker.Stats(opts)
		},
	}

	cmd.Flags().BoolVar(&noStream, "no-stream", false, "只显示一次统计（不实时刷新）")
	cmd.Flags().BoolVar(&noTrunc, "no-trunc", false, "不截断容器 ID")

	return cmd
}

// NewDockerTopCmd 创建 docker top 命令
func NewDockerTopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top <container>",
		Short: "查看容器内运行的进程",
		Long: `显示指定容器内正在运行的进程

示例:
  opsxcli docker top nginx
  opsxcli docker top abc123def456`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return docker.Top(args[0])
		},
	}

	return cmd
}

// NewDockerEventsCmd 创建 docker events 命令
func NewDockerEventsCmd() *cobra.Command {
	var (
		since    string
		until    string
		filter   string
		duration int
	)

	cmd := &cobra.Command{
		Use:   "events",
		Short: "监听 Docker 事件",
		Long: `实时监听 Docker 引擎事件（容器启停、镜像拉取等）

示例:
  # 监听实时事件（默认 60 秒）
  opsxcli docker events

  # 监听最近 1 小时的事件
  opsxcli docker events --since 1h

  # 监听指定容器的事件
  opsxcli docker events --filter container=nginx

  # 监听 30 秒
  opsxcli docker events --duration 30`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &docker.EventsOptions{
				Since:    since,
				Until:    until,
				Filters:  filter,
				Duration: duration,
			}
			return docker.Events(opts)
		},
	}

	cmd.Flags().StringVar(&since, "since", "", "显示指定时间之后的事件 (如 1h, 30m, 2024-01-01)")
	cmd.Flags().StringVar(&until, "until", "", "显示指定时间之前的事件")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "过滤条件 (container=xxx, type=container 等)")
	cmd.Flags().IntVarP(&duration, "duration", "d", 0, "监听持续时间（秒），默认 60 秒")

	return cmd
}

// NewDockerSystemDFCmd 创建 docker system-df 命令
func NewDockerSystemDFCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system-df",
		Short: "查看 Docker 磁盘使用情况",
		Long: `显示 Docker 使用的磁盘空间概览

示例:
  opsxcli docker system-df`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return docker.SystemDF()
		},
	}

	return cmd
}
