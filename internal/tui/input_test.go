package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// ===== InputModel Init 测试 =====

func TestInputModel_Init(t *testing.T) {
	model := NewInputModel("测试提示")
	cmd := model.Init()
	assert.Nil(t, cmd)
}

// ===== InputModel Update 测试 =====

func TestInputModel_Update_CtrlC(t *testing.T) {
	model := NewInputModel("测试")
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.True(t, model.IsCancelled())
	assert.NotNil(t, cmd) // tea.Quit cmd
}

func TestInputModel_Update_Esc(t *testing.T) {
	model := NewInputModel("测试")
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.True(t, model.IsCancelled())
	assert.NotNil(t, cmd)
}

func TestInputModel_Update_EnterWithContent(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("hello")
	model.cursor = 5

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, model.IsSubmitted())
	assert.NotNil(t, cmd)
}

func TestInputModel_Update_EnterEmpty(t *testing.T) {
	model := NewInputModel("测试")
	assert.Equal(t, 0, len(model.value))

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, model.IsSubmitted())
	assert.Nil(t, cmd) // 空输入不应提交
}

func TestInputModel_Update_LeftArrow(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("hello")
	model.cursor = 3

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 2, model.cursor)

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 1, model.cursor)
}

func TestInputModel_Update_LeftArrow_AtBeginning(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("hello")
	model.cursor = 0

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 0, model.cursor) // 不应小于 0
}

func TestInputModel_Update_RightArrow(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("hello")
	model.cursor = 3

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 4, model.cursor)
}

func TestInputModel_Update_RightArrow_AtEnd(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("hello")
	model.cursor = 5

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 5, model.cursor) // 不应超过 len(value)
}

func TestInputModel_Update_Home(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("hello")
	model.cursor = 3

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyHome})
	assert.Equal(t, 0, model.cursor)
}

func TestInputModel_Update_End(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("hello")
	model.cursor = 2

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, 5, model.cursor)
}

func TestInputModel_Update_Backspace(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("hello")
	model.cursor = 3

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "helo", model.GetValue())
	assert.Equal(t, 2, model.cursor)
}

func TestInputModel_Update_Backspace_AtBeginning(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("hello")
	model.cursor = 0

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "hello", model.GetValue())
	assert.Equal(t, 0, model.cursor)
}

func TestInputModel_Update_Delete(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("hello")
	model.cursor = 2

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyDelete})
	assert.Equal(t, "helo", model.GetValue())
	assert.Equal(t, 2, model.cursor)
}

func TestInputModel_Update_Delete_AtEnd(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("hello")
	model.cursor = 5

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyDelete})
	assert.Equal(t, "hello", model.GetValue())
	assert.Equal(t, 5, model.cursor)
}

func TestInputModel_Update_TypeRunes(t *testing.T) {
	model := NewInputModel("测试")

	// 模拟输入字符 'a'
	_, cmd := model.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'a'},
	})
	assert.Equal(t, "a", model.GetValue())
	assert.Equal(t, 1, model.cursor)
	assert.Nil(t, cmd)
}

func TestInputModel_Update_TypeMultipleRunes(t *testing.T) {
	model := NewInputModel("测试")

	// 依次输入 'a', 'b', 'c'
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	assert.Equal(t, "abc", model.GetValue())
	assert.Equal(t, 3, model.cursor)
}

func TestInputModel_Update_TypeChinese(t *testing.T) {
	model := NewInputModel("测试")

	// 模拟输入中文字符
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'你'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'好'}})

	assert.Equal(t, "你好", model.GetValue())
	assert.Equal(t, 2, model.cursor)
}

func TestInputModel_Update_InsertInMiddle(t *testing.T) {
	model := NewInputModel("测试")
	model.value = []rune("abc")
	model.cursor = 1

	// 在 'a' 和 'b' 之间插入 'X'
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	assert.Equal(t, "aXbc", model.GetValue())
	assert.Equal(t, 2, model.cursor)
}

func TestInputModel_Update_Disabled(t *testing.T) {
	model := NewInputModel("测试")
	model.SetEnabled(false)

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Equal(t, "", model.GetValue())
	assert.Nil(t, cmd)
}

// ===== InputModel View 扩展测试 =====

func TestInputModel_View_CursorAtEnd(t *testing.T) {
	model := NewInputModel("输入:")
	model.value = []rune("abc")
	model.cursor = 3
	view := model.View()
	assert.NotEmpty(t, view)
}

func TestInputModel_View_CursorInMiddle(t *testing.T) {
	model := NewInputModel("输入:")
	model.value = []rune("abc")
	model.cursor = 1
	view := model.View()
	assert.NotEmpty(t, view)
}

func TestInputModel_View_CursorAtStart(t *testing.T) {
	model := NewInputModel("输入:")
	model.value = []rune("abc")
	model.cursor = 0
	view := model.View()
	assert.NotEmpty(t, view)
}

func TestInputModel_View_DisabledNil(t *testing.T) {
	model := NewInputModel("输入:")
	model.SetEnabled(false)
	view := model.View()
	assert.Equal(t, "", view)
}

// ===== InputModel 完整交互测试 =====

func TestInputModel_FullInteraction(t *testing.T) {
	model := NewInputModel("请输入:")

	// 输入 "hello"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	assert.Equal(t, "hello", model.GetValue())
	assert.Equal(t, 5, model.cursor)

	// 左移2次
	model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 3, model.cursor)

	// 删除 'l' (at index 3)
	model.Update(tea.KeyMsg{Type: tea.KeyDelete})
	assert.Equal(t, "helo", model.GetValue())

	// 退格 - removes char at cursor-1=2 ('l')
	model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "heo", model.GetValue())
	assert.Equal(t, 2, model.cursor)

	// Home
	model.Update(tea.KeyMsg{Type: tea.KeyHome})
	assert.Equal(t, 0, model.cursor)

	// End - "heo" has length 3
	model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, 3, model.cursor)

	// 提交
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, model.IsSubmitted())
	assert.NotNil(t, cmd)
}

func TestInputModel_CancelFlow(t *testing.T) {
	model := NewInputModel("请输入:")

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Equal(t, "a", model.GetValue())

	// 取消
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.True(t, model.IsCancelled())
	assert.NotNil(t, cmd)
}

// ===== InputModel Reset 测试 =====

func TestInputModel_ResetAfterInput(t *testing.T) {
	model := NewInputModel("测试")
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	model.Reset()

	assert.Equal(t, "", model.GetValue())
	assert.Equal(t, 0, model.cursor)
	assert.False(t, model.IsSubmitted())
	assert.False(t, model.IsCancelled())
}
