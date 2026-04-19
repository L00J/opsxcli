package sys

import (
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// TabType 标签页类型
type TabType int

const (
	TabOverview TabType = iota
	TabProcesses
	TabCPU
	TabMemory
	TabDisk
	// TabNetwork 已移除，使用 opsxcli net 查看详细网络信息
)

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID           int32
	Name          string
	CPU           float64
	CPUTime       float64 // CPU 累计时间（秒）
	Mem           float32 // 内存百分比
	MemRSS        uint64  // 实际物理内存（字节）
	MemVMS        uint64  // 虚拟内存（字节）
	Status        string
	User          string
	Command       string
	Threads       int32
	CreateTime    int64
	RunTime       time.Duration // 运行时间
	DiskRead      uint64        // 磁盘读取（字节）
	DiskWrite     uint64        // 磁盘写入（字节）
	DiskReadRate  float64       // 磁盘读取速率（字节/秒）
	DiskWriteRate float64       // 磁盘写入速率（字节/秒）
}

// CPUTimesStat CPU 时间统计（类似 mpstat）
type CPUTimesStat struct {
	CPU       string  // CPU 编号或 "all"
	User      float64 // %usr
	Nice      float64 // %nice
	System    float64 // %sys
	Idle      float64 // %idle
	Iowait    float64 // %iowait
	Irq       float64 // %irq
	Softirq   float64 // %soft
	Steal     float64 // %steal
	Guest     float64 // %guest
	GuestNice float64 // %gnice
}

// DiskIOStat 磁盘 I/O 统计（类似 iostat）
type DiskIOStat struct {
	Name        string  // 设备名
	ReadIOPS    float64 // r/s
	WriteIOPS   float64 // w/s
	ReadKBps    float64 // rkB/s
	WriteKBps   float64 // wkB/s
	AvgRqSz     float64 // avgrq-sz
	AvgQuSz     float64 // avgqu-sz
	Await       float64 // await
	RAwait      float64 // r_await
	WAwait      float64 // w_await
	Svctm       float64 // svctm
	UtilPercent float64 // %util
}

// DiskUsageInfo 磁盘使用信息（包含所有挂载点）
type DiskUsageInfo struct {
	Device        string  // 设备名（如 sda, nvme0）
	MountPoint    string  // 挂载点
	Fstype        string  // 文件系统类型
	Total         uint64  // 总容量（字节）
	Used          uint64  // 已用（字节）
	Free          uint64  // 空闲（字节）
	UsedPercent   float64 // 使用百分比
	InodesTotal   uint64  // Inode 总数
	InodesUsed    uint64  // Inode 已用
	InodesFree    uint64  // Inode 空闲
	InodesPercent float64 // Inode 使用百分比
	IOUtilPercent float64 // I/O 利用率（从 DiskIOStats 获取）
	DiskType      string  // 磁盘类型（HDD, NVMe SSD, RAID等）
}

// SystemData 系统数据缓存
type SystemData struct {
	CPUPercent           []float64
	CPUTimes             []CPUTimesStat // CPU 详细时间统计（类似 mpstat）
	CPUCores             int            // CPU 核心数
	MemInfo              *mem.VirtualMemoryStat
	LoadAvg              *load.AvgStat
	DiskInfo             *disk.UsageStat
	DiskUsageList        []DiskUsageInfo // 所有挂载点的磁盘使用信息
	DiskIOStats          []DiskIOStat    // 磁盘 I/O 详细统计（类似 iostat）
	NetStats             []net.IOCountersStat
	Processes            []*ProcessInfo
	LastNetStats         map[string]*NetStatSnapshot // 用于计算网络速率
	UpdateTime           time.Time                   // 数据更新时间
	TotalProcesses       int                         // 当前系统总进程数
	MaxUserProcesses     int                         // 用户最大进程数限制（ulimit -u）
	Platform             string                      // 运行平台（darwin, linux等）
	ProcessIOUnsupported bool                        // macOS 下进程 I/O 统计不可用
	DiskIOLimited        bool                        // macOS 下磁盘 I/O 详细统计受限
}

// NetStatSnapshot 网络统计快照
type NetStatSnapshot struct {
	BytesSent uint64
	BytesRecv uint64
	Timestamp time.Time
}
