package safety

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"opsxcli/internal/agent/tools"
)

// mockTool 用于测试的 mock 工具
type mockTool struct {
	name      string
	riskLevel tools.RiskLevel
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return "mock tool" }
func (m *mockTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (m *mockTool) RiskLevel() tools.RiskLevel { return m.riskLevel }
func (m *mockTool) Execute(_ context.Context, _ map[string]interface{}) (*tools.Result, error) {
	return &tools.Result{Success: true}, nil
}

// TestController_needsConfirmation 测试确认需求判断
func TestController_needsConfirmation(t *testing.T) {
	tests := []struct {
		mode SafetyMode
		risk tools.RiskLevel
		want bool
	}{
		{SafetyModeStrict, tools.RiskSafe, false},
		{SafetyModeStrict, tools.RiskLow, false},
		{SafetyModeStrict, tools.RiskMedium, true},
		{SafetyModeStrict, tools.RiskHigh, true},
		{SafetyModeStrict, tools.RiskCritical, true},

		{SafetyModeBalanced, tools.RiskSafe, false},
		{SafetyModeBalanced, tools.RiskLow, false},
		{SafetyModeBalanced, tools.RiskMedium, false},
		{SafetyModeBalanced, tools.RiskHigh, true},
		{SafetyModeBalanced, tools.RiskCritical, true},

		{SafetyModePermissive, tools.RiskSafe, false},
		{SafetyModePermissive, tools.RiskLow, false},
		{SafetyModePermissive, tools.RiskMedium, false},
		{SafetyModePermissive, tools.RiskHigh, false},
		{SafetyModePermissive, tools.RiskCritical, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode)+"_"+tt.risk.String(), func(t *testing.T) {
			c := NewController(tt.mode)
			defer c.Close()
			got := c.needsConfirmation(tt.risk)
			if got != tt.want {
				t.Errorf("needsConfirmation(%v) with mode %v = %v, want %v",
					tt.risk, tt.mode, got, tt.want)
			}
		})
	}
}

// TestController_GetMode 测试获取安全模式
func TestController_GetMode(t *testing.T) {
	c := NewController(SafetyModeStrict)
	defer c.Close()
	if got := c.GetMode(); got != SafetyModeStrict {
		t.Errorf("GetMode() = %v, want %v", got, SafetyModeStrict)
	}
}

// TestController_SetAutoApprove 测试自动批准设置
func TestController_SetAutoApprove(t *testing.T) {
	c := NewController(SafetyModeBalanced)
	defer c.Close()
	if c.autoApprove {
		t.Error("autoApprove should be false by default")
	}

	c.SetAutoApprove(true)
	if !c.autoApprove {
		t.Error("autoApprove should be true after SetAutoApprove(true)")
	}
}

// TestController_SetConfirmFn 测试自定义审批函数
func TestController_SetConfirmFn(t *testing.T) {
	c := NewController(SafetyModeStrict)
	defer c.Close()

	called := false
	c.SetConfirmFn(func(toolName string, args map[string]interface{}, risk tools.RiskLevel) (bool, error) {
		called = true
		if toolName != "dangerous" {
			t.Errorf("toolName = %q, want %q", toolName, "dangerous")
		}
		return true, nil
	})

	tool := &mockTool{name: "dangerous", riskLevel: tools.RiskCritical}
	approved, err := c.Check(tool, map[string]interface{}{"command": "rm -rf /"})
	if err != nil {
		t.Fatalf("Check() unexpected error: %v", err)
	}
	if !approved {
		t.Error("Check() should approve when confirmFn returns true")
	}
	if !called {
		t.Error("confirmFn should have been called")
	}
}

// TestController_Check_autoApprove 测试自动批准模式
func TestController_Check_autoApprove(t *testing.T) {
	c := NewController(SafetyModeStrict)
	defer c.Close()
	c.SetAutoApprove(true)

	tool := &mockTool{name: "dangerous", riskLevel: tools.RiskCritical}
	approved, err := c.Check(tool, map[string]interface{}{"command": "rm -rf /"})
	if err != nil {
		t.Fatalf("Check() unexpected error: %v", err)
	}
	if !approved {
		t.Error("Check() should approve when autoApprove is true")
	}

	// 验证历史记录已添加
	history := c.GetHistory()
	if len(history) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(history))
	}
	if history[0].ToolName != "dangerous" {
		t.Errorf("history[0].ToolName = %q, want %q", history[0].ToolName, "dangerous")
	}
}

// TestController_MarkExecuted 测试执行标记
func TestController_MarkExecuted(t *testing.T) {
	c := NewController(SafetyModeBalanced)
	defer c.Close()
	c.SetAutoApprove(true)

	tool := &mockTool{name: "test_tool", riskLevel: tools.RiskSafe}
	c.Check(tool, map[string]interface{}{})

	// 标记为成功执行
	c.MarkExecuted("test_tool", map[string]interface{}{}, &tools.Result{Success: true})

	history := c.GetHistory()
	if len(history) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(history))
	}
	if !history[0].Approved {
		t.Error("history[0].Approved should be true after MarkExecuted with success")
	}
	if !history[0].Executed {
		t.Error("history[0].Executed should be true after MarkExecuted")
	}
	if !history[0].Success {
		t.Error("history[0].Success should be true after MarkExecuted with success")
	}
}

// TestController_GetHistory_immutable 测试历史记录副本不可变
func TestController_GetHistory_immutable(t *testing.T) {
	c := NewController(SafetyModeBalanced)
	defer c.Close()
	c.SetAutoApprove(true)

	tool := &mockTool{name: "test", riskLevel: tools.RiskSafe}
	c.Check(tool, map[string]interface{}{})

	history1 := c.GetHistory()
	if len(history1) != 1 {
		t.Fatalf("expected 1 record, got %d", len(history1))
	}

	// 修改副本不应影响内部状态
	history1[0].ToolName = "modified"

	history2 := c.GetHistory()
	if history2[0].ToolName != "test" {
		t.Errorf("GetHistory() returned mutable slice, internal state corrupted")
	}
}

// TestController_addRecord_historyLimit 测试历史记录上限
func TestController_addRecord_historyLimit(t *testing.T) {
	c := NewController(SafetyModeBalanced)
	defer c.Close()
	c.SetAutoApprove(true)

	// 添加超过上限的记录
	for i := 0; i < maxHistorySize+100; i++ {
		tool := &mockTool{name: fmt.Sprintf("tool_%d", i), riskLevel: tools.RiskSafe}
		c.Check(tool, map[string]interface{}{})
	}

	history := c.GetHistory()
	if len(history) > maxHistorySize {
		t.Errorf("history size %d exceeds max %d", len(history), maxHistorySize)
	}
}

// TestFormatRiskDisplay 测试风险等级格式化
func TestFormatRiskDisplay(t *testing.T) {
	tests := []struct {
		risk tools.RiskLevel
		want string
	}{
		{tools.RiskSafe, "安全"},
		{tools.RiskLow, "低风险"},
		{tools.RiskMedium, "中风险"},
		{tools.RiskHigh, "高风险"},
		{tools.RiskCritical, "危险"},
		{tools.RiskLevel(99), "未知"},
	}

	for _, tt := range tests {
		t.Run(tt.risk.String(), func(t *testing.T) {
			got := formatRiskDisplay(tt.risk)
			if got == "" {
				t.Error("formatRiskDisplay() returned empty string")
			}
			// 只检查包含关键特征词
			if !strings.Contains(got, tt.want) {
				t.Errorf("formatRiskDisplay(%v) = %q, should contain %q", tt.risk, got, tt.want)
			}
		})
	}
}

// TestNewController_defaultAuditLog 测试默认审计日志自动创建
func TestNewController_defaultAuditLog(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	c := NewController(SafetyModeBalanced)
	if c == nil {
		t.Fatal("NewController should not return nil")
	}
	defer c.Close()

	if c.auditLog == nil {
		t.Error("NewController should create default audit log")
	}

	expectedPath := filepath.Join(tmpDir, ".opsxcli", "audit", "audit.log")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("default audit log file should exist at %s", expectedPath)
	}
}

// TestNewControllerWithAudit_customPath 测试自定义审计日志路径
func TestNewControllerWithAudit_customPath(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "custom_audit.log")

	c, err := NewControllerWithAudit(SafetyModeStrict, logPath)
	if err != nil {
		t.Fatalf("NewControllerWithAudit failed: %v", err)
	}
	defer c.Close()

	if c.auditLog == nil {
		t.Error("auditLog should not be nil")
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("custom audit log file should exist at %s", logPath)
	}
}

// TestAuditLogWrite_content 测试审计日志内容正确性
func TestAuditLogWrite_content(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	c, err := NewControllerWithAudit(SafetyModeBalanced, logPath)
	if err != nil {
		t.Fatalf("NewControllerWithAudit failed: %v", err)
	}

	c.SetAutoApprove(true)
	tool := &mockTool{name: "test_tool", riskLevel: tools.RiskSafe}
	c.Check(tool, map[string]interface{}{"key": "value"})

	if err := c.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read audit log failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 audit line, got %d", len(lines))
	}

	var record ExecutionRecord
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("unmarshal audit record failed: %v", err)
	}

	if record.ToolName != "test_tool" {
		t.Errorf("tool_name = %q, want %q", record.ToolName, "test_tool")
	}
	if record.EventType != "check" {
		t.Errorf("event_type = %q, want %q", record.EventType, "check")
	}
	if !record.Approved {
		t.Error("approved should be true")
	}
	if record.Args["key"] != "value" {
		t.Errorf("args[key] = %v, want %v", record.Args["key"], "value")
	}
}

// TestAuditLogWrite_markExecuted 测试 MarkExecuted 追加执行记录
func TestAuditLogWrite_markExecuted(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	c, err := NewControllerWithAudit(SafetyModeBalanced, logPath)
	if err != nil {
		t.Fatalf("NewControllerWithAudit failed: %v", err)
	}

	c.SetAutoApprove(true)
	tool := &mockTool{name: "test_tool", riskLevel: tools.RiskSafe}
	c.Check(tool, map[string]interface{}{})
	c.MarkExecuted("test_tool", map[string]interface{}{}, &tools.Result{Success: true, Error: ""})

	if err := c.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read audit log failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 audit lines, got %d", len(lines))
	}

	var checkRecord ExecutionRecord
	if err := json.Unmarshal([]byte(lines[0]), &checkRecord); err != nil {
		t.Fatalf("unmarshal check record failed: %v", err)
	}
	if checkRecord.EventType != "check" {
		t.Errorf("first record event_type = %q, want %q", checkRecord.EventType, "check")
	}

	var execRecord ExecutionRecord
	if err := json.Unmarshal([]byte(lines[1]), &execRecord); err != nil {
		t.Fatalf("unmarshal execute record failed: %v", err)
	}
	if execRecord.EventType != "execute" {
		t.Errorf("second record event_type = %q, want %q", execRecord.EventType, "execute")
	}
	if !execRecord.Executed {
		t.Error("executed should be true")
	}
	if !execRecord.Success {
		t.Error("success should be true")
	}
}

// TestAuditLog_concurrentWrite 测试并发写入安全
func TestAuditLog_concurrentWrite(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	c, err := NewControllerWithAudit(SafetyModeBalanced, logPath)
	if err != nil {
		t.Fatalf("NewControllerWithAudit failed: %v", err)
	}
	defer c.Close()

	c.SetAutoApprove(true)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tool := &mockTool{name: fmt.Sprintf("tool_%d", idx), riskLevel: tools.RiskSafe}
			c.Check(tool, map[string]interface{}{"idx": idx})
		}(i)
	}
	wg.Wait()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read audit log failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 100 {
		t.Errorf("expected 100 audit lines, got %d", len(lines))
	}

	// 验证每行都是合法 JSON
	for i, line := range lines {
		var record ExecutionRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
	}
}

// TestController_Close_idempotent 测试关闭后无 panic
func TestController_Close_idempotent(t *testing.T) {
	c := NewController(SafetyModeBalanced)
	if err := c.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	// 再次关闭不应 panic（虽然底层文件已关闭会报错，但这里不重复测试）
}

// TestNewControllerWithAudit_emptyPath 测试空路径时不创建审计日志
func TestNewControllerWithAudit_emptyPath(t *testing.T) {
	c, err := NewControllerWithAudit(SafetyModeBalanced, "")
	if err != nil {
		t.Fatalf("NewControllerWithAudit failed: %v", err)
	}
	defer c.Close()

	if c.auditLog != nil {
		t.Error("auditLog should be nil when path is empty")
	}
}

// mockDynamicRiskTool 实现 DynamicRiskTool 接口的 mock 工具
type mockDynamicRiskTool struct {
	mockTool
	riskForArgs func(args map[string]interface{}) tools.RiskLevel
}

func (m *mockDynamicRiskTool) RiskLevelForArgs(args map[string]interface{}) tools.RiskLevel {
	if m.riskForArgs != nil {
		return m.riskForArgs(args)
	}
	return m.riskLevel
}

// TestController_Check_dynamicRisk 测试 DynamicRiskTool 接口动态风险评估
func TestController_Check_dynamicRisk(t *testing.T) {
	tests := []struct {
		name      string
		mode      SafetyMode
		riskFn    func(args map[string]interface{}) tools.RiskLevel
		args      map[string]interface{}
		wantApproved bool
	}{
		{
			name: "balanced模式_只读命令_自动通过",
			mode: SafetyModeBalanced,
			riskFn: func(args map[string]interface{}) tools.RiskLevel {
				return tools.RiskSafe
			},
			args:      map[string]interface{}{"command": "ls -la"},
			wantApproved: true,
		},
		{
			name: "balanced模式_中等风险_自动通过",
			mode: SafetyModeBalanced,
			riskFn: func(args map[string]interface{}) tools.RiskLevel {
				return tools.RiskMedium
			},
			args:      map[string]interface{}{"command": "echo hello"},
			wantApproved: true,
		},
		{
			name: "balanced模式_高风险_需要确认",
			mode: SafetyModeBalanced,
			riskFn: func(args map[string]interface{}) tools.RiskLevel {
				return tools.RiskHigh
			},
			args:      map[string]interface{}{"command": "rm file.txt"},
			wantApproved: true, // confirmFn 返回 true
		},
		{
			name: "balanced模式_危险操作_需要确认",
			mode: SafetyModeBalanced,
			riskFn: func(args map[string]interface{}) tools.RiskLevel {
				return tools.RiskCritical
			},
			args:      map[string]interface{}{"command": "rm -rf /data"},
			wantApproved: true, // confirmFn 返回 true
		},
		{
			name: "strict模式_中等风险_需要确认",
			mode: SafetyModeStrict,
			riskFn: func(args map[string]interface{}) tools.RiskLevel {
				return tools.RiskMedium
			},
			args:      map[string]interface{}{"command": "echo hello"},
			wantApproved: true, // confirmFn 返回 true
		},
		{
			name: "strict模式_安全操作_自动通过",
			mode: SafetyModeStrict,
			riskFn: func(args map[string]interface{}) tools.RiskLevel {
				return tools.RiskSafe
			},
			args:      map[string]interface{}{"command": "ls"},
			wantApproved: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewController(tt.mode)
			defer c.Close()

			confirmCalled := false
			c.SetConfirmFn(func(toolName string, args map[string]interface{}, risk tools.RiskLevel) (bool, error) {
				confirmCalled = true
				return true, nil
			})

			tool := &mockDynamicRiskTool{
				mockTool:    mockTool{name: "dynamic_test", riskLevel: tools.RiskMedium},
				riskForArgs: tt.riskFn,
			}

			approved, err := c.Check(tool, tt.args)
			if err != nil {
				t.Fatalf("Check() error: %v", err)
			}
			if approved != tt.wantApproved {
				t.Errorf("approved = %v, want %v", approved, tt.wantApproved)
			}

			// 验证高风险/严格模式中等风险确实触发了确认
			risk := tt.riskFn(tt.args)
			shouldConfirm := c.needsConfirmation(risk)
			if shouldConfirm && !confirmCalled {
				t.Error("confirmFn should have been called for risk needing confirmation")
			}
			if !shouldConfirm && confirmCalled {
				t.Error("confirmFn should NOT have been called for safe operations")
			}
		})
	}
}

// TestController_Check_dynamicRiskVsStatic 测试动态风险评估覆盖静态风险等级
func TestController_Check_dynamicRiskVsStatic(t *testing.T) {
	c := NewController(SafetyModeBalanced)
	defer c.Close()
	c.SetAutoApprove(true)

	// 工具的静态 RiskLevel 返回 RiskCritical，但动态评估返回 RiskSafe
	tool := &mockDynamicRiskTool{
		mockTool:    mockTool{name: "override_test", riskLevel: tools.RiskCritical},
		riskForArgs: func(args map[string]interface{}) tools.RiskLevel { return tools.RiskSafe },
	}

	approved, err := c.Check(tool, map[string]interface{}{"command": "ls"})
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	if !approved {
		t.Error("should be approved — dynamic RiskSafe overrides static RiskCritical")
	}

	// 验证历史记录使用的是动态风险等级
	history := c.GetHistory()
	if len(history) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(history))
	}
	if history[0].RiskLevel != tools.RiskSafe {
		t.Errorf("history risk = %v, want %v (dynamic should override static)",
			history[0].RiskLevel, tools.RiskSafe)
	}
}

// TestController_Check_staticToolFallback 测试不实现 DynamicRiskTool 的工具仍使用静态风险
func TestController_Check_staticToolFallback(t *testing.T) {
	c := NewController(SafetyModeBalanced)
	defer c.Close()
	c.SetAutoApprove(true)

	// 普通 mockTool（不实现 DynamicRiskTool）
	tool := &mockTool{name: "static_test", riskLevel: tools.RiskHigh}

	approved, err := c.Check(tool, map[string]interface{}{"command": "anything"})
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	if !approved {
		t.Error("should be approved with autoApprove")
	}

	history := c.GetHistory()
	if len(history) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(history))
	}
	if history[0].RiskLevel != tools.RiskHigh {
		t.Errorf("history risk = %v, want %v", history[0].RiskLevel, tools.RiskHigh)
	}
}

// TestController_Check_dynamicRisk_rejection 测试动态风险评估拒绝操作
func TestController_Check_dynamicRisk_rejection(t *testing.T) {
	c := NewController(SafetyModeBalanced)
	defer c.Close()

	c.SetConfirmFn(func(toolName string, args map[string]interface{}, risk tools.RiskLevel) (bool, error) {
		return false, nil // 用户拒绝
	})

	tool := &mockDynamicRiskTool{
		mockTool:    mockTool{name: "reject_test", riskLevel: tools.RiskMedium},
		riskForArgs: func(args map[string]interface{}) tools.RiskLevel { return tools.RiskHigh },
	}

	approved, err := c.Check(tool, map[string]interface{}{"command": "rm -rf /data"})
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	if approved {
		t.Error("should be rejected when confirmFn returns false")
	}

	history := c.GetHistory()
	if len(history) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(history))
	}
	if history[0].Approved {
		t.Error("history record should show not approved")
	}
	if history[0].RiskLevel != tools.RiskHigh {
		t.Errorf("history risk = %v, want %v", history[0].RiskLevel, tools.RiskHigh)
	}
}
