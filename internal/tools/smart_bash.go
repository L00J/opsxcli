package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SmartBashTool 智能Bash工具(带命令分析)
type SmartBashTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	analyzer    *BashCommandAnalyzer
}

// NewSmartBashTool 创建智能bash工具
func NewSmartBashTool() *SmartBashTool {
	return &SmartBashTool{
		name:        "bash",
		description: "执行Shell命令或脚本(智能风险评估)",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "要执行的shell命令或脚本内容",
				},
			},
			"required": []string{"command"},
		},
		analyzer: NewBashCommandAnalyzer(),
	}
}

func (t *SmartBashTool) Name() string {
	return t.name
}

func (t *SmartBashTool) Description() string {
	return t.description
}

func (t *SmartBashTool) Parameters() map[string]interface{} {
	return t.parameters
}

func (t *SmartBashTool) RiskLevel() RiskLevel {
	// 默认风险等级,实际会在执行时动态分析
	return RiskMedium
}

// GetDynamicRiskLevel 获取动态风险等级(基于命令内容)
func (t *SmartBashTool) GetDynamicRiskLevel(args map[string]interface{}) RiskLevel {
	command := parseStringParam(args, "command")
	return t.analyzer.AnalyzeRisk(command)
}

// GetRiskDescription 获取风险描述
func (t *SmartBashTool) GetRiskDescription(args map[string]interface{}) string {
	command := parseStringParam(args, "command")
	return t.analyzer.GetRiskDescription(command)
}

func (t *SmartBashTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	command := parseStringParam(args, "command")
	if command == "" {
		return &ToolResult{
			Success: false,
			Error:   "command 参数不能为空",
		}, fmt.Errorf("command 参数不能为空")
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	// 设置超时
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd = exec.CommandContext(timeoutCtx, "bash", "-c", command)

	// 执行命令
	output, err := cmd.CombinedOutput()

	if err != nil {
		return &ToolResult{
			Success: false,
			Output:  string(output),
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  strings.TrimSpace(string(output)),
	}, nil
}
