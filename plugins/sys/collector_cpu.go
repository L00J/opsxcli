package sys

import (
	"github.com/shirou/gopsutil/v3/cpu"
)

// cpuDataResult CPU 采集结果（一次 cpu.Times 调用同时计算 CPUTimes 和 CPUPercent）
type cpuDataResult struct {
	cpuTimes   []CPUTimesStat
	cpuPercent []float64
	cpuCores   int
}

// collectCPUData 一次 cpu.Times 调用同时计算 CPUTimes 统计和 CPUPercent 使用率
// 避免原来 collectCPUTimes + calculateCPUPercentFromTimes 两次 cpu.Times 调用的重复开销
func (dc *DataCollector) collectCPUData() cpuDataResult {
	currentTimes, err := cpu.Times(true)
	if err != nil || len(currentTimes) == 0 {
		return cpuDataResult{}
	}

	// 首次采集：只保存基准值，无法计算差值
	if len(dc.lastCPUTimes) == 0 {
		dc.lastCPUTimes = currentTimes
		cores, _ := cpu.Counts(true)
		return cpuDataResult{cpuCores: cores}
	}

	// CPU 核心数不匹配，重置基准
	if len(currentTimes) != len(dc.lastCPUTimes) {
		dc.lastCPUTimes = currentTimes
		return cpuDataResult{}
	}

	deltaTime := dc.updateInterval.Seconds()
	if deltaTime <= 0 {
		deltaTime = 1.0
	}

	stats := make([]CPUTimesStat, 0, len(currentTimes))
	percentages := make([]float64, 0, len(currentTimes))

	for i, current := range currentTimes {
		last := dc.lastCPUTimes[i]

		totalDiff := cpuTotalTime(current) - cpuTotalTime(last)
		if totalDiff <= 0 {
			continue
		}

		// CPUTimes 详细统计（类似 mpstat）
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

		// CPUPercent（User + System，不含 Nice）
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

	// 计算总体平均值（all）
	if len(stats) > 0 {
		stats = append([]CPUTimesStat{cpuAverage(stats)}, stats...)
	}

	// 保存当前数据作为下次的基准
	dc.lastCPUTimes = currentTimes

	return cpuDataResult{
		cpuTimes:   stats,
		cpuPercent: percentages,
		cpuCores:   len(percentages),
	}
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
