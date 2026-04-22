package llm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== NewGeminiClient 测试 =====

func TestNewGeminiClient(t *testing.T) {
	// 正常创建
	client := NewGeminiClient("api-key", "gemini-pro")
	require.NotNil(t, client)
	assert.Equal(t, "api-key", client.apiKey)
	assert.Equal(t, "gemini-pro", client.model)
}

func TestNewGeminiClient_DefaultModel(t *testing.T) {
	// 空模型使用默认值
	client := NewGeminiClient("key", "")
	require.NotNil(t, client)
	assert.Equal(t, "gemini-2.5-pro", client.model)
}

// ===== GeminiClient.Name 测试 =====

func TestGeminiClient_Name(t *testing.T) {
	client := NewGeminiClient("key", "model")
	assert.Equal(t, "gemini", client.Name())
}

// ===== GeminiClient.Complete 测试 =====

func TestGeminiClient_Complete_Success(t *testing.T) {
	// 测试正常请求
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		// Gemini 使用 URL 参数传递 key
		assert.Contains(t, r.URL.String(), "key=test-key")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [{"text": "你好！"}]
				}
			}],
			"usageMetadata": {
				"promptTokenCount": 10,
				"candidatesTokenCount": 5,
				"totalTokenCount": 15
			}
		}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "test-key",
		model:      "gemini-pro",
		httpClient: server.Client(),
	}
	// 需要覆盖 httpClient 的请求目标
	// 直接用 httptest server 的 URL 替换
	// 由于 Complete 内部硬编码 Google URL，我们需要通过 httpClient 来重定向

	// 更好的方式：通过修改 client 的 httpClient 使用自定义 Transport
	client.httpClient = &http.Client{
		Transport: &testTransport{serverURL: server.URL},
	}

	resp, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "gemini-response", resp.ID)
	assert.Equal(t, "gemini-pro", resp.Model)
	assert.Equal(t, "你好！", resp.Message.Content)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestGeminiClient_Complete_WithTools(t *testing.T) {
	// 测试带工具定义的请求
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		assert.Contains(t, bodyStr, `"tools"`)
		assert.Contains(t, bodyStr, "functionDeclarations")
		assert.Contains(t, bodyStr, "search")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [
						{"text": "搜索结果"},
						{"functionCall": {"name": "search", "args": {"query": "test"}}}
					]
				}
			}],
			"usageMetadata": {
				"promptTokenCount": 10,
				"candidatesTokenCount": 10,
				"totalTokenCount": 20
			}
		}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "key",
		model:      "gemini-pro",
		httpClient: &http.Client{Transport: &testTransport{serverURL: server.URL}},
	}

	resp, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "搜索"}},
		Tools: []Tool{{
			Type: "function",
			Function: ToolFunction{
				Name:        "search",
				Description: "搜索",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, "搜索结果", resp.Message.Content)
	require.Len(t, resp.Message.ToolCalls, 1)
	assert.Equal(t, "search", resp.Message.ToolCalls[0].Function.Name)
}

func TestGeminiClient_Complete_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid key","type":"auth"}}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "bad-key",
		model:      "gemini-pro",
		httpClient: &http.Client{Transport: &testTransport{serverURL: server.URL}},
	}
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API密钥无效")
}

func TestGeminiClient_Complete_429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limit","type":"rate"}}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "key",
		model:      "gemini-pro",
		httpClient: &http.Client{Transport: &testTransport{serverURL: server.URL}},
	}
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请求过于频繁")
}

func TestGeminiClient_Complete_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"internal error","type":"server"}}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "key",
		model:      "gemini-pro",
		httpClient: &http.Client{Transport: &testTransport{serverURL: server.URL}},
	}
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "服务端暂时不可用")
}

func TestGeminiClient_Complete_OtherError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"message":"forbidden","type":"auth"}}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "key",
		model:      "gemini-pro",
		httpClient: &http.Client{Transport: &testTransport{serverURL: server.URL}},
	}
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestGeminiClient_Complete_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "key",
		model:      "gemini-pro",
		httpClient: &http.Client{Transport: &testTransport{serverURL: server.URL}},
	}
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "解析响应失败")
}

func TestGeminiClient_Complete_NoCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"candidates":[],"usageMetadata":{"promptTokenCount":0,"candidatesTokenCount":0,"totalTokenCount":0}}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "key",
		model:      "gemini-pro",
		httpClient: &http.Client{Transport: &testTransport{serverURL: server.URL}},
	}
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "响应中没有候选结果")
}

func TestGeminiClient_Complete_ConnectionError(t *testing.T) {
	// 测试连接失败
	client := NewGeminiClient("key", "model")
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
}

func TestGeminiClient_Complete_ErrorWithDetail(t *testing.T) {
	// 测试错误响应带 detail 字段
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"detail":[{"msg":"bad request"}]}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "key",
		model:      "gemini-pro",
		httpClient: &http.Client{Transport: &testTransport{serverURL: server.URL}},
	}
	_, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad request")
}

// ===== GeminiClient.Stream 测试 =====

func TestGeminiClient_Stream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"candidates": [{"content": {"parts": [{"text": "流式回复"}]}}],
			"usageMetadata": {"promptTokenCount": 5, "candidatesTokenCount": 3, "totalTokenCount": 8}
		}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "key",
		model:      "gemini-pro",
		httpClient: &http.Client{Transport: &testTransport{serverURL: server.URL}},
	}
	ch, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	chunk := <-ch
	assert.Equal(t, "流式回复", chunk.Delta.Content)
	assert.True(t, chunk.Finish)
}

func TestGeminiClient_Stream_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"bad key","type":"auth"}}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "bad-key",
		model:      "gemini-pro",
		httpClient: &http.Client{Transport: &testTransport{serverURL: server.URL}},
	}
	_, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
}

// ===== GeminiClient.convertMessages 测试 =====

func TestGeminiClient_ConvertMessages_User(t *testing.T) {
	client := NewGeminiClient("key", "model")
	messages := []Message{
		{Role: "user", Content: "你好"},
	}
	converted := client.convertMessages(messages)
	assert.Len(t, converted, 1)
	assert.Equal(t, "user", converted[0]["role"])
}

func TestGeminiClient_ConvertMessages_System(t *testing.T) {
	// system 角色转换为 user
	client := NewGeminiClient("key", "model")
	messages := []Message{
		{Role: "system", Content: "你是助手"},
	}
	converted := client.convertMessages(messages)
	assert.Len(t, converted, 1)
	assert.Equal(t, "user", converted[0]["role"])
}

func TestGeminiClient_ConvertMessages_Assistant(t *testing.T) {
	// assistant 角色转换为 model
	client := NewGeminiClient("key", "model")
	messages := []Message{
		{Role: "assistant", Content: "你好！"},
	}
	converted := client.convertMessages(messages)
	assert.Len(t, converted, 1)
	assert.Equal(t, "model", converted[0]["role"])
}

func TestGeminiClient_ConvertMessages_ToolResult(t *testing.T) {
	// tool 角色消息转换为 functionResponse
	client := NewGeminiClient("key", "model")
	messages := []Message{
		{Role: "tool", Content: "25度", Name: "get_weather"},
	}
	converted := client.convertMessages(messages)
	assert.Len(t, converted, 1)
}

func TestGeminiClient_ConvertMessages_EmptyContent(t *testing.T) {
	// 空内容消息不产生 parts
	client := NewGeminiClient("key", "model")
	messages := []Message{
		{Role: "user", Content: ""},
	}
	converted := client.convertMessages(messages)
	// 空 Content 不产生 parts，但 len(parts)==0 时不会 append
	assert.Len(t, converted, 0)
}

func TestGeminiClient_Complete_WithTemperatureAndMaxTokens(t *testing.T) {
	// 测试 Temperature 和 MaxTokens 设置
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		assert.Contains(t, bodyStr, `"generationConfig"`)
		assert.Contains(t, bodyStr, `"temperature"`)
		assert.Contains(t, bodyStr, `"maxOutputTokens"`)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"candidates": [{"content": {"parts": [{"text": "ok"}]}}],
			"usageMetadata": {"promptTokenCount": 1, "candidatesTokenCount": 1, "totalTokenCount": 2}
		}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "key",
		model:      "gemini-pro",
		httpClient: &http.Client{Transport: &testTransport{serverURL: server.URL}},
	}
	resp, err := client.Complete(context.Background(), &CompletionRequest{
		Messages:    []Message{{Role: "user", Content: "test"}},
		Temperature: 0.5,
		MaxTokens:   100,
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Message.Content)
}

// testTransport 用于将请求重定向到 httptest server
type testTransport struct {
	serverURL string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 替换 URL 的 Scheme 和 Host，保留 Path 和 Query
	newReq := req.Clone(req.Context())
	newReq.URL.Scheme = "http"
	newReq.URL.Host = ""
	// 从 serverURL 提取 host
	host := t.serverURL
	newReq.URL.Host = host[len("http://"):]
	newReq.URL.Scheme = "http"
	return http.DefaultTransport.RoundTrip(newReq)
}
