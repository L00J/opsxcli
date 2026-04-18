package ssh

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"opsxcli/internal/logger"
	"opsxcli/plugins/ssh/client"
)

// NewForwardCmd 创建端口转发命令
func NewForwardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "forward [local|remote|dynamic] [bind:port] [target:port] [user@]host",
		Short: "SSH端口转发",
		Long: `支持三种端口转发模式：
  - local:  本地端口转发 (-L)
  - remote: 远程端口转发 (-R)
  - dynamic: 动态端口转发/SOCKS代理 (-D)`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := args[0]
			keyPath, _ := cmd.Flags().GetString("key")
			port, _ := cmd.Flags().GetInt("port")
			password, _ := cmd.Flags().GetString("password")

			switch mode {
			case "local":
				if len(args) < 4 {
					err := fmt.Errorf("本地端口转发需要: forward local <bind:port> <target:port> <user@host>")
					logger.Error("%v", err)
					return err
				}
				return forwardLocal(args[1], args[2], args[3], keyPath, port, password)
			case "remote":
				if len(args) < 4 {
					err := fmt.Errorf("远程端口转发需要: forward remote <bind:port> <target:port> <user@host>")
					logger.Error("%v", err)
					return err
				}
				return forwardRemote(args[1], args[2], args[3], keyPath, port, password)
			case "dynamic":
				if len(args) < 3 {
					err := fmt.Errorf("动态端口转发需要: forward dynamic <bind:port> <user@host>")
					logger.Error("%v", err)
					return err
				}
				return forwardDynamic(args[1], args[2], keyPath, port, password)
			default:
				err := fmt.Errorf("未知的转发模式: %s (支持: local, remote, dynamic)", mode)
				logger.Error("%v", err)
				return err
			}
		},
	}

	cmd.Flags().StringP("key", "i", "", "SSH私钥路径")
	cmd.Flags().IntP("port", "p", 22, "SSH端口")
	cmd.Flags().StringP("password", "P", "", "密码")

	return cmd
}

func forwardLocal(bindAddr, targetAddr, target, keyPath string, port int, password string) error {
	// 解析绑定地址
	bindHost, bindPort, err := parseAddr(bindAddr)
	if err != nil {
		return err
	}

	// 解析目标地址
	targetHost, targetPort, err := parseAddr(targetAddr)
	if err != nil {
		return err
	}

	// 解析SSH目标
	user, host, err := parseTarget(target)
	if err != nil {
		return err
	}

	// 创建SSH客户端
	sshClient, err := client.NewSSHClient(host, port, user, keyPath, password)
	if err != nil {
		logger.Error("SSH端口转发连接失败 %s@%s:%d: %v", user, host, port, err)
		return fmt.Errorf("连接失败: %v", err)
	}
	defer sshClient.Close()

	logger.Success("本地端口转发: %s:%d -> %s:%d (通过 %s@%s)", bindHost, bindPort, targetHost, targetPort, user, host)

	// 启动端口转发
	return sshClient.LocalForward(bindHost, bindPort, targetHost, targetPort)
}

func forwardRemote(bindAddr, targetAddr, target, keyPath string, port int, password string) error {
	// 解析绑定地址
	bindHost, bindPort, err := parseAddr(bindAddr)
	if err != nil {
		return err
	}

	// 解析目标地址
	targetHost, targetPort, err := parseAddr(targetAddr)
	if err != nil {
		return err
	}

	// 解析SSH目标
	user, host, err := parseTarget(target)
	if err != nil {
		return err
	}

	// 创建SSH客户端
	sshClient, err := client.NewSSHClient(host, port, user, keyPath, password)
	if err != nil {
		logger.Error("SSH端口转发连接失败 %s@%s:%d: %v", user, host, port, err)
		return fmt.Errorf("连接失败: %v", err)
	}
	defer sshClient.Close()

	logger.Success("远程端口转发: %s:%d -> %s:%d (通过 %s@%s)", bindHost, bindPort, targetHost, targetPort, user, host)

	// 启动端口转发
	return sshClient.RemoteForward(bindHost, bindPort, targetHost, targetPort)
}

func forwardDynamic(bindAddr, target, keyPath string, port int, password string) error {
	// 解析绑定地址
	bindHost, bindPort, err := parseAddr(bindAddr)
	if err != nil {
		return err
	}

	// 解析SSH目标
	user, host, err := parseTarget(target)
	if err != nil {
		return err
	}

	// 创建SSH客户端
	sshClient, err := client.NewSSHClient(host, port, user, keyPath, password)
	if err != nil {
		logger.Error("SSH端口转发连接失败 %s@%s:%d: %v", user, host, port, err)
		return fmt.Errorf("连接失败: %v", err)
	}
	defer sshClient.Close()

	logger.Success("动态端口转发/SOCKS代理: %s:%d (通过 %s@%s)", bindHost, bindPort, user, host)

	// 启动SOCKS代理
	return sshClient.DynamicForward(bindHost, bindPort)
}

// 辅助函数
func parseAddr(addr string) (string, int, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("无效的地址格式: %s (需要 host:port)", addr)
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("无效的端口: %s", parts[1])
	}

	return parts[0], port, nil
}
