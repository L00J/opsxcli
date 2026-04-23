package sys

import (
	"context"
	"runtime"
	"sync"
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
	maxUserProcs   int                             // 缓存 ulimit -u 结果
	maxProcsOnce   sync.Once                       // 确保 ulimit 只执行一次
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

// getCachedMaxUserProcesses 获取缓存的用户最大进程数限制
// ulimit -u 在会话期间不会变化，只需执行一次
func (dc *DataCollector) getCachedMaxUserProcesses() int {
	dc.maxProcsOnce.Do(func() {
		dc.maxUserProcs = getMaxUserProcesses()
	})
	return dc.maxUserProcs
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

// collect 并行收集系统数据
// 所有独立数据源通过 goroutine 并行采集，大幅缩短 collect 总耗时
func (dc *DataCollector) collect() *SystemData {
	data := &SystemData{
		UpdateTime: time.Now(),
	}

	// 平台检测和降级提示
	data.Platform = runtime.GOOS
	data.ProcessIOUnsupported = runtime.GOOS == "darwin"
	data.DiskIOLimited = runtime.GOOS == "darwin"

	var wg sync.WaitGroup

	// 并行组 1: CPU 数据（单次 cpu.Times 调用 → 同时计算 CPUTimes + CPUPercent）
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := dc.collectCPUData()
		data.CPUTimes = result.cpuTimes
		data.CPUPercent = result.cpuPercent
		data.CPUCores = result.cpuCores
		if data.CPUCores == 0 {
			cores, _ := cpu.Counts(true)
			data.CPUCores = cores
		}
	}()

	// 并行组 2: 内存 + 负载
	wg.Add(1)
	go func() {
		defer wg.Done()
		data.MemInfo, _ = mem.VirtualMemory()
		data.LoadAvg, _ = load.Avg()
	}()

	// 并行组 3: 磁盘（IO 统计 → 使用信息，内部顺序执行避免重复 IO 查询）
	wg.Add(1)
	go func() {
		defer wg.Done()
		diskIOStats := dc.collectDiskIO()
		data.DiskIOStats = diskIOStats
		data.DiskInfo, _ = disk.Usage("/")
		data.DiskUsageList = dc.collectDiskUsage(diskIOStats)
	}()

	// 并行组 4: 网络
	wg.Add(1)
	go func() {
		defer wg.Done()
		data.NetStats, _ = netutil.IOCounters(true)
	}()

	// 并行组 5: 进程（最重的采集任务，独立 goroutine 并行）
	isFirst := dc.firstCollect
	dc.firstCollect = false
	var procs []*ProcessInfo
	var totalProcCount int
	wg.Add(1)
	go func() {
		defer wg.Done()
		procs, totalProcCount = dc.collectProcesses(isFirst)
	}()

	// 等待所有并行采集完成
	wg.Wait()

	// 组装进程相关数据
	data.Processes = procs
	data.TotalProcesses = getTotalProcesses(totalProcCount)
	data.MaxUserProcesses = dc.getCachedMaxUserProcesses()

	// 保存网络统计快照用于下次计算速率
	data.LastNetStats = dc.lastNetStats
	dc.lastNetStats = make(map[string]*NetStatSnapshot)
	for _, stat := range data.NetStats {
		dc.lastNetStats[stat.Name] = &NetStatSnapshot{
			BytesSent: stat.BytesSent,
			BytesRecv: stat.BytesRecv,
			Timestamp: time.Now(),
		}
	}

	return data
}
