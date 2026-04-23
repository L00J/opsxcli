package session

import "time"

// Session 会话信息
type Session struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
}

// JSONLRecord JSONL 文件中的一行记录
type JSONLRecord struct {
	Type      string    `json:"type"` // meta / message / tool_result
	Timestamp time.Time `json:"timestamp"`

	// 对于 type=meta:
	ID        string    `json:"id,omitempty"`
	Title     string    `json:"title,omitempty"`
	Provider  string    `json:"provider,omitempty"`
	Model     string    `json:"model,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	// 对于 type=message:
	Role       string     `json:"role,omitempty"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`

	// 对于 type=tool_result:
	ToolCallID_ string `json:"tool_call_id_,omitempty"`
	Name_       string `json:"name_,omitempty"`
	Success     bool   `json:"success,omitempty"`
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ToolCall 工具调用（简化版，用于 JSONL 序列化）
type ToolCall struct {
	ID       string `json:"id"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
