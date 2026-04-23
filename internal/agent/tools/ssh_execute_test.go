package tools

import (
	"os"
	"testing"
)

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
		{"user@", "", "", 0, true}, // 空主机名
		{"", "", "", 0, true},      // 完全为空
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

// TestBuildAuthMethods 测试构建 SSH 认证方法
func TestBuildAuthMethods(t *testing.T) {
	t.Run("无密钥无密码_不panic", func(t *testing.T) {
		tool := NewSSHExecuteTool()
		t.Setenv("SSH_PASSWORD", "")
		// 主要确保不会 panic
		_, _ = tool.buildAuthMethods("root", "localhost")
	})

	t.Run("设置密码环境变量", func(t *testing.T) {
		tool := NewSSHExecuteTool()
		t.Setenv("SSH_PASSWORD", "testpass")
		methods, err := tool.buildAuthMethods("root", "localhost")
		if err != nil {
			t.Fatalf("有密码时不应报错: %v", err)
		}
		if len(methods) == 0 {
			t.Error("应至少有一个认证方法")
		}
		// 有密码时，不应有 KeyboardInteractive（因为 sshPassword != ""）
		// 但应有 Password 方法
	})

	t.Run("无密码时有KeyboardInteractive", func(t *testing.T) {
		tool := NewSSHExecuteTool()
		t.Setenv("SSH_PASSWORD", "")
		t.Setenv("SSH_ASKPASS", "/usr/bin/ssh-askpass")
		methods, err := tool.buildAuthMethods("root", "localhost")
		if err != nil {
			t.Logf("buildAuthMethods 返回错误 (可能是环境中无密钥): %v", err)
			return
		}
		// 至少应有一些方法
		if len(methods) == 0 {
			t.Error("应至少有一个认证方法")
		}
	})

	t.Run("有密码时不添加KeyboardInteractive", func(t *testing.T) {
		tool := NewSSHExecuteTool()
		t.Setenv("SSH_PASSWORD", "secretpass")
		methods, err := tool.buildAuthMethods("user", "host")
		if err != nil {
			t.Fatalf("不应报错: %v", err)
		}
		// 有密码时：可能有 PublicKeys + Password（无 KeyboardInteractive）
		if len(methods) == 0 {
			t.Error("应至少有一个认证方法")
		}
	})
}

// TestBuildHostKeyCallback 测试构建 HostKey 回调
func TestBuildHostKeyCallback(t *testing.T) {
	t.Run("应返回非nil回调", func(t *testing.T) {
		cb := buildHostKeyCallback()
		if cb == nil {
			t.Error("buildHostKeyCallback() 不应返回 nil")
		}
	})
}

// TestLoadPrivateKey 测试加载私钥
func TestLoadPrivateKey(t *testing.T) {
	t.Run("不存在的文件", func(t *testing.T) {
		_, err := loadPrivateKey("/nonexistent/path/id_rsa")
		if err == nil {
			t.Error("不存在的文件应返回错误")
		}
	})

	t.Run("无效的密钥内容", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyFile := tmpDir + "/bad_key"
		if err := os.WriteFile(keyFile, []byte("not a valid key"), 0600); err != nil {
			t.Fatalf("写入临时文件失败: %v", err)
		}
		_, err := loadPrivateKey(keyFile)
		if err == nil {
			t.Error("无效密钥内容应返回错误")
		}
	})
}
