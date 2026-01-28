package tools

import (
	"regexp"
	"strings"
)

// BashCommandAnalyzer bash命令分析器
type BashCommandAnalyzer struct {
	// 只读命令模式
	readOnlyPatterns []*regexp.Regexp

	// 写入命令模式
	writePatterns []*regexp.Regexp

	// 危险命令模式
	dangerousPatterns []*regexp.Regexp
}

// NewBashCommandAnalyzer 创建bash命令分析器
func NewBashCommandAnalyzer() *BashCommandAnalyzer {
	return &BashCommandAnalyzer{
		readOnlyPatterns: []*regexp.Regexp{
			// 查看类命令
			regexp.MustCompile(`^(cat|less|more|head|tail|grep|awk|sed)\s+`),
			regexp.MustCompile(`^ls\s*`),
			regexp.MustCompile(`^find\s+.*-print`),
			regexp.MustCompile(`^ps\s*`),
			regexp.MustCompile(`^top\s*`),
			regexp.MustCompile(`^free\s*`),
			regexp.MustCompile(`^df\s*`),
			regexp.MustCompile(`^du\s+`),
			regexp.MustCompile(`^uname\s*`),
			regexp.MustCompile(`^hostname\s*$`),
			regexp.MustCompile(`^whoami\s*$`),
			regexp.MustCompile(`^id\s*`),
			regexp.MustCompile(`^date\s*`),
			regexp.MustCompile(`^uptime\s*$`),
			regexp.MustCompile(`^w\s*$`),
			regexp.MustCompile(`^who\s*$`),
			regexp.MustCompile(`^pwd\s*$`),

			// 网络查看
			regexp.MustCompile(`^(ping|traceroute|nslookup|dig|host)\s+`),
			regexp.MustCompile(`^netstat\s*`),
			regexp.MustCompile(`^ss\s*`),
			regexp.MustCompile(`^ip\s+addr`),
			regexp.MustCompile(`^ip\s+route\s+show`), // 只查看路由
			regexp.MustCompile(`^ifconfig\s*$`),
			regexp.MustCompile(`^telnet\s+`),          // Telnet 连接测试
			regexp.MustCompile(`^nc\s+-[zv]+\s+`),     // netcat 端口扫描
			regexp.MustCompile(`^nmap\s+`),            // nmap 端口扫描

			// 文件系统查看
			regexp.MustCompile(`^stat\s+`),
			regexp.MustCompile(`^file\s+`),
			regexp.MustCompile(`^wc\s+`),
			regexp.MustCompile(`^od\s+`),
			regexp.MustCompile(`^xxd\s+`),
			regexp.MustCompile(`^tree\s*`),            // 树形目录

			// Java相关查看
			regexp.MustCompile(`^java\s+-version`),
			regexp.MustCompile(`^javac\s+-version`),
			regexp.MustCompile(`^which\s+`),
			regexp.MustCompile(`^whereis\s+`),

			// 环境变量查看
			regexp.MustCompile(`^echo\s+[\"']?\$`),
			regexp.MustCompile(`^echo\s+.*`),  // echo 本身是安全的
			regexp.MustCompile(`^printenv`),
			regexp.MustCompile(`^env\s*`),

			// 命令存在性检查
			regexp.MustCompile(`^command\s+-v\s+`),
			regexp.MustCompile(`^type\s+`),
			regexp.MustCompile(`^hash\s+`),

			// 文本处理工具
			regexp.MustCompile(`^(jq|yq)\s+`),         // JSON/YAML 查询
			regexp.MustCompile(`^column\s+`),          // 格式化输出
			regexp.MustCompile(`^nl\s+`),              // 添加行号
			regexp.MustCompile(`^tr\s+`),              // 字符转换
			regexp.MustCompile(`^sort\s+`),            // 排序
			regexp.MustCompile(`^uniq\s+`),            // 去重

			// 系统诊断工具
			regexp.MustCompile(`^(lsof|strace|ltrace)\s+`), // 系统调用跟踪
			regexp.MustCompile(`^(htop|iotop|vmstat|iostat)\s*`), // 系统监控
			regexp.MustCompile(`^tcpdump\s+`),         // 抓包工具

			// 容器工具 - 只读操作
			regexp.MustCompile(`^docker\s+(ps|images|logs|inspect|stats|version|info)`),
			regexp.MustCompile(`^kubectl\s+(get|describe|logs|top|explain|version|api-resources|api-versions)`),

			// Git - 只读操作
			regexp.MustCompile(`^git\s+(status|log|diff|show|branch(\s+-[vla])?)`),

			// 压缩工具 - 查看内容
			regexp.MustCompile(`^(tar|unzip|zipinfo)\s+.*-[lt]`), // tar -t, unzip -l

			// 数据库 - 只读查询 (基本匹配,后续可优化为 SQL 分析)
			regexp.MustCompile(`(?i)^(mysql|psql).*\s+(SELECT|SHOW|EXPLAIN|DESC|DESCRIBE)\s+`),
			regexp.MustCompile(`^redis-cli\s+(GET|KEYS|INFO|MONITOR|TTL|TYPE|SCAN|EXISTS|LLEN|SCARD|ZCARD|HLEN)`),

			// 包管理器 - 查看已安装
			regexp.MustCompile(`^(apt|yum|dnf)\s+list\s+(installed|upgradable)`),
			regexp.MustCompile(`^(pip|pip3)\s+list`),
			regexp.MustCompile(`^npm\s+list`),

			// 编辑器只读模式
			regexp.MustCompile(`^(vim|vi)\s+-R\s+`),   // vim 只读模式
			regexp.MustCompile(`^view\s+`),            // view = vim -R

			// 组合命令(只包含只读操作)
			regexp.MustCompile(`^.*(cat|grep|awk|sed|head|tail|wc|sort|uniq).*\|.*(cat|grep|awk|sed|head|tail|wc|sort|uniq)`),
		},

		writePatterns: []*regexp.Regexp{
			// 只匹配真正的写入重定向 (> 或 >>), 但排除错误重定向 (2>, &>已在cleanRedirects处理)
			regexp.MustCompile(`[^0-9]>\s*[^&/]`),    // 匹配 > 但不是 2> 且不是 >/dev/null
			regexp.MustCompile(`[^0-9]>>\s*[^&/]`),   // 匹配 >> 但不是 2>>

			// 文本编辑器 (会修改文件,需要高风险确认)
			regexp.MustCompile(`^(vi|vim|nano|emacs|gedit)\s+`), // 编辑器 (只读模式在 readOnlyPatterns 已匹配)

			// 压缩/解压工具
			regexp.MustCompile(`^tar\s+.*-[xc]`),      // tar 解压或创建
			regexp.MustCompile(`^(gzip|gunzip|bzip2|bunzip2|xz|unxz)\s+`), // 压缩(会删除原文件)
			regexp.MustCompile(`^unzip\s+[^-]`),       // unzip 解压 (不以 - 开头的参数)
			regexp.MustCompile(`^zip\s+-r`),           // 创建压缩包

			// 网络下载
			regexp.MustCompile(`^(wget|curl).*-[oO]`), // 下载到文件
			regexp.MustCompile(`^scp\s+`),             // 文件传输

			// 容器工具 - 修改操作
			regexp.MustCompile(`^docker\s+(run|create|start|stop|restart|pause|unpause|exec|build|commit|tag|push|pull)`),
			regexp.MustCompile(`^kubectl\s+(apply|create|patch|replace|scale|expose|rollout|set|edit|annotate|label)`),

			// Git - 修改操作
			regexp.MustCompile(`^git\s+(add|commit|push|pull|fetch|merge|rebase|cherry-pick|stash)`),
			regexp.MustCompile(`^git\s+checkout`),     // 切换分支

			// 数据库 - 写入操作 (基本匹配)
			regexp.MustCompile(`(?i)^(mysql|psql).*\s+(INSERT|UPDATE|CREATE|ALTER)\s+`),
			regexp.MustCompile(`^redis-cli\s+(SET|SETEX|SETNX|MSET|HSET|LPUSH|RPUSH|SADD|ZADD|INCR|DECR|APPEND)`),

			// Python 包管理器 (高风险,会修改用户环境,但不是危险操作)
			regexp.MustCompile(`^(pip|pip3)\s+(install|uninstall)`),
		},

		dangerousPatterns: []*regexp.Regexp{
			// 文件删除
			regexp.MustCompile(`^rm\s+`),

			// 磁盘操作
			regexp.MustCompile(`^dd\s+`),
			regexp.MustCompile(`^mkfs`),
			regexp.MustCompile(`^fdisk`),
			regexp.MustCompile(`^parted`),

			// 进程控制
			regexp.MustCompile(`^kill\s+`),
			regexp.MustCompile(`^killall\s+`),

			// 系统控制
			regexp.MustCompile(`^shutdown`),
			regexp.MustCompile(`^reboot`),
			regexp.MustCompile(`^halt`),
			regexp.MustCompile(`^init\s+`),
			regexp.MustCompile(`^systemctl\s+(stop|restart|reload|disable)`),
			regexp.MustCompile(`^service\s+.*\s+(stop|restart)`),

			// 容器 - 删除操作
			regexp.MustCompile(`^docker\s+(rm|rmi|system\s+prune|volume\s+rm|network\s+rm)`),
			regexp.MustCompile(`^kubectl\s+(delete|drain)`),

			// Git - 危险操作
			regexp.MustCompile(`^git\s+reset\s+--hard`),
			regexp.MustCompile(`^git\s+clean\s+-[fd]`),
			regexp.MustCompile(`^git\s+push\s+.*--force`),

			// 数据库 - 删除操作
			regexp.MustCompile(`(?i)^(mysql|psql).*\s+(DROP|DELETE|TRUNCATE)\s+`),
			regexp.MustCompile(`^redis-cli\s+(DEL|FLUSHALL|FLUSHDB|CONFIG\s+SET)`),

			// 网络配置
			regexp.MustCompile(`^route\s+(add|del)`),
			regexp.MustCompile(`^ip\s+route\s+(add|del)`),
			regexp.MustCompile(`^iptables\s+-[ADI]`),

			// 包管理器 - 只保留系统级包管理器(更危险)
			regexp.MustCompile(`^(apt|yum|dnf)\s+(install|remove|purge|autoremove)`),
			regexp.MustCompile(`^npm\s+(install|uninstall)\s+-g`), // 全局安装/卸载
		},
	}
}

// AnalyzeRisk 分析命令风险等级
func (a *BashCommandAnalyzer) AnalyzeRisk(command string) RiskLevel {
	// 去除首尾空格
	command = strings.TrimSpace(command)

	// 清理安全的重定向模式 (2>&1, 2>/dev/null 等)
	// 这些不应该被算作写入操作
	cleanCommand := a.cleanRedirects(command)

	// 处理复合命令 (使用 || 或 && 连接的命令)
	// 例如: "java -version 2>&1 || echo 'not installed'"
	if strings.Contains(cleanCommand, "||") || strings.Contains(cleanCommand, "&&") {
		return a.analyzeCompoundCommand(cleanCommand)
	}

	// 处理管道命令 (使用 | 连接的命令)
	// 例如: "du -sh /* | sort -hr | head -20"
	if strings.Contains(cleanCommand, "|") && !strings.Contains(cleanCommand, "||") {
		return a.analyzePipelineCommand(cleanCommand)
	}

	// 检查是否是危险命令
	for _, pattern := range a.dangerousPatterns {
		if pattern.MatchString(cleanCommand) {
			return RiskCritical
		}
	}

	// 检查是否包含写入操作
	for _, pattern := range a.writePatterns {
		if pattern.MatchString(cleanCommand) {
			return RiskHigh
		}
	}

	// 检查是否是只读命令
	for _, pattern := range a.readOnlyPatterns {
		if pattern.MatchString(cleanCommand) {
			return RiskSafe
		}
	}

	// 默认为中等风险
	return RiskMedium
}

// analyzeCompoundCommand 分析复合命令 (使用 || 或 && 连接)
func (a *BashCommandAnalyzer) analyzeCompoundCommand(command string) RiskLevel {
	// 拆分命令: 先按 || 拆分,再按 && 拆分
	var parts []string

	// 按 || 拆分
	orParts := strings.Split(command, "||")
	for _, orPart := range orParts {
		// 再按 && 拆分
		andParts := strings.Split(orPart, "&&")
		parts = append(parts, andParts...)
	}

	// 分析每个子命令的风险
	maxRisk := RiskSafe

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// 移除常见的重定向(2>&1, >/dev/null等)用于分析
		part = a.cleanRedirects(part)

		// 递归分析子命令(避免再次进入复合命令分析)
		risk := a.analyzeSingleCommand(part)

		// 取最高风险等级
		if risk > maxRisk {
			maxRisk = risk
		}
	}

	return maxRisk
}

// cleanRedirects 清理命令中的重定向用于风险分析
func (a *BashCommandAnalyzer) cleanRedirects(command string) string {
	// 移除常见的安全重定向模式
	command = regexp.MustCompile(`\s+2>&1`).ReplaceAllString(command, "")
	command = regexp.MustCompile(`\s+>/dev/null`).ReplaceAllString(command, "")
	command = regexp.MustCompile(`\s+2>/dev/null`).ReplaceAllString(command, "")
	command = regexp.MustCompile(`\s+&>/dev/null`).ReplaceAllString(command, "")

	return strings.TrimSpace(command)
}

// analyzeSingleCommand 分析单个命令(不处理复合命令)
func (a *BashCommandAnalyzer) analyzeSingleCommand(command string) RiskLevel {
	// 检查是否是危险命令
	for _, pattern := range a.dangerousPatterns {
		if pattern.MatchString(command) {
			return RiskCritical
		}
	}

	// 检查是否包含写入操作
	for _, pattern := range a.writePatterns {
		if pattern.MatchString(command) {
			return RiskHigh
		}
	}

	// 检查是否是只读命令
	for _, pattern := range a.readOnlyPatterns {
		if pattern.MatchString(command) {
			return RiskSafe
		}
	}

	// 默认为中等风险
	return RiskMedium
}

// IsReadOnly 判断命令是否只读
func (a *BashCommandAnalyzer) IsReadOnly(command string) bool {
	return a.AnalyzeRisk(command) == RiskSafe
}

// GetRiskDescription 获取风险描述
func (a *BashCommandAnalyzer) GetRiskDescription(command string) string {
	risk := a.AnalyzeRisk(command)

	switch risk {
	case RiskSafe:
		return "只读操作,安全"
	case RiskLow:
		return "低风险操作"
	case RiskMedium:
		return "中等风险,可能修改系统状态"
	case RiskHigh:
		return "高风险,会修改文件或配置"
	case RiskCritical:
		return "危险操作,可能导致系统不稳定或数据丢失"
	default:
		return "未知风险"
	}
}

// analyzePipelineCommand 分析管道命令 (使用 | 连接)
func (a *BashCommandAnalyzer) analyzePipelineCommand(command string) RiskLevel {
	// 按管道符号拆分命令
	parts := strings.Split(command, "|")

	// 分析每个子命令的风险
	maxRisk := RiskSafe

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// 移除常见的重定向用于分析
		part = a.cleanRedirects(part)

		// 分析子命令
		risk := a.analyzeSingleCommand(part)

		// 取最高风险等级
		if risk > maxRisk {
			maxRisk = risk
		}
	}

	return maxRisk
}
