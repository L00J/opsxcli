package sys

import (
	"context"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// DataCollector 数据收集器
type DataCollector struct {
	ctx            context.Context
	updateInterval time.Duration
	lastNetStats   map[string]*NetStatSnapshot     // 用于计算网络速率
	lastProcIO     map[int32]*ProcessIO            // 用于计算进程磁盘 I/O 速率
	lastCPUTimes   []cpu.TimesStat                 // 用于计算 CPU 详细统计
	lastDiskIO     map[string]*disk.IOCountersStat // 用于计算磁盘 I/O 详细统计
	firstCollect   bool                            // 标记是否是首次收集
}

// ProcessIO 进程 I/O 快照
type ProcessIO struct {
	ReadBytes  uint64
	WriteBytes uint64
	Timestamp  time.Time
}

// NewDataCollector 创建数据收集器
func NewDataCollector(ctx context.Context, interval time.Duration) *DataCollector {
	return &DataCollector{
		ctx:            ctx,
		updateInterval: interval,
		lastNetStats:   make(map[string]*NetStatSnapshot),
		lastProcIO:     make(map[int32]*ProcessIO),
		lastDiskIO:     make(map[string]*disk.IOCountersStat),
		firstCollect:   true,
	}
}

// Start 启动数据收集循环
func (dc *DataCollector) Start(callback func(*SystemData)) {
	// 后台初始化 lastCPUTimes，确保首次收集时基于时间差的计算可用
	go func() {
		// 初始化 lastCPUTimes
		currentTimes, _ := cpu.Times(true)
		if len(currentTimes) > 0 {
			dc.lastCPUTimes = currentTimes
		}

		// 等待一小段时间，确保有时间差来计算 CPU 使用率
		time.Sleep(100 * time.Millisecond)

		// 再次更新 lastCPUTimes
		currentTimes, _ = cpu.Times(true)
		if len(currentTimes) > 0 {
			dc.lastCPUTimes = currentTimes
		}

		// 首次收集并回调
		data := dc.collect()
		callback(data)
	}()

	// 启动定时收集循环
	go func() {
		// 等待首次收集完成后再启动 ticker，避免与首次收集冲突
		time.Sleep(200 * time.Millisecond)

		ticker := time.NewTicker(dc.updateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-dc.ctx.Done():
				return
			case <-ticker.C:
				data := dc.collect()
				callback(data)
			}
		}
	}()
}

// collect 收集系统数据
func (dc *DataCollector) collect() *SystemData {
	data := &SystemData{
		UpdateTime: time.Now(), // 记录更新时间
	}

	// 平台检测和降级提示
	data.Platform = runtime.GOOS
	data.ProcessIOUnsupported = runtime.GOOS == "darwin"
	data.DiskIOLimited = runtime.GOOS == "darwin"

	// 收集 CPU 详细时间统计（类似 mpstat）
	data.CPUTimes = dc.collectCPUTimes()

	// 基于 CPUTimes 计算 CPU 使用率（只计算 User + System，不包括 Nice）
	// 这样可以更准确地反映实际 CPU 负载，排除低优先级的 nice 进程
	data.CPUPercent = dc.calculateCPUPercentFromTimes()

	// 收集 CPU 核心数
	data.CPUCores = len(data.CPUPercent)
	if data.CPUCores == 0 {
		// 如果没有获取到核心使用率，尝试直接获取核心数
		cores, _ := cpu.Counts(true)
		data.CPUCores = cores
	}

	data.MemInfo, _ = mem.VirtualMemory()
	data.LoadAvg, _ = load.Avg()
	data.DiskInfo, _ = disk.Usage("/")

	// 收集所有挂载点的磁盘使用信息
	data.DiskUsageList = dc.collectDiskUsage()

	// 收集磁盘 I/O 详细统计（类似 iostat）
	data.DiskIOStats = dc.collectDiskIO()

	data.NetStats, _ = net.IOCounters(true)

	// 保存网络统计快照用于计算速率
	data.LastNetStats = dc.lastNetStats
	dc.lastNetStats = make(map[string]*NetStatSnapshot)
	for _, stat := range data.NetStats {
		dc.lastNetStats[stat.Name] = &NetStatSnapshot{
			BytesSent: stat.BytesSent,
			BytesRecv: stat.BytesRecv,
			Timestamp: time.Now(),
		}
	}

	// 收集进程信息（优化：首次收集时减少数量，加快速度）
	isFirst := dc.firstCollect
	dc.firstCollect = false
	data.Processes = dc.collectProcesses(isFirst)

	// 收集进程数量统计
	data.TotalProcesses = getTotalProcesses()
	data.MaxUserProcesses = getMaxUserProcesses()

	return data
}

// collectProcesses 收集进程信息
func (dc *DataCollector) collectProcesses(isFirst bool) []*ProcessInfo {
	processes, err := process.Processes()
	if err != nil {
		return []*ProcessInfo{}
	}

	if len(processes) == 0 {
		return []*ProcessInfo{}
	}

	procInfos := make([]*ProcessInfo, 0, 200)

	// 限制处理的进程数量，加快收集速度
	// 首次收集时只处理50个进程，加快显示速度
	maxProcs := 200
	if isFirst {
		maxProcs = 50 // 首次只处理50个，加快首次加载
	}
	if len(processes) > maxProcs {
		processes = processes[:maxProcs]
	}

	// 优化并发策略：减少并发数，避免过度上下文切换
	semaphore := make(chan struct{}, 20) // 降低到20个并发，减少系统负担
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, p := range processes {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量

		go func(proc *process.Process) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量

			// 快速获取基本信息，跳过错误
			name, err := proc.Name()
			if err != nil {
				return
			}

			// 顺序获取数据（减少goroutine创建开销）
			var cpuPercent float64
			var memPercent float32
			var status []string
			var username, cmdline string
			var threads int32
			var createTime int64

			// 顺序获取，忽略错误继续执行
			cpuPercent, _ = proc.CPUPercent()
			memPercent, _ = proc.MemoryPercent()
			status, _ = proc.Status()
			username, _ = proc.Username()
			cmdline, _ = proc.Cmdline()
			threads, _ = proc.NumThreads()
			createTime, _ = proc.CreateTime()

			// 获取 CPU 时间
			times, _ := proc.Times()
			cpuTime := 0.0
			if times != nil {
				cpuTime = times.User + times.System
			}

			// 获取内存信息
			memInfo, _ := proc.MemoryInfo()
			memRSS := uint64(0)
			memVMS := uint64(0)
			if memInfo != nil {
				memRSS = memInfo.RSS
				memVMS = memInfo.VMS
			}

			// 获取磁盘 I/O
			ioStat, ioErr := proc.IOCounters()
			diskRead := uint64(0)
			diskWrite := uint64(0)
			diskReadRate := 0.0
			diskWriteRate := 0.0

			if ioErr != nil {
				// macOS 上 process.IOCounters() 返回 "not implemented yet"
				// 优雅降级：保持为 0，不计算速率
				if runtime.GOOS == "darwin" {
					// macOS 不支持进程级 I/O 统计
					diskRead = 0
					diskWrite = 0
				}
			} else if ioStat != nil {
				diskRead = ioStat.ReadBytes
				diskWrite = ioStat.WriteBytes

				// 计算速率（需要加锁保护）
				mu.Lock()
				if lastIO, ok := dc.lastProcIO[proc.Pid]; ok {
					deltaTime := time.Since(lastIO.Timestamp).Seconds()
					if deltaTime > 0 && deltaTime < 10 {
						diskReadRate = float64(diskRead-lastIO.ReadBytes) / deltaTime
						diskWriteRate = float64(diskWrite-lastIO.WriteBytes) / deltaTime
						if diskReadRate < 0 {
							diskReadRate = 0
						}
						if diskWriteRate < 0 {
							diskWriteRate = 0
						}
					}
				}

				// 保存当前 I/O 快照
				dc.lastProcIO[proc.Pid] = &ProcessIO{
					ReadBytes:  diskRead,
					WriteBytes: diskWrite,
					Timestamp:  time.Now(),
				}
				mu.Unlock()
			}

			// 计算运行时间
			runTime := time.Duration(0)
			if createTime > 0 {
				runTime = time.Since(time.Unix(createTime, 0))
			}

			procInfo := &ProcessInfo{
				PID:           proc.Pid,
				Name:          name,
				CPU:           cpuPercent,
				CPUTime:       cpuTime,
				Mem:           float32(memPercent),
				MemRSS:        memRSS,
				MemVMS:        memVMS,
				Status:        strings.Join(status, ","),
				User:          username,
				Command:       cmdline,
				Threads:       threads,
				CreateTime:    createTime,
				RunTime:       runTime,
				DiskRead:      diskRead,
				DiskWrite:     diskWrite,
				DiskReadRate:  diskReadRate,
				DiskWriteRate: diskWriteRate,
			}

			mu.Lock()
			procInfos = append(procInfos, procInfo)
			mu.Unlock()
		}(p)
	}

	wg.Wait()

	// 按CPU使用率排序
	sort.Slice(procInfos, func(i, j int) bool {
		return procInfos[i].CPU > procInfos[j].CPU
	})

	// 只保留前150个（增加数量，确保有足够的数据显示）
	if len(procInfos) > 150 {
		procInfos = procInfos[:150]
	}

	return procInfos
}

// collectCPUTimes 收集 CPU 详细时间统计（类似 mpstat）
func (dc *DataCollector) collectCPUTimes() []CPUTimesStat {
	// 获取当前 CPU 时间统计
	currentTimes, err := cpu.Times(true) // true 表示获取所有核心
	if err != nil {
		return []CPUTimesStat{}
	}

	// 如果没有上次的数据，保存当前数据并返回空（需要等待下一次收集才能计算百分比）
	// 注意：初始化在 Start() 函数中完成，这里只是检查
	if len(dc.lastCPUTimes) == 0 {
		dc.lastCPUTimes = currentTimes
		return []CPUTimesStat{}
	}

	// 计算时间差
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

		// 计算总时间差
		totalDiff := (current.User + current.Nice + current.System + current.Idle + current.Iowait +
			current.Irq + current.Softirq + current.Steal + current.Guest + current.GuestNice) -
			(last.User + last.Nice + last.System + last.Idle + last.Iowait +
				last.Irq + last.Softirq + last.Steal + last.Guest + last.GuestNice)

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

		// 将 all 放在最前面
		stats = append([]CPUTimesStat{allStat}, stats...)
	}

	// 保存当前数据作为下次的基准
	dc.lastCPUTimes = currentTimes

	return stats
}

// calculateCPUPercentFromTimes 基于 CPUTimes 计算 CPU 使用率（只计算 User + System，不包括 Nice）
func (dc *DataCollector) calculateCPUPercentFromTimes() []float64 {
	// 如果还没有 CPUTimes 数据，使用 cpu.Percent 作为后备
	if len(dc.lastCPUTimes) == 0 {
		// 首次加载使用1秒采样，确保数据准确
		cpuPercents, _ := cpu.Percent(1*time.Second, true)
		return cpuPercents
	}

	// 获取当前 CPU 时间
	currentTimes, err := cpu.Times(true)
	if err != nil {
		// 如果失败，使用 cpu.Percent 作为后备
		cpuPercents, _ := cpu.Percent(1*time.Second, true)
		return cpuPercents
	}

	// 如果数量不匹配，使用 cpu.Percent 作为后备
	if len(currentTimes) != len(dc.lastCPUTimes) {
		cpuPercents, _ := cpu.Percent(1*time.Second, true)
		return cpuPercents
	}

	percentages := make([]float64, 0, len(currentTimes))

	// 计算每个 CPU 核心的使用率
	for i, current := range currentTimes {
		if i >= len(dc.lastCPUTimes) {
			continue
		}
		last := dc.lastCPUTimes[i]

		// 计算总时间差（包括所有状态）
		totalDiff := (current.User + current.Nice + current.System + current.Idle + current.Iowait +
			current.Irq + current.Softirq + current.Steal + current.Guest + current.GuestNice) -
			(last.User + last.Nice + last.System + last.Idle + last.Iowait +
				last.Irq + last.Softirq + last.Steal + last.Guest + last.GuestNice)

		// 如果时间差太小（小于0.5秒），说明时间间隔不够，无法准确计算
		if totalDiff <= 0.5 {
			// 使用1秒采样获取当前CPU使用率
			allPercents, _ := cpu.Percent(1*time.Second, true)
			if i < len(allPercents) {
				percentages = append(percentages, allPercents[i])
			} else {
				percentages = append(percentages, 0.0)
			}
			continue
		}

		// 只计算 User + System，不包括 Nice（nice 进程是低优先级的，不应该算作高负载）
		// 这与 top 命令的计算方式一致：%CPU = (us + sy) / total * 100
		userDiff := current.User - last.User
		systemDiff := current.System - last.System
		usagePercent := ((userDiff + systemDiff) / totalDiff) * 100.0

		// 确保百分比在合理范围内
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

// collectDiskIO 收集磁盘 I/O 详细统计（类似 iostat）
func (dc *DataCollector) collectDiskIO() []DiskIOStat {
	// 获取当前磁盘 I/O 统计
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

		// 如果有上次的数据，计算速率
		if last, ok := dc.lastDiskIO[name]; ok && last != nil {
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

			// macOS 下部分字段可能为 0，需要判断
			if runtime.GOOS == "darwin" {
				// macOS 磁盘 I/O 统计受限，加权 I/O 和时间字段可能为 0
				// 只使用基础字段，避免显示空值
				if current.WeightedIO == 0 && last.WeightedIO == 0 {
					stat.AvgQuSz = 0
				} else {
					stat.AvgQuSz = float64(current.WeightedIO-last.WeightedIO) / deltaTime / 1000.0
				}

				if current.IoTime == 0 && last.IoTime == 0 {
					stat.Await = 0
					stat.UtilPercent = 0
				} else {
					// 平均等待时间 (ms)
					if stat.ReadIOPS+stat.WriteIOPS > 0 {
						ioTime := float64(current.IoTime-last.IoTime) / deltaTime / 1000.0
						stat.Await = ioTime / (stat.ReadIOPS + stat.WriteIOPS)
					}
					// 利用率 (%)
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
			} else {
				// Linux 下完整计算
				// 平均队列长度
				stat.AvgQuSz = float64(current.WeightedIO-last.WeightedIO) / deltaTime / 1000.0 // 转换为毫秒

				// 平均等待时间 (ms)
				if stat.ReadIOPS+stat.WriteIOPS > 0 {
					ioTime := float64(current.IoTime-last.IoTime) / deltaTime / 1000.0 // 转换为毫秒
					stat.Await = ioTime / (stat.ReadIOPS + stat.WriteIOPS)
				}

				// 读取等待时间 (ms)
				if stat.ReadIOPS > 0 {
					readTime := float64(current.ReadTime-last.ReadTime) / deltaTime / 1000.0
					stat.RAwait = readTime / stat.ReadIOPS
				}

				// 写入等待时间 (ms)
				if stat.WriteIOPS > 0 {
					writeTime := float64(current.WriteTime-last.WriteTime) / deltaTime / 1000.0
					stat.WAwait = writeTime / stat.WriteIOPS
				}
			}

			// 服务时间 (ms) - 简化计算
			if stat.ReadIOPS+stat.WriteIOPS > 0 {
				stat.Svctm = stat.Await * 0.8 // 估算值
			}

			// Linux 利用率 (%)
			if runtime.GOOS != "darwin" {
				stat.UtilPercent = float64(current.IoTime-last.IoTime) / deltaTime / 10.0
				if stat.UtilPercent > 100.0 {
					stat.UtilPercent = 100.0
				}
			}
		}

		stats = append(stats, stat)
	}

	// 保存当前数据作为下次的基准（转换为指针类型）
	dc.lastDiskIO = make(map[string]*disk.IOCountersStat)
	for name, stat := range currentIO {
		statCopy := stat // 创建副本
		dc.lastDiskIO[name] = &statCopy
	}

	return stats
}

// collectDiskUsage 收集所有挂载点的磁盘使用信息
func (dc *DataCollector) collectDiskUsage() []DiskUsageInfo {
	// 获取所有分区
	partitions, err := disk.Partitions(false) // false 表示不包含虚拟文件系统
	if err != nil {
		return []DiskUsageInfo{}
	}

	diskUsageList := make([]DiskUsageInfo, 0)

	// 获取磁盘 I/O 统计（用于计算 I/O 利用率）
	// 注意：collectDiskIO 会更新 dc.lastDiskIO，所以需要先调用
	ioStats := dc.collectDiskIO()
	ioStatsMap := make(map[string]float64)
	for _, ioStat := range ioStats {
		ioStatsMap[ioStat.Name] = ioStat.UtilPercent
	}

	for _, part := range partitions {
		// 跳过虚拟文件系统
		if strings.HasPrefix(part.Fstype, "proc") || strings.HasPrefix(part.Fstype, "sysfs") ||
			strings.HasPrefix(part.Fstype, "devtmpfs") || strings.HasPrefix(part.Fstype, "tmpfs") ||
			strings.HasPrefix(part.Fstype, "cgroup") || strings.HasPrefix(part.Fstype, "overlay") {
			continue
		}

		// macOS 下跳过不需要的文件系统
		if runtime.GOOS == "darwin" {
			if part.Fstype == "autofs" || part.Fstype == "devfs" {
				continue
			}
		}

		// 获取磁盘使用情况
		usage, err := disk.Usage(part.Mountpoint)
		if err != nil {
			continue
		}

		// 提取设备名（从 /dev/sda1 提取 sda）
		device := part.Device
		if strings.HasPrefix(device, "/dev/") {
			device = device[5:] // 移除 /dev/
			// 移除分区号（如 sda1 -> sda）
			if len(device) > 0 && device[len(device)-1] >= '0' && device[len(device)-1] <= '9' {
				device = device[:len(device)-1]
			}
		}

		// macOS 设备名处理（如 disk0s1 -> disk0）
		if runtime.GOOS == "darwin" && strings.HasPrefix(device, "disk") {
			// 提取 diskN 部分
			for i := 4; i < len(device); i++ {
				if device[i] >= '0' && device[i] <= '9' {
					continue
				}
				device = device[:i]
				break
			}
		}

		// 判断磁盘类型
		diskType := "HDD"
		if strings.Contains(part.Device, "nvme") {
			diskType = "NVMe SSD"
		} else if strings.Contains(part.Device, "md") {
			diskType = "RAID"
		} else if strings.Contains(part.Fstype, "ext") || strings.Contains(part.Fstype, "xfs") {
			// 可以根据需要进一步判断
		} else if runtime.GOOS == "darwin" {
			if strings.Contains(part.Device, "disk") {
				diskType = "APFS SSD"
			}
		}

		// 获取 Inode 信息
		inodesTotal := usage.InodesTotal
		inodesUsed := usage.InodesUsed
		inodesFree := usage.InodesFree
		inodesPercent := usage.InodesUsedPercent

		// 获取 I/O 利用率
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
			InodesTotal:   inodesTotal,
			InodesUsed:    inodesUsed,
			InodesFree:    inodesFree,
			InodesPercent: inodesPercent,
			IOUtilPercent: ioUtilPercent,
			DiskType:      diskType,
		})
	}

	return diskUsageList
}

// getMaxUserProcesses 获取用户最大进程数限制（ulimit -u）
func getMaxUserProcesses() int {
	// 使用 ulimit -u 命令获取用户最大进程数
	cmd := exec.Command("sh", "-c", "ulimit -u")
	output, err := cmd.Output()
	if err != nil {
		return -1
	}

	// 解析输出
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

// getTotalProcesses 获取当前系统总进程数
func getTotalProcesses() int {
	processes, err := process.Processes()
	if err != nil {
		return -1
	}
	return len(processes)
}
