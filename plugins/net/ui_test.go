package net

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- runeWidth 测试 ---

func TestRuneWidth_ASCII(t *testing.T) {
	assert.Equal(t, 5, runeWidth("hello"))
}

func TestRuneWidth_Empty(t *testing.T) {
	assert.Equal(t, 0, runeWidth(""))
}

func TestRuneWidth_Chinese(t *testing.T) {
	// 中文字符通常占2个显示宽度
	assert.Equal(t, 4, runeWidth("你好"))
}

func TestRuneWidth_Mixed(t *testing.T) {
	// 混合 ASCII + 中文: "hi"=2 + "你好"=4 = 6
	assert.Equal(t, 6, runeWidth("hi你好"))
}

func TestRuneWidth_Japanese(t *testing.T) {
	// 日文平假名每个占2个显示宽度: 4字符×2=8
	assert.Equal(t, 8, runeWidth("ひらがな"))
}

func TestRuneWidth_SpecialChars(t *testing.T) {
	// 特殊字符
	assert.Equal(t, 3, runeWidth("abc"))
	assert.Equal(t, 1, runeWidth(" "))
}
