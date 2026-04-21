package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRetryableError_Nil(t *testing.T) {
	assert.False(t, isRetryableError(nil))
}

func TestIsRetryableError_Timeout(t *testing.T) {
	assert.True(t, isRetryableError(errors.New("connection timeout")))
	assert.True(t, isRetryableError(errors.New("request timeout after 30s")))
}

func TestIsRetryableError_ConnectionReset(t *testing.T) {
	assert.True(t, isRetryableError(errors.New("connection reset by peer")))
}

func TestIsRetryableError_NoSuchHost(t *testing.T) {
	assert.True(t, isRetryableError(errors.New("dial tcp: no such host")))
}

func TestIsRetryableError_EOF(t *testing.T) {
	assert.True(t, isRetryableError(errors.New("unexpected EOF")))
}

func TestIsRetryableError_ConnectionRefused(t *testing.T) {
	assert.True(t, isRetryableError(errors.New("connection refused")))
}

func TestIsRetryableError_NonRetryable(t *testing.T) {
	assert.False(t, isRetryableError(errors.New("invalid api key")))
	assert.False(t, isRetryableError(errors.New("bad request")))
	assert.False(t, isRetryableError(errors.New("permission denied")))
}

func TestNewOpenAIClient(t *testing.T) {
	client := NewOpenAIClient("https://api.openai.com/v1/", "test-key", "gpt-4")
	require.NotNil(t, client)
	assert.Equal(t, "https://api.openai.com/v1", client.baseURL) // trailing slash trimmed
	assert.Equal(t, "test-key", client.apiKey)
	assert.Equal(t, "gpt-4", client.model)
	assert.NotNil(t, client.httpClient)
}

func TestNewOpenAIClient_EmptyBaseURL(t *testing.T) {
	client := NewOpenAIClient("", "key", "model")
	require.NotNil(t, client)
	assert.Equal(t, "", client.baseURL)
}

func TestOpenAIClient_Name(t *testing.T) {
	client := NewOpenAIClient("http://localhost", "", "test")
	assert.Equal(t, "openai-compatible", client.Name())
}

func TestOpenAIClient_Complete_ContextCancelled(t *testing.T) {
	client := NewOpenAIClient("http://localhost:1", "key", "model")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.Complete(ctx, &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
}

func TestOpenAIClient_doComplete_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key","type":"invalid_request_error"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "bad-key", "test")
	_, err := client.doComplete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API密钥无效")
}

func TestOpenAIClient_doComplete_429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limit exceeded","type":"rate_limit_error"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.doComplete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请求过于频繁")
}

func TestOpenAIClient_doComplete_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"internal server error","type":"server_error"}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.doComplete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "服务端暂时不可用")
}

func TestOpenAIClient_doComplete_200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "chatcmpl-123",
			"model": "gpt-4",
			"choices": [{"index": 0, "message": {"role": "assistant", "content": "hello"}}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "test-key", "gpt-4")
	resp, err := client.doComplete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "chatcmpl-123", resp.ID)
	assert.Equal(t, "gpt-4", resp.Model)
	assert.Equal(t, "hello", resp.Message.Content)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestOpenAIClient_doComplete_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"x","model":"m","choices":[],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.doComplete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "响应中没有选择")
}

func TestOpenAIClient_doComplete_WithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request body includes tools
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		assert.Contains(t, bodyStr, "tools")
		assert.Contains(t, bodyStr, "test_func")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"x","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	resp, err := client.doComplete(context.Background(), &CompletionRequest{
		Messages:    []Message{{Role: "user", Content: "test"}},
		Temperature: 0.7,
		MaxTokens:   100,
		Tools: []Tool{{
			Type: "function",
			Function: ToolFunction{
				Name:        "test_func",
				Description: "A test function",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Message.Content)
}

func TestOpenAIClient_doComplete_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When no API key, Authorization header should not be set
		assert.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"x","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "", "test")
	_, err := client.doComplete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	require.NoError(t, err)
}

func TestOpenAIClient_doComplete_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.doComplete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "解析响应失败")
}

func TestOpenAIClient_doComplete_ErrorWithDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"detail":[{"msg":"invalid model name"}]}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.doComplete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid model name")
}

func TestOpenAIClient_doComplete_ErrorWithRawBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`raw error text without json structure`))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "key", "test")
	_, err := client.doComplete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "raw error text without json structure")
}
