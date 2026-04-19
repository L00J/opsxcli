package cmd

import "github.com/spf13/cobra"

func init() {
	RegisterCommand("date", "工具", "显示或设置日期时间", NewDateCmd)
	RegisterCommand("sleep", "工具", "延迟指定时间", NewSleepCmd)
}

// === 时间日期 ===

func NewDateCmd() *cobra.Command {
	return createForwardCmd("date", "显示或设置日期时间", "显示或设置系统日期和时间")
}

func NewSleepCmd() *cobra.Command {
	return createForwardCmd("sleep", "延迟指定时间", "暂停指定的秒数")
}
