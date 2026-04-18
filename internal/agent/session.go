package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"opsxcli/internal/db"
	"opsxcli/internal/llm"
)

// SessionManager 会话管理器
type SessionManager struct {
	db           *db.DB
	currentUser  *db.User
	sessionRepo  *db.SessionRepository
	messageRepo  *db.MessageRepository
}

// NewSessionManager 创建会话管理器
func NewSessionManager(database *db.DB, user *db.User) *SessionManager {
	return &SessionManager{
		db:          database,
		currentUser: user,
		sessionRepo: db.NewSessionRepository(database),
		messageRepo: db.NewMessageRepository(database),
	}
}

// CreateSession 创建新会话
func (sm *SessionManager) CreateSession(provider, model, title string) (*db.Session, error) {
	session := &db.Session{
		UserID:   sm.currentUser.ID,
		Title:    title,
		Provider: provider,
		Model:    model,
	}

	err := sm.sessionRepo.Create(session)
	if err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	return session, nil
}

// SaveMessage 保存消息
func (sm *SessionManager) SaveMessage(sessionID string, msg llm.Message) error {
	message := &db.Message{
		SessionID: sessionID,
		Role:      msg.Role,
		Content:   msg.Content,
	}

	// 保存 tool_calls (如果有)
	if len(msg.ToolCalls) > 0 {
		// 序列化为 JSON
		toolCallsJSON, err := json.Marshal(msg.ToolCalls)
		if err == nil {
			message.ToolCalls = string(toolCallsJSON)
		}
	}

	return sm.messageRepo.Create(message)
}

// LoadSession 加载会话
func (sm *SessionManager) LoadSession(sessionID string) (*db.Session, []llm.Message, error) {
	// 获取会话信息
	session, err := sm.sessionRepo.GetByID(sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取会话失败: %w", err)
	}

	// 检查权限
	if session.UserID != sm.currentUser.ID {
		return nil, nil, fmt.Errorf("无权访问此会话")
	}

	// 获取消息历史
	messages, err := sm.messageRepo.GetBySession(sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取消息历史失败: %w", err)
	}

	// 转换为 LLM 消息格式
	llmMessages := make([]llm.Message, len(messages))
	for i, msg := range messages {
		llmMessages[i] = llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// 反序列化 tool_calls (如果有)
		if msg.ToolCalls != "" {
			var toolCalls []llm.ToolCall
			if err := json.Unmarshal([]byte(msg.ToolCalls), &toolCalls); err == nil {
				llmMessages[i].ToolCalls = toolCalls
			}
		}
	}

	return session, llmMessages, nil
}

// ListSessions 列出用户的所有会话
func (sm *SessionManager) ListSessions(limit int) ([]*db.Session, error) {
	return sm.sessionRepo.GetByUser(sm.currentUser.ID, limit)
}

// DeleteSession 删除会话
func (sm *SessionManager) DeleteSession(sessionID string) error {
	// 检查权限
	session, err := sm.sessionRepo.GetByID(sessionID)
	if err != nil {
		return err
	}

	if session.UserID != sm.currentUser.ID {
		return fmt.Errorf("无权删除此会话")
	}

	return sm.sessionRepo.Delete(sessionID)
}

// UpdateSessionTitle 更新会话标题
func (sm *SessionManager) UpdateSessionTitle(sessionID, title string) error {
	session, err := sm.sessionRepo.GetByID(sessionID)
	if err != nil {
		return err
	}

	if session.UserID != sm.currentUser.ID {
		return fmt.Errorf("无权修改此会话")
	}

	session.Title = title
	return sm.sessionRepo.Update(session)
}

// ExportSession 导出会话为 Markdown
func (sm *SessionManager) ExportSession(sessionID string) (string, error) {
	session, messages, err := sm.LoadSession(sessionID)
	if err != nil {
		return "", err
	}

	var md strings.Builder

	// 标题
	md.WriteString(fmt.Sprintf("# %s\n\n", session.Title))
	md.WriteString(fmt.Sprintf("- **Provider**: %s\n", session.Provider))
	md.WriteString(fmt.Sprintf("- **Model**: %s\n", session.Model))
	md.WriteString(fmt.Sprintf("- **Created**: %s\n\n", session.CreatedAt.Format("2006-01-02 15:04:05")))
	md.WriteString("---\n\n")

	// 消息
	for _, msg := range messages {
		var roleIcon string
		switch msg.Role {
		case "user":
			roleIcon = "👤"
		case "assistant":
			roleIcon = "🤖"
		case "system":
			roleIcon = "⚙️"
		case "tool":
			roleIcon = "🔧"
		}

		md.WriteString(fmt.Sprintf("## %s %s\n\n", roleIcon, strings.Title(msg.Role)))
		md.WriteString(msg.Content)
		md.WriteString("\n\n")
	}

	md.WriteString("---\n\n")
	md.WriteString(fmt.Sprintf("*Exported at %s*\n", time.Now().Format("2006-01-02 15:04:05")))

	return md.String(), nil
}
