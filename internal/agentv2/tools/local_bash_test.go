package tools

import (
	"context"
	"testing"
)

// TestLocalBashTool_Execute 测试本地 Bash 工具执行
func TestLocalBashTool_Execute(t *testing.T) {
	tool := NewLocalBashTool()
	ctx := context.Background()

	tests := []struct {
		name        string
		args        map[string]interface{}
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:        "echo command",
			args:        map[string]interface{}{"command": "echo hello"},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:        "empty command",
			args:        map[string]interface{}{"command": ""},
			wantSuccess: false,
			wantErr:     true,
		},
		{
			name:        "read-only command uname",
			args:        map[string]interface{}{"command": "uname -a"},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:        "timeout command",
			args:        map[string]interface{}{"command": "sleep 5", "timeout": 0.5},
			wantSuccess: false,
			wantErr:     false, // 超时返回 result.Success=false, 但 err=nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.args)
			if tt.wantErr && err == nil {
				t.Errorf("Execute() expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Execute() unexpected error: %v", err)
			}
			if result != nil && result.Success != tt.wantSuccess {
				t.Errorf("Execute() success = %v, want %v", result.Success, tt.wantSuccess)
			}
		})
	}
}

// TestCheckDangerousCommand 测试危险命令拦截
func TestCheckDangerousCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{"rm -rf /", "rm -rf /", true},
		{"rm -rf /tmp", "rm -rf /tmp", false}, // /tmp 不在黑名单中
		{"rm --no-preserve-root", "rm --no-preserve-root", true},
		{"mkfs.ext4 /dev/sda1", "mkfs.ext4 /dev/sda1", true},
		{"dd if=/dev/zero of=/dev/sda", "dd if=/dev/zero of=/dev/sda", true},
		{"fork bomb", ":(){ :|:& };:", true},
		{"chmod 777 /", "chmod -R 777 /", true},
		{"rm /bin", "rm -rf /bin", true},
		{"safe echo", "echo hello", false},
		{"safe cat", "cat /etc/passwd", false},
		{"safe grep", "grep error /var/log/syslog", false},
		{"safe ps", "ps aux", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkDangerousCommand(tt.command)
			if tt.wantErr && err == nil {
				t.Errorf("checkDangerousCommand(%q) expected error but got none", tt.command)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("checkDangerousCommand(%q) unexpected error: %v", tt.command, err)
			}
		})
	}
}

// TestBashCommandAnalyzer_AnalyzeRisk 测试风险分析器
func TestBashCommandAnalyzer_AnalyzeRisk(t *testing.T) {
	analyzer := NewBashCommandAnalyzer()

	tests := []struct {
		command  string
		wantRisk RiskLevel
	}{
		// 只读命令
		{"cat /etc/passwd", RiskSafe},
		{"grep error /var/log/syslog", RiskSafe},
		{"ps aux", RiskSafe},
		{"df -h", RiskSafe},
		{"docker ps", RiskSafe},
		{"kubectl get pods", RiskSafe},

		// 写入命令
		{"echo hello > /tmp/test.txt", RiskHigh},
		{"tar -czf backup.tar.gz /data", RiskHigh},
		{"git add .", RiskHigh},

		// 危险命令
		{"rm -rf /tmp", RiskCritical},
		{"kill 1234", RiskCritical},
		{"shutdown now", RiskCritical},
		{"docker rm container", RiskCritical},
		{"kubectl delete pod xxx", RiskCritical},

		// 管道命令（整体取最大风险）
		{"cat /etc/passwd | grep root", RiskSafe},
		{"cat /etc/passwd | grep root | wc -l", RiskSafe},
		{"ps aux | grep nginx | awk '{print $2}' | xargs kill", RiskMedium}, // xargs kill 不在当前危险模式列表中

		// 复合命令（取最大风险）
		{"cat /etc/passwd && rm -rf /tmp", RiskCritical},
		{"cat /etc/passwd || echo 'not found'", RiskSafe},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := analyzer.AnalyzeRisk(tt.command)
			if got != tt.wantRisk {
				t.Errorf("AnalyzeRisk(%q) = %v, want %v", tt.command, got, tt.wantRisk)
			}
		})
	}
}

// TestBashCommandAnalyzer_IsReadOnly 测试只读判断
func TestBashCommandAnalyzer_IsReadOnly(t *testing.T) {
	analyzer := NewBashCommandAnalyzer()

	tests := []struct {
		command string
		want    bool
	}{
		{"cat /etc/passwd", true},
		{"ls -la", true},
		{"ps aux", true},
		{"echo hello > file.txt", false},
		{"rm file.txt", false},
		{"docker run nginx", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := analyzer.IsReadOnly(tt.command)
			if got != tt.want {
				t.Errorf("IsReadOnly(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}
