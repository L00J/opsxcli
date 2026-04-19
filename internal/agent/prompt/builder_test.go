package prompt

import (
	"strings"
	"testing"

	"opsxcli/internal/agent/evolver"
	"opsxcli/internal/agent/tools"
	"opsxcli/internal/llm"
)

// TestNewBuilderV2 测试创建 BuilderV2
func TestNewBuilderV2(t *testing.T) {
	b := NewBuilderV2()
	if b == nil {
		t.Fatal("NewBuilderV2() returned nil")
	}
	if b.systemPrompt == "" {
		t.Error("systemPrompt should not be empty")
	}
	if b.enableMemory {
		t.Error("enableMemory should be false by default")
	}
}

// TestNewBuilderV2WithMemory 测试创建带记忆的 BuilderV2
func TestNewBuilderV2WithMemory(t *testing.T) {
	injector := NewMemoryInjector(&evolver.EnvironmentMemory{}, &evolver.ExperienceMemory{})
	b := NewBuilderV2WithMemory(injector)
	if b == nil {
		t.Fatal("NewBuilderV2WithMemory() returned nil")
	}
	if !b.enableMemory {
		t.Error("enableMemory should be true")
	}
	if b.memoryInjector != injector {
		t.Error("memoryInjector should be set")
	}
}

// TestBuilderV2_BuildSystemMessage 测试系统消息构建
func TestBuilderV2_BuildSystemMessage(t *testing.T) {
	b := NewBuilderV2()
	msg := b.BuildSystemMessage()

	if msg.Role != "system" {
		t.Errorf("Role = %q, want %q", msg.Role, "system")
	}
	if msg.Content == "" {
		t.Error("Content should not be empty")
	}
	if !strings.Contains(msg.Content, "opsxcli Agent V2") {
		t.Error("Content should contain agent identity")
	}
}

// TestBuilderV2_BuildUserMessage 测试用户消息构建
func TestBuilderV2_BuildUserMessage(t *testing.T) {
	b := NewBuilderV2()
	msg := b.BuildUserMessage("测试查询")

	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}
	if msg.Content != "测试查询" {
		t.Errorf("Content = %q, want %q", msg.Content, "测试查询")
	}
}

// TestBuilderV2_BuildObservationMessage_success 测试成功观察消息
func TestBuilderV2_BuildObservationMessage_success(t *testing.T) {
	b := NewBuilderV2()
	result := &tools.Result{
		Success: true,
		Output:  "hello world",
		Summary: "测试摘要",
	}
	msg := b.BuildObservationMessage("local_bash", result)

	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}
	if !strings.Contains(msg.Content, "hello world") {
		t.Error("Content should contain output")
	}
	if !strings.Contains(msg.Content, "测试摘要") {
		t.Error("Content should contain summary")
	}
}

// TestBuilderV2_BuildObservationMessage_failure 测试失败观察消息
func TestBuilderV2_BuildObservationMessage_failure(t *testing.T) {
	b := NewBuilderV2()
	result := &tools.Result{
		Success: false,
		Error:   "命令未找到",
	}
	msg := b.BuildObservationMessage("local_bash", result)

	if !strings.Contains(msg.Content, "执行失败") {
		t.Error("Content should indicate failure")
	}
	if !strings.Contains(msg.Content, "命令未找到") {
		t.Error("Content should contain error message")
	}
}

// TestBuilderV2_BuildToolCallMessage 测试工具调用消息
func TestBuilderV2_BuildToolCallMessage(t *testing.T) {
	b := NewBuilderV2()
	toolCalls := []llm.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: llm.FunctionCall{
				Name:      "local_bash",
				Arguments: `{"command":"echo test"}`,
			},
		},
	}
	msg := b.BuildToolCallMessage(toolCalls)

	if msg.Role != "assistant" {
		t.Errorf("Role = %q, want %q", msg.Role, "assistant")
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("ToolCalls length = %d, want 1", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].Function.Name != "local_bash" {
		t.Errorf("ToolCalls[0].Function.Name = %q, want %q", msg.ToolCalls[0].Function.Name, "local_bash")
	}
}

// TestBuilderV2_BuildErrorObservationMessage 测试错误观察消息
func TestBuilderV2_BuildErrorObservationMessage(t *testing.T) {
	b := NewBuilderV2()
	msg := b.BuildErrorObservationMessage("ssh_execute", nil)

	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}
	if !strings.Contains(msg.Content, "ssh_execute") {
		t.Error("Content should contain tool name")
	}
	if !strings.Contains(msg.Content, "执行异常") {
		t.Error("Content should indicate error")
	}
}

// TestBuilderV2_BuildLoopDetectionMessage 测试循环检测消息
func TestBuilderV2_BuildLoopDetectionMessage(t *testing.T) {
	b := NewBuilderV2()
	msg := b.BuildLoopDetectionMessage("local_bash", 5)

	if msg.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Role, "user")
	}
	if !strings.Contains(msg.Content, "循环检测警告") {
		t.Error("Content should contain loop warning")
	}
	if !strings.Contains(msg.Content, "5") {
		t.Error("Content should contain call count")
	}
}

// TestEstimateTokenCount 测试 Token 估算
func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		msgs     []llm.Message
		wantMin  int
		wantMax  int
	}{
		{
			name:    "empty",
			msgs:    []llm.Message{},
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "single short",
			msgs:    []llm.Message{{Role: "user", Content: "hi"}},
			wantMin: 0,
			wantMax: 1,
		},
		{
			name:    "single long",
			msgs:    []llm.Message{{Role: "user", Content: strings.Repeat("a", 300)}},
			wantMin: 50,
			wantMax: 150,
		},
		{
			name: "with tool calls",
			msgs: []llm.Message{{
				Role: "assistant",
				ToolCalls: []llm.ToolCall{{
					Function: llm.FunctionCall{Name: "local_bash", Arguments: `{"command":"echo test"}`},
				}},
			}},
			wantMin: 5,
			wantMax: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokenCount(tt.msgs)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("EstimateTokenCount() = %d, want between %d and %d", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestCompressOldMessages 测试消息压缩
func TestCompressOldMessages(t *testing.T) {
	// 构建一个包含 system + 10 轮对话的消息列表
	msgs := []llm.Message{
		{Role: "system", Content: "system prompt"},
	}
	for i := 0; i < 10; i++ {
		msgs = append(msgs, llm.Message{Role: "user", Content: "question"})
		msgs = append(msgs, llm.Message{Role: "assistant", Content: "answer"})
	}

	result := CompressOldMessages(msgs, 2)

	// System message 应该保留
	if result[0].Role != "system" {
		t.Error("first message should be system")
	}

	// 最近 2 轮应该保留完整
	if len(result) < 5 {
		t.Errorf("result length = %d, expected at least 5", len(result))
	}
}

// TestCompressOldMessages_short 测试短消息不压缩
func TestCompressOldMessages_short(t *testing.T) {
	msgs := []llm.Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "q1"},
		{Role: "assistant", Content: "a1"},
	}

	result := CompressOldMessages(msgs, 5)
	if len(result) != len(msgs) {
		t.Errorf("short messages should not be compressed, got %d, want %d", len(result), len(msgs))
	}
}

// TestSerializeToolCalls 测试工具调用序列化
func TestSerializeToolCalls(t *testing.T) {
	// 空列表
	if got := SerializeToolCalls([]llm.ToolCall{}); got != "[]" {
		t.Errorf("SerializeToolCalls(empty) = %q, want %q", got, "[]")
	}

	// 有内容
	calls := []llm.ToolCall{
		{ID: "call_1", Function: llm.FunctionCall{Name: "local_bash", Arguments: `{"command":"ls"}`}},
	}
	got := SerializeToolCalls(calls)
	if !strings.Contains(got, "local_bash") {
		t.Errorf("SerializeToolCalls() = %q, should contain 'local_bash'", got)
	}
}

// TestGetStaticSystemPrompt 测试获取静态 System Prompt
func TestGetStaticSystemPrompt(t *testing.T) {
	prompt := GetStaticSystemPrompt()
	if prompt == "" {
		t.Error("GetStaticSystemPrompt() returned empty string")
	}
	if !strings.Contains(prompt, "opsxcli Agent V2") {
		t.Error("System Prompt should contain agent identity")
	}
}

// TestBuildSystemPrompt 测试动态 System Prompt 构建
func TestBuildSystemPrompt(t *testing.T) {
	memoryContext := "【历史经验】\n已知服务器: 192.168.1.100 (Ubuntu)"
	prompt := BuildSystemPrompt(memoryContext)

	if !strings.Contains(prompt, memoryContext) {
		t.Error("BuildSystemPrompt() should inject memory context")
	}
	if !strings.Contains(prompt, "opsxcli Agent V2") {
		t.Error("BuildSystemPrompt() should contain base system prompt")
	}
}

// TestBuildSystemPrompt_empty 测试空记忆上下文
func TestBuildSystemPrompt_empty(t *testing.T) {
	prompt := BuildSystemPrompt("")
	if !strings.Contains(prompt, "opsxcli Agent V2") {
		t.Error("BuildSystemPrompt() should still contain base prompt with empty memory")
	}
}
