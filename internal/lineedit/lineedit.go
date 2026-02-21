// Package lineedit provides a minimal single-line text editing model.
// It encapsulates the cursor, text manipulation, and rendering logic shared
// by the built-in filters and cell editors.
package lineedit

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Model is a single-line text buffer with a cursor.
type Model struct {
	text   string
	cursor int
}

// Text returns the current text.
func (m *Model) Text() string { return m.text }

// SetText replaces the text and clamps the cursor.
func (m *Model) SetText(s string) {
	m.text = s
	if m.cursor > len(m.text) {
		m.cursor = len(m.text)
	}
}

// Cursor returns the current cursor position.
func (m *Model) Cursor() int { return m.cursor }

// SetCursor sets the cursor position, clamping to [0, len(text)].
func (m *Model) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(m.text) {
		pos = len(m.text)
	}
	m.cursor = pos
}

// CursorToEnd moves the cursor to the end of the text.
func (m *Model) CursorToEnd() { m.cursor = len(m.text) }

// Insert inserts s at the cursor position and advances the cursor.
func (m *Model) Insert(s string) {
	m.text = m.text[:m.cursor] + s + m.text[m.cursor:]
	m.cursor += len(s)
}

// Backspace deletes the character before the cursor.
func (m *Model) Backspace() bool {
	if m.cursor > 0 {
		m.text = m.text[:m.cursor-1] + m.text[m.cursor:]
		m.cursor--
		return true
	}
	return false
}

// Delete deletes the character at the cursor.
func (m *Model) Delete() bool {
	if m.cursor < len(m.text) {
		m.text = m.text[:m.cursor] + m.text[m.cursor+1:]
		return true
	}
	return false
}

// Left moves the cursor one position to the left.
func (m *Model) Left() { m.SetCursor(m.cursor - 1) }

// Right moves the cursor one position to the right.
func (m *Model) Right() { m.SetCursor(m.cursor + 1) }

// Home moves the cursor to position 0.
func (m *Model) Home() { m.cursor = 0 }

// End moves the cursor past the last character.
func (m *Model) End() { m.cursor = len(m.text) }

// HandleKeyMsg dispatches a tea.KeyMsg to the appropriate editing operation.
// It returns true if the key was handled (i.e. a text mutation or cursor move).
func (m *Model) HandleKeyMsg(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyBackspace:
		return m.Backspace()
	case tea.KeyDelete:
		return m.Delete()
	case tea.KeyLeft:
		m.Left()
		return true
	case tea.KeyRight:
		m.Right()
		return true
	case tea.KeyHome, tea.KeyCtrlA:
		m.Home()
		return true
	case tea.KeyEnd, tea.KeyCtrlE:
		m.End()
		return true
	case tea.KeySpace:
		m.Insert(" ")
		return true
	case tea.KeyRunes:
		m.Insert(string(msg.Runes))
		return true
	default:
		return false
	}
}

// ANSI escape sequences for reverse video.
const (
	reverseOn  = "\x1b[7m"
	reverseOff = "\x1b[27m"
)

// RenderLine renders the text as a single-line viewport of the given width,
// with the cursor shown in reverse video. suffix is appended after the viewport
// (e.g. an error indicator) and counts against the available width.
func (m *Model) RenderLine(width int, suffix string) string {
	if width <= 0 {
		return ""
	}

	suffixRunes := []rune(suffix)
	viewWidth := width - len(suffixRunes)
	if viewWidth < 1 {
		viewWidth = 1
	}

	runes := []rune(m.text)

	var cursorRune rune = ' '
	if m.cursor < len(runes) {
		cursorRune = runes[m.cursor]
	}

	start := 0
	if m.cursor >= viewWidth {
		start = m.cursor - viewWidth + 1
	}

	end := start + viewWidth
	if end > len(runes)+1 {
		end = len(runes) + 1
	}

	var before, after string
	if start < len(runes) {
		beforeEnd := m.cursor
		if beforeEnd > len(runes) {
			beforeEnd = len(runes)
		}
		if start < beforeEnd {
			before = string(runes[start:beforeEnd])
		}
	}

	afterStart := m.cursor + 1
	if afterStart < len(runes) {
		afterEnd := end
		if afterEnd > len(runes) {
			afterEnd = len(runes)
		}
		if afterStart < afterEnd {
			after = string(runes[afterStart:afterEnd])
		}
	}

	rendered := before + reverseOn + string(cursorRune) + reverseOff + after

	visibleLen := len([]rune(before)) + 1 + len([]rune(after))
	if visibleLen < viewWidth {
		rendered += strings.Repeat(" ", viewWidth-visibleLen)
	}

	return rendered + suffix
}

// TruncateOrPad truncates (with ellipsis) or right-pads s to exactly width runes.
func TruncateOrPad(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > width {
		if width > 1 {
			return string(runes[:width-1]) + "\u2026"
		}
		return string(runes[:width])
	}
	if len(runes) < width {
		return s + strings.Repeat(" ", width-len(runes))
	}
	return s
}
