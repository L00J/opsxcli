package ssh

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"opsxcli/internal/logger"
	"opsxcli/plugins/ssh/client"
)

// NewExecCmd 创建执行命令
func NewExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [user@]host [command]",
		Short: "在远程主机执行命令",
		Long:  "连接到远程主机并执行命令",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			command := strings.Join(args[1:], " ")
			keyPath, _ := cmd.Flags().GetString("key")
			port, _ := cmd.Flags().GetInt("port")
			password, _ := cmd.Flags().GetString("password")

			return execCommand(target, keyPath, port, password, command)
		},
	}

	cmd.Flags().StringP("key", "i", "", "SSH私钥路径")
	cmd.Flags().IntP("port", "p", 22, "SSH端口")
	cmd.Flags().StringP("password", "P", "", "密码")

	return cmd
}

// ExecCommand 执行命令（导出函数）
func ExecCommand(target, keyPath string, port int, password, command string) error {
	return execCommand(target, keyPath, port, password, command)
}

func execCommand(target, keyPath string, port int, password, command string) error {
	// 解析目标
	user, host, err := parseTarget(target)
	if err != nil {
		return err
	}

	// 创建SSH客户端
	sshClient, err := client.NewSSHClient(host, port, user, keyPath, password)
	if err != nil {
		logger.Error("SSH执行命令失败 %s@%s:%d: %v", user, host, port, err)
		return fmt.Errorf("连接失败: %v", err)
	}
	defer sshClient.Close()

	// 执行命令
	output, err := sshClient.Exec(command)
	if err != nil {
		// 检查是否是退出错误
		if exitError, ok := err.(*ssh.ExitError); ok {
			exitStatus := exitError.ExitStatus()
			// 输出命令结果（即使有退出状态）
			if output != "" {
				fmt.Print(output)
			}
			// 非零退出状态才作为错误返回
			if exitStatus != 0 {
				return fmt.Errorf("远程命令退出状态: %d", exitStatus)
			}
			return nil
		}
		return err
	}

	fmt.Print(output)
	return nil
}
