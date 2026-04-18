package cmd

import "github.com/spf13/cobra"

// === 网络传输 ===

func NewFtpCmd() *cobra.Command {
	return createForwardCmd("ftp", "FTP 客户端", "文件传输协议客户端")
}

func NewTftpCmd() *cobra.Command {
	return createForwardCmd("tftp", "TFTP 客户端", "简单文件传输协议客户端")
}
