// memory.go - 动态记忆注入逻辑
// 负责将 Evolver 三层记忆（经验/环境/偏好）格式化为 Prompt 可读的上下文
// 作为 Layer 5 动态记忆层的数据源
package prompt

import (
	"fmt"
	"strings"

	"opsxcli/internal/agentv2/evolver"
)

// MemoryInjector 记忆注入器
// 负责从 Evolver 引擎获取记忆并格式化为 Prompt 上下文
type MemoryInjector struct {
	envMemory *evolver.EnvironmentMemory
	expMemory *evolver.ExperienceMemory
}

// NewMemoryInjector 创建记忆注入器
func NewMemoryInjector(envMem *evolver.EnvironmentMemory, expMem *evolver.ExperienceMemory) *MemoryInjector {
	return &MemoryInjector{
		envMemory: envMem,
		expMemory: expMem,
	}
}

// ═══════════════════════════════════════════════════════════════
// BuildMemoryContext 构建完整的记忆上下文（三层融合）
// 这是 Layer 5 动态记忆层的核心数据源
// ═══════════════════════════════════════════════════════════════
func (m *MemoryInjector) BuildMemoryContext(query string) string {
	if m == nil {
		return ""
	}

	parts := make([]string, 0, 3)

	// Layer 5.1: 经验提示（基于查询匹配）
	if hint := m.buildExperienceHints(query); hint != "" {
		parts = append(parts, hint)
	}

	// Layer 5.2: 环境上下文（已知服务器 + 常用路径）
	if env := m.buildEnvironmentContext(query); env != "" {
		parts = append(parts, env)
	}

	// Layer 5.3: 用户偏好
	if prefs := m.buildUserPreferences(); prefs != "" {
		parts = append(parts, prefs)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n\n")
}

// ═══════════════════════════════════════════════════════════════
// Layer 5.1: 经验提示注入
// 根据查询内容匹配相关经验，生成最佳实践提示
// ═══════════════════════════════════════════════════════════════
func (m *MemoryInjector) buildExperienceHints(query string) string {
	if m.expMemory == nil {
		return ""
	}

	// 获取任务类型
	taskType := classifyTaskTypeForPrompt(query)

	// 查找最佳经验
	best := m.expMemory.GetBestExperience(taskType)
	if best == nil || best.UsageCount < 1 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("📌 相关经验:\n")

	// 写入经验提示
	sb.WriteString(fmt.Sprintf("   任务类型: %s\n", best.TaskType))
	sb.WriteString(fmt.Sprintf("   推荐工具序列: %s\n", strings.Join(best.ToolSequence, " → ")))
	if best.Hint != "" {
		// 截取提示的前 200 字符，避免过长
		hint := best.Hint
		if len(hint) > 200 {
			hint = hint[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("   提示: %s\n", hint))
	}
	sb.WriteString(fmt.Sprintf("   成功率: %.0f%% (已使用 %d 次)",
		best.SuccessRate*100, best.UsageCount))

	return sb.String()
}

// ═══════════════════════════════════════════════════════════════
// Layer 5.2: 环境上下文注入
// 根据查询中提到的服务器，注入相关环境信息
// ═══════════════════════════════════════════════════════════════
func (m *MemoryInjector) buildEnvironmentContext(query string) string {
	if m.envMemory == nil {
		return ""
	}

	var sb strings.Builder
	hasContent := false

	// 从查询中提取可能的服务器地址
	mentionedHosts := extractHostsFromQuery(query)

	// 如果有提到具体服务器，注入该服务器的环境信息
	if len(mentionedHosts) > 0 {
		for _, host := range mentionedHosts {
			server := m.envMemory.GetServer(host)
			if server != nil {
				if !hasContent {
					sb.WriteString("🖥️ 已知服务器环境:\n")
					hasContent = true
				}
				sb.WriteString(fmt.Sprintf("   • %s@%s", server.DefaultUser, server.Host))
				if server.OS != "" {
					sb.WriteString(fmt.Sprintf(" (%s)", server.OS))
				}
				if len(server.Tags) > 0 {
					sb.WriteString(fmt.Sprintf(" [%s]", strings.Join(server.Tags, ", ")))
				}
				// 注入常用路径
				if len(server.CommonPaths) > 0 {
					sb.WriteString(fmt.Sprintf(" — 常用路径: %s", strings.Join(server.CommonPaths[:min(3, len(server.CommonPaths))], ", ")))
				}
				sb.WriteString("\n")
			}
		}
	}

	// 如果没有提到具体服务器，显示最近使用的 3 台
	if !hasContent {
		recent := m.envMemory.GetRecentServers(3)
		if len(recent) > 0 {
			sb.WriteString("🖥️ 最近使用的服务器:\n")
			for _, s := range recent {
				sb.WriteString(fmt.Sprintf("   • %s@%s", s.DefaultUser, s.Host))
				if s.OS != "" {
					sb.WriteString(fmt.Sprintf(" (%s)", s.OS))
				}
				sb.WriteString("\n")
			}
			hasContent = true
		}
	}

	// 注入用户自定义提示
	if hasContent {
		taskType := classifyTaskTypeForPrompt(query)
		customHint := m.envMemory.GetCustomHint(taskType)
		if customHint != "" {
			sb.WriteString(fmt.Sprintf("\n💡 自定义提示: %s", customHint))
		}
	}

	if !hasContent {
		return ""
	}

	return sb.String()
}

// ═══════════════════════════════════════════════════════════════
// Layer 5.3: 用户偏好注入
// 将用户的安全模式、超时等偏好注入 Prompt
// ═══════════════════════════════════════════════════════════════
func (m *MemoryInjector) buildUserPreferences() string {
	if m.envMemory == nil {
		return ""
	}

	mode := m.envMemory.GetUserPreference("safety_mode")
	if mode == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("⚙️ 当前设置:\n")
	sb.WriteString(fmt.Sprintf("   安全模式: %s", mode))

	// 超时设置
	timeout := m.envMemory.GetUserPreference("timeout")
	if timeout != "" && timeout != "60" {
		sb.WriteString(fmt.Sprintf(" | 超时: %ss", timeout))
	}

	// sudo 偏好
	if m.envMemory.GetUserPreference("prefer_sudo") == "true" {
		sb.WriteString(" | sudo: 默认启用")
	}

	return sb.String()
}

// ═══════════════════════════════════════════════════════════════
// BuildObservationMessage 构建工具结果 Observation 消息
// 优化格式，让 LLM 更容易理解工具返回的内容
// ═══════════════════════════════════════════════════════════════
func BuildObservationMessage(toolName string, result *evolver.ToolCallRecord) string {
	var sb strings.Builder

	if result.Success {
		sb.WriteString(fmt.Sprintf("[%s] ✅ 执行成功 (耗时 %.1fs)\n", toolName, result.Duration.Seconds()))
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")

		// 截断过长输出
		output := result.Output
		if len(output) > 3000 {
			lines := strings.Split(output, "\n")
			if len(lines) > 30 {
				output = strings.Join(lines[:20], "\n")
				output += fmt.Sprintf("\n... (%d 行已省略) ...", len(lines)-20)
			} else {
				output = output[:3000] + "\n... [输出已截断]"
			}
		}
		sb.WriteString(output)
		sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━")
	} else {
		sb.WriteString(fmt.Sprintf("[%s] ❌ 执行失败 (耗时 %.1fs)\n", toolName, result.Duration.Seconds()))
		sb.WriteString(fmt.Sprintf("⚠️ 错误: %s\n", result.Output))
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("建议: 检查权限/路径/服务状态，或尝试用更基础的方式重新执行")
	}

	return sb.String()
}

// ═══════════════════════════════════════════════════════════════
// 辅助函数
// ═══════════════════════════════════════════════════════════════

// classifyTaskTypeForPrompt 根据查询内容分类任务类型（供 Prompt 使用）
// 与 evolver/engine.go 中的 classifyTaskType 保持一致
func classifyTaskTypeForPrompt(query string) string {
	q := strings.ToLower(query)

	if strings.Contains(q, "磁盘") || strings.Contains(q, "df") ||
		strings.Contains(q, "空间") || strings.Contains(q, "disk") ||
		strings.Contains(q, "du") || strings.Contains(q, "满") {
		return "磁盘分析"
	}
	if strings.Contains(q, "内存") || strings.Contains(q, "memory") ||
		strings.Contains(q, "mem") || strings.Contains(q, "free") ||
		strings.Contains(q, "ram") || strings.Contains(q, "oom") {
		return "内存分析"
	}
	if strings.Contains(q, "cpu") || strings.Contains(q, "负载") ||
		strings.Contains(q, "load") || strings.Contains(q, "top") ||
		strings.Contains(q, "进程") || strings.Contains(q, "process") {
		return "CPU分析"
	}
	if strings.Contains(q, "网络") || strings.Contains(q, "端口") ||
		strings.Contains(q, "net") || strings.Contains(q, "ping") ||
		strings.Contains(q, "连接") || strings.Contains(q, "port") ||
		strings.Contains(q, "tcp") || strings.Contains(q, "http") {
		return "网络诊断"
	}
	if strings.Contains(q, "日志") || strings.Contains(q, "log") ||
		strings.Contains(q, "tail") || strings.Contains(q, "journal") {
		return "日志分析"
	}
	if strings.Contains(q, "服务") || strings.Contains(q, "service") ||
		strings.Contains(q, "systemctl") || strings.Contains(q, "nginx") ||
		strings.Contains(q, "mysql") || strings.Contains(q, "redis") {
		return "服务管理"
	}
	if strings.Contains(q, "文件") || strings.Contains(q, "find") ||
		strings.Contains(q, "grep") || strings.Contains(q, "awk") ||
		strings.Contains(q, "sed") {
		return "文件操作"
	}
	if strings.Contains(q, "ssh") || strings.Contains(q, "远程") ||
		strings.Contains(q, "remote") {
		return "远程操作"
	}
	return "通用运维"
}

// extractHostsFromQuery 从查询中提取可能的服务器地址
// 支持的格式: IP地址 (192.168.1.100)、user@host、纯域名
func extractHostsFromQuery(query string) []string {
	mentioned := make([]string, 0)
	words := strings.FieldsFunc(query, func(r rune) bool {
		return r == ' ' || r == ',' || r == '；' || r == '"' || r == '\''
	})

	for _, word := range words {
		word = strings.TrimSpace(word)
		// 匹配 IP 地址模式
		if isIPAddress(word) || strings.Contains(word, "@") {
			mentioned = append(mentioned, word)
		}
	}

	return mentioned
}

// isIPAddress 简单判断是否为 IP 地址
func isIPAddress(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err != nil {
			return false
		}
		if n < 0 || n > 255 {
			return false
		}
	}
	return true
}


