package session

import (
	"opsxcli/internal/llm"
)

// Manager 会话管理器接口
type Manager interface {
	// Create 创建新会话
	Create(title, provider, model string) (*Session, error)

	// SaveMessage 保存消息到会话
	SaveMessage(sessionID string, msg llm.Message) error

	// SaveToolResult 保存工具执行结果到会话
	SaveToolResult(sessionID string, toolCallID, name string, success bool, output, err string) error

	// Load 加载会话信息和所有消息
	Load(sessionID string) (*Session, []llm.Message, error)

	// List 列出会话（按时间倒序）
	List(limit int) ([]*Session, error)

	// Delete 删除会话
	Delete(sessionID string) error

	// ExportMarkdown 导出会话为 Markdown
	ExportMarkdown(sessionID string) (string, error)

	// ExportJSON 导出会话为 JSON
	ExportJSON(sessionID string) (string, error)

	// ExportToFile 导出会话到文件，返回实际写入的文件路径
	ExportToFile(sessionID, format, filePath string) (string, error)

	// UpdateTitle 更新会话标题
	UpdateTitle(sessionID, title string) error
}

// DefaultManager 默认会话管理器实现
type DefaultManager struct {
	store *JSONLStore
}

// NewManager 创建会话管理器
func NewManager(store *JSONLStore) Manager {
	return &DefaultManager{store: store}
}

// Create 创建新会话
func (m *DefaultManager) Create(title, provider, model string) (*Session, error) {
	if title == "" {
		title = "新会话"
	}
	return m.store.Create(title, provider, model)
}

// SaveMessage 保存消息到会话
func (m *DefaultManager) SaveMessage(sessionID string, msg llm.Message) error {
	if err := m.store.WriteMessage(sessionID, msg); err != nil {
		return err
	}
	return m.store.UpdateMetaTimestamp(sessionID)
}

// SaveToolResult 保存工具执行结果到会话
func (m *DefaultManager) SaveToolResult(sessionID string, toolCallID, name string, success bool, output, err string) error {
	if err := m.store.WriteToolResult(sessionID, toolCallID, name, success, output, err); err != nil {
		return err
	}
	return m.store.UpdateMetaTimestamp(sessionID)
}

// Load 加载会话信息和所有消息
func (m *DefaultManager) Load(sessionID string) (*Session, []llm.Message, error) {
	session, err := m.store.LoadSession(sessionID)
	if err != nil {
		return nil, nil, err
	}

	messages, err := m.store.LoadMessages(sessionID)
	if err != nil {
		return nil, nil, err
	}

	return session, messages, nil
}

// List 列出会话
func (m *DefaultManager) List(limit int) ([]*Session, error) {
	return m.store.List(limit)
}

// Delete 删除会话
func (m *DefaultManager) Delete(sessionID string) error {
	return m.store.Delete(sessionID)
}

// ExportMarkdown 导出会话为 Markdown
func (m *DefaultManager) ExportMarkdown(sessionID string) (string, error) {
	return m.store.ExportMarkdown(sessionID)
}

// ExportJSON 导出会话为 JSON
func (m *DefaultManager) ExportJSON(sessionID string) (string, error) {
	return m.store.ExportJSON(sessionID)
}

// ExportToFile 导出会话到文件，返回实际写入的文件路径
func (m *DefaultManager) ExportToFile(sessionID, format, filePath string) (string, error) {
	return m.store.ExportToFile(sessionID, format, filePath)
}

// UpdateTitle 更新会话标题
func (m *DefaultManager) UpdateTitle(sessionID, title string) error {
	return m.store.UpdateTitle(sessionID, title)
}
