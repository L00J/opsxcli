/*
 * @Author: Logan.Li
 * @Gitee: https://gitee.com/attacker
 * @email: admin@attacker.club
 * @Date: 2025-12-16 16:33:40
 * @LastEditTime: 2025-12-16 17:16:23
 * @Description:
 */
package cmd

import (
	"opsxcli/plugins/netstat"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("ss", "网络", "Socket统计", NewSsCmd)
}

// NewSsCmd 创建ss命令（在Linux上使用ss命令，类似ss命令）
func NewSsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ss",
		Short: "网络连接状态查看（高性能，类似ss命令）",
		Long: `显示网络连接状态，直接读取/proc/net文件，性能优于netstat

示例:
  # 查看所有TCP和UDP连接
  opsxcli ss -tunap

  # 查看TCP状态统计
  opsxcli ss -ant --stats

  # 查看TIME_WAIT状态的目标地址TOP 10
  opsxcli ss -tan --timewait

  # 查看目标地址TOP 10
  opsxcli ss -an --top 10

  # 只显示监听状态的连接
  opsxcli ss -l

  # 显示所有连接（包括监听）
  opsxcli ss -a`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			listen, _ := cmd.Flags().GetBool("listen")
			all, _ := cmd.Flags().GetBool("all")
			tcp, _ := cmd.Flags().GetBool("tcp")
			udp, _ := cmd.Flags().GetBool("udp")
			numeric, _ := cmd.Flags().GetBool("numeric")
			programs, _ := cmd.Flags().GetBool("programs")
			stats, _ := cmd.Flags().GetBool("stats")
			timewait, _ := cmd.Flags().GetBool("timewait")
			top, _ := cmd.Flags().GetInt("top")

			// 使用高性能的ss实现（直接读取/proc/net，类似ss命令）
			return netstat.Ss(listen, all, tcp, udp, numeric, programs, stats, timewait, top)
		},
	}

	cmd.Flags().BoolP("listen", "l", false, "只显示监听状态的连接")
	cmd.Flags().BoolP("all", "a", false, "显示所有连接")
	cmd.Flags().BoolP("tcp", "t", false, "显示TCP连接")
	cmd.Flags().BoolP("udp", "u", false, "显示UDP连接")
	cmd.Flags().BoolP("numeric", "n", false, "以数字形式显示地址和端口")
	cmd.Flags().BoolP("programs", "p", false, "显示PID和程序名")
	cmd.Flags().BoolP("stats", "s", false, "统计TCP状态数量")
	cmd.Flags().BoolP("timewait", "w", false, "显示TIME_WAIT状态的目标地址TOP 10")
	cmd.Flags().Int("top", 0, "显示目标地址TOP N（例如: --top 10）")

	// 添加使用示例
	cmd.Example = `  # 查看所有TCP和UDP连接
  opsxcli ss -tunap

  # 查看TCP状态统计
  # 用于查看本机并发和超时等TCP状态
  opsxcli ss -ant --stats

  # 查看TIME_WAIT状态的目标地址TOP 10
  opsxcli ss -tan --timewait

  # 查看目标地址TOP 10
  opsxcli ss -an --top 10

  # 只显示监听状态的连接
  opsxcli ss -l

  # 显示所有连接（包括监听）
  opsxcli ss -a`

	return cmd
}
