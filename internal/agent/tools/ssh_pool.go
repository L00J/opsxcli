package tools

import (
	"fmt"
	"sync"

	"golang.org/x/crypto/ssh"
)

// SSHConnKey 连接池键，用于唯一标识一个 SSH 连接
// 基于主机地址、端口和用户名，同一组凭据共享一个连接
type SSHConnKey struct {
	Host string
	Port int
	User string
}

// SSHPool SSH 持久连接池
// 在 Agent 生命周期内保持 SSH 连接复用，避免每次工具调用都重新建立连接（1-3 秒延迟）
type SSHPool struct {
	mu    sync.RWMutex
	conns map[SSHConnKey]*ssh.Client
}

// NewSSHPool 创建新的 SSH 连接池
func NewSSHPool() *SSHPool {
	return &SSHPool{
		conns: make(map[SSHConnKey]*ssh.Client),
	}
}

// Get 从连接池获取 SSH 连接
// 如果连接池中存在有效连接，直接返回；否则新建连接并放入池中
func (p *SSHPool) Get(host string, port int, user string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	key := SSHConnKey{Host: host, Port: port, User: user}

	// 先读锁检查，避免每次调用都竞争写锁
	p.mu.RLock()
	client, exists := p.conns[key]
	p.mu.RUnlock()

	if exists && isSSHConnAlive(client) {
		return client, nil
	}

	// 需要新建连接，获取写锁
	p.mu.Lock()
	defer p.mu.Unlock()

	// 双重检查：其他 goroutine 可能已在此间隙中创建连接
	if client, exists := p.conns[key]; exists && isSSHConnAlive(client) {
		return client, nil
	}

	// 关闭已失效的旧连接
	if exists && client != nil {
		client.Close()
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}

	p.conns[key] = client
	return client, nil
}

// isSSHConnAlive 检查 SSH 连接是否仍然有效
// 通过尝试创建临时会话验证连接状态，创建成功后立即关闭
func isSSHConnAlive(client *ssh.Client) bool {
	if client == nil {
		return false
	}
	session, err := client.NewSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
}

// Close 关闭连接池中所有 SSH 连接并清空池
// 应在 Agent 退出时调用，确保资源释放
func (p *SSHPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for key, client := range p.conns {
		if client != nil {
			client.Close()
		}
		delete(p.conns, key)
	}
}
