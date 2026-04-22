package prompt

import (
	"fmt"
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
	if !strings.Contains(msg.Content, "opsxcli") {
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
	if !strings.Contains(prompt, "opsxcli") {
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
	if !strings.Contains(prompt, "opsxcli") {
		t.Error("BuildSystemPrompt() should contain base system prompt")
	}
}

// TestBuildSystemPrompt_empty 测试空记忆上下文
func TestBuildSystemPrompt_empty(t *testing.T) {
	prompt := BuildSystemPrompt("")
	if !strings.Contains(prompt, "opsxcli") {
		t.Error("BuildSystemPrompt() should still contain base prompt with empty memory")
	}
}

// ═══════════════════════════════════════════════════════════════
// V1 Builder 兼容层测试
// ═══════════════════════════════════════════════════════════════

// TestNewBuilder 测试创建 V1 Builder
func TestNewBuilder(t *testing.T) {
	b := NewBuilder()
	if b == nil {
		t.Fatal("NewBuilder() 返回 nil")
	}
}

// TestV1Builder_BuildSystemMessage 测试 V1 系统消息构建
func TestV1Builder_BuildSystemMessage(t *testing.T) {
	b := NewBuilder()
	msg := b.BuildSystemMessage()

	if msg.Role != "system" {
		t.Errorf("Role = %q, 期望 %q", msg.Role, "system")
	}
	if msg.Content == "" {
		t.Error("Content 不应为空")
	}
	// V1 使用 SystemPrompt 常量
	if !strings.Contains(msg.Content, "opsxcli") {
		t.Error("Content 应包含 agent 身份标识")
	}
}

// TestV1Builder_BuildUserMessage 测试 V1 用户消息构建
func TestV1Builder_BuildUserMessage(t *testing.T) {
	b := NewBuilder()
	msg := b.BuildUserMessage("检查服务器状态")

	if msg.Role != "user" {
		t.Errorf("Role = %q, 期望 %q", msg.Role, "user")
	}
	if msg.Content != "检查服务器状态" {
		t.Errorf("Content = %q, 期望 %q", msg.Content, "检查服务器状态")
	}
}

// TestV1Builder_BuildObservationMessage 测试 V1 观察消息构建
func TestV1Builder_BuildObservationMessage(t *testing.T) {
	b := NewBuilder()

	t.Run("成功_带摘要", func(t *testing.T) {
		result := &tools.Result{
			Success: true,
			Output:  "total 128\ndrwxr-xr-x 5 root root 4096 Jan 1 .",
			Summary: "共 5 个文件",
		}
		msg := b.BuildObservationMessage("local_bash", result)

		if msg.Role != "user" {
			t.Errorf("Role = %q, 期望 %q", msg.Role, "user")
		}
		if !strings.Contains(msg.Content, "工具 'local_bash' 执行成功") {
			t.Errorf("Content 应包含成功前缀, 实际: %q", msg.Content)
		}
		if !strings.Contains(msg.Content, "total 128") {
			t.Error("Content 应包含 output 内容")
		}
		if !strings.Contains(msg.Content, "共 5 个文件") {
			t.Error("Content 应包含摘要")
		}
	})

	t.Run("成功_无摘要", func(t *testing.T) {
		result := &tools.Result{
			Success: true,
			Output:  "OK",
		}
		msg := b.BuildObservationMessage("local_bash", result)

		if !strings.Contains(msg.Content, "工具 'local_bash' 执行成功") {
			t.Errorf("Content 应包含成功前缀, 实际: %q", msg.Content)
		}
		if strings.Contains(msg.Content, "[摘要]") {
			t.Error("Content 不应包含摘要部分")
		}
	})

	t.Run("失败", func(t *testing.T) {
		result := &tools.Result{
			Success: false,
			Error:   "连接超时",
		}
		msg := b.BuildObservationMessage("ssh_execute", result)

		if !strings.Contains(msg.Content, "工具 'ssh_execute' 执行失败") {
			t.Errorf("Content 应包含失败前缀, 实际: %q", msg.Content)
		}
		if !strings.Contains(msg.Content, "连接超时") {
			t.Error("Content 应包含错误信息")
		}
	})
}

// TestV1Builder_BuildAssistantMessage 测试 V1 助手消息构建
func TestV1Builder_BuildAssistantMessage(t *testing.T) {
	b := NewBuilder()
	msg := b.BuildAssistantMessage("我来帮你检查服务器")

	if msg.Role != "assistant" {
		t.Errorf("Role = %q, 期望 %q", msg.Role, "assistant")
	}
	if msg.Content != "我来帮你检查服务器" {
		t.Errorf("Content = %q, 期望 %q", msg.Content, "我来帮你检查服务器")
	}
}

// TestV1Builder_BuildToolCallMessage 测试 V1 工具调用消息构建
func TestV1Builder_BuildToolCallMessage(t *testing.T) {
	b := NewBuilder()
	toolCalls := []llm.ToolCall{
		{
			ID:   "call_v1_1",
			Type: "function",
			Function: llm.FunctionCall{
				Name:      "local_bash",
				Arguments: `{"command":"df -h"}`,
			},
		},
		{
			ID:   "call_v1_2",
			Type: "function",
			Function: llm.FunctionCall{
				Name:      "ssh_execute",
				Arguments: `{"host":"192.168.1.100","command":"uptime"}`,
			},
		},
	}
	msg := b.BuildToolCallMessage(toolCalls)

	if msg.Role != "assistant" {
		t.Errorf("Role = %q, 期望 %q", msg.Role, "assistant")
	}
	if msg.Content != "" {
		t.Errorf("Content 应为空, 实际: %q", msg.Content)
	}
	if len(msg.ToolCalls) != 2 {
		t.Fatalf("ToolCalls 长度 = %d, 期望 2", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].ID != "call_v1_1" {
		t.Errorf("ToolCalls[0].ID = %q, 期望 %q", msg.ToolCalls[0].ID, "call_v1_1")
	}
	if msg.ToolCalls[1].Function.Name != "ssh_execute" {
		t.Errorf("ToolCalls[1].Function.Name = %q, 期望 %q", msg.ToolCalls[1].Function.Name, "ssh_execute")
	}
}

// TestV1Builder_BuildToolResponseMessage 测试 V1 工具响应消息构建
func TestV1Builder_BuildToolResponseMessage(t *testing.T) {
	b := NewBuilder()

	t.Run("成功_带摘要", func(t *testing.T) {
		result := &tools.Result{
			Success: true,
			Output:  "Filesystem Size Used Avail Use%\n/dev/sda1 50G 20G 30G 40%",
			Summary: "磁盘使用率 40%",
		}
		msg := b.BuildToolResponseMessage("tc_001", "local_bash", result)

		if msg.Role != "tool" {
			t.Errorf("Role = %q, 期望 %q", msg.Role, "tool")
		}
		if msg.ToolCallID != "tc_001" {
			t.Errorf("ToolCallID = %q, 期望 %q", msg.ToolCallID, "tc_001")
		}
		if msg.Name != "local_bash" {
			t.Errorf("Name = %q, 期望 %q", msg.Name, "local_bash")
		}
		if !strings.Contains(msg.Content, "成功:") {
			t.Errorf("Content 应包含 '成功:', 实际: %q", msg.Content)
		}
		if !strings.Contains(msg.Content, "磁盘使用率 40%") {
			t.Error("Content 应包含摘要")
		}
	})

	t.Run("成功_无摘要", func(t *testing.T) {
		result := &tools.Result{
			Success: true,
			Output:  "done",
		}
		msg := b.BuildToolResponseMessage("tc_002", "ssh_execute", result)

		if !strings.Contains(msg.Content, "成功: done") {
			t.Errorf("Content = %q, 期望包含 '成功: done'", msg.Content)
		}
		if strings.Contains(msg.Content, "摘要") {
			t.Error("Content 不应包含摘要")
		}
	})

	t.Run("失败", func(t *testing.T) {
		result := &tools.Result{
			Success: false,
			Error:   "Permission denied",
		}
		msg := b.BuildToolResponseMessage("tc_003", "local_bash", result)

		if !strings.Contains(msg.Content, "失败:") {
			t.Errorf("Content 应包含 '失败:', 实际: %q", msg.Content)
		}
		if !strings.Contains(msg.Content, "Permission denied") {
			t.Error("Content 应包含错误信息")
		}
	})
}

// TestV1Builder_BuildErrorObservationMessage 测试 V1 错误观察消息构建
func TestV1Builder_BuildErrorObservationMessage(t *testing.T) {
	b := NewBuilder()

	t.Run("有错误", func(t *testing.T) {
		err := fmt.Errorf("连接被拒绝")
		msg := b.BuildErrorObservationMessage("ssh_execute", err)

		if msg.Role != "user" {
			t.Errorf("Role = %q, 期望 %q", msg.Role, "user")
		}
		if !strings.Contains(msg.Content, "工具 'ssh_execute' 执行异常") {
			t.Errorf("Content 应包含异常前缀, 实际: %q", msg.Content)
		}
		if !strings.Contains(msg.Content, "连接被拒绝") {
			t.Error("Content 应包含错误信息")
		}
	})

	t.Run("nil错误", func(t *testing.T) {
		msg := b.BuildErrorObservationMessage("local_bash", nil)

		if !strings.Contains(msg.Content, "工具 'local_bash' 执行异常") {
			t.Errorf("Content 应包含工具名, 实际: %q", msg.Content)
		}
	})
}

// TestSerializeToolCallArguments 测试 V1 工具调用参数序列化
func TestSerializeToolCallArguments(t *testing.T) {
	t.Run("空列表", func(t *testing.T) {
		got := SerializeToolCallArguments([]llm.ToolCall{})
		if got != "[]" {
			t.Errorf("SerializeToolCallArguments(empty) = %q, 期望 %q", got, "[]")
		}
	})

	t.Run("单个调用", func(t *testing.T) {
		calls := []llm.ToolCall{
			{
				ID:       "call_args_1",
				Function: llm.FunctionCall{Name: "local_bash", Arguments: `{"command":"ls -la"}`},
			},
		}
		got := SerializeToolCallArguments(calls)
		if !strings.Contains(got, "local_bash") {
			t.Errorf("结果应包含 'local_bash', 实际: %q", got)
		}
		if !strings.Contains(got, "call_args_1") {
			t.Errorf("结果应包含 ID, 实际: %q", got)
		}
	})

	t.Run("多个调用", func(t *testing.T) {
		calls := []llm.ToolCall{
			{ID: "c1", Function: llm.FunctionCall{Name: "tool_a", Arguments: `{"x":1}`}},
			{ID: "c2", Function: llm.FunctionCall{Name: "tool_b", Arguments: `{"y":2}`}},
		}
		got := SerializeToolCallArguments(calls)
		if !strings.Contains(got, "tool_a") || !strings.Contains(got, "tool_b") {
			t.Errorf("结果应包含两个工具名, 实际: %q", got)
		}
	})
}

// ═══════════════════════════════════════════════════════════════
// V2 未覆盖函数测试
// ═══════════════════════════════════════════════════════════════

// TestBuilderV2_BuildSystemMessageWithMemory 测试带记忆注入的系统消息
func TestBuilderV2_BuildSystemMessageWithMemory(t *testing.T) {
	t.Run("未启用记忆_回退到BuildSystemMessage", func(t *testing.T) {
		b := NewBuilderV2() // enableMemory = false
		msg := b.BuildSystemMessageWithMemory("检查磁盘空间")

		if msg.Role != "system" {
			t.Errorf("Role = %q, 期望 %q", msg.Role, "system")
		}
		// 应与 BuildSystemMessage 结果一致
		normalMsg := b.BuildSystemMessage()
		if msg.Content != normalMsg.Content {
			t.Error("未启用记忆时, Content 应与 BuildSystemMessage() 一致")
		}
	})

	t.Run("nil注入器_回退到BuildSystemMessage", func(t *testing.T) {
		b := &BuilderV2{
			systemPrompt:     GetStaticSystemPrompt(),
			memoryInjector:   nil,
			enableMemory:     true, // 即使启用, injector 为 nil 也回退
		}
		msg := b.BuildSystemMessageWithMemory("检查内存")
		normalMsg := b.BuildSystemMessage()

		if msg.Content != normalMsg.Content {
			t.Error("injector 为 nil 时, Content 应与 BuildSystemMessage() 一致")
		}
	})

	t.Run("有记忆注入_包含记忆上下文", func(t *testing.T) {
		injector := NewMemoryInjector(&evolver.EnvironmentMemory{}, &evolver.ExperienceMemory{})
		b := NewBuilderV2WithMemory(injector)

		msg := b.BuildSystemMessageWithMemory("查看服务器状态")

		if msg.Role != "system" {
			t.Errorf("Role = %q, 期望 %q", msg.Role, "system")
		}
		if msg.Content == "" {
			t.Error("Content 不应为空")
		}
		// 应该包含基础 system prompt 的内容
		if !strings.Contains(msg.Content, "opsxcli") {
			t.Error("Content 应包含 agent 身份标识")
		}
	})
}

// TestBuilderV2_BuildToolResponseMessage 测试 V2 工具响应消息
func TestBuilderV2_BuildToolResponseMessage(t *testing.T) {
	b := NewBuilderV2()

	t.Run("成功_带摘要", func(t *testing.T) {
		result := &tools.Result{
			Success: true,
			Output:  "nginx is running",
			Summary: "服务运行正常",
		}
		msg := b.BuildToolResponseMessage("tc_v2_1", "ssh_execute", result)

		if msg.Role != "tool" {
			t.Errorf("Role = %q, 期望 %q", msg.Role, "tool")
		}
		if msg.ToolCallID != "tc_v2_1" {
			t.Errorf("ToolCallID = %q, 期望 %q", msg.ToolCallID, "tc_v2_1")
		}
		if msg.Name != "ssh_execute" {
			t.Errorf("Name = %q, 期望 %q", msg.Name, "ssh_execute")
		}
		if !strings.Contains(msg.Content, "成功: nginx is running") {
			t.Errorf("Content 应包含成功输出, 实际: %q", msg.Content)
		}
		if !strings.Contains(msg.Content, "摘要: 服务运行正常") {
			t.Errorf("Content 应包含摘要, 实际: %q", msg.Content)
		}
	})

	t.Run("成功_无摘要", func(t *testing.T) {
		result := &tools.Result{
			Success: true,
			Output:  "OK",
		}
		msg := b.BuildToolResponseMessage("tc_v2_2", "local_bash", result)

		if !strings.Contains(msg.Content, "成功: OK") {
			t.Errorf("Content = %q, 期望包含 '成功: OK'", msg.Content)
		}
		if strings.Contains(msg.Content, "摘要") {
			t.Error("Content 不应包含摘要")
		}
	})

	t.Run("失败", func(t *testing.T) {
		result := &tools.Result{
			Success: false,
			Error:   "command not found",
		}
		msg := b.BuildToolResponseMessage("tc_v2_3", "local_bash", result)

		if !strings.Contains(msg.Content, "失败: command not found") {
			t.Errorf("Content 应包含失败信息, 实际: %q", msg.Content)
		}
	})
}

// TestBuilderV2_BuildRetryObservationMessage 测试 V2 重试观察消息
func TestBuilderV2_BuildRetryObservationMessage(t *testing.T) {
	b := NewBuilderV2()

	t.Run("重试消息格式", func(t *testing.T) {
		msg := b.BuildRetryObservationMessage("ssh_execute", 3, "connection refused")

		if msg.Role != "user" {
			t.Errorf("Role = %q, 期望 %q", msg.Role, "user")
		}
		if !strings.Contains(msg.Content, "ssh_execute") {
			t.Errorf("Content 应包含工具名, 实际: %q", msg.Content)
		}
		if !strings.Contains(msg.Content, "第 3 次") {
			t.Errorf("Content 应包含重试次数, 实际: %q", msg.Content)
		}
		if !strings.Contains(msg.Content, "connection refused") {
			t.Errorf("Content 应包含错误信息, 实际: %q", msg.Content)
		}
	})

	t.Run("第一次重试", func(t *testing.T) {
		msg := b.BuildRetryObservationMessage("local_bash", 1, "timeout")
		if !strings.Contains(msg.Content, "第 1 次") {
			t.Errorf("Content 应包含 '第 1 次', 实际: %q", msg.Content)
		}
	})

	t.Run("包含策略建议", func(t *testing.T) {
		msg := b.BuildRetryObservationMessage("tool", 2, "err")
		if !strings.Contains(msg.Content, "重试") {
			t.Errorf("Content 应包含重试提示, 实际: %q", msg.Content)
		}
	})
}

// TestMin 测试 min 辅助函数
func TestMin(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{"a小于b", 1, 5, 1},
		{"b小于a", 10, 3, 3},
		{"相等", 7, 7, 7},
		{"负数_a更小", -5, 3, -5},
		{"负数_b更小", 5, -10, -10},
		{"零和正数", 0, 100, 0},
		{"零和负数", 0, -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := min(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
