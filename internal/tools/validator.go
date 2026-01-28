package tools

import (
	"fmt"
	"regexp"
	"strings"
)

// ToolCallValidator 工具调用验证器 - 引导优先使用 opsxcli 工具
type ToolCallValidator struct {
	// 内置工具列表 - 这些工具有专用实现,建议优先使用
	builtinTools map[string]bool
}

// NewToolCallValidator 创建工具调用验证器
func NewToolCallValidator() *ToolCallValidator {
	return &ToolCallValidator{
		builtinTools: map[string]bool{
			"kubectl":     true,
			"docker":      true,
			"redis":       true,
			"mysql":       true,
			"psql":        true,
			"sys_monitor": true,
			"net_monitor": true,
		},
	}
}

// ValidateToolCall 验证工具调用是否符合规范
// 返回: (是否合规, 错误消息, 建议)
// 新策略: 不拦截,只给建议,允许 bash fallback
func (v *ToolCallValidator) ValidateToolCall(toolName string, args map[string]interface{}) (bool, string, string) {
	// 新策略: 优先级引导,不是禁止
	if toolName == "bash" {
		command, ok := args["command"].(string)
		if !ok {
			return true, "", "" // 没有命令,允许通过
		}

		// 检查命令中是否包含内置工具关键字
		forbiddenTools := map[string]string{
			"kubectl":   "kubectl",
			"docker":    "docker",
			"redis-cli": "redis",
			"mysql":     "mysql",
			"psql":      "psql",
		}

		for keyword, toolName := range forbiddenTools {
			if strings.Contains(command, keyword) {
				// 检查是否是白名单场景
				if v.isWhitelistedUsage(command, keyword) {
					continue // 允许这种用法
				}

				// 不拒绝执行,只返回建议
				suggestion := fmt.Sprintf(
					"opsxcli 提供了 %s 专用工具,建议优先使用以获得更好的体验。"+
						"如果专用工具不可用或失败,可以使用 bash 作为 fallback。",
					toolName,
				)
				return true, "", suggestion // 允许通过,但有建议
			}
		}
	}

	// 鼓励使用内置工具
	if v.builtinTools[toolName] {
		// 这是正确的用法,鼓励
		return true, "", ""
	}

	return true, "", ""
}

// isWhitelistedUsage 检查是否是允许的使用场景
func (v *ToolCallValidator) isWhitelistedUsage(command, keyword string) bool {
	// 允许检查工具是否存在
	whitelistPatterns := []string{
		"which " + keyword,
		"command -v " + keyword,
		"type " + keyword,
		"whereis " + keyword,
	}

	commandLower := strings.ToLower(strings.TrimSpace(command))
	for _, pattern := range whitelistPatterns {
		if strings.HasPrefix(commandLower, pattern) {
			return true
		}
	}

	// 允许 echo 输出文本(但不允许命令替换)
	if strings.HasPrefix(commandLower, "echo") && !strings.Contains(command, "$(") && !strings.Contains(command, "`") {
		return true
	}

	return false
}

// extractToolName 从命令中提取工具名
func (v *ToolCallValidator) extractToolName(command string) string {
	command = strings.TrimSpace(command)

	// 移除管道前缀
	if idx := strings.Index(command, "|"); idx >= 0 {
		command = command[idx+1:]
		command = strings.TrimSpace(command)
	}

	// 提取第一个词
	parts := strings.Fields(command)
	if len(parts) > 0 {
		return parts[0]
	}

	return "unknown"
}

// AutoCorrect 自动修正不合规的工具调用
// 如果可以修正,返回修正后的 (toolName, args, true)
// 如果无法修正,返回 ("", nil, false)
func (v *ToolCallValidator) AutoCorrect(toolName string, args map[string]interface{}) (string, map[string]interface{}, bool) {
	if toolName != "bash" {
		return "", nil, false
	}

	command, ok := args["command"].(string)
	if !ok {
		return "", nil, false
	}

	command = strings.TrimSpace(command)

	// 检测复杂特征 - 使用正则精确匹配,避免误判
	// 例如: -o 不应该被当作重定向 >

	// 1. 逻辑运算符 (&&, ||)
	if strings.Contains(command, "&&") || strings.Contains(command, "||") {
		return "", nil, false
	}

	// 2. 管道 (但要精确检测,避免误判)
	if regexp.MustCompile(`[^|]\|[^|]`).MatchString(command) {
		// 单个 | 是管道,|| 是逻辑或
		return "", nil, false
	}

	// 3. 重定向 (精确检测: 空格+重定向符号)
	if regexp.MustCompile(`\s+>`).MatchString(command) ||
		regexp.MustCompile(`\s+<`).MatchString(command) ||
		regexp.MustCompile(`\s+2>&1`).MatchString(command) {
		return "", nil, false
	}

	// 4. 命令替换
	if strings.Contains(command, "$(") || strings.Contains(command, "`") {
		return "", nil, false
	}

	// 尝试修正 bash kubectl → kubectl
	// 只修正最简单的场景: 行首就是 kubectl,且只有一个 kubectl 调用
	if strings.Contains(command, "kubectl") {
		// 确保只有一个 kubectl 调用
		if strings.Count(command, "kubectl") > 1 {
			return "", nil, false
		}

		// 提取纯粹的 kubectl 命令
		if matched := regexp.MustCompile(`^\s*kubectl\s+(.+)$`).FindStringSubmatch(command); matched != nil {
			subcommand := strings.TrimSpace(matched[1])

			// 最后检查:subcommand 不应该包含 kubectl
			if strings.Contains(subcommand, "kubectl") {
				return "", nil, false
			}

			return "kubectl", map[string]interface{}{
				"subcommand": subcommand,
			}, true
		}

		return "", nil, false
	}

	// 尝试修正 bash docker → docker
	if strings.Contains(command, "docker") {
		if strings.Count(command, "docker") > 1 {
			return "", nil, false
		}

		if matched := regexp.MustCompile(`^\s*docker\s+(.+)$`).FindStringSubmatch(command); matched != nil {
			subcommand := strings.TrimSpace(matched[1])

			if strings.Contains(subcommand, "docker") {
				return "", nil, false
			}

			return "docker", map[string]interface{}{
				"subcommand": subcommand,
			}, true
		}

		return "", nil, false
	}

	// 其他工具暂不支持自动修正
	return "", nil, false
}

// GetViolationStats 获取违规统计 (用于监控和改进)
type ViolationStats struct {
	TotalCalls     int            // 总调用次数
	Violations     int            // 违规次数
	AutoCorrected  int            // 自动修正次数
	ViolationTypes map[string]int // 按类型统计违规
}

var globalStats = &ViolationStats{
	ViolationTypes: make(map[string]int),
}

// RecordViolation 记录违规
func (v *ToolCallValidator) RecordViolation(toolName string, violationType string) {
	globalStats.TotalCalls++
	globalStats.Violations++
	globalStats.ViolationTypes[violationType]++
}

// RecordCorrection 记录自动修正
func (v *ToolCallValidator) RecordCorrection() {
	globalStats.AutoCorrected++
}

// GetStats 获取统计信息
func (v *ToolCallValidator) GetStats() *ViolationStats {
	return globalStats
}
