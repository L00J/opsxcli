package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"opsxcli/internal/db"
	"opsxcli/internal/tools"
)

// SafetyMode 安全模式
type SafetyMode string

const (
	SafetyModeStrict     SafetyMode = "strict"     // 严格模式：所有medium及以上需要确认
	SafetyModeBalanced   SafetyMode = "balanced"   // 平衡模式：high及以上需要确认
	SafetyModePermissive SafetyMode = "permissive" // 宽松模式：只有critical需要确认
)

// SafetyController 安全控制器
type SafetyController struct {
	mode          SafetyMode
	auditLogRepo  *db.AuditLogRepository
	userID        int64
	username      string
	autoApprove   bool
	confirmReader *bufio.Reader
}

// NewSafetyController 创建安全控制器
func NewSafetyController(mode SafetyMode, auditLogRepo *db.AuditLogRepository, userID int64, username string) *SafetyController {
	return &SafetyController{
		mode:          mode,
		auditLogRepo:  auditLogRepo,
		userID:        userID,
		username:      username,
		autoApprove:   false,
		confirmReader: bufio.NewReader(os.Stdin),
	}
}

// SetAutoApprove 设置自动批准（用于测试）
func (s *SafetyController) SetAutoApprove(auto bool) {
	s.autoApprove = auto
}

// CheckToolExecution 检查工具执行权限
func (s *SafetyController) CheckToolExecution(tool tools.Tool, args map[string]interface{}) (bool, *db.AuditLog, error) {
	// 获取风险等级（支持动态风险评估）
	var riskLevel tools.RiskLevel
	var riskDesc string

	// 如果是SmartBashTool,使用动态风险评估
	if smartBash, ok := tool.(*tools.SmartBashTool); ok {
		riskLevel = smartBash.GetDynamicRiskLevel(args)
		riskDesc = smartBash.GetRiskDescription(args)
	} else {
		riskLevel = tool.RiskLevel()
		riskDesc = getRiskDescription(riskLevel)
	}

	// 创建审计日志
	auditLog := &db.AuditLog{
		UserID:      s.userID,
		Username:    s.username,
		ActionType:  "tool_execution",
		Command:     tool.Name(),
		Arguments:   formatArgs(args),
		RiskLevel:   string(riskLevel),
		Approved:    false,
		Executed:    false,
		Environment: "cli",
	}

	// 保存审计日志
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		return false, nil, fmt.Errorf("创建审计日志失败: %w", err)
	}

	// 判断是否需要用户确认(使用动态风险等级)
	needsConfirm := s.needsConfirmation(riskLevel)

	if !needsConfirm {
		// 自动批准
		if err := s.auditLogRepo.MarkApproved(auditLog.ID, s.username); err != nil {
			return false, nil, fmt.Errorf("标记批准失败: %w", err)
		}
		auditLog.Approved = true
		return true, auditLog, nil
	}

	// 需要用户确认
	if s.autoApprove {
		// 自动批准（测试模式）
		if err := s.auditLogRepo.MarkApproved(auditLog.ID, s.username); err != nil {
			return false, nil, fmt.Errorf("标记批准失败: %w", err)
		}
		auditLog.Approved = true
		return true, auditLog, nil
	}

	// 请求用户确认(传递风险描述)
	approved := s.requestUserConfirmationWithDesc(tool, args, riskLevel, riskDesc)
	if approved {
		if err := s.auditLogRepo.MarkApproved(auditLog.ID, s.username); err != nil {
			return false, nil, fmt.Errorf("标记批准失败: %w", err)
		}
		auditLog.Approved = true
		return true, auditLog, nil
	}

	return false, auditLog, nil
}

// needsConfirmation 判断是否需要确认
func (s *SafetyController) needsConfirmation(riskLevel tools.RiskLevel) bool {
	switch s.mode {
	case SafetyModeStrict:
		// 严格模式：medium及以上需要确认
		return riskLevel == tools.RiskMedium || riskLevel == tools.RiskHigh || riskLevel == tools.RiskCritical

	case SafetyModeBalanced:
		// 平衡模式：high及以上需要确认
		return riskLevel == tools.RiskHigh || riskLevel == tools.RiskCritical

	case SafetyModePermissive:
		// 宽松模式：只有critical需要确认
		return riskLevel == tools.RiskCritical

	default:
		// 默认使用平衡模式
		return riskLevel == tools.RiskHigh || riskLevel == tools.RiskCritical
	}
}

// requestUserConfirmation 请求用户确认
func (s *SafetyController) requestUserConfirmation(tool tools.Tool, args map[string]interface{}) bool {
	fmt.Println()
	fmt.Println("⚠️  需要您的确认:")
	fmt.Printf("工具: %s\n", tool.Name())
	fmt.Printf("描述: %s\n", tool.Description())
	fmt.Printf("风险: %s\n", getRiskLevelDisplay(tool.RiskLevel()))
	fmt.Printf("参数: %s\n", formatArgs(args))
	fmt.Println()

	for {
		fmt.Print("是否继续执行? (yes/no): ")
		input, err := s.confirmReader.ReadString('\n')
		if err != nil {
			fmt.Printf("读取输入失败: %v\n", err)
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

// requestUserConfirmationWithDesc 请求用户确认(带自定义风险描述)
func (s *SafetyController) requestUserConfirmationWithDesc(tool tools.Tool, args map[string]interface{}, riskLevel tools.RiskLevel, riskDesc string) bool {
	fmt.Println()
	fmt.Println("⚠️  需要您的确认:")
	fmt.Printf("工具: %s\n", tool.Name())
	fmt.Printf("描述: %s\n", tool.Description())
	fmt.Printf("风险: %s (%s)\n", getRiskLevelDisplay(riskLevel), riskDesc)
	fmt.Printf("参数: %s\n", formatArgs(args))
	fmt.Println()

	for {
		fmt.Print("是否继续执行? (yes/no): ")
		input, err := s.confirmReader.ReadString('\n')
		if err != nil {
			fmt.Printf("读取输入失败: %v\n", err)
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

// MarkExecuted 标记工具已执行
func (s *SafetyController) MarkExecuted(auditLog *db.AuditLog, result *tools.ToolResult) error {
	return s.auditLogRepo.MarkExecuted(
		auditLog.ID,
		result.Success,
		result.Output,
		result.Error,
	)
}

// formatArgs 格式化参数
func formatArgs(args map[string]interface{}) string {
	if len(args) == 0 {
		return "{}"
	}

	parts := make([]string, 0, len(args))
	for k, v := range args {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

// getRiskLevelDisplay 获取风险等级显示
func getRiskLevelDisplay(level tools.RiskLevel) string {
	switch level {
	case tools.RiskSafe:
		return "🟢 安全（只读操作）"
	case tools.RiskLow:
		return "🟡 低风险（轻微修改）"
	case tools.RiskMedium:
		return "🟠 中风险（文件操作）"
	case tools.RiskHigh:
		return "🔴 高风险（系统配置）"
	case tools.RiskCritical:
		return "💀 危险操作（删除/格式化）"
	default:
		return "❓ 未知"
	}
}

// getRiskDescription 获取风险描述
func getRiskDescription(level tools.RiskLevel) string {
	switch level {
	case tools.RiskSafe:
		return "只读操作,安全"
	case tools.RiskLow:
		return "低风险操作"
	case tools.RiskMedium:
		return "中等风险,可能修改系统状态"
	case tools.RiskHigh:
		return "高风险,会修改文件或配置"
	case tools.RiskCritical:
		return "危险操作,可能导致系统不稳定或数据丢失"
	default:
		return "未知风险"
	}
}
