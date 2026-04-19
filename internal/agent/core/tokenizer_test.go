package core

import (
	"strings"
	"testing"

	"opsxcli/internal/llm"
)

func TestNewTokenEstimator(t *testing.T) {
	t.Run("default value", func(t *testing.T) {
		e := NewTokenEstimator(0)
		if e.MaxContextTokens() != 6000 {
			t.Errorf("expected default 6000, got %d", e.MaxContextTokens())
		}
	})

	t.Run("custom value", func(t *testing.T) {
		e := NewTokenEstimator(8192)
		if e.MaxContextTokens() != 8192 {
			t.Errorf("expected 8192, got %d", e.MaxContextTokens())
		}
	})
}

func TestTokenEstimatorEstimate(t *testing.T) {
	est := NewTokenEstimator(6000)

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"empty", "", 0},
		{"纯中文", "你好世界", 4},
		{"纯中文长句", "今天天气不错适合出门散步", 12},
		{"纯英文单词", "hello world", 4},
		{"混合内容", "hello 世界", 4},
		{"连续数字", "12345", 3},
		{"连续标点", "!!!", 2},
		{"英文字母短", "abc", 1},
		{"英文字母长", "abcdefghijklmnopqrstuvwxyz", 7},
		{"日文假名", "こんにちは", 5},
		{"韩文", "안녕하세요", 5},
		{"空格分隔", "a b c d", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := est.Estimate(tt.text)
			if got != tt.expected {
				t.Errorf("Estimate(%q) = %d, want %d", tt.text, got, tt.expected)
			}
		})
	}
}

func TestTokenEstimatorEstimateMessages(t *testing.T) {
	est := NewTokenEstimator(6000)

	msgs := []llm.Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "hello world"},
		{Role: "assistant", Content: "你好世界"},
	}

	// system prompt: 13 letters -> 4 tokens
	// hello world: 4 tokens
	// 你好世界: 4 tokens
	// total = 12
	expected := 12
	got := est.EstimateMessages(msgs)
	if got != expected {
		t.Errorf("EstimateMessages() = %d, want %d", got, expected)
	}
}

func TestTrimMessagesWithinLimit(t *testing.T) {
	// 设置较小的上限以便测试
	config := &Config{MaxContextTokens: 200}
	agent := NewAgent(nil, nil, config, nil)

	// 生成少量短消息，token 数远小于上限
	msgs := generateMessages(10)
	result := agent.trimMessages(msgs)

	// 10 条短消息总 token 数很少，应保留全部
	if len(result) != 10 {
		t.Errorf("expected 10 messages, got %d", len(result))
	}
	if result[0].Role != "system" {
		t.Errorf("expected first message role system, got %s", result[0].Role)
	}
	if result[len(result)-1].Content != msgs[len(msgs)-1].Content {
		t.Errorf("expected last content %q, got %q", msgs[len(msgs)-1].Content, result[len(result)-1].Content)
	}
}

func TestTrimMessagesExceedsLimit(t *testing.T) {
	// 设置较小的上限以便测试
	config := &Config{MaxContextTokens: 200}
	agent := NewAgent(nil, nil, config, nil)

	// 每条长消息约 65 tokens（260 字母 / 4）
	longContent := strings.Repeat("abcdefghijklmnopqrstuvwxyz", 10)
	msgs := generateLongMessages(10, longContent)

	// system "system-prompt" = 12 字母 -> 3 tokens
	// max = 200 * 0.9 = 180
	// system(3) + 2*65 = 133 < 180
	// system(3) + 3*65 = 198 > 180
	// 应保留 system + 最近 2 条 = 3 条
	result := agent.trimMessages(msgs)
	if len(result) != 3 {
		t.Errorf("expected 3 messages, got %d", len(result))
	}
	if result[0].Role != "system" {
		t.Errorf("expected first message role system, got %s", result[0].Role)
	}
	if result[len(result)-1].Content != msgs[len(msgs)-1].Content {
		t.Errorf("expected last content to be the latest message")
	}
}

func TestTrimMessagesSingleMessageTruncation(t *testing.T) {
	// 设置较小的上限以便测试
	config := &Config{MaxContextTokens: 200}
	agent := NewAgent(nil, nil, config, nil)

	// 单条超长消息：800 字母 -> 200 tokens，超过 max-system=197
	veryLongContent := strings.Repeat("abcdefghij", 80)
	msgs := []llm.Message{
		{Role: "system", Content: "system-prompt"},
		{Role: "user", Content: veryLongContent},
	}

	result := agent.trimMessages(msgs)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Role != "system" {
		t.Errorf("expected first message role system, got %s", result[0].Role)
	}
	// 内容应被截断为原来的 80%
	originalRunes := []rune(veryLongContent)
	expectedRunes := int(float64(len(originalRunes)) * 0.8)
	resultRunes := []rune(result[1].Content)
	if len(resultRunes) != expectedRunes {
		t.Errorf("expected truncated content length %d runes, got %d", expectedRunes, len(resultRunes))
	}
}

func TestTrimMessagesFallbackWithoutTokenizer(t *testing.T) {
	agent := &Agent{}

	// 21 条消息在兜底策略下应保留 21 条（system + 20）
	msgs := generateMessages(21)
	result := agent.trimMessages(msgs)
	if len(result) != 21 {
		t.Errorf("expected 21 messages, got %d", len(result))
	}

	// 22 条消息在兜底策略下应裁剪为 21 条
	msgs = generateMessages(22)
	result = agent.trimMessages(msgs)
	if len(result) != 21 {
		t.Errorf("expected 21 messages, got %d", len(result))
	}
}

// generateLongMessages 生成指定数量的消息，第一条为 system，其余内容相同
func generateLongMessages(count int, content string) []llm.Message {
	msgs := make([]llm.Message, count)
	msgs[0] = llm.Message{Role: "system", Content: "system-prompt"}
	for i := 1; i < count; i++ {
		msgs[i] = llm.Message{Role: "user", Content: content}
	}
	return msgs
}
