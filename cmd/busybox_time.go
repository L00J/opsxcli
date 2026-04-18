package cmd

import "github.com/spf13/cobra"

// === 时间日期 ===

func NewDateCmd() *cobra.Command {
	return createForwardCmd("date", "显示或设置日期时间", "显示或设置系统日期和时间")
}

func NewSleepCmd() *cobra.Command {
	return createForwardCmd("sleep", "延迟指定时间", "暂停指定的秒数")
}

func NewWatchCmd() *cobra.Command {
	return createForwardCmd("watch", "周期性执行命令", "定期执行命令并显示输出")
}
