package sys

import (
	"github.com/shirou/gopsutil/v3/cpu"
)

// collectCPUTimes 收集 CPU 详细时间统计（类似 mpstat）
func (dc *DataCollector) collectCPUTimes() []CPUTimesStat {
	currentTimes, err := cpu.Times(true)
	if err != nil {
		return []CPUTimesStat{}
	}

	if len(dc.lastCPUTimes) == 0 {
		dc.lastCPUTimes = currentTimes
		return []CPUTimesStat{}
	}

	deltaTime := dc.updateInterval.Seconds()
	if deltaTime <= 0 {
		deltaTime = 1.0
	}

	stats := make([]CPUTimesStat, 0, len(currentTimes)+1)

	// 计算每个 CPU 核心的统计
	for i, current := range currentTimes {
		if i >= len(dc.lastCPUTimes) {
			continue
		}
		last := dc.lastCPUTimes[i]

		totalDiff := cpuTotalTime(current) - cpuTotalTime(last)
		if totalDiff <= 0 {
			continue
		}

		stat := CPUTimesStat{
			CPU:       current.CPU,
			User:      ((current.User - last.User) / totalDiff) * 100.0,
			Nice:      ((current.Nice - last.Nice) / totalDiff) * 100.0,
			System:    ((current.System - last.System) / totalDiff) * 100.0,
			Idle:      ((current.Idle - last.Idle) / totalDiff) * 100.0,
			Iowait:    ((current.Iowait - last.Iowait) / totalDiff) * 100.0,
			Irq:       ((current.Irq - last.Irq) / totalDiff) * 100.0,
			Softirq:   ((current.Softirq - last.Softirq) / totalDiff) * 100.0,
			Steal:     ((current.Steal - last.Steal) / totalDiff) * 100.0,
			Guest:     ((current.Guest - last.Guest) / totalDiff) * 100.0,
			GuestNice: ((current.GuestNice - last.GuestNice) / totalDiff) * 100.0,
		}

		stats = append(stats, stat)
	}

	// 计算总体平均值（all）
	if len(stats) > 0 {
		stats = append([]CPUTimesStat{cpuAverage(stats)}, stats...)
	}

	// 保存当前数据作为下次的基准
	dc.lastCPUTimes = currentTimes

	return stats
}

// calculateCPUPercentFromTimes 基于 CPUTimes 计算 CPU 使用率（只计算 User + System，不包括 Nice）
func (dc *DataCollector) calculateCPUPercentFromTimes() []float64 {
	if len(dc.lastCPUTimes) == 0 {
		// 数据尚未就绪，返回空切片，等下一次 tick 自然有数据
		return nil
	}

	currentTimes, err := cpu.Times(true)
	if err != nil {
		return nil
	}

	if len(currentTimes) != len(dc.lastCPUTimes) {
		// CPU 核心数不匹配，等下一次 tick 自然对齐
		return nil
	}

	percentages := make([]float64, 0, len(currentTimes))

	for i, current := range currentTimes {
		if i >= len(dc.lastCPUTimes) {
			continue
		}
		last := dc.lastCPUTimes[i]

		totalDiff := cpuTotalTime(current) - cpuTotalTime(last)

		if totalDiff <= 0.5 {
			// 时间差太小（首次 tick），返回 0.0 等下次自然有数据，不再阻塞调用 cpu.Percent
			percentages = append(percentages, 0.0)
			continue
		}

		userDiff := current.User - last.User
		systemDiff := current.System - last.System
		usagePercent := ((userDiff + systemDiff) / totalDiff) * 100.0

		if usagePercent < 0 {
			usagePercent = 0
		}
		if usagePercent > 100 {
			usagePercent = 100
		}

		percentages = append(percentages, usagePercent)
	}

	return percentages
}

// cpuTotalTime 计算 CPU 时间统计的总时间
func cpuTotalTime(t cpu.TimesStat) float64 {
	return t.User + t.Nice + t.System + t.Idle + t.Iowait +
		t.Irq + t.Softirq + t.Steal + t.Guest + t.GuestNice
}

// cpuAverage 计算多个核心的平均值
func cpuAverage(stats []CPUTimesStat) CPUTimesStat {
	var allStat CPUTimesStat
	allStat.CPU = "all"
	count := float64(len(stats))
	for _, stat := range stats {
		allStat.User += stat.User
		allStat.Nice += stat.Nice
		allStat.System += stat.System
		allStat.Idle += stat.Idle
		allStat.Iowait += stat.Iowait
		allStat.Irq += stat.Irq
		allStat.Softirq += stat.Softirq
		allStat.Steal += stat.Steal
		allStat.Guest += stat.Guest
		allStat.GuestNice += stat.GuestNice
	}
	allStat.User /= count
	allStat.Nice /= count
	allStat.System /= count
	allStat.Idle /= count
	allStat.Iowait /= count
	allStat.Irq /= count
	allStat.Softirq /= count
	allStat.Steal /= count
	allStat.Guest /= count
	allStat.GuestNice /= count
	return allStat
}
