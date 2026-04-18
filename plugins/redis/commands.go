package redis

import (
	"time"

	"github.com/spf13/cobra"
)

// NewGetCmd 创建GET命令
func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "获取键值",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			password, _ := cmd.Flags().GetString("password")
			db, _ := cmd.Flags().GetInt("db")
			cluster, _ := cmd.Flags().GetBool("cluster")
			addrs, _ := cmd.Flags().GetString("addrs")

			return Get(host, port, password, db, cluster, addrs, args[0])
		},
	}

	return cmd
}

// NewSetCmd 创建SET命令
func NewSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "设置键值",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			password, _ := cmd.Flags().GetString("password")
			db, _ := cmd.Flags().GetInt("db")
			cluster, _ := cmd.Flags().GetBool("cluster")
			addrs, _ := cmd.Flags().GetString("addrs")
			expireStr, _ := cmd.Flags().GetString("expire")

			var expire time.Duration
			if expireStr != "" {
				var err error
				expire, err = time.ParseDuration(expireStr)
				if err != nil {
					return err
				}
			}

			return Set(host, port, password, db, cluster, addrs, args[0], args[1], expire)
		},
	}

	cmd.Flags().StringP("expire", "e", "", "过期时间（如: 1h, 30m, 60s）")

	return cmd
}

// NewInteractiveCmd 创建交互式命令
func NewInteractiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "interactive",
		Short: "进入交互式Redis shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			password, _ := cmd.Flags().GetString("password")
			db, _ := cmd.Flags().GetInt("db")
			cluster, _ := cmd.Flags().GetBool("cluster")
			addrs, _ := cmd.Flags().GetString("addrs")

			return Interactive(host, port, password, db, cluster, addrs)
		},
	}

	return cmd
}
