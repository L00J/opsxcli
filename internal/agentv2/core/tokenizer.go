// tokenizer.go - 轻量级 Token 估算器
package core

import (
	"unicode"

	"opsxcli/internal/llm"
)

// TokenEstimator 轻量级 token 估算器
// 策略：
// - 中文字符（CJK）：1 字 ≈ 1 token（保守估算）
// - 英文单词/ASCII 字母：每 4 个字符 ≈ 1 token
// - 数字和标点符号：每 2 个字符 ≈ 1 token
// - 空白字符：不计 token
// 这不是精确 tokenizer，但足够用于裁剪决策
type TokenEstimator struct {
	maxContextTokens int // 上下文窗口上限
}

// NewTokenEstimator 创建 Token 估算器
func NewTokenEstimator(maxContextTokens int) *TokenEstimator {
	if maxContextTokens <= 0 {
		maxContextTokens = 6000
	}
	return &TokenEstimator{maxContextTokens: maxContextTokens}
}

// MaxContextTokens 返回上下文窗口上限
func (e *TokenEstimator) MaxContextTokens() int {
	return e.maxContextTokens
}

// Estimate 估算单条文本的 token 数量
func (e *TokenEstimator) Estimate(text string) int {
	if text == "" {
		return 0
	}

	tokens := 0
	var state runeType
	count := 0

	for _, r := range text {
		rt := classifyRune(r)
		if rt == rtSpace {
			if count > 0 {
				tokens += countTokens(state, count)
			}
			state = rtSpace
			count = 0
			continue
		}

		if rt != state {
			if count > 0 {
				tokens += countTokens(state, count)
			}
			state = rt
			count = 1
		} else {
			count++
		}
	}
	if count > 0 {
		tokens += countTokens(state, count)
	}

	return tokens
}

// EstimateMessages 估算消息列表的总 token 数量
func (e *TokenEstimator) EstimateMessages(msgs []llm.Message) int {
	total := 0
	for _, msg := range msgs {
		total += e.Estimate(msg.Content)
	}
	return total
}

// runeType 字符类型
type runeType int

const (
	rtCJK runeType = iota
	rtLetter
	rtDigit
	rtPunct
	rtSpace
	rtOther
)

// classifyRune 分类单个 rune
func classifyRune(r rune) runeType {
	if unicode.IsSpace(r) {
		return rtSpace
	}
	// CJK 统一表意文字（中文）
	if unicode.Is(unicode.Han, r) {
		return rtCJK
	}
	// 日文假名、韩文统一按 CJK 处理
	if (r >= '\u3040' && r <= '\u309F') || // Hiragana
		(r >= '\u30A0' && r <= '\u30FF') || // Katakana
		(r >= '\uAC00' && r <= '\uD7AF') { // Hangul
		return rtCJK
	}
	// ASCII 字母
	if unicode.IsLetter(r) && r < 128 {
		return rtLetter
	}
	// 数字
	if unicode.IsDigit(r) {
		return rtDigit
	}
	// 标点和符号
	if unicode.IsPunct(r) || unicode.IsSymbol(r) {
		return rtPunct
	}
	return rtOther
}

// countTokens 根据字符类型和连续数量计算 token 数
func countTokens(rt runeType, count int) int {
	switch rt {
	case rtCJK:
		return count
	case rtLetter:
		// 每 4 个字母 1 token，向上取整，最少 1
		return max(1, (count+3)/4)
	case rtDigit, rtPunct:
		// 每 2 个数字/标点 1 token，向上取整，最少 1
		return max(1, (count+1)/2)
	case rtOther:
		return count
	default:
		return count
	}
}
