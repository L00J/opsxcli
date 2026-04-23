package llm

import "context"

// Message 聊天消息
type Message struct {
	Role       string     `json:"role"`    // system, user, assistant, tool
	Content    string     `json:"content"` // 消息内容
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // function
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON字符串
}

// Tool 工具定义
type Tool struct {
	Type     string       `json:"type"` // function
	Function ToolFunction `json:"function"`
}

// ToolFunction 工具函数定义
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// CompletionRequest 请求参数
type CompletionRequest struct {
	Messages    []Message   `json:"messages"`
	Tools       []Tool      `json:"tools,omitempty"`
	ToolChoice  interface{} `json:"tool_choice,omitempty"` // "auto", "none", "required" 或 {"type": "function", "function": {"name": "xxx"}}
	Temperature float64     `json:"temperature,omitempty"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
	Stream      bool        `json:"stream,omitempty"`
}

// CompletionResponse 响应结果
type CompletionResponse struct {
	ID      string  `json:"id"`
	Model   string  `json:"model"`
	Message Message `json:"message"`
	Usage   Usage   `json:"usage"`
}

// Usage 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	Delta  Message `json:"delta"`
	Finish bool    `json:"finish"`
}

// Client LLM客户端接口
type Client interface {
	// Complete 完成对话
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Stream 流式对话
	Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)

	// Name 返回客户端名称
	Name() string
}
