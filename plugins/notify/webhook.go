package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookNotifier sends notifications via generic webhook POST.
type WebhookNotifier struct {
	targetURL string
	client    *http.Client
}

// NewWebhookNotifier creates a new generic webhook notifier.
func NewWebhookNotifier(targetURL string) (*WebhookNotifier, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("webhook target URL is required")
	}
	return &WebhookNotifier{
		targetURL: targetURL,
		client:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// webhookPayload is the JSON body sent to the webhook endpoint.
type webhookPayload struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Level   string `json:"level"`
}

// Name returns the notifier name.
func (w *WebhookNotifier) Name() string {
	return "webhook"
}

// Send posts the message as JSON to the webhook URL.
func (w *WebhookNotifier) Send(msg *Message) error {
	payload := webhookPayload{
		Title:   msg.Title,
		Content: msg.Content,
		Level:   msg.Level,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, w.targetURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d", resp.StatusCode)
	}

	return nil
}
