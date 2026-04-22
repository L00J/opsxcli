package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================
// parseSizeString 扩展边界用例
// ============================

func TestParseSizeString_TB(t *testing.T) {
	// TB 单位
	result := parseSizeString("2TB")
	assert.Equal(t, int64(2*1024*1024*1024*1024), result)
}

func TestParseSizeString_ZeroWithUnit(t *testing.T) {
	// 零值带单位
	result := parseSizeString("0MiB")
	assert.Equal(t, int64(0), result)
}

func TestParseSizeString_LargeGiB(t *testing.T) {
	// 大数 GiB
	result := parseSizeString("100GiB")
	expected := int64(100) * 1024 * 1024 * 1024
	assert.Equal(t, expected, result)
}

func TestParseSizeString_FractionalKB(t *testing.T) {
	// 小数 KB
	result := parseSizeString("1.5KB")
	assert.Equal(t, int64(1.5*1024), result)
}

func TestParseSizeString_FractionalMB(t *testing.T) {
	// 小数 MB
	result := parseSizeString("0.5MB")
	assert.Equal(t, int64(0.5*1024*1024), result)
}

func TestParseSizeString_JustNumber(t *testing.T) {
	// 纯数字（无单位）— unit 为空，multiplier=1
	result := parseSizeString("512")
	assert.Equal(t, int64(512), result)
}

func TestParseSizeString_WhitespaceVariations(t *testing.T) {
	// 各种空格
	result := parseSizeString("  50MiB  ")
	assert.Equal(t, int64(50*1024*1024), result)
}

func TestParseSizeString_ZeroBytes(t *testing.T) {
	// 零字节
	result := parseSizeString("0B")
	assert.Equal(t, int64(0), result)
}

func TestParseSizeString_SmallValue(t *testing.T) {
	// 很小的值
	result := parseSizeString("1B")
	assert.Equal(t, int64(1), result)
}

func TestParseSizeString_VeryLargeTiB(t *testing.T) {
	// 很大的 TiB
	result := parseSizeString("10TiB")
	expected := int64(10) * 1024 * 1024 * 1024 * 1024
	assert.Equal(t, expected, result)
}

func TestParseSizeString_FractionalTiB(t *testing.T) {
	// 小数 TiB
	result := parseSizeString("0.25TiB")
	expected := int64(0.25 * float64(1024*1024*1024*1024))
	assert.Equal(t, expected, result)
}

func TestParseSizeString_UnknownUnit(t *testing.T) {
	// 未知单位 — multiplier=1，按原始值返回
	result := parseSizeString("100XYZ")
	assert.Equal(t, int64(100), result)
}

// ============================
// getMapStr 扩展用例
// ============================

func TestGetMapStr_EmptyStringValue(t *testing.T) {
	// 空字符串值
	m := map[string]interface{}{"Name": ""}
	result := getMapStr(m, "Name")
	assert.Equal(t, "", result)
}

func TestGetMapStr_IntValue(t *testing.T) {
	// int 值不是字符串
	m := map[string]interface{}{"Port": 8080}
	result := getMapStr(m, "Port")
	assert.Equal(t, "", result)
}

func TestGetMapStr_BoolValue(t *testing.T) {
	// bool 值不是字符串
	m := map[string]interface{}{"Active": true}
	result := getMapStr(m, "Active")
	assert.Equal(t, "", result)
}

func TestGetMapStr_SliceValue(t *testing.T) {
	// slice 值不是字符串
	m := map[string]interface{}{"Tags": []string{"latest"}}
	result := getMapStr(m, "Tags")
	assert.Equal(t, "", result)
}

func TestGetMapStr_NilValue(t *testing.T) {
	// nil 值
	m := map[string]interface{}{"Val": nil}
	result := getMapStr(m, "Val")
	assert.Equal(t, "", result)
}

func TestGetMapStr_EmptyMap(t *testing.T) {
	// 空 map
	m := map[string]interface{}{}
	result := getMapStr(m, "anything")
	assert.Equal(t, "", result)
}

func TestGetMapStr_MultipleKeys(t *testing.T) {
	// 多 key map 中取特定 key
	m := map[string]interface{}{
		"ID":     "abc123",
		"Name":   "mycontainer",
		"Status": "running",
	}
	assert.Equal(t, "abc123", getMapStr(m, "ID"))
	assert.Equal(t, "mycontainer", getMapStr(m, "Name"))
	assert.Equal(t, "running", getMapStr(m, "Status"))
	assert.Equal(t, "", getMapStr(m, "Missing"))
}

func TestGetMapStr_UnicodeKey(t *testing.T) {
	// Unicode key
	m := map[string]interface{}{"名称": "测试容器"}
	result := getMapStr(m, "名称")
	assert.Equal(t, "测试容器", result)
}

// ============================
// ContainerStats 扩展用例
// ============================

func TestContainerStats_NonZeroValues(t *testing.T) {
	// 非零值验证
	stats := &ContainerStats{
		ContainerID:   "abc123def456",
		Name:          "nginx-prod",
		CPUPercent:    12.5,
		MemoryUsage:   512 * 1024 * 1024,
		MemoryLimit:   2 * 1024 * 1024 * 1024,
		MemoryPercent: 25.0,
		NetIO:         "1.2kB / 3.4kB",
		BlockIO:       "5.6MB / 0B",
		PIDs:          42,
	}
	assert.Equal(t, "abc123def456", stats.ContainerID)
	assert.Equal(t, "nginx-prod", stats.Name)
	assert.Equal(t, float64(12.5), stats.CPUPercent)
	assert.Equal(t, int64(512*1024*1024), stats.MemoryUsage)
	assert.Equal(t, int64(2*1024*1024*1024), stats.MemoryLimit)
	assert.Equal(t, float64(25.0), stats.MemoryPercent)
	assert.Equal(t, "1.2kB / 3.4kB", stats.NetIO)
	assert.Equal(t, "5.6MB / 0B", stats.BlockIO)
	assert.Equal(t, int64(42), stats.PIDs)
}

// ============================
// StatsOptions 扩展用例
// ============================

func TestStatsOptions_WithContainers(t *testing.T) {
	// 指定容器列表
	opts := &StatsOptions{
		Containers: []string{"nginx", "redis"},
		NoStream:   true,
		NoTrunc:    false,
	}
	assert.Len(t, opts.Containers, 2)
	assert.Equal(t, "nginx", opts.Containers[0])
	assert.Equal(t, "redis", opts.Containers[1])
	assert.True(t, opts.NoStream)
	assert.False(t, opts.NoTrunc)
}

func TestStatsOptions_NoTrunc(t *testing.T) {
	// NoTrunc=true
	opts := &StatsOptions{NoTrunc: true}
	assert.True(t, opts.NoTrunc)
}

// ============================
// TopResult 扩展用例
// ============================

func TestTopResult_Empty(t *testing.T) {
	// 空 TopResult
	result := &TopResult{}
	assert.Equal(t, "", result.ContainerID)
	assert.Equal(t, "", result.ContainerName)
	assert.Empty(t, result.Processes)
}

func TestTopResult_MultipleProcesses(t *testing.T) {
	// 多进程
	result := &TopResult{
		ContainerID:   "abc123",
		ContainerName: "web",
		Processes: []ProcessInfo{
			{UID: "root", PID: "1", CMD: "nginx"},
			{UID: "www", PID: "7", CMD: "worker"},
			{UID: "www", PID: "8", CMD: "worker"},
		},
	}
	assert.Len(t, result.Processes, 3)
	assert.Equal(t, "root", result.Processes[0].UID)
	assert.Equal(t, "www", result.Processes[1].UID)
}

// ============================
// ProcessInfo 完整字段
// ============================

func TestProcessInfo_AllFields(t *testing.T) {
	// 所有字段
	proc := ProcessInfo{
		UID:   "root",
		PID:   "42",
		PPID:  "1",
		C:     "0.5",
		STIME: "10:30",
		TTY:   "pts/0",
		TIME:  "00:00:01",
		CMD:   "sleep 100",
	}
	assert.Equal(t, "root", proc.UID)
	assert.Equal(t, "42", proc.PID)
	assert.Equal(t, "1", proc.PPID)
	assert.Equal(t, "0.5", proc.C)
	assert.Equal(t, "10:30", proc.STIME)
	assert.Equal(t, "pts/0", proc.TTY)
	assert.Equal(t, "00:00:01", proc.TIME)
	assert.Equal(t, "sleep 100", proc.CMD)
}

// ============================
// EventsOptions 扩展用例
// ============================

func TestEventsOptions_WithValues(t *testing.T) {
	// 所有字段设置值
	opts := &EventsOptions{
		Since:    "2024-01-01",
		Until:    "2024-12-31",
		Filters:  "type=container",
		Duration: 60,
	}
	assert.Equal(t, "2024-01-01", opts.Since)
	assert.Equal(t, "2024-12-31", opts.Until)
	assert.Equal(t, "type=container", opts.Filters)
	assert.Equal(t, 60, opts.Duration)
}

// ============================
// parseSizeString 更多单位组合
// ============================

func TestParseSizeString_AllUnits(t *testing.T) {
	// 系统化验证所有单位
	tests := []struct {
		name       string
		input      string
		multiplier int64
		value      float64
	}{
		{name: "B", input: "5B", multiplier: 1, value: 5},
		{name: "KiB", input: "5KiB", multiplier: 1024, value: 5},
		{name: "KB", input: "5KB", multiplier: 1024, value: 5},
		{name: "MiB", input: "5MiB", multiplier: 1024 * 1024, value: 5},
		{name: "MB", input: "5MB", multiplier: 1024 * 1024, value: 5},
		{name: "GiB", input: "5GiB", multiplier: 1024 * 1024 * 1024, value: 5},
		{name: "GB", input: "5GB", multiplier: 1024 * 1024 * 1024, value: 5},
		{name: "TiB", input: "5TiB", multiplier: 1024 * 1024 * 1024 * 1024, value: 5},
		{name: "TB", input: "5TB", multiplier: 1024 * 1024 * 1024 * 1024, value: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := int64(tt.value * float64(tt.multiplier))
			result := parseSizeString(tt.input)
			assert.Equal(t, expected, result)
		})
	}
}

func TestParseSizeString_CaseSensitive(t *testing.T) {
	// 验证单位匹配是区分大小写的（小写应不匹配）
	result := parseSizeString("5mib")
	// "mib" 不是已知单位，multiplier=1，按原始值 5 返回
	assert.Equal(t, int64(5), result)
}

func TestParseSizeString_UnitWithExtraSpaces(t *testing.T) {
	// unit 前有空格（strings.TrimSpace 处理）
	result := parseSizeString("100 MiB")
	assert.Equal(t, int64(100*1024*1024), result)
}
