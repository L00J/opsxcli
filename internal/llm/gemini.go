package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GeminiClient Google Gemini API 客户端
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewGeminiClient 创建 Gemini 客户端
func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = "gemini-2.5-pro"
	}
	return &GeminiClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Name 返回客户端名称
func (c *GeminiClient) Name() string {
	return "gemini"
}

// Complete 完成对话
func (c *GeminiClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	// 转换消息格式
	contents := c.convertMessages(req.Messages)

	// 构建请求体
	reqBody := map[string]interface{}{
		"contents": contents,
	}

	// 生成配置
	generationConfig := map[string]interface{}{}
	if req.Temperature > 0 {
		generationConfig["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		generationConfig["maxOutputTokens"] = req.MaxTokens
	}
	if len(generationConfig) > 0 {
		reqBody["generationConfig"] = generationConfig
	}

	// Gemini 的工具格式
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = map[string]interface{}{
				"functionDeclarations": []map[string]interface{}{
					{
						"name":        tool.Function.Name,
						"description": tool.Function.Description,
						"parameters":  tool.Function.Parameters,
					},
				},
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
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text,omitempty"`
					FunctionCall *struct {
						Name string                 `json:"name"`
						Args map[string]interface{} `json:"args"`
					} `json:"functionCall,omitempty"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(apiResp.Candidates) == 0 {
		return nil, fmt.Errorf("响应中没有候选结果")
	}

	// 转换响应消息
	message := Message{Role: "assistant"}
	var toolCalls []ToolCall

	for i, part := range apiResp.Candidates[0].Content.Parts {
		if part.Text != "" {
			message.Content += part.Text
		}
		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			toolCalls = append(toolCalls, ToolCall{
				ID:   fmt.Sprintf("call_%d", i),
				Type: "function",
				Function: FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(args),
				},
			})
		}
	}

	if len(toolCalls) > 0 {
		message.ToolCalls = toolCalls
	}

	return &CompletionResponse{
		ID:      "gemini-response",
		Model:   c.model,
		Message: message,
		Usage: Usage{
			PromptTokens:     apiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: apiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      apiResp.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

// Stream 流式对话
func (c *GeminiClient) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	// 简化实现，先用非流式
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
func (c *GeminiClient) convertMessages(messages []Message) []map[string]interface{} {
	converted := make([]map[string]interface{}, 0)

	for _, msg := range messages {
		// Gemini 不直接支持 system role，需要转换为 user
		role := msg.Role
		if role == "system" {
			role = "user"
		} else if role == "assistant" {
			role = "model"
		}

		parts := make([]interface{}, 0)

		// 文本内容
		if msg.Content != "" {
			parts = append(parts, map[string]interface{}{
				"text": msg.Content,
			})
		}

		// 工具调用结果
		if msg.Role == "tool" {
			parts = append(parts, map[string]interface{}{
				"functionResponse": map[string]interface{}{
					"name": msg.Name,
					"response": map[string]interface{}{
						"content": msg.Content,
					},
				},
			})
		}

		if len(parts) > 0 {
			converted = append(converted, map[string]interface{}{
				"role":  role,
				"parts": parts,
			})
		}
	}

	return converted
}

func init() {
	// Gemini 专用客户端
	RegisterProvider("gemini", func(config *ProviderConfig) (Client, error) {
		return NewGeminiClient(config.APIKey, config.Model), nil
	})
}
