package cmd

import (
	"github.com/spf13/cobra"
	"opsxcli/internal/exec"
)

func init() {
	RegisterCommand("ps", "系统", "进程查看", NewPsCmd)
	RegisterCommand("top", "系统", "进程监控", NewTopCmd)
	RegisterCommand("kill", "系统", "终止进程", NewKillCmd)
	RegisterCommand("pstree", "系统", "进程树显示", NewPstreeCmd)
}

// === 进程管理 ===

func NewPsCmd() *cobra.Command {
	return createForwardCmd("ps", "显示进程信息", "显示当前运行的进程")
}

func NewTopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "top",
		Short: "实时系统监控",
		Long:  "实时显示系统进程信息（推荐使用 'opsxcli sys' 获得更好的体验）",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 优先使用 opsxcli sys，但保留 top 兼容性
			return exec.ForwardCommand("top", args)
		},
	}
}

func NewKillCmd() *cobra.Command {
	return createForwardCmd("kill", "终止进程", "发送信号到指定进程")
}

func NewPstreeCmd() *cobra.Command {
	return createForwardCmd("pstree", "显示进程树", "以树形结构显示进程关系")
}
