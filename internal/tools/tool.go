package tools

import (
	"context"
	"fmt"
	"os/exec"
)

// RiskLevel 风险等级
type RiskLevel string

const (
	RiskSafe     RiskLevel = "safe"     // 安全操作（只读）
	RiskLow      RiskLevel = "low"      // 低风险（轻微修改）
	RiskMedium   RiskLevel = "medium"   // 中风险（文件操作）
	RiskHigh     RiskLevel = "high"     // 高风险（系统配置）
	RiskCritical RiskLevel = "critical" // 危险操作（删除、格式化）
)

// ToolResult 工具执行结果
type ToolResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// Tool 工具接口
type Tool interface {
	// Name 工具名称
	Name() string

	// Description 工具描述
	Description() string

	// Parameters 参数Schema（JSON Schema格式）
	Parameters() map[string]interface{}

	// RiskLevel 风险等级
	RiskLevel() RiskLevel

	// Execute 执行工具
	Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}

// CommandTool 基于命令行的工具
type CommandTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	riskLevel   RiskLevel
	buildCmd    func(args map[string]interface{}) (string, []string, error)
}

// NewCommandTool 创建命令行工具
func NewCommandTool(
	name, description string,
	parameters map[string]interface{},
	riskLevel RiskLevel,
	buildCmd func(args map[string]interface{}) (string, []string, error),
) *CommandTool {
	return &CommandTool{
		name:        name,
		description: description,
		parameters:  parameters,
		riskLevel:   riskLevel,
		buildCmd:    buildCmd,
	}
}

func (t *CommandTool) Name() string {
	return t.name
}

func (t *CommandTool) Description() string {
	return t.description
}

func (t *CommandTool) Parameters() map[string]interface{} {
	return t.parameters
}

func (t *CommandTool) RiskLevel() RiskLevel {
	return t.riskLevel
}

func (t *CommandTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	// 构建命令
	cmdName, cmdArgs, err := t.buildCmd(args)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// 执行命令
	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
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
		Output:  string(output),
	}, nil
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register 注册工具
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (Tool, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("工具 %s 不存在", name)
	}
	return tool, nil
}

// List 列出所有工具
func (r *ToolRegistry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListByRisk 按风险等级列出工具
func (r *ToolRegistry) ListByRisk(level RiskLevel) []Tool {
	tools := make([]Tool, 0)
	for _, tool := range r.tools {
		if tool.RiskLevel() == level {
			tools = append(tools, tool)
		}
	}
	return tools
}

// ToLLMTools 转换为LLM工具格式
func (r *ToolRegistry) ToLLMTools() []map[string]interface{} {
	llmTools := make([]map[string]interface{}, 0, len(r.tools))

	for _, tool := range r.tools {
		llmTools = append(llmTools, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters":  tool.Parameters(),
			},
		})
	}

	return llmTools
}

// parseStringParam 解析字符串参数
func parseStringParam(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// parseStringSliceParam 解析字符串数组参数
func parseStringSliceParam(args map[string]interface{}, key string) []string {
	if v, ok := args[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// parseBoolParam 解析布尔参数
func parseBoolParam(args map[string]interface{}, key string) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// parseIntParam 解析整数参数
func parseIntParam(args map[string]interface{}, key string) int {
	if v, ok := args[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		}
	}
	return 0
}
