package sys

import (
	"testing"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/stretchr/testify/assert"
)

// ===================== calcDiskIOLinux 跨平台测试 =====================
// 注意：不使用 _linux 后缀的文件名，否则在 macOS 上会被 Go 忽略

func TestCalcDiskIOLinux_CrossPlatform(t *testing.T) {
	tests := []struct {
		name               string
		stat               *DiskIOStat
		current            disk.IOCountersStat
		last               disk.IOCountersStat
		deltaTime          float64
		wantAvgQuSz        float64
		wantUtilPercent    float64
		wantAwaitPositive  bool
		wantRAwaitPositive bool
		wantWAwaitPositive bool
	}{
		{
			name: "基本IO计算",
			stat: &DiskIOStat{
				ReadIOPS:  100.0,
				WriteIOPS: 50.0,
			},
			current: disk.IOCountersStat{
				ReadTime:   10000,
				WriteTime:  5000,
				IoTime:     20000,
				WeightedIO: 30000,
			},
			last: disk.IOCountersStat{
				ReadTime:   5000,
				WriteTime:  2000,
				IoTime:     10000,
				WeightedIO: 15000,
			},
			deltaTime:          1.0,
			wantAvgQuSz:        15.0,  // (30000-15000)/1.0/1000
			wantUtilPercent:    100.0, // (20000-10000)/1.0/10 = 1000, capped to 100
			wantAwaitPositive:  true,
			wantRAwaitPositive: true,
			wantWAwaitPositive: true,
		},
		{
			name: "无IO操作",
			stat: &DiskIOStat{
				ReadIOPS:  0.0,
				WriteIOPS: 0.0,
			},
			current: disk.IOCountersStat{
				ReadTime:   1000,
				WriteTime:  1000,
				IoTime:     1000,
				WeightedIO: 1000,
			},
			last: disk.IOCountersStat{
				ReadTime:   1000,
				WriteTime:  1000,
				IoTime:     1000,
				WeightedIO: 1000,
			},
			deltaTime:          1.0,
			wantAvgQuSz:        0.0,
			wantUtilPercent:    0.0,
			wantAwaitPositive:  false,
			wantRAwaitPositive: false,
			wantWAwaitPositive: false,
		},
		{
			name: "高利用率cap100",
			stat: &DiskIOStat{
				ReadIOPS:  1000.0,
				WriteIOPS: 1000.0,
			},
			current: disk.IOCountersStat{
				ReadTime:   50000,
				WriteTime:  50000,
				IoTime:     50000,
				WeightedIO: 200000,
			},
			last: disk.IOCountersStat{
				ReadTime:   0,
				WriteTime:  0,
				IoTime:     0,
				WeightedIO: 0,
			},
			deltaTime:          0.1,
			wantAvgQuSz:        2000.0, // 200000/0.1/1000
			wantUtilPercent:    100.0,  // capped
			wantAwaitPositive:  true,
			wantRAwaitPositive: true,
			wantWAwaitPositive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calcDiskIOLinux(tt.stat, tt.current, tt.last, tt.deltaTime)

			assert.InDelta(t, tt.wantAvgQuSz, tt.stat.AvgQuSz, 0.01, "AvgQuSz")
			assert.LessOrEqual(t, tt.stat.UtilPercent, 100.0, "UtilPercent should be capped at 100")

			if tt.wantUtilPercent > 0 {
				assert.Greater(t, tt.stat.UtilPercent, 0.0, "UtilPercent should be positive")
			}

			if tt.wantAwaitPositive {
				assert.Greater(t, tt.stat.Await, 0.0, "Await should be positive")
			}
			if tt.wantRAwaitPositive {
				assert.Greater(t, tt.stat.RAwait, 0.0, "RAwait should be positive")
			}
			if tt.wantWAwaitPositive {
				assert.Greater(t, tt.stat.WAwait, 0.0, "WAwait should be positive")
			}
		})
	}
}

func TestCalcDiskIOLinux_OnlyReads(t *testing.T) {
	stat := &DiskIOStat{
		ReadIOPS:  100.0,
		WriteIOPS: 0.0,
	}
	current := disk.IOCountersStat{
		ReadTime:   10000,
		WriteTime:  5000,
		IoTime:     15000,
		WeightedIO: 20000,
	}
	last := disk.IOCountersStat{
		ReadTime:   5000,
		WriteTime:  5000,
		IoTime:     10000,
		WeightedIO: 10000,
	}

	calcDiskIOLinux(stat, current, last, 1.0)

	// 只有读操作，RAwait 应该被计算
	assert.Greater(t, stat.RAwait, 0.0, "RAwait should be positive when ReadIOPS > 0")
	// WAwait 不应该被计算（WriteIOPS = 0）
	assert.Equal(t, 0.0, stat.WAwait, "WAwait should be 0 when WriteIOPS = 0")
	// Await 应该被计算（总IOPS > 0）
	assert.Greater(t, stat.Await, 0.0, "Await should be positive when total IOPS > 0")
}

func TestCalcDiskIOLinux_OnlyWrites(t *testing.T) {
	stat := &DiskIOStat{
		ReadIOPS:  0.0,
		WriteIOPS: 100.0,
	}
	current := disk.IOCountersStat{
		ReadTime:   5000,
		WriteTime:  10000,
		IoTime:     15000,
		WeightedIO: 20000,
	}
	last := disk.IOCountersStat{
		ReadTime:   5000,
		WriteTime:  5000,
		IoTime:     10000,
		WeightedIO: 10000,
	}

	calcDiskIOLinux(stat, current, last, 1.0)

	assert.Greater(t, stat.WAwait, 0.0, "WAwait should be positive when WriteIOPS > 0")
	assert.Equal(t, 0.0, stat.RAwait, "RAwait should be 0 when ReadIOPS = 0")
}

func TestCalcDiskIOLinux_UtilPercentExact(t *testing.T) {
	// 精确验证 UtilPercent 计算
	stat := &DiskIOStat{
		ReadIOPS:  500.0,
		WriteIOPS: 500.0,
	}
	current := disk.IOCountersStat{
		ReadTime:   10000,
		WriteTime:  10000,
		IoTime:     10000, // deltaTime=1s → (10000-5000)/1/10 = 500 → capped to 100
		WeightedIO: 20000,
	}
	last := disk.IOCountersStat{
		ReadTime:   5000,
		WriteTime:  5000,
		IoTime:     5000,
		WeightedIO: 10000,
	}

	calcDiskIOLinux(stat, current, last, 1.0)

	// UtilPercent = (10000-5000)/1.0/10.0 = 500 → capped to 100
	assert.Equal(t, 100.0, stat.UtilPercent, "UtilPercent should be capped at 100")
}

func TestCalcDiskIOLinux_SmallDelta(t *testing.T) {
	stat := &DiskIOStat{
		ReadIOPS:  100.0,
		WriteIOPS: 50.0,
	}
	current := disk.IOCountersStat{
		ReadTime:   5000,
		WriteTime:  3000,
		IoTime:     10000,
		WeightedIO: 15000,
	}
	last := disk.IOCountersStat{
		ReadTime:   4000,
		WriteTime:  2500,
		IoTime:     8000,
		WeightedIO: 12000,
	}

	calcDiskIOLinux(stat, current, last, 0.1)

	// AvgQuSz = (15000-12000)/0.1/1000 = 30.0
	assert.InDelta(t, 30.0, stat.AvgQuSz, 0.01, "AvgQuSz with small delta")
	// UtilPercent = (10000-8000)/0.1/10 = 2000 → capped to 100
	assert.Equal(t, 100.0, stat.UtilPercent, "UtilPercent capped with small delta")
}
