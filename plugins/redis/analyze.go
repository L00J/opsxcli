package redis

// analyze.go — Redis 内存分析与键空间诊断
//
// 提供纯函数实现，便于单元测试。
// 需要实际 Redis 连接的功能通过后续 DBQuerier 接口解耦。

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ==================== 数据结构 ====================

// MemoryInfo 表示 INFO memory 命令返回的内存信息
type MemoryInfo struct {
	UsedMemoryBytes       int64   `json:"used_memory_bytes"`
	UsedMemoryHuman       string  `json:"used_memory_human"`
	UsedMemoryPeakBytes   int64   `json:"used_memory_peak_bytes"`
	UsedMemoryPeakHuman   string  `json:"used_memory_peak_human"`
	UsedMemoryRSS         int64   `json:"used_memory_rss"`
	TotalSystemMemory     int64   `json:"total_system_memory"`
	MaxMemoryBytes        int64   `json:"max_memory_bytes"`
	MaxMemoryPolicy       string  `json:"max_memory_policy"`
	MemFragmentationRatio float64 `json:"mem_fragmentation_ratio"`
	MemFragmentationBytes int64   `json:"mem_fragmentation_bytes"`
	UsedMemoryDataset     int64   `json:"used_memory_dataset"`
	UsedMemoryOverhead    int64   `json:"used_memory_overhead"`
}

// DBKeySpace 表示单个数据库的键空间信息（来自 INFO keyspace）
type DBKeySpace struct {
	DBName  string `json:"db"`
	Keys    int64  `json:"keys"`
	Expires int64  `json:"expires"`
	AvgTTL  int64  `json:"avg_ttl_ms"`
}

// KeyDistribution 表示键类型分布统计
type KeyDistribution struct {
	TypeName  string `json:"type"`
	Count     int64  `json:"count"`
	TotalSize int64  `json:"total_size_bytes"`
}

// MemoryWarning 表示内存告警信息
type MemoryWarning struct {
	Level   string  `json:"level"` // 正常/低风险/中风险/高风险/严重
	Message string  `json:"message"`
	Metric  string  `json:"metric"`
	Value   float64 `json:"value"`
}

// SlowLogEntry 表示 SLOWLOG GET 的一条记录
type SlowLogEntry struct {
	ID         int64  `json:"id"`
	StartTime  int64  `json:"start_time"`  // Unix 时间戳
	Duration   int64  `json:"duration_us"` // 微秒
	Command    string `json:"command"`
	ClientIP   string `json:"client_ip"`
	ClientPort int    `json:"client_port"`
}

// MemoryReport 是完整的内存分析报告
type MemoryReport struct {
	MemoryInfo       *MemoryInfo     `json:"memory_info"`
	KeySpaces        []DBKeySpace    `json:"keyspaces"`
	Warnings         []MemoryWarning `json:"warnings"`
	MemoryEfficiency float64         `json:"memory_efficiency"` // 字节/键
	TotalKeys        int64           `json:"total_keys"`
	TotalExpires     int64           `json:"total_expires"`
}

// ==================== 纯函数：INFO memory 解析 ====================

// ParseMemoryInfo 从 INFO memory 的原始文本中提取内存信息
// 输入格式: "used_memory:1048576\nused_memory_human:1.00M\n..."
func ParseMemoryInfo(raw string) *MemoryInfo {
	info := &MemoryInfo{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		// 跳过注释行和空行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "used_memory":
			info.UsedMemoryBytes = parseInt64Safe(value)
		case "used_memory_human":
			info.UsedMemoryHuman = value
		case "used_memory_peak":
			info.UsedMemoryPeakBytes = parseInt64Safe(value)
		case "used_memory_peak_human":
			info.UsedMemoryPeakHuman = value
		case "used_memory_rss":
			info.UsedMemoryRSS = parseInt64Safe(value)
		case "total_system_memory":
			info.TotalSystemMemory = parseInt64Safe(value)
		case "maxmemory":
			info.MaxMemoryBytes = parseInt64Safe(value)
		case "maxmemory_policy":
			info.MaxMemoryPolicy = value
		case "mem_fragmentation_ratio":
			info.MemFragmentationRatio = parseFloatSafe(value)
		case "mem_fragmentation_bytes":
			info.MemFragmentationBytes = parseInt64Safe(value)
		case "used_memory_dataset":
			info.UsedMemoryDataset = parseInt64Safe(value)
		case "used_memory_overhead":
			info.UsedMemoryOverhead = parseInt64Safe(value)
		}
	}
	return info
}

// ==================== 纯函数：键空间解析 ====================

// ParseKeySpace 从 INFO keyspace 的原始文本中提取各数据库键统计
// 输入格式: "db0:keys=1000,expires=500,avg_ttl=3600000\ndb1:keys=200,..."
func ParseKeySpace(raw string) []DBKeySpace {
	var result []DBKeySpace
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		// 跳过注释行和空行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 格式: db0:keys=1000,expires=500,avg_ttl=3600000
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		dbName := line[:colonIdx]
		// 检查是否以 "db" 开头
		if !strings.HasPrefix(dbName, "db") {
			continue
		}

		ks := DBKeySpace{DBName: dbName}

		// 解析键值对
		params := strings.Split(line[colonIdx+1:], ",")
		for _, param := range params {
			kv := strings.SplitN(param, "=", 2)
			if len(kv) != 2 {
				continue
			}
			switch strings.TrimSpace(kv[0]) {
			case "keys":
				ks.Keys = parseInt64Safe(strings.TrimSpace(kv[1]))
			case "expires":
				ks.Expires = parseInt64Safe(strings.TrimSpace(kv[1]))
			case "avg_ttl":
				ks.AvgTTL = parseInt64Safe(strings.TrimSpace(kv[1]))
			}
		}

		result = append(result, ks)
	}
	return result
}

// ==================== 纯函数：内存健康评估 ====================

// AnalyzeMemoryHealth 根据内存信息生成告警列表
func AnalyzeMemoryHealth(info *MemoryInfo) []MemoryWarning {
	if info == nil {
		return nil
	}

	var warnings []MemoryWarning

	// 检查1: 未设置 maxmemory
	if info.MaxMemoryBytes == 0 {
		warnings = append(warnings, MemoryWarning{
			Level:   "中风险",
			Message: "未设置 maxmemory，Redis 可能无限使用内存导致 OOM",
			Metric:  "maxmemory",
			Value:   0,
		})
	} else {
		// 检查2: 内存使用率
		usageRatio := float64(info.UsedMemoryBytes) / float64(info.MaxMemoryBytes) * 100
		switch {
		case usageRatio > 90:
			warnings = append(warnings, MemoryWarning{
				Level:   "严重",
				Message: fmt.Sprintf("内存使用率 %.1f%% 极高，即将触发淘汰策略", usageRatio),
				Metric:  "usage_ratio",
				Value:   usageRatio,
			})
		case usageRatio > 85:
			warnings = append(warnings, MemoryWarning{
				Level:   "高风险",
				Message: fmt.Sprintf("内存使用率 %.1f%% 过高，建议扩容或清理数据", usageRatio),
				Metric:  "usage_ratio",
				Value:   usageRatio,
			})
		case usageRatio > 70:
			warnings = append(warnings, MemoryWarning{
				Level:   "中风险",
				Message: fmt.Sprintf("内存使用率 %.1f%% 偏高，需关注增长趋势", usageRatio),
				Metric:  "usage_ratio",
				Value:   usageRatio,
			})
		}
	}

	// 检查3: 内存碎片率
	switch {
	case info.MemFragmentationRatio > 5.0:
		warnings = append(warnings, MemoryWarning{
			Level:   "严重",
			Message: fmt.Sprintf("内存碎片率 %.1f 极高，实际物理内存远超分配量", info.MemFragmentationRatio),
			Metric:  "fragmentation_ratio",
			Value:   info.MemFragmentationRatio,
		})
	case info.MemFragmentationRatio > 3.0:
		warnings = append(warnings, MemoryWarning{
			Level:   "高风险",
			Message: fmt.Sprintf("内存碎片率 %.1f 过高，建议执行 MEMORY PURGE 或重启", info.MemFragmentationRatio),
			Metric:  "fragmentation_ratio",
			Value:   info.MemFragmentationRatio,
		})
	case info.MemFragmentationRatio > 1.5:
		warnings = append(warnings, MemoryWarning{
			Level:   "低风险",
			Message: fmt.Sprintf("内存碎片率 %.1f 偏高，存在一定内存浪费", info.MemFragmentationRatio),
			Metric:  "fragmentation_ratio",
			Value:   info.MemFragmentationRatio,
		})
	}

	// 检查4: 内存使用超过系统内存 50%
	if info.TotalSystemMemory > 0 {
		systemRatio := float64(info.UsedMemoryBytes) / float64(info.TotalSystemMemory) * 100
		if systemRatio > 50 {
			warnings = append(warnings, MemoryWarning{
				Level:   "中风险",
				Message: fmt.Sprintf("Redis 内存占系统总内存 %.1f%%，可能影响其他进程", systemRatio),
				Metric:  "system_ratio",
				Value:   systemRatio,
			})
		}
	}

	// 检查5: 淘汰策略为 noeviction（写入将被拒绝）
	if info.MaxMemoryPolicy == "noeviction" {
		warnings = append(warnings, MemoryWarning{
			Level:   "中风险",
			Message: "淘汰策略为 noeviction，内存满时写入将被拒绝",
			Metric:  "eviction_policy",
			Value:   0,
		})
	}

	return warnings
}

// CalculateMemoryEfficiency 计算每键内存效率（字节/键）
func CalculateMemoryEfficiency(usedBytes, totalKeys int64) float64 {
	if totalKeys <= 0 || usedBytes <= 0 {
		return 0
	}
	return float64(usedBytes) / float64(totalKeys)
}

// ==================== 纯函数：键类型分布分析 ====================

// ClassifyKeyPatterns 按键名前缀分类统计键分布
// 输入键名列表和分隔符，返回按前缀分组的计数
func ClassifyKeyPatterns(keys []string, separator string) map[string]int64 {
	result := make(map[string]int64)
	for _, key := range keys {
		prefix := key
		if idx := strings.Index(key, separator); idx > 0 {
			prefix = key[:idx]
		}
		result[prefix]++
	}
	return result
}

// ==================== 纯函数：慢查询日志解析 ====================

// ParseSlowLogEntry 从 SLOWLOG GET 单条原始数据解析慢查询条目
// raw 是一个 []interface{} 格式: [id, start_time, duration, [args...], [ip:port]]
func ParseSlowLogEntry(raw []interface{}) *SlowLogEntry {
	if len(raw) < 3 {
		return nil
	}

	entry := &SlowLogEntry{}

	// 解析 ID
	entry.ID = toInt64(raw[0])

	// 解析开始时间
	entry.StartTime = toInt64(raw[1])

	// 解析执行耗时（微秒）
	entry.Duration = toInt64(raw[2])

	// 解析命令参数（raw[3] 是 []interface{}）
	if len(raw) > 3 {
		if args, ok := raw[3].([]interface{}); ok {
			var cmdParts []string
			for _, arg := range args {
				cmdParts = append(cmdParts, fmt.Sprintf("%v", arg))
			}
			entry.Command = strings.Join(cmdParts, " ")
		}
	}

	// 解析客户端地址（raw[4] 是 "ip:port" 格式）
	if len(raw) > 4 {
		if addr, ok := raw[4].(string); ok {
			entry.ClientIP, entry.ClientPort = parseAddr(addr)
		}
	}

	return entry
}

// ==================== 纯函数：报告格式化 ====================

// FormatMemoryReport 格式化内存分析报告为可读文本
func FormatMemoryReport(report *MemoryReport) string {
	if report == nil || report.MemoryInfo == nil {
		return "无内存数据"
	}

	var b strings.Builder
	info := report.MemoryInfo

	b.WriteString("=== Redis 内存分析报告 ===\n\n")

	// 内存使用
	fmt.Fprintf(&b, "已用内存:     %s (%d 字节)\n", info.UsedMemoryHuman, info.UsedMemoryBytes)
	fmt.Fprintf(&b, "内存峰值:     %s\n", info.UsedMemoryPeakHuman)

	if info.MaxMemoryBytes > 0 {
		usageRatio := float64(info.UsedMemoryBytes) / float64(info.MaxMemoryBytes) * 100
		fmt.Fprintf(&b, "内存限制:     %.0fMB (%s)\n",
			float64(info.MaxMemoryBytes)/1024/1024, info.MaxMemoryPolicy)
		fmt.Fprintf(&b, "使用率:       %.1f%%\n", usageRatio)
	} else {
		b.WriteString("内存限制:     未设置\n")
	}

	fmt.Fprintf(&b, "碎片率:       %.2f\n", info.MemFragmentationRatio)
	fmt.Fprintf(&b, "碎片大小:     %d 字节\n", info.MemFragmentationBytes)

	// 键空间概览
	if report.TotalKeys > 0 {
		fmt.Fprintf(&b, "\n总键数:       %d\n", report.TotalKeys)
		fmt.Fprintf(&b, "有过期时间:   %d (%.1f%%)\n",
			report.TotalExpires,
			float64(report.TotalExpires)/float64(report.TotalKeys)*100)
		fmt.Fprintf(&b, "内存效率:     %.1f 字节/键\n", report.MemoryEfficiency)
	}

	// 各数据库详情
	if len(report.KeySpaces) > 0 {
		b.WriteString("\n--- 键空间分布 ---\n")
		for _, ks := range report.KeySpaces {
			fmt.Fprintf(&b, "  %s: %d 键, %d 过期", ks.DBName, ks.Keys, ks.Expires)
			if ks.AvgTTL > 0 {
				fmt.Fprintf(&b, ", 平均TTL: %dms", ks.AvgTTL)
			}
			b.WriteString("\n")
		}
	}

	// 告警
	if len(report.Warnings) > 0 {
		fmt.Fprintf(&b, "\n--- 告警 (%d) ---\n", len(report.Warnings))
		for _, w := range report.Warnings {
			fmt.Fprintf(&b, "  [%s] %s\n", w.Level, w.Message)
		}
	}

	return b.String()
}

// FormatKeySpaceReport 格式化键空间分析报告
func FormatKeySpaceReport(keyspaces []DBKeySpace, totalKeys int64) string {
	if len(keyspaces) == 0 {
		return "无键空间数据"
	}

	var b strings.Builder
	b.WriteString("=== 键空间分析 ===\n\n")
	fmt.Fprintf(&b, "总键数: %d\n\n", totalKeys)

	for _, ks := range keyspaces {
		pct := float64(0)
		if totalKeys > 0 {
			pct = float64(ks.Keys) / float64(totalKeys) * 100
		}
		fmt.Fprintf(&b, "%s: %d 键 (%.1f%%), %d 过期",
			ks.DBName, ks.Keys, pct, ks.Expires)
		if ks.AvgTTL > 0 {
			fmt.Fprintf(&b, ", 平均TTL: %dms", ks.AvgTTL)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// FormatSlowLogReport 格式化慢查询日志报告
func FormatSlowLogReport(entries []SlowLogEntry, topN int) string {
	if len(entries) == 0 {
		return "无慢查询记录"
	}

	// 按耗时降序排序
	sorted := make([]SlowLogEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Duration > sorted[j].Duration
	})

	// 取 Top N
	if topN > 0 && topN < len(sorted) {
		sorted = sorted[:topN]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "=== 慢查询日志 (Top %d) ===\n\n", topN)

	for i, entry := range sorted {
		durationMs := float64(entry.Duration) / 1000.0
		fmt.Fprintf(&b, "[%d] 耗时: %.1fms | 命令: %s\n", i+1, durationMs, entry.Command)
		if entry.ClientIP != "" {
			fmt.Fprintf(&b, "    客户端: %s:%d\n", entry.ClientIP, entry.ClientPort)
		}
	}

	return b.String()
}

// ==================== 内部辅助函数 ====================

// parseInt64Safe 安全解析 int64，失败返回 0
func parseInt64Safe(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// parseFloatSafe 安全解析 float64，失败返回 0
func parseFloatSafe(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// toInt64 将 interface{} 转为 int64
func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case string:
		return parseInt64Safe(n)
	default:
		return 0
	}
}

// parseAddr 解析 "ip:port" 格式地址
func parseAddr(addr string) (host string, port int) {
	// 处理 IPv6 地址格式 [::1]:6379
	if strings.HasPrefix(addr, "[") {
		closeIdx := strings.Index(addr, "]")
		if closeIdx < 0 {
			return addr, 0
		}
		ip := addr[1:closeIdx]
		if closeIdx+2 < len(addr) && addr[closeIdx+1] == ':' {
			return ip, int(parseInt64Safe(addr[closeIdx+2:]))
		}
		return ip, 0
	}

	lastColon := strings.LastIndex(addr, ":")
	if lastColon < 0 {
		return addr, 0
	}
	return addr[:lastColon], int(parseInt64Safe(addr[lastColon+1:]))
}
