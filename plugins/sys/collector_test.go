package sys

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/stretchr/testify/assert"
)

// === calcDiskIORates 测试 ===

func TestCalcDiskIORates_BasicRates(t *testing.T) {
	current := disk.IOCountersStat{
		ReadCount:  2000,
		WriteCount: 1000,
		ReadBytes:  1024 * 1024 * 10, // 10 MB
		WriteBytes: 1024 * 1024 * 5,  // 5 MB
		IoTime:     50000,
		WeightedIO: 100000,
		ReadTime:   30000,
		WriteTime:  20000,
	}
	last := disk.IOCountersStat{
		ReadCount:  1000,
		WriteCount: 500,
		ReadBytes:  1024 * 1024 * 5,
		WriteBytes: 1024 * 1024 * 2,
		IoTime:     25000,
		WeightedIO: 50000,
		ReadTime:   15000,
		WriteTime:  10000,
	}

	stat := &DiskIOStat{Name: "sda"}
	calcDiskIORates(stat, current, last, 1.0)

	// IOPS
	assert.InDelta(t, 1000.0, stat.ReadIOPS, 0.1)
	assert.InDelta(t, 500.0, stat.WriteIOPS, 0.1)

	// Throughput (KB/s)
	assert.InDelta(t, 5120.0, stat.ReadKBps, 1.0)  // 5MB / 1s = 5120 KB/s
	assert.InDelta(t, 3072.0, stat.WriteKBps, 1.0) // 3MB / 1s = 3072 KB/s

	// Average request size should be calculated
	assert.Greater(t, stat.AvgRqSz, 0.0)

	// Svctm should be calculated when IOPS > 0
	assert.Greater(t, stat.Svctm, 0.0)
}

func TestCalcDiskIORates_NoIOPS(t *testing.T) {
	// When IOPS is 0, AvgRqSz and Svctm should remain 0
	current := disk.IOCountersStat{
		ReadCount:  100,
		WriteCount: 50,
	}
	last := disk.IOCountersStat{
		ReadCount:  100,
		WriteCount: 50,
	}

	stat := &DiskIOStat{Name: "sdb"}
	calcDiskIORates(stat, current, last, 1.0)

	assert.Equal(t, 0.0, stat.AvgRqSz)
	assert.Equal(t, 0.0, stat.Svctm)
}

func TestCalcDiskIORates_SmallDelta(t *testing.T) {
	current := disk.IOCountersStat{
		ReadCount:  110,
		WriteCount: 55,
		ReadBytes:  10240,
		WriteBytes: 5120,
		IoTime:     100,
		WeightedIO: 200,
		ReadTime:   60,
		WriteTime:  40,
	}
	last := disk.IOCountersStat{
		ReadCount:  100,
		WriteCount: 50,
		ReadBytes:  5120,
		WriteBytes: 2560,
		IoTime:     50,
		WeightedIO: 100,
		ReadTime:   30,
		WriteTime:  20,
	}

	stat := &DiskIOStat{Name: "sdc"}
	calcDiskIORates(stat, current, last, 0.5)

	assert.InDelta(t, 20.0, stat.ReadIOPS, 0.1)
	assert.InDelta(t, 10.0, stat.WriteIOPS, 0.1)
}

// === calcDiskIOMacOS 测试 ===

func TestCalcDiskIOMacOS_ZeroWeightedIO(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS specific test")
	}
	current := disk.IOCountersStat{IoTime: 0, WeightedIO: 0}
	last := disk.IOCountersStat{IoTime: 0, WeightedIO: 0}

	stat := &DiskIOStat{ReadIOPS: 100, WriteIOPS: 50}
	calcDiskIOMacOS(stat, current, last, 1.0)

	assert.Equal(t, 0.0, stat.AvgQuSz)
	assert.Equal(t, 0.0, stat.Await)
	assert.Equal(t, 0.0, stat.UtilPercent)
}

func TestCalcDiskIOMacOS_WithIO(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS specific test")
	}
	current := disk.IOCountersStat{
		IoTime:     100000,
		WeightedIO: 200000,
		ReadTime:   60000,
		WriteTime:  40000,
	}
	last := disk.IOCountersStat{
		IoTime:     50000,
		WeightedIO: 100000,
		ReadTime:   30000,
		WriteTime:  20000,
	}

	stat := &DiskIOStat{ReadIOPS: 100, WriteIOPS: 50}
	calcDiskIOMacOS(stat, current, last, 1.0)

	assert.Greater(t, stat.AvgQuSz, 0.0)
	assert.Greater(t, stat.Await, 0.0)
	assert.Greater(t, stat.UtilPercent, 0.0)
	assert.Greater(t, stat.RAwait, 0.0)
	assert.Greater(t, stat.WAwait, 0.0)
}

// === calcDiskIOLinux 测试 ===

func TestCalcDiskIOLinux_Basic(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Linux specific test")
	}
	current := disk.IOCountersStat{
		IoTime:     100000,
		WeightedIO: 200000,
		ReadTime:   60000,
		WriteTime:  40000,
	}
	last := disk.IOCountersStat{
		IoTime:     50000,
		WeightedIO: 100000,
		ReadTime:   30000,
		WriteTime:  20000,
	}

	stat := &DiskIOStat{ReadIOPS: 100, WriteIOPS: 50}
	calcDiskIOLinux(stat, current, last, 1.0)

	assert.InDelta(t, 100.0, stat.AvgQuSz, 0.1)
	assert.Greater(t, stat.Await, 0.0)
	assert.Greater(t, stat.RAwait, 0.0)
	assert.Greater(t, stat.WAwait, 0.0)
	assert.Greater(t, stat.UtilPercent, 0.0)
}

func TestCalcDiskIOLinux_UtilCapped(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Linux specific test")
	}
	// UtilPercent should be capped at 100
	current := disk.IOCountersStat{
		IoTime:     1000000,
		WeightedIO: 2000000,
		ReadTime:   600000,
		WriteTime:  400000,
	}
	last := disk.IOCountersStat{
		IoTime:     0,
		WeightedIO: 0,
		ReadTime:   0,
		WriteTime:  0,
	}

	stat := &DiskIOStat{ReadIOPS: 0, WriteIOPS: 0}
	calcDiskIOLinux(stat, current, last, 1.0)

	assert.LessOrEqual(t, stat.UtilPercent, 100.0)
}

// === getTotalProcesses 测试 ===

func TestGetTotalProcesses_Normal(t *testing.T) {
	assert.Equal(t, 200, getTotalProcesses(200))
}

func TestGetTotalProcesses_Zero(t *testing.T) {
	assert.Equal(t, 0, getTotalProcesses(0))
}

func TestGetTotalProcesses_Negative(t *testing.T) {
	assert.Equal(t, -1, getTotalProcesses(-1))
	assert.Equal(t, -1, getTotalProcesses(-100))
}

// === getCachedMaxUserProcesses 测试 ===

func TestGetCachedMaxUserProcesses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dc := NewDataCollector(ctx, 1*time.Second)

	// First call should execute and cache
	result1 := dc.getCachedMaxUserProcesses()
	// Second call should return cached value (same)
	result2 := dc.getCachedMaxUserProcesses()
	assert.Equal(t, result1, result2)

	// Should return a reasonable value: either -1 (unlimited/error) or a positive number
	assert.True(t, result1 == -1 || result1 > 0)
}

// === collectCPUData 集成测试 ===

func TestCollectCPUData_FirstCollect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dc := NewDataCollector(ctx, 1*time.Second)
	assert.True(t, dc.firstCollect)

	// Skip if cpu.Times is unavailable (e.g. CGO_ENABLED=0 on macOS)
	times, err := cpu.Times(true)
	if err != nil || len(times) == 0 {
		t.Skip("cpu.Times unavailable in this environment")
	}

	// First collect should save baseline and return empty stats
	result := dc.collectCPUData()
	// On first call, lastCPUTimes is empty, so we just save baseline
	// cpuCores should be set
	assert.GreaterOrEqual(t, result.cpuCores, 0)

	// lastCPUTimes should now be populated
	assert.NotEmpty(t, dc.lastCPUTimes)
}

func TestCollectCPUData_SecondCollect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Skip if cpu.Times is unavailable
	times, err := cpu.Times(true)
	if err != nil || len(times) == 0 {
		t.Skip("cpu.Times unavailable in this environment")
	}

	dc := NewDataCollector(ctx, 1*time.Second)

	// First collect to establish baseline
	_ = dc.collectCPUData()
	assert.NotEmpty(t, dc.lastCPUTimes)

	// Wait a tiny bit for CPU times to change
	time.Sleep(100 * time.Millisecond)

	// Second collect should produce actual data
	result := dc.collectCPUData()
	// Should have results now (may be empty if delta is too small, but cores should be set)
	assert.GreaterOrEqual(t, result.cpuCores, 0)
}

func TestCollectCPUData_CoreCountMismatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dc := NewDataCollector(ctx, 1*time.Second)

	// Manually set lastCPUTimes with wrong length
	times, err := cpu.Times(true)
	if err != nil || len(times) == 0 {
		t.Skip("Cannot get CPU times")
	}

	// Set an intentionally wrong length
	dc.lastCPUTimes = []cpu.TimesStat{{CPU: "cpu0"}}
	dc.firstCollect = false

	result := dc.collectCPUData()
	// Should handle mismatch gracefully
	_ = result
}

// === collectDiskIO 测试 ===

func TestCollectDiskIO_FirstCollect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dc := NewDataCollector(ctx, 1*time.Second)

	stats := dc.collectDiskIO()
	// Should return a slice (may be empty on some systems)
	assert.NotNil(t, stats)

	// After first collect, lastDiskIO should be populated
	assert.NotNil(t, dc.lastDiskIO)
}

func TestCollectDiskIO_SecondCollect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dc := NewDataCollector(ctx, 1*time.Second)

	// First collect
	_ = dc.collectDiskIO()

	// Second collect should produce rate calculations
	time.Sleep(100 * time.Millisecond)
	stats := dc.collectDiskIO()
	assert.NotNil(t, stats)

	// Stats should have names
	for _, s := range stats {
		assert.NotEmpty(t, s.Name)
	}
}

// === collect 方法集成测试 ===

func TestCollect_Integration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dc := NewDataCollector(ctx, 1*time.Second)

	// First collect
	data := dc.collect()
	assert.NotNil(t, data)
	assert.False(t, data.UpdateTime.IsZero())
	assert.Equal(t, runtime.GOOS == "darwin", data.ProcessIOUnsupported)
	assert.Equal(t, runtime.GOOS == "darwin", data.DiskIOLimited)

	// Platform should be set
	assert.NotEmpty(t, data.Platform)
}

func TestCollect_SecondCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dc := NewDataCollector(ctx, 1*time.Second)

	// First collect
	_ = dc.collect()
	time.Sleep(200 * time.Millisecond)

	// Second collect
	data := dc.collect()
	assert.NotNil(t, data)
	assert.NotEmpty(t, data.Platform)
	// MaxUserProcesses should be populated via cache
	assert.True(t, data.MaxUserProcesses == -1 || data.MaxUserProcesses > 0)
}
