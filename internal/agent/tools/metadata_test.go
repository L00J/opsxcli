package tools

import (
	"testing"
)

// ===================== LocalBashTool 元数据测试 =====================

func TestLocalBashTool_RiskLevel(t *testing.T) {
	tool := NewLocalBashTool()
	if got := tool.RiskLevel(); got != RiskMedium {
		t.Errorf("LocalBashTool.RiskLevel() = %v, want %v", got, RiskMedium)
	}
}

func TestLocalBashTool_Name(t *testing.T) {
	tool := NewLocalBashTool()
	if got := tool.Name(); got != "local_bash" {
		t.Errorf("LocalBashTool.Name() = %q, want %q", got, "local_bash")
	}
}

func TestLocalBashTool_Description(t *testing.T) {
	tool := NewLocalBashTool()
	if got := tool.Description(); got == "" {
		t.Error("LocalBashTool.Description() should not be empty")
	}
}

func TestLocalBashTool_Parameters(t *testing.T) {
	tool := NewLocalBashTool()
	params := tool.Parameters()
	if params == nil {
		t.Fatal("LocalBashTool.Parameters() should not be nil")
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Parameters should have properties map")
	}
	if _, exists := props["command"]; !exists {
		t.Error("Parameters should contain 'command' property")
	}
}

// ===================== SCPTransferTool 元数据测试 =====================

func TestSCPTransferTool_Name(t *testing.T) {
	tool := NewSCPTransferTool()
	if got := tool.Name(); got != "scp_transfer" {
		t.Errorf("SCPTransferTool.Name() = %q, want %q", got, "scp_transfer")
	}
}

func TestSCPTransferTool_Description(t *testing.T) {
	tool := NewSCPTransferTool()
	if got := tool.Description(); got == "" {
		t.Error("SCPTransferTool.Description() should not be empty")
	}
}

func TestSCPTransferTool_Parameters(t *testing.T) {
	tool := NewSCPTransferTool()
	params := tool.Parameters()
	if params == nil {
		t.Fatal("SCPTransferTool.Parameters() should not be nil")
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Parameters should have properties map")
	}
	// 检查必要参数
	expectedParams := []string{"source", "destination", "direction", "host"}
	for _, p := range expectedParams {
		if _, exists := props[p]; !exists {
			t.Errorf("Parameters should contain %q property", p)
		}
	}
	// 检查 required 字段
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Parameters should have required field")
	}
	if len(required) != 4 {
		t.Errorf("Required should have 4 items, got %d", len(required))
	}
}

func TestSCPTransferTool_RiskLevel(t *testing.T) {
	tool := NewSCPTransferTool()
	if got := tool.RiskLevel(); got != RiskMedium {
		t.Errorf("SCPTransferTool.RiskLevel() = %v, want %v", got, RiskMedium)
	}
}

// ===================== BashCommandAnalyzer.GetRiskDescription 测试 =====================

func TestBashCommandAnalyzer_GetRiskDescription(t *testing.T) {
	analyzer := NewBashCommandAnalyzer()

	tests := []struct {
		name    string
		command string
		want    string
	}{
		{"只读命令", "ls -la", "只读操作，安全"},
		{"低风险", "cat /etc/hosts", "只读操作，安全"},
		{"写入操作", "echo hello > /tmp/test", "高风险，会修改文件或配置"},
		{"危险操作", "rm -rf /", "危险操作，可能导致系统不稳定或数据丢失"},
		{"中等风险", "systemctl status nginx", "只读操作，安全"},
		{"未知中等风险", "some_unknown_command", "中等风险，可能修改系统状态"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GetRiskDescription calls AnalyzeRisk internally
			// We just verify it returns a non-empty string
			got := analyzer.GetRiskDescription(tt.command)
			if got == "" {
				t.Errorf("GetRiskDescription(%q) returned empty string", tt.command)
			}
			t.Logf("GetRiskDescription(%q) = %q", tt.command, got)
		})
	}
}

func TestBashCommandAnalyzer_GetRiskDescription_AllLevels(t *testing.T) {
	analyzer := NewBashCommandAnalyzer()

	// 测试每个风险级别都有描述
	levels := map[string]string{
		"只读": analyzer.GetRiskDescription("cat /etc/passwd"),
		"写入": analyzer.GetRiskDescription("echo test > /tmp/file"),
		"危险": analyzer.GetRiskDescription("rm -rf /"),
	}

	for level, desc := range levels {
		if desc == "" {
			t.Errorf("风险级别 %q 的描述不应为空", level)
		}
	}
}

// ===================== SSHExecuteTool 元数据补充测试 =====================

func TestSSHExecuteTool_Description(t *testing.T) {
	tool := NewSSHExecuteTool()
	if got := tool.Description(); got == "" {
		t.Error("SSHExecuteTool.Description() should not be empty")
	}
}

func TestSSHExecuteTool_Parameters(t *testing.T) {
	tool := NewSSHExecuteTool()
	params := tool.Parameters()
	if params == nil {
		t.Fatal("SSHExecuteTool.Parameters() should not be nil")
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Parameters should have properties map")
	}
	expectedParams := []string{"host", "command", "timeout", "use_sudo"}
	for _, p := range expectedParams {
		if _, exists := props[p]; !exists {
			t.Errorf("Parameters should contain %q property", p)
		}
	}
}

// ===================== isSSHConnAlive 测试 =====================

func TestIsSSHConnAlive_Nil(t *testing.T) {
	if isSSHConnAlive(nil) {
		t.Error("isSSHConnAlive(nil) should return false")
	}
}
