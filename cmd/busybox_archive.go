package cmd

import "github.com/spf13/cobra"

func init() {
	RegisterCommand("tar", "工具", "归档工具", NewTarCmd)
	RegisterCommand("gzip", "工具", "压缩文件", NewGzipCmd)
	RegisterCommand("unzip", "工具", "解压 ZIP 文件", NewUnzipCmd)
}

// === 归档压缩 ===

func NewTarCmd() *cobra.Command {
	return createForwardCmd("tar", "归档工具", "创建、提取或列出 tar 归档文件")
}

func NewGzipCmd() *cobra.Command {
	return createForwardCmd("gzip", "压缩文件", "使用 gzip 压缩文件")
}

func NewUnzipCmd() *cobra.Command {
	return createForwardCmd("unzip", "解压 ZIP 文件", "解压 ZIP 格式的压缩文件")
}
