package evolver

import (
	"os"
	"path/filepath"
	"testing"
)

// ===================== FactualMemory 基础测试 =====================

func TestNewFactualMemory(t *testing.T) {
	fm := NewFactualMemory(t.TempDir())
	if fm.Count() != 0 {
		t.Errorf("新创建的 FactualMemory 应该为空, got %d", fm.Count())
	}
}

func TestFactualMemory_SetAndGet(t *testing.T) {
	fm := NewFactualMemory(t.TempDir())

	fm.SetFact("操作系统", "macOS", "environment")
	val, ok := fm.GetFact("操作系统")
	if !ok {
		t.Error("应该能找到已设置的事实")
	}
	if val != "macOS" {
		t.Errorf("got %q, want %q", val, "macOS")
	}
}

func TestFactualMemory_Update(t *testing.T) {
	fm := NewFactualMemory(t.TempDir())

	fm.SetFact("操作系统", "macOS", "environment")
	fm.SetFact("操作系统", "Linux", "environment")

	val, ok := fm.GetFact("操作系统")
	if !ok {
		t.Error("应该能找到更新后的事实")
	}
	if val != "Linux" {
		t.Errorf("got %q, want %q", val, "Linux")
	}
	if fm.Count() != 1 {
		t.Errorf("更新不应增加计数, got %d", fm.Count())
	}
}

func TestFactualMemory_Delete(t *testing.T) {
	fm := NewFactualMemory(t.TempDir())

	fm.SetFact("key1", "value1", "environment")
	fm.SetFact("key2", "value2", "preference")

	if !fm.DeleteFact("key1") {
		t.Error("删除已存在的 key 应返回 true")
	}
	if fm.DeleteFact("不存在") {
		t.Error("删除不存在的 key 应返回 false")
	}
	if fm.Count() != 1 {
		t.Errorf("删除后应剩 1 条, got %d", fm.Count())
	}
}

func TestFactualMemory_GetFactsByCategory(t *testing.T) {
	fm := NewFactualMemory(t.TempDir())

	fm.SetFact("env1", "v1", "environment")
	fm.SetFact("env2", "v2", "environment")
	fm.SetFact("pref1", "v3", "preference")

	envFacts := fm.GetFactsByCategory("environment")
	if len(envFacts) != 2 {
		t.Errorf("environment 分类应有 2 条, got %d", len(envFacts))
	}

	prefFacts := fm.GetFactsByCategory("preference")
	if len(prefFacts) != 1 {
		t.Errorf("preference 分类应有 1 条, got %d", len(prefFacts))
	}

	emptyFacts := fm.GetFactsByCategory("不存在")
	if len(emptyFacts) != 0 {
		t.Errorf("不存在的分类应返回空, got %d", len(emptyFacts))
	}
}

func TestFactualMemory_GetAllFacts(t *testing.T) {
	fm := NewFactualMemory(t.TempDir())

	fm.SetFact("k1", "v1", "environment")
	fm.SetFact("k2", "v2", "preference")
	fm.SetFact("k3", "v3", "tool_quirk")

	all := fm.GetAllFacts()
	if len(all) != 3 {
		t.Errorf("应有 3 条事实, got %d", len(all))
	}
}

func TestFactualMemory_GetNonExistent(t *testing.T) {
	fm := NewFactualMemory(t.TempDir())

	_, ok := fm.GetFact("不存在")
	if ok {
		t.Error("不存在的 key 应返回 false")
	}
}

// ===================== 持久化测试 =====================

func TestFactualMemory_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	// 创建并保存
	fm := NewFactualMemory(dir)
	fm.SetFact("操作系统", "macOS 14.5", "environment")
	fm.SetFact("默认编辑器", "vim", "preference")
	fm.SetFact("SSH超时", "30秒", "tool_quirk")

	if err := fm.Save(); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(filepath.Join(dir, "MEMORY.md")); os.IsNotExist(err) {
		t.Error("MEMORY.md 应该被创建")
	}
	if _, err := os.Stat(filepath.Join(dir, "USER.md")); os.IsNotExist(err) {
		t.Error("USER.md 应该被创建")
	}

	// 重新加载
	fm2, err := LoadFactualMemory(dir)
	if err != nil {
		t.Fatalf("LoadFactualMemory 失败: %v", err)
	}

	// 验证数据一致性
	val, ok := fm2.GetFact("操作系统")
	if !ok || val != "macOS 14.5" {
		t.Errorf("加载后'操作系统' = %q, want %q", val, "macOS 14.5")
	}

	val, ok = fm2.GetFact("默认编辑器")
	if !ok || val != "vim" {
		t.Errorf("加载后'默认编辑器' = %q, want %q", val, "vim")
	}

	val, ok = fm2.GetFact("SSH超时")
	if !ok || val != "30秒" {
		t.Errorf("加载后'SSH超时' = %q, want %q", val, "30秒")
	}
}

func TestFactualMemory_SaveNotDirty(t *testing.T) {
	dir := t.TempDir()
	fm := NewFactualMemory(dir)

	// 没有修改，Save 应该不写文件
	if err := fm.Save(); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	// 文件不应存在（因为 dirty=false）
	if _, err := os.Stat(filepath.Join(dir, "MEMORY.md")); !os.IsNotExist(err) {
		t.Error("MEMORY.md 不应该被创建（因为没有修改）")
	}
}

func TestFactualMemory_LoadFromEmpty(t *testing.T) {
	dir := t.TempDir()

	fm, err := LoadFactualMemory(dir)
	if err != nil {
		t.Fatalf("加载空目录应成功: %v", err)
	}
	if fm.Count() != 0 {
		t.Errorf("空目录应加载为空, got %d", fm.Count())
	}
}

// ===================== Markdown 解析测试 =====================

func TestFactualMemory_ParseMarkdownFile(t *testing.T) {
	dir := t.TempDir()

	// 手动创建 MEMORY.md
	content := `# 环境记忆

> 测试用

## 环境事实

- 操作系统: macOS 14.5
- Go版本: 1.24
- 架构: arm64

## 工具特性

- SSH默认端口: 22
- MySQL超时: 10秒
`
	if err := os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fm := NewFactualMemory(dir)
	if err := fm.parseMarkdownFile(filepath.Join(dir, "MEMORY.md"), "environment"); err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if fm.Count() != 5 {
		t.Errorf("应解析出 5 条事实, got %d", fm.Count())
	}

	val, ok := fm.GetFact("操作系统")
	if !ok || val != "macOS 14.5" {
		t.Errorf("'操作系统' = %q, want %q", val, "macOS 14.5")
	}

	val, ok = fm.GetFact("MySQL超时")
	if !ok || val != "10秒" {
		t.Errorf("'MySQL超时' = %q, want %q", val, "10秒")
	}
}

func TestFactualMemory_ParseChineseColon(t *testing.T) {
	dir := t.TempDir()

	content := `# 测试

- 用户名：张三
- 邮箱：test@example.com
`
	if err := os.WriteFile(filepath.Join(dir, "USER.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fm := NewFactualMemory(dir)
	if err := fm.parseMarkdownFile(filepath.Join(dir, "USER.md"), "preference"); err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	val, ok := fm.GetFact("用户名")
	if !ok || val != "张三" {
		t.Errorf("'用户名' = %q, want %q", val, "张三")
	}
}

// ===================== BuildMemoryContext 测试 =====================

func TestFactualMemory_BuildMemoryContext_Empty(t *testing.T) {
	fm := NewFactualMemory(t.TempDir())

	ctx := fm.BuildMemoryContext()
	if ctx != "" {
		t.Errorf("空记忆应返回空字符串, got %q", ctx)
	}
}

func TestFactualMemory_BuildMemoryContext_WithFacts(t *testing.T) {
	fm := NewFactualMemory(t.TempDir())
	fm.SetFact("操作系统", "Linux", "environment")
	fm.SetFact("默认编辑器", "nvim", "preference")
	fm.SetFact("SSH超时", "30秒", "tool_quirk")

	ctx := fm.BuildMemoryContext()

	// 验证包含各分类
	if ctx == "" {
		t.Error("有事实时应返回非空上下文")
	}
}

// ===================== MergeFromEvolveResult 测试 =====================

func TestFactualMemory_MergeFromEvolveResult(t *testing.T) {
	fm := NewFactualMemory(t.TempDir())
	fm.SetFact("已有key", "旧值", "environment")

	newFacts := map[string]string{
		"已有key":  "新值",
		"新发现key": "新值",
	}

	added := fm.MergeFromEvolveResult(newFacts, "environment")
	if added != 1 {
		t.Errorf("应新增 1 条, got %d", added)
	}

	val, _ := fm.GetFact("已有key")
	if val != "新值" {
		t.Errorf("已有 key 应被更新, got %q", val)
	}

	val, _ = fm.GetFact("新发现key")
	if val != "新值" {
		t.Errorf("新 key 应被添加, got %q", val)
	}
}

// ===================== Round-trip 测试 =====================

func TestFactualMemory_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	// 创建并保存
	fm1 := NewFactualMemory(dir)
	fm1.SetFact("操作系统", "Linux", "environment")
	fm1.SetFact("Go版本", "1.24", "environment")
	fm1.SetFact("默认编辑器", "vim", "preference")
	fm1.SetFact("sudo偏好", "默认启用", "preference")
	fm1.SetFact("SSH端口", "22", "tool_quirk")
	fm1.Save()

	// 加载
	fm2, err := LoadFactualMemory(dir)
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}

	// 验证所有数据
	tests := []struct {
		key  string
		want string
	}{
		{"操作系统", "Linux"},
		{"Go版本", "1.24"},
		{"默认编辑器", "vim"},
		{"sudo偏好", "默认启用"},
		{"SSH端口", "22"},
	}

	for _, tt := range tests {
		val, ok := fm2.GetFact(tt.key)
		if !ok {
			t.Errorf("key %q 应存在", tt.key)
		}
		if val != tt.want {
			t.Errorf("key %q = %q, want %q", tt.key, val, tt.want)
		}
	}

	// 更新并保存
	fm2.SetFact("操作系统", "macOS", "environment")
	fm2.Save()

	// 再次加载验证更新
	fm3, err := LoadFactualMemory(dir)
	if err != nil {
		t.Fatalf("二次加载失败: %v", err)
	}

	val, _ := fm3.GetFact("操作系统")
	if val != "macOS" {
		t.Errorf("更新后'操作系统' = %q, want %q", val, "macOS")
	}
}
