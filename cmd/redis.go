package cmd

import (
	"os"

	"opsxcli/plugins/redis"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("redis", "数据库", "Redis数据库操作", NewRedisCmd)
}

// NewRedisCmd 创建Redis命令
func NewRedisCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "redis",
		Short: "Redis操作工具（支持单机和集群）",
		Long: `Redis工具支持以下功能：
  - 交互式Redis shell
  - 基本命令（GET, SET等）
  - 集群模式支持

Examples:
  # 连接本地Redis（默认端口6379）
  opsxcli redis

  # 使用环境变量密码（官方标准）
  export REDISCLI_AUTH=your_password
  opsxcli redis -h 192.168.1.100 -p 6379

  # 直接指定密码
  opsxcli redis -h 192.168.1.100 -p 6379 -a password

  # 连接指定数据库
  opsxcli redis -h localhost -p 6379 -d 1

  # 连接Redis集群
  opsxcli redis --cluster --addrs "192.168.1.100:7000,192.168.1.100:7001,192.168.1.100:7002"

  # 带密码的集群连接（使用环境变量）
  export REDISCLI_AUTH=password
  opsxcli redis --cluster --addrs "node1:7000,node2:7000,node3:7000"`,
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
	}

	// 如果没有子命令，进入交互模式
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		password, _ := cmd.Flags().GetString("password")
		db, _ := cmd.Flags().GetInt("db")
		cluster, _ := cmd.Flags().GetBool("cluster")
		addrs, _ := cmd.Flags().GetString("addrs")

		// Redis 密码处理（按优先级）：
		// 1. 命令行 -a 参数
		// 2. REDISCLI_AUTH 环境变量（官方标准）
		// 3. 无密码
		if password == "" {
			password = os.Getenv("REDISCLI_AUTH")
		}

		return redis.Interactive(host, port, password, db, cluster, addrs)
	}

	// 先添加自定义 flags
	rootCmd.PersistentFlags().StringP("host", "h", "127.0.0.1", "Redis主机地址")
	rootCmd.PersistentFlags().IntP("port", "p", 6379, "Redis端口")
	rootCmd.PersistentFlags().StringP("password", "a", "", "Redis密码（或使用环境变量 REDISCLI_AUTH）")
	rootCmd.PersistentFlags().IntP("db", "d", 0, "数据库编号")
	rootCmd.PersistentFlags().BoolP("cluster", "c", false, "集群模式")
	rootCmd.PersistentFlags().String("addrs", "", "集群地址列表（逗号分隔）")

	// 手动添加 help flag（没有 shorthand）
	rootCmd.PersistentFlags().Bool("help", false, "help for redis")

	// 添加子命令
	rootCmd.AddCommand(
		redis.NewGetCmd(),
		redis.NewSetCmd(),
		redis.NewInteractiveCmd(),
		redis.NewAnalyzeCmd(),
	)

	return rootCmd
}
