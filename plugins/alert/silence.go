package alert

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SilenceManager 管理告警静默规则。
type SilenceManager struct {
	mu       sync.RWMutex
	filePath string
	rules    []*SilenceRule
}

// NewSilenceManager 创建静默规则管理器。
func NewSilenceManager(dataDir string) (*SilenceManager, error) {
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户目录失败: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".opsxcli", "alert")
	}

	sm := &SilenceManager{
		filePath: filepath.Join(dataDir, "silences.json"),
		rules:    make([]*SilenceRule, 0),
	}

	// 加载已有静默规则
	if err := sm.load(); err != nil {
		// 文件不存在是正常的
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("加载静默规则失败: %w", err)
		}
	}

	return sm, nil
}

// Add 添加静默规则。
func (sm *SilenceManager) Add(rule *SilenceRule) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if rule.Reason == "" {
		return fmt.Errorf("静默原因不能为空")
	}
	if rule.Until.IsZero() {
		return fmt.Errorf("静默截止时间不能为空")
	}
	if rule.Until.Before(time.Now()) {
		return fmt.Errorf("静默截止时间不能早于当前时间")
	}
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("silence-%d", time.Now().UnixNano())
	}
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}

	sm.rules = append(sm.rules, rule)
	return sm.save()
}

// Remove 移除静默规则。
func (sm *SilenceManager) Remove(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, r := range sm.rules {
		if r.ID == id {
			sm.rules = append(sm.rules[:i], sm.rules[i+1:]...)
			return sm.save()
		}
	}

	return fmt.Errorf("静默规则 %q 不存在", id)
}

// List 列出所有活跃的静默规则。
func (sm *SilenceManager) List() []*SilenceRule {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sm.cleanExpired()

	result := make([]*SilenceRule, len(sm.rules))
	copy(result, sm.rules)
	return result
}

// IsSilenced 检查给定告警是否被静默。
func (sm *SilenceManager) IsSilenced(alert *Alert) (bool, *SilenceRule) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sm.cleanExpired()

	for _, rule := range sm.rules {
		if sm.matchSilence(alert, rule) {
			return true, rule
		}
	}
	return false, nil
}

// matchSilence 检查告警是否匹配静默规则。
func (sm *SilenceManager) matchSilence(alert *Alert, rule *SilenceRule) bool {
	// 匹配规则名称
	if rule.MatchName != "" {
		if !matchPattern(rule.MatchName, alert.RuleName) {
			return false
		}
	}

	// 匹配级别
	if rule.MatchLevel != "" && rule.MatchLevel != alert.Level {
		return false
	}

	// 匹配标签
	if len(rule.MatchLabels) > 0 {
		for k, v := range rule.MatchLabels {
			if alert.Labels == nil || alert.Labels[k] != v {
				return false
			}
		}
	}

	// 没有匹配条件时匹配所有
	if rule.MatchName == "" && rule.MatchLevel == "" && len(rule.MatchLabels) == 0 {
		return false // 空规则不匹配任何告警
	}

	return true
}

// matchPattern 简单通配符匹配。
func matchPattern(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == s {
		return true
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(s, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, pattern[:len(pattern)-1])
	}
	return pattern == s
}

// cleanExpired 清理过期的静默规则。
func (sm *SilenceManager) cleanExpired() {
	now := time.Now()
	var active []*SilenceRule
	for _, r := range sm.rules {
		if r.Until.After(now) {
			active = append(active, r)
		}
	}
	sm.rules = active
}

// load 从文件加载静默规则。
func (sm *SilenceManager) load() error {
	data, err := os.ReadFile(sm.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &sm.rules)
}

// save 保存静默规则到文件。
func (sm *SilenceManager) save() error {
	dir := filepath.Dir(sm.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(sm.rules, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化静默规则失败: %w", err)
	}

	return os.WriteFile(sm.filePath, data, 0644)
}

// --- 升级策略管理 ---

// EscalationManager 管理告警升级策略。
type EscalationManager struct {
	mu       sync.RWMutex
	policies []*EscalationPolicy
	router   *Router
}

// NewEscalationManager 创建告警升级管理器。
func NewEscalationManager(router *Router) *EscalationManager {
	return &EscalationManager{
		policies: make([]*EscalationPolicy, 0),
		router:   router,
	}
}

// SetPolicies 设置升级策略列表。
func (em *EscalationManager) SetPolicies(policies []*EscalationPolicy) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.policies = policies
}

// CheckEscalation 检查告警是否需要升级。
func (em *EscalationManager) CheckEscalation(alert *Alert, now time.Time) (*EscalationPolicy, bool) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if alert.Escalated {
		return nil, false
	}

	for _, policy := range em.policies {
		if policy.Level != alert.Level {
			continue
		}

		duration, err := ParseDuration(policy.AfterDuration)
		if err != nil {
			continue
		}

		if now.Sub(alert.FiredAt) >= duration {
			return policy, true
		}
	}

	return nil, false
}

// Escalate 执行告警升级。
func (em *EscalationManager) Escalate(alert *Alert, policy *EscalationPolicy) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	// 升级告警级别
	alert.Level = policy.EscalateTo
	alert.Escalated = true

	// 通过升级策略中的渠道发送通知
	rule := &Rule{
		Name:     alert.RuleName + " (升级)",
		Channels: policy.NotifyChannels,
		Targets:  policy.NotifyTargets,
		Labels:   alert.Labels,
	}

	return em.router.SendAlert(alert, rule)
}

// DefaultPolicies 返回默认升级策略。
func DefaultPolicies() []*EscalationPolicy {
	return []*EscalationPolicy{
		{
			Level:          LevelWarning,
			AfterDuration:  "10m",
			EscalateTo:     LevelError,
			NotifyChannels: []string{"feishu"},
		},
		{
			Level:          LevelError,
			AfterDuration:  "5m",
			EscalateTo:     LevelCritical,
			NotifyChannels: []string{"dingtalk", "feishu"},
		},
	}
}
