package core

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"opsxcli/internal/agent/safety"
	"opsxcli/internal/agent/tools"
	"opsxcli/internal/llm"
)

// mockLLMClient implements llm.Client for testing.
// It cycles through a slice of prepared responses on each Complete call.
type mockLLMClient struct {
	name      string
	responses []llm.CompletionResponse
	callIndex int
	err       error
}

func (m *mockLLMClient) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
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
		MaxIterations:     5,
		Temperature:       0.5,
		ToolTimeout:       30 * time.Second,
		MaxTokens:         2048,
		SafetyMode:        SafetyModeBalanced,
		SessionDir:        sessionDir,
		AutoApprove:       true,
		OutputMaxLength:   5000,
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
	if agent.config.MaxIterations != 10 {
		t.Errorf("expected default MaxIterations 10, got %d", agent.config.MaxIterations)
	}
}

func TestNewAgentWithNilSafetyCtl(t *testing.T) {
	mockLLM := &mockLLMClient{name: "test-llm"}
	registry := tools.NewRegistry()
	sessionDir := t.TempDir()

	config := &Config{
		MaxIterations:     5,
		SafetyMode:        SafetyModeStrict,
		SessionDir:        sessionDir,
		OutputMaxLength:   5000,
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
	if mockLLM.callIndex != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mockLLM.callIndex)
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
	if mockLLM.callIndex != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mockLLM.callIndex)
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
	result, err := agent.Run(ctx, "This will hit max iterations.")

	if err == nil {
		t.Fatal("expected error due to max iterations")
	}
	if result != nil {
		t.Error("expected result to be nil when max iterations reached")
	}
	if mockLLM.callIndex != 3 {
		t.Errorf("expected 3 LLM calls, got %d", mockLLM.callIndex)
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
	_, err := agent.Run(ctx, "Trigger loop detection.")

	// Loop detection kicks in on the 4th identical call, but max iterations
	// will be reached after 5 rounds since the LLM never returns a direct answer.
	if err == nil {
		t.Fatal("expected error due to max iterations or loop")
	}
	if mockLLM.callIndex < 4 {
		t.Errorf("expected at least 4 LLM calls, got %d", mockLLM.callIndex)
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
		MaxIterations:     5,
		Temperature:       0.3,
		ToolTimeout:       10 * time.Second,
		MaxTokens:         1024,
		SafetyMode:        SafetyModeBalanced,
		SessionDir:        sessionDir,
		AutoApprove:       true,
		OutputMaxLength:   50, // Very small to force truncation
	}

	safetyCtl := safety.NewController(safety.SafetyModeBalanced)
	safetyCtl.SetAutoApprove(true)

	agent := NewAgent(mockLLM, registry, config, safetyCtl)

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
