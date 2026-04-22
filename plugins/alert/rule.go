package alert

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// IsValidLevel 检查告警级别是否合法。
func IsValidLevel(level string) bool {
	switch Level(strings.ToLower(level)) {
	case LevelInfo, LevelWarning, LevelError, LevelCritical:
		return true
	default:
		return false
	}
}

// 支持的检查指标名称
var supportedMetrics = map[string]bool{
	"cpu_percent":    true,
	"memory_percent": true,
	"disk_percent":   true,
	"disk_used_gb":   true,
	"load1":          true,
	"load5":          true,
	"load15":         true,
	"process_count":  true,
	"uptime_seconds": true,
}

// 支持的比较运算符
var operatorPattern = regexp.MustCompile(`^\s*(\w+)\s*(>|>=|<|<=|==|!=)\s*([\d.]+)\s*$`)

// ParseCheckExpr 解析检查表达式，返回指标名、运算符、阈值。
// 表达式格式: "metric operator value"，如 "cpu_percent > 80"
func ParseCheckExpr(expr string) (metric string, operator string, threshold float64, err error) {
	matches := operatorPattern.FindStringSubmatch(expr)
	if matches == nil {
		return "", "", 0, fmt.Errorf("无法解析检查表达式: %q，格式应为 '指标 运算符 值'", expr)
	}

	metric = matches[1]
	operator = matches[2]

	if !supportedMetrics[metric] {
		return "", "", 0, fmt.Errorf("不支持的指标: %q，支持: %s", metric, strings.Join(supportedKeys(), ", "))
	}

	threshold, err = strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return "", "", 0, fmt.Errorf("无法解析阈值: %q: %w", matches[3], err)
	}

	return metric, operator, threshold, nil
}

// Evaluate 比较 value 和 threshold。
func Evaluate(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}

// ParseDuration 解析持续时间字符串，支持 Go 格式和简单格式。
// 支持格式: "5m", "1h", "30s", "2m30s"
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	return time.ParseDuration(s)
}

// ParseInterval 解析检查间隔，支持 cron 表达式和简单间隔。
// 简单格式: "5m", "1h", "30s" 直接转为 time.Duration
func ParseInterval(s string) (time.Duration, error) {
	if s == "" {
		return 5 * time.Minute, nil // 默认 5 分钟
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("无法解析间隔: %q: %w", s, err)
	}
	return d, nil
}

// ValidateRule 校验告警规则是否合法。
func ValidateRule(rule *Rule) error {
	if rule.Name == "" {
		return fmt.Errorf("规则名称不能为空")
	}

	metric, operator, threshold, err := ParseCheckExpr(rule.Check)
	if err != nil {
		return fmt.Errorf("规则 %q 检查表达式无效: %w", rule.Name, err)
	}
	_ = metric
	_ = threshold

	// 校验运算符
	validOps := map[string]bool{">": true, ">=": true, "<": true, "<=": true, "==": true, "!=": true}
	if !validOps[operator] {
		return fmt.Errorf("规则 %q 运算符无效: %q", rule.Name, operator)
	}

	// 校验级别
	if !IsValidLevel(string(rule.Level)) {
		return fmt.Errorf("规则 %q 级别无效: %q，支持: info, warning, error, critical", rule.Name, rule.Level)
	}

	// 校验间隔
	if rule.Interval != "" {
		if _, err := ParseInterval(rule.Interval); err != nil {
			return fmt.Errorf("规则 %q 间隔无效: %w", rule.Name, err)
		}
	}

	// 校验持续时长
	if rule.For != "" {
		if _, err := ParseDuration(rule.For); err != nil {
			return fmt.Errorf("规则 %q 持续时长无效: %w", rule.Name, err)
		}
	}

	// 校验通知渠道和目标
	if len(rule.Channels) == 0 {
		return fmt.Errorf("规则 %q 至少需要一个通知渠道", rule.Name)
	}
	if len(rule.Targets) == 0 {
		return fmt.Errorf("规则 %q 至少需要一个通知目标 URL", rule.Name)
	}

	validChannels := map[string]bool{"webhook": true, "feishu": true, "dingtalk": true}
	for _, ch := range rule.Channels {
		if !validChannels[ch] {
			return fmt.Errorf("规则 %q 渠道无效: %q，支持: webhook, feishu, dingtalk", rule.Name, ch)
		}
	}

	return nil
}

// LoadRules 从 YAML 文件加载告警规则配置。
func LoadRules(path string) (*RuleConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取规则文件失败: %w", err)
	}

	config := &RuleConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("解析规则文件失败: %w", err)
	}

	// 校验所有规则
	for i := range config.Rules {
		if err := ValidateRule(&config.Rules[i]); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// SaveRules 将告警规则配置保存到 YAML 文件。
func SaveRules(path string, config *RuleConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化规则失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入规则文件失败: %w", err)
	}
	return nil
}

// DefaultRuleConfig 返回默认的告警规则配置示例。
func DefaultRuleConfig() *RuleConfig {
	return &RuleConfig{
		Version: "1",
		Rules: []Rule{
			{
				Name:        "high_cpu",
				Description: "CPU 使用率过高",
				Check:       "cpu_percent > 80",
				Interval:    "5m",
				For:         "2m",
				Level:       LevelWarning,
				Channels:    []string{"feishu"},
				Targets:     []string{"https://open.feishu.cn/open-apis/bot/v2/hook/your-hook-id"},
				Enabled:     true,
			},
			{
				Name:        "disk_full",
				Description: "磁盘空间不足",
				Check:       "disk_percent > 90",
				Interval:    "10m",
				Level:       LevelCritical,
				Channels:    []string{"dingtalk"},
				Targets:     []string{"https://oapi.dingtalk.com/robot/send?access_token=your-token"},
				Enabled:     true,
			},
			{
				Name:        "high_memory",
				Description: "内存使用率过高",
				Check:       "memory_percent > 85",
				Interval:    "5m",
				For:         "3m",
				Level:       LevelError,
				Channels:    []string{"feishu", "dingtalk"},
				Targets:     []string{"https://open.feishu.cn/open-apis/bot/v2/hook/your-hook-id", "https://oapi.dingtalk.com/robot/send?access_token=your-token"},
				Enabled:     true,
			},
		},
	}
}

// FilterRules 返回启用的规则。
func FilterRules(rules []Rule) []Rule {
	var enabled []Rule
	for _, r := range rules {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	return enabled
}

// FindRule 根据名称查找规则。
func FindRule(rules []Rule, name string) (*Rule, bool) {
	for i := range rules {
		if rules[i].Name == name {
			return &rules[i], true
		}
	}
	return nil, false
}

// supportedKeys 返回支持的指标名列表。
func supportedKeys() []string {
	keys := make([]string, 0, len(supportedMetrics))
	for k := range supportedMetrics {
		keys = append(keys, k)
	}
	return keys
}
