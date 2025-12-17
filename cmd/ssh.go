package cmd

import (
	"strings"

	"opsxcli/plugins/ssh"

	"github.com/spf13/cobra"
)

// NewSSHCmd 创建SSH命令
func NewSSHCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ssh [user@]host [command]",
		Short: "SSH连接、命令执行、文件传输、端口转发",
		Long: `SSH工具支持以下功能：
  - 交互式登录
  - 远程命令执行
  - SFTP文件传输（上传/下载）
  - 端口转发（本地/远程/动态）

使用示例:
  opsxcli ssh root@10.10.10.156              # 交互式登录
  opsxcli ssh root@10.10.10.156 "ls -la"     # 执行命令
  opsxcli ssh put local.txt root@host:/tmp/  # 上传文件
  opsxcli ssh get root@host:/tmp/file .      # 下载文件
  opsxcli ssh forward local 8080:80 root@host # 端口转发`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			// 如果没有子命令且有参数，直接执行连接或命令
			target := args[0]
			keyPath, _ := cmd.Flags().GetString("key")
			port, _ := cmd.Flags().GetInt("port")
			password, _ := cmd.Flags().GetString("password")

			// 如果只有一个参数，执行交互式连接
			if len(args) == 1 {
				return ssh.Connect(target, keyPath, port, password)
			}

			// 如果有多个参数，执行命令
			command := strings.Join(args[1:], " ")
			return ssh.ExecCommand(target, keyPath, port, password, command)
		},
	}

	// 添加全局参数
	rootCmd.Flags().StringP("key", "i", "", "SSH私钥路径")
	rootCmd.Flags().IntP("port", "p", 22, "SSH端口")
	rootCmd.Flags().StringP("password", "P", "", "密码（不指定则提示输入）")

	// 添加子命令（connect 和 exec 已通过主命令支持，保留用于向后兼容）
	rootCmd.AddCommand(
		ssh.NewPutCmd(),
		ssh.NewGetCmd(),
		ssh.NewForwardCmd(),
	)

	return rootCmd
}
