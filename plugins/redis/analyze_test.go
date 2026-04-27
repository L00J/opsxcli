package redis

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== ParseMemoryInfo 测试 ====================

func TestParseMemoryInfo_完整数据(t *testing.T) {
	raw := `# Memory
used_memory:1048576
used_memory_human:1.00M
used_memory_peak:2097152
used_memory_peak_human:2.00M
used_memory_rss:3145728
total_system_memory:8589934592
maxmemory:4294967296
maxmemory_policy:allkeys-lru
mem_fragmentation_ratio:3.00
mem_fragmentation_bytes:2097152
used_memory_dataset:524288
used_memory_overhead:524288`

	info := ParseMemoryInfo(raw)
	assert.NotNil(t, info)
	assert.Equal(t, int64(1048576), info.UsedMemoryBytes)
	assert.Equal(t, "1.00M", info.UsedMemoryHuman)
	assert.Equal(t, int64(2097152), info.UsedMemoryPeakBytes)
	assert.Equal(t, "2.00M", info.UsedMemoryPeakHuman)
	assert.Equal(t, int64(3145728), info.UsedMemoryRSS)
	assert.Equal(t, int64(8589934592), info.TotalSystemMemory)
	assert.Equal(t, int64(4294967296), info.MaxMemoryBytes)
	assert.Equal(t, "allkeys-lru", info.MaxMemoryPolicy)
	assert.InDelta(t, 3.0, info.MemFragmentationRatio, 0.01)
	assert.Equal(t, int64(2097152), info.MemFragmentationBytes)
	assert.Equal(t, int64(524288), info.UsedMemoryDataset)
	assert.Equal(t, int64(524288), info.UsedMemoryOverhead)
}

func TestParseMemoryInfo_空输入(t *testing.T) {
	info := ParseMemoryInfo("")
	assert.NotNil(t, info)
	// 空输入应返回零值结构
	assert.Equal(t, int64(0), info.UsedMemoryBytes)
	assert.Equal(t, int64(0), info.UsedMemoryPeakBytes)
}

func TestParseMemoryInfo_部分数据(t *testing.T) {
	raw := `used_memory:2048
used_memory_human:2.00K
mem_fragmentation_ratio:1.5`

	info := ParseMemoryInfo(raw)
	assert.NotNil(t, info)
	assert.Equal(t, int64(2048), info.UsedMemoryBytes)
	assert.Equal(t, "2.00K", info.UsedMemoryHuman)
	assert.InDelta(t, 1.5, info.MemFragmentationRatio, 0.01)
	// 缺失字段应为零值
	assert.Equal(t, int64(0), info.TotalSystemMemory)
	assert.Equal(t, "", info.MaxMemoryPolicy)
}

func TestParseMemoryInfo_无maxmemory(t *testing.T) {
	// Redis 默认 maxmemory 为 0（不限制）
	raw := `used_memory:1048576
used_memory_human:1.00M
maxmemory:0
maxmemory_policy:noeviction`

	info := ParseMemoryInfo(raw)
	assert.NotNil(t, info)
	assert.Equal(t, int64(0), info.MaxMemoryBytes)
	assert.Equal(t, "noeviction", info.MaxMemoryPolicy)
}

// ==================== ParseKeySpace 测试 ====================

func TestParseKeySpace_多数据库(t *testing.T) {
	raw := `# Keyspace
db0:keys=1000,expires=500,avg_ttl=3600000
db1:keys=200,expires=100,avg_ttl=1800000`

	ks := ParseKeySpace(raw)
	assert.Len(t, ks, 2)

	assert.Equal(t, "db0", ks[0].DBName)
	assert.Equal(t, int64(1000), ks[0].Keys)
	assert.Equal(t, int64(500), ks[0].Expires)
	assert.Equal(t, int64(3600000), ks[0].AvgTTL)

	assert.Equal(t, "db1", ks[1].DBName)
	assert.Equal(t, int64(200), ks[1].Keys)
	assert.Equal(t, int64(100), ks[1].Expires)
	assert.Equal(t, int64(1800000), ks[1].AvgTTL)
}

func TestParseKeySpace_空输入(t *testing.T) {
	ks := ParseKeySpace("")
	assert.Empty(t, ks)
}

func TestParseKeySpace_仅注释行(t *testing.T) {
	raw := `# Keyspace
`
	ks := ParseKeySpace(raw)
	assert.Empty(t, ks)
}

func TestParseKeySpace_无过期键(t *testing.T) {
	raw := `db0:keys=500,expires=0,avg_ttl=0`

	ks := ParseKeySpace(raw)
	assert.Len(t, ks, 1)
	assert.Equal(t, int64(500), ks[0].Keys)
	assert.Equal(t, int64(0), ks[0].Expires)
	assert.Equal(t, int64(0), ks[0].AvgTTL)
}

// ==================== AnalyzeMemoryHealth 测试 ====================

func TestAnalyzeMemoryHealth_正常状态(t *testing.T) {
	info := &MemoryInfo{
		UsedMemoryBytes:       1024 * 1024 * 100,      // 100MB
		UsedMemoryPeakBytes:   1024 * 1024 * 200,      // 200MB
		TotalSystemMemory:     1024 * 1024 * 1024 * 8, // 8GB
		MaxMemoryBytes:        1024 * 1024 * 1024 * 2, // 2GB
		MaxMemoryPolicy:       "allkeys-lru",
		MemFragmentationRatio: 1.2,
	}

	warnings := AnalyzeMemoryHealth(info)
	// 正常状态不应有高级别告警
	for _, w := range warnings {
		assert.NotEqual(t, "严重", w.Level)
		assert.NotEqual(t, "高风险", w.Level)
	}
}

func TestAnalyzeMemoryHealth_内存使用率高(t *testing.T) {
	info := &MemoryInfo{
		UsedMemoryBytes:       1024 * 1024 * 1800, // 1.8GB
		MaxMemoryBytes:        1024 * 1024 * 2048, // 2GB
		MaxMemoryPolicy:       "allkeys-lru",
		MemFragmentationRatio: 1.1,
		TotalSystemMemory:     1024 * 1024 * 1024 * 8,
		UsedMemoryPeakBytes:   1024 * 1024 * 1900,
	}

	warnings := AnalyzeMemoryHealth(info)
	// 使用率 >85% 应有告警
	assert.NotEmpty(t, warnings)
	hasMemoryWarning := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "内存使用率") {
			hasMemoryWarning = true
			assert.Equal(t, "高风险", w.Level)
		}
	}
	assert.True(t, hasMemoryWarning, "应包含内存使用率告警")
}

func TestAnalyzeMemoryHealth_碎片率过高(t *testing.T) {
	info := &MemoryInfo{
		UsedMemoryBytes:       1024 * 1024 * 100,
		MaxMemoryBytes:        1024 * 1024 * 1024 * 2,
		MaxMemoryPolicy:       "allkeys-lru",
		MemFragmentationRatio: 5.0,
		MemFragmentationBytes: 1024 * 1024 * 400, // 400MB 碎片
		TotalSystemMemory:     1024 * 1024 * 1024 * 8,
		UsedMemoryPeakBytes:   1024 * 1024 * 150,
	}

	warnings := AnalyzeMemoryHealth(info)
	hasFragmentationWarning := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "碎片") {
			hasFragmentationWarning = true
			assert.Equal(t, "高风险", w.Level)
		}
	}
	assert.True(t, hasFragmentationWarning, "应包含内存碎片告警")
}

func TestAnalyzeMemoryHealth_无maxmemory限制(t *testing.T) {
	info := &MemoryInfo{
		UsedMemoryBytes:       1024 * 1024 * 500,
		MaxMemoryBytes:        0, // 无限制
		MaxMemoryPolicy:       "noeviction",
		MemFragmentationRatio: 1.1,
		TotalSystemMemory:     1024 * 1024 * 1024 * 8,
		UsedMemoryPeakBytes:   1024 * 1024 * 600,
	}

	warnings := AnalyzeMemoryHealth(info)
	hasNoMaxMemory := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "maxmemory") || strings.Contains(w.Message, "未设置") {
			hasNoMaxMemory = true
		}
	}
	assert.True(t, hasNoMaxMemory, "应警告未设置 maxmemory")
}

func TestAnalyzeMemoryHealth_nil输入(t *testing.T) {
	warnings := AnalyzeMemoryHealth(nil)
	assert.Empty(t, warnings)
}

// ==================== CalculateMemoryEfficiency 测试 ====================

func TestCalculateMemoryEfficiency_正常计算(t *testing.T) {
	// 1MB 用于 1000 个键 = 1048.576 字节/键
	eff := CalculateMemoryEfficiency(1024*1024, 1000)
	assert.InDelta(t, 1048.576, eff, 0.1)
}

func TestCalculateMemoryEfficiency_零键(t *testing.T) {
	eff := CalculateMemoryEfficiency(1024*1024, 0)
	assert.Equal(t, float64(0), eff)
}

func TestCalculateMemoryEfficiency_零内存(t *testing.T) {
	eff := CalculateMemoryEfficiency(0, 1000)
	assert.Equal(t, float64(0), eff)
}

// ==================== ClassifyKeyPatterns 测试 ====================

func TestClassifyKeyPatterns_按冒号分类(t *testing.T) {
	keys := []string{
		"user:1001",
		"user:1002",
		"user:1003",
		"session:abc",
		"session:def",
		"cache:product:1",
		"order:2001",
	}

	result := ClassifyKeyPatterns(keys, ":")
	assert.Equal(t, int64(3), result["user"])
	assert.Equal(t, int64(2), result["session"])
	assert.Equal(t, int64(1), result["cache"])
	assert.Equal(t, int64(1), result["order"])
}

func TestClassifyKeyPatterns_空列表(t *testing.T) {
	result := ClassifyKeyPatterns([]string{}, ":")
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestClassifyKeyPatterns_无分隔符(t *testing.T) {
	keys := []string{"simplekey1", "simplekey2"}
	result := ClassifyKeyPatterns(keys, ":")
	// 无分隔符时以完整键名作为分类
	assert.Equal(t, int64(1), result["simplekey1"])
	assert.Equal(t, int64(1), result["simplekey2"])
}

func TestClassifyKeyPatterns_嵌套前缀(t *testing.T) {
	keys := []string{
		"app:user:profile:1",
		"app:user:settings:1",
		"app:product:detail:1",
	}

	result := ClassifyKeyPatterns(keys, ":")
	assert.Equal(t, int64(3), result["app"])
}

// ==================== ParseSlowLogEntry 测试 ====================

func TestParseSlowLogEntry_标准格式(t *testing.T) {
	raw := []interface{}{
		int64(42),
		int64(1700000000),
		int64(15000), // 15ms
		[]interface{}{"GET", "user:1001"},
		"127.0.0.1:54321",
	}

	entry := ParseSlowLogEntry(raw)
	assert.NotNil(t, entry)
	assert.Equal(t, int64(42), entry.ID)
	assert.Equal(t, int64(1700000000), entry.StartTime)
	assert.Equal(t, int64(15000), entry.Duration)
	assert.Equal(t, "GET user:1001", entry.Command)
	assert.Equal(t, "127.0.0.1", entry.ClientIP)
	assert.Equal(t, 54321, entry.ClientPort)
}

func TestParseSlowLogEntry_复杂命令(t *testing.T) {
	raw := []interface{}{
		int64(100),
		int64(1700000000),
		int64(50000), // 50ms
		[]interface{}{"HMSET", "user:1001", "name", "张三", "age", "30"},
		"10.0.0.1:12345",
	}

	entry := ParseSlowLogEntry(raw)
	assert.NotNil(t, entry)
	assert.Equal(t, int64(100), entry.ID)
	assert.Equal(t, int64(50000), entry.Duration)
	assert.Contains(t, entry.Command, "HMSET")
	assert.Contains(t, entry.Command, "user:1001")
	assert.Equal(t, "10.0.0.1", entry.ClientIP)
}

func TestParseSlowLogEntry_空数据(t *testing.T) {
	entry := ParseSlowLogEntry([]interface{}{})
	assert.Nil(t, entry)
}

func TestParseSlowLogEntry_字段不足(t *testing.T) {
	raw := []interface{}{int64(1)}
	entry := ParseSlowLogEntry(raw)
	assert.Nil(t, entry)
}

// ==================== FormatMemoryReport 测试 ====================

func TestFormatMemoryReport_完整报告(t *testing.T) {
	report := &MemoryReport{
		MemoryInfo: &MemoryInfo{
			UsedMemoryBytes:       1024 * 1024 * 100,
			UsedMemoryHuman:       "100.00M",
			UsedMemoryPeakBytes:   1024 * 1024 * 200,
			UsedMemoryPeakHuman:   "200.00M",
			MaxMemoryBytes:        1024 * 1024 * 1024 * 2,
			MaxMemoryPolicy:       "allkeys-lru",
			MemFragmentationRatio: 1.2,
		},
		KeySpaces: []DBKeySpace{
			{DBName: "db0", Keys: 10000, Expires: 5000, AvgTTL: 3600000},
		},
		MemoryEfficiency: 10485.76,
		TotalKeys:        10000,
		TotalExpires:     5000,
		Warnings: []MemoryWarning{
			{Level: "正常", Message: "内存使用正常", Metric: "usage_ratio", Value: 4.88},
		},
	}

	output := FormatMemoryReport(report)
	assert.Contains(t, output, "Redis 内存分析报告")
	assert.Contains(t, output, "100.00M")
	assert.Contains(t, output, "allkeys-lru")
	assert.Contains(t, output, "db0")
	assert.Contains(t, output, "10000")
}

func TestFormatMemoryReport_nil报告(t *testing.T) {
	output := FormatMemoryReport(nil)
	assert.Contains(t, output, "无内存数据")
}

// ==================== FormatKeySpaceReport 测试 ====================

func TestFormatKeySpaceReport_多数据库(t *testing.T) {
	keyspaces := []DBKeySpace{
		{DBName: "db0", Keys: 10000, Expires: 5000, AvgTTL: 3600000},
		{DBName: "db1", Keys: 2000, Expires: 1000, AvgTTL: 1800000},
	}

	output := FormatKeySpaceReport(keyspaces, 12000)
	assert.Contains(t, output, "键空间分析")
	assert.Contains(t, output, "db0")
	assert.Contains(t, output, "db1")
	assert.Contains(t, output, "10000")
	assert.Contains(t, output, "12000")
}

func TestFormatKeySpaceReport_空数据(t *testing.T) {
	output := FormatKeySpaceReport([]DBKeySpace{}, 0)
	assert.Contains(t, output, "无键空间数据")
}

// ==================== FormatSlowLogReport 测试 ====================

func TestFormatSlowLogReport_有数据(t *testing.T) {
	entries := []SlowLogEntry{
		{ID: 1, StartTime: 1700000000, Duration: 15000, Command: "GET user:1001", ClientIP: "127.0.0.1", ClientPort: 54321},
		{ID: 2, StartTime: 1700000001, Duration: 50000, Command: "HMSET user:1002 name test", ClientIP: "10.0.0.1", ClientPort: 12345},
		{ID: 3, StartTime: 1700000002, Duration: 8000, Command: "DEL session:old", ClientIP: "127.0.0.1", ClientPort: 54322},
	}

	output := FormatSlowLogReport(entries, 3)
	assert.Contains(t, output, "慢查询日志")
	assert.Contains(t, output, "GET user:1001")
	assert.Contains(t, output, "15.0ms")
	assert.Contains(t, output, "50.0ms")
}

func TestFormatSlowLogReport_TopN限制(t *testing.T) {
	entries := []SlowLogEntry{
		{ID: 1, Duration: 10000, Command: "CMD1"},
		{ID: 2, Duration: 50000, Command: "CMD2"},
		{ID: 3, Duration: 30000, Command: "CMD3"},
	}

	output := FormatSlowLogReport(entries, 2)
	// Top 2 应按耗时降序排列
	assert.Contains(t, output, "CMD2") // 50ms
	assert.Contains(t, output, "CMD3") // 30ms
	// CMD1 不在 Top 2 中
	assert.Equal(t, -1, strings.Index(output, "CMD1"))
}

func TestFormatSlowLogReport_空数据(t *testing.T) {
	output := FormatSlowLogReport([]SlowLogEntry{}, 10)
	assert.Contains(t, output, "无慢查询记录")
}

// ==================== 额外边界测试 ====================

func TestParseMemoryInfo_非法数值(t *testing.T) {
	raw := `used_memory:abc
used_memory_human:1.00M
mem_fragmentation_ratio:invalid`

	info := ParseMemoryInfo(raw)
	assert.NotNil(t, info)
	assert.Equal(t, int64(0), info.UsedMemoryBytes) // 解析失败应为 0
	assert.Equal(t, "1.00M", info.UsedMemoryHuman)
	assert.InDelta(t, 0.0, info.MemFragmentationRatio, 0.01) // 解析失败应为 0
}

func TestAnalyzeMemoryHealth_严重碎片率(t *testing.T) {
	info := &MemoryInfo{
		UsedMemoryBytes:       1024 * 1024 * 100,
		MaxMemoryBytes:        1024 * 1024 * 1024 * 2,
		MaxMemoryPolicy:       "allkeys-lru",
		MemFragmentationRatio: 8.0,
		MemFragmentationBytes: 1024 * 1024 * 700,
		TotalSystemMemory:     1024 * 1024 * 1024 * 8,
		UsedMemoryPeakBytes:   1024 * 1024 * 150,
	}

	warnings := AnalyzeMemoryHealth(info)
	// 碎片率 >5.0 应为严重级别
	hasCritical := false
	for _, w := range warnings {
		if w.Level == "严重" {
			hasCritical = true
		}
	}
	assert.True(t, hasCritical, "碎片率 8.0 应产生严重告警")
}

func TestAnalyzeMemoryHealth_使用率90以上(t *testing.T) {
	info := &MemoryInfo{
		UsedMemoryBytes:       1024 * 1024 * 1920, // 1.92GB
		MaxMemoryBytes:        1024 * 1024 * 2048, // 2GB
		MaxMemoryPolicy:       "volatile-lru",
		MemFragmentationRatio: 1.0,
		TotalSystemMemory:     1024 * 1024 * 1024 * 8,
		UsedMemoryPeakBytes:   1024 * 1024 * 2000,
	}

	warnings := AnalyzeMemoryHealth(info)
	hasCritical := false
	for _, w := range warnings {
		if w.Level == "严重" && strings.Contains(w.Message, "使用率") {
			hasCritical = true
		}
	}
	assert.True(t, hasCritical, "使用率 >90%% 应产生严重告警")
}

func TestParseSlowLogEntry_非字符串地址(t *testing.T) {
	raw := []interface{}{
		int64(1),
		int64(1700000000),
		int64(1000),
		[]interface{}{"PING"},
		int64(999), // 非字符串地址
	}

	entry := ParseSlowLogEntry(raw)
	assert.NotNil(t, entry)
	assert.Equal(t, int64(1), entry.ID)
	assert.Equal(t, "", entry.ClientIP) // 非字符串应跳过
	assert.Equal(t, 0, entry.ClientPort)
}

func TestParseSlowLogEntry_无命令参数(t *testing.T) {
	raw := []interface{}{
		int64(1),
		int64(1700000000),
		int64(1000),
	}

	entry := ParseSlowLogEntry(raw)
	assert.NotNil(t, entry)
	assert.Equal(t, "", entry.Command)
	assert.Equal(t, "", entry.ClientIP)
}

func TestCalculateMemoryEfficiency_大数据量(t *testing.T) {
	// 4GB / 1亿键 ≈ 42.9 字节/键
	eff := CalculateMemoryEfficiency(4*1024*1024*1024, 100000000)
	assert.InDelta(t, 42.94967296, eff, 0.01)
}

func TestClassifyKeyPatterns_大量键(t *testing.T) {
	keys := make([]string, 10000)
	for i := 0; i < 5000; i++ {
		keys[i] = fmt.Sprintf("user:%d", i)
	}
	for i := 5000; i < 10000; i++ {
		keys[i] = fmt.Sprintf("session:%d", i)
	}

	result := ClassifyKeyPatterns(keys, ":")
	assert.Equal(t, int64(5000), result["user"])
	assert.Equal(t, int64(5000), result["session"])
}

func TestParseKeySpace_格式异常(t *testing.T) {
	raw := `db0:keys=abc,expires=500,avg_ttl=3600000`
	ks := ParseKeySpace(raw)
	assert.Len(t, ks, 1)
	assert.Equal(t, "db0", ks[0].DBName)
	assert.Equal(t, int64(0), ks[0].Keys) // 解析失败应为 0
	assert.Equal(t, int64(500), ks[0].Expires)
}
