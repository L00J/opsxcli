package safety

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"opsxcli/internal/agentv2/tools"
)

// mockTool 用于测试的 mock 工具
type mockTool struct {
	name       string
	riskLevel  tools.RiskLevel
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
		mode     SafetyMode
		risk     tools.RiskLevel
		want     bool
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
	if got := c.GetMode(); got != SafetyModeStrict {
		t.Errorf("GetMode() = %v, want %v", got, SafetyModeStrict)
	}
}

// TestController_SetAutoApprove 测试自动批准设置
func TestController_SetAutoApprove(t *testing.T) {
	c := NewController(SafetyModeBalanced)
	if c.autoApprove {
		t.Error("autoApprove should be false by default")
	}

	c.SetAutoApprove(true)
	if !c.autoApprove {
		t.Error("autoApprove should be true after SetAutoApprove(true)")
	}
}

// TestController_Check_autoApprove 测试自动批准模式
func TestController_Check_autoApprove(t *testing.T) {
	c := NewController(SafetyModeStrict)
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
}

// TestController_GetHistory_immutable 测试历史记录副本不可变
func TestController_GetHistory_immutable(t *testing.T) {
	c := NewController(SafetyModeBalanced)
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
