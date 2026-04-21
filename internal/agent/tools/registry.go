// registry.go - 工具注册中心
// 由另一个代理实现具体工具，本文件仅提供接口和注册中心框架
package tools

import (
	"context"
	"fmt"
	"sync"

	"opsxcli/internal/llm"
)

// RiskLevel 工具风险等级
type RiskLevel int

const (
	RiskSafe     RiskLevel = iota // 只读操作（cat, grep, df, ps）
	RiskLow                       // 低风险（mkdir, touch）
	RiskMedium                    // 中风险（chmod, 文件修改）
	RiskHigh                      // 高风险（ssh_execute 远程操作, 服务控制）
	RiskCritical                  // 危险操作（rm -rf, mkfs, dd, 数据删除）
)

// String 返回风险等级的字符串表示
func (r RiskLevel) String() string {
	switch r {
	case RiskSafe:
		return "safe"
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	case RiskCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Result 工具执行结果
type Result struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// Tool V2 核心工具接口
type Tool interface {
	Name() string                       // 工具名称
	Description() string                // 工具描述（给 LLM 看的）
	Parameters() map[string]interface{} // JSON Schema 参数定义
	RiskLevel() RiskLevel               // 静态风险等级
	Execute(ctx context.Context, args map[string]interface{}) (*Result, error)
}

// Closer 可关闭的工具接口
// 支持资源释放的工具可实现此接口，Registry.Close() 会自动调用
type Closer interface {
	Close() error
}

// DynamicRiskTool 动态风险评估接口（v0.5.0 风险审批集成）
// 工具可额外实现此接口，根据参数动态计算风险等级
// 安全控制器在 Check 时会优先使用此接口而非静态 RiskLevel()
type DynamicRiskTool interface {
	Tool
	RiskLevelForArgs(args map[string]interface{}) RiskLevel
}

// Registry 工具注册中心
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry 创建新的工具注册中心
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register 注册工具
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Get 根据名称获取工具
func (r *Registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("工具未找到: %s", name)
	}
	return tool, nil
}

// List 列出所有已注册工具
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		list = append(list, tool)
	}
	return list
}

// ToLLMTools 将注册的工具转换为 LLM 工具格式
func (r *Registry) ToLLMTools() []llm.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	llmTools := make([]llm.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		llmTool := llm.Tool{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
			},
		}
		llmTools = append(llmTools, llmTool)
	}
	return llmTools
}

// RegisterDefaults 注册默认工具集合
// v0.5.0+: 注册统一工具（execute/transfer）+ 保留旧工具（向后兼容）
// LLM 优先使用 execute（替代 local_bash/ssh_execute）和 transfer（替代 scp_transfer）
func (r *Registry) RegisterDefaults() {
	// v0.5.0 统一工具（LLM 优先使用）
	r.Register(NewUnifiedExecuteTool())
	r.Register(NewUnifiedTransferTool())

	// 保留旧工具（向后兼容，LLM 不再主动使用）
	r.Register(NewLocalBashTool())
	r.Register(NewSSHExecuteTool())
	r.Register(NewSCPTransferTool())

	// 分析和文件工具
	r.Register(NewAnalyzeOutputTool())
	r.Register(NewFileReadTool())
	r.Register(NewFileSearchTool())
}

// Close 关闭注册表中所有支持关闭的工具
// 遍历所有已注册工具，如果实现了 Closer 接口则调用其 Close 方法
// 用于 Agent 退出时释放连接池等资源
func (r *Registry) Close() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, tool := range r.tools {
		if closer, ok := tool.(Closer); ok {
			closer.Close()
		}
	}
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

// parseNumberParam 解析数值参数（int 或 float64）
func parseNumberParam(args map[string]interface{}, key string) (int, bool) {
	if v, ok := args[key]; ok {
		switch val := v.(type) {
		case int:
			return val, true
		case float64:
			return int(val), true
		case int64:
			return int(val), true
		}
	}
	return 0, false
}

// parseFloat64Param 解析浮点数参数
func parseFloat64Param(args map[string]interface{}, key string) (float64, bool) {
	if v, ok := args[key]; ok {
		switch val := v.(type) {
		case float64:
			return val, true
		case int:
			return float64(val), true
		case int64:
			return float64(val), true
		}
	}
	return 0, false
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
