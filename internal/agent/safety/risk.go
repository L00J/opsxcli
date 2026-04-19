// risk.go - 风险评估模块
// 提供危险命令检测、风险动态评估和等级显示功能
package safety

import (
	"strings"

	"opsxcli/internal/agent/tools"
)

// DangerousCommands 危险命令黑名单
// 包含可能导致系统损坏、数据丢失或服务中断的命令模式
var DangerousCommands = []string{
	// 文件系统毁灭性操作
	"rm -rf /",
	"rm -rf /*",
	"rm -rf /bin",
	"rm -rf /boot",
	"rm -rf /etc",
	"rm -rf /home",
	"rm -rf /lib",
	"rm -rf /lib64",
	"rm -rf /proc",
	"rm -rf /root",
	"rm -rf /sbin",
	"rm -rf /sys",
	"rm -rf /usr",
	"rm -rf /var",

	// 磁盘格式化
	"mkfs",
	"mkfs.ext",
	"mkfs.xfs",
	"mkfs.btrfs",
	"mkfs.vfat",

	// 磁盘覆写
	"dd if=/dev/zero",
	"dd if=/dev/random",
	"dd if=/dev/urandom",

	// Fork bomb
	":(){ :|:& };:",
	":() { : | : & }; :",

	// 设备重定向破坏
	"> /dev/sda",
	"> /dev/sdb",
	"> /dev/sdc",
	"> /dev/hda",
	"> /dev/hdb",

	// 极端目录操作
	"mv / /dev/null",
	"mv /* /dev/null",

	// 权限破坏
	"chmod -R 000 /",
	"chmod -R 777 /",

	// 服务级破坏
	"systemctl stop ssh",
	"systemctl disable ssh",
	"kill -9 1",

	// 网络配置破坏（需谨慎）
	"iptables -F",
	"iptables --flush",
}

// ReadOnlyCommands 只读命令列表，这些命令通常被认为是安全的
var ReadOnlyCommands = []string{
	"cat",
	"grep",
	"egrep",
	"fgrep",
	"awk",
	"sed",
	"head",
	"tail",
	"less",
	"more",
	"sort",
	"uniq",
	"wc",
	"ps",
	"top",
	"df",
	"du",
	"free",
	"uptime",
	"uname",
	"hostname",
	"ifconfig",
	"ip addr",
	"ip link",
	"ip route",
	"netstat",
	"ss",
	"ls",
	"ll",
	"pwd",
	"who",
	"w",
	"last",
	"vmstat",
	"iostat",
	"mpstat",
	"sar",
	"lsof",
	"find",
	"which",
	"whereis",
	"file",
	"stat",
	"date",
	"echo",
	"id",
	"whoami",
	"groups",
	"getent",
	"dmesg",
	"journalctl --no-pager -n",
	"systemctl status",
	"systemctl list-units",
	"systemctl list-unit-files",
	"lsblk",
	"blkid",
	"fdisk -l",
	"parted -l",
	"mount",
	"cat /proc",
	"cat /sys",
}

// ModificationCommands 文件修改类命令，中等风险
var ModificationCommands = []string{
	"cp",
	"mv",
	"mkdir",
	"rmdir",
	"touch",
	"chmod",
	"chown",
	"chgrp",
	"ln",
	"tar",
	"gzip",
	"gunzip",
	"zip",
	"unzip",
	"rsync",
}

// HighRiskCommands 高风险命令
var HighRiskCommands = []string{
	"rm",
	"dd",
	"shutdown",
	"reboot",
	"halt",
	"poweroff",
	"init ",
	"systemctl restart",
	"systemctl stop",
	"systemctl start",
	"kill ",
	"killall",
	"pkill",
	"userdel",
	"groupdel",
	"fdisk",
	"parted",
	"mkfs",
}

// IsDangerousCommand 检查命令是否包含危险模式
// 返回 (是否危险, 匹配的危险模式)
func IsDangerousCommand(command string) (bool, string) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return false, ""
	}

	// 将命令转为小写进行匹配
	lowerCmd := strings.ToLower(trimmed)

	for _, dangerous := range DangerousCommands {
		// 精确匹配或前缀匹配
		if strings.Contains(lowerCmd, strings.ToLower(dangerous)) {
			return true, dangerous
		}
	}

	// 特殊模式检测：rm -rf 后跟绝对路径
	if strings.HasPrefix(lowerCmd, "rm") && strings.Contains(lowerCmd, "-rf") {
		// 检查是否包含根目录路径模式
		if containsRootPath(lowerCmd) {
			return true, "rm -rf 包含根目录路径"
		}
	}

	return false, ""
}

// containsRootPath 检查命令是否包含根目录路径
func containsRootPath(cmd string) bool {
	// 匹配 /xxx 格式的绝对路径（不包括 / 后跟相对路径的情况）
	patterns := []string{
		" /boot", " /etc", " /usr", " /var", " /lib",
		" /lib64", " /sbin", " /bin", " /opt", " /home",
		" /root", " /tmp", " /var/log",
	}
	for _, p := range patterns {
		if strings.Contains(cmd, p) {
			return true
		}
	}
	return false
}

// AssessBashRisk 动态评估 bash 命令的风险等级
// 基于命令内容和模式进行风险分类
func AssessBashRisk(command string) tools.RiskLevel {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return tools.RiskSafe
	}

	lowerCmd := strings.ToLower(trimmed)

	// 第一步：检查是否在黑名单中
	if isDangerous, _ := IsDangerousCommand(command); isDangerous {
		return tools.RiskCritical
	}

	// 第二步：检查是否是纯只读命令
	if isReadOnlyCommand(lowerCmd) {
		return tools.RiskSafe
	}

	// 第三步：检查是否包含管道或组合命令
	if strings.Contains(lowerCmd, "&&") || strings.Contains(lowerCmd, "||") || strings.Contains(lowerCmd, "|") {
		return assessCompoundCommand(lowerCmd)
	}

	// 第四步：检查是否包含 sudo（提升风险等级）
	isSudo := strings.HasPrefix(lowerCmd, "sudo ")
	cmdWithoutSudo := strings.TrimPrefix(lowerCmd, "sudo ")

	// 第五步：基于命令前缀判断风险
	return assessSingleCommand(cmdWithoutSudo, isSudo)
}

// isReadOnlyCommand 检查是否为纯只读命令
func isReadOnlyCommand(cmd string) bool {
	// 去掉 sudo 前缀
	cmd = strings.TrimPrefix(cmd, "sudo ")
	cmd = strings.TrimSpace(cmd)

	for _, roCmd := range ReadOnlyCommands {
		roCmdLower := strings.ToLower(roCmd)
		if strings.HasPrefix(cmd, roCmdLower) {
			return true
		}
	}
	return false
}

// assessCompoundCommand 评估组合命令（含 &&, ||, |）的风险
// 取所有子命令中的最高风险等级
func assessCompoundCommand(cmd string) tools.RiskLevel {
	maxRisk := tools.RiskSafe

	// 分割组合命令（简单分割，不处理引号和转义）
	separators := []string{"&&", "||", "|", ";"}
	parts := []string{cmd}

	for _, sep := range separators {
		var newParts []string
		for _, part := range parts {
			subParts := strings.Split(part, sep)
			for _, sp := range subParts {
				sp = strings.TrimSpace(sp)
				if sp != "" {
					newParts = append(newParts, sp)
				}
			}
		}
		parts = newParts
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 递归评估每个子命令（去掉 sudo 前缀）
		risk := assessSingleCommand(strings.TrimPrefix(part, "sudo "), false)
		if risk > maxRisk {
			maxRisk = risk
		}
	}

	// 如果原始命令有 sudo，提升一个风险等级
	if strings.HasPrefix(cmd, "sudo ") && maxRisk < tools.RiskCritical {
		maxRisk++
	}

	return maxRisk
}

// assessSingleCommand 评估单个命令的风险等级
func assessSingleCommand(cmd string, isSudo bool) tools.RiskLevel {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return tools.RiskSafe
	}

	// 提取命令的第一个词（主命令）
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return tools.RiskSafe
	}
	mainCmd := strings.ToLower(fields[0])

	// 检查是否为高风险命令
	for _, hrCmd := range HighRiskCommands {
		if mainCmd == strings.ToLower(hrCmd) || strings.HasPrefix(mainCmd, strings.ToLower(hrCmd)) {
			if isSudo {
				return tools.RiskHigh
			}
			return tools.RiskMedium
		}
	}

	// 检查是否为文件修改命令
	for _, modCmd := range ModificationCommands {
		if mainCmd == strings.ToLower(modCmd) {
			if isSudo {
				return tools.RiskMedium
			}
			return tools.RiskLow
		}
	}

	// 检查远程 SSH 命令（通常高风险）
	if mainCmd == "ssh" {
		return tools.RiskHigh
	}

	// 检查 SCP 文件传输
	if mainCmd == "scp" {
		return tools.RiskMedium
	}

	// 检查编辑文件命令
	if mainCmd == "vi" || mainCmd == "vim" || mainCmd == "nano" || mainCmd == "emacs" {
		if isSudo {
			return tools.RiskMedium
		}
		return tools.RiskLow
	}

	// 无法识别的命令，如果有 sudo 则默认为 medium
	if isSudo {
		return tools.RiskMedium
	}

	// 默认 low risk
	return tools.RiskLow
}

// GetRiskDisplay 获取风险等级的显示文本（中文）
func GetRiskDisplay(level tools.RiskLevel) string {
	switch level {
	case tools.RiskSafe:
		return "安全"
	case tools.RiskLow:
		return "低风险"
	case tools.RiskMedium:
		return "中风险"
	case tools.RiskHigh:
		return "高风险"
	case tools.RiskCritical:
		return "危险"
	default:
		return "未知"
	}
}

// GetRiskDisplayEN 获取风险等级的英文显示文本
func GetRiskDisplayEN(level tools.RiskLevel) string {
	switch level {
	case tools.RiskSafe:
		return "SAFE"
	case tools.RiskLow:
		return "LOW"
	case tools.RiskMedium:
		return "MEDIUM"
	case tools.RiskHigh:
		return "HIGH"
	case tools.RiskCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}
