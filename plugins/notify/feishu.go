package notify

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FeishuNotifier sends notifications via Feishu (Lark) custom bot webhook.
type FeishuNotifier struct {
	targetURL string
	secret    string
	client    *http.Client
}

// NewFeishuNotifier creates a new Feishu notifier.
func NewFeishuNotifier(targetURL string) (*FeishuNotifier, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("feishu webhook URL is required")
	}
	return &FeishuNotifier{
		targetURL: targetURL,
		client:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// SetSecret sets the signing secret for the Feishu bot.
func (f *FeishuNotifier) SetSecret(secret string) {
	f.secret = secret
}

// Name returns the notifier name.
func (f *FeishuNotifier) Name() string {
	return "feishu"
}

// levelToColor maps message levels to Feishu card template colors.
func levelToColor(level string) string {
	switch strings.ToLower(level) {
	case "info":
		return "blue"
	case "warning":
		return "orange"
	case "error":
		return "red"
	case "critical":
		return "red"
	default:
		return "blue"
	}
}

// feishuCard represents the Feishu interactive card message.
type feishuCard struct {
	MsgType string `json:"msg_type"`
	Card    struct {
		Header  feishuCardHeader  `json:"header"`
		Elements []feishuCardElement `json:"elements"`
	} `json:"card"`
}

type feishuCardHeader struct {
	Title    feishuText `json:"title"`
	Template string     `json:"template"`
}

type feishuText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type feishuCardElement struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// generateSign generates the Feishu webhook signature.
// timestamp + "\n" + secret, then HMAC-SHA256 and base64 encode.
func (f *FeishuNotifier) generateSign(timestamp int64) string {
	if f.secret == "" {
		return ""
	}
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, f.secret)
	mac := hmac.New(sha256.New, []byte(f.secret))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// buildURL returns the webhook URL, optionally appending sign parameters.
func (f *FeishuNotifier) buildURL() string {
	if f.secret == "" {
		return f.targetURL
	}
	timestamp := time.Now().Unix()
	sign := f.generateSign(timestamp)
	u, err := url.Parse(f.targetURL)
	if err != nil {
		return f.targetURL
	}
	q := u.Query()
	q.Set("timestamp", fmt.Sprintf("%d", timestamp))
	q.Set("sign", sign)
	u.RawQuery = q.Encode()
	return u.String()
}

// Send posts a card message to the Feishu webhook.
func (f *FeishuNotifier) Send(msg *Message) error {
	card := feishuCard{}
	card.MsgType = "interactive"
	card.Card.Header.Title = feishuText{Tag: "plain_text", Content: msg.Title}
	card.Card.Header.Template = levelToColor(msg.Level)

	// Build content, optionally appending @mentions
	content := msg.Content
	if len(msg.AtUsers) > 0 {
		var atParts []string
		for _, user := range msg.AtUsers {
			atParts = append(atParts, fmt.Sprintf(`<at user_id="%s"></at>`, user))
		}
		content = content + "\n" + strings.Join(atParts, " ")
	}

	card.Card.Elements = append(card.Card.Elements, feishuCardElement{
		Tag:     "markdown",
		Content: content,
	})

	body, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("failed to marshal feishu payload: %w", err)
	}

	reqURL := f.buildURL()
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create feishu request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("feishu request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("feishu returned non-success status: %d", resp.StatusCode)
	}

	return nil
}
