package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewTokenStatsManager 测试创建统计管理器
func TestNewTokenStatsManager(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test_stats.jsonl")

	mgr, err := NewTokenStatsManager(filePath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}
	if mgr == nil {
		t.Fatal("管理器不应为 nil")
	}
	if len(mgr.records) != 0 {
		t.Fatalf("初始记录数应为 0，实际为 %d", len(mgr.records))
	}
}

// TestTokenStatsManager_RecordAndRead 测试记录和读取
func TestTokenStatsManager_RecordAndRead(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test_stats.jsonl")

	// 先手动创建带内容的文件
	records := []TokenRecord{
		{Provider: "deepseek", Model: "deepseek-chat", SessionID: "s1", PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150, Timestamp: time.Now()},
		{Provider: "openai", Model: "gpt-4", SessionID: "s2", PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300, Timestamp: time.Now()},
		{Provider: "deepseek", Model: "deepseek-chat", SessionID: "s1", PromptTokens: 50, CompletionTokens: 30, TotalTokens: 80, Timestamp: time.Now()},
	}
	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}
	for _, r := range records {
		data, _ := json.Marshal(r)
		f.Write(data)
		f.Write([]byte("\n"))
	}
	f.Close()

	// 加载已有记录
	mgr, err := NewTokenStatsManager(filePath)
	if err != nil {
		t.Fatalf("加载管理器失败: %v", err)
	}
	if len(mgr.records) != 3 {
		t.Fatalf("应加载 3 条记录，实际为 %d", len(mgr.records))
	}

	// 再追加一条
	mgr.Record("claude", "claude-3", "s3", &Usage{PromptTokens: 300, CompletionTokens: 150, TotalTokens: 450})
	time.Sleep(300 * time.Millisecond)

	if len(mgr.records) != 4 {
		t.Errorf("追加后应有 4 条记录，实际为 %d", len(mgr.records))
	}

	// 重新加载验证持久化
	mgr2, err := NewTokenStatsManager(filePath)
	if err != nil {
		t.Fatalf("重新加载管理器失败: %v", err)
	}
	if len(mgr2.records) != 4 {
		t.Errorf("重新加载后记录数应为 4，实际为 %d", len(mgr2.records))
	}
}

// TestTokenStatsManager_RecordNilUsage 测试 nil usage 不记录
func TestTokenStatsManager_RecordNilUsage(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test_stats.jsonl")

	mgr, err := NewTokenStatsManager(filePath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	mgr.Record("deepseek", "deepseek-chat", "session-1", nil)
	mgr.Record("deepseek", "deepseek-chat", "session-1", &Usage{})

	if len(mgr.records) != 0 {
		t.Errorf("nil/零值 usage 不应记录，实际记录了 %d 条", len(mgr.records))
	}
}

// TestTokenStatsManager_GetProviderStats 测试按 Provider 汇总统计
func TestTokenStatsManager_GetProviderStats(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test_stats.jsonl")

	mgr, err := NewTokenStatsManager(filePath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	mgr.Record("deepseek", "deepseek-chat", "s1", &Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150})
	mgr.Record("deepseek", "deepseek-chat", "s1", &Usage{PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300})
	mgr.Record("openai", "gpt-4", "s1", &Usage{PromptTokens: 500, CompletionTokens: 250, TotalTokens: 750})

	stats := mgr.GetProviderStats()
	if len(stats) != 2 {
		t.Fatalf("应有 2 个 Provider 统计，实际为 %d", len(stats))
	}

	// 查找 deepseek 统计
	var dsStats *ProviderStats
	var oaStats *ProviderStats
	for i := range stats {
		if stats[i].Provider == "deepseek" {
			dsStats = &stats[i]
		}
		if stats[i].Provider == "openai" {
			oaStats = &stats[i]
		}
	}

	if dsStats == nil {
		t.Fatal("未找到 deepseek 统计")
	}
	if dsStats.TotalRequests != 2 {
		t.Errorf("deepseek 请求数应为 2，实际为 %d", dsStats.TotalRequests)
	}
	if dsStats.TotalTokens != 450 {
		t.Errorf("deepseek 总 Token 应为 450，实际为 %d", dsStats.TotalTokens)
	}
	if dsStats.AvgTokensPerReq != 225.0 {
		t.Errorf("deepseek 平均 Token 应为 225.0，实际为 %.1f", dsStats.AvgTokensPerReq)
	}

	if oaStats == nil {
		t.Fatal("未找到 openai 统计")
	}
	if oaStats.TotalRequests != 1 {
		t.Errorf("openai 请求数应为 1，实际为 %d", oaStats.TotalRequests)
	}
	if oaStats.TotalTokens != 750 {
		t.Errorf("openai 总 Token 应为 750，实际为 %d", oaStats.TotalTokens)
	}
}

// TestTokenStatsManager_GetSessionStats 测试按会话统计
func TestTokenStatsManager_GetSessionStats(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test_stats.jsonl")

	mgr, err := NewTokenStatsManager(filePath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	mgr.Record("deepseek", "deepseek-chat", "session-abc", &Usage{TotalTokens: 150})
	mgr.Record("deepseek", "deepseek-chat", "session-abc", &Usage{TotalTokens: 200})
	mgr.Record("openai", "gpt-4", "session-xyz", &Usage{TotalTokens: 300})

	stats := mgr.GetSessionStats("session-abc")
	if stats.TotalTokens != 350 {
		t.Errorf("session-abc 总 Token 应为 350，实际为 %d", stats.TotalTokens)
	}
	if stats.Requests != 2 {
		t.Errorf("session-abc 请求数应为 2，实际为 %d", stats.Requests)
	}

	statsXYZ := mgr.GetSessionStats("session-xyz")
	if statsXYZ.TotalTokens != 300 {
		t.Errorf("session-xyz 总 Token 应为 300，实际为 %d", statsXYZ.TotalTokens)
	}

	statsEmpty := mgr.GetSessionStats("nonexistent")
	if statsEmpty.TotalTokens != 0 || statsEmpty.Requests != 0 {
		t.Errorf("不存在的会话应为零值")
	}
}

// TestTokenStatsManager_GetTotalUsage 测试总使用量
func TestTokenStatsManager_GetTotalUsage(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test_stats.jsonl")

	mgr, err := NewTokenStatsManager(filePath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	mgr.Record("p1", "m1", "s1", &Usage{TotalTokens: 100})
	mgr.Record("p2", "m2", "s2", &Usage{TotalTokens: 200})
	mgr.Record("p3", "m3", "s3", &Usage{TotalTokens: 300})

	requests, tokens := mgr.GetTotalUsage()
	if requests != 3 {
		t.Errorf("总请求数应为 3，实际为 %d", requests)
	}
	if tokens != 600 {
		t.Errorf("总 Token 应为 600，实际为 %d", tokens)
	}
}

// TestTokenStatsManager_GetRecentUsage 测试最近使用量
func TestTokenStatsManager_GetRecentUsage(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test_stats.jsonl")

	mgr, err := NewTokenStatsManager(filePath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	mgr.Record("p1", "m1", "s1", &Usage{TotalTokens: 100})

	// 最近 1 小时
	requests, tokens := mgr.GetRecentUsage(1)
	if requests != 1 {
		t.Errorf("最近 1 小时请求数应为 1，实际为 %d", requests)
	}
	if tokens != 100 {
		t.Errorf("最近 1 小时 Token 应为 100，实际为 %d", tokens)
	}

	// 最近 0 小时（应该为 0）
	requests, tokens = mgr.GetRecentUsage(0)
	if requests != 0 {
		t.Errorf("最近 0 小时应无请求，实际为 %d", requests)
	}
}

// TestTokenStatsManager_MaxRows 测试记录上限
func TestTokenStatsManager_MaxRows(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test_stats.jsonl")

	mgr, err := NewTokenStatsManager(filePath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}
	mgr.maxRows = 10 // 设置为较小的上限便于测试

	// 插入 15 条记录（同步追加到内存，截断逻辑在 Record 内部触发）
	for i := 0; i < 15; i++ {
		mgr.Record("p1", "m1", "s1", &Usage{TotalTokens: i + 1})
		time.Sleep(2 * time.Millisecond) // 让异步 persist 有时间完成
	}

	// 验证内存中保留数量
	if len(mgr.records) > 10 {
		t.Errorf("内存记录数不应超过 maxRows(10)，实际为 %d", len(mgr.records))
	}
}

// TestTokenStatsManager_LoadFromFile 测试从文件加载
func TestTokenStatsManager_LoadFromFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test_stats.jsonl")

	// 手动写入一些记录
	records := []TokenRecord{
		{Provider: "deepseek", Model: "deepseek-chat", SessionID: "s1", TotalTokens: 100, Timestamp: time.Now()},
		{Provider: "openai", Model: "gpt-4", SessionID: "s2", TotalTokens: 200, Timestamp: time.Now()},
	}

	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}
	for _, r := range records {
		data, _ := json.Marshal(r)
		f.Write(data)
		f.Write([]byte("\n"))
	}
	f.Close()

	// 加载
	mgr, err := NewTokenStatsManager(filePath)
	if err != nil {
		t.Fatalf("加载管理器失败: %v", err)
	}

	if len(mgr.records) != 2 {
		t.Fatalf("应加载 2 条记录，实际为 %d", len(mgr.records))
	}

	totalReqs, totalTokens := mgr.GetTotalUsage()
	if totalReqs != 2 {
		t.Errorf("总请求数应为 2，实际为 %d", totalReqs)
	}
	if totalTokens != 300 {
		t.Errorf("总 Token 应为 300，实际为 %d", totalTokens)
	}
}

// TestFormatTokenCount 测试 Token 数量格式化
func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{10000, "10.0K"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
	}

	for _, tt := range tests {
		result := FormatTokenCount(tt.input)
		if result != tt.expected {
			t.Errorf("FormatTokenCount(%d) = %q, 期望 %q", tt.input, result, tt.expected)
		}
	}
}

// TestSplitLines 测试行分割
func TestSplitLines(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"line1", 1},
		{"line1\nline2", 2},
		{"line1\nline2\n", 2},
		{"line1\n\nline3", 3},
	}

	for _, tt := range tests {
		result := splitLines(tt.input)
		if len(result) != tt.expected {
			t.Errorf("splitLines(%q) 返回 %d 行，期望 %d", tt.input, len(result), tt.expected)
		}
	}
}
