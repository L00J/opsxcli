package ssh

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"opsxcli/internal/logger"
	"opsxcli/plugins/ssh/client"
)

// NewConnectCmd 创建连接命令
func NewConnectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect [user@]host",
		Short: "交互式SSH连接",
		Long:  "建立SSH连接并进入交互式shell",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			keyPath, _ := cmd.Flags().GetString("key")
			port, _ := cmd.Flags().GetInt("port")
			password, _ := cmd.Flags().GetString("password")

			return connect(target, keyPath, port, password)
		},
	}

	cmd.Flags().StringP("key", "i", "", "SSH私钥路径")
	cmd.Flags().IntP("port", "p", 22, "SSH端口")
	cmd.Flags().StringP("password", "P", "", "密码（不指定则提示输入）")

	return cmd
}

// Connect 连接SSH（导出函数）
func Connect(target, keyPath string, port int, password string) error {
	return connect(target, keyPath, port, password)
}

func connect(target, keyPath string, port int, password string) error {
	// 解析目标
	user, host, err := parseTarget(target)
	if err != nil {
		return err
	}

	// 创建SSH客户端
	sshClient, err := client.NewSSHClient(host, port, user, keyPath, password)
	if err != nil {
		logger.Error("SSH连接失败 %s@%s:%d: %v", user, host, port, err)
		return fmt.Errorf("连接失败: %v", err)
	}
	defer sshClient.Close()

	logger.Success("已连接到 %s@%s", user, host)

	// 创建交互式会话
	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 设置终端
	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer terminal.Restore(fd, oldState)

	// 设置终端大小
	width, height, err := terminal.GetSize(fd)
	if err == nil {
		terminalWidth := int(width)
		terminalHeight := int(height)
		if err := session.RequestPty("xterm-256color", terminalHeight, terminalWidth, ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}); err != nil {
			return err
		}
	}

	// 连接标准输入输出
	session.Stdout = os.Stdout
	session.Stdin = os.Stdin
	session.Stderr = os.Stderr

	// 启动shell
	if err := session.Shell(); err != nil {
		return err
	}

	// 等待会话结束
	err = session.Wait()
	if err != nil {
		// 检查是否是退出错误（正常退出或Ctrl+C）
		if exitError, ok := err.(*ssh.ExitError); ok {
			exitStatus := exitError.ExitStatus()
			// 状态 130 通常是 Ctrl+C (SIGINT)，状态 0 是正常退出
			// 这些情况不应该作为错误返回
			if exitStatus == 0 || exitStatus == 130 {
				return nil
			}
			// 其他非零退出状态才作为错误
			return fmt.Errorf("远程命令退出状态: %d", exitStatus)
		}
		// 其他错误正常返回
		return err
	}
	return nil
}
