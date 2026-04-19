package logs

import (
	"sort"
	"strings"
	"time"
)

// CalculateStats 计算日志统计信息
func CalculateStats(entries []LogEntry, opts AnalyzeOptions) *LogStats {
	stats := &LogStats{
		StatusCodes: make(map[int]int),
	}

	if len(entries) == 0 {
		return stats
	}

	// 统计变量
	ipCount := make(map[string]int)
	urlCount := make(map[string]int)
	uaCount := make(map[string]int)
	errorIPCount := make(map[string]int)
	timeBucketCount := make(map[string]int)
	var totalResponseTime int64
	var responseCount int64
	var totalBytes int64
	var minTime, maxTime time.Time

	// 初始化时间范围
	minTime = entries[0].Timestamp
	maxTime = entries[0].Timestamp

	for _, entry := range entries {
		// IP 统计
		ipCount[entry.RemoteIP]++

		// URL 统计
		if entry.Path != "" {
			urlCount[entry.Path]++
		}

		// 状态码统计
		if entry.StatusCode > 0 {
			stats.StatusCodes[entry.StatusCode]++

			// 错误统计 (4xx/5xx)
			if entry.StatusCode >= 400 {
				stats.ErrorCount++
				errorIPCount[entry.RemoteIP]++
			}
		}

		// 响应时间统计
		if entry.ResponseTime > 0 {
			totalResponseTime += entry.ResponseTime
			responseCount++
		}

		// 字节数统计
		totalBytes += entry.BodyBytes

		// User-Agent 统计
		if entry.UserAgent != "" {
			ua := simplifyUA(entry.UserAgent)
			uaCount[ua]++
		}

		// 时间范围
		if entry.Timestamp.Before(minTime) {
			minTime = entry.Timestamp
		}
		if entry.Timestamp.After(maxTime) {
			maxTime = entry.Timestamp
		}

		// 时间分桶 (按小时)
		bucket := entry.Timestamp.Format("2006-01-02 15:00")
		timeBucketCount[bucket]++

		// 慢请求检测
		if entry.ResponseTime >= opts.SlowThreshold {
			stats.SlowRequests = append(stats.SlowRequests, entry)
		}
	}

	// 设置时间范围
	stats.TimeRange = TimeRange{Start: minTime, End: maxTime}

	// 平均响应时间
	if responseCount > 0 {
		stats.AvgResponseMs = totalResponseTime / responseCount
	}

	stats.TotalBytes = totalBytes

	// Top N IP
	stats.TopIPs = topNMap(ipCount, opts.TopN)

	// Top N URL
	stats.TopURLs = topNMapURL(urlCount, opts.TopN)

	// Top N User-Agent
	stats.UserAgents = topNMapUA(uaCount, opts.TopN)

	// 错误 IP 排名
	stats.ErrorIPs = topNMap(errorIPCount, opts.TopN)

	// 时间线排序
	stats.Timeline = buildTimeline(timeBucketCount)

	// 慢请求排序 (按响应时间降序)
	sort.Slice(stats.SlowRequests, func(i, j int) bool {
		return stats.SlowRequests[i].ResponseTime > stats.SlowRequests[j].ResponseTime
	})
	// 限制慢请求数量
	if len(stats.SlowRequests) > opts.TopN {
		stats.SlowRequests = stats.SlowRequests[:opts.TopN]
	}

	return stats
}

// topNMap 从 map 中取 Top N
func topNMap(m map[string]int, n int) []IPCount {
	result := make([]IPCount, 0, len(m))
	for k, v := range m {
		result = append(result, IPCount{IP: k, Count: v})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	if len(result) > n {
		result = result[:n]
	}
	return result
}

// topNMapURL 从 URL map 中取 Top N
func topNMapURL(m map[string]int, n int) []URLCount {
	result := make([]URLCount, 0, len(m))
	for k, v := range m {
		result = append(result, URLCount{URL: k, Count: v})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	if len(result) > n {
		result = result[:n]
	}
	return result
}

// topNMapUA 从 User-Agent map 中取 Top N
func topNMapUA(m map[string]int, n int) []UACount {
	result := make([]UACount, 0, len(m))
	for k, v := range m {
		result = append(result, UACount{Agent: k, Count: v})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	if len(result) > n {
		result = result[:n]
	}
	return result
}

// buildTimeline 构建有序时间线
func buildTimeline(m map[string]int) []TimeBucket {
	result := make([]TimeBucket, 0, len(m))
	for k, v := range m {
		result = append(result, TimeBucket{Time: k, Count: v})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Time < result[j].Time
	})
	return result
}

// simplifyUA 简化 User-Agent 字符串
func simplifyUA(ua string) string {
	// 提取主要信息
	ua = strings.TrimSpace(ua)
	if len(ua) > 80 {
		ua = ua[:80] + "..."
	}
	return ua
}
