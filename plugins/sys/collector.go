package sys

import (
	"context"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	netutil "github.com/shirou/gopsutil/v3/net"
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
		// 初始化 lastCPUTimes（不再 sleep，直接读取当前值作为基准）
		currentTimes, _ := cpu.Times(true)
		if len(currentTimes) > 0 {
			dc.lastCPUTimes = currentTimes
		}

		// 首次收集并回调
		data := dc.collect()
		callback(data)
	}()

	// 启动定时收集循环（不再 sleep 等待，立即启动 ticker）
	go func() {
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
		UpdateTime: time.Now(),
	}

	// 平台检测和降级提示
	data.Platform = runtime.GOOS
	data.ProcessIOUnsupported = runtime.GOOS == "darwin"
	data.DiskIOLimited = runtime.GOOS == "darwin"

	// 收集 CPU 详细时间统计（类似 mpstat）
	data.CPUTimes = dc.collectCPUTimes()

	// 基于 CPUTimes 计算 CPU 使用率
	data.CPUPercent = dc.calculateCPUPercentFromTimes()

	// 收集 CPU 核心数
	data.CPUCores = len(data.CPUPercent)
	if data.CPUCores == 0 {
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

	data.NetStats, _ = netutil.IOCounters(true)

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
	// 同时获取系统总进程数，避免重复调用 process.Processes()
	isFirst := dc.firstCollect
	dc.firstCollect = false
	procs, totalProcCount := dc.collectProcesses(isFirst)
	data.Processes = procs

	// 收集进程数量统计（直接使用 collectProcesses 返回的总数，不再重复查询）
	data.TotalProcesses = getTotalProcesses(totalProcCount)
	data.MaxUserProcesses = getMaxUserProcesses()

	return data
}
