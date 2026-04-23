// factual.go - 事实记忆层 (GEP 三层记忆架构)
// Layer 1: 事实层 — MEMORY.md + USER.md
// 永久存储用户偏好、环境事实、常用配置，供 Agent 在每次会话中读取
// 与现有的 ExperienceMemory (经验层) 和 EnvironmentMemory (环境层) 互补
package evolver

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FactEntry 事实条目（键值对形式）
type FactEntry struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Category  string    `json:"category"` // preference, environment, convention, tool_quirk
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FactualMemory 事实记忆管理器
// 管理 MEMORY.md 和 USER.md 两个文件的读写
type FactualMemory struct {
	mu      sync.RWMutex
	baseDir string
	facts   map[string]*FactEntry // key -> entry
	dirty   bool                  // 是否有未持久化的变更
}

// NewFactualMemory 创建事实记忆管理器
func NewFactualMemory(baseDir string) *FactualMemory {
	return &FactualMemory{
		baseDir: baseDir,
		facts:   make(map[string]*FactEntry),
		dirty:   false,
	}
}

// LoadFactualMemory 从磁盘加载事实记忆
// 读取 MEMORY.md 和 USER.md 两个文件
func LoadFactualMemory(baseDir string) (*FactualMemory, error) {
	fm := NewFactualMemory(baseDir)

	// 读取 MEMORY.md — 环境事实和工具特性
	memoryFile := filepath.Join(baseDir, "MEMORY.md")
	if err := fm.parseMarkdownFile(memoryFile, "environment"); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("加载 MEMORY.md 失败: %w", err)
		}
	}

	// 读取 USER.md — 用户偏好和习惯
	userFile := filepath.Join(baseDir, "USER.md")
	if err := fm.parseMarkdownFile(userFile, "preference"); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("加载 USER.md 失败: %w", err)
		}
	}

	return fm, nil
}

// parseMarkdownFile 解析 Markdown 格式的事实文件
// 格式约定:
//
//	## 类别标题
//	- key: value
//	- key: 多行值用缩进
func (fm *FactualMemory) parseMarkdownFile(filePath string, defaultCategory string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	currentCategory := defaultCategory
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// 检测分类标题 (## Category)
		if strings.HasPrefix(line, "## ") {
			currentCategory = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}

		// 跳过空行和注释
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析键值对: "- key: value" 或 "- key：value"
		if strings.HasPrefix(strings.TrimSpace(line), "- ") {
			entry := fm.parseLineEntry(line, currentCategory)
			if entry != nil {
				fm.facts[entry.Key] = entry
			}
		}
	}

	return scanner.Err()
}

// parseLineEntry 从一行 Markdown 列表项解析事实条目
func (fm *FactualMemory) parseLineEntry(line string, category string) *FactEntry {
	content := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))

	// 支持 "key: value" 和 "key：value" 两种分隔符
	var key, value string
	if idx := strings.Index(content, ": "); idx >= 0 {
		key = strings.TrimSpace(content[:idx])
		value = strings.TrimSpace(content[idx+2:])
	} else if idx := strings.Index(content, "："); idx >= 0 {
		key = strings.TrimSpace(content[:idx])
		value = strings.TrimSpace(content[len("：")+idx:])
	} else {
		// 没有分隔符，跳过
		return nil
	}

	if key == "" {
		return nil
	}

	now := time.Now()
	return &FactEntry{
		Key:       key,
		Value:     value,
		Category:  category,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Save 保存事实记忆到磁盘
// 同时生成 MEMORY.md 和 USER.md
func (fm *FactualMemory) Save() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if !fm.dirty {
		return nil
	}

	// 按分类分组
	envFacts := make([]*FactEntry, 0)
	userFacts := make([]*FactEntry, 0)

	for _, entry := range fm.facts {
		switch entry.Category {
		case "environment", "tool_quirk", "convention":
			envFacts = append(envFacts, entry)
		case "preference", "user_profile":
			userFacts = append(userFacts, entry)
		default:
			envFacts = append(envFacts, entry)
		}
	}

	// 保存 MEMORY.md
	if err := fm.writeMarkdownFile(
		filepath.Join(fm.baseDir, "MEMORY.md"),
		"OpsXCLI Agent 环境记忆",
		"此文件由 Agent 自动维护，记录环境事实和工具特性",
		envFacts,
	); err != nil {
		return fmt.Errorf("保存 MEMORY.md 失败: %w", err)
	}

	// 保存 USER.md
	if err := fm.writeMarkdownFile(
		filepath.Join(fm.baseDir, "USER.md"),
		"OpsXCLI Agent 用户偏好",
		"此文件由 Agent 自动维护，记录用户偏好和习惯",
		userFacts,
	); err != nil {
		return fmt.Errorf("保存 USER.md 失败: %w", err)
	}

	fm.dirty = false
	return nil
}

// writeMarkdownFile 将事实条目写入 Markdown 文件
func (fm *FactualMemory) writeMarkdownFile(filePath string, title string, description string, entries []*FactEntry) error {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	sb.WriteString(fmt.Sprintf("> %s\n\n", description))

	if len(entries) == 0 {
		sb.WriteString("*(暂无记录)*\n")
	} else {
		// 按子分类分组
		groups := make(map[string][]*FactEntry)
		for _, e := range entries {
			groups[e.Category] = append(groups[e.Category], e)
		}

		for cat, items := range groups {
			categoryTitle := fm.categoryTitle(cat)
			sb.WriteString(fmt.Sprintf("## %s\n\n", categoryTitle))
			for _, item := range items {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", item.Key, item.Value))
			}
			sb.WriteString("\n")
		}
	}

	// 写入文件
	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}

// categoryTitle 将分类标识转为可读标题
func (fm *FactualMemory) categoryTitle(category string) string {
	titles := map[string]string{
		"environment":  "环境事实",
		"tool_quirk":   "工具特性",
		"convention":   "约定俗成",
		"preference":   "用户偏好",
		"user_profile": "用户画像",
	}
	if title, ok := titles[category]; ok {
		return title
	}
	return category
}

// === 查询接口 ===

// GetFact 获取事实
func (fm *FactualMemory) GetFact(key string) (string, bool) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if entry, ok := fm.facts[key]; ok {
		return entry.Value, true
	}
	return "", false
}

// SetFact 设置事实（新增或更新）
func (fm *FactualMemory) SetFact(key, value, category string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	now := time.Now()
	if existing, ok := fm.facts[key]; ok {
		existing.Value = value
		existing.Category = category
		existing.UpdatedAt = now
	} else {
		fm.facts[key] = &FactEntry{
			Key:       key,
			Value:     value,
			Category:  category,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	fm.dirty = true
}

// DeleteFact 删除事实
func (fm *FactualMemory) DeleteFact(key string) bool {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if _, ok := fm.facts[key]; ok {
		delete(fm.facts, key)
		fm.dirty = true
		return true
	}
	return false
}

// GetFactsByCategory 按分类获取事实
func (fm *FactualMemory) GetFactsByCategory(category string) []*FactEntry {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	result := make([]*FactEntry, 0)
	for _, entry := range fm.facts {
		if entry.Category == category {
			result = append(result, entry)
		}
	}
	return result
}

// GetAllFacts 获取所有事实
func (fm *FactualMemory) GetAllFacts() []*FactEntry {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	result := make([]*FactEntry, 0, len(fm.facts))
	for _, entry := range fm.facts {
		result = append(result, entry)
	}
	return result
}

// Count 获取事实总数
func (fm *FactualMemory) Count() int {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return len(fm.facts)
}

// BuildMemoryContext 构建事实层的 Prompt 上下文
// 返回格式化的 Markdown 文本，供 MemoryInjector 注入
func (fm *FactualMemory) BuildMemoryContext() string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if len(fm.facts) == 0 {
		return ""
	}

	var sb strings.Builder

	// 用户偏好
	prefs := make([]*FactEntry, 0)
	for _, e := range fm.facts {
		if e.Category == "preference" || e.Category == "user_profile" {
			prefs = append(prefs, e)
		}
	}
	if len(prefs) > 0 {
		sb.WriteString("👤 用户偏好:\n")
		for _, p := range prefs {
			sb.WriteString(fmt.Sprintf("   • %s: %s\n", p.Key, p.Value))
		}
	}

	// 环境事实
	envs := make([]*FactEntry, 0)
	for _, e := range fm.facts {
		if e.Category == "environment" {
			envs = append(envs, e)
		}
	}
	if len(envs) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("🌍 环境事实:\n")
		for _, e := range envs {
			sb.WriteString(fmt.Sprintf("   • %s: %s\n", e.Key, e.Value))
		}
	}

	// 工具特性
	quirks := make([]*FactEntry, 0)
	for _, e := range fm.facts {
		if e.Category == "tool_quirk" {
			quirks = append(quirks, e)
		}
	}
	if len(quirks) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("🔧 工具特性:\n")
		for _, e := range quirks {
			sb.WriteString(fmt.Sprintf("   • %s: %s\n", e.Key, e.Value))
		}
	}

	return sb.String()
}

// MergeFromEvolveResult 从进化结果中提取事实并合并
// 当 Evolver 发现新的事实时调用
func (fm *FactualMemory) MergeFromEvolveResult(facts map[string]string, category string) int {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	added := 0
	now := time.Now()

	for key, value := range facts {
		if existing, ok := fm.facts[key]; ok {
			// 已存在，更新值
			if existing.Value != value {
				existing.Value = value
				existing.UpdatedAt = now
				fm.dirty = true
			}
		} else {
			// 新增
			fm.facts[key] = &FactEntry{
				Key:       key,
				Value:     value,
				Category:  category,
				CreatedAt: now,
				UpdatedAt: now,
			}
			fm.dirty = true
			added++
		}
	}

	return added
}
