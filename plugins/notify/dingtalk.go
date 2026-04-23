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

// DingtalkNotifier sends notifications via DingTalk custom bot webhook.
type DingtalkNotifier struct {
	targetURL string
	secret    string
	client    *http.Client
}

// NewDingtalkNotifier creates a new DingTalk notifier.
func NewDingtalkNotifier(targetURL string) (*DingtalkNotifier, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("dingtalk webhook URL is required")
	}
	return &DingtalkNotifier{
		targetURL: targetURL,
		client:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// SetSecret sets the signing secret for the DingTalk bot.
func (d *DingtalkNotifier) SetSecret(secret string) {
	d.secret = secret
}

// Name returns the notifier name.
func (d *DingtalkNotifier) Name() string {
	return "dingtalk"
}

// dingtalkPayload represents the DingTalk markdown message.
type dingtalkPayload struct {
	MsgType  string           `json:"msgtype"`
	Markdown dingtalkMarkdown `json:"markdown"`
	At       *dingtalkAt      `json:"at,omitempty"`
}

type dingtalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type dingtalkAt struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
}

// generateSign generates the DingTalk webhook signature.
// timestamp + "\n" + secret, then HMAC-SHA256 and base64 encode.
func (d *DingtalkNotifier) generateSign(timestamp int64) string {
	if d.secret == "" {
		return ""
	}
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, d.secret)
	mac := hmac.New(sha256.New, []byte(d.secret))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// buildURL returns the webhook URL, optionally appending sign parameters.
func (d *DingtalkNotifier) buildURL() string {
	if d.secret == "" {
		return d.targetURL
	}
	timestamp := time.Now().UnixMilli()
	sign := d.generateSign(timestamp)
	u, err := url.Parse(d.targetURL)
	if err != nil {
		return d.targetURL
	}
	q := u.Query()
	q.Set("timestamp", fmt.Sprintf("%d", timestamp))
	q.Set("sign", sign)
	u.RawQuery = q.Encode()
	return u.String()
}

// Send posts a markdown message to the DingTalk webhook.
func (d *DingtalkNotifier) Send(msg *Message) error {
	payload := dingtalkPayload{
		MsgType: "markdown",
		Markdown: dingtalkMarkdown{
			Title: msg.Title,
			Text:  fmt.Sprintf("# %s\n\n%s", msg.Title, msg.Content),
		},
	}

	// Add @mentions
	if len(msg.AtUsers) > 0 {
		var atMobiles []string
		var atTexts []string
		for _, phone := range msg.AtUsers {
			atMobiles = append(atMobiles, phone)
			atTexts = append(atTexts, fmt.Sprintf("@%s", phone))
		}
		payload.At = &dingtalkAt{AtMobiles: atMobiles}
		// Append at mentions to markdown text
		payload.Markdown.Text += "\n\n" + strings.Join(atTexts, " ")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal dingtalk payload: %w", err)
	}

	reqURL := d.buildURL()
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create dingtalk request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("dingtalk request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("dingtalk returned non-success status: %d", resp.StatusCode)
	}

	return nil
}
