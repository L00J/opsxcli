package sys

import (
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/stretchr/testify/assert"
)

// --- cpuTotalTime (collector_cpu.go) ---

func TestCPUTotalTime_AllFields(t *testing.T) {
	stat := cpu.TimesStat{
		User: 10, System: 5, Idle: 80, Nice: 2, Iowait: 1,
		Irq: 0.5, Softirq: 0.5, Steal: 0, Guest: 0.5, GuestNice: 0.5,
	}
	total := cpuTotalTime(stat)
	assert.InDelta(t, 100.0, total, 0.01)
}

func TestCPUTotalTime_Zero(t *testing.T) {
	stat := cpu.TimesStat{}
	total := cpuTotalTime(stat)
	assert.Equal(t, 0.0, total)
}

// --- cpuAverage (collector_cpu.go) ---

func TestCPUAverage_Single(t *testing.T) {
	stats := []CPUTimesStat{
		{User: 10, System: 5, Idle: 80},
	}
	avg := cpuAverage(stats)
	assert.Equal(t, 10.0, avg.User)
	assert.Equal(t, 5.0, avg.System)
	assert.Equal(t, "all", avg.CPU)
}

func TestCPUAverage_Multiple(t *testing.T) {
	stats := []CPUTimesStat{
		{User: 10, System: 5, Idle: 80},
		{User: 20, System: 10, Idle: 60},
	}
	avg := cpuAverage(stats)
	assert.Equal(t, 15.0, avg.User)
	assert.Equal(t, 7.5, avg.System)
}

func TestCPUAverage_Three(t *testing.T) {
	stats := []CPUTimesStat{
		{User: 30, System: 10, Idle: 50},
		{User: 60, System: 20, Idle: 100},
		{User: 90, System: 30, Idle: 150},
	}
	avg := cpuAverage(stats)
	assert.Equal(t, 60.0, avg.User)
	assert.Equal(t, 20.0, avg.System)
}

// --- extractDeviceName (collector_disk.go) ---

func TestExtractDeviceName_Partition(t *testing.T) {
	// /dev/sda1 -> strips "1" -> "sda"
	assert.Equal(t, "sda", extractDeviceName("/dev/sda1"))
}

func TestExtractDeviceName_NoPath(t *testing.T) {
	assert.Equal(t, "sda", extractDeviceName("sda"))
}

func TestExtractDeviceName_Mapper(t *testing.T) {
	// mapper/vg-lv doesn't end in digit, so stays as-is (minus /dev/)
	result := extractDeviceName("/dev/mapper/vg-lv")
	if runtime.GOOS == "darwin" {
		// on darwin, no special handling for mapper
		assert.Contains(t, result, "mapper")
	} else {
		assert.Contains(t, result, "mapper")
	}
}

// --- detectDiskType (collector_disk.go) ---

func TestDetectDiskType_NVMe(t *testing.T) {
	assert.Equal(t, "NVMe SSD", detectDiskType("nvme0n1", ""))
}

func TestDetectDiskType_RAID(t *testing.T) {
	assert.Equal(t, "RAID", detectDiskType("md0", ""))
}

func TestDetectDiskType_DefaultHDD(t *testing.T) {
	assert.Equal(t, "HDD", detectDiskType("sda", "ext4"))
}
