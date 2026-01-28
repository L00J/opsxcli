package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ClaudeClient Claude API 客户端
type ClaudeClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClaudeClient 创建 Claude 客户端
func NewClaudeClient(apiKey, model string) *ClaudeClient {
	return &ClaudeClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
	}
}

// Name 返回客户端名称
func (c *ClaudeClient) Name() string {
	return "claude"
}

// Complete 完成对话
func (c *ClaudeClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	// 转换消息格式
	messages := c.convertMessages(req.Messages)

	// 构建请求体
	reqBody := map[string]interface{}{
		"model":      c.model,
		"messages":   messages,
		"max_tokens": 4096,
	}

	if req.MaxTokens > 0 {
		reqBody["max_tokens"] = req.MaxTokens
	}

	if req.Temperature > 0 {
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

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonData))
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
		return nil, fmt.Errorf("API错误 (%d): %s", resp.StatusCode, string(body))
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

// convertMessages 转换消息格式
func (c *ClaudeClient) convertMessages(messages []Message) []map[string]interface{} {
	converted := make([]map[string]interface{}, 0, len(messages))

	for _, msg := range messages {
		// Claude 不需要 system role，会单独处理
		if msg.Role == "system" {
			continue
		}

		content := make([]interface{}, 0)

		// 文本内容
		if msg.Content != "" {
			content = append(content, map[string]interface{}{
				"type": "text",
				"text": msg.Content,
			})
		}

		// 工具调用
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

		// 工具结果
		if msg.Role == "tool" {
			content = append(content, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": msg.ToolCallID,
				"content":     msg.Content,
			})
		}

		if len(content) > 0 {
			converted = append(converted, map[string]interface{}{
				"role":    msg.Role,
				"content": content,
			})
		}
	}

	return converted
}
