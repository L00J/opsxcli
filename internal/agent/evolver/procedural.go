// procedural.go - 程序层记忆 (Procedural Memory Layer)
// v0.5.0 三层记忆架构的第二层: SKILL_xxx.md 版本化技能文档
// 管理 Agent 在执行运维任务中习得的可复用操作技能
package evolver

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// SkillEntry 技能条目 — 一个可复用的运维操作技能
// ═══════════════════════════════════════════════════════════════
type SkillEntry struct {
	ID          string    `json:"id"`            // 技能ID (如 "network_diagnosis")
	Name        string    `json:"name"`          // 技能名称
	Category    string    `json:"category"`      // 分类 (network, database, system, security, deploy)
	Version     int       `json:"version"`       // 版本号（每次更新+1）
	Description string    `json:"description"`   // 技能描述
	Steps       []string  `json:"steps"`         // 操作步骤
	ToolSeq     []string  `json:"tool_seq"`      // 推荐工具序列
	Triggers    []string  `json:"triggers"`      // 触发条件（关键词）
	Pitfalls    []string  `json:"pitfalls"`      // 注意事项/陷阱
	SuccessRate float64   `json:"success_rate"`  // 历史成功率
	UsageCount  int       `json:"usage_count"`   // 使用次数
	Source      string    `json:"source"`        // 来源: "seed" (内置种子), "learned" (自动学习), "manual" (用户创建)
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProceduralMemory 程序层记忆管理器
// 管理 SKILL_xxx.md 格式的技能文档
type ProceduralMemory struct {
	mu      sync.RWMutex
	baseDir string
	skills  map[string]*SkillEntry // ID -> SkillEntry
	dirty   bool
}

// ═══════════════════════════════════════════════════════════════
// 创建与加载
// ═══════════════════════════════════════════════════════════════

// NewProceduralMemory 创建空的程序层记忆
func NewProceduralMemory(baseDir string) *ProceduralMemory {
	return &ProceduralMemory{
		baseDir: baseDir,
		skills:  make(map[string]*SkillEntry),
	}
}

// LoadProceduralMemory 从磁盘加载程序层记忆
// 读取 skills/ 目录下所有 SKILL_xxx.md 文件
func LoadProceduralMemory(baseDir string) (*ProceduralMemory, error) {
	pm := NewProceduralMemory(baseDir)

	skillsDir := filepath.Join(baseDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return pm, nil // 目录不存在，返回空记忆
		}
		return nil, fmt.Errorf("读取技能目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "SKILL_") || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(skillsDir, entry.Name())
		skill, err := parseSkillFile(filePath)
		if err != nil {
			continue // 跳过解析失败的文件
		}
		pm.skills[skill.ID] = skill
	}

	return pm, nil
}

// parseSkillFile 解析 SKILL_xxx.md 文件
// 文件格式:
//
//	# SKILL: network_diagnosis
//	> 名称: 网络诊断
//	> 分类: network
//	> 版本: 1
//	> 来源: seed
//	> 成功率: 0.85
//	> 使用次数: 10
//
//	## 描述
//	...
//
//	## 触发条件
//	- 关键词1
//	- 关键词2
//
//	## 操作步骤
//	1. 步骤1
//	2. 步骤2
//
//	## 推荐工具
//	- local_bash
//	- ssh_execute
//
//	## 注意事项
//	- 陷阱1
func parseSkillFile(filePath string) (*SkillEntry, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	skill := &SkillEntry{
		Source:    "seed",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 从文件名提取 ID: SKILL_network_diagnosis.md -> network_diagnosis
	baseName := strings.TrimSuffix(filepath.Base(filePath), ".md")
	skill.ID = strings.TrimPrefix(baseName, "SKILL_")

	// 解析头部元数据
	skill.Name = extractMetaField(content, "名称")
	skill.Category = extractMetaField(content, "分类")
	skill.Source = extractMetaField(content, "来源")
	if skill.Source == "" {
		skill.Source = "seed"
	}

	versionStr := extractMetaField(content, "版本")
	if versionStr != "" {
		fmt.Sscanf(versionStr, "%d", &skill.Version)
	}
	if skill.Version == 0 {
		skill.Version = 1
	}

	rateStr := extractMetaField(content, "成功率")
	if rateStr != "" {
		fmt.Sscanf(rateStr, "%f", &skill.SuccessRate)
	}

	countStr := extractMetaField(content, "使用次数")
	if countStr != "" {
		fmt.Sscanf(countStr, "%d", &skill.UsageCount)
	}

	// 解析各节
	skill.Description = extractSection(content, "描述")
	skill.Triggers = extractListSection(content, "触发条件")
	skill.Steps = extractNumberedList(content, "操作步骤")
	skill.ToolSeq = extractListSection(content, "推荐工具")
	skill.Pitfalls = extractListSection(content, "注意事项")

	return skill, nil
}

// extractMetaField 从 Markdown 头部提取 > 字段: value
func extractMetaField(content, field string) string {
	re := regexp.MustCompile(`(?m)> ` + regexp.QuoteMeta(field) + `\s*[:：]\s*(.+)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractSection 提取 Markdown 节内容（## Title 之后的文本直到下一个 ## ）
func extractSection(content, title string) string {
	re := regexp.MustCompile(`(?s)## ` + regexp.QuoteMeta(title) + `\s*\n(.*?)(?:\n## |\z)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractListSection 提取列表节内容（- item）
func extractListSection(content, title string) []string {
	section := extractSection(content, title)
	if section == "" {
		return nil
	}
	items := make([]string, 0)
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			items = append(items, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
		}
	}
	return items
}

// extractNumberedList 提取编号列表（1. item）
func extractNumberedList(content, title string) []string {
	section := extractSection(content, title)
	if section == "" {
		return nil
	}
	items := make([]string, 0)
	numRe := regexp.MustCompile(`^\d+\.\s+`)
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if numRe.MatchString(line) {
			items = append(items, numRe.ReplaceAllString(line, ""))
		}
	}
	return items
}

// ═══════════════════════════════════════════════════════════════
// CRUD 操作
// ═══════════════════════════════════════════════════════════════

// GetSkill 获取技能
func (pm *ProceduralMemory) GetSkill(id string) (*SkillEntry, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	s, ok := pm.skills[id]
	if !ok {
		return nil, false
	}
	// 返回副本
	cp := *s
	return &cp, true
}

// SetSkill 设置/更新技能
func (pm *ProceduralMemory) SetSkill(skill *SkillEntry) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if existing, ok := pm.skills[skill.ID]; ok {
		// 版本递增
		skill.Version = existing.Version + 1
		skill.UsageCount = existing.UsageCount
		skill.CreatedAt = existing.CreatedAt
	} else {
		// 新技能，初始版本为 1
		if skill.Version == 0 {
			skill.Version = 1
		}
	}
	skill.UpdatedAt = time.Now()
	pm.skills[skill.ID] = skill
	pm.dirty = true
}

// DeleteSkill 删除技能
func (pm *ProceduralMemory) DeleteSkill(id string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, ok := pm.skills[id]; !ok {
		return false
	}
	delete(pm.skills, id)
	pm.dirty = true
	return true
}

// IncrementUsage 增加技能使用计数
func (pm *ProceduralMemory) IncrementUsage(id string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if s, ok := pm.skills[id]; ok {
		s.UsageCount++
		s.UpdatedAt = time.Now()
		pm.dirty = true
	}
}

// GetAllSkills 获取所有技能
func (pm *ProceduralMemory) GetAllSkills() []*SkillEntry {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]*SkillEntry, 0, len(pm.skills))
	for _, s := range pm.skills {
		cp := *s
		result = append(result, &cp)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UsageCount > result[j].UsageCount
	})
	return result
}

// GetSkillsByCategory 按分类获取技能
func (pm *ProceduralMemory) GetSkillsByCategory(category string) []*SkillEntry {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]*SkillEntry, 0)
	for _, s := range pm.skills {
		if s.Category == category {
			cp := *s
			result = append(result, &cp)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UsageCount > result[j].UsageCount
	})
	return result
}

// FindMatchingSkills 根据查询匹配相关技能
func (pm *ProceduralMemory) FindMatchingSkills(query string) []*SkillEntry {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	q := strings.ToLower(query)
	result := make([]*SkillEntry, 0)

	for _, s := range pm.skills {
		// 检查触发条件
		for _, trigger := range s.Triggers {
			if strings.Contains(q, strings.ToLower(trigger)) {
				cp := *s
				result = append(result, &cp)
				break
			}
		}
		// 检查名称和描述
		if strings.Contains(q, strings.ToLower(s.Name)) ||
			strings.Contains(q, strings.ToLower(s.ID)) {
			cp := *s
			result = append(result, &cp)
		}
	}

	// 去重
	seen := make(map[string]bool)
	unique := make([]*SkillEntry, 0, len(result))
	for _, s := range result {
		if !seen[s.ID] {
			seen[s.ID] = true
			unique = append(unique, s)
		}
	}

	sort.Slice(unique, func(i, j int) bool {
		return unique[i].UsageCount > unique[j].UsageCount
	})
	return unique
}

// Count 获取技能总数
func (pm *ProceduralMemory) Count() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.skills)
}

// ═══════════════════════════════════════════════════════════════
// 持久化
// ═══════════════════════════════════════════════════════════════

// Save 保存所有技能到磁盘
func (pm *ProceduralMemory) Save() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.dirty {
		return nil
	}

	skillsDir := filepath.Join(pm.baseDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}

	for _, s := range pm.skills {
		if err := pm.writeSkillFile(skillsDir, s); err != nil {
			return err
		}
	}

	pm.dirty = false
	return nil
}

// writeSkillFile 将技能写入 Markdown 文件
func (pm *ProceduralMemory) writeSkillFile(dir string, s *SkillEntry) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# SKILL: %s\n", s.ID))
	sb.WriteString(fmt.Sprintf("> 名称: %s\n", s.Name))
	sb.WriteString(fmt.Sprintf("> 分类: %s\n", s.Category))
	sb.WriteString(fmt.Sprintf("> 版本: %d\n", s.Version))
	sb.WriteString(fmt.Sprintf("> 来源: %s\n", s.Source))
	sb.WriteString(fmt.Sprintf("> 成功率: %.2f\n", s.SuccessRate))
	sb.WriteString(fmt.Sprintf("> 使用次数: %d\n\n", s.UsageCount))

	// 描述
	sb.WriteString("## 描述\n\n")
	sb.WriteString(s.Description + "\n\n")

	// 触发条件
	if len(s.Triggers) > 0 {
		sb.WriteString("## 触发条件\n\n")
		for _, t := range s.Triggers {
			sb.WriteString(fmt.Sprintf("- %s\n", t))
		}
		sb.WriteString("\n")
	}

	// 操作步骤
	if len(s.Steps) > 0 {
		sb.WriteString("## 操作步骤\n\n")
		for i, step := range s.Steps {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
		sb.WriteString("\n")
	}

	// 推荐工具
	if len(s.ToolSeq) > 0 {
		sb.WriteString("## 推荐工具\n\n")
		for _, t := range s.ToolSeq {
			sb.WriteString(fmt.Sprintf("- %s\n", t))
		}
		sb.WriteString("\n")
	}

	// 注意事项
	if len(s.Pitfalls) > 0 {
		sb.WriteString("## 注意事项\n\n")
		for _, p := range s.Pitfalls {
			sb.WriteString(fmt.Sprintf("- %s\n", p))
		}
		sb.WriteString("\n")
	}

	filePath := filepath.Join(dir, fmt.Sprintf("SKILL_%s.md", s.ID))
	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}

// ═══════════════════════════════════════════════════════════════
// 上下文生成
// ═══════════════════════════════════════════════════════════════

// BuildContext 为 Prompt 生成程序层上下文
// 根据查询匹配相关技能，返回格式化的技能提示
func (pm *ProceduralMemory) BuildContext(query string) string {
	skills := pm.FindMatchingSkills(query)
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, s := range skills {
		if s.UsageCount < 1 && s.Source != "seed" {
			continue // 跳过未使用过的非种子技能
		}

		sb.WriteString(fmt.Sprintf("📖 技能[%s]: %s\n", s.ID, s.Name))
		if s.Description != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", s.Description))
		}
		if len(s.Steps) > 0 {
			sb.WriteString("   步骤: ")
			for i, step := range s.Steps {
				if i > 0 {
					sb.WriteString(" → ")
				}
				if len(step) > 30 {
					sb.WriteString(step[:30] + "...")
				} else {
					sb.WriteString(step)
				}
			}
			sb.WriteString("\n")
		}
		if len(s.Pitfalls) > 0 {
			sb.WriteString(fmt.Sprintf("   ⚠️ %s\n", s.Pitfalls[0]))
		}
	}

	return sb.String()
}
