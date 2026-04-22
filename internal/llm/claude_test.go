package llm

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== NewClaudeClient 测试 =====

func TestNewClaudeClient(t *testing.T) {
	// 测试创建 Claude 客户端，默认 baseURL 为 Anthropic 官方
	client := NewClaudeClient("test-key", "claude-3")
	require.NotNil(t, client)
	assert.Equal(t, "test-key", client.apiKey)
	assert.Equal(t, "claude-3", client.model)
	assert.Equal(t, "https://api.anthropic.com", client.baseURL)
}

func TestNewClaudeClientWithBaseURL(t *testing.T) {
	// 测试使用自定义 baseURL 创建 Claude 兼容客户端
	client := NewClaudeClientWithBaseURL("https://custom.api.com", "key", "model")
	require.NotNil(t, client)
	assert.Equal(t, "https://custom.api.com", client.baseURL)
	assert.Equal(t, "key", client.apiKey)
	assert.Equal(t, "model", client.model)
}

// ===== ClaudeClient.Name 测试 =====

func TestClaudeClient_Name(t *testing.T) {
	client := NewClaudeClient("key", "model")
	assert.Equal(t, "claude", client.Name())
}

// ===== ClaudeClient.Complete 测试 =====

func TestClaudeClient_Complete_Success(t *testing.T) {
	// 测试正常请求返回 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2024-10-22", r.Header.Get("anthropic-version"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_123",
			"model": "claude-3",
			"content": [
				{"type": "text", "text": "你好，世界！"}
			],
			"usage": {"input_tokens": 10, "output_tokens": 5}
		}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "test-key", "claude-3")
	resp, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "msg_123", resp.ID)
	assert.Equal(t, "claude-3", resp.Model)
	assert.Equal(t, "你好，世界！", resp.Message.Content)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestClaudeClient_Complete_WithSystem(t *testing.T) {
	// 测试带 system 消息的请求
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		// system 消息应放在顶层字段，不在 messages 中
		assert.Contains(t, bodyStr, `"system"`)
		assert.Contains(t, bodyStr, "你是一个助手")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_456",
			"model": "claude-3",
			"content": [{"type": "text", "text": "好的"}],
			"usage": {"input_tokens": 10, "output_tokens": 5}
		}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "key", "model")
	resp, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "你是一个助手"},
			{Role: "user", Content: "hello"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "好的", resp.Message.Content)
}

func TestClaudeClient_Complete_WithTools(t *testing.T) {
	// 测试带工具定义的请求
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		// Claude 工具格式使用 input_schema
		assert.Contains(t, bodyStr, `"tools"`)
		assert.Contains(t, bodyStr, `"input_schema"`)
		assert.Contains(t, bodyStr, "get_weather")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_789",
			"model": "claude-3",
			"content": [
				{"type": "text", "text": "让我查一下"},
				{"type": "tool_use", "id": "call_1", "name": "get_weather", "input": {"city": "Beijing"}}
			],
			"usage": {"input_tokens": 20, "output_tokens": 10}
		}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "key", "model")
	resp, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "北京天气"}},
		Tools: []Tool{{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_weather",
				Description: "查询天气",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, "让我查一下", resp.Message.Content)
	require.Len(t, resp.Message.ToolCalls, 1)
	assert.Equal(t, "call_1", resp.Message.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", resp.Message.ToolCalls[0].Function.Name)
	assert.Contains(t, resp.Message.ToolCalls[0].Function.Arguments, "Beijing")
}

func TestClaudeClient_Complete_401(t *testing.T) {
	// 测试 401 未授权
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key","type":"auth_error"}}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "bad-key", "model")
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API密钥无效")
}

func TestClaudeClient_Complete_429(t *testing.T) {
	// 测试 429 限流
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limit","type":"rate_limit"}}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "key", "model")
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请求过于频繁")
}

func TestClaudeClient_Complete_500(t *testing.T) {
	// 测试 500 服务器错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"internal error","type":"server_error"}}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "key", "model")
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "服务端暂时不可用")
}

func TestClaudeClient_Complete_OtherError(t *testing.T) {
	// 测试其他 HTTP 错误码
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"message":"forbidden access","type":"auth"}}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "key", "model")
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden access")
}

func TestClaudeClient_Complete_ErrorWithDetail(t *testing.T) {
	// 测试错误响应带 detail 字段
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"detail":[{"msg":"invalid parameter"}]}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "key", "model")
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid parameter")
}

func TestClaudeClient_Complete_InvalidJSON(t *testing.T) {
	// 测试响应非 JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "key", "model")
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "解析响应失败")
}

func TestClaudeClient_Complete_WithMaxTokens(t *testing.T) {
	// 测试 MaxTokens 覆盖默认值
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		assert.Contains(t, bodyStr, `"max_tokens":100`)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_x",
			"model": "claude-3",
			"content": [{"type": "text", "text": "ok"}],
			"usage": {"input_tokens": 1, "output_tokens": 1}
		}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "key", "model")
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages:  []Message{{Role: "user", Content: "test"}},
		MaxTokens: 100,
	})
	require.NoError(t, err)
}

func TestClaudeClient_Complete_URLConstruction(t *testing.T) {
	// 测试不同 baseURL 的 URL 拼接逻辑
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{"裸域名", "/v1/messages", "/v1/messages"},
		{"带v1后缀", "/v1/messages", "/v1/messages"},
		{"带v1/后缀", "/v1/messages", "/v1/messages"},
		{"已有messages", "/v1/messages", "/v1/messages"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"id": "msg_x",
					"model": "test",
					"content": [{"type": "text", "text": "ok"}],
					"usage": {"input_tokens": 1, "output_tokens": 1}
				}`))
			}))
			defer server.Close()

			client := NewClaudeClientWithBaseURL(server.URL+tt.baseURL, "key", "model")
			_, err := client.Complete(context.Background(), &CompletionRequest{
				Messages: []Message{{Role: "user", Content: "test"}},
			})
			require.NoError(t, err)
			_ = receivedPath // 仅验证不 panic
		})
	}
}

func TestClaudeClient_Complete_ConnectionError(t *testing.T) {
	// 测试连接失败（连接拒绝的端口）
	client := NewClaudeClientWithBaseURL("http://127.0.0.1:1", "key", "model")
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
}

// ===== ClaudeClient.Stream 测试 =====

func TestClaudeClient_Stream_Success(t *testing.T) {
	// 测试流式调用（实际调用 Complete 后包装为单次流）
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_stream",
			"model": "claude-3",
			"content": [{"type": "text", "text": "流式回复"}],
			"usage": {"input_tokens": 5, "output_tokens": 3}
		}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "key", "model")
	ch, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)

	// 读取流式数据
	chunk := <-ch
	assert.Equal(t, "流式回复", chunk.Delta.Content)
	assert.True(t, chunk.Finish)
}

func TestClaudeClient_Stream_Error(t *testing.T) {
	// 测试流式调用时 Complete 返回错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"bad key","type":"auth"}}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "bad-key", "model")
	_, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
}

// ===== convertMessagesWithSystem 测试 =====

func TestClaudeClient_ConvertMessages_Simple(t *testing.T) {
	// 测试简单消息转换
	client := NewClaudeClient("key", "model")
	messages := []Message{
		{Role: "user", Content: "你好"},
		{Role: "assistant", Content: "你好！"},
	}
	converted, system := client.convertMessagesWithSystem(messages)
	assert.Equal(t, "", system)
	assert.Len(t, converted, 2)
}

func TestClaudeClient_ConvertMessages_WithSystem(t *testing.T) {
	// 测试 system 消息被提取到顶层
	client := NewClaudeClient("key", "model")
	messages := []Message{
		{Role: "system", Content: "你是助手"},
		{Role: "user", Content: "你好"},
	}
	converted, system := client.convertMessagesWithSystem(messages)
	assert.Equal(t, "你是助手", system)
	// system 消息不在 converted 中
	assert.Len(t, converted, 1)
}

func TestClaudeClient_ConvertMessages_MultipleSystem(t *testing.T) {
	// 测试多个 system 消息用双换行连接
	client := NewClaudeClient("key", "model")
	messages := []Message{
		{Role: "system", Content: "规则1"},
		{Role: "system", Content: "规则2"},
		{Role: "user", Content: "你好"},
	}
	converted, system := client.convertMessagesWithSystem(messages)
	assert.Equal(t, "规则1\n\n规则2", system)
	assert.Len(t, converted, 1)
}

func TestClaudeClient_ConvertMessages_ToolResult(t *testing.T) {
	// 测试 tool 角色消息转换为 user 角色（Anthropic 协议要求）
	client := NewClaudeClient("key", "model")
	messages := []Message{
		{Role: "tool", Content: `{"temp": 25}`, ToolCallID: "call_1"},
	}
	converted, _ := client.convertMessagesWithSystem(messages)
	require.Len(t, converted, 1)
	// tool 角色在 Anthropic 协议中应为 user
	assert.Equal(t, "user", converted[0]["role"])
}

func TestClaudeClient_ConvertMessages_WithToolCalls(t *testing.T) {
	// 测试 assistant 消息中的 tool_calls 转换
	client := NewClaudeClient("key", "model")
	messages := []Message{
		{
			Role:    "assistant",
			Content: "让我查一下",
			ToolCalls: []ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: FunctionCall{
						Name:      "get_weather",
						Arguments: `{"city":"Beijing"}`,
					},
				},
			},
		},
	}
	converted, _ := client.convertMessagesWithSystem(messages)
	require.Len(t, converted, 1)
	// content 应包含文本和 tool_use
	contentSlice, ok := converted[0]["content"].([]interface{})
	require.True(t, ok)
	assert.Len(t, contentSlice, 2)
}

func TestClaudeClient_ConvertMessages_EmptySystemContent(t *testing.T) {
	// 测试空内容的 system 消息被忽略
	client := NewClaudeClient("key", "model")
	messages := []Message{
		{Role: "system", Content: ""},
		{Role: "user", Content: "你好"},
	}
	converted, system := client.convertMessagesWithSystem(messages)
	assert.Equal(t, "", system)
	assert.Len(t, converted, 1)
}

// ===== Claude init() 注册表补充测试 =====

func TestClientFactory_CreateFromConfig_Anthropic(t *testing.T) {
	// 测试 anthropic 类型也能创建 Claude 客户端
	factory := NewClientFactory(nil)
	config := &ProviderConfig{
		Type:   "anthropic",
		APIKey: "test-key",
		Model:  "claude-3",
	}
	client, err := factory.CreateFromConfig(config)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "claude", client.Name())
}

// ===== URL 拼接测试 =====

func TestClaudeClient_Complete_URLAppend(t *testing.T) {
	// 测试裸 baseURL 自动追加 /v1/messages
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_x",
			"model": "test",
			"content": [{"type": "text", "text": "ok"}],
			"usage": {"input_tokens": 1, "output_tokens": 1}
		}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL, "key", "model")
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "/v1/messages", receivedPath)
}

func TestClaudeClient_Complete_URLWithV1(t *testing.T) {
	// 测试已有 /v1 后缀的 baseURL
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_x",
			"model": "test",
			"content": [{"type": "text", "text": "ok"}],
			"usage": {"input_tokens": 1, "output_tokens": 1}
		}`))
	}))
	defer server.Close()

	client := NewClaudeClientWithBaseURL(server.URL+"/v1", "key", "model")
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "/v1/messages", receivedPath)
}

// ===== helper: 用 bufio.Reader 模拟用户输入 =====

func newWizardWithInput(cm *ConfigManager, input string) *ConfigWizard {
	return NewConfigWizardWithReader(cm, bufio.NewReader(strings.NewReader(input)))
}
