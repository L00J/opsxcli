package logs

import (
	"sort"
	"testing"
	"time"
)

func TestCalculateStats_Empty(t *testing.T) {
	opts := DefaultOptions()
	stats := CalculateStats(nil, opts)
	if stats == nil {
		t.Fatal("CalculateStats(nil) 不应返回 nil")
	}
	if stats.TotalRequests != 0 {
		t.Errorf("TotalRequests = %d, want 0", stats.TotalRequests)
	}
	if len(stats.TopIPs) != 0 {
		t.Errorf("TopIPs 长度 = %d, want 0", len(stats.TopIPs))
	}
	if len(stats.StatusCodes) != 0 {
		t.Errorf("StatusCodes 长度 = %d, want 0", len(stats.StatusCodes))
	}

	// 空切片也应返回空结果
	entries := []LogEntry{}
	stats2 := CalculateStats(entries, opts)
	if stats2.TotalRequests != 0 {
		t.Errorf("空切片 TotalRequests = %d, want 0", stats2.TotalRequests)
	}
}

func TestCalculateStats_BasicCounts(t *testing.T) {
	ts := time.Date(2026, time.April, 21, 10, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{Timestamp: ts, RemoteIP: "1.1.1.1", Method: "GET", Path: "/a", StatusCode: 200, BodyBytes: 100, ResponseTime: 50},
		{Timestamp: ts.Add(time.Minute), RemoteIP: "1.1.1.1", Method: "GET", Path: "/a", StatusCode: 200, BodyBytes: 200, ResponseTime: 100},
		{Timestamp: ts.Add(2 * time.Minute), RemoteIP: "2.2.2.2", Method: "POST", Path: "/b", StatusCode: 404, BodyBytes: 50, ResponseTime: 200},
		{Timestamp: ts.Add(3 * time.Minute), RemoteIP: "3.3.3.3", Method: "GET", Path: "/c", StatusCode: 500, BodyBytes: 0, ResponseTime: 5000},
		{Timestamp: ts.Add(4 * time.Minute), RemoteIP: "2.2.2.2", Method: "GET", Path: "/a", StatusCode: 200, BodyBytes: 150, ResponseTime: 80},
	}

	opts := DefaultOptions()
	opts.TopN = 2
	stats := CalculateStats(entries, opts)

	// 状态码统计
	if stats.StatusCodes[200] != 3 {
		t.Errorf("StatusCodes[200] = %d, want 3", stats.StatusCodes[200])
	}
	if stats.StatusCodes[404] != 1 {
		t.Errorf("StatusCodes[404] = %d, want 1", stats.StatusCodes[404])
	}
	if stats.StatusCodes[500] != 1 {
		t.Errorf("StatusCodes[500] = %d, want 1", stats.StatusCodes[500])
	}

	// 错误计数 (4xx + 5xx)
	if stats.ErrorCount != 2 {
		t.Errorf("ErrorCount = %d, want 2", stats.ErrorCount)
	}

	// 总字节数
	wantBytes := int64(100 + 200 + 50 + 0 + 150)
	if stats.TotalBytes != wantBytes {
		t.Errorf("TotalBytes = %d, want %d", stats.TotalBytes, wantBytes)
	}

	// 平均响应时间: (50+100+200+5000+80)/5 = 1086
	wantAvg := int64((50 + 100 + 200 + 5000 + 80) / 5)
	if stats.AvgResponseMs != wantAvg {
		t.Errorf("AvgResponseMs = %d, want %d", stats.AvgResponseMs, wantAvg)
	}

	// TopN IP (Top 2)
	if len(stats.TopIPs) > 2 {
		t.Errorf("TopIPs 长度 = %d, want <= 2", len(stats.TopIPs))
	}
	if len(stats.TopIPs) >= 1 && stats.TopIPs[0].IP != "1.1.1.1" && stats.TopIPs[0].IP != "2.2.2.2" {
		t.Errorf("TopIPs[0].IP = %q, 排名最高的 IP 应为 1.1.1.1 或 2.2.2.2", stats.TopIPs[0].IP)
	}

	// TopN URL (Top 2)
	if len(stats.TopURLs) > 2 {
		t.Errorf("TopURLs 长度 = %d, want <= 2", len(stats.TopURLs))
	}

	// 时间范围
	if !stats.TimeRange.Start.Equal(ts) {
		t.Errorf("TimeRange.Start = %v, want %v", stats.TimeRange.Start, ts)
	}
	if !stats.TimeRange.End.Equal(ts.Add(4 * time.Minute)) {
		t.Errorf("TimeRange.End = %v, want %v", stats.TimeRange.End, ts.Add(4*time.Minute))
	}
}

func TestCalculateStats_SlowRequests(t *testing.T) {
	ts := time.Date(2026, time.April, 21, 10, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{Timestamp: ts, RemoteIP: "1.1.1.1", StatusCode: 200, ResponseTime: 50},
		{Timestamp: ts, RemoteIP: "2.2.2.2", StatusCode: 200, ResponseTime: 2000},
		{Timestamp: ts, RemoteIP: "3.3.3.3", StatusCode: 200, ResponseTime: 5000},
		{Timestamp: ts, RemoteIP: "4.4.4.4", StatusCode: 200, ResponseTime: 1500},
		{Timestamp: ts, RemoteIP: "5.5.5.5", StatusCode: 200, ResponseTime: 100},
	}

	opts := DefaultOptions()
	opts.SlowThreshold = 1000
	opts.TopN = 2
	stats := CalculateStats(entries, opts)

	// 慢请求应为 2000, 5000, 1500 (>= 1000ms)，但 TopN=2 限制为 2 条
	if len(stats.SlowRequests) != 2 {
		t.Fatalf("SlowRequests 长度 = %d, want 2", len(stats.SlowRequests))
	}
	// 按响应时间降序排列
	if stats.SlowRequests[0].ResponseTime != 5000 {
		t.Errorf("SlowRequests[0].ResponseTime = %d, want 5000", stats.SlowRequests[0].ResponseTime)
	}
	if stats.SlowRequests[1].ResponseTime != 2000 {
		t.Errorf("SlowRequests[1].ResponseTime = %d, want 2000", stats.SlowRequests[1].ResponseTime)
	}
}

func TestCalculateStats_ErrorIPs(t *testing.T) {
	ts := time.Date(2026, time.April, 21, 10, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{Timestamp: ts, RemoteIP: "1.1.1.1", StatusCode: 200},
		{Timestamp: ts, RemoteIP: "2.2.2.2", StatusCode: 500},
		{Timestamp: ts, RemoteIP: "2.2.2.2", StatusCode: 502},
		{Timestamp: ts, RemoteIP: "3.3.3.3", StatusCode: 400},
		{Timestamp: ts, RemoteIP: "1.1.1.1", StatusCode: 404},
	}

	opts := DefaultOptions()
	opts.TopN = 10
	stats := CalculateStats(entries, opts)

	// 错误 IP: 2.2.2.2 出现 2 次, 3.3.3.3 出现 1 次, 1.1.1.1 出现 1 次
	if len(stats.ErrorIPs) == 0 {
		t.Fatal("ErrorIPs 不应为空")
	}
	if stats.ErrorIPs[0].IP != "2.2.2.2" {
		t.Errorf("ErrorIPs[0].IP = %q, want %q", stats.ErrorIPs[0].IP, "2.2.2.2")
	}
	if stats.ErrorIPs[0].Count != 2 {
		t.Errorf("ErrorIPs[0].Count = %d, want 2", stats.ErrorIPs[0].Count)
	}
}

func TestCalculateStats_TimeTimeline(t *testing.T) {
	ts := time.Date(2026, time.April, 21, 10, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{Timestamp: ts, RemoteIP: "1.1.1.1", StatusCode: 200},
		{Timestamp: ts.Add(30 * time.Minute), RemoteIP: "1.1.1.1", StatusCode: 200},
		{Timestamp: ts.Add(60 * time.Minute), RemoteIP: "1.1.1.1", StatusCode: 200},
		{Timestamp: ts.Add(90 * time.Minute), RemoteIP: "1.1.1.1", StatusCode: 200},
	}

	opts := DefaultOptions()
	stats := CalculateStats(entries, opts)

	// 应该有不同的时间桶 (按小时)
	if len(stats.Timeline) == 0 {
		t.Fatal("Timeline 不应为空")
	}
	// 时间线应按时间排序
	for i := 1; i < len(stats.Timeline); i++ {
		if stats.Timeline[i].Time < stats.Timeline[i-1].Time {
			t.Errorf("Timeline 未排序: [%d]=%q > [%d]=%q", i-1, stats.Timeline[i-1].Time, i, stats.Timeline[i].Time)
		}
	}
}

func TestCalculateStats_UserAgents(t *testing.T) {
	ts := time.Date(2026, time.April, 21, 10, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{Timestamp: ts, RemoteIP: "1.1.1.1", StatusCode: 200, UserAgent: "Mozilla/5.0 Chrome"},
		{Timestamp: ts, RemoteIP: "2.2.2.2", StatusCode: 200, UserAgent: "Mozilla/5.0 Chrome"},
		{Timestamp: ts, RemoteIP: "3.3.3.3", StatusCode: 200, UserAgent: "curl/7.88"},
		{Timestamp: ts, RemoteIP: "4.4.4.4", StatusCode: 200, UserAgent: ""},
	}

	opts := DefaultOptions()
	opts.TopN = 2
	stats := CalculateStats(entries, opts)

	if len(stats.UserAgents) > 2 {
		t.Errorf("UserAgents 长度 = %d, want <= 2", len(stats.UserAgents))
	}
	if len(stats.UserAgents) >= 1 {
		if stats.UserAgents[0].Agent != "Mozilla/5.0 Chrome" {
			t.Errorf("UserAgents[0].Agent = %q, want %q", stats.UserAgents[0].Agent, "Mozilla/5.0 Chrome")
		}
		if stats.UserAgents[0].Count != 2 {
			t.Errorf("UserAgents[0].Count = %d, want 2", stats.UserAgents[0].Count)
		}
	}
}

func TestTopNMap(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]int
		n    int
		want []IPCount
	}{
		{
			name: "正常排序取 Top 2",
			m:    map[string]int{"a": 5, "b": 3, "c": 8, "d": 1},
			n:    2,
			want: []IPCount{{IP: "c", Count: 8}, {IP: "a", Count: 5}},
		},
		{
			name: "n 大于 map 长度",
			m:    map[string]int{"x": 1},
			n:    10,
			want: []IPCount{{IP: "x", Count: 1}},
		},
		{
			name: "空 map",
			m:    map[string]int{},
			n:    5,
			want: []IPCount{},
		},
		{
			name: "n 为 0",
			m:    map[string]int{"a": 1, "b": 2},
			n:    0,
			want: []IPCount{},
		},
		{
			name: "相同计数",
			m:    map[string]int{"a": 5, "b": 5},
			n:    5,
			want: []IPCount{{IP: "a", Count: 5}, {IP: "b", Count: 5}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := topNMap(tt.m, tt.n)
			if len(got) != len(tt.want) {
				t.Errorf("topNMap() 长度 = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, w := range tt.want {
				if got[i].Count != w.Count {
					t.Errorf("topNMap()[%d].Count = %d, want %d", i, got[i].Count, w.Count)
				}
			}
		})
	}
}

func TestTopNMapURL(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]int
		n    int
		want []URLCount
	}{
		{
			name: "正常排序取 Top 3",
			m:    map[string]int{"/a": 10, "/b": 5, "/c": 20, "/d": 1},
			n:    3,
			want: []URLCount{{URL: "/c", Count: 20}, {URL: "/a", Count: 10}, {URL: "/b", Count: 5}},
		},
		{
			name: "空 map",
			m:    map[string]int{},
			n:    5,
			want: []URLCount{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := topNMapURL(tt.m, tt.n)
			if len(got) != len(tt.want) {
				t.Errorf("topNMapURL() 长度 = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, w := range tt.want {
				if got[i].Count != w.Count {
					t.Errorf("topNMapURL()[%d].Count = %d, want %d", i, got[i].Count, w.Count)
				}
			}
		})
	}
}

func TestTopNMapUA(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]int
		n    int
		want []UACount
	}{
		{
			name: "正常排序取 Top 2",
			m:    map[string]int{"Chrome": 50, "Safari": 30, "curl": 10},
			n:    2,
			want: []UACount{{Agent: "Chrome", Count: 50}, {Agent: "Safari", Count: 30}},
		},
		{
			name: "空 map",
			m:    map[string]int{},
			n:    5,
			want: []UACount{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := topNMapUA(tt.m, tt.n)
			if len(got) != len(tt.want) {
				t.Errorf("topNMapUA() 长度 = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, w := range tt.want {
				if got[i].Count != w.Count {
					t.Errorf("topNMapUA()[%d].Count = %d, want %d", i, got[i].Count, w.Count)
				}
			}
		})
	}
}

func TestBuildTimeline(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]int
		want []TimeBucket
	}{
		{
			name: "已排序",
			m:    map[string]int{"2026-04-21 10:00": 5, "2026-04-21 11:00": 3},
			want: []TimeBucket{
				{Time: "2026-04-21 10:00", Count: 5},
				{Time: "2026-04-21 11:00", Count: 3},
			},
		},
		{
			name: "逆序输入应得到正序输出",
			m:    map[string]int{"2026-04-21 12:00": 1, "2026-04-21 09:00": 4},
			want: []TimeBucket{
				{Time: "2026-04-21 09:00", Count: 4},
				{Time: "2026-04-21 12:00", Count: 1},
			},
		},
		{
			name: "空 map",
			m:    map[string]int{},
			want: []TimeBucket{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTimeline(tt.m)
			if len(got) != len(tt.want) {
				t.Errorf("buildTimeline() 长度 = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, w := range tt.want {
				if got[i].Time != w.Time || got[i].Count != w.Count {
					t.Errorf("buildTimeline()[%d] = {%q, %d}, want {%q, %d}", i, got[i].Time, got[i].Count, w.Time, w.Count)
				}
			}
		})
	}
}

func TestSimplifyUA(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "短 UA 不变", input: "curl/7.88", want: "curl/7.88"},
		{name: "长 UA 截断", input: stringsRepeat("x", 100), want: stringsRepeat("x", 80) + "..."},
		{name: "恰好 80 字符不截断", input: stringsRepeat("a", 80), want: stringsRepeat("a", 80)},
		{name: "81 字符截断", input: stringsRepeat("b", 81), want: stringsRepeat("b", 80) + "..."},
		{name: "空字符串", input: "", want: ""},
		{name: "带前后空格", input: "  agent  ", want: "agent"},
		{name: "超长带空格先 trim 再截断", input: "  " + stringsRepeat("c", 90) + "  ", want: stringsRepeat("c", 80) + "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := simplifyUA(tt.input)
			if got != tt.want {
				t.Errorf("simplifyUA() = %q, want %q", got, tt.want)
			}
		})
	}
}

func stringsRepeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

func TestCalculateStats_NoResponseTime(t *testing.T) {
	ts := time.Date(2026, time.April, 21, 10, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{Timestamp: ts, RemoteIP: "1.1.1.1", StatusCode: 200, ResponseTime: 0},
		{Timestamp: ts, RemoteIP: "2.2.2.2", StatusCode: 200, ResponseTime: 0},
	}

	opts := DefaultOptions()
	stats := CalculateStats(entries, opts)

	// 无响应时间时平均应为 0
	if stats.AvgResponseMs != 0 {
		t.Errorf("AvgResponseMs = %d, want 0 (无响应时间数据)", stats.AvgResponseMs)
	}
}

func TestCalculateStats_SingleEntry(t *testing.T) {
	ts := time.Date(2026, time.April, 21, 10, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{Timestamp: ts, RemoteIP: "1.1.1.1", Method: "GET", Path: "/", StatusCode: 200, BodyBytes: 42, ResponseTime: 100, UserAgent: "test"},
	}

	opts := DefaultOptions()
	stats := CalculateStats(entries, opts)

	if stats.StatusCodes[200] != 1 {
		t.Errorf("StatusCodes[200] = %d, want 1", stats.StatusCodes[200])
	}
	if stats.TotalBytes != 42 {
		t.Errorf("TotalBytes = %d, want 42", stats.TotalBytes)
	}
	if stats.AvgResponseMs != 100 {
		t.Errorf("AvgResponseMs = %d, want 100", stats.AvgResponseMs)
	}
	if len(stats.TopIPs) != 1 || stats.TopIPs[0].IP != "1.1.1.1" {
		t.Errorf("TopIPs = %v, want [{1.1.1.1 1}]", stats.TopIPs)
	}
	if len(stats.TopURLs) != 1 || stats.TopURLs[0].URL != "/" {
		t.Errorf("TopURLs = %v, want [{/ 1}]", stats.TopURLs)
	}
	if !stats.TimeRange.Start.Equal(ts) || !stats.TimeRange.End.Equal(ts) {
		t.Errorf("TimeRange = %v ~ %v, want both %v", stats.TimeRange.Start, stats.TimeRange.End, ts)
	}
}

func TestCalculateStats_TopNSorting(t *testing.T) {
	// 确认 topNMap 系列函数是稳定排序（降序）
	m := map[string]int{
		"d": 1, "c": 3, "a": 10, "b": 5,
	}
	result := topNMap(m, 10)
	for i := 1; i < len(result); i++ {
		if result[i].Count > result[i-1].Count {
			t.Errorf("结果未按降序排列: [%d]=%d > [%d]=%d", i, result[i].Count, i-1, result[i-1].Count)
		}
	}
	_ = sort.Ints // 验证 sort 可用
}
