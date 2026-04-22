// token_stats.go - Token 消耗统计模块
// 按 Provider 和会话统计 Token 使用量，支持 TUI 面板展示
package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TokenRecord 单次请求的 Token 使用记录
type TokenRecord struct {
	Provider         string    `json:"provider"`          // 提供商 (deepseek, openai, claude, gemini)
	Model            string    `json:"model"`             // 模型名 (deepseek-chat, gpt-4)
	SessionID        string    `json:"session_id"`        // 会话 ID
	PromptTokens     int       `json:"prompt_tokens"`     // 输入 Token 数
	CompletionTokens int       `json:"completion_tokens"` // 输出 Token 数
	TotalTokens      int       `json:"total_tokens"`      // 总 Token 数
	Timestamp        time.Time `json:"timestamp"`         // 记录时间
}

// ProviderStats 按 Provider 汇总的统计
type ProviderStats struct {
	Provider         string  `json:"provider"`
	TotalRequests    int     `json:"total_requests"`
	TotalTokens      int     `json:"total_tokens"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	AvgTokensPerReq  float64 `json:"avg_tokens_per_req"`
	LastUsed         string  `json:"last_used"`
}

// SessionStats 按会话汇总的统计
type SessionStats struct {
	SessionID   string `json:"session_id"`
	TotalTokens int    `json:"total_tokens"`
	Requests    int    `json:"requests"`
}

// TokenStatsManager Token 统计管理器
type TokenStatsManager struct {
	mu       sync.RWMutex
	records  []TokenRecord
	filePath string // 持久化文件路径
	maxRows  int    // 最大记录条数（防止文件过大）
}

// 全局统计管理器实例
var globalTokenStats *TokenStatsManager

// InitTokenStats 初始化全局 Token 统计管理器
func InitTokenStats(dataDir string) error {
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("获取用户目录失败: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".opsxcli", "data")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	filePath := filepath.Join(dataDir, "token_stats.jsonl")
	mgr, err := NewTokenStatsManager(filePath)
	if err != nil {
		return fmt.Errorf("初始化 Token 统计管理器失败: %w", err)
	}

	globalTokenStats = mgr
	return nil
}

// NewTokenStatsManager 创建 Token 统计管理器
func NewTokenStatsManager(filePath string) (*TokenStatsManager, error) {
	mgr := &TokenStatsManager{
		records:  make([]TokenRecord, 0),
		filePath: filePath,
		maxRows:  10000, // 最多保留 10000 条记录
	}

	// 加载已有记录
	if err := mgr.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("加载 Token 统计记录失败: %w", err)
	}

	return mgr, nil
}

// Record 记录一次 Token 使用
func (m *TokenStatsManager) Record(provider, model, sessionID string, usage *Usage) {
	if usage == nil || usage.TotalTokens == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	record := TokenRecord{
		Provider:         provider,
		Model:            model,
		SessionID:        sessionID,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		Timestamp:        time.Now(),
	}

	m.records = append(m.records, record)

	// 超过最大记录数时，保留最新的一半
	if len(m.records) > m.maxRows {
		m.records = m.records[len(m.records)-m.maxRows/2:]
	}

	// 异步持久化（不阻塞调用方）
	go m.persist(record)
}

// persist 将单条记录追加到文件
func (m *TokenStatsManager) persist(record TokenRecord) {
	f, err := os.OpenFile(m.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	data, _ := json.Marshal(record)
	f.Write(data)
	f.Write([]byte("\n"))
}

// load 从文件加载历史记录
func (m *TokenStatsManager) load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	lines := 0
	for _, line := range splitLines(string(data)) {
		if line == "" {
			continue
		}
		var record TokenRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		m.records = append(m.records, record)
		lines++
	}

	// 超过上限时截断
	if len(m.records) > m.maxRows {
		m.records = m.records[len(m.records)-m.maxRows:]
	}

	_ = lines
	return nil
}

// GetProviderStats 获取按 Provider 汇总的统计
func (m *TokenStatsManager) GetProviderStats() []ProviderStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providerMap := make(map[string]*ProviderStats)

	for _, r := range m.records {
		stats, ok := providerMap[r.Provider]
		if !ok {
			stats = &ProviderStats{Provider: r.Provider}
			providerMap[r.Provider] = stats
		}
		stats.TotalRequests++
		stats.TotalTokens += r.TotalTokens
		stats.PromptTokens += r.PromptTokens
		stats.CompletionTokens += r.CompletionTokens
		stats.LastUsed = r.Timestamp.Format("2006-01-02 15:04")
	}

	// 计算平均值
	result := make([]ProviderStats, 0, len(providerMap))
	for _, stats := range providerMap {
		if stats.TotalRequests > 0 {
			stats.AvgTokensPerReq = float64(stats.TotalTokens) / float64(stats.TotalRequests)
		}
		result = append(result, *stats)
	}

	return result
}

// GetSessionStats 获取按会话汇总的统计
func (m *TokenStatsManager) GetSessionStats(sessionID string) SessionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := SessionStats{SessionID: sessionID}
	for _, r := range m.records {
		if r.SessionID == sessionID {
			stats.TotalTokens += r.TotalTokens
			stats.Requests++
		}
	}
	return stats
}

// GetTotalUsage 获取总使用量
func (m *TokenStatsManager) GetTotalUsage() (totalRequests, totalTokens int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, r := range m.records {
		totalRequests++
		totalTokens += r.TotalTokens
	}
	return
}

// GetRecentUsage 获取最近 N 小时的使用量
func (m *TokenStatsManager) GetRecentUsage(hours int) (requests, tokens int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	for _, r := range m.records {
		if r.Timestamp.After(cutoff) {
			requests++
			tokens += r.TotalTokens
		}
	}
	return
}

// RecordTokenUsage 记录 Token 使用（全局快捷方法）
func RecordTokenUsage(provider, model, sessionID string, usage *Usage) {
	if globalTokenStats != nil {
		globalTokenStats.Record(provider, model, sessionID, usage)
	}
}

// GetTokenStats 获取全局 Token 统计管理器
func GetTokenStats() *TokenStatsManager {
	return globalTokenStats
}

// FormatTokenCount 格式化 Token 数量显示
func FormatTokenCount(count int) string {
	if count >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(count)/1_000_000)
	}
	if count >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(count)/1_000)
	}
	return fmt.Sprintf("%d", count)
}

// splitLines 按换行符分割字符串
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
