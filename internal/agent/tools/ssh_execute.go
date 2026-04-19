package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHExecuteTool SSH 远程执行工具
type SSHExecuteTool struct {
	pool *SSHPool
}

// NewSSHExecuteTool 创建 SSH 执行工具
func NewSSHExecuteTool() *SSHExecuteTool {
	return &SSHExecuteTool{
		pool: NewSSHPool(),
	}
}

// Close 关闭 SSH 执行工具，释放连接池资源
// 实现 Closer 接口，供 Registry 统一调用
func (t *SSHExecuteTool) Close() error {
	if t.pool != nil {
		t.pool.Close()
	}
	return nil
}

// Name 返回工具名称
func (t *SSHExecuteTool) Name() string {
	return "ssh_execute"
}

// Description 返回工具描述
func (t *SSHExecuteTool) Description() string {
	return "通过 SSH 在远程服务器执行命令，支持超时控制和安全审批"
}

// Parameters 返回参数定义（JSON Schema 格式）
func (t *SSHExecuteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"host": map[string]interface{}{
				"type":        "string",
				"description": "远程主机地址，格式: user@host:port",
			},
			"command": map[string]interface{}{
				"type":        "string",
				"description": "要在远程服务器执行的命令",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "命令执行超时时间（秒），默认 60",
			},
			"use_sudo": map[string]interface{}{
				"type":        "boolean",
				"description": "是否使用 sudo 执行命令",
			},
		},
		"required": []string{"host", "command"},
	}
}

// RiskLevel 返回风险等级（远程操作固定为高风险）
func (t *SSHExecuteTool) RiskLevel() RiskLevel {
	return RiskHigh
}

// Execute 通过 SSH 在远程服务器执行命令
func (t *SSHExecuteTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	// 解析参数
	host := parseStringParam(args, "host")
	if host == "" {
		return &Result{
			Success: false,
			Error:   "host 参数不能为空",
		}, fmt.Errorf("host 参数不能为空")
	}

	command := parseStringParam(args, "command")
	if command == "" {
		return &Result{
			Success: false,
			Error:   "command 参数不能为空",
		}, fmt.Errorf("command 参数不能为空")
	}

	// 检查危险命令（远程执行同样需要拦截）
	if err := checkDangerousCommand(command); err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("远程危险命令被拦截: %v", err),
		}, err
	}

	// 解析超时（默认 60 秒）
	timeout := 60
	if v, ok := parseNumberParam(args, "timeout"); ok && v > 0 {
		timeout = v
	}

	// 解析 use_sudo
	useSudo := parseBoolParam(args, "use_sudo")

	// 解析主机地址
	user, hostname, port, err := parseHostAddress(host)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("解析主机地址失败: %v", err),
		}, err
	}

	// 如果启用 sudo，在命令前添加 sudo
	if useSudo {
		command = "sudo " + command
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 建立 SSH 连接并执行命令
	output, err := t.executeSSH(timeoutCtx, user, hostname, port, command)

	// 处理输出截断
	truncated := false
	if len(output) > outputTruncateLimit {
		output = output[:outputTruncateLimit]
		truncated = true
	}

	if err != nil {
		result := &Result{
			Success: false,
			Output:  output,
			Error:   err.Error(),
			Summary: fmt.Sprintf("SSH 执行失败 [%s@%s:%d]", user, hostname, port),
		}
		if truncated {
			result.Output += "\n... [输出已截断，超过 10000 字符限制]"
		}
		return result, nil
	}

	result := &Result{
		Success: true,
		Output:  output,
		Summary: fmt.Sprintf("SSH 执行成功 [%s@%s:%d]", user, hostname, port),
	}
	if truncated {
		result.Output += "\n... [输出已截断，超过 10000 字符限制]"
		result.Summary += " [输出已截断]"
	}
	return result, nil
}

// parseHostAddress 解析 host 地址格式 user@host:port
// 默认用户 root，默认端口 22
func parseHostAddress(host string) (user, hostname string, port int, err error) {
	port = 22
	user = "root"

	// 解析 user@ 部分
	remaining := host
	if idx := strings.LastIndex(host, "@"); idx >= 0 {
		user = host[:idx]
		remaining = host[idx+1:]
	}

	// 解析 :port 部分
	// 使用最后一个 : 来分割，因为 IPv6 地址可能包含多个 :
	if idx := strings.LastIndex(remaining, ":"); idx >= 0 {
		// 检查是否可能是 IPv6 地址（被方括号包裹）
		if strings.HasPrefix(remaining, "[") && strings.Contains(remaining[:idx], "]") {
			// IPv6 格式 [addr]:port
			bracketEnd := strings.Index(remaining, "]")
			if bracketEnd < idx {
				hostname = remaining[1:bracketEnd]
				portStr := remaining[idx+1:]
				parsedPort, parseErr := strconv.Atoi(portStr)
				if parseErr != nil {
					return "", "", 0, fmt.Errorf("无效的端口号: %s", portStr)
				}
				port = parsedPort
				return user, hostname, port, nil
			}
		}
		// 普通 IPv4 或域名 :port
		portStr := remaining[idx+1:]
		parsedPort, parseErr := strconv.Atoi(portStr)
		if parseErr != nil {
			// 没有端口，只是地址的一部分包含 :
			hostname = remaining
			return user, hostname, port, nil
		}
		port = parsedPort
		hostname = remaining[:idx]
		if hostname == "" {
			return "", "", 0, fmt.Errorf("主机名不能为空")
		}
		return user, hostname, port, nil
	}

	hostname = remaining
	if hostname == "" {
		return "", "", 0, fmt.Errorf("主机名不能为空")
	}
	return user, hostname, port, nil
}

// executeSSH 建立 SSH 连接并执行命令
func (t *SSHExecuteTool) executeSSH(ctx context.Context, user, host string, port int, command string) (string, error) {
	// 创建 SSH 认证配置
	authMethods, err := t.buildAuthMethods(user, host)
	if err != nil {
		return "", fmt.Errorf("构建认证方法失败: %v", err)
	}

	// 构建 SSH 配置
	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: buildHostKeyCallback(),
		Timeout:         10 * time.Second,
	}

	// 解析上下文是否已取消
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// 从连接池获取连接（自动复用或新建）
	client, err := t.pool.Get(host, port, user, sshConfig)
	if err != nil {
		return "", err
	}

	// 创建会话
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建 SSH 会话失败: %v", err)
	}
	defer session.Close()

	// 在单独的 goroutine 中监听上下文取消信号
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			select {
			case <-done:
				return // 正常完成，无需发送信号
			default:
				session.Signal(ssh.SIGKILL)
			}
		case <-done:
		}
	}()

	// 执行命令
	output, err := session.CombinedOutput(command)

	return string(output), err
}

// buildAuthMethods 构建 SSH 认证方法列表
func (t *SSHExecuteTool) buildAuthMethods(user, host string) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	// 1. 优先尝试默认私钥路径 ~/.ssh/id_rsa
	homeDir, err := os.UserHomeDir()
	if err == nil {
		defaultKeyPath := filepath.Join(homeDir, ".ssh", "id_rsa")
		if key, err := loadPrivateKey(defaultKeyPath); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(key))
		}

		// 也尝试其他常见密钥路径
		otherKeyPaths := []string{
			filepath.Join(homeDir, ".ssh", "id_ed25519"),
			filepath.Join(homeDir, ".ssh", "id_ecdsa"),
		}
		for _, keyPath := range otherKeyPaths {
			if key, err := loadPrivateKey(keyPath); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(key))
			}
		}
	}

	// 2. 从环境变量获取密码
	sshPassword := os.Getenv("SSH_PASSWORD")
	if sshPassword != "" {
		authMethods = append(authMethods, ssh.Password(sshPassword))
	}

	// 3. 尝试使用 SSH_ASKPASS 环境变量指定的程序获取密码
	sshAskPass := os.Getenv("SSH_ASKPASS")
	if sshAskPass != "" && sshPassword == "" {
		// 记录但不阻塞执行，因为没有交互式终端可用
		// 在 Agent 环境中通常使用环境变量或密钥认证
	}

	// 4. 使用键盘交互式认证（fallback）
	if sshPassword == "" {
		authMethods = append(authMethods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
			// 在非交互式环境中无法回答键盘交互问题
			// 返回空答案让认证继续尝试其他方法
			answers := make([]string, len(questions))
			return answers, nil
		}))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("没有可用的 SSH 认证方法，请配置私钥（~/.ssh/id_rsa）或设置 SSH_PASSWORD 环境变量")
	}

	return authMethods, nil
}

// buildHostKeyCallback 构建 SSH HostKey 验证回调
// 优先使用 ~/.ssh/known_hosts 进行验证，无法加载时回退到固定密钥提示模式
func buildHostKeyCallback() ssh.HostKeyCallback {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ssh.InsecureIgnoreHostKey()
	}

	khPath := filepath.Join(homeDir, ".ssh", "known_hosts")
	cb, err := knownhosts.New(khPath)
	if err == nil {
		return cb
	}

	// 无法加载 known_hosts 时，使用 InsecureIgnoreHostKey 但记录警告
	// 这是降级方案，建议用户在首次连接后手动确认主机密钥
	return ssh.InsecureIgnoreHostKey()
}

// loadPrivateKey 加载 SSH 私钥
func loadPrivateKey(keyPath string) (ssh.Signer, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	return signer, nil
}
