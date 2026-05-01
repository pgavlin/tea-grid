package lineedit

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// --- Text / SetText ---

func TestText_Default(t *testing.T) {
	var m Model
	if m.Text() != "" {
		t.Errorf("expected empty text, got %q", m.Text())
	}
}

func TestSetText(t *testing.T) {
	var m Model
	m.SetText("hello")
	if m.Text() != "hello" {
		t.Errorf("expected 'hello', got %q", m.Text())
	}
}

func TestSetText_ClampsCursor(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(5) // at end
	m.SetText("hi")
	if m.Cursor() != 2 {
		t.Errorf("expected cursor clamped to 2, got %d", m.Cursor())
	}
}

// --- Cursor / SetCursor ---

func TestCursor_Default(t *testing.T) {
	var m Model
	if m.Cursor() != 0 {
		t.Errorf("expected cursor 0, got %d", m.Cursor())
	}
}

func TestSetCursor_Valid(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(3)
	if m.Cursor() != 3 {
		t.Errorf("expected cursor 3, got %d", m.Cursor())
	}
}

func TestSetCursor_Negative(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(-5)
	if m.Cursor() != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", m.Cursor())
	}
}

func TestSetCursor_BeyondLength(t *testing.T) {
	var m Model
	m.SetText("hi")
	m.SetCursor(100)
	if m.Cursor() != 2 {
		t.Errorf("expected cursor clamped to 2, got %d", m.Cursor())
	}
}

// --- CursorToEnd ---

func TestCursorToEnd(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(0)
	m.CursorToEnd()
	if m.Cursor() != 5 {
		t.Errorf("expected cursor 5, got %d", m.Cursor())
	}
}

// --- Insert ---

func TestInsert_AtBeginning(t *testing.T) {
	var m Model
	m.SetText("world")
	m.SetCursor(0)
	m.Insert("hello ")
	if m.Text() != "hello world" {
		t.Errorf("expected 'hello world', got %q", m.Text())
	}
	if m.Cursor() != 6 {
		t.Errorf("expected cursor 6, got %d", m.Cursor())
	}
}

func TestInsert_AtMiddle(t *testing.T) {
	var m Model
	m.SetText("hllo")
	m.SetCursor(1)
	m.Insert("e")
	if m.Text() != "hello" {
		t.Errorf("expected 'hello', got %q", m.Text())
	}
	if m.Cursor() != 2 {
		t.Errorf("expected cursor 2, got %d", m.Cursor())
	}
}

func TestInsert_AtEnd(t *testing.T) {
	var m Model
	m.SetText("hell")
	m.CursorToEnd()
	m.Insert("o")
	if m.Text() != "hello" {
		t.Errorf("expected 'hello', got %q", m.Text())
	}
	if m.Cursor() != 5 {
		t.Errorf("expected cursor 5, got %d", m.Cursor())
	}
}

// --- Backspace ---

func TestBackspace_AtZero(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(0)
	ok := m.Backspace()
	if ok {
		t.Error("expected Backspace at pos 0 to return false")
	}
	if m.Text() != "hello" {
		t.Errorf("text should be unchanged, got %q", m.Text())
	}
}

func TestBackspace_AtMiddle(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(3)
	ok := m.Backspace()
	if !ok {
		t.Error("expected Backspace to return true")
	}
	if m.Text() != "helo" {
		t.Errorf("expected 'helo', got %q", m.Text())
	}
	if m.Cursor() != 2 {
		t.Errorf("expected cursor 2, got %d", m.Cursor())
	}
}

// --- Delete ---

func TestDelete_AtEnd(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.CursorToEnd()
	ok := m.Delete()
	if ok {
		t.Error("expected Delete at end to return false")
	}
	if m.Text() != "hello" {
		t.Errorf("text should be unchanged, got %q", m.Text())
	}
}

func TestDelete_AtMiddle(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(2)
	ok := m.Delete()
	if !ok {
		t.Error("expected Delete to return true")
	}
	if m.Text() != "helo" {
		t.Errorf("expected 'helo', got %q", m.Text())
	}
	if m.Cursor() != 2 {
		t.Errorf("expected cursor 2, got %d", m.Cursor())
	}
}

// --- Left / Right ---

func TestLeft_AtZero(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(0)
	m.Left()
	if m.Cursor() != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", m.Cursor())
	}
}

func TestLeft(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(3)
	m.Left()
	if m.Cursor() != 2 {
		t.Errorf("expected cursor 2, got %d", m.Cursor())
	}
}

func TestRight_AtEnd(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.CursorToEnd()
	m.Right()
	if m.Cursor() != 5 {
		t.Errorf("expected cursor to stay at 5, got %d", m.Cursor())
	}
}

func TestRight(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(2)
	m.Right()
	if m.Cursor() != 3 {
		t.Errorf("expected cursor 3, got %d", m.Cursor())
	}
}

// --- Home / End ---

func TestHome(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(3)
	m.Home()
	if m.Cursor() != 0 {
		t.Errorf("expected cursor 0, got %d", m.Cursor())
	}
}

func TestEnd(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(0)
	m.End()
	if m.Cursor() != 5 {
		t.Errorf("expected cursor 5, got %d", m.Cursor())
	}
}

// --- HandleKeyMsg ---

func TestHandleKeyMsg_Backspace(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(1)
	ok := m.HandleKeyMsg(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if !ok {
		t.Error("expected handled")
	}
	if m.Text() != "ello" {
		t.Errorf("expected 'ello', got %q", m.Text())
	}
}

func TestHandleKeyMsg_Delete(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(0)
	ok := m.HandleKeyMsg(tea.KeyPressMsg{Code: tea.KeyDelete})
	if !ok {
		t.Error("expected handled")
	}
	if m.Text() != "ello" {
		t.Errorf("expected 'ello', got %q", m.Text())
	}
}

func TestHandleKeyMsg_Left(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(3)
	ok := m.HandleKeyMsg(tea.KeyPressMsg{Code: tea.KeyLeft})
	if !ok {
		t.Error("expected handled")
	}
	if m.Cursor() != 2 {
		t.Errorf("expected cursor 2, got %d", m.Cursor())
	}
}

func TestHandleKeyMsg_Right(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(2)
	ok := m.HandleKeyMsg(tea.KeyPressMsg{Code: tea.KeyRight})
	if !ok {
		t.Error("expected handled")
	}
	if m.Cursor() != 3 {
		t.Errorf("expected cursor 3, got %d", m.Cursor())
	}
}

func TestHandleKeyMsg_Home(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(3)
	ok := m.HandleKeyMsg(tea.KeyPressMsg{Code: tea.KeyHome})
	if !ok {
		t.Error("expected handled")
	}
	if m.Cursor() != 0 {
		t.Errorf("expected cursor 0, got %d", m.Cursor())
	}
}

func TestHandleKeyMsg_CtrlA(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(3)
	ok := m.HandleKeyMsg(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	if !ok {
		t.Error("expected handled")
	}
	if m.Cursor() != 0 {
		t.Errorf("expected cursor 0, got %d", m.Cursor())
	}
}

func TestHandleKeyMsg_End(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(0)
	ok := m.HandleKeyMsg(tea.KeyPressMsg{Code: tea.KeyEnd})
	if !ok {
		t.Error("expected handled")
	}
	if m.Cursor() != 5 {
		t.Errorf("expected cursor 5, got %d", m.Cursor())
	}
}

func TestHandleKeyMsg_CtrlE(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(0)
	ok := m.HandleKeyMsg(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	if !ok {
		t.Error("expected handled")
	}
	if m.Cursor() != 5 {
		t.Errorf("expected cursor 5, got %d", m.Cursor())
	}
}

func TestHandleKeyMsg_Space(t *testing.T) {
	var m Model
	m.SetText("ab")
	m.SetCursor(1)
	ok := m.HandleKeyMsg(tea.KeyPressMsg{Code: tea.KeySpace})
	if !ok {
		t.Error("expected handled")
	}
	if m.Text() != "a b" {
		t.Errorf("expected 'a b', got %q", m.Text())
	}
}

func TestHandleKeyMsg_Runes(t *testing.T) {
	var m Model
	m.SetText("hllo")
	m.SetCursor(1)
	ok := m.HandleKeyMsg(tea.KeyPressMsg{Code: 'e', Text: "e"})
	if !ok {
		t.Error("expected handled")
	}
	if m.Text() != "hello" {
		t.Errorf("expected 'hello', got %q", m.Text())
	}
}

func TestHandleKeyMsg_Unhandled(t *testing.T) {
	var m Model
	m.SetText("hello")
	ok := m.HandleKeyMsg(tea.KeyPressMsg{Code: tea.KeyEnter})
	if ok {
		t.Error("expected unhandled key to return false")
	}
}

// --- RenderLine ---

func TestRenderLine_ZeroWidth(t *testing.T) {
	var m Model
	m.SetText("hello")
	result := m.RenderLine(0, "")
	if result != "" {
		t.Errorf("expected empty string for width=0, got %q", result)
	}
}

func TestRenderLine_NegativeWidth(t *testing.T) {
	var m Model
	m.SetText("hello")
	result := m.RenderLine(-5, "")
	if result != "" {
		t.Errorf("expected empty string for negative width, got %q", result)
	}
}

func TestRenderLine_ShortText(t *testing.T) {
	var m Model
	m.SetText("ab")
	m.SetCursor(0)
	result := m.RenderLine(10, "")
	// Should contain reverse video around 'a' and padding
	if !strings.Contains(result, reverseOn) || !strings.Contains(result, reverseOff) {
		t.Errorf("expected reverse video markers in result: %q", result)
	}
}

func TestRenderLine_CursorBeyondText(t *testing.T) {
	var m Model
	m.SetText("ab")
	m.SetCursor(2) // cursor at end (space cursor)
	result := m.RenderLine(10, "")
	// Cursor is a space in reverse video
	if !strings.Contains(result, reverseOn+" "+reverseOff) {
		t.Errorf("expected reverse space at cursor, got %q", result)
	}
}

func TestRenderLine_CursorScrolled(t *testing.T) {
	var m Model
	m.SetText("abcdefghij")
	m.SetCursor(9) // cursor near end, viewWidth=5 means cursor >= viewWidth
	result := m.RenderLine(5, "")
	// The cursor should still be visible (scrolled view)
	if !strings.Contains(result, reverseOn) {
		t.Errorf("expected cursor visible in scrolled view: %q", result)
	}
}

func TestRenderLine_SuffixExceedsWidth(t *testing.T) {
	var m Model
	m.SetText("hi")
	m.SetCursor(0)
	// Suffix is 5 chars, width is 3 → viewWidth would be -2, clamped to 1
	result := m.RenderLine(3, "[ERR]")
	if !strings.Contains(result, "[ERR]") {
		t.Errorf("expected suffix in result: %q", result)
	}
	if !strings.Contains(result, reverseOn) {
		t.Errorf("expected cursor in result: %q", result)
	}
}

func TestRenderLine_WithSuffix(t *testing.T) {
	var m Model
	m.SetText("hello")
	m.SetCursor(0)
	result := m.RenderLine(10, "!")
	if !strings.HasSuffix(result, "!") {
		t.Errorf("expected result to end with suffix '!', got %q", result)
	}
}

func TestRenderLine_EmptyText(t *testing.T) {
	var m Model
	m.SetCursor(0)
	result := m.RenderLine(5, "")
	// Should show cursor (space in reverse video) and padding
	if !strings.Contains(result, reverseOn+" "+reverseOff) {
		t.Errorf("expected reverse space for empty text cursor: %q", result)
	}
}

// --- TruncateOrPad ---

func TestTruncateOrPad_ZeroWidth(t *testing.T) {
	result := TruncateOrPad("hello", 0)
	if result != "" {
		t.Errorf("expected empty string for width=0, got %q", result)
	}
}

func TestTruncateOrPad_Width1LongString(t *testing.T) {
	result := TruncateOrPad("hello", 1)
	// width=1, no room for ellipsis, just truncate to 1 rune
	if result != "h" {
		t.Errorf("expected 'h', got %q", result)
	}
}

func TestTruncateOrPad_TruncateWithEllipsis(t *testing.T) {
	result := TruncateOrPad("hello world", 6)
	// Should be 5 chars + ellipsis = 6
	if result != "hello\u2026" {
		t.Errorf("expected 'hello…', got %q", result)
	}
}

func TestTruncateOrPad_PadShort(t *testing.T) {
	result := TruncateOrPad("hi", 5)
	if result != "hi   " {
		t.Errorf("expected 'hi   ', got %q", result)
	}
}

func TestTruncateOrPad_ExactFit(t *testing.T) {
	result := TruncateOrPad("hello", 5)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

// --- readline editing operations ---

func TestKillToEnd(t *testing.T) {
	m := &Model{}
	m.SetText("hello world")
	m.SetCursor(5)
	if !m.KillToEnd() {
		t.Errorf("KillToEnd returned false, want true")
	}
	if m.Text() != "hello" {
		t.Errorf("Text() = %q, want %q", m.Text(), "hello")
	}
	if m.Cursor() != 5 {
		t.Errorf("Cursor() = %d, want 5", m.Cursor())
	}
}

func TestKillToEnd_AtEnd(t *testing.T) {
	m := &Model{}
	m.SetText("hello")
	m.CursorToEnd()
	if m.KillToEnd() {
		t.Errorf("KillToEnd at end returned true, want false (no-op)")
	}
}

func TestKillToStart(t *testing.T) {
	m := &Model{}
	m.SetText("hello world")
	m.SetCursor(6)
	if !m.KillToStart() {
		t.Errorf("KillToStart returned false, want true")
	}
	if m.Text() != "world" {
		t.Errorf("Text() = %q, want %q", m.Text(), "world")
	}
	if m.Cursor() != 0 {
		t.Errorf("Cursor() = %d, want 0", m.Cursor())
	}
}

func TestKillToStart_AtZero(t *testing.T) {
	m := &Model{}
	m.SetText("hello")
	if m.KillToStart() {
		t.Errorf("KillToStart at 0 returned true, want false (no-op)")
	}
}

func TestWordLeft(t *testing.T) {
	m := &Model{}
	m.SetText("state:open priority:>5")
	m.CursorToEnd() // at 22

	m.WordLeft()
	if m.Cursor() != 11 { // start of "priority:>5"
		t.Errorf("WordLeft: cursor=%d, want 11", m.Cursor())
	}

	m.WordLeft()
	if m.Cursor() != 0 { // start of "state:open"
		t.Errorf("WordLeft: cursor=%d, want 0", m.Cursor())
	}

	m.WordLeft()
	if m.Cursor() != 0 {
		t.Errorf("WordLeft at 0: cursor=%d, want 0 (no-op)", m.Cursor())
	}
}

func TestWordRight(t *testing.T) {
	m := &Model{}
	m.SetText("state:open priority:>5")
	// at 0
	m.WordRight()
	if m.Cursor() != 11 { // start of "priority:>5"
		t.Errorf("WordRight: cursor=%d, want 11", m.Cursor())
	}

	m.WordRight()
	if m.Cursor() != 22 { // end
		t.Errorf("WordRight: cursor=%d, want 22 (end)", m.Cursor())
	}

	m.WordRight()
	if m.Cursor() != 22 {
		t.Errorf("WordRight at end: cursor=%d, want 22 (no-op)", m.Cursor())
	}
}

func TestKillWordLeft(t *testing.T) {
	m := &Model{}
	m.SetText("state:open priority:>5")
	m.CursorToEnd()
	if !m.KillWordLeft() {
		t.Errorf("KillWordLeft returned false")
	}
	if m.Text() != "state:open " {
		t.Errorf("Text() = %q, want %q", m.Text(), "state:open ")
	}
	if m.Cursor() != 11 {
		t.Errorf("Cursor() = %d, want 11", m.Cursor())
	}
}

func TestKillWordRight(t *testing.T) {
	m := &Model{}
	m.SetText("state:open priority:>5")
	// at 0
	if !m.KillWordRight() {
		t.Errorf("KillWordRight returned false")
	}
	if m.Text() != "priority:>5" {
		t.Errorf("Text() = %q, want %q", m.Text(), "priority:>5")
	}
	if m.Cursor() != 0 {
		t.Errorf("Cursor() = %d, want 0", m.Cursor())
	}
}

func TestHandleKeyMsg_CtrlB_CtrlF(t *testing.T) {
	m := &Model{}
	m.SetText("hello")
	m.CursorToEnd()
	m.HandleKeyMsg(tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
	if m.Cursor() != 4 {
		t.Errorf("Ctrl+B: cursor=%d, want 4", m.Cursor())
	}
	m.HandleKeyMsg(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	if m.Cursor() != 5 {
		t.Errorf("Ctrl+F: cursor=%d, want 5", m.Cursor())
	}
}

func TestHandleKeyMsg_CtrlD(t *testing.T) {
	m := &Model{}
	m.SetText("hello")
	m.SetCursor(2)
	m.HandleKeyMsg(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if m.Text() != "helo" {
		t.Errorf("Ctrl+D: text=%q, want %q", m.Text(), "helo")
	}
}

func TestHandleKeyMsg_CtrlH(t *testing.T) {
	m := &Model{}
	m.SetText("hello")
	m.CursorToEnd()
	m.HandleKeyMsg(tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl})
	if m.Text() != "hell" {
		t.Errorf("Ctrl+H: text=%q, want %q", m.Text(), "hell")
	}
}

func TestHandleKeyMsg_CtrlK(t *testing.T) {
	m := &Model{}
	m.SetText("hello world")
	m.SetCursor(5)
	m.HandleKeyMsg(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	if m.Text() != "hello" {
		t.Errorf("Ctrl+K: text=%q, want %q", m.Text(), "hello")
	}
}

func TestHandleKeyMsg_CtrlU(t *testing.T) {
	m := &Model{}
	m.SetText("hello world")
	m.SetCursor(6)
	m.HandleKeyMsg(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	if m.Text() != "world" {
		t.Errorf("Ctrl+U: text=%q, want %q", m.Text(), "world")
	}
}

func TestHandleKeyMsg_CtrlW(t *testing.T) {
	m := &Model{}
	m.SetText("hello world")
	m.CursorToEnd()
	m.HandleKeyMsg(tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl})
	if m.Text() != "hello " {
		t.Errorf("Ctrl+W: text=%q, want %q", m.Text(), "hello ")
	}
}

func TestHandleKeyMsg_AltB_AltF(t *testing.T) {
	m := &Model{}
	m.SetText("foo bar baz")
	m.CursorToEnd()
	m.HandleKeyMsg(tea.KeyPressMsg{Code: 'b', Mod: tea.ModAlt})
	if m.Cursor() != 8 {
		t.Errorf("Alt+B: cursor=%d, want 8", m.Cursor())
	}
	m.HandleKeyMsg(tea.KeyPressMsg{Code: 'f', Mod: tea.ModAlt})
	if m.Cursor() != 11 {
		t.Errorf("Alt+F: cursor=%d, want 11", m.Cursor())
	}
}

func TestHandleKeyMsg_AltD(t *testing.T) {
	m := &Model{}
	m.SetText("foo bar baz")
	m.HandleKeyMsg(tea.KeyPressMsg{Code: 'd', Mod: tea.ModAlt})
	if m.Text() != "bar baz" {
		t.Errorf("Alt+D: text=%q, want %q", m.Text(), "bar baz")
	}
}

func TestHandleKeyMsg_AltBackspace(t *testing.T) {
	m := &Model{}
	m.SetText("foo bar baz")
	m.CursorToEnd()
	m.HandleKeyMsg(tea.KeyPressMsg{Code: tea.KeyBackspace, Mod: tea.ModAlt})
	if m.Text() != "foo bar " {
		t.Errorf("Alt+Backspace: text=%q, want %q", m.Text(), "foo bar ")
	}
}

func TestRenderLineDim_NoDim(t *testing.T) {
	m := &Model{}
	m.SetText("hello")
	m.SetCursor(2)
	plain := m.RenderLine(10, "")
	dim := m.RenderLineDim(10, "", 0, 0)
	if plain != dim {
		t.Errorf("RenderLineDim with empty range != RenderLine\nplain=%q\ndim=%q", plain, dim)
	}
}

func TestRenderLineDim_AppliesFaintInRange(t *testing.T) {
	m := &Model{}
	m.SetText("state:Open")
	m.SetCursor(10)
	out := m.RenderLineDim(15, "", 6, 10)
	if !strings.Contains(out, "\x1b[2m") {
		t.Errorf("output missing faintOn (\\x1b[2m): %q", out)
	}
	if !strings.Contains(out, "\x1b[22m") {
		t.Errorf("output missing faintOff (\\x1b[22m): %q", out)
	}
}

func TestKillWordLeft_AtZero(t *testing.T) {
	m := &Model{}
	m.SetText("hello")
	if m.KillWordLeft() {
		t.Errorf("KillWordLeft at cursor=0 returned true, want false")
	}
}

func TestKillWordLeft_NoChange(t *testing.T) {
	m := &Model{}
	m.SetText("   ") // only whitespace
	m.SetCursor(0)
	if m.KillWordLeft() {
		t.Errorf("KillWordLeft on whitespace-only at 0 returned true")
	}
}

func TestKillWordRight_AtEnd(t *testing.T) {
	m := &Model{}
	m.SetText("hello")
	m.CursorToEnd()
	if m.KillWordRight() {
		t.Errorf("KillWordRight at end returned true, want false")
	}
}

func TestKillWordRight_TrailingWhitespaceOnly(t *testing.T) {
	m := &Model{}
	m.SetText("hello   ")
	m.SetCursor(5) // after "hello"
	if !m.KillWordRight() {
		t.Errorf("KillWordRight with trailing whitespace returned false")
	}
	if m.Text() != "hello" {
		t.Errorf("Text = %q, want %q", m.Text(), "hello")
	}
}

func TestRenderLineDim_DimAtCursor(t *testing.T) {
	// When the cursor sits inside the dim range, both reverse and faint
	// codes should appear in the output.
	m := &Model{}
	m.SetText("state:Open")
	m.SetCursor(7) // inside "Open"
	out := m.RenderLineDim(15, "", 6, 10)
	if !strings.Contains(out, "\x1b[2m") || !strings.Contains(out, "\x1b[7m") {
		t.Errorf("output missing faint or reverse codes: %q", out)
	}
}

func TestRenderLineDim_ZeroWidth(t *testing.T) {
	m := &Model{}
	m.SetText("hello")
	if got := m.RenderLineDim(0, "", 0, 5); got != "" {
		t.Errorf("RenderLineDim(0) = %q, want empty", got)
	}
}
