package client

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"opsxcli/internal/config"
	"opsxcli/internal/logger"
)

// SSHClient SSH客户端封装
type SSHClient struct {
	client *ssh.Client
	config *ssh.ClientConfig
	host   string
	port   int
}

// NewSSHClient 创建SSH客户端
func NewSSHClient(host string, port int, user, keyPath, password string) (*SSHClient, error) {
	// 如果没有指定keyPath，使用默认配置
	if keyPath == "" {
		cfg := config.Get()
		keyPath = cfg.SSH.DefaultKeyPath
	}

	// 构建认证方法
	var authMethods []ssh.AuthMethod

	// 尝试使用私钥认证
	if keyPath != "" {
		if key, err := loadPrivateKey(keyPath); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(key))
		}
	}

	// 如果提供了密码，添加密码认证
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	} else if password == "" && keyPath == "" {
		// 如果没有私钥也没有密码，尝试交互式输入密码
		fmt.Fprintf(os.Stderr, "%s@%s's password: ", user, host)
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return nil, fmt.Errorf("读取密码失败: %v", err)
		}
		fmt.Fprintf(os.Stderr, "\n")
		password = string(passwordBytes)
		if password != "" {
			authMethods = append(authMethods, ssh.Password(password))
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("没有可用的认证方法，请提供私钥或密码")
	}

	// 构建SSH配置
	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应使用knownhosts
		Timeout:         time.Duration(config.Get().SSH.Timeout) * time.Second,
	}

	// 连接SSH服务器
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, err
	}

	return &SSHClient{
		client: client,
		config: sshConfig,
		host:   host,
		port:   port,
	}, nil
}

// NewSession 创建新会话
func (c *SSHClient) NewSession() (*ssh.Session, error) {
	return c.client.NewSession()
}

// Exec 执行命令
// Exec 执行远程命令并返回输出
func (c *SSHClient) Exec(command string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	return string(output), err
}

// Close 关闭连接
func (c *SSHClient) Close() error {
	return c.client.Close()
}

// LocalForward 本地端口转发
func (c *SSHClient) LocalForward(bindHost string, bindPort int, targetHost string, targetPort int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bindHost, bindPort))
	if err != nil {
		return err
	}
	defer listener.Close()

	logger.Info("监听 %s:%d，转发到 %s:%d", bindHost, bindPort, targetHost, targetPort)

	for {
		localConn, err := listener.Accept()
		if err != nil {
			logger.Error("接受连接失败: %v", err)
			continue
		}

		go func() {
			defer localConn.Close()
			remoteConn, err := c.client.Dial("tcp", fmt.Sprintf("%s:%d", targetHost, targetPort))
			if err != nil {
				logger.Error("连接远程失败: %v", err)
				return
			}
			defer remoteConn.Close()

			// 双向转发
			go io.Copy(remoteConn, localConn)
			io.Copy(localConn, remoteConn)
		}()
	}
}

// RemoteForward 远程端口转发
func (c *SSHClient) RemoteForward(bindHost string, bindPort int, targetHost string, targetPort int) error {
	listener, err := c.client.Listen("tcp", fmt.Sprintf("%s:%d", bindHost, bindPort))
	if err != nil {
		return err
	}
	defer listener.Close()

	logger.Info("远程监听 %s:%d，转发到 %s:%d", bindHost, bindPort, targetHost, targetPort)

	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			logger.Error("接受连接失败: %v", err)
			continue
		}

		go func() {
			defer remoteConn.Close()
			localConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", targetHost, targetPort))
			if err != nil {
				logger.Error("连接本地失败: %v", err)
				return
			}
			defer localConn.Close()

			// 双向转发
			go io.Copy(localConn, remoteConn)
			io.Copy(remoteConn, localConn)
		}()
	}
}

// DynamicForward 动态端口转发（SOCKS代理）
func (c *SSHClient) DynamicForward(bindHost string, bindPort int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bindHost, bindPort))
	if err != nil {
		return err
	}
	defer listener.Close()

	logger.Info("SOCKS代理监听 %s:%d", bindHost, bindPort)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			logger.Error("接受连接失败: %v", err)
			continue
		}

		go handleSOCKS(clientConn, c.client)
	}
}

// loadPrivateKey 加载私钥
func loadPrivateKey(keyPath string) (ssh.Signer, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		// 尝试使用密码保护的私钥
		return nil, err
	}

	return signer, nil
}

// handleSOCKS 处理SOCKS连接（简化版）
func handleSOCKS(clientConn net.Conn, sshClient *ssh.Client) {
	defer clientConn.Close()
	// SOCKS协议实现（简化版，实际应实现完整SOCKS5协议）
	// 这里只是示例，实际使用需要完整的SOCKS5实现
}
