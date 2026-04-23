package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- parseSizeString ---

func TestParseSizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{name: "字节", input: "100B", want: 100},
		{name: "KiB", input: "10KiB", want: 10 * 1024},
		{name: "MiB", input: "100MiB", want: 100 * 1024 * 1024},
		{name: "GiB", input: "2GiB", want: 2 * 1024 * 1024 * 1024},
		{name: "TiB", input: "1TiB", want: 1 * 1024 * 1024 * 1024 * 1024},
		{name: "KB格式", input: "512KB", want: 512 * 1024},
		{name: "MB格式", input: "256MB", want: 256 * 1024 * 1024},
		{name: "GB格式", input: "1GB", want: 1 * 1024 * 1024 * 1024},
		{name: "带小数", input: "1.5GiB", want: int64(1.5 * float64(1024*1024*1024))},
		{name: "带空格", input: " 100MiB ", want: 100 * 1024 * 1024},
		{name: "空字符串", input: "", want: 0},
		{name: "无单位数字", input: "1024", want: 1024}, // 无单位按字节处理
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSizeString(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- getMapStr ---

func TestGetMapStr(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "字符串值",
			m:    map[string]interface{}{"Name": "nginx"},
			key:  "Name",
			want: "nginx",
		},
		{
			name: "非字符串值",
			m:    map[string]interface{}{"Count": float64(42)},
			key:  "Count",
			want: "",
		},
		{
			name: "key不存在",
			m:    map[string]interface{}{"Name": "nginx"},
			key:  "Missing",
			want: "",
		},
		{
			name: "nil map",
			m:    nil,
			key:  "Name",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMapStr(tt.m, tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- ContainerStats 结构体 ---

func TestContainerStatsDefaults(t *testing.T) {
	stats := &ContainerStats{}
	assert.Equal(t, "", stats.ContainerID)
	assert.Equal(t, "", stats.Name)
	assert.Equal(t, float64(0), stats.CPUPercent)
	assert.Equal(t, int64(0), stats.MemoryUsage)
	assert.Equal(t, int64(0), stats.MemoryLimit)
	assert.Equal(t, int64(0), stats.PIDs)
}

// --- StatsOptions ---

func TestStatsOptionsDefaults(t *testing.T) {
	opts := &StatsOptions{}
	assert.Empty(t, opts.Containers)
	assert.False(t, opts.NoStream)
	assert.False(t, opts.NoTrunc)
}

// --- ProcessInfo ---

func TestProcessInfo(t *testing.T) {
	proc := ProcessInfo{
		UID:  "root",
		PID:  "1234",
		PPID: "1",
		CMD:  "nginx -g daemon off;",
	}
	assert.Equal(t, "root", proc.UID)
	assert.Equal(t, "1234", proc.PID)
	assert.Equal(t, "1", proc.PPID)
	assert.Equal(t, "nginx -g daemon off;", proc.CMD)
}

// --- TopResult ---

func TestTopResult(t *testing.T) {
	result := &TopResult{
		ContainerID:   "abc123",
		ContainerName: "nginx",
		Processes: []ProcessInfo{
			{PID: "1", CMD: "nginx"},
			{PID: "7", CMD: "worker"},
		},
	}
	assert.Equal(t, "abc123", result.ContainerID)
	assert.Equal(t, "nginx", result.ContainerName)
	assert.Len(t, result.Processes, 2)
}

// --- EventsOptions ---

func TestEventsOptionsDefaults(t *testing.T) {
	opts := &EventsOptions{}
	assert.Equal(t, "", opts.Since)
	assert.Equal(t, "", opts.Until)
	assert.Equal(t, "", opts.Filters)
	assert.Equal(t, 0, opts.Duration)
}

// --- 错误路径 ---

func TestStats_NilOptions(t *testing.T) {
	// 验证 nil 选项不会 panic
	opts := &StatsOptions{}
	assert.False(t, opts.NoStream)
}

func TestTop_EmptyContainer(t *testing.T) {
	err := Top("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请指定")
}

func TestStatsJSON_EmptyContainer(t *testing.T) {
	_, err := StatsJSON("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请指定")
}
