package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

func TestColorConstants_NotDefault(t *testing.T) {
	assert.NotEqual(t, tcell.ColorDefault, ColorPrimary)
	assert.NotEqual(t, tcell.ColorDefault, ColorSecondary)
	assert.NotEqual(t, tcell.ColorDefault, ColorSuccess)
	assert.NotEqual(t, tcell.ColorDefault, ColorDanger)
	assert.NotEqual(t, tcell.ColorDefault, ColorWarning)
	assert.NotEqual(t, tcell.ColorDefault, ColorInfo)
}

func TestColorValues(t *testing.T) {
	assert.Equal(t, tcell.ColorDarkBlue, ColorPrimary)
	assert.Equal(t, tcell.ColorDarkGreen, ColorSecondary)
	assert.Equal(t, tcell.ColorYellow, ColorAccent)
	assert.Equal(t, tcell.ColorGreen, ColorSuccess)
	assert.Equal(t, tcell.ColorOrange, ColorWarning)
	assert.Equal(t, tcell.ColorRed, ColorDanger)
	assert.Equal(t, tcell.ColorAqua, ColorInfo)
	assert.Equal(t, tcell.ColorGray, ColorMuted)
	assert.Equal(t, tcell.ColorWhite, ColorText)
	assert.Equal(t, tcell.ColorBlack, ColorBackground)
}

func TestBoxDrawingCharacters(t *testing.T) {
	assert.Equal(t, '┌', BoxTopLeft)
	assert.Equal(t, '┐', BoxTopRight)
	assert.Equal(t, '└', BoxBottomLeft)
	assert.Equal(t, '┘', BoxBottomRight)
	assert.Equal(t, '─', BoxHorizontal)
	assert.Equal(t, '│', BoxVertical)
	assert.Equal(t, '├', BoxTeeLeft)
	assert.Equal(t, '┤', BoxTeeRight)
	assert.Equal(t, '┬', BoxTeeTop)
	assert.Equal(t, '┴', BoxTeeBottom)
	assert.Equal(t, '┼', BoxCross)
	assert.Equal(t, '═', BoxDoubleHorizontal)
	assert.Equal(t, '║', BoxDoubleVertical)
}

func TestArrowCharacters(t *testing.T) {
	assert.Equal(t, '▲', ArrowUp)
	assert.Equal(t, '▼', ArrowDown)
	assert.Equal(t, '►', ArrowRight)
	assert.Equal(t, '◄', ArrowLeft)
}

func TestSymbolCharacters(t *testing.T) {
	assert.Equal(t, '•', Bullet)
	assert.Equal(t, '✓', Check)
	assert.Equal(t, '✗', Cross)
}

// mockScreen implements tcell.Screen for DrawBox testing
type mockScreen struct {
	cells map[[2]int]rune
}

func newMockScreen() *mockScreen {
	return &mockScreen{cells: make(map[[2]int]rune)}
}

func (m *mockScreen) Init() error                                                       { return nil }
func (m *mockScreen) Fini()                                                             {}
func (m *mockScreen) Clear()                                                            {}
func (m *mockScreen) Fill(rune, tcell.Style)                                            {}
func (m *mockScreen) SetCell(x int, y int, style tcell.Style, ch ...rune)              {}
func (m *mockScreen) SetContent(x, y int, main rune, comb []rune, st tcell.Style) {
	m.cells[[2]int{x, y}] = main
}
func (m *mockScreen) GetContent(x, y int) (rune, []rune, tcell.Style, int) {
	return m.cells[[2]int{x, y}], nil, tcell.StyleDefault, 1
}
func (m *mockScreen) SetStyle(tcell.Style)                                              {}
func (m *mockScreen) ShowCursor(int, int)                                               {}
func (m *mockScreen) HideCursor()                                                       {}
func (m *mockScreen) SetCursorStyle(tcell.CursorStyle)                                  {}
func (m *mockScreen) Size() (int, int)                                                  { return 40, 20 }
func (m *mockScreen) ChannelEvents(ch chan<- tcell.Event, quit <-chan struct{})         {}
func (m *mockScreen) PollEvent() tcell.Event                                            { return nil }
func (m *mockScreen) HasPendingEvent() bool                                             { return false }
func (m *mockScreen) PostEvent(tcell.Event) error                                       { return nil }
func (m *mockScreen) PostEventWait(tcell.Event)                                         {}
func (m *mockScreen) EnableMouse(...tcell.MouseFlags)                                   {}
func (m *mockScreen) DisableMouse()                                                     {}
func (m *mockScreen) EnablePaste()                                                      {}
func (m *mockScreen) DisablePaste()                                                     {}
func (m *mockScreen) EnableFocus()                                                      {}
func (m *mockScreen) DisableFocus()                                                     {}
func (m *mockScreen) HasMouse() bool                                                    { return false }
func (m *mockScreen) Colors() int                                                       { return 256 }
func (m *mockScreen) Show()                                                             {}
func (m *mockScreen) Sync()                                                             {}
func (m *mockScreen) CharacterSet() string                                              { return "UTF-8" }
func (m *mockScreen) RegisterRuneFallback(rune, string)                                 {}
func (m *mockScreen) UnregisterRuneFallback(rune)                                       {}
func (m *mockScreen) CanDisplay(r rune, checkFallbacks bool) bool                      { return true }
func (m *mockScreen) Resize(int, int, int, int)                                         {}
func (m *mockScreen) SetSize(int, int)                                                  {}
func (m *mockScreen) LockRegion(int, int, int, int, bool)                               {}
func (m *mockScreen) HasKey(tcell.Key) bool                                             { return true }
func (m *mockScreen) Suspend() error                                                    { return nil }
func (m *mockScreen) Resume() error                                                     { return nil }
func (m *mockScreen) Beep() error                                                       { return nil }
func (m *mockScreen) Tty() (tcell.Tty, bool)                                            { return nil, false }

func TestDrawBox_NoTitle(t *testing.T) {
	s := newMockScreen()
	DrawBox(s, 0, 0, 10, 5, "", tcell.ColorWhite)
	assert.Equal(t, BoxTopLeft, s.cells[[2]int{0, 0}])
	assert.Equal(t, BoxTopRight, s.cells[[2]int{9, 0}])
	assert.Equal(t, BoxBottomLeft, s.cells[[2]int{0, 4}])
	assert.Equal(t, BoxBottomRight, s.cells[[2]int{9, 4}])
	assert.Equal(t, BoxHorizontal, s.cells[[2]int{1, 0}])
	assert.Equal(t, BoxVertical, s.cells[[2]int{0, 1}])
}

func TestDrawBox_WithTitle(t *testing.T) {
	s := newMockScreen()
	DrawBox(s, 0, 0, 20, 5, "Test", tcell.ColorGreen)
	assert.Equal(t, 'T', s.cells[[2]int{2, 0}])
	assert.Equal(t, 'e', s.cells[[2]int{3, 0}])
	assert.Equal(t, 's', s.cells[[2]int{4, 0}])
	assert.Equal(t, 't', s.cells[[2]int{5, 0}])
}

func TestDrawBox_SmallBox(t *testing.T) {
	s := newMockScreen()
	DrawBox(s, 0, 0, 3, 3, "", tcell.ColorWhite)
	assert.Equal(t, BoxTopLeft, s.cells[[2]int{0, 0}])
	assert.Equal(t, BoxTopRight, s.cells[[2]int{2, 0}])
	assert.Equal(t, BoxBottomLeft, s.cells[[2]int{0, 2}])
	assert.Equal(t, BoxBottomRight, s.cells[[2]int{2, 2}])
}

