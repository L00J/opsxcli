package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Session 会话
type Session struct {
	ID        string
	UserID    int64
	Title     string
	Provider  string
	Model     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Message 消息
type Message struct {
	ID        int64
	SessionID string
	Role      string
	Content   string
	ToolCalls string
	Metadata  string
	CreatedAt time.Time
}

// SessionRepository 会话仓储
type SessionRepository struct {
	db *DB
}

// NewSessionRepository 创建会话仓储
func NewSessionRepository(db *DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create 创建会话
func (r *SessionRepository) Create(session *Session) error {
	// 生成会话 ID
	if session.ID == "" {
		session.ID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}

	query := `
		INSERT INTO sessions (id, user_id, title, provider, model, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	_, err := r.db.Exec(query,
		session.ID,
		session.UserID,
		session.Title,
		session.Provider,
		session.Model,
		now,
		now,
	)

	return err
}

// GetByID 根据 ID 获取会话
func (r *SessionRepository) GetByID(id string) (*Session, error) {
	query := `
		SELECT id, user_id, title, provider, model, created_at, updated_at
		FROM sessions
		WHERE id = ?
	`

	session := &Session{}
	err := r.db.QueryRow(query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.Title,
		&session.Provider,
		&session.Model,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("会话不存在")
	}

	return session, err
}

// GetByUser 获取用户的会话列表
func (r *SessionRepository) GetByUser(userID int64, limit int) ([]*Session, error) {
	query := `
		SELECT id, user_id, title, provider, model, created_at, updated_at
		FROM sessions
		WHERE user_id = ?
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]*Session, 0)
	for rows.Next() {
		session := &Session{}
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.Title,
			&session.Provider,
			&session.Model,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// Update 更新会话
func (r *SessionRepository) Update(session *Session) error {
	query := `
		UPDATE sessions
		SET title = ?, updated_at = ?
		WHERE id = ?
	`

	session.UpdatedAt = time.Now()

	_, err := r.db.Exec(query, session.Title, session.UpdatedAt, session.ID)
	return err
}

// Delete 删除会话
func (r *SessionRepository) Delete(id string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// MessageRepository 消息仓储
type MessageRepository struct {
	db *DB
}

// NewMessageRepository 创建消息仓储
func NewMessageRepository(db *DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create 创建消息
func (r *MessageRepository) Create(message *Message) error {
	query := `
		INSERT INTO messages (session_id, role, content, tool_calls, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	message.CreatedAt = time.Now()

	result, err := r.db.Exec(query,
		message.SessionID,
		message.Role,
		message.Content,
		message.ToolCalls,
		message.Metadata,
		message.CreatedAt,
	)

	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	message.ID = id

	return nil
}

// GetBySession 获取会话的所有消息
func (r *MessageRepository) GetBySession(sessionID string) ([]*Message, error) {
	query := `
		SELECT id, session_id, role, content, tool_calls, metadata, created_at
		FROM messages
		WHERE session_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]*Message, 0)
	for rows.Next() {
		message := &Message{}
		var toolCalls, metadata sql.NullString

		err := rows.Scan(
			&message.ID,
			&message.SessionID,
			&message.Role,
			&message.Content,
			&toolCalls,
			&metadata,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if toolCalls.Valid {
			message.ToolCalls = toolCalls.String
		}
		if metadata.Valid {
			message.Metadata = metadata.String
		}

		messages = append(messages, message)
	}

	return messages, rows.Err()
}
