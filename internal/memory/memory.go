package memory

import (
	"encoding/json"
	"fmt"
	"time"

	"opsxcli/internal/db"
	"opsxcli/internal/llm"
)

// MemoryType 记忆类型
type MemoryType string

const (
	MemoryTypeShortTerm MemoryType = "short_term" // 短暂记忆（会话级别）
	MemoryTypeLongTerm  MemoryType = "long_term"  // 永久记忆（持久化）
)

// Memory 记忆项
type Memory struct {
	ID          int64       `json:"id"`
	Type        MemoryType  `json:"type"`
	SessionID   *string     `json:"session_id,omitempty"`   // 会话ID（短暂记忆）
	Category    string      `json:"category"`               // 分类：knowledge, preference, environment
	Key         string      `json:"key"`                    // 键
	Value       string      `json:"value"`                  // 值（JSON格式）
	Metadata    string      `json:"metadata,omitempty"`     // 元数据
	AccessCount int         `json:"access_count"`           // 访问次数
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	ExpiresAt   *time.Time  `json:"expires_at,omitempty"`   // 过期时间
}

// MemoryManager 记忆管理器
type MemoryManager struct {
	db               *db.DB
	shortTermCache   map[string]*ConversationContext // 会话级缓存
	maxShortTermSize int64                           // 短暂记忆最大大小（字节）
	maxLongTermSize  int64                           // 永久记忆最大大小（字节）
	totalLimit       int64                           // 总容量限制
}

// ConversationContext 对话上下文（短暂记忆）
type ConversationContext struct {
	SessionID    string          `json:"session_id"`
	Messages     []llm.Message   `json:"messages"`      // 对话历史
	Summary      string          `json:"summary"`       // 对话摘要
	ToolUsage    map[string]int  `json:"tool_usage"`    // 工具使用统计
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	MaxMessages  int             `json:"max_messages"`  // 最大消息数
}

// NewMemoryManager 创建记忆管理器
func NewMemoryManager(database *db.DB) *MemoryManager {
	return &MemoryManager{
		db:               database,
		shortTermCache:   make(map[string]*ConversationContext),
		maxShortTermSize: 100 * 1024 * 1024,  // 100MB 短暂记忆
		maxLongTermSize:  400 * 1024 * 1024,  // 400MB 永久记忆
		totalLimit:       500 * 1024 * 1024,  // 500MB 总限制
	}
}

// === 短暂记忆（会话级别）===

// GetOrCreateContext 获取或创建会话上下文
func (m *MemoryManager) GetOrCreateContext(sessionID string) *ConversationContext {
	if ctx, ok := m.shortTermCache[sessionID]; ok {
		return ctx
	}

	ctx := &ConversationContext{
		SessionID:   sessionID,
		Messages:    make([]llm.Message, 0),
		ToolUsage:   make(map[string]int),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		MaxMessages: 50, // 保留最近50条消息
	}

	m.shortTermCache[sessionID] = ctx
	return ctx
}

// AddMessage 添加消息到对话上下文
func (m *MemoryManager) AddMessage(sessionID string, message llm.Message) {
	ctx := m.GetOrCreateContext(sessionID)
	ctx.Messages = append(ctx.Messages, message)
	ctx.UpdatedAt = time.Now()

	// 如果超过最大消息数，压缩旧消息
	if len(ctx.Messages) > ctx.MaxMessages {
		m.compressOldMessages(ctx)
	}
}

// GetRecentMessages 获取最近的消息（用于LLM上下文）
func (m *MemoryManager) GetRecentMessages(sessionID string, limit int) []llm.Message {
	ctx := m.GetOrCreateContext(sessionID)

	if limit <= 0 || limit >= len(ctx.Messages) {
		return ctx.Messages
	}

	return ctx.Messages[len(ctx.Messages)-limit:]
}

// RecordToolUsage 记录工具使用
func (m *MemoryManager) RecordToolUsage(sessionID, toolName string) {
	ctx := m.GetOrCreateContext(sessionID)
	ctx.ToolUsage[toolName]++
}

// compressOldMessages 压缩旧消息
func (m *MemoryManager) compressOldMessages(ctx *ConversationContext) {
	// 保留系统消息和最近的N条消息，其他的生成摘要
	const keepRecent = 20

	systemMsgs := make([]llm.Message, 0)
	oldMsgs := make([]llm.Message, 0)
	recentMsgs := make([]llm.Message, 0)

	for i, msg := range ctx.Messages {
		if msg.Role == "system" {
			systemMsgs = append(systemMsgs, msg)
		} else if i < len(ctx.Messages)-keepRecent {
			oldMsgs = append(oldMsgs, msg)
		} else {
			recentMsgs = append(recentMsgs, msg)
		}
	}

	// 生成旧消息的摘要
	if len(oldMsgs) > 0 {
		summary := m.generateSummary(oldMsgs)
		ctx.Summary = summary
	}

	// 重建消息列表：系统消息 + 摘要 + 最近消息
	newMessages := make([]llm.Message, 0)
	newMessages = append(newMessages, systemMsgs...)

	if ctx.Summary != "" {
		newMessages = append(newMessages, llm.Message{
			Role:    "system",
			Content: "之前对话摘要: " + ctx.Summary,
		})
	}

	newMessages = append(newMessages, recentMsgs...)
	ctx.Messages = newMessages
}

// generateSummary 生成对话摘要
func (m *MemoryManager) generateSummary(messages []llm.Message) string {
	// 简化版：提取关键工具调用和结果
	summary := "历史操作: "
	toolCalls := 0

	for _, msg := range messages {
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				summary += fmt.Sprintf("%s, ", tc.Function.Name)
				toolCalls++
			}
		}
	}

	if toolCalls > 0 {
		summary += fmt.Sprintf("(共%d次工具调用)", toolCalls)
	} else {
		summary += "无特殊操作"
	}

	return summary
}

// ClearSession 清除会话上下文
func (m *MemoryManager) ClearSession(sessionID string) {
	delete(m.shortTermCache, sessionID)
}

// === 永久记忆（持久化）===

// SaveKnowledge 保存知识到永久记忆
func (m *MemoryManager) SaveKnowledge(category, key, value string) error {
	return m.saveMemory(MemoryTypeLongTerm, nil, category, key, value, nil)
}

// GetKnowledge 获取知识
func (m *MemoryManager) GetKnowledge(category, key string) (string, error) {
	query := `
		SELECT value FROM memories
		WHERE type = ? AND category = ? AND key = ?
		ORDER BY updated_at DESC LIMIT 1
	`

	var value string
	err := m.db.QueryRow(query, MemoryTypeLongTerm, category, key).Scan(&value)
	if err != nil {
		return "", err
	}

	// 更新访问次数
	m.db.Exec(`UPDATE memories SET access_count = access_count + 1 WHERE category = ? AND key = ?`, category, key)

	return value, nil
}

// SavePreference 保存用户偏好
func (m *MemoryManager) SavePreference(key, value string) error {
	return m.saveMemory(MemoryTypeLongTerm, nil, "preference", key, value, nil)
}

// GetPreference 获取用户偏好
func (m *MemoryManager) GetPreference(key string) (string, error) {
	return m.GetKnowledge("preference", key)
}

// SaveEnvironmentInfo 保存环境信息
func (m *MemoryManager) SaveEnvironmentInfo(key, value string) error {
	return m.saveMemory(MemoryTypeLongTerm, nil, "environment", key, value, nil)
}

// GetEnvironmentInfo 获取环境信息
func (m *MemoryManager) GetEnvironmentInfo(key string) (string, error) {
	return m.GetKnowledge("environment", key)
}

// saveMemory 保存记忆
func (m *MemoryManager) saveMemory(
	memType MemoryType,
	sessionID *string,
	category, key, value string,
	expiresAt *time.Time,
) error {
	query := `
		INSERT INTO memories (type, session_id, category, key, value, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(type, category, key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := m.db.Exec(query, memType, sessionID, category, key, value, expiresAt)
	return err
}

// CleanExpiredMemories 清理过期记忆
func (m *MemoryManager) CleanExpiredMemories() error {
	query := `DELETE FROM memories WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP`
	_, err := m.db.Exec(query)
	return err
}

// GetMemoryStats 获取记忆统计
func (m *MemoryManager) GetMemoryStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 短暂记忆统计
	shortTermCount := len(m.shortTermCache)
	shortTermSize := int64(0)
	for _, ctx := range m.shortTermCache {
		data, _ := json.Marshal(ctx)
		shortTermSize += int64(len(data))
	}

	// 永久记忆统计
	var longTermCount int64
	var longTermSize int64

	m.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(LENGTH(value)), 0)
		FROM memories WHERE type = ?
	`, MemoryTypeLongTerm).Scan(&longTermCount, &longTermSize)

	stats["short_term_sessions"] = shortTermCount
	stats["short_term_size_bytes"] = shortTermSize
	stats["short_term_size_mb"] = float64(shortTermSize) / 1024 / 1024

	stats["long_term_items"] = longTermCount
	stats["long_term_size_bytes"] = longTermSize
	stats["long_term_size_mb"] = float64(longTermSize) / 1024 / 1024

	totalSize := shortTermSize + longTermSize
	stats["total_size_bytes"] = totalSize
	stats["total_size_mb"] = float64(totalSize) / 1024 / 1024
	stats["total_limit_mb"] = float64(m.totalLimit) / 1024 / 1024
	stats["usage_percent"] = float64(totalSize) / float64(m.totalLimit) * 100

	return stats, nil
}

// TrimMemoryIfNeeded 如果超过容量限制，则清理记忆
func (m *MemoryManager) TrimMemoryIfNeeded() error {
	stats, err := m.GetMemoryStats()
	if err != nil {
		return err
	}

	totalSize := stats["total_size_bytes"].(int64)

	// 如果超过90%容量，开始清理
	if totalSize > m.totalLimit*9/10 {
		// 清理最少访问的长期记忆
		query := `
			DELETE FROM memories
			WHERE type = ? AND id IN (
				SELECT id FROM memories
				WHERE type = ?
				ORDER BY access_count ASC, updated_at ASC
				LIMIT 100
			)
		`
		_, err := m.db.Exec(query, MemoryTypeLongTerm, MemoryTypeLongTerm)
		if err != nil {
			return err
		}

		// 清理旧的短暂记忆（超过1小时的会话）
		cutoff := time.Now().Add(-1 * time.Hour)
		for sessionID, ctx := range m.shortTermCache {
			if ctx.UpdatedAt.Before(cutoff) {
				delete(m.shortTermCache, sessionID)
			}
		}
	}

	return nil
}
