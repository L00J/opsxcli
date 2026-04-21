package evolver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProceduralMemory_SetAndGet(t *testing.T) {
	dir := t.TempDir()
	pm := NewProceduralMemory(dir)

	skill := &SkillEntry{
		ID:          "network_diagnosis",
		Name:        "网络诊断",
		Category:    "network",
		Description: "网络连通性诊断与故障排查",
		Steps:       []string{"检查网络接口", "测试DNS解析", "测试连通性"},
		ToolSeq:     []string{"local_bash", "ssh_execute"},
		Triggers:    []string{"网络", "ping", "连接超时"},
		Source:      "seed",
	}

	pm.SetSkill(skill)

	got, ok := pm.GetSkill("network_diagnosis")
	if !ok {
		t.Fatal("should find skill")
	}
	if got.Name != "网络诊断" {
		t.Errorf("Name = %q, want %q", got.Name, "网络诊断")
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}
	if got.Source != "seed" {
		t.Errorf("Source = %q, want %q", got.Source, "seed")
	}
}

func TestProceduralMemory_Update(t *testing.T) {
	dir := t.TempDir()
	pm := NewProceduralMemory(dir)

	// 创建
	skill := &SkillEntry{
		ID:      "test_skill",
		Name:    "测试技能",
		Category: "system",
	}
	pm.SetSkill(skill)

	// 更新
	skill2 := &SkillEntry{
		ID:          "test_skill",
		Name:        "更新后的技能",
		Category:    "system",
		Description: "新增描述",
	}
	pm.SetSkill(skill2)

	got, _ := pm.GetSkill("test_skill")
	if got.Name != "更新后的技能" {
		t.Errorf("Name = %q, want updated name", got.Name)
	}
	if got.Version != 2 {
		t.Errorf("Version = %d, want 2 (updated)", got.Version)
	}
}

func TestProceduralMemory_Delete(t *testing.T) {
	dir := t.TempDir()
	pm := NewProceduralMemory(dir)

	pm.SetSkill(&SkillEntry{ID: "to_delete", Name: "待删除"})
	
	if !pm.DeleteSkill("to_delete") {
		t.Fatal("should delete existing skill")
	}
	if pm.DeleteSkill("non_existent") {
		t.Fatal("should not delete non-existent skill")
	}
	if pm.Count() != 0 {
		t.Errorf("Count = %d, want 0", pm.Count())
	}
}

func TestProceduralMemory_GetSkillsByCategory(t *testing.T) {
	dir := t.TempDir()
	pm := NewProceduralMemory(dir)

	pm.SetSkill(&SkillEntry{ID: "net1", Name: "网络1", Category: "network", UsageCount: 5})
	pm.SetSkill(&SkillEntry{ID: "sys1", Name: "系统1", Category: "system", UsageCount: 3})
	pm.SetSkill(&SkillEntry{ID: "net2", Name: "网络2", Category: "network", UsageCount: 10})

	network := pm.GetSkillsByCategory("network")
	if len(network) != 2 {
		t.Fatalf("network skills = %d, want 2", len(network))
	}
	// Should be sorted by usage count desc
	if network[0].ID != "net2" {
		t.Errorf("first network skill = %q, want net2 (highest usage)", network[0].ID)
	}
}

func TestProceduralMemory_FindMatchingSkills(t *testing.T) {
	dir := t.TempDir()
	pm := NewProceduralMemory(dir)

	pm.SetSkill(&SkillEntry{
		ID:       "disk_check",
		Name:     "磁盘检查",
		Category: "system",
		Triggers: []string{"磁盘", "空间", "disk"},
		UsageCount: 5,
	})
	pm.SetSkill(&SkillEntry{
		ID:       "net_diag",
		Name:     "网络诊断",
		Category: "network",
		Triggers: []string{"网络", "ping", "连接"},
		UsageCount: 3,
	})

	// 匹配触发条件
	results := pm.FindMatchingSkills("磁盘空间不足")
	if len(results) != 1 || results[0].ID != "disk_check" {
		t.Errorf("FindMatchingSkills(磁盘空间不足) = %v, want disk_check", results)
	}

	// 匹配名称
	results = pm.FindMatchingSkills("网络诊断")
	if len(results) != 1 || results[0].ID != "net_diag" {
		t.Errorf("FindMatchingSkills(网络诊断) = %v, want net_diag", results)
	}

	// 无匹配
	results = pm.FindMatchingSkills("docker部署")
	if len(results) != 0 {
		t.Errorf("FindMatchingSkills(docker部署) = %d, want 0", len(results))
	}
}

func TestProceduralMemory_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	pm := NewProceduralMemory(dir)

	pm.SetSkill(&SkillEntry{
		ID:          "save_test",
		Name:        "保存测试",
		Category:    "system",
		Description: "测试保存和加载",
		Steps:       []string{"步骤1", "步骤2", "步骤3"},
		ToolSeq:     []string{"local_bash"},
		Triggers:    []string{"测试"},
		Pitfalls:    []string{"注意陷阱"},
		Source:      "learned",
	})

	// 保存
	if err := pm.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// 验证文件存在
	skillFile := filepath.Join(dir, "skills", "SKILL_save_test.md")
	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		t.Fatal("skill file should exist after save")
	}

	// 加载
	pm2, err := LoadProceduralMemory(dir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	got, ok := pm2.GetSkill("save_test")
	if !ok {
		t.Fatal("should load saved skill")
	}
	if got.Name != "保存测试" {
		t.Errorf("Name = %q, want %q", got.Name, "保存测试")
	}
	if got.Source != "learned" {
		t.Errorf("Source = %q, want %q", got.Source, "learned")
	}
	if len(got.Steps) != 3 {
		t.Errorf("Steps count = %d, want 3", len(got.Steps))
	}
	if len(got.Triggers) != 1 || got.Triggers[0] != "测试" {
		t.Errorf("Triggers = %v, want [测试]", got.Triggers)
	}
	if len(got.Pitfalls) != 1 {
		t.Errorf("Pitfalls count = %d, want 1", len(got.Pitfalls))
	}
}

func TestProceduralMemory_SaveNotDirty(t *testing.T) {
	dir := t.TempDir()
	pm := NewProceduralMemory(dir)

	// 没有修改，保存应该无操作
	if err := pm.Save(); err != nil {
		t.Fatalf("Save on clean memory should not error: %v", err)
	}

	// 不应该创建目录
	if _, err := os.Stat(filepath.Join(dir, "skills")); !os.IsNotExist(err) {
		t.Error("skills dir should not be created when nothing to save")
	}
}

func TestProceduralMemory_LoadFromEmpty(t *testing.T) {
	dir := t.TempDir()
	pm, err := LoadProceduralMemory(dir)
	if err != nil {
		t.Fatalf("Load from empty dir: %v", err)
	}
	if pm.Count() != 0 {
		t.Errorf("Count = %d, want 0", pm.Count())
	}
}

func TestProceduralMemory_IncrementUsage(t *testing.T) {
	dir := t.TempDir()
	pm := NewProceduralMemory(dir)

	pm.SetSkill(&SkillEntry{ID: "inc_test", Name: "计数测试"})
	pm.IncrementUsage("inc_test")
	pm.IncrementUsage("inc_test")

	got, _ := pm.GetSkill("inc_test")
	if got.UsageCount != 2 {
		t.Errorf("UsageCount = %d, want 2", got.UsageCount)
	}
}

func TestProceduralMemory_BuildContext(t *testing.T) {
	dir := t.TempDir()
	pm := NewProceduralMemory(dir)

	pm.SetSkill(&SkillEntry{
		ID:          "ctx_test",
		Name:        "上下文测试",
		Category:    "network",
		Description: "测试上下文生成",
		Steps:       []string{"检查状态", "分析日志"},
		Triggers:    []string{"网络"},
		Pitfalls:    []string{"注意防火墙规则"},
		UsageCount:  5,
		Source:      "seed",
	})

	ctx := pm.BuildContext("网络故障排查")
	if ctx == "" {
		t.Fatal("BuildContext should return non-empty for matching query")
	}
	if !contains(ctx, "ctx_test") {
		t.Error("context should contain skill ID")
	}
	if !contains(ctx, "上下文测试") {
		t.Error("context should contain skill name")
	}
}

func TestProceduralMemory_BuildContext_Empty(t *testing.T) {
	dir := t.TempDir()
	pm := NewProceduralMemory(dir)

	ctx := pm.BuildContext("任何查询")
	if ctx != "" {
		t.Errorf("BuildContext on empty memory = %q, want empty", ctx)
	}
}

func TestProceduralMemory_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	// 创建并保存
	pm1 := NewProceduralMemory(dir)
	pm1.SetSkill(&SkillEntry{
		ID:          "roundtrip",
		Name:        "往返测试",
		Category:    "database",
		Description: "完整往返测试",
		Steps:       []string{"连接数据库", "执行查询"},
		ToolSeq:     []string{"mysql_query", "local_bash"},
		Triggers:    []string{"mysql", "数据库"},
		Pitfalls:    []string{"注意SQL注入"},
		Source:      "learned",
		SuccessRate: 0.92,
		UsageCount:  8,
	})
	pm1.Save()

	// 重新加载
	pm2, _ := LoadProceduralMemory(dir)
	got, ok := pm2.GetSkill("roundtrip")
	if !ok {
		t.Fatal("should load skill after roundtrip")
	}

	if got.Name != "往返测试" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.Category != "database" {
		t.Errorf("Category = %q", got.Category)
	}
	if len(got.Steps) != 2 {
		t.Errorf("Steps = %v", got.Steps)
	}
	if len(got.ToolSeq) != 2 {
		t.Errorf("ToolSeq = %v", got.ToolSeq)
	}
	if got.SuccessRate != 0.92 {
		t.Errorf("SuccessRate = %f", got.SuccessRate)
	}
	if got.UsageCount != 8 {
		t.Errorf("UsageCount = %d", got.UsageCount)
	}
	if got.Source != "learned" {
		t.Errorf("Source = %q", got.Source)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
