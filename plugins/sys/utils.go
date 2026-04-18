package sys

import (
	"fmt"
	"time"
)

// formatBytes 格式化字节数(使用二进制单位 KiB, MiB, GiB)
func formatBytes(bytes float64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%.0fB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	// 使用二进制单位标识(KiB, MiB, GiB等)更准确
	return fmt.Sprintf("%.1f%ciB", bytes/float64(div), "KMGTPE"[exp])
}

// formatNumber 格式化数字
func formatNumber(n uint64) string {
	return fmt.Sprintf("%d", n)
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// formatCPUTime 格式化 CPU 时间
func formatCPUTime(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.2fs", seconds)
	}
	minutes := int(seconds / 60)
	secs := int(seconds) % 60
	if minutes < 60 {
		return fmt.Sprintf("%dm %02ds", minutes, secs)
	}
	hours := minutes / 60
	mins := minutes % 60
	return fmt.Sprintf("%dh %02dm", hours, mins)
}

// formatDiskIO 格式化磁盘 I/O
func formatDiskIO(readRate, writeRate float64) string {
	if readRate == 0 && writeRate == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%s/%s", formatBytes(readRate), formatBytes(writeRate))
}

// formatRuntime 格式化运行时间
func formatRuntime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

// formatBytesShort 格式化字节数（htop 风格，更紧凑，使用二进制单位）
func formatBytesShort(bytes float64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%.0fB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	// htop 风格：使用单字母单位（不显示小数点，如果小于10则显示）
	value := bytes / float64(div)
	if value < 10 && exp > 0 {
		return fmt.Sprintf("%.1f%c", value, "KMGTPE"[exp])
	}
	return fmt.Sprintf("%.0f%c", value, "KMGTPE"[exp])
}
