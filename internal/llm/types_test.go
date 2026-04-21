package llm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage_JSONMarshal(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "hello",
	}
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"role":"user"`)
	assert.Contains(t, string(data), `"content":"hello"`)
}

func TestMessage_JSONUnmarshal(t *testing.T) {
	raw := `{"role":"assistant","content":"world"}`
	var msg Message
	err := json.Unmarshal([]byte(raw), &msg)
	require.NoError(t, err)
	assert.Equal(t, "assistant", msg.Role)
	assert.Equal(t, "world", msg.Content)
}

func TestMessage_OmitEmpty(t *testing.T) {
	msg := Message{Role: "user", Content: "test"}
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "name")
	assert.NotContains(t, string(data), "tool_call_id")
	assert.NotContains(t, string(data), "tool_calls")
}

func TestMessage_WithToolCalls(t *testing.T) {
	msg := Message{
		Role:    "assistant",
		Content: "",
		ToolCalls: []ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: FunctionCall{
					Name:      "get_weather",
					Arguments: `{"city":"Beijing"}`,
				},
			},
		},
	}
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	assert.Contains(t, string(data), "call_123")
	assert.Contains(t, string(data), "get_weather")
}

func TestMessage_ToolResponse(t *testing.T) {
	msg := Message{
		Role:       "tool",
		Content:    `{"temp": 25}`,
		ToolCallID: "call_123",
	}
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	assert.Contains(t, string(data), "call_123")
}

func TestToolCall_JSONRoundTrip(t *testing.T) {
	tc := ToolCall{
		ID:   "call_abc",
		Type: "function",
		Function: FunctionCall{
			Name:      "search",
			Arguments: `{"query":"test"}`,
		},
	}
	data, err := json.Marshal(tc)
	require.NoError(t, err)

	var decoded ToolCall
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, tc.ID, decoded.ID)
	assert.Equal(t, tc.Type, decoded.Type)
	assert.Equal(t, tc.Function.Name, decoded.Function.Name)
	assert.Equal(t, tc.Function.Arguments, decoded.Function.Arguments)
}

func TestTool_JSONRoundTrip(t *testing.T) {
	tool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "calculator",
			Description: "Performs calculations",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "Math expression",
					},
				},
			},
		},
	}
	data, err := json.Marshal(tool)
	require.NoError(t, err)

	var decoded Tool
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "function", decoded.Type)
	assert.Equal(t, "calculator", decoded.Function.Name)
	assert.Equal(t, "Performs calculations", decoded.Function.Description)
}

func TestCompletionRequest_JSONMarshal(t *testing.T) {
	req := CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"temperature":0.7`)
	assert.Contains(t, string(data), `"max_tokens":100`)
}

func TestCompletionRequest_OmitEmpty(t *testing.T) {
	req := CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	// tools, tool_choice, temperature, max_tokens, stream should be omitted
	assert.NotContains(t, string(data), "tools")
	assert.NotContains(t, string(data), "tool_choice")
	assert.NotContains(t, string(data), "stream")
}

func TestCompletionResponse_JSONRoundTrip(t *testing.T) {
	resp := CompletionResponse{
		ID:    "chatcmpl-456",
		Model: "gpt-4",
		Message: Message{
			Role:    "assistant",
			Content: "Hi there!",
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded CompletionResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, resp.ID, decoded.ID)
	assert.Equal(t, resp.Model, decoded.Model)
	assert.Equal(t, resp.Message.Content, decoded.Message.Content)
	assert.Equal(t, resp.Usage.TotalTokens, decoded.Usage.TotalTokens)
}

func TestUsage_JSON(t *testing.T) {
	usage := Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	data, err := json.Marshal(usage)
	require.NoError(t, err)

	var decoded Usage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, 100, decoded.PromptTokens)
	assert.Equal(t, 50, decoded.CompletionTokens)
	assert.Equal(t, 150, decoded.TotalTokens)
}

func TestStreamChunk_JSON(t *testing.T) {
	chunk := StreamChunk{
		Delta: Message{
			Role:    "assistant",
			Content: "Hello",
		},
		Finish: false,
	}
	data, err := json.Marshal(chunk)
	require.NoError(t, err)

	var decoded StreamChunk
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "Hello", decoded.Delta.Content)
	assert.False(t, decoded.Finish)
}

func TestCompletionRequest_WithToolChoice(t *testing.T) {
	req := CompletionRequest{
		Messages:   []Message{{Role: "user", Content: "test"}},
		ToolChoice: "auto",
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"tool_choice":"auto"`)
}

func TestCompletionRequest_WithComplexToolChoice(t *testing.T) {
	req := CompletionRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
		ToolChoice: map[string]interface{}{
			"type": "function",
			"function": map[string]string{
				"name": "get_weather",
			},
		},
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), "get_weather")
}
