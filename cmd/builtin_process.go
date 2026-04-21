package cmd

import (
	"fmt"
	"strconv"

	"opsxcli/plugins/builtin"
	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("ps", "系统", "显示进程列表", NewPsCmd)
	RegisterCommand("pstree", "系统", "以树形结构显示进程", NewPstreeCmd)
	RegisterCommand("top", "系统", "显示系统进程(简化版)", NewTopCmd)
}

func NewPsCmd() *cobra.Command {
	var full bool
	var showAll bool

	cmd := &cobra.Command{
		Use:   "ps",
		Short: "显示进程列表",
		Long:  "显示系统进程列表，类似 Linux ps 命令",
		RunE: func(cmd *cobra.Command, args []string) error {
			return builtin.Ps(builtin.PsOptions{
				Full:    full,
				ShowAll: showAll,
			})
		},
	}

	cmd.Flags().BoolVarP(&full, "full", "f", false, "完整格式显示 (UID, PID, PPID)")
	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "显示所有进程")
	cmd.Flags().BoolVarP(&showAll, "A", "A", false, "显示所有进程 (同 -a)")
	_ = cmd.Flags().MarkHidden("A")

	return cmd
}

func NewPstreeCmd() *cobra.Command {
	var pid int
	var showPID bool
	var fullCmd bool

	cmd := &cobra.Command{
		Use:   "pstree [PID]",
		Short: "以树形结构显示进程",
		Long:  "以树形结构显示进程父子关系",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				p, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("无效的 PID: %s", args[0])
				}
				pid = p
			}
			return builtin.Pstree(builtin.PstreeOptions{
				PID:     pid,
				ShowPID: showPID,
				FullCmd: fullCmd,
			})
		},
	}

	cmd.Flags().IntVarP(&pid, "pid", "p", 0, "指定根进程 PID")
	cmd.Flags().BoolVar(&showPID, "show-pid", false, "显示进程 PID")
	cmd.Flags().BoolVarP(&fullCmd, "args", "a", false, "显示完整命令行")

	return cmd
}

func NewTopCmd() *cobra.Command {
	var delay int
	var count int

	cmd := &cobra.Command{
		Use:   "top",
		Short: "显示系统进程(简化版)",
		Long:  "显示系统进程快照，按 CPU 排序。完整监控请使用 opsxcli sys",
		RunE: func(cmd *cobra.Command, args []string) error {
			return builtin.Top(builtin.TopOptions{
				Delay: delay,
				Count: count,
			})
		},
	}

	cmd.Flags().IntVarP(&delay, "delay", "d", 3, "刷新间隔(秒)")
	cmd.Flags().IntVarP(&count, "number", "n", 0, "刷新次数 (0=无限)")

	return cmd
}
