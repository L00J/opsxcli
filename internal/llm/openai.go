package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIClient OpenAI 兼容的客户端（DeepSeek、Ollama等）
type OpenAIClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenAIClient 创建 OpenAI 兼容客户端
func NewOpenAIClient(baseURL, apiKey, model string) *OpenAIClient {
	return &OpenAIClient{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
	}
}

// Name 返回客户端名称
func (c *OpenAIClient) Name() string {
	return "openai-compatible"
}

// Complete 完成对话
func (c *OpenAIClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	// 构建请求体
	reqBody := map[string]interface{}{
		"model":    c.model,
		"messages": req.Messages,
	}

	if len(req.Tools) > 0 {
		reqBody["tools"] = req.Tools
	}

	if req.Temperature > 0 {
		reqBody["temperature"] = req.Temperature
	}

	if req.MaxTokens > 0 {
		reqBody["max_tokens"] = req.MaxTokens
	}

	// 序列化请求
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

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
		Choices []struct {
			Index   int     `json:"index"`
			Message Message `json:"message"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("响应中没有选择")
	}

	return &CompletionResponse{
		ID:      apiResp.ID,
		Model:   apiResp.Model,
		Message: apiResp.Choices[0].Message,
		Usage:   apiResp.Usage,
	}, nil
}

// Stream 流式对话
func (c *OpenAIClient) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	// 构建请求体
	reqBody := map[string]interface{}{
		"model":    c.model,
		"messages": req.Messages,
		"stream":   true,
	}

	if len(req.Tools) > 0 {
		reqBody["tools"] = req.Tools
	}

	if req.Temperature > 0 {
		reqBody["temperature"] = req.Temperature
	}

	if req.MaxTokens > 0 {
		reqBody["max_tokens"] = req.MaxTokens
	}

	// 序列化请求
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API错误 (%d): %s", resp.StatusCode, string(body))
	}

	// 创建流通道
	chunks := make(chan StreamChunk, 10)

	go func() {
		defer resp.Body.Close()
		defer close(chunks)

		buf := make([]byte, 4096)

		for {
			// 读取一行
			n, err := resp.Body.Read(buf)
			if err != nil {
				if err != io.EOF {
					chunks <- StreamChunk{
						Delta:  Message{Role: "error", Content: err.Error()},
						Finish: true,
					}
				}
				return
			}

			line := string(buf[:n])
			lines := strings.Split(line, "\n")

			for _, l := range lines {
				l = strings.TrimSpace(l)
				if l == "" || l == "data: [DONE]" {
					continue
				}

				if !strings.HasPrefix(l, "data: ") {
					continue
				}

				data := strings.TrimPrefix(l, "data: ")

				var chunk struct {
					Choices []struct {
						Delta struct {
							Role      string     `json:"role,omitempty"`
							Content   string     `json:"content,omitempty"`
							ToolCalls []ToolCall `json:"tool_calls,omitempty"`
						} `json:"delta"`
						FinishReason string `json:"finish_reason,omitempty"`
					} `json:"choices"`
				}

				if err := json.Unmarshal([]byte(data), &chunk); err != nil {
					continue
				}

				if len(chunk.Choices) > 0 {
					delta := chunk.Choices[0].Delta
					finish := chunk.Choices[0].FinishReason != ""

					chunks <- StreamChunk{
						Delta: Message{
							Role:      delta.Role,
							Content:   delta.Content,
							ToolCalls: delta.ToolCalls,
						},
						Finish: finish,
					}

					if finish {
						return
					}
				}
			}
		}
	}()

	return chunks, nil
}
