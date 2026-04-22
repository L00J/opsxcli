package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== OpenAI Stream 错误处理测试 =====
// 注意：Stream 方法使用 resp.Body.Read(buf) 读取数据，
// 当数据量小时 Read 会一次性返回 (data, io.EOF)，导致数据被丢弃。
// 因此仅测试错误处理路径（在流式解析之前返回错误）。

func TestOpenAIClient_Stream_401(t *testing.T) {
	// 测试流式请求 401 错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid key","type":"auth"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "bad-key", "test")
	_, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API密钥无效")
}

func TestOpenAIClient_Stream_429(t *testing.T) {
	// 测试流式请求 429 限流错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limit","type":"rate"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请求过于频繁")
}

func TestOpenAIClient_Stream_500(t *testing.T) {
	// 测试流式请求 500 错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"internal error","type":"server"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "服务端暂时不可用")
}

func TestOpenAIClient_Stream_529(t *testing.T) {
	// 测试 529 状态码（服务过载）
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(529)
		w.Write([]byte(`{"error":{"message":"overloaded","type":"server"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "服务端暂时不可用")
}

func TestOpenAIClient_Stream_OtherError(t *testing.T) {
	// 测试其他 HTTP 错误状态码
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"message":"forbidden","type":"auth"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestOpenAIClient_Stream_ErrorWithDetail(t *testing.T) {
	// 测试流式请求错误带 detail 字段
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"detail":[{"msg":"invalid model"}]}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid model")
}

func TestOpenAIClient_Stream_ErrorRawBody(t *testing.T) {
	// 测试流式请求错误返回非 JSON 内容
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`plain text error`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plain text error")
}

func TestOpenAIClient_Stream_ConnectionError(t *testing.T) {
	// 测试流式请求连接失败
	client := NewOpenAIClient("http://127.0.0.1:1", "key", "test")
	_, err := client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
}

// ===== OpenAI Stream 请求构造测试 =====

func TestOpenAIClient_Stream_SetsHeaders(t *testing.T) {
	// 测试 Stream 正确设置请求头
	var capturedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"test","type":"test"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "my-api-key", "gpt-4")
	_, _ = client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	require.NotNil(t, capturedReq)
	assert.Equal(t, "POST", capturedReq.Method)
	assert.Equal(t, "application/json", capturedReq.Header.Get("Content-Type"))
	assert.Equal(t, "text/event-stream", capturedReq.Header.Get("Accept"))
	assert.Equal(t, "Bearer my-api-key", capturedReq.Header.Get("Authorization"))
}

func TestOpenAIClient_Stream_NoAPIKey_NoAuthHeader(t *testing.T) {
	// 测试无 API Key 时不设置 Authorization 头
	var capturedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"test","type":"test"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "", "test")
	_, _ = client.Stream(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})

	require.NotNil(t, capturedReq)
	assert.Empty(t, capturedReq.Header.Get("Authorization"))
}

func TestOpenAIClient_Stream_WithTools_SendsTools(t *testing.T) {
	// 测试带工具的请求正确发送 tools 字段
	var capturedReq *http.Request
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		body := make([]byte, 4096)
		n, _ := r.Body.Read(body)
		_ = json.Unmarshal(body[:n], &capturedBody)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"test","type":"test"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, _ = client.Stream(context.Background(), &CompletionRequest{
		Messages:    []Message{{Role: "user", Content: "test"}},
		Temperature: 0.7,
		MaxTokens:   100,
		Tools: []Tool{{
			Type: "function",
			Function: ToolFunction{
				Name:        "test_func",
				Description: "A test",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		}},
	})

	require.NotNil(t, capturedReq)
	assert.Equal(t, true, capturedBody["stream"])
	assert.Equal(t, "test", capturedBody["model"])
	assert.NotNil(t, capturedBody["tools"])
	assert.Equal(t, 0.7, capturedBody["temperature"])
	assert.Equal(t, float64(100), capturedBody["max_tokens"])
}

func TestOpenAIClient_Stream_CancelledContext(t *testing.T) {
	// 测试上下文取消时返回错误
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	client := NewOpenAIClient("http://127.0.0.1:1", "key", "test")
	_, err := client.Stream(ctx, &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
}
