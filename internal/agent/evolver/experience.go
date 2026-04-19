// experience.go - 经验记忆系统
// Layer 2: 经验记忆 - 存储任务类型到最佳工具序列的映射
// 格式: ~/.opsxcli/agentv2/experience.jsonl
package evolver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Experience 单条经验记录
type Experience struct {
	TaskType     string    `json:"task_type"`      // 任务类型（如"磁盘分析"）
	ToolSequence []string  `json:"tool_sequence"`  // 最佳工具序列
	Hint         string    `json:"hint"`           // 学习到的提示
	SuccessRate  float64   `json:"success_rate"`   // 成功率 (0-1)
	UsageCount   int       `json:"usage_count"`    // 使用次数
	CreatedAt    time.Time `json:"created_at"`     // 创建时间
	LastUsedAt   time.Time `json:"last_used_at"`   // 最后使用时间
}

// ExperienceMemory 经验记忆管理器
type ExperienceMemory struct {
	mu           sync.RWMutex
	experiences  []*Experience     // 所有经验
	baseDir      string            // 存储目录
	maxEntries   int               // 最大条目数（防止无限增长）
}

// NewExperienceMemory 创建经验记忆
func NewExperienceMemory(baseDir string) *ExperienceMemory {
	return &ExperienceMemory{
		experiences: make([]*Experience, 0),
		baseDir:     baseDir,
		maxEntries:  1000, // 最多保存 1000 条经验
	}
}

// LoadExperienceMemory 从磁盘加载经验记忆
func LoadExperienceMemory(baseDir string) (*ExperienceMemory, error) {
	em := NewExperienceMemory(baseDir)

	filePath := filepath.Join(baseDir, "experience.jsonl")
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return em, nil // 文件不存在，返回空记忆
		}
		return nil, fmt.Errorf("打开经验文件失败: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var exp Experience
		if err := json.Unmarshal([]byte(line), &exp); err != nil {
			continue // 跳过损坏的行
		}

		em.experiences = append(em.experiences, &exp)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取经验文件失败: %w", err)
	}

	return em, nil
}

// Save 保存到磁盘
func (em *ExperienceMemory) Save() error {
	em.mu.Lock()
	defer em.mu.Unlock()

	filePath := filepath.Join(em.baseDir, "experience.jsonl")
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建经验文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, exp := range em.experiences {
		if err := encoder.Encode(exp); err != nil {
			return fmt.Errorf("写入经验记录失败: %w", err)
		}
	}

	return nil
}

// AddExperience 添加经验
func (em *ExperienceMemory) AddExperience(exp *Experience) {
	em.mu.Lock()
	defer em.mu.Unlock()

	// 检查是否已有相同任务类型和工具序列的经验
	key := em.makeKey(exp.TaskType, exp.ToolSequence)
	for i, existing := range em.experiences {
		if em.makeKey(existing.TaskType, existing.ToolSequence) == key {
			// 更新已有经验
			existing.SuccessRate = (existing.SuccessRate*float64(existing.UsageCount) + exp.SuccessRate) / float64(existing.UsageCount+1)
			existing.UsageCount++
			existing.LastUsedAt = time.Now()
			if len(exp.Hint) > len(existing.Hint) {
				existing.Hint = exp.Hint // 保留更详细的提示
			}
			em.experiences[i] = existing
			return
		}
	}

	// 新增经验
	em.experiences = append(em.experiences, exp)

	// 如果超过最大条目数，清理最旧的经验
	if len(em.experiences) > em.maxEntries {
		em.cleanup()
	}
}

// FindSimilar 查找相似经验（相同任务类型）
func (em *ExperienceMemory) FindSimilar(taskType string, toolSequence []string) *Experience {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// 首先找完全匹配任务类型的
	var candidates []*Experience
	for _, exp := range em.experiences {
		if exp.TaskType == taskType {
			candidates = append(candidates, exp)
		}
	}

	// 按成功率排序，返回最佳匹配
	if len(candidates) > 0 {
		best := candidates[0]
		for _, c := range candidates[1:] {
			if c.SuccessRate > best.SuccessRate {
				best = c
			}
		}
		return best
	}

	// 没有相同类型的，找工具序列相似的
	for _, exp := range em.experiences {
		if em.sequenceSimilarity(exp.ToolSequence, toolSequence) > 0.5 {
			return exp
		}
	}

	return nil
}

// FindByTaskType 按任务类型查找所有经验
func (em *ExperienceMemory) FindByTaskType(taskType string) []*Experience {
	em.mu.RLock()
	defer em.mu.RUnlock()

	result := make([]*Experience, 0)
	for _, exp := range em.experiences {
		if exp.TaskType == taskType {
			result = append(result, exp)
		}
	}
	return result
}

// GetBestExperience 获取某类任务的最佳经验
func (em *ExperienceMemory) GetBestExperience(taskType string) *Experience {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var best *Experience
	for _, exp := range em.experiences {
		if exp.TaskType != taskType {
			continue
		}
		if best == nil || exp.SuccessRate > best.SuccessRate {
			best = exp
		}
	}
	return best
}

// GetAllExperiences 获取所有经验（返回副本）
func (em *ExperienceMemory) GetAllExperiences() []Experience {
	em.mu.RLock()
	defer em.mu.RUnlock()

	result := make([]Experience, len(em.experiences))
	for i, exp := range em.experiences {
		result[i] = *exp
	}
	return result
}

// Count 获取经验总数
func (em *ExperienceMemory) Count() int {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return len(em.experiences)
}

// RemoveExperience 从经验列表中移除指定经验
func (em *ExperienceMemory) RemoveExperience(exp *Experience) {
	em.mu.Lock()
	defer em.mu.Unlock()

	key := em.makeKey(exp.TaskType, exp.ToolSequence)
	newList := make([]*Experience, 0, len(em.experiences))
	for _, e := range em.experiences {
		if em.makeKey(e.TaskType, e.ToolSequence) != key {
			newList = append(newList, e)
		}
	}
	em.experiences = newList
}

// GetTaskTypeStats 获取任务类型统计
func (em *ExperienceMemory) GetTaskTypeStats() map[string]int {
	em.mu.RLock()
	defer em.mu.RUnlock()

	stats := make(map[string]int)
	for _, exp := range em.experiences {
		stats[exp.TaskType]++
	}
	return stats
}

// DeleteByTaskType 删除某类任务的所有经验
func (em *ExperienceMemory) DeleteByTaskType(taskType string) int {
	em.mu.Lock()
	defer em.mu.Unlock()

	newList := make([]*Experience, 0, len(em.experiences))
	deleted := 0
	for _, exp := range em.experiences {
		if exp.TaskType == taskType {
			deleted++
			continue
		}
		newList = append(newList, exp)
	}
	em.experiences = newList
	return deleted
}

// Clear 清空所有经验
func (em *ExperienceMemory) Clear() {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.experiences = make([]*Experience, 0)
}

// makeKey 生成经验的唯一键
func (em *ExperienceMemory) makeKey(taskType string, toolSequence []string) string {
	return taskType + "::" + strings.Join(toolSequence, "→")
}

// sequenceSimilarity 计算两个工具序列的相似度 (0-1)
func (em *ExperienceMemory) sequenceSimilarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// 使用最长公共子序列 (LCS) 计算相似度
	lcs := em.lcsLength(a, b)
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	return float64(lcs) / float64(maxLen)
}

// lcsLength 计算最长公共子序列长度
func (em *ExperienceMemory) lcsLength(a, b []string) int {
	m, n := len(a), len(b)
	if m == 0 || n == 0 {
		return 0
	}

	// 使用一维 DP 节省空间
	prev := make([]int, n+1)
	curr := make([]int, n+1)

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1] + 1
			} else {
				if curr[j-1] > prev[j] {
					curr[j] = curr[j-1]
				} else {
					curr[j] = prev[j]
				}
			}
		}
		prev, curr = curr, prev
	}

	return prev[n]
}

// cleanup 清理过期/低效的经验
func (em *ExperienceMemory) cleanup() {
	// 保留规则：
	// 1. 最近使用过的（30 天内）
	// 2. 成功率高的 (>0.5)
	// 3. 使用次数多的 (>3 次)

	cutoff := time.Now().AddDate(0, 0, -30)
	newList := make([]*Experience, 0, em.maxEntries)

	for _, exp := range em.experiences {
		keep := false
		if exp.LastUsedAt.After(cutoff) {
			keep = true
		}
		if exp.SuccessRate > 0.5 {
			keep = true
		}
		if exp.UsageCount > 3 {
			keep = true
		}
		if keep {
			newList = append(newList, exp)
		}
	}

	// 如果仍然超过限制，按成功率排序保留前 maxEntries 条
	if len(newList) > em.maxEntries {
		em.sortByQuality(newList)
		newList = newList[:em.maxEntries]
	}

	em.experiences = newList
}

// sortByQuality 按综合质量排序
func (em *ExperienceMemory) sortByQuality(exps []*Experience) {
	// 使用成功率 × 使用次数作为综合质量分
	type scored struct {
		exp   *Experience
		score float64
	}

	scoredList := make([]scored, len(exps))
	for i, exp := range exps {
		scoredList[i] = scored{
			exp:   exp,
			score: exp.SuccessRate * float64(exp.UsageCount),
		}
	}

	// 按分数降序排列
	for i := 0; i < len(scoredList)-1; i++ {
		for j := i + 1; j < len(scoredList); j++ {
			if scoredList[j].score > scoredList[i].score {
				scoredList[i], scoredList[j] = scoredList[j], scoredList[i]
			}
		}
	}

	for i, s := range scoredList {
		exps[i] = s.exp
	}
}
