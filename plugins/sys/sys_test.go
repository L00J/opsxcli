package sys

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- formatBytes ---

func TestFormatBytes_Zero(t *testing.T) {
	assert.Equal(t, "0B", formatBytes(0))
}

func TestFormatBytes_Bytes(t *testing.T) {
	assert.Equal(t, "512B", formatBytes(512))
}

func TestFormatBytes_KiB(t *testing.T) {
	assert.Equal(t, "1.0KiB", formatBytes(1024))
}

func TestFormatBytes_MiB(t *testing.T) {
	assert.Equal(t, "1.5MiB", formatBytes(1.5*1024*1024))
}

func TestFormatBytes_GiB(t *testing.T) {
	assert.Equal(t, "2.0GiB", formatBytes(2.0*1024*1024*1024))
}

func TestFormatBytes_TiB(t *testing.T) {
	assert.Equal(t, "1.0TiB", formatBytes(1.0*1024*1024*1024*1024))
}

// --- formatBytesShort ---

func TestFormatBytesShort_Zero(t *testing.T) {
	assert.Equal(t, "0B", formatBytesShort(0))
}

func TestFormatBytesSmall(t *testing.T) {
	assert.Equal(t, "512B", formatBytesShort(512))
}

func TestFormatBytesShort_SmallKiB(t *testing.T) {
	// value=1.5, but exp=0 means the < 10 && exp > 0 check fails, so %.0f rounds up
	assert.Equal(t, "2K", formatBytesShort(1.5*1024))
}

func TestFormatBytesShort_LargeKiB(t *testing.T) {
	// >= 10 should show no decimal
	assert.Equal(t, "512K", formatBytesShort(512*1024))
}

func TestFormatBytesShort_SmallMiB(t *testing.T) {
	assert.Equal(t, "5.5M", formatBytesShort(5.5*1024*1024))
}

func TestFormatBytesShort_LargeMiB(t *testing.T) {
	assert.Equal(t, "512M", formatBytesShort(512*1024*1024))
}

func TestFormatBytesShort_GiB(t *testing.T) {
	// 2.0GiB: value=2.0, exp=2, so < 10 && exp > 0 => one decimal
	assert.Equal(t, "2.0G", formatBytesShort(2.0*1024*1024*1024))
}

// --- truncate ---

func TestTruncate_Short(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
}

func TestTruncate_Exact(t *testing.T) {
	assert.Equal(t, "12345", truncate("12345", 5))
}

func TestTruncate_Long(t *testing.T) {
	assert.Equal(t, "12...", truncate("1234567890", 5))
}

func TestTruncate_Empty(t *testing.T) {
	assert.Equal(t, "", truncate("", 5))
}

// --- formatCPUTime ---

func TestFormatCPUTime_Seconds(t *testing.T) {
	assert.Equal(t, "30.00s", formatCPUTime(30))
}

func TestFormatCPUTime_LessThanMinute(t *testing.T) {
	assert.Equal(t, "45.50s", formatCPUTime(45.5))
}

func TestFormatCPUTime_Minutes(t *testing.T) {
	assert.Equal(t, "5m 30s", formatCPUTime(330))
}

func TestFormatCPUTime_Hours(t *testing.T) {
	assert.Equal(t, "2h 05m", formatCPUTime(7500))
}

func TestFormatCPUTime_ExactMinute(t *testing.T) {
	assert.Equal(t, "1m 00s", formatCPUTime(60))
}

func TestFormatCPUTime_ExactHour(t *testing.T) {
	assert.Equal(t, "1h 00m", formatCPUTime(3600))
}

// --- formatRuntime ---

func TestFormatRuntime_Seconds(t *testing.T) {
	assert.Equal(t, "30s", formatRuntime(30*time.Second))
}

func TestFormatRuntime_Minutes(t *testing.T) {
	assert.Equal(t, "5m", formatRuntime(5*time.Minute))
}

func TestFormatRuntime_Hours(t *testing.T) {
	assert.Equal(t, "2h5m", formatRuntime(2*time.Hour+5*time.Minute))
}

func TestFormatRuntime_ExactHour(t *testing.T) {
	assert.Equal(t, "1h0m", formatRuntime(1*time.Hour))
}

// --- formatDiskIO ---

func TestFormatDiskIO_BothZero(t *testing.T) {
	assert.Equal(t, "0/0", formatDiskIO(0, 0))
}

func TestFormatDiskIO_ReadOnly(t *testing.T) {
	result := formatDiskIO(1024, 0)
	assert.Contains(t, result, "1.0KiB")
}

func TestFormatDiskIO_WriteOnly(t *testing.T) {
	result := formatDiskIO(0, 2048)
	assert.Contains(t, result, "2.0KiB")
}

func TestFormatDiskIO_BothNonZero(t *testing.T) {
	result := formatDiskIO(1024, 2048)
	assert.Contains(t, result, "/")
}

// --- formatNumber ---

func TestFormatNumber(t *testing.T) {
	assert.Equal(t, "12345", formatNumber(12345))
}

func TestFormatNumber_Zero(t *testing.T) {
	assert.Equal(t, "0", formatNumber(0))
}

// --- TabType constants ---

func TestTabTypeConstants(t *testing.T) {
	assert.Equal(t, TabType(0), TabOverview)
	assert.Equal(t, TabType(1), TabProcesses)
	assert.Equal(t, TabType(2), TabCPU)
	assert.Equal(t, TabType(3), TabMemory)
	assert.Equal(t, TabType(4), TabDisk)
}
