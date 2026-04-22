package alert

import (
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/process"
)

// getProcessCount 获取当前进程数量。
func getProcessCount() (int, error) {
	pids, err := process.Pids()
	if err != nil {
		return 0, err
	}
	return len(pids), nil
}

// getUptimeSeconds 获取系统运行时间（秒）。
func getUptimeSeconds() (float64, error) {
	uptime, err := host.Uptime()
	if err != nil {
		return 0, err
	}
	return float64(uptime), nil
}
