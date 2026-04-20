package notify

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- notify.go tests ---

func TestValidLevels(t *testing.T) {
	levels := ValidLevels()
	if len(levels) != 4 {
		t.Fatalf("ValidLevels() returned %d levels, want 4", len(levels))
	}
	expected := []string{"info", "warning", "error", "critical"}
	for i, l := range expected {
		if levels[i] != l {
			t.Errorf("levels[%d] = %q, want %q", i, levels[i], l)
		}
	}
}

func TestIsValidLevel(t *testing.T) {
	tests := []struct {
		level string
		want  bool
	}{
		{"info", true},
		{"warning", true},
		{"error", true},
		{"critical", true},
		{"INFO", true},
		{"Warning", true},
		{"unknown", false},
		{"", false},
		{"debug", false},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			if got := IsValidLevel(tt.level); got != tt.want {
				t.Errorf("IsValidLevel(%q) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

func TestNewNotifier(t *testing.T) {
	tests := []struct {
		notifierType string
		wantName     string
		wantErr      string
	}{
		{"webhook", "webhook", ""},
		{"feishu", "feishu", ""},
		{"dingtalk", "dingtalk", ""},
		{"Webhook", "webhook", ""}, // case insensitive
		{"FEISHU", "feishu", ""},
		{"DingTalk", "dingtalk", ""},
		{"slack", "", "unsupported notifier type"},
		{"", "", "unsupported notifier type"},
	}

	for _, tt := range tests {
		t.Run(tt.notifierType, func(t *testing.T) {
			n, err := NewNotifier(tt.notifierType, "https://example.com/hook")
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", n.Name(), tt.wantName)
			}
		})
	}
}

// --- dingtalk.go tests ---

func TestNewDingtalkNotifier(t *testing.T) {
	t.Run("empty URL", func(t *testing.T) {
		_, err := NewDingtalkNotifier("")
		if err == nil {
			t.Fatal("expected error for empty URL")
		}
		if !strings.Contains(err.Error(), "required") {
			t.Errorf("error = %q, want to contain 'required'", err.Error())
		}
	})

	t.Run("valid URL", func(t *testing.T) {
		n, err := NewDingtalkNotifier("https://oapi.dingtalk.com/robot/send?access_token=test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n.Name() != "dingtalk" {
			t.Errorf("Name() = %q, want %q", n.Name(), "dingtalk")
		}
	})
}

func TestDingtalkNotifier_GenerateSign(t *testing.T) {
	n, _ := NewDingtalkNotifier("https://example.com")
	// Without secret
	if sign := n.generateSign(1234567890); sign != "" {
		t.Errorf("generateSign without secret = %q, want empty", sign)
	}

	// With secret
	n.SetSecret("test-secret")
	sign := n.generateSign(1234567890)
	if sign == "" {
		t.Error("generateSign with secret returned empty string")
	}
	// Sign should be deterministic for same timestamp + secret
	sign2 := n.generateSign(1234567890)
	if sign != sign2 {
		t.Errorf("generateSign not deterministic: %q != %q", sign, sign2)
	}
	// Different timestamp should give different sign
	sign3 := n.generateSign(1234567891)
	if sign == sign3 {
		t.Error("generateSign should differ for different timestamps")
	}
}

func TestDingtalkNotifier_BuildURL(t *testing.T) {
	baseURL := "https://oapi.dingtalk.com/robot/send?access_token=abc123"
	n, _ := NewDingtalkNotifier(baseURL)

	// Without secret, should return base URL
	if got := n.buildURL(); got != baseURL {
		t.Errorf("buildURL without secret = %q, want %q", got, baseURL)
	}

	// With secret, should append timestamp and sign
	n.SetSecret("mysecret")
	got := n.buildURL()
	if !strings.Contains(got, "timestamp=") {
		t.Errorf("buildURL with secret missing timestamp: %q", got)
	}
	if !strings.Contains(got, "sign=") {
		t.Errorf("buildURL with secret missing sign: %q", got)
	}
}

func TestDingtalkNotifier_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n, err := NewDingtalkNotifier(server.URL)
	if err != nil {
		t.Fatalf("NewDingtalkNotifier: %v", err)
	}
	// Override client to use test server
	n.client = server.Client()

	msg := &Message{
		Title:   "Test Alert",
		Content: "This is a test notification",
		Level:   "info",
	}
	if err := n.Send(msg); err != nil {
		t.Errorf("Send() error: %v", err)
	}
}

func TestDingtalkNotifier_SendWithAt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n, _ := NewDingtalkNotifier(server.URL)
	n.client = server.Client()

	msg := &Message{
		Title:   "Urgent",
		Content: "Check this",
		Level:   "critical",
		AtUsers: []string{"13800138000", "13900139000"},
	}
	if err := n.Send(msg); err != nil {
		t.Errorf("Send() with @users error: %v", err)
	}
}

func TestDingtalkNotifier_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	n, _ := NewDingtalkNotifier(server.URL)
	n.client = server.Client()

	msg := &Message{Title: "Test", Content: "fail", Level: "info"}
	if err := n.Send(msg); err == nil {
		t.Error("expected error for 500 status")
	}
}

// --- feishu.go tests ---

func TestNewFeishuNotifier(t *testing.T) {
	t.Run("empty URL", func(t *testing.T) {
		_, err := NewFeishuNotifier("")
		if err == nil {
			t.Fatal("expected error for empty URL")
		}
	})

	t.Run("valid URL", func(t *testing.T) {
		n, err := NewFeishuNotifier("https://open.feishu.cn/open-apis/bot/v2/hook/test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n.Name() != "feishu" {
			t.Errorf("Name() = %q, want %q", n.Name(), "feishu")
		}
	})
}

func TestLevelToColor(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{"info", "blue"},
		{"warning", "orange"},
		{"error", "red"},
		{"critical", "red"},
		{"INFO", "blue"},
		{"unknown", "blue"},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			if got := levelToColor(tt.level); got != tt.want {
				t.Errorf("levelToColor(%q) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestFeishuNotifier_GenerateSign(t *testing.T) {
	n, _ := NewFeishuNotifier("https://example.com")
	if sign := n.generateSign(1234567890); sign != "" {
		t.Errorf("generateSign without secret = %q, want empty", sign)
	}

	n.SetSecret("test-secret")
	sign := n.generateSign(1234567890)
	if sign == "" {
		t.Error("generateSign with secret returned empty string")
	}
	// Deterministic
	sign2 := n.generateSign(1234567890)
	if sign != sign2 {
		t.Errorf("generateSign not deterministic: %q != %q", sign, sign2)
	}
}

func TestFeishuNotifier_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n, _ := NewFeishuNotifier(server.URL)
	n.client = server.Client()

	msg := &Message{
		Title:   "Deploy Success",
		Content: "Version 1.2.3 deployed",
		Level:   "info",
	}
	if err := n.Send(msg); err != nil {
		t.Errorf("Send() error: %v", err)
	}
}

func TestFeishuNotifier_SendWithAt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n, _ := NewFeishuNotifier(server.URL)
	n.client = server.Client()

	msg := &Message{
		Title:   "Alert",
		Content: "Something wrong",
		Level:   "error",
		AtUsers: []string{"user1", "user2"},
	}
	if err := n.Send(msg); err != nil {
		t.Errorf("Send() with @users error: %v", err)
	}
}

func TestFeishuNotifier_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	n, _ := NewFeishuNotifier(server.URL)
	n.client = server.Client()

	msg := &Message{Title: "Test", Content: "fail", Level: "info"}
	if err := n.Send(msg); err == nil {
		t.Error("expected error for 403 status")
	}
}

// --- webhook.go tests ---

func TestNewWebhookNotifier(t *testing.T) {
	t.Run("empty URL", func(t *testing.T) {
		_, err := NewWebhookNotifier("")
		if err == nil {
			t.Fatal("expected error for empty URL")
		}
		if !strings.Contains(err.Error(), "required") {
			t.Errorf("error = %q, want 'required'", err.Error())
		}
	})

	t.Run("valid URL", func(t *testing.T) {
		n, err := NewWebhookNotifier("https://hooks.example.com/test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n.Name() != "webhook" {
			t.Errorf("Name() = %q, want %q", n.Name(), "webhook")
		}
	})
}

func TestWebhookNotifier_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n, _ := NewWebhookNotifier(server.URL)
	n.client = server.Client()

	msg := &Message{
		Title:   "Webhook Test",
		Content: "Testing webhook notification",
		Level:   "info",
	}
	if err := n.Send(msg); err != nil {
		t.Errorf("Send() error: %v", err)
	}
}

func TestWebhookNotifier_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	n, _ := NewWebhookNotifier(server.URL)
	n.client = server.Client()

	msg := &Message{Title: "Test", Content: "fail", Level: "info"}
	if err := n.Send(msg); err == nil {
		t.Error("expected error for 502 status")
	}
}
