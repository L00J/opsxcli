package tools

import "testing"

// TestParseHostAddress 测试主机地址解析
func TestParseHostAddress(t *testing.T) {
	tests := []struct {
		input    string
		wantUser string
		wantHost string
		wantPort int
		wantErr  bool
	}{
		{"user@host:22", "user", "host", 22, false},
		{"host", "root", "host", 22, false},
		{"user@host", "user", "host", 22, false},
		{"host:2222", "root", "host", 2222, false},
		{"user@[::1]:2222", "user", "::1", 2222, false},
		{"user@192.168.1.1:22", "user", "192.168.1.1", 22, false},
		{"192.168.1.1:2222", "root", "192.168.1.1", 2222, false},
		{"user@", "", "", 0, true},   // 空主机名
		{"", "", "", 0, true},         // 完全为空
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			user, host, port, err := parseHostAddress(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseHostAddress(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseHostAddress(%q) unexpected error: %v", tt.input, err)
			}
			if user != tt.wantUser {
				t.Errorf("parseHostAddress(%q) user = %q, want %q", tt.input, user, tt.wantUser)
			}
			if host != tt.wantHost {
				t.Errorf("parseHostAddress(%q) host = %q, want %q", tt.input, host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Errorf("parseHostAddress(%q) port = %d, want %d", tt.input, port, tt.wantPort)
			}
		})
	}
}

// TestSSHExecuteTool_Name 测试工具名称
func TestSSHExecuteTool_Name(t *testing.T) {
	tool := NewSSHExecuteTool()
	if got := tool.Name(); got != "ssh_execute" {
		t.Errorf("Name() = %q, want %q", got, "ssh_execute")
	}
}

// TestSSHExecuteTool_RiskLevel 测试风险等级
func TestSSHExecuteTool_RiskLevel(t *testing.T) {
	tool := NewSSHExecuteTool()
	if got := tool.RiskLevel(); got != RiskHigh {
		t.Errorf("RiskLevel() = %v, want %v", got, RiskHigh)
	}
}
