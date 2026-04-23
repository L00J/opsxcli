package evolver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewExperienceMemory 测试创建经验记忆
func TestNewExperienceMemory(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	if em == nil {
		t.Fatal("NewExperienceMemory returned nil")
	}
	if em.Count() != 0 {
		t.Errorf("Count() = %d, want 0", em.Count())
	}
	if em.maxEntries != 1000 {
		t.Errorf("maxEntries = %d, want 1000", em.maxEntries)
	}
}

// TestAddExperience_New 测试添加新经验
func TestAddExperience_New(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{
		TaskType:     "磁盘分析",
		ToolSequence: []string{"bash", "file_read"},
		Hint:         "使用 df -h 查看磁盘",
		SuccessRate:  0.9,
		UsageCount:   1,
	})
	if em.Count() != 1 {
		t.Errorf("Count() = %d, want 1", em.Count())
	}
}

// TestAddExperience_UpdateExisting 测试更新已有经验
func TestAddExperience_UpdateExisting(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{
		TaskType:     "磁盘分析",
		ToolSequence: []string{"bash"},
		Hint:         "短提示",
		SuccessRate:  0.8,
		UsageCount:   1,
	})
	em.AddExperience(&Experience{
		TaskType:     "磁盘分析",
		ToolSequence: []string{"bash"},
		Hint:         "更详细的提示信息",
		SuccessRate:  1.0,
		UsageCount:   1,
	})
	if em.Count() != 1 {
		t.Errorf("Count() = %d, want 1 (should update, not add)", em.Count())
	}
	all := em.GetAllExperiences()
	if all[0].UsageCount != 2 {
		t.Errorf("UsageCount = %d, want 2", all[0].UsageCount)
	}
	if all[0].Hint != "更详细的提示信息" {
		t.Errorf("Hint = %q, want longer hint", all[0].Hint)
	}
}

// TestFindSimilar_ExactMatch 测试精确匹配任务类型
func TestFindSimilar_ExactMatch(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{
		TaskType:     "磁盘分析",
		ToolSequence: []string{"bash"},
		SuccessRate:  0.8,
		UsageCount:   5,
	})
	em.AddExperience(&Experience{
		TaskType:     "磁盘分析",
		ToolSequence: []string{"file_read"},
		SuccessRate:  0.95,
		UsageCount:   3,
	})
	result := em.FindSimilar("磁盘分析", []string{"bash"})
	if result == nil {
		t.Fatal("FindSimilar returned nil")
	}
	if result.SuccessRate != 0.95 {
		t.Errorf("SuccessRate = %f, want 0.95 (best match)", result.SuccessRate)
	}
}

// TestFindSimilar_ToolSequenceSimilarity 测试工具序列相似度匹配
func TestFindSimilar_ToolSequenceSimilarity(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{
		TaskType:     "内存分析",
		ToolSequence: []string{"bash", "file_read", "file_search"},
		SuccessRate:  0.9,
		UsageCount:   5,
	})
	// 不同任务类型但有相似工具序列
	result := em.FindSimilar("网络诊断", []string{"bash", "file_read", "file_search"})
	if result == nil {
		t.Fatal("FindSimilar should match by tool sequence similarity")
	}
}

// TestFindSimilar_NoMatch 测试无匹配
func TestFindSimilar_NoMatch(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{
		TaskType:     "磁盘分析",
		ToolSequence: []string{"bash"},
		SuccessRate:  0.8,
		UsageCount:   5,
	})
	result := em.FindSimilar("网络诊断", []string{"ssh_execute"})
	if result != nil {
		t.Error("FindSimilar should return nil when no match")
	}
}

// TestFindByTaskType 测试按任务类型查找
func TestFindByTaskType(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{TaskType: "磁盘分析", ToolSequence: []string{"bash"}, SuccessRate: 0.8, UsageCount: 3})
	em.AddExperience(&Experience{TaskType: "内存分析", ToolSequence: []string{"bash"}, SuccessRate: 0.9, UsageCount: 5})
	em.AddExperience(&Experience{TaskType: "磁盘分析", ToolSequence: []string{"file_read"}, SuccessRate: 0.7, UsageCount: 2})

	results := em.FindByTaskType("磁盘分析")
	if len(results) != 2 {
		t.Errorf("FindByTaskType('磁盘分析') returned %d, want 2", len(results))
	}
	results = em.FindByTaskType("内存分析")
	if len(results) != 1 {
		t.Errorf("FindByTaskType('内存分析') returned %d, want 1", len(results))
	}
	results = em.FindByTaskType("不存在")
	if len(results) != 0 {
		t.Errorf("FindByTaskType('不存在') returned %d, want 0", len(results))
	}
}

// TestGetBestExperience 测试获取最佳经验
func TestGetBestExperience(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{TaskType: "磁盘分析", ToolSequence: []string{"bash"}, SuccessRate: 0.6, UsageCount: 3})
	em.AddExperience(&Experience{TaskType: "磁盘分析", ToolSequence: []string{"file_read"}, SuccessRate: 0.95, UsageCount: 10})
	em.AddExperience(&Experience{TaskType: "磁盘分析", ToolSequence: []string{"ssh"}, SuccessRate: 0.7, UsageCount: 5})

	best := em.GetBestExperience("磁盘分析")
	if best == nil {
		t.Fatal("GetBestExperience returned nil")
	}
	if best.SuccessRate != 0.95 {
		t.Errorf("SuccessRate = %f, want 0.95", best.SuccessRate)
	}
}

// TestGetBestExperience_NoMatch 测试无匹配时返回 nil
func TestGetBestExperience_NoMatch(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{TaskType: "磁盘分析", SuccessRate: 0.8, UsageCount: 3})
	best := em.GetBestExperience("网络诊断")
	if best != nil {
		t.Error("GetBestExperience should return nil for unknown task type")
	}
}

// TestGetAllExperiences 测试获取所有经验
func TestGetAllExperiences(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{TaskType: "磁盘分析", SuccessRate: 0.8, UsageCount: 3})
	em.AddExperience(&Experience{TaskType: "内存分析", SuccessRate: 0.9, UsageCount: 5})

	all := em.GetAllExperiences()
	if len(all) != 2 {
		t.Errorf("GetAllExperiences() returned %d, want 2", len(all))
	}
	// 验证返回的是副本
	all[0].TaskType = "modified"
	if em.GetAllExperiences()[0].TaskType == "modified" {
		t.Error("GetAllExperiences should return copies, not references")
	}
}

// TestRemoveExperience 测试移除经验
func TestRemoveExperience(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	exp1 := &Experience{TaskType: "磁盘分析", ToolSequence: []string{"bash"}, SuccessRate: 0.8, UsageCount: 3}
	exp2 := &Experience{TaskType: "内存分析", ToolSequence: []string{"bash"}, SuccessRate: 0.9, UsageCount: 5}
	em.AddExperience(exp1)
	em.AddExperience(exp2)

	if em.Count() != 2 {
		t.Fatalf("Count() = %d, want 2", em.Count())
	}
	em.RemoveExperience(exp1)
	if em.Count() != 1 {
		t.Errorf("Count() after remove = %d, want 1", em.Count())
	}
}

// TestGetTaskTypeStats 测试任务类型统计
func TestGetTaskTypeStats(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{TaskType: "磁盘分析", UsageCount: 1})
	em.AddExperience(&Experience{TaskType: "磁盘分析", ToolSequence: []string{"other"}, UsageCount: 1})
	em.AddExperience(&Experience{TaskType: "内存分析", UsageCount: 1})

	stats := em.GetTaskTypeStats()
	if stats["磁盘分析"] != 2 {
		t.Errorf("stats['磁盘分析'] = %d, want 2", stats["磁盘分析"])
	}
	if stats["内存分析"] != 1 {
		t.Errorf("stats['内存分析'] = %d, want 1", stats["内存分析"])
	}
}

// TestDeleteByTaskType 测试按类型删除
func TestDeleteByTaskType(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{TaskType: "磁盘分析", UsageCount: 1})
	em.AddExperience(&Experience{TaskType: "磁盘分析", ToolSequence: []string{"other"}, UsageCount: 1})
	em.AddExperience(&Experience{TaskType: "内存分析", UsageCount: 1})

	deleted := em.DeleteByTaskType("磁盘分析")
	if deleted != 2 {
		t.Errorf("DeleteByTaskType returned %d, want 2", deleted)
	}
	if em.Count() != 1 {
		t.Errorf("Count() after delete = %d, want 1", em.Count())
	}
}

// TestClear 测试清空
func TestClear(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.AddExperience(&Experience{TaskType: "磁盘分析", UsageCount: 1})
	em.AddExperience(&Experience{TaskType: "内存分析", UsageCount: 1})
	em.Clear()
	if em.Count() != 0 {
		t.Errorf("Count() after clear = %d, want 0", em.Count())
	}
}

// TestMakeKey 测试经验键生成
func TestMakeKey(t *testing.T) {
	em := &ExperienceMemory{}
	key := em.makeKey("磁盘分析", []string{"bash", "file_read"})
	expected := "磁盘分析::bash→file_read"
	if key != expected {
		t.Errorf("makeKey() = %q, want %q", key, expected)
	}
	key2 := em.makeKey("磁盘分析", nil)
	if key2 != "磁盘分析::" {
		t.Errorf("makeKey(nil seq) = %q, want %q", key2, "磁盘分析::")
	}
}

// TestSequenceSimilarity 测试工具序列相似度
func TestSequenceSimilarity(t *testing.T) {
	em := &ExperienceMemory{}
	tests := []struct {
		name     string
		a, b     []string
		expected float64
	}{
		{"完全相同", []string{"a", "b"}, []string{"a", "b"}, 1.0},
		{"完全不同", []string{"a", "b"}, []string{"c", "d"}, 0.0},
		{"部分重叠", []string{"a", "b", "c"}, []string{"a", "c", "d"}, 2.0 / 3.0},
		{"一个空", []string{"a"}, []string{}, 0.0},
		{"两个空", []string{}, []string{}, 1.0},
		{"子集", []string{"a", "b", "c", "d"}, []string{"b", "d"}, 2.0 / 4.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := em.sequenceSimilarity(tc.a, tc.b)
			if result != tc.expected {
				t.Errorf("sequenceSimilarity(%v, %v) = %f, want %f", tc.a, tc.b, result, tc.expected)
			}
		})
	}
}

// TestLCSLength 测试最长公共子序列
func TestLCSLength(t *testing.T) {
	em := &ExperienceMemory{}
	tests := []struct {
		a, b     []string
		expected int
	}{
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}, 3},
		{[]string{"a", "b", "c"}, []string{"c", "b", "a"}, 1},
		{[]string{"a", "b", "c", "d"}, []string{"b", "d"}, 2},
		{[]string{}, []string{"a"}, 0},
		{[]string{}, []string{}, 0},
	}
	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			result := em.lcsLength(tc.a, tc.b)
			if result != tc.expected {
				t.Errorf("lcsLength(%v, %v) = %d, want %d", tc.a, tc.b, result, tc.expected)
			}
		})
	}
}

// TestSaveAndLoad 测试保存和加载
func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	em := NewExperienceMemory(dir)
	em.AddExperience(&Experience{
		TaskType:     "磁盘分析",
		ToolSequence: []string{"bash", "file_read"},
		Hint:         "df -h",
		SuccessRate:  0.9,
		UsageCount:   5,
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
	})

	if err := em.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err := LoadExperienceMemory(dir)
	if err != nil {
		t.Fatalf("LoadExperienceMemory() failed: %v", err)
	}
	if loaded.Count() != 1 {
		t.Errorf("loaded Count() = %d, want 1", loaded.Count())
	}
	all := loaded.GetAllExperiences()
	if all[0].TaskType != "磁盘分析" {
		t.Errorf("loaded TaskType = %q, want %q", all[0].TaskType, "磁盘分析")
	}
}

// TestLoadExperienceMemory_NoFile 测试加载不存在的文件
func TestLoadExperienceMemory_NoFile(t *testing.T) {
	dir := t.TempDir()
	em, err := LoadExperienceMemory(dir)
	if err != nil {
		t.Fatalf("LoadExperienceMemory() with no file failed: %v", err)
	}
	if em.Count() != 0 {
		t.Errorf("Count() = %d, want 0", em.Count())
	}
}

// TestLoadExperienceMemory_CorruptLine 测试加载含损坏行的文件
func TestLoadExperienceMemory_CorruptLine(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "experience.jsonl")
	// 写入一行有效 + 一行损坏
	validJSON, _ := json.Marshal(Experience{TaskType: "磁盘分析", SuccessRate: 0.8})
	content := string(validJSON) + "\ncorrupt line\n"
	os.WriteFile(filePath, []byte(content), 0644)

	em, err := LoadExperienceMemory(dir)
	if err != nil {
		t.Fatalf("LoadExperienceMemory() failed: %v", err)
	}
	if em.Count() != 1 {
		t.Errorf("Count() = %d, want 1 (skip corrupt lines)", em.Count())
	}
}

// TestCleanup 测试清理低效经验
func TestCleanup(t *testing.T) {
	em := NewExperienceMemory(t.TempDir())
	em.maxEntries = 3

	// 添加超过限制的经验
	em.AddExperience(&Experience{
		TaskType: "old_good", SuccessRate: 0.9, UsageCount: 10,
		LastUsedAt: time.Now().AddDate(0, 0, -60), // 60天前
	})
	em.AddExperience(&Experience{
		TaskType: "recent_bad", SuccessRate: 0.1, UsageCount: 1,
		ToolSequence: []string{"a"},
		LastUsedAt:   time.Now(), // 最近
	})
	em.AddExperience(&Experience{
		TaskType: "high_rate", SuccessRate: 0.95, UsageCount: 1,
		ToolSequence: []string{"b"},
		LastUsedAt:   time.Now().AddDate(0, 0, -60),
	})
	// 触发 cleanup
	em.AddExperience(&Experience{
		TaskType: "new", SuccessRate: 0.5, UsageCount: 5,
		ToolSequence: []string{"c"},
		LastUsedAt:   time.Now(),
	})

	// cleanup 应保留高成功率和最近使用的
	if em.Count() > 3 {
		t.Errorf("Count() = %d, should be <= 3 after cleanup", em.Count())
	}
}

// TestSortByQuality 测试按质量排序
func TestSortByQuality(t *testing.T) {
	em := &ExperienceMemory{}
	exps := []*Experience{
		{TaskType: "low", SuccessRate: 0.3, UsageCount: 2},   // score: 0.6
		{TaskType: "high", SuccessRate: 0.9, UsageCount: 10}, // score: 9.0
		{TaskType: "mid", SuccessRate: 0.7, UsageCount: 5},   // score: 3.5
	}
	em.sortByQuality(exps)
	if exps[0].TaskType != "high" {
		t.Errorf("First should be 'high', got %q", exps[0].TaskType)
	}
	if exps[2].TaskType != "low" {
		t.Errorf("Last should be 'low', got %q", exps[2].TaskType)
	}
}
