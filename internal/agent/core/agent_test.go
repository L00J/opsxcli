package core

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"opsxcli/internal/agent/evolver"
	"opsxcli/internal/agent/safety"
	"opsxcli/internal/agent/tools"
	"opsxcli/internal/llm"
)

// mockLLMClient implements llm.Client for testing.
// It cycles through a slice of prepared responses on each Complete call.
// Thread-safe: callIndex is protected by a mutex because the background
// Evolver goroutine may call Complete concurrently with test assertions.
type mockLLMClient struct {
	name      string
	responses []llm.CompletionResponse
	callIndex int
	mu        sync.Mutex
	err       error
}

func (m *mockLLMClient) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return nil, m.err
	}
	if m.callIndex >= len(m.responses) {
		m.callIndex++
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

// getCallCount returns the current call index (thread-safe).
func (m *mockLLMClient) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callIndex
}

func (m *mockLLMClient) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	ch <- llm.StreamChunk{Delta: llm.Message{Role: "assistant", Content: "streamed"}, Finish: true}
	close(ch)
	return ch, nil
}

func (m *mockLLMClient) Name() string {
	return m.name
}

// mockTool implements tools.Tool for testing.
type mockTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	risk        tools.RiskLevel
	result      *tools.Result
	execErr     error
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Parameters() map[string]interface{} {
	return m.parameters
}

func (m *mockTool) RiskLevel() tools.RiskLevel {
	return m.risk
}

func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.Result, error) {
	return m.result, m.execErr
}

func TestNewAgent(t *testing.T) {
	mockLLM := &mockLLMClient{name: "test-llm"}
	registry := tools.NewRegistry()
	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		Temperature:     0.5,
		ToolTimeout:     30 * time.Second,
		MaxTokens:       2048,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 5000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)

	if agent == nil {
		t.Fatal("expected agent to be non-nil")
	}
	if agent.llmClient != mockLLM {
		t.Error("expected llmClient to be set")
	}
	if agent.registry != registry {
		t.Error("expected registry to be set")
	}
	if agent.config != config {
		t.Error("expected config to be set")
	}
	if agent.safetyCtl != safetyCtl {
		t.Error("expected safetyCtl to be set")
	}
	if len(agent.messages) != 0 {
		t.Errorf("expected messages to be empty, got %d", len(agent.messages))
	}
}

func TestNewAgentWithNilConfig(t *testing.T) {
	mockLLM := &mockLLMClient{name: "test-llm"}
	registry := tools.NewRegistry()
	safetyCtl := safety.NewController(safety.SafetyModeBalanced)

	agent := NewAgent(mockLLM, registry, nil, safetyCtl)

	if agent == nil {
		t.Fatal("expected agent to be non-nil")
	}
	if agent.config == nil {
		t.Fatal("expected default config to be set")
	}
	if agent.config.MaxIterations != 16 {
		t.Errorf("expected default MaxIterations 16, got %d", agent.config.MaxIterations)
	}
}

func TestNewAgentWithNilSafetyCtl(t *testing.T) {
	mockLLM := &mockLLMClient{name: "test-llm"}
	registry := tools.NewRegistry()
	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		SafetyMode:      SafetyModeStrict,
		SessionDir:      sessionDir,
		OutputMaxLength: 5000,
	}

	agent := NewAgent(mockLLM, registry, config, nil)

	if agent == nil {
		t.Fatal("expected agent to be non-nil")
	}
	if agent.safetyCtl == nil {
		t.Fatal("expected safetyCtl to be auto-created")
	}
}

func TestAgentRunDirectAnswer(t *testing.T) {
	mockLLM := &mockLLMClient{
		name: "test-llm",
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
	}

	registry := tools.NewRegistry()
	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		Temperature:     0.3,
		ToolTimeout:     10 * time.Second,
		MaxTokens:       1024,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)
	defer agent.Close()

	ctx := context.Background()
	result, err := agent.Run(ctx, "What is the meaning of life?")

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
}

func TestAgentRunWithToolCall(t *testing.T) {
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

	mockLLM := &mockLLMClient{
		name: "test-llm",
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
	}

	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		Temperature:     0.3,
		ToolTimeout:     10 * time.Second,
		MaxTokens:       1024,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)
	defer agent.Close()

	ctx := context.Background()
	result, err := agent.Run(ctx, "Please run the mock tool.")

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
	// 注意：后台 Evolver goroutine 可能额外调用 LLM，所以用 >= 而非 ==
	if mockLLM.getCallCount() < 2 {
		t.Errorf("expected at least 2 LLM calls, got %d", mockLLM.getCallCount())
	}
}

func TestAgentRunWithMultipleToolCalls(t *testing.T) {
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

	mockLLM := &mockLLMClient{
		name: "test-llm",
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
								Arguments: `{"arg1":"first"}`,
							},
						},
						{
							ID:   "call-2",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "mock_tool",
								Arguments: `{"arg1":"second"}`,
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
					Content: "Done with both tools.",
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
		},
	}

	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		Temperature:     0.3,
		ToolTimeout:     10 * time.Second,
		MaxTokens:       1024,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)
	defer agent.Close()

	ctx := context.Background()
	result, err := agent.Run(ctx, "Run the mock tool twice.")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if !result.Success {
		t.Error("expected result.Success to be true")
	}
	if result.Output != "Done with both tools." {
		t.Errorf("expected output 'Done with both tools.', got %q", result.Output)
	}
	// 注意：后台 Evolver goroutine 可能额外调用 LLM，所以用 >= 而非 ==
	if mockLLM.getCallCount() < 2 {
		t.Errorf("expected at least 2 LLM calls, got %d", mockLLM.getCallCount())
	}
}

func TestAgentRunWithLLMError(t *testing.T) {
	mockLLM := &mockLLMClient{
		name: "test-llm",
		err:  errors.New("llm connection failed"),
	}

	registry := tools.NewRegistry()
	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		Temperature:     0.3,
		ToolTimeout:     10 * time.Second,
		MaxTokens:       1024,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)
	defer agent.Close()

	ctx := context.Background()
	result, err := agent.Run(ctx, "This will fail.")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Error("expected result to be nil on error")
	}
	if err.Error() != "LLM调用失败: llm connection failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAgentRunMaxIterations(t *testing.T) {
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

	mockLLM := &mockLLMClient{
		name:      "test-llm",
		responses: responses,
	}

	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   3,
		Temperature:     0.3,
		ToolTimeout:     10 * time.Second,
		MaxTokens:       1024,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)
	defer agent.Close()

	ctx := context.Background()
	result, err := agent.Run(ctx, "This will hit max iterations.")

	if err == nil {
		t.Fatal("expected error due to max iterations")
	}
	if result != nil {
		t.Error("expected result to be nil when max iterations reached")
	}
	// 注意：后台 Evolver goroutine 可能额外调用 LLM，所以用 >= 而非 ==
	if mockLLM.getCallCount() < 3 {
		t.Errorf("expected at least 3 LLM calls, got %d", mockLLM.getCallCount())
	}
}

func TestTrimMessages(t *testing.T) {
	t.Run("未初始化 tokenizer 的兜底策略", func(t *testing.T) {
		agent := &Agent{}

		tests := []struct {
			name     string
			msgs     []llm.Message
			expected int
		}{
			{
				name:     "empty",
				msgs:     []llm.Message{},
				expected: 0,
			},
			{
				name:     "single system message",
				msgs:     []llm.Message{{Role: "system", Content: "sys"}},
				expected: 1,
			},
			{
				name:     "system + user within limit",
				msgs:     []llm.Message{{Role: "system", Content: "sys"}, {Role: "user", Content: "hi"}},
				expected: 2,
			},
			{
				name:     "exactly 21 messages",
				msgs:     generateMessages(21),
				expected: 21,
			},
			{
				name:     "22 messages should trim to 21",
				msgs:     generateMessages(22),
				expected: 21,
			},
			{
				name:     "30 messages should trim to 21",
				msgs:     generateMessages(30),
				expected: 21,
			},
			{
				name:     "100 messages should trim to 21",
				msgs:     generateMessages(100),
				expected: 21,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := agent.trimMessages(tt.msgs)
				if len(result) != tt.expected {
					t.Errorf("expected %d messages, got %d", tt.expected, len(result))
				}
				if len(result) > 0 && result[0].Role != "system" {
					t.Errorf("expected first message to be system, got %s", result[0].Role)
				}
				if len(tt.msgs) > 21 {
					expectedFirstContent := tt.msgs[0].Content
					if result[0].Content != expectedFirstContent {
						t.Errorf("expected first content %q, got %q", expectedFirstContent, result[0].Content)
					}
					expectedLastContent := tt.msgs[len(tt.msgs)-1].Content
					if result[len(result)-1].Content != expectedLastContent {
						t.Errorf("expected last content %q, got %q", expectedLastContent, result[len(result)-1].Content)
					}
				}
			})
		}
	})

	t.Run("基于 token 的智能裁剪", func(t *testing.T) {
		// 使用较小的上限以便精确控制
		config := &Config{MaxContextTokens: 200}
		agent := NewAgent(nil, nil, config, nil)

		// 生成少量短消息，token 数很少，应保留全部
		t.Run("不超限时保留全部", func(t *testing.T) {
			msgs := generateMessages(10)
			result := agent.trimMessages(msgs)
			if len(result) != 10 {
				t.Errorf("expected 10 messages, got %d", len(result))
			}
			if result[0].Role != "system" {
				t.Errorf("expected first message role system, got %s", result[0].Role)
			}
		})

		// 生成大量长消息，超出 token 上限时应裁剪
		t.Run("超限时裁剪", func(t *testing.T) {
			longContent := strings.Repeat("abcdefghijklmnopqrstuvwxyz", 10) // 260 字母 ≈ 65 tokens
			msgs := generateLongMessages(10, longContent)
			result := agent.trimMessages(msgs)
			// system(3) + 2*65 = 133 < 180; system(3) + 3*65 = 198 > 180
			// 应保留 system + 最近 2 条 = 3 条
			if len(result) != 3 {
				t.Errorf("expected 3 messages, got %d", len(result))
			}
			if result[0].Role != "system" {
				t.Errorf("expected first message role system, got %s", result[0].Role)
			}
			if result[len(result)-1].Content != msgs[len(msgs)-1].Content {
				t.Errorf("expected last content to be the latest message")
			}
		})

		// 单条超长消息超过限制时应截断
		t.Run("超长单条截断", func(t *testing.T) {
			veryLongContent := strings.Repeat("abcdefghij", 80) // 800 字母 ≈ 200 tokens
			msgs := []llm.Message{
				{Role: "system", Content: "system-prompt"},
				{Role: "user", Content: veryLongContent},
			}
			result := agent.trimMessages(msgs)
			if len(result) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(result))
			}
			originalRunes := []rune(veryLongContent)
			expectedRunes := int(float64(len(originalRunes)) * 0.8)
			resultRunes := []rune(result[1].Content)
			if len(resultRunes) != expectedRunes {
				t.Errorf("expected truncated content length %d runes, got %d", expectedRunes, len(resultRunes))
			}
		})
	})
}

func generateMessages(count int) []llm.Message {
	msgs := make([]llm.Message, count)
	msgs[0] = llm.Message{Role: "system", Content: "system-prompt"}
	for i := 1; i < count; i++ {
		msgs[i] = llm.Message{Role: "user", Content: "msg-" + strconv.Itoa(i)}
	}
	return msgs
}

func TestLoadConfigFromEnv(t *testing.T) {
	envVars := []string{
		"OPSXCLI_AGENT_MAX_ITERATIONS",
		"OPSXCLI_AGENT_TEMPERATURE",
		"OPSXCLI_AGENT_TOOL_TIMEOUT",
		"OPSXCLI_AGENT_MAX_TOKENS",
		"OPSXCLI_AGENT_SAFETY_MODE",
		"OPSXCLI_AGENT_SESSION_DIR",
		"OPSXCLI_AGENT_AUTO_APPROVE",
		"OPSXCLI_AGENT_SSH_TIMEOUT",
		"OPSXCLI_AGENT_OUTPUT_MAX_LENGTH",
		"OPSXCLI_AGENT_MAX_CONTEXT_TOKENS",
	}
	original := make(map[string]string)
	for _, v := range envVars {
		original[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for k, v := range original {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	t.Run("all env vars set", func(t *testing.T) {
		os.Setenv("OPSXCLI_AGENT_MAX_ITERATIONS", "20")
		os.Setenv("OPSXCLI_AGENT_TEMPERATURE", "0.7")
		os.Setenv("OPSXCLI_AGENT_TOOL_TIMEOUT", "120")
		os.Setenv("OPSXCLI_AGENT_MAX_TOKENS", "8192")
		os.Setenv("OPSXCLI_AGENT_SAFETY_MODE", "strict")
		os.Setenv("OPSXCLI_AGENT_SESSION_DIR", "/tmp/test-sessions")
		os.Setenv("OPSXCLI_AGENT_AUTO_APPROVE", "true")
		os.Setenv("OPSXCLI_AGENT_SSH_TIMEOUT", "30")
		os.Setenv("OPSXCLI_AGENT_OUTPUT_MAX_LENGTH", "5000")
		os.Setenv("OPSXCLI_AGENT_MAX_CONTEXT_TOKENS", "8192")

		config := LoadConfig()

		if config.MaxIterations != 20 {
			t.Errorf("expected MaxIterations 20, got %d", config.MaxIterations)
		}
		if config.Temperature != 0.7 {
			t.Errorf("expected Temperature 0.7, got %f", config.Temperature)
		}
		if config.ToolTimeout != 120*time.Second {
			t.Errorf("expected ToolTimeout 120s, got %v", config.ToolTimeout)
		}
		if config.MaxTokens != 8192 {
			t.Errorf("expected MaxTokens 8192, got %d", config.MaxTokens)
		}
		if config.SafetyMode != SafetyModeStrict {
			t.Errorf("expected SafetyMode strict, got %s", config.SafetyMode)
		}
		if config.SessionDir != "/tmp/test-sessions" {
			t.Errorf("expected SessionDir /tmp/test-sessions, got %s", config.SessionDir)
		}
		if !config.AutoApprove {
			t.Error("expected AutoApprove true")
		}
		if config.SSHConnectTimeout != 30*time.Second {
			t.Errorf("expected SSHConnectTimeout 30s, got %v", config.SSHConnectTimeout)
		}
		if config.OutputMaxLength != 5000 {
			t.Errorf("expected OutputMaxLength 5000, got %d", config.OutputMaxLength)
		}
		if config.MaxContextTokens != 8192 {
			t.Errorf("expected MaxContextTokens 8192, got %d", config.MaxContextTokens)
		}
	})

	t.Run("no env vars set", func(t *testing.T) {
		for _, v := range envVars {
			os.Unsetenv(v)
		}

		config := LoadConfig()
		defaultConfig := NewDefaultConfig()

		if config.MaxIterations != defaultConfig.MaxIterations {
			t.Errorf("expected default MaxIterations %d, got %d", defaultConfig.MaxIterations, config.MaxIterations)
		}
		if config.Temperature != defaultConfig.Temperature {
			t.Errorf("expected default Temperature %f, got %f", defaultConfig.Temperature, config.Temperature)
		}
		if config.ToolTimeout != defaultConfig.ToolTimeout {
			t.Errorf("expected default ToolTimeout %v, got %v", defaultConfig.ToolTimeout, config.ToolTimeout)
		}
		if config.MaxTokens != defaultConfig.MaxTokens {
			t.Errorf("expected default MaxTokens %d, got %d", defaultConfig.MaxTokens, config.MaxTokens)
		}
		if config.SafetyMode != defaultConfig.SafetyMode {
			t.Errorf("expected default SafetyMode %s, got %s", defaultConfig.SafetyMode, config.SafetyMode)
		}
		if config.AutoApprove != defaultConfig.AutoApprove {
			t.Errorf("expected default AutoApprove %v, got %v", defaultConfig.AutoApprove, config.AutoApprove)
		}
		if config.SSHConnectTimeout != defaultConfig.SSHConnectTimeout {
			t.Errorf("expected default SSHConnectTimeout %v, got %v", defaultConfig.SSHConnectTimeout, config.SSHConnectTimeout)
		}
		if config.OutputMaxLength != defaultConfig.OutputMaxLength {
			t.Errorf("expected default OutputMaxLength %d, got %d", defaultConfig.OutputMaxLength, config.OutputMaxLength)
		}
		if config.MaxContextTokens != defaultConfig.MaxContextTokens {
			t.Errorf("expected default MaxContextTokens %d, got %d", defaultConfig.MaxContextTokens, config.MaxContextTokens)
		}
	})

	t.Run("invalid values ignored", func(t *testing.T) {
		for _, v := range envVars {
			os.Unsetenv(v)
		}

		os.Setenv("OPSXCLI_AGENT_MAX_ITERATIONS", "not-a-number")
		os.Setenv("OPSXCLI_AGENT_TEMPERATURE", "2.5") // out of range
		os.Setenv("OPSXCLI_AGENT_TOOL_TIMEOUT", "-10")
		os.Setenv("OPSXCLI_AGENT_SAFETY_MODE", "unknown_mode")

		config := LoadConfig()
		defaultConfig := NewDefaultConfig()

		if config.MaxIterations != defaultConfig.MaxIterations {
			t.Errorf("expected default MaxIterations %d, got %d", defaultConfig.MaxIterations, config.MaxIterations)
		}
		if config.Temperature != defaultConfig.Temperature {
			t.Errorf("expected default Temperature %f, got %f", defaultConfig.Temperature, config.Temperature)
		}
		if config.ToolTimeout != defaultConfig.ToolTimeout {
			t.Errorf("expected default ToolTimeout %v, got %v", defaultConfig.ToolTimeout, config.ToolTimeout)
		}
		if config.SafetyMode != defaultConfig.SafetyMode {
			t.Errorf("expected default SafetyMode %s, got %s", defaultConfig.SafetyMode, config.SafetyMode)
		}
	})
}

func TestAgentRunWithToolExecutionError(t *testing.T) {
	mockT := &mockTool{
		name:        "failing_tool",
		description: "A tool that fails",
		parameters: map[string]interface{}{
			"type": "object",
		},
		risk:    tools.RiskSafe,
		result:  nil,
		execErr: errors.New("execution failed"),
	}

	registry := tools.NewRegistry()
	registry.Register(mockT)

	mockLLM := &mockLLMClient{
		name: "test-llm",
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
								Name:      "failing_tool",
								Arguments: `{}`,
							},
						},
					},
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
			{
				ID:    "resp-2",
				Model: "mock",
				Message: llm.Message{
					Role:    "assistant",
					Content: "The tool failed, but I handled it.",
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
		},
	}

	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		Temperature:     0.3,
		ToolTimeout:     10 * time.Second,
		MaxTokens:       1024,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)
	defer agent.Close()

	ctx := context.Background()
	result, err := agent.Run(ctx, "Run the failing tool.")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if !result.Success {
		t.Error("expected result.Success to be true")
	}
	if result.Output != "The tool failed, but I handled it." {
		t.Errorf("unexpected output: %q", result.Output)
	}
}

func TestAgentRunWithToolResultFailure(t *testing.T) {
	mockT := &mockTool{
		name:        "failing_tool",
		description: "A tool that returns failure",
		parameters: map[string]interface{}{
			"type": "object",
		},
		risk: tools.RiskSafe,
		result: &tools.Result{
			Success: false,
			Output:  "partial output",
			Error:   "something went wrong",
		},
	}

	registry := tools.NewRegistry()
	registry.Register(mockT)

	mockLLM := &mockLLMClient{
		name: "test-llm",
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
								Name:      "failing_tool",
								Arguments: `{}`,
							},
						},
					},
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
			{
				ID:    "resp-2",
				Model: "mock",
				Message: llm.Message{
					Role:    "assistant",
					Content: "The tool reported a failure.",
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
		},
	}

	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		Temperature:     0.3,
		ToolTimeout:     10 * time.Second,
		MaxTokens:       1024,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)
	defer agent.Close()

	ctx := context.Background()
	result, err := agent.Run(ctx, "Run the failing tool.")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if !result.Success {
		t.Error("expected result.Success to be true")
	}
	if result.Output != "The tool reported a failure." {
		t.Errorf("unexpected output: %q", result.Output)
	}
}

func TestAgentRunWithLoopDetection(t *testing.T) {
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

	// Always return the same tool call to trigger loop detection.
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
							Arguments: `{"same":"args"}`,
						},
					},
				},
			},
			Usage: llm.Usage{TotalTokens: 5},
		})
	}

	mockLLM := &mockLLMClient{
		name:      "test-llm",
		responses: responses,
	}

	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		Temperature:     0.3,
		ToolTimeout:     10 * time.Second,
		MaxTokens:       1024,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)
	defer agent.Close()

	ctx := context.Background()
	_, err := agent.Run(ctx, "Trigger loop detection.")

	// Loop detection kicks in on the 4th identical call, but max iterations
	// will be reached after 5 rounds since the LLM never returns a direct answer.
	if err == nil {
		t.Fatal("expected error due to max iterations or loop")
	}
	if mockLLM.getCallCount() < 4 {
		t.Errorf("expected at least 4 LLM calls, got %d", mockLLM.getCallCount())
	}
}

func TestAgentRunWithUnknownTool(t *testing.T) {
	mockLLM := &mockLLMClient{
		name: "test-llm",
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
								Name:      "nonexistent_tool",
								Arguments: `{}`,
							},
						},
					},
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
			{
				ID:    "resp-2",
				Model: "mock",
				Message: llm.Message{
					Role:    "assistant",
					Content: "The tool was not found.",
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
		},
	}

	registry := tools.NewRegistry()
	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		Temperature:     0.3,
		ToolTimeout:     10 * time.Second,
		MaxTokens:       1024,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)
	defer agent.Close()

	ctx := context.Background()
	result, err := agent.Run(ctx, "Call unknown tool.")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if !result.Success {
		t.Error("expected result.Success to be true")
	}
	if result.Output != "The tool was not found." {
		t.Errorf("unexpected output: %q", result.Output)
	}
}

func TestAgentRunWithInvalidToolArguments(t *testing.T) {
	mockLLM := &mockLLMClient{
		name: "test-llm",
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
								Arguments: `not valid json`,
							},
						},
					},
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
			{
				ID:    "resp-2",
				Model: "mock",
				Message: llm.Message{
					Role:    "assistant",
					Content: "Arguments were invalid.",
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
		},
	}

	mockT := &mockTool{
		name:        "mock_tool",
		description: "A mock tool",
		parameters:  map[string]interface{}{"type": "object"},
		risk:        tools.RiskSafe,
		result: &tools.Result{
			Success: true,
			Output:  "never reached",
		},
	}

	registry := tools.NewRegistry()
	registry.Register(mockT)

	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		Temperature:     0.3,
		ToolTimeout:     10 * time.Second,
		MaxTokens:       1024,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 10000,
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)
	defer agent.Close()

	ctx := context.Background()
	result, err := agent.Run(ctx, "Call with bad args.")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if !result.Success {
		t.Error("expected result.Success to be true")
	}
	if result.Output != "Arguments were invalid." {
		t.Errorf("unexpected output: %q", result.Output)
	}
}

func TestAgentRunWithOutputTruncation(t *testing.T) {
	longOutput := make([]byte, 200)
	for i := range longOutput {
		longOutput[i] = 'a'
	}

	mockT := &mockTool{
		name:        "mock_tool",
		description: "A mock tool",
		parameters:  map[string]interface{}{"type": "object"},
		risk:        tools.RiskSafe,
		result: &tools.Result{
			Success: true,
			Output:  string(longOutput),
		},
	}

	registry := tools.NewRegistry()
	registry.Register(mockT)

	mockLLM := &mockLLMClient{
		name: "test-llm",
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
								Arguments: `{}`,
							},
						},
					},
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
			{
				ID:    "resp-2",
				Model: "mock",
				Message: llm.Message{
					Role:    "assistant",
					Content: "Got truncated output.",
				},
				Usage: llm.Usage{TotalTokens: 10},
			},
		},
	}

	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:   5,
		Temperature:     0.3,
		ToolTimeout:     10 * time.Second,
		MaxTokens:       1024,
		SafetyMode:      SafetyModeBalanced,
		SessionDir:      sessionDir,
		AutoApprove:     true,
		OutputMaxLength: 50, // Very small to force truncation
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)
	defer agent.Close()

	ctx := context.Background()
	result, err := agent.Run(ctx, "Get long output.")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if !result.Success {
		t.Error("expected result.Success to be true")
	}
}

func TestAgentStream(t *testing.T) {
	mockLLM := &mockLLMClient{name: "test-llm"}
	registry := tools.NewRegistry()
	sessionDir := t.TempDir()

	config := NewDefaultConfig()
	config.SessionDir = sessionDir

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)

	ctx := context.Background()
	ch, err := agent.llmClient.Stream(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
	})

	if err != nil {
		t.Fatalf("expected no error from stream, got %v", err)
	}

	chunk, ok := <-ch
	if !ok {
		t.Fatal("expected at least one chunk")
	}
	if !chunk.Finish {
		t.Error("expected finish to be true")
	}
	if chunk.Delta.Content != "streamed" {
		t.Errorf("expected content 'streamed', got %q", chunk.Delta.Content)
	}
}

// TestGetLastEvolveHint 测试会话级进化结果提示
// 验证 W4 修复: Evolver 结果不再被丢弃，而是反馈到后续 Prompt
func TestGetLastEvolveHint(t *testing.T) {
	t.Run("nil_result_returns_empty", func(t *testing.T) {
		mockLLM := &mockLLMClient{
			responses: []llm.CompletionResponse{},
		}
		registry := tools.NewRegistry()
		sessionDir := t.TempDir()
		config := NewDefaultConfig()
		config.SessionDir = sessionDir
		safetyCtl := safety.NewController(safety.SafetyModeBalanced)
		safetyCtl.SetAutoApprove(true)

		agent := NewAgent(mockLLM, registry, config, safetyCtl)
		defer agent.Close()

		hint := agent.getLastEvolveHint()
		if hint != "" {
			t.Errorf("expected empty hint for nil result, got %q", hint)
		}
	})

	t.Run("no_experience_added_returns_empty", func(t *testing.T) {
		mockLLM := &mockLLMClient{
			responses: []llm.CompletionResponse{},
		}
		registry := tools.NewRegistry()
		sessionDir := t.TempDir()
		config := NewDefaultConfig()
		config.SessionDir = sessionDir
		safetyCtl := safety.NewController(safety.SafetyModeBalanced)
		safetyCtl.SetAutoApprove(true)

		agent := NewAgent(mockLLM, registry, config, safetyCtl)
		defer agent.Close()

		// 设置一个没有新增经验的进化结果
		agent.evolveResultMu.Lock()
		agent.lastEvolveResult = &evolver.EvolveResult{
			ExperienceAdded: false,
		}
		agent.evolveResultMu.Unlock()

		hint := agent.getLastEvolveHint()
		if hint != "" {
			t.Errorf("expected empty hint when no experience added, got %q", hint)
		}
	})

	t.Run("experience_added_returns_hint", func(t *testing.T) {
		mockLLM := &mockLLMClient{
			responses: []llm.CompletionResponse{},
		}
		registry := tools.NewRegistry()
		sessionDir := t.TempDir()
		config := NewDefaultConfig()
		config.SessionDir = sessionDir
		safetyCtl := safety.NewController(safety.SafetyModeBalanced)
		safetyCtl.SetAutoApprove(true)

		agent := NewAgent(mockLLM, registry, config, safetyCtl)
		defer agent.Close()

		// 设置一个有经验的进化结果
		agent.evolveResultMu.Lock()
		agent.lastEvolveResult = &evolver.EvolveResult{
			TaskType:        "磁盘分析",
			ToolSequence:    []string{"local_bash", "ssh_execute"},
			LearnedHint:     "优先使用 df -h 检查磁盘使用情况",
			ExperienceAdded: true,
			Consolidated:    true,
		}
		agent.evolveResultMu.Unlock()

		hint := agent.getLastEvolveHint()
		if hint == "" {
			t.Fatal("expected non-empty hint, got empty")
		}
		if !strings.Contains(hint, "磁盘分析") {
			t.Errorf("expected hint to contain task type '磁盘分析', got %q", hint)
		}
		if !strings.Contains(hint, "df -h") {
			t.Errorf("expected hint to contain learned hint, got %q", hint)
		}
		if !strings.Contains(hint, "local_bash → ssh_execute") {
			t.Errorf("expected hint to contain tool sequence, got %q", hint)
		}
		if !strings.Contains(hint, "整合") {
			t.Errorf("expected hint to mention consolidation, got %q", hint)
		}
	})

	t.Run("long_hint_truncated", func(t *testing.T) {
		mockLLM := &mockLLMClient{
			responses: []llm.CompletionResponse{},
		}
		registry := tools.NewRegistry()
		sessionDir := t.TempDir()
		config := NewDefaultConfig()
		config.SessionDir = sessionDir
		safetyCtl := safety.NewController(safety.SafetyModeBalanced)
		safetyCtl.SetAutoApprove(true)

		agent := NewAgent(mockLLM, registry, config, safetyCtl)
		defer agent.Close()

		longHint := ""
		for i := 0; i < 50; i++ {
			longHint += "这是一段很长的提示内容用于测试截断功能"
		}

		agent.evolveResultMu.Lock()
		agent.lastEvolveResult = &evolver.EvolveResult{
			TaskType:        "通用运维",
			LearnedHint:     longHint,
			ExperienceAdded: true,
		}
		agent.evolveResultMu.Unlock()

		hint := agent.getLastEvolveHint()
		if !strings.Contains(hint, "...") {
			t.Errorf("expected hint to be truncated with '...', got %d chars", len(hint))
		}
	})
}

// ============================================================================
// trimMessages 相关测试
// ============================================================================

// TestEnsureToolMessageIntegrity_PairedToolMessages 验证正常配对的 tool 消息不受影响
func TestEnsureToolMessageIntegrity_PairedToolMessages(t *testing.T) {
	msgs := []llm.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Run the tool"},
		{Role: "assistant", ToolCalls: []llm.ToolCall{
			{ID: "call-1", Type: "function", Function: llm.FunctionCall{Name: "test_tool", Arguments: "{}"}},
		}},
		{Role: "tool", ToolCallID: "call-1", Content: "tool result"},
		{Role: "assistant", Content: "Done!"},
	}

	result := ensureToolMessageIntegrity(msgs)

	if len(result) != 5 {
		t.Fatalf("expected 5 messages (all preserved), got %d", len(result))
	}
	// 验证所有消息角色顺序正确
	expectedRoles := []string{"system", "user", "assistant", "tool", "assistant"}
	for i, msg := range result {
		if msg.Role != expectedRoles[i] {
			t.Errorf("message[%d]: expected role %q, got %q", i, expectedRoles[i], msg.Role)
		}
	}
}

// TestEnsureToolMessageIntegrity_OrphanedToolMessages 验证孤立的 tool 消息被移除
func TestEnsureToolMessageIntegrity_OrphanedToolMessages(t *testing.T) {
	// 模拟裁剪后 assistant(tool_calls) 被裁掉的场景
	msgs := []llm.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		// assistant(tool_calls) 被裁掉了，直接从 tool 消息开始
		{Role: "tool", ToolCallID: "call-1", Content: "tool result 1"},
		{Role: "tool", ToolCallID: "call-2", Content: "tool result 2"},
		{Role: "assistant", Content: "Done!"},
	}

	result := ensureToolMessageIntegrity(msgs)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages (tool messages removed), got %d: %+v", len(result), result)
	}
	// 验证 tool 消息被移除，只剩 system 和 assistant
	expectedRoles := []string{"system", "assistant"}
	for i, msg := range result {
		if msg.Role != expectedRoles[i] {
			t.Errorf("message[%d]: expected role %q, got %q", i, expectedRoles[i], msg.Role)
		}
	}
}

// TestEnsureToolMessageIntegrity_PartiallyOrphanedToolBlock 验证 assistant 存在但 tool_call_id 不匹配时 tool 消息被移除
func TestEnsureToolMessageIntegrity_PartiallyOrphanedToolBlock(t *testing.T) {
	msgs := []llm.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Do something"},
		// assistant 有 tool_calls 但 ID 不匹配
		{Role: "assistant", ToolCalls: []llm.ToolCall{
			{ID: "call-other", Type: "function", Function: llm.FunctionCall{Name: "other_tool", Arguments: "{}"}},
		}},
		{Role: "tool", ToolCallID: "call-1", Content: "result for call-1"},
		{Role: "assistant", Content: "Final answer"},
	}

	result := ensureToolMessageIntegrity(msgs)

	if len(result) != 4 {
		t.Fatalf("expected 4 messages (orphaned tool removed, mismatched assistant kept), got %d", len(result))
	}
	// tool 消息应被移除，因为 call-1 不在 assistant 的 tool_calls 中
	expectedRoles := []string{"system", "user", "assistant", "assistant"}
	for i, msg := range result {
		if msg.Role != expectedRoles[i] {
			t.Errorf("message[%d]: expected role %q, got %q", i, expectedRoles[i], msg.Role)
		}
	}
}

// TestEnsureToolMessageIntegrity_MultipleToolBlocks 验证多组 tool 消息块独立处理
func TestEnsureToolMessageIntegrity_MultipleToolBlocks(t *testing.T) {
	msgs := []llm.Message{
		{Role: "system", Content: "system"},
		// 第一组：完整配对
		{Role: "assistant", ToolCalls: []llm.ToolCall{
			{ID: "call-1", Type: "function", Function: llm.FunctionCall{Name: "tool_a", Arguments: "{}"}},
		}},
		{Role: "tool", ToolCallID: "call-1", Content: "result 1"},
		// 第二组：assistant 被裁掉了（tool 消息孤立）
		{Role: "tool", ToolCallID: "call-2", Content: "result 2"},
		{Role: "tool", ToolCallID: "call-3", Content: "result 3"},
		// 第三组：完整配对
		{Role: "assistant", ToolCalls: []llm.ToolCall{
			{ID: "call-4", Type: "function", Function: llm.FunctionCall{Name: "tool_b", Arguments: "{}"}},
		}},
		{Role: "tool", ToolCallID: "call-4", Content: "result 4"},
		{Role: "assistant", Content: "final"},
	}

	result := ensureToolMessageIntegrity(msgs)

	// 第二组 tool 消息（call-2, call-3）应被移除，因为前面没有 assistant(tool_calls)
	expectedRoles := []string{"system", "assistant", "tool", "assistant", "tool", "assistant"}
	if len(result) != len(expectedRoles) {
		t.Fatalf("expected %d messages, got %d", len(expectedRoles), len(result))
	}
	for i, msg := range result {
		if msg.Role != expectedRoles[i] {
			t.Errorf("message[%d]: expected role %q, got %q", i, expectedRoles[i], msg.Role)
		}
	}
}

// TestEnsureToolMessageIntegrity_EmptyMessages 验证空消息列表不会 panic
func TestEnsureToolMessageIntegrity_EmptyMessages(t *testing.T) {
	result := ensureToolMessageIntegrity(nil)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %d messages", len(result))
	}

	result = ensureToolMessageIntegrity([]llm.Message{})
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %d messages", len(result))
	}
}

// TestEnsureToolMessageIntegrity_NoToolMessages 验证没有 tool 消息时不影响
func TestEnsureToolMessageIntegrity_NoToolMessages(t *testing.T) {
	msgs := []llm.Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	result := ensureToolMessageIntegrity(msgs)

	if len(result) != 3 {
		t.Fatalf("expected 3 messages (unchanged), got %d", len(result))
	}
}

// TestEnsureToolMessageIntegrity_ToolBlockAtStart 验证 tool 消息在消息列表开头时被移除
func TestEnsureToolMessageIntegrity_ToolBlockAtStart(t *testing.T) {
	// 极端场景：消息列表第一条就是 tool（system 被裁掉或 tool 在最前面）
	msgs := []llm.Message{
		{Role: "tool", ToolCallID: "call-1", Content: "result 1"},
		{Role: "tool", ToolCallID: "call-2", Content: "result 2"},
		{Role: "assistant", Content: "Done!"},
	}

	result := ensureToolMessageIntegrity(msgs)

	if len(result) != 1 {
		t.Fatalf("expected 1 message (tool block at start removed), got %d", len(result))
	}
	if result[0].Role != "assistant" {
		t.Errorf("expected remaining message to be assistant, got %q", result[0].Role)
	}
}

// TestTrimMessages_OrphanedToolFix 集成测试：验证 trimMessages 裁剪后不会产生孤立 tool 消息
func TestTrimMessages_OrphanedToolFix(t *testing.T) {
	// 构造一个 Agent，设置较小的 maxContextTokens 使得裁剪发生
	mockLLM := &mockLLMClient{name: "test-llm"}
	registry := tools.NewRegistry()
	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, &Config{
		MaxIterations:    5,
		MaxContextTokens: 200, // 很小的上下文窗口
		SessionDir:       t.TempDir(),
		AutoApprove:      true,
		OutputMaxLength:  5000,
	}, safetyCtl)
	defer agent.Close()

	// 构造一组很长的历史消息，其中包含 tool_calls/tool 对
	msgs := []llm.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		// 早期对话（内容很长，会被裁掉）
		{Role: "user", Content: strings.Repeat("This is a very long early message. ", 100)},
		{Role: "assistant", ToolCalls: []llm.ToolCall{
			{ID: "call-old-1", Type: "function", Function: llm.FunctionCall{Name: "tool_a", Arguments: "{}"}},
		}},
		{Role: "tool", ToolCallID: "call-old-1", Content: strings.Repeat("old tool result ", 50)},
		{Role: "assistant", Content: strings.Repeat("Old response ", 50)},
		// 近期对话（会被保留）
		{Role: "user", Content: "Recent question"},
		{Role: "assistant", ToolCalls: []llm.ToolCall{
			{ID: "call-new-1", Type: "function", Function: llm.FunctionCall{Name: "tool_b", Arguments: "{}"}},
		}},
		{Role: "tool", ToolCallID: "call-new-1", Content: "new tool result"},
		{Role: "assistant", Content: "Final answer"},
	}

	trimmed := agent.trimMessages(msgs)

	// 验证裁剪后不会出现孤立的 tool 消息
	for i, msg := range trimmed {
		if msg.Role == "tool" {
			if i == 0 {
				t.Errorf("tool message at index 0 has no preceding assistant")
				continue
			}
			prev := trimmed[i-1]
			if prev.Role != "assistant" || len(prev.ToolCalls) == 0 {
				t.Errorf("tool message at index %d has no preceding assistant(tool_calls), prev role=%q", i, prev.Role)
			}
			// 验证 tool_call_id 匹配
			found := false
			for _, tc := range prev.ToolCalls {
				if tc.ID == msg.ToolCallID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("tool message at index %d (tool_call_id=%q) not found in preceding assistant's tool_calls", i, msg.ToolCallID)
			}
		}
	}
}
