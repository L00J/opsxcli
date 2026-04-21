package sys

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- runeWidth ---

func TestRuneWidth_ASCII(t *testing.T) {
	assert.Equal(t, 5, runeWidth("hello"))
}

func TestRuneWidth_CJK(t *testing.T) {
	// Chinese characters are typically 2 cells wide
	assert.Equal(t, 4, runeWidth("你好"))
}

func TestRuneWidth_Empty(t *testing.T) {
	assert.Equal(t, 0, runeWidth(""))
}

func TestRuneWidth_Mixed(t *testing.T) {
	// "hi你好" = 2 ASCII + 4 CJK = 6
	assert.Equal(t, 6, runeWidth("hi你好"))
}

// --- getSortedDiskList ---

func TestGetSortedDiskList_BySpace(t *testing.T) {
	data := &SystemData{
		DiskUsageList: []DiskUsageInfo{
			{Device: "sda", UsedPercent: 30.0},
			{Device: "sdb", UsedPercent: 80.0},
			{Device: "sdc", UsedPercent: 50.0},
		},
	}
	result := getSortedDiskList(data, DiskSortBySpace)
	assert.Equal(t, "sdb", result[0].Device)
	assert.Equal(t, "sdc", result[1].Device)
	assert.Equal(t, "sda", result[2].Device)
}

func TestGetSortedDiskList_ByInodes(t *testing.T) {
	data := &SystemData{
		DiskUsageList: []DiskUsageInfo{
			{Device: "sda", InodesPercent: 20.0},
			{Device: "sdb", InodesPercent: 90.0},
		},
	}
	result := getSortedDiskList(data, DiskSortByInodes)
	assert.Equal(t, "sdb", result[0].Device)
	assert.Equal(t, "sda", result[1].Device)
}

func TestGetSortedDiskList_ByIO(t *testing.T) {
	data := &SystemData{
		DiskUsageList: []DiskUsageInfo{
			{Device: "sda", IOUtilPercent: 10.0},
			{Device: "sdb", IOUtilPercent: 95.0},
		},
	}
	result := getSortedDiskList(data, DiskSortByIO)
	assert.Equal(t, "sdb", result[0].Device)
	assert.Equal(t, "sda", result[1].Device)
}

func TestGetSortedDiskList_ByName(t *testing.T) {
	data := &SystemData{
		DiskUsageList: []DiskUsageInfo{
			{Device: "sdc"},
			{Device: "sda"},
			{Device: "sdb"},
		},
	}
	result := getSortedDiskList(data, DiskSortByName)
	assert.Equal(t, "sda", result[0].Device)
	assert.Equal(t, "sdb", result[1].Device)
	assert.Equal(t, "sdc", result[2].Device)
}

func TestGetSortedDiskList_DefaultSort(t *testing.T) {
	data := &SystemData{
		DiskUsageList: []DiskUsageInfo{
			{Device: "sda", UsedPercent: 10.0},
			{Device: "sdb", UsedPercent: 90.0},
		},
	}
	// 未知排序类型应默认按空间排序
	result := getSortedDiskList(data, DiskSortType(999))
	assert.Equal(t, "sdb", result[0].Device)
	assert.Equal(t, "sda", result[1].Device)
}

func TestGetSortedDiskList_Empty(t *testing.T) {
	data := &SystemData{
		DiskUsageList: []DiskUsageInfo{},
	}
	result := getSortedDiskList(data, DiskSortBySpace)
	assert.Empty(t, result)
}

func TestGetSortedDiskList_DoesNotModifyOriginal(t *testing.T) {
	data := &SystemData{
		DiskUsageList: []DiskUsageInfo{
			{Device: "sda", UsedPercent: 30.0},
			{Device: "sdb", UsedPercent: 80.0},
		},
	}
	_ = getSortedDiskList(data, DiskSortBySpace)
	// 原始数据顺序不变
	assert.Equal(t, "sda", data.DiskUsageList[0].Device)
	assert.Equal(t, "sdb", data.DiskUsageList[1].Device)
}

// --- DiskSortType constants ---

func TestDiskSortTypeConstants(t *testing.T) {
	assert.Equal(t, DiskSortType(0), DiskSortBySpace)
	assert.Equal(t, DiskSortType(1), DiskSortByInodes)
	assert.Equal(t, DiskSortType(2), DiskSortByIO)
	assert.Equal(t, DiskSortType(3), DiskSortByName)
}

// --- SortType constants ---

func TestSortTypeConstants(t *testing.T) {
	assert.Equal(t, SortType(0), SortByCPU)
	assert.Equal(t, SortType(1), SortByMemory)
	assert.Equal(t, SortType(2), SortByDiskIO)
	assert.Equal(t, SortType(3), SortByCPUTime)
}

// --- SetDiskSortType ---

func TestSetDiskSortType(t *testing.T) {
	SetDiskSortType(DiskSortByIO)
	assert.Equal(t, DiskSortByIO, currentDiskSort)
	// 恢复默认
	SetDiskSortType(DiskSortBySpace)
	assert.Equal(t, DiskSortBySpace, currentDiskSort)
}

// --- SetSortType ---

func TestSetSortType(t *testing.T) {
	SetSortType(SortByMemory)
	assert.Equal(t, SortByMemory, currentSort)
	// 恢复默认
	SetSortType(SortByCPU)
	assert.Equal(t, SortByCPU, currentSort)
}

// --- NewDataCollector ---

func TestNewDataCollector(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dc := NewDataCollector(ctx, 2*time.Second)
	assert.NotNil(t, dc)
	assert.Equal(t, 2*time.Second, dc.updateInterval)
	assert.NotNil(t, dc.lastNetStats)
	assert.NotNil(t, dc.lastProcIO)
	assert.NotNil(t, dc.lastDiskIO)
	assert.True(t, dc.firstCollect)
}

// --- NewSysMonitor ---

func TestNewSysMonitor(t *testing.T) {
	sm := NewSysMonitor()
	assert.NotNil(t, sm)
}

// --- formatCPUTime 边界场景 ---

func TestFormatCPUTime_Zero(t *testing.T) {
	assert.Equal(t, "0.00s", formatCPUTime(0))
}

func TestFormatCPUTime_VeryLarge(t *testing.T) {
	// 100 hours
	result := formatCPUTime(360000)
	assert.Equal(t, "100h 00m", result)
}

// --- formatRuntime 边界场景 ---

func TestFormatRuntime_Zero(t *testing.T) {
	assert.Equal(t, "0s", formatRuntime(0))
}

func TestFormatRuntime_LessThanSecond(t *testing.T) {
	assert.Equal(t, "0s", formatRuntime(500*time.Millisecond))
}

func TestFormatRuntime_VeryLarge(t *testing.T) {
	result := formatRuntime(72*time.Hour + 30*time.Minute)
	assert.Equal(t, "72h30m", result)
}

// --- formatBytes 边界场景 ---

func TestFormatBytes_VeryLarge(t *testing.T) {
	// PiB
	pib := 1.0 * 1024 * 1024 * 1024 * 1024 * 1024
	assert.Equal(t, "1.0PiB", formatBytes(pib))
}

func TestFormatBytes_Negative(t *testing.T) {
	// 负数应该被当作小于 unit 的值处理
	result := formatBytes(-512)
	assert.Contains(t, result, "-")
}

// --- formatDiskIO 更多场景 ---

func TestFormatDiskIO_EqualReadWrite(t *testing.T) {
	result := formatDiskIO(1024, 1024)
	assert.Contains(t, result, "1.0KiB/1.0KiB")
}

// --- SystemData 字段验证 ---

func TestSystemData_Fields(t *testing.T) {
	data := &SystemData{
		CPUPercent:       []float64{50.0, 30.0},
		CPUCores:         2,
		TotalProcesses:   200,
		MaxUserProcesses: 4096,
		Platform:         "linux",
		UpdateTime:       time.Now(),
	}
	assert.Equal(t, 2, len(data.CPUPercent))
	assert.Equal(t, 2, data.CPUCores)
	assert.Equal(t, 200, data.TotalProcesses)
	assert.Equal(t, "linux", data.Platform)
	assert.False(t, data.UpdateTime.IsZero())
}

// --- DiskUsageInfo 字段验证 ---

func TestDiskUsageInfo_Fields(t *testing.T) {
	info := DiskUsageInfo{
		Device:        "nvme0n1p1",
		MountPoint:    "/",
		Fstype:        "ext4",
		Total:         500 * 1024 * 1024 * 1024,
		Used:          250 * 1024 * 1024 * 1024,
		Free:          250 * 1024 * 1024 * 1024,
		UsedPercent:   50.0,
		InodesTotal:   30000000,
		InodesUsed:    15000000,
		InodesFree:    15000000,
		InodesPercent: 50.0,
		DiskType:      "NVMe SSD",
	}
	assert.Equal(t, "nvme0n1p1", info.Device)
	assert.Equal(t, "/", info.MountPoint)
	assert.Equal(t, "ext4", info.Fstype)
	assert.Equal(t, 50.0, info.UsedPercent)
	assert.Equal(t, "NVMe SSD", info.DiskType)
}

// --- ProcessInfo 字段验证 ---

func TestProcessInfo_Fields(t *testing.T) {
	pi := ProcessInfo{
		PID:           1234,
		Name:          "nginx",
		CPU:           5.5,
		Mem:           2.3,
		MemRSS:        64 * 1024 * 1024,
		Status:        "S",
		User:          "www-data",
		Command:       "/usr/sbin/nginx",
		Threads:       4,
		DiskReadRate:  1024.0,
		DiskWriteRate: 2048.0,
	}
	assert.Equal(t, int32(1234), pi.PID)
	assert.Equal(t, "nginx", pi.Name)
	assert.Equal(t, 5.5, pi.CPU)
	assert.Equal(t, float32(2.3), pi.Mem)
	assert.Equal(t, "S", pi.Status)
}

// --- CPUTimesStat 字段验证 ---

func TestCPUTimesStat_Fields(t *testing.T) {
	stat := CPUTimesStat{
		CPU:      "cpu0",
		User:     45.0,
		Nice:     1.0,
		System:   20.0,
		Idle:     30.0,
		Iowait:   2.0,
		Steal:    0.5,
	}
	assert.Equal(t, "cpu0", stat.CPU)
	assert.Equal(t, 45.0, stat.User)
	assert.Equal(t, 30.0, stat.Idle)
}

// --- NetStatSnapshot 字段验证 ---

func TestNetStatSnapshot_Fields(t *testing.T) {
	now := time.Now()
	snap := NetStatSnapshot{
		BytesSent: 1024000,
		BytesRecv: 2048000,
		Timestamp: now,
	}
	assert.Equal(t, uint64(1024000), snap.BytesSent)
	assert.Equal(t, uint64(2048000), snap.BytesRecv)
	assert.Equal(t, now, snap.Timestamp)
}

// --- DiskIOStat 字段验证 ---

func TestDiskIOStat_Fields(t *testing.T) {
	stat := DiskIOStat{
		Name:        "sda",
		ReadIOPS:    100.0,
		WriteIOPS:   50.0,
		AvgRqSz:     128.0,
		UtilPercent: 85.5,
	}
	assert.Equal(t, "sda", stat.Name)
	assert.Equal(t, 100.0, stat.ReadIOPS)
	assert.Equal(t, 85.5, stat.UtilPercent)
}
