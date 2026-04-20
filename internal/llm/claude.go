package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ClaudeClient Claude/Anthropic 兼容 API 客户端
type ClaudeClient struct {
	apiKey     string
	model      string
	baseURL    string // API 基础地址，默认 https://api.anthropic.com
	httpClient *http.Client
}

// NewClaudeClient 创建 Claude 客户端
func NewClaudeClient(apiKey, model string) *ClaudeClient {
	return &ClaudeClient{
		apiKey:     apiKey,
		model:      model,
		baseURL:    "https://api.anthropic.com",
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// NewClaudeClientWithBaseURL 创建带自定义 base URL 的 Claude 兼容客户端
func NewClaudeClientWithBaseURL(baseURL, apiKey, model string) *ClaudeClient {
	return &ClaudeClient{
		apiKey:     apiKey,
		model:      model,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// Name 返回客户端名称
func (c *ClaudeClient) Name() string {
	return "claude"
}

// Complete 完成对话
func (c *ClaudeClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	// 转换消息格式，提取 system 消息
	messages, systemPrompt := c.convertMessagesWithSystem(req.Messages)

	// 构建请求体
	reqBody := map[string]interface{}{
		"model":      c.model,
		"messages":   messages,
		"max_tokens": 4096,
	}

	// Anthropic 协议: system 消息放在顶层 "system" 字段
	if systemPrompt != "" {
		reqBody["system"] = systemPrompt
	}

	if req.MaxTokens > 0 {
		reqBody["max_tokens"] = req.MaxTokens
	}

	// 支持显式设置 Temperature=0（用指针判断是否设置）
	if req.Temperature >= 0 {
		reqBody["temperature"] = req.Temperature
	}

	// Claude 的工具格式
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = map[string]interface{}{
				"name":         tool.Function.Name,
				"description":  tool.Function.Description,
				"input_schema": tool.Function.Parameters,
			}
		}
		reqBody["tools"] = tools
	}

	// 序列化请求
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求（使用 baseURL 支持第三方 Anthropic 兼容接口）
	// URL 拼接：如果 baseURL 已包含 /v1/messages 则直接用，否则追加
	url := c.baseURL
	if !strings.HasSuffix(url, "/messages") {
		if !strings.HasSuffix(url, "/v1") && !strings.HasSuffix(url, "/v1/") {
			url = url + "/v1/messages"
		} else {
			url = strings.TrimRight(url, "/") + "/messages"
		}
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2024-10-22")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// 尝试解析友好错误信息
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
			Detail []struct {
				Msg string `json:"msg"`
			} `json:"detail"`
		}
		friendlyMsg := string(body)
		if json.Unmarshal(body, &errResp) == nil {
			if errResp.Error.Message != "" {
				friendlyMsg = errResp.Error.Message
			} else if len(errResp.Detail) > 0 {
				friendlyMsg = errResp.Detail[0].Msg
			}
		}
		// 常见状态码友好提示
		switch resp.StatusCode {
		case 401:
			return nil, fmt.Errorf("API密钥无效或已过期，请运行 opsxcli setup 重新配置")
		case 429:
			return nil, fmt.Errorf("请求过于频繁，请稍后再试")
		case 500, 529:
			return nil, fmt.Errorf("服务端暂时不可用 (%d)，请稍后再试", resp.StatusCode)
		default:
			return nil, fmt.Errorf("API错误 (%d): %s", resp.StatusCode, friendlyMsg)
		}
	}

	// 解析响应
	var apiResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text,omitempty"`
			ID    string `json:"id,omitempty"`
			Name  string `json:"name,omitempty"`
			Input interface{} `json:"input,omitempty"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 转换响应消息
	message := Message{Role: "assistant"}
	var toolCalls []ToolCall

	for _, content := range apiResp.Content {
		if content.Type == "text" {
			message.Content += content.Text
		} else if content.Type == "tool_use" {
			args, _ := json.Marshal(content.Input)
			toolCalls = append(toolCalls, ToolCall{
				ID:   content.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      content.Name,
					Arguments: string(args),
				},
			})
		}
	}

	if len(toolCalls) > 0 {
		message.ToolCalls = toolCalls
	}

	return &CompletionResponse{
		ID:      apiResp.ID,
		Model:   apiResp.Model,
		Message: message,
		Usage: Usage{
			PromptTokens:     apiResp.Usage.InputTokens,
			CompletionTokens: apiResp.Usage.OutputTokens,
			TotalTokens:      apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
	}, nil
}

// Stream 流式对话 (简化实现，暂不支持)
func (c *ClaudeClient) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	// Claude流式实现较复杂，这里先用非流式实现
	resp, err := c.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	chunks := make(chan StreamChunk, 1)
	go func() {
		chunks <- StreamChunk{
			Delta:  resp.Message,
			Finish: true,
		}
		close(chunks)
	}()

	return chunks, nil
}

// convertMessagesWithSystem 转换消息格式，提取 system 消息到顶层字段
func (c *ClaudeClient) convertMessagesWithSystem(messages []Message) ([]map[string]interface{}, string) {
	converted := make([]map[string]interface{}, 0, len(messages))
	var systemParts []string

	for _, msg := range messages {
		// Anthropic 协议: system 消息放在顶层 "system" 字段，不在 messages 中
		if msg.Role == "system" {
			if msg.Content != "" {
				systemParts = append(systemParts, msg.Content)
			}
			continue
		}

		content := make([]interface{}, 0)

		// 文本内容
		if msg.Content != "" && msg.Role != "tool" {
			content = append(content, map[string]interface{}{
				"type": "text",
				"text": msg.Content,
			})
		}

		// 工具调用（assistant 消息中的 tool_use）
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				var input interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &input)

				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": input,
				})
			}
		}

		// 工具结果 — Anthropic 协议要求 role 为 "user"
		if msg.Role == "tool" {
			content = append(content, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": msg.ToolCallID,
				"content":     msg.Content,
			})
		}

		if len(content) > 0 {
			// Anthropic 协议: tool_result 必须放在 role="user" 的消息中
			role := msg.Role
			if msg.Role == "tool" {
				role = "user"
			}
			converted = append(converted, map[string]interface{}{
				"role":    role,
				"content": content,
			})
		}
	}

	systemPrompt := strings.Join(systemParts, "\n\n")
	return converted, systemPrompt
}
