package alert

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// HistoryManager 管理告警历史记录的持久化存储。
type HistoryManager struct {
	mu       sync.Mutex
	filePath string
	file     *os.File
	writer   *bufio.Writer
}

// NewHistoryManager 创建告警历史管理器。
// 文件路径默认为 dataDir/alerts_history.jsonl
func NewHistoryManager(dataDir string) (*HistoryManager, error) {
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户目录失败: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".opsxcli", "alert")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建告警数据目录失败: %w", err)
	}

	filePath := filepath.Join(dataDir, "alerts_history.jsonl")

	hm := &HistoryManager{
		filePath: filePath,
	}

	return hm, nil
}

// Record 记录一条告警历史。
func (hm *HistoryManager) Record(entry *HistoryEntry) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// 确保时间戳已设置
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	f, err := os.OpenFile(hm.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开历史文件失败: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("序列化历史记录失败: %w", err)
	}

	_, err = f.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("写入历史记录失败: %w", err)
	}

	return nil
}

// HistoryFilter 定义历史查询过滤条件。
type HistoryFilter struct {
	RuleName  string    `json:"rule_name,omitempty"`
	Level     Level     `json:"level,omitempty"`
	State     State     `json:"state,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

// Query 查询告警历史。
func (hm *HistoryManager) Query(filter *HistoryFilter) ([]*HistoryEntry, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	f, err := os.Open(hm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 文件不存在则返回空
		}
		return nil, fmt.Errorf("打开历史文件失败: %w", err)
	}
	defer f.Close()

	var results []*HistoryEntry
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry HistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // 跳过损坏的行
		}

		if hm.matchFilter(&entry, filter) {
			results = append(results, &entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取历史文件失败: %w", err)
	}

	// 按时间倒序排列
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	// 限制返回数量
	if filter != nil && filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	return results, nil
}

// GetStats 获取告警统计信息。
func (hm *HistoryManager) GetStats(rules []Rule, activeAlerts map[string]*Alert) *Stats {
	stats := &Stats{
		TotalRules:   len(rules),
		ActiveAlerts: len(activeAlerts),
		ByLevel:      make(map[Level]int),
	}

	for _, alert := range activeAlerts {
		switch alert.State {
		case StateFiring:
			stats.FiringAlerts++
		case StateSilenced:
			stats.SilencedAlerts++
		}
		stats.ByLevel[alert.Level]++
	}

	return stats
}

// RecentAlerts 获取最近的告警列表。
func (hm *HistoryManager) RecentAlerts(count int) ([]*HistoryEntry, error) {
	return hm.Query(&HistoryFilter{Limit: count})
}

// AlertsByRule 获取指定规则的告警历史。
func (hm *HistoryManager) AlertsByRule(ruleName string, limit int) ([]*HistoryEntry, error) {
	return hm.Query(&HistoryFilter{RuleName: ruleName, Limit: limit})
}

// AlertsByLevel 获取指定级别的告警历史。
func (hm *HistoryManager) AlertsByLevel(level Level, limit int) ([]*HistoryEntry, error) {
	return hm.Query(&HistoryFilter{Level: level, Limit: limit})
}

// Purge 清理指定时间之前的历史记录。
func (hm *HistoryManager) Purge(before time.Time) (int, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	f, err := os.Open(hm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("打开历史文件失败: %w", err)
	}

	var keep []*HistoryEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry HistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Timestamp.After(before) {
			keep = append(keep, &entry)
		}
	}
	f.Close()

	purged := 0
	// 读取原始行数
	origF, _ := os.Open(hm.filePath)
	if origF != nil {
		origScanner := bufio.NewScanner(origF)
		origCount := 0
		for origScanner.Scan() {
			if strings.TrimSpace(origScanner.Text()) != "" {
				origCount++
			}
		}
		origF.Close()
		purged = origCount - len(keep)
	}

	// 重写文件
	newF, err := os.Create(hm.filePath)
	if err != nil {
		return 0, fmt.Errorf("重写历史文件失败: %w", err)
	}
	defer newF.Close()

	writer := bufio.NewWriter(newF)
	for _, entry := range keep {
		data, _ := json.Marshal(entry)
		writer.Write(data)
		writer.WriteByte('\n')
	}
	writer.Flush()

	return purged, nil
}

// matchFilter 检查记录是否匹配过滤条件。
func (hm *HistoryManager) matchFilter(entry *HistoryEntry, filter *HistoryFilter) bool {
	if filter == nil {
		return true
	}
	if filter.RuleName != "" && entry.RuleName != filter.RuleName {
		return false
	}
	if filter.Level != "" && entry.Level != filter.Level {
		return false
	}
	if filter.State != "" && entry.State != filter.State {
		return false
	}
	if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
		return false
	}
	if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
		return false
	}
	return true
}

// Close 关闭历史管理器。
func (hm *HistoryManager) Close() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	if hm.writer != nil {
		hm.writer.Flush()
	}
	if hm.file != nil {
		return hm.file.Close()
	}
	return nil
}
