package tools

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// AnalyzeOutputTool 输出分析工具
type AnalyzeOutputTool struct{}

// NewAnalyzeOutputTool 创建输出分析工具
func NewAnalyzeOutputTool() *AnalyzeOutputTool {
	return &AnalyzeOutputTool{}
}

// Name 返回工具名称
func (t *AnalyzeOutputTool) Name() string {
	return "analyze_output"
}

// Description 返回工具描述
func (t *AnalyzeOutputTool) Description() string {
	return "分析命令输出，提取关键信息、发现异常、生成摘要。此工具不直接执行命令，而是对已有输出进行智能分析"
}

// Parameters 返回参数定义（JSON Schema 格式）
func (t *AnalyzeOutputTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"output": map[string]interface{}{
				"type":        "string",
				"description": "要分析的命令输出文本",
			},
			"analysis_type": map[string]interface{}{
				"type":        "string",
				"description": "分析类型: summary(摘要), error_detect(错误检测), key_extract(关键信息提取), compare(对比)",
				"enum":        []string{"summary", "error_detect", "key_extract", "compare"},
			},
			"context": map[string]interface{}{
				"type":        "string",
				"description": "额外的上下文信息，用于辅助分析",
			},
		},
		"required": []string{"output"},
	}
}

// RiskLevel 返回风险等级（文本分析是安全的只读操作）
func (t *AnalyzeOutputTool) RiskLevel() RiskLevel {
	return RiskSafe
}

// Execute 执行输出分析
func (t *AnalyzeOutputTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	// 解析参数
	output := parseStringParam(args, "output")
	if output == "" {
		return &Result{
			Success: true,
			Output:  "输入为空，无内容可分析",
			Summary: "空输入分析",
		}, nil
	}

	// 解析分析类型（默认 summary）
	analysisType := parseStringParam(args, "analysis_type")
	if analysisType == "" {
		analysisType = "summary"
	}

	// 解析上下文
	context := parseStringParam(args, "context")

	var analysisResult string
	var summary string

	// 根据分析类型执行不同的分析
	switch analysisType {
	case "summary":
		analysisResult, summary = t.analyzeSummary(output)
	case "error_detect":
		analysisResult, summary = t.analyzeErrorDetect(output)
	case "key_extract":
		analysisResult, summary = t.analyzeKeyExtract(output, context)
	case "compare":
		analysisResult, summary = t.analyzeCompare(output, context)
	default:
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("不支持的分析类型: %s", analysisType),
		}, fmt.Errorf("不支持的分析类型: %s", analysisType)
	}

	return &Result{
		Success: true,
		Output:  analysisResult,
		Summary: summary,
	}, nil
}

// analyzeSummary 生成输出摘要
func (t *AnalyzeOutputTool) analyzeSummary(output string) (string, string) {
	lines := strings.Split(output, "\n")
	totalLines := len(lines)

	// 统计非空行数
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	charCount := len(output)

	// 获取前 10 行预览
	previewLines := 10
	if totalLines < previewLines {
		previewLines = totalLines
	}
	preview := strings.Join(lines[:previewLines], "\n")

	result := fmt.Sprintf(`=== 输出摘要 ===
总 行 数: %d
非空行数: %d
字符总数: %d

=== 前 %d 行预览 ===
%s`, totalLines, nonEmptyLines, charCount, previewLines, preview)

	if totalLines > previewLines {
		result += fmt.Sprintf("\n... (共 %d 行, 仅显示前 %d 行)", totalLines, previewLines)
	}

	summary := fmt.Sprintf("文本摘要: %d 行, %d 字符", totalLines, charCount)

	return result, summary
}

// analyzeErrorDetect 扫描输出中的错误信息
func (t *AnalyzeOutputTool) analyzeErrorDetect(output string) (string, string) {
	// 定义错误关键词模式
	errorPatterns := []struct {
		pattern *regexp.Regexp
		level   string
		desc    string
	}{
		{regexp.MustCompile(`(?i)error\s*[:\-]`), "error", "错误"},
		{regexp.MustCompile(`(?i)\bfatal\b`), "critical", "致命错误"},
		{regexp.MustCompile(`(?i)\bfailed?\b`), "error", "失败"},
		{regexp.MustCompile(`(?i)\bfailure\b`), "error", "失败"},
		{regexp.MustCompile(`(?i)timeout`), "warning", "超时"},
		{regexp.MustCompile(`(?i)\bunreachable\b`), "error", "不可达"},
		{regexp.MustCompile(`(?i)permission\s+denied`), "error", "权限拒绝"},
		{regexp.MustCompile(`(?i)access\s+denied`), "error", "访问拒绝"},
		{regexp.MustCompile(`(?i)not\s+found`), "warning", "未找到"},
		{regexp.MustCompile(`(?i)no\s+such\s+(file|directory)`), "warning", "文件/目录不存在"},
		{regexp.MustCompile(`(?i)connection\s+refused`), "error", "连接拒绝"},
		{regexp.MustCompile(`(?i)connection\s+reset`), "error", "连接重置"},
		{regexp.MustCompile(`(?i)connection\s+timed?\s*out`), "error", "连接超时"},
		{regexp.MustCompile(`(?i)\brefused\b`), "error", "拒绝"},
		{regexp.MustCompile(`(?i)cannot\s+access`), "warning", "无法访问"},
		{regexp.MustCompile(`(?i)invalid\s+(argument|option)`), "warning", "无效参数"},
		{regexp.MustCompile(`(?i)segfault|segmentation\s+fault`), "critical", "段错误"},
		{regexp.MustCompile(`(?i)out\s+of\s+(memory|space)`), "critical", "内存/空间不足"},
		{regexp.MustCompile(`(?i)disk\s+full`), "critical", "磁盘已满"},
		{regexp.MustCompile(`(?i)killed\b`), "critical", "进程被终止"},
		{regexp.MustCompile(`(?i)abort`), "critical", "异常终止"},
		{regexp.MustCompile(`(?i)warn(ing)?\s*[:\-]`), "warning", "警告"},
		{regexp.MustCompile(`(?i)deprecated`), "warning", "已弃用"},
		{regexp.MustCompile(`(?i)unauthorized`), "error", "未授权"},
		{regexp.MustCompile(`(?i)authentication\s+fail`), "error", "认证失败"},
		{regexp.MustCompile(`(?i)ssl\s+error`), "error", "SSL 错误"},
		{regexp.MustCompile(`(?i)certificate\s+(error|invalid|expired)`), "error", "证书错误"},
	}

	lines := strings.Split(output, "\n")
	var findings []string
	errorCount := 0
	warningCount := 0
	criticalCount := 0

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		for _, ep := range errorPatterns {
			if ep.pattern.MatchString(trimmed) {
				findings = append(findings, fmt.Sprintf("[%s] 行 %d: %s - %s", ep.level, lineNum+1, ep.desc, trimmed))
				switch ep.level {
				case "error":
					errorCount++
				case "warning":
					warningCount++
				case "critical":
					criticalCount++
				}
				// 每行只匹配一个模式
				break
			}
		}
	}

	// 根据退出码检测错误（如果输出包含 exit code）
	exitCodePattern := regexp.MustCompile(`(?i)exit\s+(code\s+)?(\d+)`)
	if matches := exitCodePattern.FindAllStringSubmatch(output, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 3 {
				code, _ := strconv.Atoi(match[2])
				if code != 0 {
					findings = append(findings, fmt.Sprintf("[error] 非零退出码: %d", code))
					errorCount++
				}
			}
		}
	}

	// 生成结果
	var result strings.Builder
	result.WriteString("=== 错误检测结果 ===\n")
	result.WriteString(fmt.Sprintf("严重错误: %d\n", criticalCount))
	result.WriteString(fmt.Sprintf("普通错误: %d\n", errorCount))
	result.WriteString(fmt.Sprintf("警告信息: %d\n", warningCount))
	result.WriteString(fmt.Sprintf("总计发现: %d\n\n", criticalCount+errorCount+warningCount))

	if len(findings) > 0 {
		result.WriteString("=== 详细信息 ===\n")
		for _, finding := range findings {
			result.WriteString(finding)
			result.WriteString("\n")
		}
	} else {
		result.WriteString("未发现错误或警告。")
	}

	summary := fmt.Sprintf("错误检测: %d 严重, %d 错误, %d 警告", criticalCount, errorCount, warningCount)

	return result.String(), summary
}

// analyzeKeyExtract 提取关键信息
func (t *AnalyzeOutputTool) analyzeKeyExtract(output string, context string) (string, string) {
	var result strings.Builder
	var findings []string

	result.WriteString("=== 关键信息提取 ===\n")

	// 1. 提取 IP 地址
	ipPattern := regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\b`)
	ipMatches := ipPattern.FindAllString(output, -1)
	if len(ipMatches) > 0 {
		// 去重
		seen := make(map[string]bool)
		var uniqueIPs []string
		for _, ip := range ipMatches {
			if !seen[ip] {
				seen[ip] = true
				uniqueIPs = append(uniqueIPs, ip)
			}
		}
		result.WriteString(fmt.Sprintf("\n[IP 地址] (%d 个):\n", len(uniqueIPs)))
		for _, ip := range uniqueIPs {
			// 验证是有效 IP
			if net.ParseIP(ip) != nil {
				result.WriteString(fmt.Sprintf("  - %s\n", ip))
				findings = append(findings, "IP: "+ip)
			}
		}
	}

	// 2. 提取端口号（PORT 或 port 后跟随的数字）
	portPattern := regexp.MustCompile(`(?i)(port[:\s]+|port=)(\d{2,5})\b`)
	portMatches := portPattern.FindAllStringSubmatch(output, -1)
	if len(portMatches) > 0 {
		seen := make(map[string]bool)
		result.WriteString(fmt.Sprintf("\n[端口号] (%d 个):\n", len(portMatches)))
		for _, match := range portMatches {
			if len(match) >= 3 && !seen[match[2]] {
				seen[match[2]] = true
				port, _ := strconv.Atoi(match[2])
				if port > 0 && port <= 65535 {
					serviceName := getWellKnownServiceName(port)
					if serviceName != "" {
						result.WriteString(fmt.Sprintf("  - %d (%s)\n", port, serviceName))
					} else {
						result.WriteString(fmt.Sprintf("  - %d\n", port))
					}
					findings = append(findings, fmt.Sprintf("Port: %d", port))
				}
			}
		}
	}

	// 3. 提取 PID（进程 ID）
	pidPattern := regexp.MustCompile(`(?i)(?:pid|process\s+id)[:\s]+(\d+)`)
	pidMatches := pidPattern.FindAllStringSubmatch(output, -1)
	if len(pidMatches) > 0 {
		seen := make(map[string]bool)
		result.WriteString(fmt.Sprintf("\n[进程 PID] (%d 个):\n", len(pidMatches)))
		for _, match := range pidMatches {
			if len(match) >= 2 && !seen[match[1]] {
				seen[match[1]] = true
				result.WriteString(fmt.Sprintf("  - %s\n", match[1]))
				findings = append(findings, "PID: "+match[1])
			}
		}
	}

	// 4. 提取文件路径
	pathPattern := regexp.MustCompile(`(/[\w\-./]+[\w\-]+)`)
	pathMatches := pathPattern.FindAllString(output, -1)
	if len(pathMatches) > 0 {
		seen := make(map[string]bool)
		var uniquePaths []string
		for _, path := range pathMatches {
			lowerPath := strings.ToLower(path)
			if !seen[lowerPath] && len(path) > 2 {
				seen[lowerPath] = true
				uniquePaths = append(uniquePaths, path)
			}
		}
		if len(uniquePaths) > 0 {
			// 限制显示数量
			displayCount := len(uniquePaths)
			if displayCount > 20 {
				displayCount = 20
			}
			result.WriteString(fmt.Sprintf("\n[文件路径] (%d 个, 显示前 %d):\n", len(uniquePaths), displayCount))
			for i := 0; i < displayCount; i++ {
				result.WriteString(fmt.Sprintf("  - %s\n", uniquePaths[i]))
				findings = append(findings, "Path: "+uniquePaths[i])
			}
			if len(uniquePaths) > displayCount {
				result.WriteString(fmt.Sprintf("  ... 还有 %d 个路径未显示\n", len(uniquePaths)-displayCount))
			}
		}
	}

	// 5. 提取 URL
	urlPattern := regexp.MustCompile(`(https?://[^\s]+)`)
	urlMatches := urlPattern.FindAllString(output, -1)
	if len(urlMatches) > 0 {
		seen := make(map[string]bool)
		var uniqueURLs []string
		for _, url := range urlMatches {
			if !seen[url] {
				seen[url] = true
				uniqueURLs = append(uniqueURLs, url)
			}
		}
		result.WriteString(fmt.Sprintf("\n[URL 地址] (%d 个):\n", len(uniqueURLs)))
		for _, url := range uniqueURLs {
			result.WriteString(fmt.Sprintf("  - %s\n", url))
			findings = append(findings, "URL: "+url)
		}
	}

	// 6. 提取版本号
	versionPattern := regexp.MustCompile(`(?i)(?:version|v)[:\s]+([\d.]+[\w.-]*)`)
	versionMatches := versionPattern.FindAllStringSubmatch(output, -1)
	if len(versionMatches) > 0 {
		seen := make(map[string]bool)
		result.WriteString(fmt.Sprintf("\n[版本号] (%d 个):\n", len(versionMatches)))
		for _, match := range versionMatches {
			if len(match) >= 2 && !seen[match[1]] {
				seen[match[1]] = true
				result.WriteString(fmt.Sprintf("  - %s\n", match[1]))
				findings = append(findings, "Version: "+match[1])
			}
		}
	}

	// 7. 如果提供了上下文信息，添加到结果
	if context != "" {
		result.WriteString(fmt.Sprintf("\n[上下文] %s\n", context))
	}

	if len(findings) == 0 {
		result.WriteString("\n未发现可提取的关键信息。\n")
	}

	summary := fmt.Sprintf("关键信息提取: %d 项发现", len(findings))

	return result.String(), summary
}

// analyzeCompare 比较两段输出
func (t *AnalyzeOutputTool) analyzeCompare(output1 string, output2 string) (string, string) {
	// output2 来自 context 参数
	if output2 == "" {
		return "compare 分析需要两段文本。请将第二段文本通过 context 参数传入。", "比较失败: 缺少第二段文本"
	}

	var result strings.Builder
	result.WriteString("=== 输出对比分析 ===\n\n")

	// 基础统计对比
	lines1 := strings.Split(output1, "\n")
	lines2 := strings.Split(output2, "\n")
	chars1 := len(output1)
	chars2 := len(output2)

	result.WriteString(fmt.Sprintf("文本1: %d 行, %d 字符\n", len(lines1), chars1))
	result.WriteString(fmt.Sprintf("文本2: %d 行, %d 字符\n\n", len(lines2), chars2))

	// 行数差异
	lineDiff := len(lines1) - len(lines2)
	if lineDiff > 0 {
		result.WriteString(fmt.Sprintf("行数差异: 文本1 多 %d 行\n", lineDiff))
	} else if lineDiff < 0 {
		result.WriteString(fmt.Sprintf("行数差异: 文本2 多 %d 行\n", -lineDiff))
	} else {
		result.WriteString("行数差异: 相同\n")
	}

	// 字符数差异
	charDiff := chars1 - chars2
	if charDiff > 0 {
		result.WriteString(fmt.Sprintf("字符差异: 文本1 多 %d 字符\n", charDiff))
	} else if charDiff < 0 {
		result.WriteString(fmt.Sprintf("字符差异: 文本2 多 %d 字符\n", -charDiff))
	} else {
		result.WriteString("字符差异: 相同\n")
	}

	// 查找相同行
	set1 := make(map[string]bool)
	for _, line := range lines1 {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			set1[trimmed] = true
		}
	}

	var commonLines []string
	for _, line := range lines2 {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && set1[trimmed] {
			commonLines = append(commonLines, trimmed)
			set1[trimmed] = false // 避免重复计数
		}
	}

	nonEmpty1 := 0
	for _, line := range lines1 {
		if strings.TrimSpace(line) != "" {
			nonEmpty1++
		}
	}
	nonEmpty2 := 0
	for _, line := range lines2 {
		if strings.TrimSpace(line) != "" {
			nonEmpty2++
		}
	}

	result.WriteString(fmt.Sprintf("\n文本1 非空行: %d\n", nonEmpty1))
	result.WriteString(fmt.Sprintf("文本2 非空行: %d\n", nonEmpty2))
	result.WriteString(fmt.Sprintf("共同非空行: %d\n", len(commonLines)))

	// 如果完全相等
	if output1 == output2 {
		result.WriteString("\n*** 两段文本完全相同 ***\n")
	} else if len(commonLines) == 0 {
		result.WriteString("\n*** 两段文本无共同内容 ***\n")
	}

	// 显示部分相同行
	if len(commonLines) > 0 && len(commonLines) <= 20 {
		result.WriteString("\n=== 共同内容 ===\n")
		for _, line := range commonLines {
			result.WriteString(fmt.Sprintf("  %s\n", line))
		}
	} else if len(commonLines) > 20 {
		result.WriteString(fmt.Sprintf("\n共同内容较多 (%d 行), 省略显示\n", len(commonLines)))
	}

	summary := fmt.Sprintf("对比结果: 文本1(%d行) vs 文本2(%d行), 共同行%d", len(lines1), len(lines2), len(commonLines))

	return result.String(), summary
}

// getWellKnownServiceName 返回知名服务端口号对应的服务名
func getWellKnownServiceName(port int) string {
	services := map[int]string{
		20:    "FTP Data",
		21:    "FTP",
		22:    "SSH",
		23:    "Telnet",
		25:    "SMTP",
		53:    "DNS",
		80:    "HTTP",
		110:   "POP3",
		143:   "IMAP",
		443:   "HTTPS",
		3306:  "MySQL",
		5432:  "PostgreSQL",
		6379:  "Redis",
		27017: "MongoDB",
		8080:  "HTTP Proxy",
		8443:  "HTTPS Alt",
		9200:  "Elasticsearch",
		9090:  "Prometheus",
		3389:  "RDP",
		5900:  "VNC",
		68:    "DHCP Client",
		67:    "DHCP Server",
		161:   "SNMP",
		162:   "SNMP Trap",
		389:   "LDAP",
		636:   "LDAPS",
		993:   "IMAPS",
		995:   "POP3S",
		587:   "SMTP Submission",
		465:   "SMTPS",
		3000:  "Grafana/Dev",
		5000:  "Flask/Dev",
		8000:  "HTTP Alt/Dev",
	}

	if name, ok := services[port]; ok {
		return name
	}
	return ""
}
