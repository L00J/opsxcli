package alert

import (
	"time"
)

// Level 表示告警级别。
type Level string

const (
	LevelInfo     Level = "info"
	LevelWarning  Level = "warning"
	LevelError    Level = "error"
	LevelCritical Level = "critical"
)

// State 表示告警状态的当前阶段。
type State string

const (
	StatePending  State = "pending"  // 等待中（首次触发但未确认）
	StateFiring   State = "firing"   // 告警中（正在触发）
	StateResolved State = "resolved" // 已恢复
	StateSilenced State = "silenced" // 已静默
)

// Rule 定义一条告警规则。
type Rule struct {
	Name        string   `yaml:"name" json:"name"`                 // 规则名称（唯一标识）
	Description string   `yaml:"description" json:"description"`   // 规则描述
	Check       string   `yaml:"check" json:"check"`               // 检查表达式，如 "cpu_percent > 80"
	Interval    string   `yaml:"interval" json:"interval"`         // 检查间隔（cron 表达式或简单间隔如 "5m"）
	For         string   `yaml:"for" json:"for"`                   // 持续时长（触发条件满足多久后才告警），如 "2m"
	Level       Level    `yaml:"level" json:"level"`               // 告警级别
	Channels    []string `yaml:"channels" json:"channels"`         // 通知渠道: feishu, dingtalk, webhook
	Targets     []string `yaml:"targets" json:"targets"`           // 各渠道对应的 URL
	Secrets     []string `yaml:"secrets" json:"secrets,omitempty"` // 各渠道对应的签名密钥
	Labels      map[string]string `yaml:"labels" json:"labels,omitempty"` // 自定义标签
	Enabled     bool     `yaml:"enabled" json:"enabled"`           // 是否启用
}

// RuleConfig 定义告警规则配置文件格式。
type RuleConfig struct {
	Version string `yaml:"version" json:"version"` // 配置版本
	Rules   []Rule `yaml:"rules" json:"rules"`     // 告警规则列表
}

// Alert 表示一次告警实例。
type Alert struct {
	ID          string            `json:"id"`           // 唯一 ID
	RuleName    string            `json:"rule_name"`    // 关联规则名称
	Level       Level             `json:"level"`        // 告警级别
	State       State             `json:"state"`        // 告警状态
	Value       float64           `json:"value"`        // 检查值
	Threshold   float64           `json:"threshold"`    // 阈值
	Message     string            `json:"message"`      // 告警消息
	Labels      map[string]string `json:"labels"`       // 规则标签
	FiredAt     time.Time         `json:"fired_at"`     // 触发时间
	ResolvedAt  *time.Time        `json:"resolved_at"`  // 恢复时间
	SilencedAt  *time.Time        `json:"silenced_at"`  // 静默时间
	SilenceUntil *time.Time       `json:"silence_until"` // 静默截止时间
	Acknowledged bool             `json:"acknowledged"` // 是否已确认
	Escalated   bool             `json:"escalated"`    // 是否已升级
	NotifyCount int              `json:"notify_count"` // 通知次数
}

// CheckResult 表示一次规则检查的结果。
type CheckResult struct {
	RuleName  string    `json:"rule_name"`
	Pass      bool      `json:"pass"`       // true=通过（未触发），false=未通过（触发）
	Value     float64   `json:"value"`      // 实际值
	Threshold float64   `json:"threshold"`  // 阈值
	Message   string    `json:"message"`    // 检查结果描述
	CheckedAt time.Time `json:"checked_at"` // 检查时间
}

// SilenceRule 定义告警静默规则。
type SilenceRule struct {
	ID         string            `json:"id"`
	MatchName  string            `json:"match_name,omitempty"`  // 匹配规则名称（支持通配符）
	MatchLevel Level             `json:"match_level,omitempty"` // 匹配级别
	MatchLabels map[string]string `json:"match_labels,omitempty"` // 匹配标签
	Reason     string            `json:"reason"`                // 静默原因
	CreatedAt  time.Time         `json:"created_at"`            // 创建时间
	Until      time.Time         `json:"until"`                 // 静默截止时间
	CreatedBy  string            `json:"created_by,omitempty"`  // 创建者
}

// EscalationPolicy 定义告警升级策略。
type EscalationPolicy struct {
	Level          Level   `json:"level"`           // 触发升级的原始级别
	AfterDuration  string  `json:"after_duration"`  // 持续多久后升级，如 "10m"
	EscalateTo     Level   `json:"escalate_to"`     // 升级到的级别
	NotifyChannels []string `json:"notify_channels"` // 升级后通知渠道
	NotifyTargets  []string `json:"notify_targets"`  // 升级后通知目标
}

// HistoryEntry 表示告警历史记录条目。
type HistoryEntry struct {
	AlertID   string    `json:"alert_id"`
	RuleName  string    `json:"rule_name"`
	Level     Level     `json:"level"`
	State     State     `json:"state"`
	Message   string    `json:"message"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Timestamp time.Time `json:"timestamp"`
}

// Stats 表示告警统计信息。
type Stats struct {
	TotalRules    int            `json:"total_rules"`
	ActiveAlerts  int            `json:"active_alerts"`
	FiringAlerts  int            `json:"firing_alerts"`
	SilencedAlerts int           `json:"silenced_alerts"`
	ByLevel       map[Level]int  `json:"by_level"`
	LastCheckTime *time.Time     `json:"last_check_time"`
}
