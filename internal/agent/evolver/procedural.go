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
	ID          string    `json:"id"`           // 技能ID (如 "network_diagnosis")
	Name        string    `json:"name"`         // 技能名称
	Category    string    `json:"category"`     // 分类 (network, database, system, security, deploy)
	Version     int       `json:"version"`      // 版本号（每次更新+1）
	Description string    `json:"description"`  // 技能描述
	Steps       []string  `json:"steps"`        // 操作步骤
	ToolSeq     []string  `json:"tool_seq"`     // 推荐工具序列
	Triggers    []string  `json:"triggers"`     // 触发条件（关键词）
	Pitfalls    []string  `json:"pitfalls"`     // 注意事项/陷阱
	SuccessRate float64   `json:"success_rate"` // 历史成功率
	UsageCount  int       `json:"usage_count"`  // 使用次数
	Source      string    `json:"source"`       // 来源: "seed" (内置种子), "learned" (自动学习), "manual" (用户创建)
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

// SeedBuiltinSkills 填充内置种子技能（v0.5.0: 5个运维种子Skill）
// 仅在技能不存在时创建，不覆盖已有技能
func (pm *ProceduralMemory) SeedBuiltinSkills() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()

	seeds := []*SkillEntry{
		{
			ID:          "local_common_ops",
			Name:        "常用本地操作",
			Category:    "system",
			Version:     1,
			Description: "文件管理、进程管理、服务管理等常用本地运维操作的标准流程",
			Steps: []string{
				"确认操作目标（文件/进程/服务）",
				"检查当前状态（ls/ps/systemctl status）",
				"执行操作（cp/mv/kill/systemctl restart）",
				"验证操作结果（再次检查状态）",
				"记录操作日志",
			},
			ToolSeq:     []string{"execute"},
			Triggers:    []string{"文件", "进程", "服务", "重启", "复制", "移动", "删除", "ls", "ps", "systemctl"},
			Pitfalls:    []string{"删除文件前确认路径，避免误删", "杀进程前确认无关联服务", "重启服务前通知相关方"},
			SuccessRate: 0.90,
			UsageCount:  0,
			Source:      "seed",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "install_software",
			Name:        "软件安装标准化流程",
			Category:    "deploy",
			Version:     1,
			Description: "通过 apt/yum/brew 等包管理器安装软件的标准化流程",
			Steps: []string{
				"确认目标系统类型（Ubuntu/CentOS/macOS）",
				"更新包索引（apt update / yum makecache）",
				"安装软件包（apt install / yum install / brew install）",
				"验证安装（which/version检查）",
				"配置环境变量（如需要）",
			},
			ToolSeq:     []string{"execute"},
			Triggers:    []string{"安装", "install", "apt", "yum", "brew", "软件", "包管理"},
			Pitfalls:    []string{"安装前先更新包索引避免依赖冲突", "注意区分系统版本选择正确包管理器", "安装后验证版本是否符合预期"},
			SuccessRate: 0.85,
			UsageCount:  0,
			Source:      "seed",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "risk_approval",
			Name:        "高危操作审批流程",
			Category:    "security",
			Version:     1,
			Description: "执行高危操作（rm -rf、重启服务、修改配置）前的标准审批和回滚准备流程",
			Steps: []string{
				"评估操作风险等级（低/中/高/严重）",
				"备份当前状态（配置备份/快照）",
				"准备回滚方案（明确的撤销步骤）",
				"通知相关方（如为生产环境）",
				"执行操作并监控结果",
				"确认操作成功或执行回滚",
			},
			ToolSeq:     []string{"execute", "file_read"},
			Triggers:    []string{"rm", "删除", "重启", "restart", "修改配置", "高危", "危险", "生产环境", "reboot", "shutdown"},
			Pitfalls:    []string{"永远先备份再操作", "确保回滚方案可行并已测试", "生产环境操作需要审批确认"},
			SuccessRate: 0.95,
			UsageCount:  0,
			Source:      "seed",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "network_diagnosis",
			Name:        "网络故障诊断",
			Category:    "network",
			Version:     1,
			Description: "网络连接异常、DNS问题、防火墙排查的标准诊断流程",
			Steps: []string{
				"检查本地网络接口状态（ip addr/ifconfig）",
				"测试基本连通性（ping 目标）",
				"检查DNS解析（nslookup/dig）",
				"测试端口连通性（telnet/nc）",
				"追踪路由路径（traceroute）",
				"检查防火墙规则（iptables/firewall-cmd）",
				"检查服务端状态",
			},
			ToolSeq:     []string{"execute", "ping", "nc", "traceroute"},
			Triggers:    []string{"网络", "连接", "ping", "DNS", "超时", "timeout", "拒绝", "refused", "防火墙", "端口", "不通"},
			Pitfalls:    []string{"先确认本地网络正常再排查远程", "注意 ICMP 可能被禁用导致 ping 失败但端口仍可达", "检查双向防火墙规则"},
			SuccessRate: 0.80,
			UsageCount:  0,
			Source:      "seed",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "basic_recovery",
			Name:        "基础故障恢复",
			Category:    "system",
			Version:     1,
			Description: "服务重启、日志清理、磁盘空间回收等基础故障恢复操作",
			Steps: []string{
				"识别故障类型（服务/磁盘/内存/网络）",
				"收集故障信息（日志/状态/资源使用）",
				"执行临时恢复（重启服务/清理空间/释放内存）",
				"验证恢复效果（检查服务状态和资源使用）",
				"分析根因并记录经验",
			},
			ToolSeq:     []string{"execute", "file_read", "file_search"},
			Triggers:    []string{"故障", "恢复", "OOM", "磁盘满", "服务挂", "崩溃", "重启服务", "清理", "recovery", "crash"},
			Pitfalls:    []string{"重启前确认不是系统性故障（如底层存储故障）", "清理日志前确认无需保留", "恢复后持续观察防止反复"},
			SuccessRate: 0.75,
			UsageCount:  0,
			Source:      "seed",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, seed := range seeds {
		if _, exists := pm.skills[seed.ID]; !exists {
			pm.skills[seed.ID] = seed
			pm.dirty = true
		}
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
