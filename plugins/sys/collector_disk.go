package sys

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/disk"
)

// collectDiskIO 收集磁盘 I/O 详细统计（类似 iostat）
func (dc *DataCollector) collectDiskIO() []DiskIOStat {
	currentIO, err := disk.IOCounters()
	if err != nil {
		return []DiskIOStat{}
	}

	deltaTime := dc.updateInterval.Seconds()
	if deltaTime <= 0 {
		deltaTime = 1.0
	}

	stats := make([]DiskIOStat, 0, len(currentIO))

	for name, current := range currentIO {
		stat := DiskIOStat{
			Name: name,
		}

		if last, ok := dc.lastDiskIO[name]; ok && last != nil {
			calcDiskIORates(&stat, current, *last, deltaTime)
		}

		stats = append(stats, stat)
	}

	// 保存当前数据作为下次的基准
	dc.lastDiskIO = make(map[string]*disk.IOCountersStat)
	for name, stat := range currentIO {
		statCopy := stat
		dc.lastDiskIO[name] = &statCopy
	}

	return stats
}

// calcDiskIORates 计算磁盘 I/O 速率指标
func calcDiskIORates(stat *DiskIOStat, current, last disk.IOCountersStat, deltaTime float64) {
	// IOPS
	stat.ReadIOPS = float64(current.ReadCount-last.ReadCount) / deltaTime
	stat.WriteIOPS = float64(current.WriteCount-last.WriteCount) / deltaTime

	// 吞吐量 (KB/s)
	stat.ReadKBps = float64(current.ReadBytes-last.ReadBytes) / deltaTime / 1024.0
	stat.WriteKBps = float64(current.WriteBytes-last.WriteBytes) / deltaTime / 1024.0

	// 平均请求大小 (KB)
	totalIOPS := stat.ReadIOPS + stat.WriteIOPS
	if totalIOPS > 0 {
		totalBytes := float64(current.ReadBytes-last.ReadBytes) + float64(current.WriteBytes-last.WriteBytes)
		stat.AvgRqSz = (totalBytes / deltaTime) / totalIOPS / 1024.0
	}

	// 平台分支处理
	if runtime.GOOS == "darwin" {
		calcDiskIOMacOS(stat, current, last, deltaTime)
	} else {
		calcDiskIOLinux(stat, current, last, deltaTime)
	}

	// 服务时间 (ms) - 估算
	if stat.ReadIOPS+stat.WriteIOPS > 0 {
		stat.Svctm = stat.Await * 0.8
	}
}

// calcDiskIOMacOS macOS 磁盘 I/O 计算（部分字段可能为0，需降级）
func calcDiskIOMacOS(stat *DiskIOStat, current, last disk.IOCountersStat, deltaTime float64) {
	if current.WeightedIO == 0 && last.WeightedIO == 0 {
		stat.AvgQuSz = 0
	} else {
		stat.AvgQuSz = float64(current.WeightedIO-last.WeightedIO) / deltaTime / 1000.0
	}

	if current.IoTime == 0 && last.IoTime == 0 {
		stat.Await = 0
		stat.UtilPercent = 0
	} else {
		if stat.ReadIOPS+stat.WriteIOPS > 0 {
			ioTime := float64(current.IoTime-last.IoTime) / deltaTime / 1000.0
			stat.Await = ioTime / (stat.ReadIOPS + stat.WriteIOPS)
		}
		stat.UtilPercent = float64(current.IoTime-last.IoTime) / deltaTime / 10.0
		if stat.UtilPercent > 100.0 {
			stat.UtilPercent = 100.0
		}
	}

	if current.ReadTime == 0 && last.ReadTime == 0 {
		stat.RAwait = 0
	} else if stat.ReadIOPS > 0 {
		readTime := float64(current.ReadTime-last.ReadTime) / deltaTime / 1000.0
		stat.RAwait = readTime / stat.ReadIOPS
	}

	if current.WriteTime == 0 && last.WriteTime == 0 {
		stat.WAwait = 0
	} else if stat.WriteIOPS > 0 {
		writeTime := float64(current.WriteTime-last.WriteTime) / deltaTime / 1000.0
		stat.WAwait = writeTime / stat.WriteIOPS
	}
}

// calcDiskIOLinux Linux 完整磁盘 I/O 计算
func calcDiskIOLinux(stat *DiskIOStat, current, last disk.IOCountersStat, deltaTime float64) {
	stat.AvgQuSz = float64(current.WeightedIO-last.WeightedIO) / deltaTime / 1000.0

	if stat.ReadIOPS+stat.WriteIOPS > 0 {
		ioTime := float64(current.IoTime-last.IoTime) / deltaTime / 1000.0
		stat.Await = ioTime / (stat.ReadIOPS + stat.WriteIOPS)
	}

	if stat.ReadIOPS > 0 {
		readTime := float64(current.ReadTime-last.ReadTime) / deltaTime / 1000.0
		stat.RAwait = readTime / stat.ReadIOPS
	}

	if stat.WriteIOPS > 0 {
		writeTime := float64(current.WriteTime-last.WriteTime) / deltaTime / 1000.0
		stat.WAwait = writeTime / stat.WriteIOPS
	}

	stat.UtilPercent = float64(current.IoTime-last.IoTime) / deltaTime / 10.0
	if stat.UtilPercent > 100.0 {
		stat.UtilPercent = 100.0
	}
}

// collectDiskUsage 收集所有挂载点的磁盘使用信息
func (dc *DataCollector) collectDiskUsage() []DiskUsageInfo {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return []DiskUsageInfo{}
	}

	// 获取磁盘 I/O 统计（用于计算 I/O 利用率）
	ioStats := dc.collectDiskIO()
	ioStatsMap := make(map[string]float64)
	for _, ioStat := range ioStats {
		ioStatsMap[ioStat.Name] = ioStat.UtilPercent
	}

	diskUsageList := make([]DiskUsageInfo, 0, len(partitions))

	for _, part := range partitions {
		if shouldSkipFilesystem(part.Fstype) {
			continue
		}

		usage, err := disk.Usage(part.Mountpoint)
		if err != nil {
			continue
		}

		device := extractDeviceName(part.Device)

		ioUtilPercent := 0.0
		if util, ok := ioStatsMap[device]; ok {
			ioUtilPercent = util
		}

		diskUsageList = append(diskUsageList, DiskUsageInfo{
			Device:        device,
			MountPoint:    part.Mountpoint,
			Fstype:        part.Fstype,
			Total:         usage.Total,
			Used:          usage.Used,
			Free:          usage.Free,
			UsedPercent:   usage.UsedPercent,
			InodesTotal:   usage.InodesTotal,
			InodesUsed:    usage.InodesUsed,
			InodesFree:    usage.InodesFree,
			InodesPercent: usage.InodesUsedPercent,
			IOUtilPercent: ioUtilPercent,
			DiskType:      detectDiskType(part.Device, part.Fstype),
		})
	}

	return diskUsageList
}

// shouldSkipFilesystem 判断是否应跳过虚拟文件系统
func shouldSkipFilesystem(fstype string) bool {
	skip := []string{"proc", "sysfs", "devtmpfs", "tmpfs", "cgroup", "overlay"}
	for _, prefix := range skip {
		if strings.HasPrefix(fstype, prefix) {
			return true
		}
	}
	if runtime.GOOS == "darwin" && (fstype == "autofs" || fstype == "devfs") {
		return true
	}
	return false
}

// extractDeviceName 从设备路径提取设备名（/dev/sda1 -> sda）
func extractDeviceName(device string) string {
	if !strings.HasPrefix(device, "/dev/") {
		return device
	}
	name := device[5:]
	// macOS: disk0s1 -> disk0, disk1 -> disk1（需在通用分区号移除前处理）
	if runtime.GOOS == "darwin" && strings.HasPrefix(name, "disk") {
		for i := 4; i < len(name); i++ {
			if name[i] < '0' || name[i] > '9' {
				name = name[:i]
				break
			}
		}
		return name
	}
	// 移除分区号
	if len(name) > 0 && name[len(name)-1] >= '0' && name[len(name)-1] <= '9' {
		name = name[:len(name)-1]
	}
	return name
}

// detectDiskType 根据设备路径和文件系统判断磁盘类型
func detectDiskType(device, fstype string) string {
	if strings.Contains(device, "nvme") {
		return "NVMe SSD"
	}
	if strings.Contains(device, "md") {
		return "RAID"
	}
	if runtime.GOOS == "darwin" && strings.Contains(device, "disk") {
		return "APFS SSD"
	}
	return "HDD"
}

// getMaxUserProcesses 获取用户最大进程数限制（ulimit -u）
func getMaxUserProcesses() int {
	cmd := exec.Command("sh", "-c", "ulimit -u")
	output, err := cmd.Output()
	if err != nil {
		return -1
	}
	limitStr := strings.TrimSpace(string(output))
	if limitStr == "unlimited" {
		return -1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return -1
	}
	return limit
}

// getTotalProcesses 使用已查询的进程数作为系统总进程数
// 不再重复调用 process.Processes()，避免重复开销
func getTotalProcesses(count int) int {
	if count < 0 {
		return -1
	}
	return count
}
