package client

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"

	"opsxcli/internal/logger"
)

// SFTPClient SFTP客户端封装
type SFTPClient struct {
	client *sftp.Client
}

// NewSFTPClient 创建SFTP客户端
func NewSFTPClient(sshClient *SSHClient) (*SFTPClient, error) {
	client, err := sftp.NewClient(sshClient.client)
	if err != nil {
		return nil, err
	}

	return &SFTPClient{client: client}, nil
}

// Close 关闭SFTP连接
func (c *SFTPClient) Close() error {
	return c.client.Close()
}

// Upload 上传文件或目录
func (c *SFTPClient) Upload(local, remote string, recursive bool) error {
	info, err := os.Stat(local)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if !recursive {
			return fmt.Errorf("%s 是目录，请使用 -r 参数", local)
		}
		return c.uploadDir(local, remote)
	}

	return c.uploadFile(local, remote)
}

// Download 下载文件或目录
func (c *SFTPClient) Download(remote, local string, recursive bool) error {
	info, err := c.client.Stat(remote)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if !recursive {
			return fmt.Errorf("%s 是目录，请使用 -r 参数", remote)
		}
		return c.downloadDir(remote, local)
	}

	return c.downloadFile(remote, local)
}

// uploadFile 上传单个文件
func (c *SFTPClient) uploadFile(local, remote string) error {
	// 确保远程目录存在
	remoteDir := filepath.Dir(remote)
	if err := c.client.MkdirAll(remoteDir); err != nil {
		return err
	}

	// 打开本地文件
	srcFile, err := os.Open(local)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 创建远程文件
	dstFile, err := c.client.Create(remote)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 复制文件
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	logger.Info("上传: %s -> %s", local, remote)
	return nil
}

// uploadDir 上传目录
func (c *SFTPClient) uploadDir(local, remote string) error {
	return filepath.Walk(local, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(local, path)
		if err != nil {
			return err
		}

		remotePath := filepath.Join(remote, relPath)

		if info.IsDir() {
			return c.client.MkdirAll(remotePath)
		}

		return c.uploadFile(path, remotePath)
	})
}

// downloadFile 下载单个文件
func (c *SFTPClient) downloadFile(remote, local string) error {
	// 确保本地目录存在
	localDir := filepath.Dir(local)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return err
	}

	// 打开远程文件
	srcFile, err := c.client.Open(remote)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 创建本地文件
	dstFile, err := os.Create(local)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 复制文件
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	logger.Info("下载: %s -> %s", remote, local)
	return nil
}

// downloadDir 下载目录
func (c *SFTPClient) downloadDir(remote, local string) error {
	walker := c.client.Walk(remote)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}

		remotePath := walker.Path()
		relPath, err := filepath.Rel(remote, remotePath)
		if err != nil {
			return err
		}

		localPath := filepath.Join(local, relPath)

		if walker.Stat().IsDir() {
			if err := os.MkdirAll(localPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := c.downloadFile(remotePath, localPath); err != nil {
			return err
		}
	}

	return nil
}
