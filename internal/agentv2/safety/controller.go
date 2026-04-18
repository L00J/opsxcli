// controller.go - 安全控制器
// 提供工具执行前的安全检查、风险确认和审计记录功能
package safety

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"opsxcli/internal/agentv2/tools"
)

// SafetyMode 安全模式
type SafetyMode string

const (
	SafetyModeStrict     SafetyMode = "strict"     // 严格模式：medium+ 需要确认
	SafetyModeBalanced   SafetyMode = "balanced"   // 平衡模式：high+ 需要确认（默认）
	SafetyModePermissive SafetyMode = "permissive" // 宽松模式：只有 critical 需要确认
)

// Controller 安全控制器
type Controller struct {
	mode        SafetyMode      // 当前安全模式
	autoApprove bool            // 自动批准（仅测试使用）
	reader      *bufio.Reader   // 标准输入读取器
	history     []ExecutionRecord // 执行历史记录
}

// ExecutionRecord 单次工具执行记录
type ExecutionRecord struct {
	ToolName  string                 // 工具名称
	Args      map[string]interface{} // 调用参数
	RiskLevel tools.RiskLevel        // 风险等级
	Approved  bool                   // 是否已批准
	Timestamp time.Time              // 记录时间
}

// NewController 创建安全控制器
func NewController(mode SafetyMode) *Controller {
	return &Controller{
		mode:        mode,
		autoApprove: false,
		reader:      bufio.NewReader(os.Stdin),
		history:     make([]ExecutionRecord, 0),
	}
}

// SetAutoApprove 设置自动批准（危险，仅用于自动化测试）
func (c *Controller) SetAutoApprove(auto bool) {
	c.autoApprove = auto
}

// GetMode 获取当前安全模式
func (c *Controller) GetMode() SafetyMode {
	return c.mode
}

// Check 检查工具执行是否需要用户确认
// 根据工具的风险等级和当前安全模式决定是否拦截
// 返回 (approved bool, err error)
func (c *Controller) Check(tool tools.Tool, args map[string]interface{}) (bool, error) {
	riskLevel := tool.RiskLevel()

	// 创建执行记录
	record := ExecutionRecord{
		ToolName:  tool.Name(),
		Args:      args,
		RiskLevel: riskLevel,
		Timestamp: time.Now(),
	}

	// 判断是否需要用户确认
	needsConfirm := c.needsConfirmation(riskLevel)

	// 不需要确认或已开启自动批准
	if !needsConfirm || c.autoApprove {
		record.Approved = true
		c.history = append(c.history, record)
		return true, nil
	}

	// 请求用户交互式确认
	approved := c.requestConfirmation(tool, args, riskLevel)
	record.Approved = approved
	c.history = append(c.history, record)
	return approved, nil
}

// MarkExecuted 标记工具已执行完成
func (c *Controller) MarkExecuted(toolName string, args map[string]interface{}, result *tools.Result) {
	// 更新最后一条匹配的执行记录
	for i := len(c.history) - 1; i >= 0; i-- {
		if c.history[i].ToolName == toolName {
			c.history[i].Approved = result.Success
			return
		}
	}
}

// GetHistory 获取执行历史（返回副本防止外部修改）
func (c *Controller) GetHistory() []ExecutionRecord {
	result := make([]ExecutionRecord, len(c.history))
	copy(result, c.history)
	return result
}

// needsConfirmation 根据安全模式判断是否需要确认
func (c *Controller) needsConfirmation(risk tools.RiskLevel) bool {
	switch c.mode {
	case SafetyModeStrict:
		// 严格模式：medium 及以上均需确认
		return risk >= tools.RiskMedium
	case SafetyModeBalanced:
		// 平衡模式：high 及以上需确认
		return risk >= tools.RiskHigh
	case SafetyModePermissive:
		// 宽松模式：仅 critical 需确认
		return risk >= tools.RiskCritical
	default:
		return risk >= tools.RiskHigh
	}
}

// requestConfirmation 交互式请求用户确认
// 显示工具信息、风险等级和参数，等待用户输入 yes/no
func (c *Controller) requestConfirmation(tool tools.Tool, args map[string]interface{}, risk tools.RiskLevel) bool {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("⚠️  安全确认请求")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  工具: %s\n", tool.Name())
	fmt.Printf("  描述: %s\n", tool.Description())
	fmt.Printf("  风险: %s\n", formatRiskDisplay(risk))
	fmt.Printf("  参数: %v\n", formatArgs(args))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	for {
		fmt.Print("是否继续执行? (yes/no): ")
		input, err := c.reader.ReadString('\n')
		if err != nil {
			fmt.Printf("读取输入失败: %v，默认拒绝\n", err)
			return false
		}

		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "yes", "y":
			return true
		case "no", "n":
			return false
		default:
			fmt.Println("请输入 yes 或 no")
		}
	}
}

// formatRiskDisplay 格式化风险等级显示
func formatRiskDisplay(risk tools.RiskLevel) string {
	switch risk {
	case tools.RiskSafe:
		return "🟢 安全（只读操作）"
	case tools.RiskLow:
		return "🟡 低风险"
	case tools.RiskMedium:
		return "🟠 中风险（文件操作）"
	case tools.RiskHigh:
		return "🔴 高风险（远程/系统操作）"
	case tools.RiskCritical:
		return "💀 危险（删除/格式化）"
	default:
		return "❓ 未知"
	}
}

// formatArgs 格式化参数显示
func formatArgs(args map[string]interface{}) string {
	if len(args) == 0 {
		return "{}"
	}

	parts := make([]string, 0, len(args))
	for k, v := range args {
		switch val := v.(type) {
		case string:
			if len(val) > 100 {
				val = val[:100] + "..."
			}
			parts = append(parts, fmt.Sprintf("%s=%q", k, val))
		default:
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}
