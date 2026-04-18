package cmd

import "github.com/spf13/cobra"

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

func NewMountCmd() *cobra.Command {
	return createForwardCmd("mount", "挂载文件系统", "挂载文件系统到指定挂载点")
}

func NewUmountCmd() *cobra.Command {
	return createForwardCmd("umount", "卸载文件系统", "卸载已挂载的文件系统")
}
