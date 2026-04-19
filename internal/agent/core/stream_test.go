package core

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"opsxcli/internal/agent/safety"
	"opsxcli/internal/agent/tools"
	"opsxcli/internal/llm"
)

// mockStreamLLM 支持自定义 Stream 响应的 LLM mock
type mockStreamLLM struct {
	responses    []llm.CompletionResponse
	callIndex    int
	err          error
	streamChunks []llm.StreamChunk
	streamErr    error
}

func (m *mockStreamLLM) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.callIndex >= len(m.responses) {
		return &llm.CompletionResponse{
			ID:      "mock-fallback",
			Model:   "mock",
			Message: llm.Message{Role: "assistant", Content: "fallback"},
			Usage:   llm.Usage{TotalTokens: 1},
		}, nil
	}
	resp := &m.responses[m.callIndex]
	m.callIndex++
	return resp, nil
}

func (m *mockStreamLLM) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	ch := make(chan llm.StreamChunk, len(m.streamChunks))
	for _, chunk := range m.streamChunks {
		ch <- chunk
	}
	close(ch)
	return ch, nil
}

func (m *mockStreamLLM) Name() string {
	return "mock-stream-llm"
}

func TestRunStreamDirectAnswer(t *testing.T) {
	mockLLM := &mockStreamLLM{
		responses: []llm.CompletionResponse{
			{
				ID:    "resp-1",
				Model: "mock",
				Message: llm.Message{
					Role:    "assistant",
					Content: "The answer is 42.",
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
		},
		streamChunks: []llm.StreamChunk{
			{Delta: llm.Message{Role: "assistant", Content: "The "}},
			{Delta: llm.Message{Role: "assistant", Content: "answer "}},
			{Delta: llm.Message{Role: "assistant", Content: "is 42."}, Finish: true},
		},
	}

	registry := tools.NewRegistry()
	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:     5,
		Temperature:       0.3,
		ToolTimeout:       10 * time.Second,
		MaxTokens:         1024,
		SafetyMode:        SafetyModeBalanced,
		SessionDir:        sessionDir,
		AutoApprove:       true,
		OutputMaxLength:   10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)

	ctx := context.Background()
	var out strings.Builder
	result, err := agent.RunStream(ctx, "What is the meaning of life?", &out)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if !result.Success {
		t.Error("expected result.Success to be true")
	}
	if result.Output != "The answer is 42." {
		t.Errorf("expected output 'The answer is 42.', got %q", result.Output)
	}
	if agent.totalTokens != 10 {
		t.Errorf("expected totalTokens 10, got %d", agent.totalTokens)
	}
	if mockLLM.callIndex != 1 {
		t.Errorf("expected 1 Complete call, got %d", mockLLM.callIndex)
	}
	// 输出中应包含流式内容
	outputStr := out.String()
	if !strings.Contains(outputStr, "The answer is 42.") {
		t.Errorf("expected output to contain streamed content, got %q", outputStr)
	}
}

func TestRunStreamWithToolCall(t *testing.T) {
	mockToolResult := &tools.Result{
		Success: true,
		Output:  "mock tool output",
	}

	mockT := &mockTool{
		name:        "mock_tool",
		description: "A mock tool for testing",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"arg1": map[string]interface{}{"type": "string"},
			},
		},
		risk:   tools.RiskSafe,
		result: mockToolResult,
	}

	registry := tools.NewRegistry()
	registry.Register(mockT)

	mockLLM := &mockStreamLLM{
		responses: []llm.CompletionResponse{
			{
				ID:    "resp-1",
				Model: "mock",
				Message: llm.Message{
					Role: "assistant",
					ToolCalls: []llm.ToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "mock_tool",
								Arguments: `{"arg1":"value1"}`,
							},
						},
					},
				},
				Usage: llm.Usage{TotalTokens: 15},
			},
			{
				ID:    "resp-2",
				Model: "mock",
				Message: llm.Message{
					Role:    "assistant",
					Content: "Based on the tool output, the answer is clear.",
				},
				Usage: llm.Usage{TotalTokens: 20},
			},
		},
		streamChunks: []llm.StreamChunk{
			{Delta: llm.Message{Role: "assistant", Content: "Based "}},
			{Delta: llm.Message{Role: "assistant", Content: "on the tool output, the answer is clear."}, Finish: true},
		},
	}

	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:     5,
		Temperature:       0.3,
		ToolTimeout:       10 * time.Second,
		MaxTokens:         1024,
		SafetyMode:        SafetyModeBalanced,
		SessionDir:        sessionDir,
		AutoApprove:       true,
		OutputMaxLength:   10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)

	ctx := context.Background()
	var out strings.Builder
	result, err := agent.RunStream(ctx, "Please run the mock tool.", &out)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if !result.Success {
		t.Error("expected result.Success to be true")
	}
	if result.Output != "Based on the tool output, the answer is clear." {
		t.Errorf("expected output 'Based on the tool output, the answer is clear.', got %q", result.Output)
	}
	if agent.totalTokens != 35 {
		t.Errorf("expected totalTokens 35, got %d", agent.totalTokens)
	}
	if mockLLM.callIndex != 2 {
		t.Errorf("expected 2 Complete calls, got %d", mockLLM.callIndex)
	}
	// 输出中应包含进度提示和流式内容
	outputStr := out.String()
	if !strings.Contains(outputStr, "正在执行工具") {
		t.Errorf("expected output to contain progress hint, got %q", outputStr)
	}
}

func TestRunStreamFallbackOnStreamError(t *testing.T) {
	mockLLM := &mockStreamLLM{
		responses: []llm.CompletionResponse{
			{
				ID:    "resp-1",
				Model: "mock",
				Message: llm.Message{
					Role:    "assistant",
					Content: "Fallback content from Complete.",
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
		},
		streamErr: errors.New("stream connection failed"),
	}

	registry := tools.NewRegistry()
	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:     5,
		Temperature:       0.3,
		ToolTimeout:       10 * time.Second,
		MaxTokens:         1024,
		SafetyMode:        SafetyModeBalanced,
		SessionDir:        sessionDir,
		AutoApprove:       true,
		OutputMaxLength:   10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)

	ctx := context.Background()
	var out strings.Builder
	result, err := agent.RunStream(ctx, "This will fallback.", &out)

	if err != nil {
		t.Fatalf("expected no error after fallback, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if !result.Success {
		t.Error("expected result.Success to be true")
	}
	if result.Output != "Fallback content from Complete." {
		t.Errorf("expected output 'Fallback content from Complete.', got %q", result.Output)
	}
	outputStr := out.String()
	if !strings.Contains(outputStr, "Fallback content from Complete.") {
		t.Errorf("expected output to contain fallback content, got %q", outputStr)
	}
}

func TestRunStreamWithErrorChunk(t *testing.T) {
	mockLLM := &mockStreamLLM{
		responses: []llm.CompletionResponse{
			{
				ID:    "resp-1",
				Model: "mock",
				Message: llm.Message{
					Role:    "assistant",
					Content: "Some content.",
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
		},
		streamChunks: []llm.StreamChunk{
			{Delta: llm.Message{Role: "error", Content: "something went wrong"}, Finish: true},
		},
	}

	registry := tools.NewRegistry()
	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:     5,
		Temperature:       0.3,
		ToolTimeout:       10 * time.Second,
		MaxTokens:         1024,
		SafetyMode:        SafetyModeBalanced,
		SessionDir:        sessionDir,
		AutoApprove:       true,
		OutputMaxLength:   10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)

	ctx := context.Background()
	var out strings.Builder
	_, err := agent.RunStream(ctx, "Trigger stream error.", &out)

	if err == nil {
		t.Fatal("expected error from stream chunk")
	}
	if !strings.Contains(err.Error(), "流式输出错误") {
		t.Errorf("expected error to contain '流式输出错误', got %v", err)
	}
}

func TestRunStreamMaxIterations(t *testing.T) {
	mockT := &mockTool{
		name:        "mock_tool",
		description: "A mock tool for testing",
		parameters: map[string]interface{}{
			"type": "object",
		},
		risk: tools.RiskSafe,
		result: &tools.Result{
			Success: true,
			Output:  "output",
		},
	}

	registry := tools.NewRegistry()
	registry.Register(mockT)

	// Always return a tool call so the agent never gets a direct answer.
	responses := make([]llm.CompletionResponse, 0, 5)
	for i := 0; i < 5; i++ {
		responses = append(responses, llm.CompletionResponse{
			ID:    "resp-" + strconv.Itoa(i),
			Model: "mock",
			Message: llm.Message{
				Role: "assistant",
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call-" + strconv.Itoa(i),
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "mock_tool",
							Arguments: `{"unique":` + strconv.Itoa(i) + `}`,
						},
					},
				},
			},
			Usage: llm.Usage{TotalTokens: 5},
		})
	}

	mockLLM := &mockStreamLLM{
		responses: responses,
	}

	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:     3,
		Temperature:       0.3,
		ToolTimeout:       10 * time.Second,
		MaxTokens:         1024,
		SafetyMode:        SafetyModeBalanced,
		SessionDir:        sessionDir,
		AutoApprove:       true,
		OutputMaxLength:   10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)

	ctx := context.Background()
	var out strings.Builder
	result, err := agent.RunStream(ctx, "This will hit max iterations.", &out)

	if err == nil {
		t.Fatal("expected error due to max iterations")
	}
	if result != nil {
		t.Error("expected result to be nil when max iterations reached")
	}
	if mockLLM.callIndex != 3 {
		t.Errorf("expected 3 LLM calls, got %d", mockLLM.callIndex)
	}
	// 输出中应包含多次进度提示
	outputStr := out.String()
	count := strings.Count(outputStr, "正在执行工具")
	if count != 3 {
		t.Errorf("expected 3 progress hints, got %d", count)
	}
}
