/*
 * @Author: Logan.Li
 * @Gitee: https://gitee.com/attacker
 * @email: admin@attacker.club
 * @Date: 2025-12-15 22:47:35
 * @LastEditTime: 2025-12-15 23:07:12
 * @Description:
 */
package cmd

import (
	"fmt"

	"opsxcli/plugins/server"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("server", "工具", "HTTP服务端工具", NewServerCmd)
}

// NewServerCmd 创建HTTP服务器命令
func NewServerCmd() *cobra.Command {
	var port int
	var directory string
	var host string
	var password string

	cmd := &cobra.Command{
		Use:   "server [port] [flags]",
		Short: "启动一个简单的HTTP文件服务器",
		Long: `启动一个简单的HTTP文件服务器，用于在指定目录提供文件服务。

默认在8000端口启动，绑定到0.0.0.0，服务当前目录。

使用示例:
  opsxcli server                              # 在8000端口启动，服务当前目录
  opsxcli server 9000                         # 在9000端口启动
  opsxcli server -p 8080                      # 在8080端口启动
  opsxcli server -d /tmp                      # 服务指定目录
  opsxcli server 9000 -d /var/www              # 指定端口和目录
  opsxcli server -h 0.0.0.0 -p 8000            # 绑定到0.0.0.0（允许外部访问）
  opsxcli server --password admin:abc123       # 启用HTTP基本认证（用户名:密码）`,
		Args:                  cobra.MaximumNArgs(1),
		DisableFlagsInUseLine: true,
		DisableFlagParsing:    false,
		SilenceUsage:          true,  // 错误时不显示 Usage
		SilenceErrors:         false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			// 如果提供了位置参数，使用它作为端口（优先级高于flag）
			if len(args) > 0 {
				// 尝试解析位置参数为端口
				if _, err := fmt.Sscanf(args[0], "%d", &port); err != nil {
					return fmt.Errorf("无效的端口号: %s", args[0])
				}
			}
			return server.Start(host, port, directory, password)
		},
	}

	// 先手动添加 help flag（没有 shorthand），避免 Cobra 自动添加带 -h 的 help flag
	// 必须在添加其他 flags 之前，这样 Cobra 在 InitDefaultHelpFlag 时会发现 help flag 已存在
	cmd.Flags().Bool("help", false, "help for server")

	// 然后添加 host flag（使用 -h）
	cmd.Flags().StringVarP(&host, "host", "h", "0.0.0.0", "绑定地址（默认：0.0.0.0，允许外部访问）")

	// 添加其他 flags
	cmd.Flags().IntVarP(&port, "port", "p", 8000, "服务器端口")
	cmd.Flags().StringVarP(&directory, "directory", "d", "", "服务目录（默认：当前目录）")
	cmd.Flags().StringVar(&password, "password", "", "HTTP基本认证（格式：username:password，例如：admin:abc123）")

	return cmd
}
