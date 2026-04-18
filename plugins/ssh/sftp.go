package ssh

import (
	"fmt"

	"github.com/spf13/cobra"

	"opsxcli/internal/logger"
	"opsxcli/plugins/ssh/client"
)

// NewPutCmd 创建上传命令
func NewPutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "put <local> [user@]host:<remote>",
		Short: "上传文件或目录",
		Long:  "通过SFTP上传本地文件或目录到远程主机",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			local := args[0]
			remote := args[1]
			keyPath, _ := cmd.Flags().GetString("key")
			port, _ := cmd.Flags().GetInt("port")
			password, _ := cmd.Flags().GetString("password")
			recursive, _ := cmd.Flags().GetBool("recursive")

			return putFile(local, remote, keyPath, port, password, recursive)
		},
	}

	cmd.Flags().StringP("key", "i", "", "SSH私钥路径")
	cmd.Flags().IntP("port", "p", 22, "SSH端口")
	cmd.Flags().StringP("password", "P", "", "密码")
	cmd.Flags().BoolP("recursive", "r", false, "递归上传目录")

	return cmd
}

// NewGetCmd 创建下载命令
func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [user@]host:<remote> <local>",
		Short: "下载文件或目录",
		Long:  "通过SFTP从远程主机下载文件或目录",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			remote := args[0]
			local := args[1]
			keyPath, _ := cmd.Flags().GetString("key")
			port, _ := cmd.Flags().GetInt("port")
			password, _ := cmd.Flags().GetString("password")
			recursive, _ := cmd.Flags().GetBool("recursive")

			return getFile(remote, local, keyPath, port, password, recursive)
		},
	}

	cmd.Flags().StringP("key", "i", "", "SSH私钥路径")
	cmd.Flags().IntP("port", "p", 22, "SSH端口")
	cmd.Flags().StringP("password", "P", "", "密码")
	cmd.Flags().BoolP("recursive", "r", false, "递归下载目录")

	return cmd
}

func putFile(local, remote, keyPath string, port int, password string, recursive bool) error {
	// 解析远程路径
	user, host, remotePath, err := parseRemotePath(remote)
	if err != nil {
		return err
	}

	// 创建SSH客户端
	sshClient, err := client.NewSSHClient(host, port, user, keyPath, password)
	if err != nil {
		logger.Error("SSH SFTP连接失败 %s@%s:%d: %v", user, host, port, err)
		return fmt.Errorf("连接失败: %v", err)
	}
	defer sshClient.Close()

	// 创建SFTP客户端
	sftpClient, err := client.NewSFTPClient(sshClient)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// 上传文件
	if err := sftpClient.Upload(local, remotePath, recursive); err != nil {
		return err
	}

	logger.Success("上传成功: %s -> %s@%s:%s", local, user, host, remotePath)
	return nil
}

func getFile(remote, local, keyPath string, port int, password string, recursive bool) error {
	// 解析远程路径
	user, host, remotePath, err := parseRemotePath(remote)
	if err != nil {
		return err
	}

	// 创建SSH客户端
	sshClient, err := client.NewSSHClient(host, port, user, keyPath, password)
	if err != nil {
		logger.Error("SSH SFTP连接失败 %s@%s:%d: %v", user, host, port, err)
		return fmt.Errorf("连接失败: %v", err)
	}
	defer sshClient.Close()

	// 创建SFTP客户端
	sftpClient, err := client.NewSFTPClient(sshClient)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// 下载文件
	if err := sftpClient.Download(remotePath, local, recursive); err != nil {
		return err
	}

	logger.Success("下载成功: %s@%s:%s -> %s", user, host, remotePath, local)
	return nil
}
