package notify

import (
	"fmt"
	"strings"
)

// Message represents a notification message.
type Message struct {
	Title   string   // Message title
	Content string   // Message body content
	Level   string   // Severity level: info, warning, error, critical
	AtUsers []string // Users to mention/at
}

// Notifier is the unified interface for all notification backends.
type Notifier interface {
	Send(msg *Message) error
	Name() string
}

// NotifyOptions groups the parameters needed to send a notification.
type NotifyOptions struct {
	Target  string   // URL or identifier for the notification target
	Type    string   // Notifier type: webhook, feishu, dingtalk
	Message *Message // The message to send
	Secret  string   // Optional signing secret
}

// NewNotifier is a factory function that creates a Notifier based on type.
func NewNotifier(notifierType string, target string) (Notifier, error) {
	switch strings.ToLower(notifierType) {
	case "webhook":
		return NewWebhookNotifier(target)
	case "feishu":
		return NewFeishuNotifier(target)
	case "dingtalk":
		return NewDingtalkNotifier(target)
	default:
		return nil, fmt.Errorf("unsupported notifier type: %s", notifierType)
	}
}

// ValidLevels returns the list of valid message levels.
func ValidLevels() []string {
	return []string{"info", "warning", "error", "critical"}
}

// IsValidLevel checks if a given level string is valid.
func IsValidLevel(level string) bool {
	switch strings.ToLower(level) {
	case "info", "warning", "error", "critical":
		return true
	default:
		return false
	}
}
