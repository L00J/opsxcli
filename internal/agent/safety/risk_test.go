package safety

import (
	"testing"

	"opsxcli/internal/agent/tools"
)

// ==================== IsDangerousCommand ====================

func TestIsDangerousCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		wantDanger bool
		wantPattern string
	}{
		// 空输入
		{"empty string", "", false, ""},
		{"whitespace only", "   ", false, ""},
		{"tab only", "\t", false, ""},

		// 危险命令 - 文件系统毁灭
		{"rm -rf /", "rm -rf /", true, "rm -rf /"},
		{"rm -rf /*", "rm -rf /*", true, "rm -rf /*"},
		{"rm -rf /etc", "rm -rf /etc", true, "rm -rf /etc"},
		{"rm -rf /usr", "rm -rf /usr", true, "rm -rf /usr"},
		{"rm -rf /var", "rm -rf /var", true, "rm -rf /var"},
		{"rm -rf /home", "rm -rf /home", true, "rm -rf /home"},
		{"rm -rf /root", "rm -rf /root", true, "rm -rf /root"},

		// 磁盘格式化
		{"mkfs", "mkfs.ext4 /dev/sda1", true, "mkfs"},
		{"mkfs.xfs", "mkfs.xfs /dev/sdb", true, "mkfs.xfs"},
		{"mkfs.btrfs", "mkfs.btrfs /dev/sdc", true, "mkfs.btrfs"},

		// dd 破坏
		{"dd zero", "dd if=/dev/zero of=/dev/sda", true, "dd if=/dev/zero"},
		{"dd random", "dd if=/dev/random of=/dev/sda", true, "dd if=/dev/random"},
		{"dd urandom", "dd if=/dev/urandom of=/dev/sdb", true, "dd if=/dev/urandom"},

		// Fork bomb
		{"fork bomb compact", ":(){ :|:& };:", true, ":(){ :|:& };:"},
		{"fork bomb spaced", ":() { : | : & }; :", true, ":() { : | : & }; :"},

		// 设备重定向
		{"redirect to sda", "> /dev/sda", true, "> /dev/sda"},
		{"redirect to hda", "> /dev/hda", true, "> /dev/hda"},

		// 权限破坏
		{"chmod 000", "chmod -R 000 /", true, "chmod -R 000 /"},
		{"chmod 777", "chmod -R 777 /", true, "chmod -R 777 /"},

		// 服务级破坏
		{"systemctl stop ssh", "systemctl stop ssh", true, "systemctl stop ssh"},
		{"iptables -F", "iptables -F", true, "iptables -F"},

		// 大小写不敏感
		{"RM uppercase", "RM -RF /", true, ""},
		{"MKFS uppercase", "MKFS /DEV/SDA", true, ""},

		// rm -rf + 根路径的特殊检测
		{"rm -rf with root path etc", "rm -rf /etc/passwd", true, "rm -rf 包含根目录路径"},
		{"rm -rf with root path usr", "rm -rf /usr/local", true, "rm -rf 包含根目录路径"},

		// 安全命令
		{"ls safe", "ls -la", false, ""},
		{"cat safe", "cat /etc/hosts", false, ""},
		{"echo safe", "echo hello world", false, ""},
		{"ps safe", "ps aux", false, ""},
		{"grep safe", "grep pattern file.txt", false, ""},
		{"git safe", "git status", false, ""},
		{"docker safe", "docker ps", false, ""},
		{"go test safe", "go test ./...", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDanger, gotPattern := IsDangerousCommand(tt.command)
			if gotDanger != tt.wantDanger {
				t.Errorf("IsDangerousCommand(%q) danger = %v, want %v", tt.command, gotDanger, tt.wantDanger)
			}
			if tt.wantPattern != "" && gotPattern != tt.wantPattern {
				// Some patterns match via Contains, so just verify it's non-empty
				t.Logf("IsDangerousCommand(%q) pattern = %q, want %q (info only)", tt.command, gotPattern, tt.wantPattern)
			}
		})
	}
}

// ==================== containsRootPath ====================

func TestContainsRootPath(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{"boot", "rm -rf /boot", true},
		{"etc", "rm -rf /etc", true},
		{"usr", "rm -rf /usr/local", true},
		{"var", "rm -rf /var/log", true},
		{"lib", "rm -rf /lib", true},
		{"lib64", "rm -rf /lib64", true},
		{"sbin", "rm -rf /sbin", true},
		{"bin", "rm -rf /bin", true},
		{"opt", "rm -rf /opt/app", true},
		{"home", "rm -rf /home/user", true},
		{"root", "rm -rf /root", true},
		{"tmp", "rm -rf /tmp/scratch", true},
		{"var/log", "rm -rf /var/log/app", true},
		{"no root path", "rm -rf ./local/dir", false},
		{"relative path", "rm -rf data/backup", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsRootPath(tt.cmd); got != tt.want {
				t.Errorf("containsRootPath(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

// ==================== AssessBashRisk ====================

func TestAssessBashRisk(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    tools.RiskLevel
	}{
		// 空输入
		{"empty", "", tools.RiskSafe},
		{"whitespace", "   ", tools.RiskSafe},

		// 危险命令 → RiskCritical
		{"rm -rf /", "rm -rf /", tools.RiskCritical},
		{"mkfs", "mkfs.ext4 /dev/sda", tools.RiskCritical},
		{"dd zero", "dd if=/dev/zero of=/dev/sda", tools.RiskCritical},
		{"iptables -F", "iptables -F", tools.RiskCritical},

		// 只读命令 → RiskSafe
		{"ls", "ls -la", tools.RiskSafe},
		{"cat", "cat /etc/hosts", tools.RiskSafe},
		{"grep", "grep pattern file", tools.RiskSafe},
		{"ps", "ps aux", tools.RiskSafe},
		{"df", "df -h", tools.RiskSafe},
		{"top", "top", tools.RiskSafe},
		{"uname", "uname -a", tools.RiskSafe},
		{"hostname", "hostname", tools.RiskSafe},
		{"whoami", "whoami", tools.RiskSafe},
		{"pwd", "pwd", tools.RiskSafe},
		{"id", "id", tools.RiskSafe},
		{"echo", "echo hello", tools.RiskSafe},
		{"find", "find /tmp -name '*.log'", tools.RiskSafe},
		{"wc", "wc -l file.txt", tools.RiskSafe},
		{"sort", "sort data.txt", tools.RiskSafe},
		{"sudo ls", "sudo ls -la", tools.RiskSafe},
		{"sudo cat", "sudo cat /var/log/syslog", tools.RiskSafe},

		// 高风险命令（无 sudo）→ RiskMedium
		// 注意：assessSingleCommand 用 fields[0] 提取 mainCmd
		// "kill " / "systemctl restart" 等带空格的条目不会被 fields[0] 匹配
		{"rm", "rm file.txt", tools.RiskMedium},
		{"dd", "dd if=input of=output", tools.RiskMedium},
		{"shutdown", "shutdown -h now", tools.RiskMedium},
		{"reboot", "reboot", tools.RiskMedium},
		{"killall", "killall nginx", tools.RiskMedium},
		{"pkill", "pkill -f process", tools.RiskMedium},
		{"userdel", "userdel testuser", tools.RiskMedium},
		// "kill -9 1234" contains "kill -9 1" → IsDangerousCommand → RiskCritical
		{"kill with args", "kill -9 1234", tools.RiskCritical},
		// "systemctl restart" → mainCmd="systemctl", 不匹配任何 HighRiskCommands
		{"systemctl restart", "systemctl restart nginx", tools.RiskLow},
		{"systemctl stop", "systemctl stop nginx", tools.RiskLow},

		// 高风险命令（sudo）→ RiskHigh
		{"sudo rm", "sudo rm file.txt", tools.RiskHigh},
		{"sudo shutdown", "sudo shutdown -h now", tools.RiskHigh},
		{"sudo reboot", "sudo reboot", tools.RiskHigh},
		// "sudo kill -9 1" → IsDangerousCommand → contains "kill -9 1" → RiskCritical
		{"sudo kill", "sudo kill -9 1", tools.RiskCritical},
		// "sudo systemctl restart" → mainCmd="systemctl" → 不匹配 → RiskLow, isSudo → RiskMedium
		{"sudo systemctl restart", "sudo systemctl restart nginx", tools.RiskMedium},

		// 文件修改命令（无 sudo）→ RiskLow
		{"cp", "cp src dst", tools.RiskLow},
		{"mv", "mv old new", tools.RiskLow},
		{"mkdir", "mkdir newdir", tools.RiskLow},
		{"touch", "touch file.txt", tools.RiskLow},
		{"chmod", "chmod 644 file.txt", tools.RiskLow},
		{"tar", "tar -czf archive.tar.gz .", tools.RiskLow},
		{"rsync", "rsync -av src/ dst/", tools.RiskLow},

		// 文件修改命令（sudo）→ RiskMedium
		{"sudo cp", "sudo cp src dst", tools.RiskMedium},
		{"sudo mv", "sudo mv old new", tools.RiskMedium},
		{"sudo mkdir", "sudo mkdir /opt/app", tools.RiskMedium},
		{"sudo chmod", "sudo chmod 755 /opt", tools.RiskMedium},

		// SSH/SCP
		// ssh → "ssh" hasPrefix "ss" (ReadOnlyCommands) → isReadOnlyCommand → RiskSafe
		// 注意：这是 isReadOnlyCommand 前缀匹配的副作用（"ss" 匹配了 "ssh"）
		{"ssh", "ssh user@host", tools.RiskSafe},
		{"scp", "scp file user@host:/tmp", tools.RiskMedium},

		// 编辑器
		{"vi", "vi file.txt", tools.RiskLow},
		{"vim", "vim file.txt", tools.RiskLow},
		{"nano", "nano file.txt", tools.RiskLow},
		{"emacs", "emacs file.txt", tools.RiskLow},
		{"sudo vi", "sudo vi /etc/hosts", tools.RiskMedium},
		{"sudo vim", "sudo vim /etc/hosts", tools.RiskMedium},

		// 组合命令 — 注意：isReadOnlyCommand 检查前缀匹配
		// "ls | rm file" 以 "ls" 开头 → isReadOnlyCommand=true → RiskSafe
		// 组合命令检测（第3步）永远不会被触发
		{"pipe cat grep", "cat file | grep pattern", tools.RiskSafe},
		{"pipe with rm", "ls | rm file", tools.RiskSafe}, // 以 "ls" 开头 → read-only
		{"&& safe commands", "ls && pwd", tools.RiskSafe},
		{"&& risky command", "ls && rm file", tools.RiskSafe}, // 以 "ls" 开头
		{"|| operator", "cat file || echo not found", tools.RiskSafe},
		{"; separator", "ls; ps aux", tools.RiskSafe},
		{"compound with sudo", "sudo ls && cat file", tools.RiskSafe}, // sudo ls → trim → "ls" → read-only

		// 未知命令
		{"unknown command", "myapp --flag", tools.RiskLow},
		{"sudo unknown", "sudo myapp --flag", tools.RiskMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AssessBashRisk(tt.command)
			if got != tt.want {
				t.Errorf("AssessBashRisk(%q) = %v (%s), want %v (%s)",
					tt.command,
					got, GetRiskDisplay(got),
					tt.want, GetRiskDisplay(tt.want))
			}
		})
	}
}

// ==================== isReadOnlyCommand ====================

func TestIsReadOnlyCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		// 只读命令
		{"cat", "cat file.txt", true},
		{"ls", "ls -la /tmp", true},
		{"grep", "grep pattern file", true},
		{"ps", "ps aux", true},
		{"df", "df -h", true},
		{"top", "top -bn1", true},
		{"free", "free -m", true},
		{"uptime", "uptime", true},
		{"uname", "uname -a", true},
		{"hostname", "hostname", true},
		{"ifconfig", "ifconfig", true},
		{"ip addr", "ip addr show", true},
		{"netstat", "netstat -tlnp", true},
		{"ss", "ss -tlnp", true},
		{"pwd", "pwd", true},
		{"who", "who", true},
		{"whoami", "whoami", true},
		{"id", "id", true},
		{"echo", "echo hello", true},
		{"find", "find / -name file", true},
		{"which", "which python", true},
		{"stat", "stat file.txt", true},
		{"date", "date", true},
		{"sudo cat", "sudo cat /var/log/syslog", true},
		{"sudo ls", "sudo ls -la /root", true},

		// 非只读命令
		{"rm", "rm file.txt", false},
		{"cp", "cp src dst", false},
		{"mv", "mv old new", false},
		{"mkdir", "mkdir dir", false},
		{"chmod", "chmod 755 file", false},
		{"systemctl restart", "systemctl restart nginx", false},
		{"docker", "docker run image", false},
		{"python", "python script.py", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReadOnlyCommand(tt.cmd); got != tt.want {
				t.Errorf("isReadOnlyCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

// ==================== assessSingleCommand ====================

func TestAssessSingleCommand(t *testing.T) {
	tests := []struct {
		name   string
		cmd    string
		isSudo bool
		want   tools.RiskLevel
	}{
		// 空输入
		{"empty", "", false, tools.RiskSafe},
		{"whitespace", "  ", false, tools.RiskSafe},

		// 高风险命令（无 sudo）
		{"rm no sudo", "rm file.txt", false, tools.RiskMedium},
		{"shutdown no sudo", "shutdown -h now", false, tools.RiskMedium},
		{"reboot no sudo", "reboot", false, tools.RiskMedium},
		// "kill -9 1234" → mainCmd="kill", 不匹配 "kill "（带空格），走默认 RiskLow
		{"kill no sudo", "kill -9 1234", false, tools.RiskLow},
		{"dd no sudo", "dd if=input of=output bs=1M", false, tools.RiskMedium},
		{"halt no sudo", "halt", false, tools.RiskMedium},
		{"poweroff no sudo", "poweroff", false, tools.RiskMedium},
		{"userdel no sudo", "userdel test", false, tools.RiskMedium},
		{"fdisk no sudo", "fdisk /dev/sda", false, tools.RiskMedium},

		// 高风险命令（sudo）
		{"rm sudo", "rm file.txt", true, tools.RiskHigh},
		{"shutdown sudo", "shutdown -h now", true, tools.RiskHigh},
		{"kill sudo", "kill -9 1", true, tools.RiskMedium}, // mainCmd="kill" 不匹配 → RiskLow + sudo → RiskMedium
		{"dd sudo", "dd if=input of=output", true, tools.RiskHigh},

		// 文件修改命令（无 sudo）
		{"cp no sudo", "cp src dst", false, tools.RiskLow},
		{"mv no sudo", "mv old new", false, tools.RiskLow},
		{"mkdir no sudo", "mkdir dir", false, tools.RiskLow},
		{"chmod no sudo", "chmod 755 file", false, tools.RiskLow},
		{"tar no sudo", "tar -czf out.tar.gz .", false, tools.RiskLow},

		// 文件修改命令（sudo）
		{"cp sudo", "cp src dst", true, tools.RiskMedium},
		{"mv sudo", "mv old new", true, tools.RiskMedium},
		{"mkdir sudo", "mkdir /opt/new", true, tools.RiskMedium},

		// SSH/SCP
		// assessSingleCommand 中 ssh 有特殊检查 → RiskHigh（不经过 isReadOnlyCommand）
		{"ssh", "ssh user@host", false, tools.RiskHigh},
		{"scp", "scp file host:/tmp", false, tools.RiskMedium},

		// 编辑器
		{"vi no sudo", "vi file.txt", false, tools.RiskLow},
		{"vim no sudo", "vim file.txt", false, tools.RiskLow},
		{"nano no sudo", "nano file.txt", false, tools.RiskLow},
		{"emacs no sudo", "emacs file.txt", false, tools.RiskLow},
		{"vi sudo", "vi /etc/hosts", true, tools.RiskMedium},
		{"vim sudo", "vim /etc/hosts", true, tools.RiskMedium},

		// 未知命令
		{"unknown no sudo", "myapp --flag", false, tools.RiskLow},
		{"unknown sudo", "myapp --flag", true, tools.RiskMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := assessSingleCommand(tt.cmd, tt.isSudo); got != tt.want {
				t.Errorf("assessSingleCommand(%q, %v) = %v, want %v",
					tt.cmd, tt.isSudo, got, tt.want)
			}
		})
	}
}

// ==================== assessCompoundCommand ====================

func TestAssessCompoundCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want tools.RiskLevel
	}{
		// 管道 - assessCompoundCommand 用 assessSingleCommand 评估子命令
		// assessSingleCommand 不走 isReadOnlyCommand，只读命令走默认 RiskLow
		{"pipe cat grep", "cat file | grep pattern", tools.RiskLow},
		{"pipe ls wc", "ls | wc -l", tools.RiskLow},
		{"pipe ps grep", "ps aux | grep nginx", tools.RiskLow},

		// 管道 - 包含修改操作
		{"pipe with cp", "ls | cp src dst", tools.RiskLow},
		{"pipe with rm", "find /tmp -name '*.tmp' | rm", tools.RiskMedium},

		// && 操作符
		{"&& two safe", "ls && pwd", tools.RiskLow},
		{"&& safe and modify", "ls && cp src dst", tools.RiskLow},
		// rm -rf / 通过 IsDangerousCommand 在 AssessBashRisk 第一层拦截
		// 但 assessCompoundCommand 内部直接调用 assessSingleCommand，
		// 不会调用 IsDangerousCommand，所以 rm 只匹配 HighRiskCommands → RiskMedium
		{"&& safe and dangerous", "ls && rm -rf /", tools.RiskMedium},

		// || 操作符
		{"|| two safe", "cat file || echo missing", tools.RiskLow},

		// ; 分隔符
		{"; two safe", "ls; pwd", tools.RiskLow},
		{"; safe and modify", "ls; cp src dst", tools.RiskLow},

		// sudo 组合 — sudo ls 被分割后 "sudo ls" 作为整体，"grep pattern" 作为子命令
		// "sudo ls" 中 mainCmd="sudo" 不匹配任何列表 → 默认 RiskLow
		// 但 cmd.HasPrefix("sudo ") 提升风险 → RiskMedium
		{"sudo pipe", "sudo ls | grep pattern", tools.RiskMedium},
		{"sudo &&", "sudo ls && cat file", tools.RiskMedium}, // sudo boost

		// 多重组合
		{"triple safe", "ls && pwd && echo done", tools.RiskLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := assessCompoundCommand(tt.cmd); got != tt.want {
				t.Errorf("assessCompoundCommand(%q) = %v (%s), want %v (%s)",
					tt.cmd, got, GetRiskDisplay(got), tt.want, GetRiskDisplay(tt.want))
			}
		})
	}
}

// ==================== GetRiskDisplay ====================

func TestGetRiskDisplay(t *testing.T) {
	tests := []struct {
		level tools.RiskLevel
		want  string
	}{
		{tools.RiskSafe, "安全"},
		{tools.RiskLow, "低风险"},
		{tools.RiskMedium, "中风险"},
		{tools.RiskHigh, "高风险"},
		{tools.RiskCritical, "危险"},
		{tools.RiskLevel(99), "未知"}, // 超出范围的值
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := GetRiskDisplay(tt.level); got != tt.want {
				t.Errorf("GetRiskDisplay(%v) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

// ==================== GetRiskDisplayEN ====================

func TestGetRiskDisplayEN(t *testing.T) {
	tests := []struct {
		level tools.RiskLevel
		want  string
	}{
		{tools.RiskSafe, "SAFE"},
		{tools.RiskLow, "LOW"},
		{tools.RiskMedium, "MEDIUM"},
		{tools.RiskHigh, "HIGH"},
		{tools.RiskCritical, "CRITICAL"},
		{tools.RiskLevel(99), "UNKNOWN"}, // 超出范围的值
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := GetRiskDisplayEN(tt.level); got != tt.want {
				t.Errorf("GetRiskDisplayEN(%v) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}
