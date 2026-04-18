package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// outputTruncateLimit 输出截断限制（字符数）
const outputTruncateLimit = 10000

// LocalBashTool 本地 Bash 执行工具
type LocalBashTool struct {
	analyzer *BashCommandAnalyzer
}

// NewLocalBashTool 创建本地 Bash 工具
func NewLocalBashTool() *LocalBashTool {
	return &LocalBashTool{
		analyzer: NewBashCommandAnalyzer(),
	}
}

// Name 返回工具名称
func (t *LocalBashTool) Name() string {
	return "local_bash"
}

// Description 返回工具描述
func (t *LocalBashTool) Description() string {
	return "在本地服务器执行 bash 命令，支持 grep/awk/sed/cat 等所有标准 Linux 工具"
}

// Parameters 返回参数定义（JSON Schema 格式）
func (t *LocalBashTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "要执行的 bash 命令",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "命令执行超时时间（秒），默认 30",
			},
			"working_dir": map[string]interface{}{
				"type":        "string",
				"description": "命令执行的工作目录",
			},
		},
		"required": []string{"command"},
	}
}

// RiskLevel 动态评估命令风险等级
func (t *LocalBashTool) RiskLevel() RiskLevel {
	// local_bash 的风险是动态的，实际风险在执行时根据命令内容确定
	// 这里返回默认值 RiskMedium
	return RiskMedium
}

// Execute 执行 bash 命令
func (t *LocalBashTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	// 解析参数
	command := parseStringParam(args, "command")
	if command == "" {
		return &Result{
			Success: false,
			Error:   "command 参数不能为空",
		}, fmt.Errorf("command 参数不能为空")
	}

	// 检查危险命令黑名单
	if err := checkDangerousCommand(command); err != nil {
		return &Result{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// 解析超时（默认 30 秒，支持浮点数）
	timeoutSec := 30.0
	if v, ok := parseFloat64Param(args, "timeout"); ok && v > 0 {
		timeoutSec = v
	}

	// 解析工作目录
	workingDir := parseStringParam(args, "working_dir")
	if workingDir != "" {
		// 验证工作目录存在
		if info, err := os.Stat(workingDir); err != nil || !info.IsDir() {
			return &Result{
				Success: false,
				Error:   fmt.Sprintf("工作目录不存在或不是目录: %s", workingDir),
			}, fmt.Errorf("工作目录不存在: %s", workingDir)
		}
	}

	// 评估命令风险等级
	risk := t.analyzer.AnalyzeRisk(command)
	riskDesc := t.analyzer.GetRiskDescription(command)

	// 创建超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec*float64(time.Second)))
	defer cancel()

	// 执行命令
	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", command)

	// 设置工作目录
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// 执行并获取输出
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// 处理输出截断
	truncated := false
	if len(outputStr) > outputTruncateLimit {
		outputStr = outputStr[:outputTruncateLimit]
		truncated = true
	}

	if err != nil {
		result := &Result{
			Success: false,
			Output:  outputStr,
			Error:   err.Error(),
			Summary: fmt.Sprintf("命令执行失败 [风险等级: %s - %s]", risk.String(), riskDesc),
		}
		if truncated {
			result.Output += "\n... [输出已截断，超过 10000 字符限制]"
		}
		return result, nil
	}

	result := &Result{
		Success: true,
		Output:  outputStr,
		Summary: fmt.Sprintf("命令执行成功 [风险等级: %s - %s]", risk.String(), riskDesc),
	}
	if truncated {
		result.Output += "\n... [输出已截断，超过 10000 字符限制]"
		result.Summary += " [输出已截断]"
	}

	return result, nil
}

// dangerousPattern 危险命令匹配模式
type dangerousPattern struct {
	pattern *regexp.Regexp
	desc    string
}

// 包级别预编译危险命令正则（避免每次调用重复编译）
var dangerousPatterns = []dangerousPattern{
	// rm -rf / 及其变体
	{regexp.MustCompile(`(?i)rm\s+-[a-zA-Z]*f[a-zA-Z]*\s+/(\s|$)`), "禁止执行 rm -rf / 或类似命令"},
	{regexp.MustCompile(`(?i)rm\s+--no-preserve-root`), "禁止执行 rm --no-preserve-root"},
	// mkfs 系列（格式化文件系统）
	{regexp.MustCompile(`(?i)^\s*mkfs\.`), "禁止执行文件系统格式化命令"},
	{regexp.MustCompile(`(?i)^\s*mkfs\s+`), "禁止执行文件系统格式化命令"},
	// dd 命令写入 /dev/zero 到磁盘设备
	{regexp.MustCompile(`(?i)dd\s+.*if=/dev/zero\s+.*of=/dev/`), "禁止执行 dd 覆盖磁盘设备命令"},
	{regexp.MustCompile(`(?i)dd\s+.*if=/dev/urandom\s+.*of=/dev/`), "禁止执行 dd 覆盖磁盘设备命令"},
	{regexp.MustCompile(`(?i)dd\s+.*if=/dev/random\s+.*of=/dev/`), "禁止执行 dd 覆盖磁盘设备命令"},
	{regexp.MustCompile(`(?i)dd\s+.*of=/dev/[sh]d[a-z]`), "禁止执行 dd 覆盖磁盘设备命令"},
	{regexp.MustCompile(`(?i)dd\s+.*of=/dev/nvme`), "禁止执行 dd 覆盖磁盘设备命令"},
	// 著名的 fork bomb
	{regexp.MustCompile(`:?\(\)\{\s*:\|:\&\s*\};\s*:`), "禁止执行 fork bomb"},
	{regexp.MustCompile(`:?\(\)\{\s*:\|:\s*\};\s*:`), "禁止执行 fork bomb"},
	// 直接写入系统设备
	{regexp.MustCompile(`(?i)>\s*/dev/sd[a-z]`), "禁止直接写入磁盘设备"},
	{regexp.MustCompile(`(?i)>\s*/dev/hd[a-z]`), "禁止直接写入磁盘设备"},
	{regexp.MustCompile(`(?i)>\s*/dev/nvme`), "禁止直接写入磁盘设备"},
	{regexp.MustCompile(`(?i)>\s*/dev/mem`), "禁止直接写入内存设备"},
	{regexp.MustCompile(`(?i)>\s*/dev/kmem`), "禁止直接写入内核内存设备"},
	{regexp.MustCompile(`(?i)>\s*/dev/port`), "禁止直接写入端口设备"},
	{regexp.MustCompile(`(?i)>\s*/dev/zero\s+\d+`), "禁止执行覆盖设备命令"},
	// 权限提升危险操作
	{regexp.MustCompile(`(?i)chmod\s+-R\s+777\s+/`), "禁止修改根目录权限为 777"},
	{regexp.MustCompile(`(?i)chmod\s+-R\s+000\s+/`), "禁止修改根目录权限为 000"},
	// 删除关键系统目录
	{regexp.MustCompile(`(?i)rm\s+.*\s+/bin\b`), "禁止删除 /bin 目录"},
	{regexp.MustCompile(`(?i)rm\s+.*\s+/sbin\b`), "禁止删除 /sbin 目录"},
	{regexp.MustCompile(`(?i)rm\s+.*\s+/usr/bin\b`), "禁止删除 /usr/bin 目录"},
	{regexp.MustCompile(`(?i)rm\s+.*\s+/lib\b`), "禁止删除 /lib 目录"},
	{regexp.MustCompile(`(?i)rm\s+.*\s+/lib64\b`), "禁止删除 /lib64 目录"},
	{regexp.MustCompile(`(?i)rm\s+.*\s+/etc\b`), "禁止删除 /etc 目录"},
	{regexp.MustCompile(`(?i)rm\s+.*\s+/boot\b`), "禁止删除 /boot 目录"},
}

// checkDangerousCommand 检查命令是否在危险黑名单中
func checkDangerousCommand(command string) error {
	lowerCmd := strings.ToLower(strings.TrimSpace(command))

	for _, dp := range dangerousPatterns {
		if dp.pattern.MatchString(lowerCmd) {
			return fmt.Errorf("危险命令被拦截: %s", dp.desc)
		}
	}

	return nil
}

// BashCommandAnalyzer bash 命令分析器（从 internal/tools 导入并适配）
type BashCommandAnalyzer struct {
	readOnlyPatterns  []*regexp.Regexp
	writePatterns     []*regexp.Regexp
	dangerousPatterns []*regexp.Regexp
}

// 包级别预编译命令分析正则（避免每次创建分析器重复编译）
var (
	readOnlyPatterns = []*regexp.Regexp{
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
		regexp.MustCompile(`^(ping|traceroute|nslookup|dig|host)\s+`),
		regexp.MustCompile(`^netstat\s*`),
		regexp.MustCompile(`^ss\s*`),
		regexp.MustCompile(`^ip\s+addr`),
		regexp.MustCompile(`^ip\s+route\s+show`),
		regexp.MustCompile(`^ifconfig\s*$`),
		regexp.MustCompile(`^telnet\s+`),
		regexp.MustCompile(`^nc\s+-[zv]+\s+`),
		regexp.MustCompile(`^nmap\s+`),
		regexp.MustCompile(`^stat\s+`),
		regexp.MustCompile(`^file\s+`),
		regexp.MustCompile(`^wc\s+`),
		regexp.MustCompile(`^od\s+`),
		regexp.MustCompile(`^xxd\s+`),
		regexp.MustCompile(`^tree\s*`),
		regexp.MustCompile(`^java\s+-version`),
		regexp.MustCompile(`^javac\s+-version`),
		regexp.MustCompile(`^which\s+`),
		regexp.MustCompile(`^whereis\s+`),
		regexp.MustCompile(`^echo\s+["']?\$`),
		regexp.MustCompile(`^echo\s+.*`),
		regexp.MustCompile(`^printenv`),
		regexp.MustCompile(`^env\s*`),
		regexp.MustCompile(`^command\s+-v\s+`),
		regexp.MustCompile(`^type\s+`),
		regexp.MustCompile(`^hash\s+`),
		regexp.MustCompile(`^(jq|yq)\s+`),
		regexp.MustCompile(`^column\s+`),
		regexp.MustCompile(`^nl\s+`),
		regexp.MustCompile(`^tr\s+`),
		regexp.MustCompile(`^sort\s+`),
		regexp.MustCompile(`^uniq\s+`),
		regexp.MustCompile(`^(lsof|strace|ltrace)\s+`),
		regexp.MustCompile(`^(htop|iotop|vmstat|iostat)\s*`),
		regexp.MustCompile(`^tcpdump\s+`),
		regexp.MustCompile(`^docker\s+(ps|images|logs|inspect|stats|version|info)`),
		regexp.MustCompile(`^kubectl\s+(get|describe|logs|top|explain|version|api-resources|api-versions)`),
		regexp.MustCompile(`^git\s+(status|log|diff|show|branch(\s+-[vla])?)`),
		regexp.MustCompile(`^(tar|unzip|zipinfo)\s+.*-[lt]`),
		regexp.MustCompile(`(?i)^(mysql|psql).*\s+(SELECT|SHOW|EXPLAIN|DESC|DESCRIBE)\s+`),
		regexp.MustCompile(`^redis-cli\s+(GET|KEYS|INFO|MONITOR|TTL|TYPE|SCAN|EXISTS|LLEN|SCARD|ZCARD|HLEN)`),
		regexp.MustCompile(`^(apt|yum|dnf)\s+list\s+(installed|upgradable)`),
		regexp.MustCompile(`^(pip|pip3)\s+list`),
		regexp.MustCompile(`^npm\s+list`),
		regexp.MustCompile(`^(vim|vi)\s+-R\s+`),
		regexp.MustCompile(`^view\s+`),
		regexp.MustCompile(`^.*(cat|grep|awk|sed|head|tail|wc|sort|uniq).*\|.*(cat|grep|awk|sed|head|tail|wc|sort|uniq)`),
	}
	writePatterns = []*regexp.Regexp{
		regexp.MustCompile(`[^0-9]>\s*[^&/]`),
		regexp.MustCompile(`[^0-9]>>\s*[^&/]`),
		regexp.MustCompile(`^(vi|vim|nano|emacs|gedit)\s+`),
		regexp.MustCompile(`^tar\s+.*-[xc]`),
		regexp.MustCompile(`^(gzip|gunzip|bzip2|bunzip2|xz|unxz)\s+`),
		regexp.MustCompile(`^unzip\s+[^-]`),
		regexp.MustCompile(`^zip\s+-r`),
		regexp.MustCompile(`^(wget|curl).*-[oO]`),
		regexp.MustCompile(`^scp\s+`),
		regexp.MustCompile(`^docker\s+(run|create|start|stop|restart|pause|unpause|exec|build|commit|tag|push|pull)`),
		regexp.MustCompile(`^kubectl\s+(apply|create|patch|replace|scale|expose|rollout|set|edit|annotate|label)`),
		regexp.MustCompile(`^git\s+(add|commit|push|pull|fetch|merge|rebase|cherry-pick|stash)`),
		regexp.MustCompile(`^git\s+checkout`),
		regexp.MustCompile(`(?i)^(mysql|psql).*\s+(INSERT|UPDATE|CREATE|ALTER)\s+`),
		regexp.MustCompile(`^redis-cli\s+(SET|SETEX|SETNX|MSET|HSET|LPUSH|RPUSH|SADD|ZADD|INCR|DECR|APPEND)`),
		regexp.MustCompile(`^(pip|pip3)\s+(install|uninstall)`),
	}
	dangerousPatternsAnalyzer = []*regexp.Regexp{
		regexp.MustCompile(`^rm\s+`),
		regexp.MustCompile(`^dd\s+`),
		regexp.MustCompile(`^mkfs`),
		regexp.MustCompile(`^fdisk`),
		regexp.MustCompile(`^parted`),
		regexp.MustCompile(`^kill\s+`),
		regexp.MustCompile(`^killall\s+`),
		regexp.MustCompile(`^shutdown`),
		regexp.MustCompile(`^reboot`),
		regexp.MustCompile(`^halt`),
		regexp.MustCompile(`^init\s+`),
		regexp.MustCompile(`^systemctl\s+(stop|restart|reload|disable)`),
		regexp.MustCompile(`^service\s+.*\s+(stop|restart)`),
		regexp.MustCompile(`^docker\s+(rm|rmi|system\s+prune|volume\s+rm|network\s+rm)`),
		regexp.MustCompile(`^kubectl\s+(delete|drain)`),
		regexp.MustCompile(`^git\s+reset\s+--hard`),
		regexp.MustCompile(`^git\s+clean\s+-[fd]`),
		regexp.MustCompile(`^git\s+push\s+.*--force`),
		regexp.MustCompile(`(?i)^(mysql|psql).*\s+(DROP|DELETE|TRUNCATE)\s+`),
		regexp.MustCompile(`^redis-cli\s+(DEL|FLUSHALL|FLUSHDB|CONFIG\s+SET)`),
		regexp.MustCompile(`^route\s+(add|del)`),
		regexp.MustCompile(`^ip\s+route\s+(add|del)`),
		regexp.MustCompile(`^iptables\s+-[ADI]`),
		regexp.MustCompile(`^(apt|yum|dnf)\s+(install|remove|purge|autoremove)`),
		regexp.MustCompile(`^npm\s+(install|uninstall)\s+-g`),
	}
	// 预编译重定向清理正则
	redirectCleanPatterns = []*struct {
		re   *regexp.Regexp
		repl string
	}{
		{regexp.MustCompile(`\s+2>&1`), ""},
		{regexp.MustCompile(`\s+>/dev/null`), ""},
		{regexp.MustCompile(`\s+2>/dev/null`), ""},
		{regexp.MustCompile(`\s+&>/dev/null`), ""},
	}
)

// NewBashCommandAnalyzer 创建 bash 命令分析器
func NewBashCommandAnalyzer() *BashCommandAnalyzer {
	return &BashCommandAnalyzer{
		readOnlyPatterns:  readOnlyPatterns,
		writePatterns:     writePatterns,
		dangerousPatterns: dangerousPatternsAnalyzer,
	}
}

// AnalyzeRisk 分析命令风险等级
func (a *BashCommandAnalyzer) AnalyzeRisk(command string) RiskLevel {
	command = strings.TrimSpace(command)
	cleanCommand := a.cleanRedirects(command)

	// 处理复合命令
	if strings.Contains(cleanCommand, "||") || strings.Contains(cleanCommand, "&&") {
		return a.analyzeCompoundCommand(cleanCommand)
	}

	// 处理管道命令
	if strings.Contains(cleanCommand, "|") && !strings.Contains(cleanCommand, "||") {
		return a.analyzePipelineCommand(cleanCommand)
	}

	// 检查危险命令
	for _, pattern := range a.dangerousPatterns {
		if pattern.MatchString(cleanCommand) {
			return RiskCritical
		}
	}

	// 检查写入操作
	for _, pattern := range a.writePatterns {
		if pattern.MatchString(cleanCommand) {
			return RiskHigh
		}
	}

	// 检查只读命令
	for _, pattern := range a.readOnlyPatterns {
		if pattern.MatchString(cleanCommand) {
			return RiskSafe
		}
	}

	return RiskMedium
}

// cleanRedirects 清理命令中的重定向用于风险分析
func (a *BashCommandAnalyzer) cleanRedirects(command string) string {
	for _, p := range redirectCleanPatterns {
		command = p.re.ReplaceAllString(command, p.repl)
	}
	return strings.TrimSpace(command)
}

// analyzeCompoundCommand 分析复合命令
func (a *BashCommandAnalyzer) analyzeCompoundCommand(command string) RiskLevel {
	var parts []string
	orParts := strings.Split(command, "||")
	for _, orPart := range orParts {
		andParts := strings.Split(orPart, "&&")
		parts = append(parts, andParts...)
	}

	maxRisk := RiskSafe
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = a.cleanRedirects(part)
		risk := a.analyzeSingleCommand(part)
		if risk > maxRisk {
			maxRisk = risk
		}
	}
	return maxRisk
}

// analyzePipelineCommand 分析管道命令
func (a *BashCommandAnalyzer) analyzePipelineCommand(command string) RiskLevel {
	parts := strings.Split(command, "|")
	maxRisk := RiskSafe
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = a.cleanRedirects(part)
		risk := a.analyzeSingleCommand(part)
		if risk > maxRisk {
			maxRisk = risk
		}
	}
	return maxRisk
}

// analyzeSingleCommand 分析单个命令
func (a *BashCommandAnalyzer) analyzeSingleCommand(command string) RiskLevel {
	for _, pattern := range a.dangerousPatterns {
		if pattern.MatchString(command) {
			return RiskCritical
		}
	}
	for _, pattern := range a.writePatterns {
		if pattern.MatchString(command) {
			return RiskHigh
		}
	}
	for _, pattern := range a.readOnlyPatterns {
		if pattern.MatchString(command) {
			return RiskSafe
		}
	}
	return RiskMedium
}

// GetRiskDescription 获取风险描述
func (a *BashCommandAnalyzer) GetRiskDescription(command string) string {
	risk := a.AnalyzeRisk(command)
	switch risk {
	case RiskSafe:
		return "只读操作，安全"
	case RiskLow:
		return "低风险操作"
	case RiskMedium:
		return "中等风险，可能修改系统状态"
	case RiskHigh:
		return "高风险，会修改文件或配置"
	case RiskCritical:
		return "危险操作，可能导致系统不稳定或数据丢失"
	default:
		return "未知风险"
	}
}

// IsReadOnly 判断命令是否只读
func (a *BashCommandAnalyzer) IsReadOnly(command string) bool {
	return a.AnalyzeRisk(command) == RiskSafe
}

