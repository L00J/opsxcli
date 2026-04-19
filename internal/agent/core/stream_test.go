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
	streamChunks   []llm.StreamChunk
	streamChunks2  []llm.StreamChunk // 第二轮响应（工具执行后）
	streamErr      error
	streamCallCount int
}

func (m *mockStreamLLM) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{
		ID:      "mock",
		Model:   "mock",
		Message: llm.Message{Role: "assistant", Content: "mock"},
		Usage:   llm.Usage{TotalTokens: 1},
	}, nil
}

func (m *mockStreamLLM) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	m.streamCallCount++
	chunks := m.streamChunks
	if m.streamCallCount > 1 && m.streamChunks2 != nil {
		chunks = m.streamChunks2
	}
	ch := make(chan llm.StreamChunk, len(chunks))
	for _, chunk := range chunks {
		ch <- chunk
	}
	close(ch)
	return ch, nil
}

func (m *mockStreamLLM) Name() string {
	return "mock-stream-llm"
}

func newTestAgent(t *testing.T, llmClient llm.Client, registry *tools.Registry) *Agent {
	if registry == nil {
		registry = tools.NewRegistry()
	}
	config := &Config{
		MaxIterations:     5,
		Temperature:       0.3,
		ToolTimeout:       10 * time.Second,
		MaxTokens:         1024,
		SafetyMode:        SafetyModeBalanced,
		SessionDir:        t.TempDir(),
		AutoApprove:       true,
		OutputMaxLength:   10000,
		MaxContextTokens:  6000,
	}
	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)
	return NewAgent(llmClient, registry, config, safetyCtl)
}

func TestRunStreamDirectAnswer(t *testing.T) {
	mockLLM := &mockStreamLLM{
		streamChunks: []llm.StreamChunk{
			{Delta: llm.Message{Role: "assistant", Content: "The "}},
			{Delta: llm.Message{Role: "assistant", Content: "answer "}},
			{Delta: llm.Message{Role: "assistant", Content: "is 42."}, Finish: true},
		},
	}

	agent := newTestAgent(t, mockLLM, nil)
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
		streamChunks: []llm.StreamChunk{
			{Delta: llm.Message{Role: "assistant", Content: "Let me check."}},
			{Delta: llm.Message{Role: "assistant", ToolCalls: []llm.ToolCall{
				{ID: "call-1", Type: "function", Function: llm.FunctionCall{Name: "mock_tool", Arguments: `{"arg1":"value1"}`}},
			}}},
		},
		streamChunks2: []llm.StreamChunk{
			{Delta: llm.Message{Role: "assistant", Content: "Based "}},
			{Delta: llm.Message{Role: "assistant", Content: "on the tool output, the answer is clear."}, Finish: true},
		},
	}

	agent := newTestAgent(t, mockLLM, registry)
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
	// 输出中应包含进度提示和流式内容
	outputStr := out.String()
	if !strings.Contains(outputStr, "正在执行工具") {
		t.Errorf("expected output to contain progress hint, got %q", outputStr)
	}
}

func TestRunStreamStreamError(t *testing.T) {
	mockLLM := &mockStreamLLM{
		streamErr: errors.New("stream connection failed"),
	}

	agent := newTestAgent(t, mockLLM, nil)
	ctx := context.Background()
	var out strings.Builder
	_, err := agent.RunStream(ctx, "This will fail.", &out)

	if err == nil {
		t.Fatal("expected error from stream failure")
	}
	if !strings.Contains(err.Error(), "Stream 失败") {
		t.Errorf("expected error to contain 'Stream 失败', got %v", err)
	}
}

func TestRunStreamWithErrorChunk(t *testing.T) {
	mockLLM := &mockStreamLLM{
		streamChunks: []llm.StreamChunk{
			{Delta: llm.Message{Role: "error", Content: "something went wrong"}, Finish: true},
		},
	}

	agent := newTestAgent(t, mockLLM, nil)
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

	// 构造 5 轮 tool_call 的 stream chunks（每轮 2 个 chunk：思考 + tool_call）
	chunks := make([]llm.StreamChunk, 0, 10)
	for i := 0; i < 5; i++ {
		chunks = append(chunks, llm.StreamChunk{
			Delta: llm.Message{Role: "assistant", Content: "Thinking..."},
		})
		chunks = append(chunks, llm.StreamChunk{
			Delta: llm.Message{Role: "assistant", ToolCalls: []llm.ToolCall{
				{ID: "call-" + strconv.Itoa(i), Type: "function", Function: llm.FunctionCall{
					Name:      "mock_tool",
					Arguments: `{"unique":` + strconv.Itoa(i) + `}`,
				}},
			}},
		})
	}
	// 最后一个标记结束
	chunks[len(chunks)-1].Finish = true

	mockLLM := &mockStreamLLM{streamChunks: chunks}

	agent := newTestAgent(t, mockLLM, registry)
	// 限制最大迭代次数为 3
	agent.config.MaxIterations = 3

	ctx := context.Background()
	var out strings.Builder
	result, err := agent.RunStream(ctx, "This will hit max iterations.", &out)

	if err == nil {
		t.Fatal("expected error due to max iterations")
	}
	if result != nil {
		t.Error("expected result to be nil when max iterations reached")
	}
	// 输出中应包含多次进度提示
	outputStr := out.String()
	count := strings.Count(outputStr, "正在执行工具")
	if count != 3 {
		t.Errorf("expected 3 progress hints, got %d", count)
	}
}
