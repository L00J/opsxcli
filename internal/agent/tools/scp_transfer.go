package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SCPTransferTool SCP 文件传输工具
type SCPTransferTool struct{}

// NewSCPTransferTool 创建 SCP 传输工具
func NewSCPTransferTool() *SCPTransferTool {
	return &SCPTransferTool{}
}

// Name 返回工具名称
func (t *SCPTransferTool) Name() string {
	return "scp_transfer"
}

// Description 返回工具描述
func (t *SCPTransferTool) Description() string {
	return "在本地和远程服务器之间传输文件，支持上传和下载"
}

// Parameters 返回参数定义（JSON Schema 格式）
func (t *SCPTransferTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"source": map[string]interface{}{
				"type":        "string",
				"description": "源文件路径（本地或远程，取决于 direction）",
			},
			"destination": map[string]interface{}{
				"type":        "string",
				"description": "目标文件路径（本地或远程，取决于 direction）",
			},
			"direction": map[string]interface{}{
				"type":        "string",
				"description": "传输方向: upload（本地->远程）或 download（远程->本地）",
				"enum":        []string{"upload", "download"},
			},
			"host": map[string]interface{}{
				"type":        "string",
				"description": "远程主机地址，格式: user@host:port（远程传输时必填）",
			},
		},
		"required": []string{"source", "destination", "direction", "host"},
	}
}

// RiskLevel 返回风险等级
func (t *SCPTransferTool) RiskLevel() RiskLevel {
	return RiskMedium
}

// Execute 执行文件传输
func (t *SCPTransferTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	// 解析参数
	source := parseStringParam(args, "source")
	if source == "" {
		return &Result{
			Success: false,
			Error:   "source 参数不能为空",
		}, fmt.Errorf("source 参数不能为空")
	}

	destination := parseStringParam(args, "destination")
	if destination == "" {
		return &Result{
			Success: false,
			Error:   "destination 参数不能为空",
		}, fmt.Errorf("destination 参数不能为空")
	}

	direction := parseStringParam(args, "direction")
	if direction != "upload" && direction != "download" {
		return &Result{
			Success: false,
			Error:   "direction 参数必须是 upload 或 download",
		}, fmt.Errorf("direction 参数必须是 upload 或 download")
	}

	host := parseStringParam(args, "host")
	if host == "" {
		return &Result{
			Success: false,
			Error:   "host 参数不能为空",
		}, fmt.Errorf("host 参数不能为空")
	}

	// 解析主机地址
	user, hostname, port, err := parseHostAddress(host)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("解析主机地址失败: %v", err),
		}, err
	}

	// 建立 SSH 连接
	sshClient, err := connectSSH(user, hostname, port)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("SSH 连接失败: %v", err),
		}, err
	}
	defer sshClient.Close()

	// 创建 SFTP 客户端
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("创建 SFTP 客户端失败: %v", err),
		}, err
	}
	defer sftpClient.Close()

	var result *Result

	// 根据方向执行传输
	switch direction {
	case "upload":
		result, err = t.uploadFile(ctx, sftpClient, source, destination)
	case "download":
		result, err = t.downloadFile(ctx, sftpClient, source, destination)
	}

	return result, err
}

// uploadFile 上传文件（本地 -> 远程）
func (t *SCPTransferTool) uploadFile(ctx context.Context, client *sftp.Client, localPath, remotePath string) (*Result, error) {
	// 检查上下文
	if ctx.Err() != nil {
		return &Result{
			Success: false,
			Error:   "操作已取消",
		}, ctx.Err()
	}

	// 检查本地路径是否存在
	localInfo, err := os.Stat(localPath)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("无法访问本地路径: %v", err),
		}, err
	}

	// 处理目录上传
	if localInfo.IsDir() {
		return t.uploadDirectory(ctx, client, localPath, remotePath)
	}

	// 确保远程目录存在
	remoteDir := filepath.ToSlash(filepath.Dir(remotePath))
	if err := client.MkdirAll(remoteDir); err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("创建远程目录失败: %v", err),
		}, err
	}

	// 打开本地文件
	srcFile, err := os.Open(localPath)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("打开本地文件失败: %v", err),
		}, err
	}
	defer srcFile.Close()

	// 创建远程文件
	dstFile, err := client.Create(remotePath)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("创建远程文件失败: %v", err),
		}, err
	}
	defer dstFile.Close()

	// 复制文件内容
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("传输文件失败: %v", err),
		}, err
	}

	return &Result{
		Success: true,
		Output:  fmt.Sprintf("上传成功: %s -> %s (%d 字节)", localPath, remotePath, written),
		Summary: fmt.Sprintf("文件上传完成 [%d 字节]", written),
	}, nil
}

// uploadDirectory 上传目录（本地 -> 远程）
func (t *SCPTransferTool) uploadDirectory(ctx context.Context, client *sftp.Client, localDir, remoteDir string) (*Result, error) {
	var uploadedFiles int
	var totalBytes int64

	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		remotePath := filepath.ToSlash(filepath.Join(remoteDir, relPath))

		if info.IsDir() {
			return client.MkdirAll(remotePath)
		}

		// 打开本地文件
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		// 创建远程文件
		dstFile, err := client.Create(remotePath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		written, err := io.Copy(dstFile, srcFile)
		if err != nil {
			return err
		}

		uploadedFiles++
		totalBytes += written
		return nil
	})

	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("上传目录失败: %v", err),
		}, err
	}

	return &Result{
		Success: true,
		Output:  fmt.Sprintf("目录上传成功: %s -> %s (文件数: %d, 总字节: %d)", localDir, remoteDir, uploadedFiles, totalBytes),
		Summary: fmt.Sprintf("目录上传完成 [文件数: %d, 总字节: %d]", uploadedFiles, totalBytes),
	}, nil
}

// downloadFile 下载文件（远程 -> 本地）
func (t *SCPTransferTool) downloadFile(ctx context.Context, client *sftp.Client, remotePath, localPath string) (*Result, error) {
	// 检查上下文
	if ctx.Err() != nil {
		return &Result{
			Success: false,
			Error:   "操作已取消",
		}, ctx.Err()
	}

	// 检查远程路径是否存在
	remoteInfo, err := client.Stat(remotePath)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("无法访问远程路径: %v", err),
		}, err
	}

	// 处理目录下载
	if remoteInfo.IsDir() {
		return t.downloadDirectory(ctx, client, remotePath, localPath)
	}

	// 确保本地目录存在
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("创建本地目录失败: %v", err),
		}, err
	}

	// 打开远程文件
	srcFile, err := client.Open(remotePath)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("打开远程文件失败: %v", err),
		}, err
	}
	defer srcFile.Close()

	// 创建本地文件
	dstFile, err := os.Create(localPath)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("创建本地文件失败: %v", err),
		}, err
	}
	defer dstFile.Close()

	// 复制文件内容
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("下载文件失败: %v", err),
		}, err
	}

	return &Result{
		Success: true,
		Output:  fmt.Sprintf("下载成功: %s -> %s (%d 字节)", remotePath, localPath, written),
		Summary: fmt.Sprintf("文件下载完成 [%d 字节]", written),
	}, nil
}

// downloadDirectory 下载目录（远程 -> 本地）
func (t *SCPTransferTool) downloadDirectory(ctx context.Context, client *sftp.Client, remoteDir, localDir string) (*Result, error) {
	var downloadedFiles int
	var totalBytes int64

	// 确保本地目录存在
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("创建本地目录失败: %v", err),
		}, err
	}

	// 遍历远程目录
	walker := client.Walk(remoteDir)
	for walker.Step() {
		if ctx.Err() != nil {
			return &Result{
				Success: false,
				Error:   "操作已取消",
			}, ctx.Err()
		}

		if err := walker.Err(); err != nil {
			return &Result{
				Success: false,
				Error:   fmt.Sprintf("遍历远程目录失败: %v", err),
			}, err
		}

		remotePath := walker.Path()
		relPath, err := filepath.Rel(remoteDir, remotePath)
		if err != nil {
			return &Result{
				Success: false,
				Error:   fmt.Sprintf("计算相对路径失败: %v", err),
			}, err
		}
		localPath := filepath.Join(localDir, relPath)

		if walker.Stat().IsDir() {
			if err := os.MkdirAll(localPath, 0755); err != nil {
				return &Result{
					Success: false,
					Error:   fmt.Sprintf("创建本地目录失败: %v", err),
				}, err
			}
			continue
		}

		// 打开远程文件
		srcFile, err := client.Open(remotePath)
		if err != nil {
			return &Result{
				Success: false,
				Error:   fmt.Sprintf("打开远程文件失败: %v", err),
			}, err
		}

		// 创建本地文件
		dstFile, err := os.Create(localPath)
		if err != nil {
			srcFile.Close()
			return &Result{
				Success: false,
				Error:   fmt.Sprintf("创建本地文件失败: %v", err),
			}, err
		}

		written, err := io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()

		if err != nil {
			return &Result{
				Success: false,
				Error:   fmt.Sprintf("下载文件失败: %v", err),
			}, err
		}

		downloadedFiles++
		totalBytes += written
	}

	return &Result{
		Success: true,
		Output:  fmt.Sprintf("目录下载成功: %s -> %s (文件数: %d, 总字节: %d)", remoteDir, localDir, downloadedFiles, totalBytes),
		Summary: fmt.Sprintf("目录下载完成 [文件数: %d, 总字节: %d]", downloadedFiles, totalBytes),
	}, nil
}

// connectSSH 建立 SSH 连接（复用 ssh_execute 的认证逻辑）
func connectSSH(user, host string, port int) (*ssh.Client, error) {
	// 创建 SSH 认证配置
	authMethods, err := buildSSHAuthMethods(user, host)
	if err != nil {
		return nil, fmt.Errorf("构建认证方法失败: %v", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshTimeoutDuration,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	return ssh.Dial("tcp", addr, sshConfig)
}

// sshTimeoutDuration SSH 连接超时时间
const sshTimeoutDuration = 10 * time.Second

// buildSSHAuthMethods 构建 SSH 认证方法列表（与 ssh_execute 共享认证逻辑）
func buildSSHAuthMethods(user, host string) ([]ssh.AuthMethod, error) {
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

	// 3. 使用键盘交互式认证（fallback）
	if sshPassword == "" {
		authMethods = append(authMethods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
			answers := make([]string, len(questions))
			return answers, nil
		}))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("没有可用的 SSH 认证方法，请配置私钥（~/.ssh/id_rsa）或设置 SSH_PASSWORD 环境变量")
	}

	return authMethods, nil
}
