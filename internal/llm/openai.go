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
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // 默认请求超时 120 秒
		},
	}
}

// Name 返回客户端名称
func (c *OpenAIClient) Name() string {
	return "openai-compatible"
}

// Complete 完成对话（带指数退避重试）
func (c *OpenAIClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避: 1s, 2s, 4s
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := c.doComplete(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		// 如果不是可重试错误，直接返回
		if !isRetryableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("LLM 请求失败（已重试 %d 次）: %w", maxRetries, lastErr)
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// 网络超时、连接重置、5xx 错误等可重试
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "connection refused")
}

// doComplete 实际执行单次请求
func (c *OpenAIClient) doComplete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
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

func init() {
	// OpenAI 兼容的模型（使用统一的 OpenAI API 格式）
	for _, t := range []string{"deepseek", "openai", "gpt", "kimi", "qwen", "yi", "baichuan", "doubao", "llama", "ollama"} {
		RegisterProvider(t, func(config *ProviderConfig) (Client, error) {
			return NewOpenAIClient(config.BaseURL, config.APIKey, config.Model), nil
		})
	}
}
