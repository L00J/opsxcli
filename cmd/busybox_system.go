package cmd

import "github.com/spf13/cobra"

func init() {
	RegisterCommand("uname", "系统", "显示系统信息", NewUnameCmd)
	RegisterCommand("hostname", "系统", "显示或设置主机名", NewHostnameCmd)
	RegisterCommand("whoami", "系统", "显示当前用户", NewWhoamiCmd)
	RegisterCommand("id", "系统", "显示用户和组信息", NewIdCmd)
	RegisterCommand("free", "系统", "显示内存使用情况", NewFreeCmd)
	RegisterCommand("df", "系统", "显示磁盘空间", NewDfCmd)
	RegisterCommand("du", "系统", "显示目录大小", NewDuCmd)
}

// === 系统信息 ===

func NewUnameCmd() *cobra.Command {
	return createForwardCmd("uname", "显示系统信息", "显示系统名称和版本信息")
}

func NewHostnameCmd() *cobra.Command {
	return createForwardCmd("hostname", "显示或设置主机名", "显示或设置系统的主机名")
}

func NewWhoamiCmd() *cobra.Command {
	return createForwardCmd("whoami", "显示当前用户", "显示当前有效的用户名")
}

func NewIdCmd() *cobra.Command {
	return createForwardCmd("id", "显示用户和组信息", "显示用户的 UID、GID 和所属组")
}

func NewFreeCmd() *cobra.Command {
	return createForwardCmd("free", "显示内存使用情况", "显示系统内存和交换空间使用情况")
}

func NewDfCmd() *cobra.Command {
	return createForwardCmd("df", "显示磁盘空间", "显示文件系统的磁盘空间使用情况")
}

func NewDuCmd() *cobra.Command {
	return createForwardCmd("du", "显示目录大小", "显示目录或文件的磁盘使用情况")
}
