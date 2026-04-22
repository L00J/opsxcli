package alert

import (
	"fmt"
	"sync"
	"time"
)

// Scheduler 管理定时规则检查调度。
type Scheduler struct {
	mu          sync.RWMutex
	checker     Checker
	router      *Router
	history     *HistoryManager
	silence     *SilenceManager
	escalation  *EscalationManager
	rules       []Rule
	activeAlerts map[string]*Alert // ruleName -> Alert
	lastCheck   map[string]time.Time
	stopCh      chan struct{}
	running     bool
}

// SchedulerOption 配置调度器选项。
type SchedulerOption func(*Scheduler)

// WithChecker 设置自定义检查器。
func WithChecker(c Checker) SchedulerOption {
	return func(s *Scheduler) { s.checker = c }
}

// WithHistory 设置历史管理器。
func WithHistory(h *HistoryManager) SchedulerOption {
	return func(s *Scheduler) { s.history = h }
}

// WithSilence 设置静默管理器。
func WithSilence(sm *SilenceManager) SchedulerOption {
	return func(s *Scheduler) { s.silence = sm }
}

// WithEscalation 设置升级管理器。
func WithEscalation(em *EscalationManager) SchedulerOption {
	return func(s *Scheduler) { s.escalation = em }
}

// NewScheduler 创建告警调度器。
func NewScheduler(router *Router, opts ...SchedulerOption) *Scheduler {
	s := &Scheduler{
		checker:      NewSystemChecker(),
		router:       router,
		activeAlerts: make(map[string]*Alert),
		lastCheck:    make(map[string]time.Time),
		stopCh:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// SetRules 设置要监控的规则列表。
func (s *Scheduler) SetRules(rules []Rule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = rules
}

// RunOnce 执行一次所有启用的规则检查。
func (s *Scheduler) RunOnce() ([]*CheckResult, []error) {
	s.mu.RLock()
	rules := make([]Rule, len(s.rules))
	copy(rules, s.rules)
	s.mu.RUnlock()

	enabled := FilterRules(rules)
	results := make([]*CheckResult, 0, len(enabled))
	var errs []error

	now := time.Now()

	for i := range enabled {
		rule := &enabled[i]

		// 检查是否到了检查时间
		interval, err := ParseInterval(rule.Interval)
		if err != nil {
			errs = append(errs, fmt.Errorf("规则 %q 间隔无效: %w", rule.Name, err))
			continue
		}

		if lastTime, ok := s.lastCheck[rule.Name]; ok {
			if now.Sub(lastTime) < interval {
				continue // 还没到检查时间
			}
		}

		// 执行检查
		result, err := CheckRule(s.checker, rule)
		if err != nil {
			errs = append(errs, fmt.Errorf("规则 %q 检查失败: %w", rule.Name, err))
			continue
		}

		s.lastCheck[rule.Name] = now
		results = append(results, result)

		// 处理检查结果
		s.processResult(result, rule, now)
	}

	return results, errs
}

// Start 启动调度器的主循环。
func (s *Scheduler) Start(interval time.Duration) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("调度器已在运行")
	}
	s.running = true
	s.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.RunOnce()
			s.checkEscalations()
		case <-s.stopCh:
			return nil
		}
	}
}

// Stop 停止调度器。
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

// ActiveAlerts 返回当前活跃的告警列表。
func (s *Scheduler) ActiveAlerts() []*Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alerts := make([]*Alert, 0, len(s.activeAlerts))
	for _, a := range s.activeAlerts {
		alerts = append(alerts, a)
	}
	return alerts
}

// GetAlert 获取指定规则的活跃告警。
func (s *Scheduler) GetAlert(ruleName string) (*Alert, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.activeAlerts[ruleName]
	return a, ok
}

// Stats 获取告警统计。
func (s *Scheduler) Stats() *Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &Stats{
		TotalRules:   len(s.rules),
		ActiveAlerts: len(s.activeAlerts),
		ByLevel:      make(map[Level]int),
	}

	for _, alert := range s.activeAlerts {
		switch alert.State {
		case StateFiring:
			stats.FiringAlerts++
		case StateSilenced:
			stats.SilencedAlerts++
		}
		stats.ByLevel[alert.Level]++
	}

	return stats
}

// processResult 处理检查结果，管理告警生命周期。
func (s *Scheduler) processResult(result *CheckResult, rule *Rule, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if result.Pass {
		// 检查通过，如果之前有活跃告警则恢复
		if alert, ok := s.activeAlerts[rule.Name]; ok && alert.State == StateFiring {
			alert.State = StateResolved
			alert.ResolvedAt = &now

			// 记录恢复历史
			if s.history != nil {
				s.history.Record(&HistoryEntry{
					AlertID:   alert.ID,
					RuleName:  alert.RuleName,
					Level:     alert.Level,
					State:     StateResolved,
					Message:   fmt.Sprintf("告警恢复: %s", alert.Message),
					Value:     result.Value,
					Threshold: result.Threshold,
					Timestamp: now,
				})
			}

			// 从活跃列表移除
			delete(s.activeAlerts, rule.Name)
		}
		return
	}

	// 检查未通过
	alert, ok := s.activeAlerts[rule.Name]
	if !ok {
		// 新告警
		metric, _, threshold, _ := ParseCheckExpr(rule.Check)
		_ = metric

		alert = &Alert{
			ID:        fmt.Sprintf("alert-%d", now.UnixNano()),
			RuleName:  rule.Name,
			Level:     rule.Level,
			State:     StateFiring,
			Value:     result.Value,
			Threshold: threshold,
			Message:   result.Message,
			Labels:    rule.Labels,
			FiredAt:   now,
		}
		s.activeAlerts[rule.Name] = alert
	} else {
		// 更新现有告警
		alert.Value = result.Value
		alert.Message = result.Message
	}

	// 检查是否需要静默
	if s.silence != nil {
		if silenced, silenceRule := s.silence.IsSilenced(alert); silenced {
			alert.State = StateSilenced
			alert.SilencedAt = &now
			alert.SilenceUntil = &silenceRule.Until

			// 记录静默历史
			if s.history != nil {
				s.history.Record(&HistoryEntry{
					AlertID:   alert.ID,
					RuleName:  alert.RuleName,
					Level:     alert.Level,
					State:     StateSilenced,
					Message:   fmt.Sprintf("告警静默: %s", silenceRule.Reason),
					Timestamp: now,
				})
			}
			return
		}
	}

	// 发送告警通知
	if alert.State == StateFiring && alert.NotifyCount == 0 {
		// 仅首次触发时发送通知
		if err := s.router.SendAlert(alert, rule); err != nil {
			// 记录发送失败，但不阻塞
			_ = err
		}
	}

	// 记录触发历史
	if s.history != nil {
		s.history.Record(&HistoryEntry{
			AlertID:   alert.ID,
			RuleName:  alert.RuleName,
			Level:     alert.Level,
			State:     alert.State,
			Message:   result.Message,
			Value:     result.Value,
			Threshold: result.Threshold,
			Timestamp: now,
		})
	}
}

// checkEscalations 检查所有活跃告警是否需要升级。
func (s *Scheduler) checkEscalations() {
	if s.escalation == nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	for _, alert := range s.activeAlerts {
		if alert.State != StateFiring {
			continue
		}

		policy, shouldEscalate := s.escalation.CheckEscalation(alert, now)
		if shouldEscalate && policy != nil {
			if err := s.escalation.Escalate(alert, policy); err != nil {
				_ = err
			}
		}
	}
}
