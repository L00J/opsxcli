package sys

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

// mockScreen 实现 tcell.Screen 接口，用于绘制函数测试
type mockScreen struct {
	cells  map[[2]int]rune
	width  int
	height int
}

func newMockScreen(w, h int) *mockScreen {
	return &mockScreen{
		cells:  make(map[[2]int]rune),
		width:  w,
		height: h,
	}
}

func (m *mockScreen) Init() error                            { return nil }
func (m *mockScreen) Fini()                                  {}
func (m *mockScreen) Clear()                                 {}
func (m *mockScreen) Fill(rune, tcell.Style)                 {}
func (m *mockScreen) SetCell(int, int, tcell.Style, ...rune) {}
func (m *mockScreen) SetContent(x, y int, main rune, comb []rune, st tcell.Style) {
	m.cells[[2]int{x, y}] = main
}
func (m *mockScreen) GetContent(x, y int) (rune, []rune, tcell.Style, int) {
	return m.cells[[2]int{x, y}], nil, tcell.StyleDefault, 1
}
func (m *mockScreen) SetStyle(tcell.Style)                              {}
func (m *mockScreen) ShowCursor(int, int)                               {}
func (m *mockScreen) HideCursor()                                       {}
func (m *mockScreen) SetCursorStyle(tcell.CursorStyle)                  {}
func (m *mockScreen) Size() (int, int)                                  { return m.width, m.height }
func (m *mockScreen) ChannelEvents(chan<- tcell.Event, <-chan struct{}) {}
func (m *mockScreen) PollEvent() tcell.Event                            { return nil }
func (m *mockScreen) HasPendingEvent() bool                             { return false }
func (m *mockScreen) PostEvent(tcell.Event) error                       { return nil }
func (m *mockScreen) PostEventWait(tcell.Event)                         {}
func (m *mockScreen) EnableMouse(...tcell.MouseFlags)                   {}
func (m *mockScreen) DisableMouse()                                     {}
func (m *mockScreen) EnablePaste()                                      {}
func (m *mockScreen) DisablePaste()                                     {}
func (m *mockScreen) EnableFocus()                                      {}
func (m *mockScreen) DisableFocus()                                     {}
func (m *mockScreen) HasMouse() bool                                    { return false }
func (m *mockScreen) Colors() int                                       { return 256 }
func (m *mockScreen) Show()                                             {}
func (m *mockScreen) Sync()                                             {}
func (m *mockScreen) CharacterSet() string                              { return "UTF-8" }
func (m *mockScreen) RegisterRuneFallback(rune, string)                 {}
func (m *mockScreen) UnregisterRuneFallback(rune)                       {}
func (m *mockScreen) CanDisplay(r rune, checkFallbacks bool) bool       { return true }
func (m *mockScreen) Resize(int, int, int, int)                         {}
func (m *mockScreen) SetSize(int, int)                                  {}
func (m *mockScreen) LockRegion(int, int, int, int, bool)               {}
func (m *mockScreen) HasKey(tcell.Key) bool                             { return true }
func (m *mockScreen) Suspend() error                                    { return nil }
func (m *mockScreen) Resume() error                                     { return nil }
func (m *mockScreen) Beep() error                                       { return nil }
func (m *mockScreen) Tty() (tcell.Tty, bool)                            { return nil, false }

// === drawText 测试 ===

func TestDrawText_Basic(t *testing.T) {
	s := newMockScreen(40, 10)
	drawText(s, 0, 0, "hello", tcell.ColorWhite)
	// 验证字符被写入
	assert.Equal(t, 'h', s.cells[[2]int{0, 0}])
	assert.Equal(t, 'o', s.cells[[2]int{4, 0}])
}

func TestDrawText_TruncateAtScreenWidth(t *testing.T) {
	s := newMockScreen(5, 10)
	drawText(s, 0, 0, "hello world", tcell.ColorWhite)
	// 只有前5个字符应在屏幕宽度内
	assert.Equal(t, 'h', s.cells[[2]int{0, 0}])
	assert.Equal(t, 'o', s.cells[[2]int{4, 0}])
	// 第6个字符不应写入
	_, ok := s.cells[[2]int{5, 0}]
	assert.False(t, ok)
}

// === drawTextWithStyle 测试 ===

func TestDrawTextWithStyle_Basic(t *testing.T) {
	s := newMockScreen(40, 10)
	drawTextWithStyle(s, 2, 3, "ABC", tcell.StyleDefault)
	assert.Equal(t, 'A', s.cells[[2]int{2, 3}])
	assert.Equal(t, 'C', s.cells[[2]int{4, 3}])
}

func TestDrawTextWithStyle_EmptyString(t *testing.T) {
	s := newMockScreen(40, 10)
	// 不应 panic
	drawTextWithStyle(s, 0, 0, "", tcell.StyleDefault)
}

func TestDrawTextWithStyle_CJK(t *testing.T) {
	s := newMockScreen(40, 10)
	drawTextWithStyle(s, 0, 0, "你好", tcell.StyleDefault)
	assert.Equal(t, '你', s.cells[[2]int{0, 0}])
	assert.Equal(t, '好', s.cells[[2]int{2, 0}])
}

// === drawProgressBar 测试 ===

func TestDrawProgressBar_50Percent(t *testing.T) {
	s := newMockScreen(20, 10)
	drawProgressBar(s, 0, 0, 10, 50, tcell.ColorGreen)
	// 前5格应被填充（50% of 10）
	assert.Equal(t, '█', s.cells[[2]int{0, 0}])
	assert.Equal(t, '█', s.cells[[2]int{4, 0}])
	// 后5格应为空白
	assert.Equal(t, '░', s.cells[[2]int{5, 0}])
	assert.Equal(t, '░', s.cells[[2]int{9, 0}])
}

func TestDrawProgressBar_Over100(t *testing.T) {
	s := newMockScreen(20, 10)
	drawProgressBar(s, 0, 0, 10, 150, tcell.ColorGreen)
	// 超过 100% 应被限制为 100%
	for i := 0; i < 10; i++ {
		assert.Equal(t, '█', s.cells[[2]int{i, 0}])
	}
}

func TestDrawProgressBar_Negative(t *testing.T) {
	s := newMockScreen(20, 10)
	drawProgressBar(s, 0, 0, 10, -10, tcell.ColorGreen)
	// 负数应被限制为 0%
	for i := 0; i < 10; i++ {
		assert.Equal(t, '░', s.cells[[2]int{i, 0}])
	}
}

func TestDrawProgressBar_Zero(t *testing.T) {
	s := newMockScreen(20, 10)
	drawProgressBar(s, 0, 0, 10, 0, tcell.ColorGreen)
	for i := 0; i < 10; i++ {
		assert.Equal(t, '░', s.cells[[2]int{i, 0}])
	}
}

func TestDrawProgressBar_Full(t *testing.T) {
	s := newMockScreen(20, 10)
	drawProgressBar(s, 0, 0, 10, 100, tcell.ColorGreen)
	for i := 0; i < 10; i++ {
		assert.Equal(t, '█', s.cells[[2]int{i, 0}])
	}
}
