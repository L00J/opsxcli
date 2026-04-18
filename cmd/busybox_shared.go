package cmd

import (
	"github.com/spf13/cobra"
	"opsxcli/internal/exec"
)

// createForwardCmd 创建一个转发到系统命令的 cobra 命令
func createForwardCmd(name, short, long string) *cobra.Command {
	return &cobra.Command{
		Use:                name,
		Short:              short,
		Long:               long,
		DisableFlagParsing: true, // 不解析参数，直接转发
		RunE: func(cmd *cobra.Command, args []string) error {
			return exec.ForwardCommand(name, args)
		},
	}
}
