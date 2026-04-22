package alert

import (
	"fmt"
	"sync"
	"time"

	"opsxcli/plugins/notify"
)

// Router 负责将告警路由到对应的通知渠道。
type Router struct {
	mu       sync.RWMutex
	notifiers map[string]notify.Notifier // 缓存已创建的 Notifier 实例
}

// NewRouter 创建告警路由器。
func NewRouter() *Router {
	return &Router{
		notifiers: make(map[string]notify.Notifier),
	}
}

// RouteConfig 定义级别到渠道的路由配置。
type RouteConfig struct {
	Level    Level    `yaml:"level" json:"level"`       // 告警级别
	Channels []string `yaml:"channels" json:"channels"` // 通知渠道列表
	Targets  []string `yaml:"targets" json:"targets"`   // 通知目标 URL 列表
	Secrets  []string `yaml:"secrets" json:"secrets"`   // 签名密钥列表
}

// RoutePolicy 定义完整的路由策略。
type RoutePolicy struct {
	Routes []*RouteConfig `yaml:"routes" json:"routes"` // 级别路由配置
}

// SendAlert 将告警发送到指定规则的通知渠道。
func (r *Router) SendAlert(alert *Alert, rule *Rule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	sentCount := 0

	for i, channel := range rule.Channels {
		target := ""
		if i < len(rule.Targets) {
			target = rule.Targets[i]
		}
		secret := ""
		if i < len(rule.Secrets) {
			secret = rule.Secrets[i]
		}

		if target == "" {
			lastErr = fmt.Errorf("渠道 %s 缺少目标 URL", channel)
			continue
		}

		err := r.sendToChannel(channel, target, secret, alert, rule)
		if err != nil {
			lastErr = fmt.Errorf("发送到 %s 失败: %w", channel, err)
			continue
		}
		sentCount++
	}

	alert.NotifyCount += sentCount

	if sentCount == 0 && lastErr != nil {
		return lastErr
	}
	if sentCount == 0 {
		return fmt.Errorf("没有可用的通知渠道")
	}

	return nil
}

// SendWithPolicy 使用路由策略发送告警。
func (r *Router) SendWithPolicy(alert *Alert, policy *RoutePolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, route := range policy.Routes {
		if route.Level != alert.Level {
			continue
		}

		for i, channel := range route.Channels {
			target := ""
			if i < len(route.Targets) {
				target = route.Targets[i]
			}
			secret := ""
			if i < len(route.Secrets) {
				secret = route.Secrets[i]
			}
			if target == "" {
				continue
			}
			// 忽略单个渠道的错误，继续发送到其他渠道
			_ = r.sendToChannel(channel, target, secret, alert, nil)
		}
	}

	return nil
}

// sendToChannel 发送告警到指定渠道。
func (r *Router) sendToChannel(channel, target, secret string, alert *Alert, rule *Rule) error {
	// 获取或创建 Notifier
	key := channel + ":" + target
	notifier, ok := r.notifiers[key]
	if !ok {
		var err error
		notifier, err = notify.NewNotifier(channel, target)
		if err != nil {
			return fmt.Errorf("创建 %s 通知器失败: %w", channel, err)
		}

		// 设置签名密钥
		if secret != "" {
			switch n := notifier.(type) {
			case interface{ SetSecret(string) }:
				n.SetSecret(secret)
			}
		}
		r.notifiers[key] = notifier
	}

	// 构建消息
	title := fmt.Sprintf("告警 [%s] %s", alert.Level, alert.RuleName)
	content := buildAlertContent(alert, rule)

	msg := &notify.Message{
		Title:   title,
		Content: content,
		Level:   string(alert.Level),
	}

	return notifier.Send(msg)
}

// buildAlertContent 构建告警消息正文。
func buildAlertContent(alert *Alert, rule *Rule) string {
	content := fmt.Sprintf("**告警名称**: %s\n", alert.RuleName)
	if rule != nil && rule.Description != "" {
		content += fmt.Sprintf("**描述**: %s\n", rule.Description)
	}
	content += fmt.Sprintf("**级别**: %s\n", alert.Level)
	content += fmt.Sprintf("**状态**: %s\n", alert.State)
	content += fmt.Sprintf("**当前值**: %.1f (阈值: %.1f)\n", alert.Value, alert.Threshold)
	content += fmt.Sprintf("**触发时间**: %s\n", alert.FiredAt.Format("2006-01-02 15:04:05"))

	if alert.ResolvedAt != nil {
		content += fmt.Sprintf("**恢复时间**: %s\n", alert.ResolvedAt.Format("2006-01-02 15:04:05"))
	}

	if alert.SilencedAt != nil {
		content += fmt.Sprintf("**静默至**: %s\n", alert.SilenceUntil.Format("2006-01-02 15:04:05"))
	}

	if rule != nil && len(rule.Labels) > 0 {
		content += "**标签**: "
		first := true
		for k, v := range rule.Labels {
			if !first {
				content += ", "
			}
			content += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
		content += "\n"
	}

	return content
}

// FormatCheckResult 将检查结果格式化为可读文本。
func FormatCheckResult(result *CheckResult) string {
	status := "✅ 正常"
	if !result.Pass {
		status = "❌ 异常"
	}
	return fmt.Sprintf("%s %s (当前: %.1f, 阈值: %.1f)", status, result.RuleName, result.Value, result.Threshold)
}

// FormatAlert 格式化告警信息。
func FormatAlert(alert *Alert) string {
	s := fmt.Sprintf("[%s] %s - %s (值: %.1f, 阈值: %.1f)",
		alert.Level, alert.RuleName, alert.State, alert.Value, alert.Threshold)
	if alert.FiredAt.After(time.Time{}) {
		s += fmt.Sprintf(" @ %s", alert.FiredAt.Format("15:04:05"))
	}
	return s
}
