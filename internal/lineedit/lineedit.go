// Package lineedit provides a minimal single-line text editing model.
// It encapsulates the cursor, text manipulation, and rendering logic shared
// by the built-in filters and cell editors.
package lineedit

import (
	"strings"

	tea "charm.land/bubbletea/v2"
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

// KillToEnd deletes from the cursor to the end of the text. Returns
// true if any characters were removed.
func (m *Model) KillToEnd() bool {
	if m.cursor >= len(m.text) {
		return false
	}
	m.text = m.text[:m.cursor]
	return true
}

// KillToStart deletes from the start of the text to the cursor.
// Returns true if any characters were removed.
func (m *Model) KillToStart() bool {
	if m.cursor == 0 {
		return false
	}
	m.text = m.text[m.cursor:]
	m.cursor = 0
	return true
}

// WordLeft moves the cursor to the start of the previous whitespace-
// delimited word. If the cursor is mid-word it moves to the start of
// the current word; otherwise it skips trailing whitespace and lands
// at the start of the prior word.
func (m *Model) WordLeft() {
	if m.cursor == 0 {
		return
	}
	i := m.cursor
	for i > 0 && isWordSep(m.text[i-1]) {
		i--
	}
	for i > 0 && !isWordSep(m.text[i-1]) {
		i--
	}
	m.cursor = i
}

// WordRight moves the cursor to the start of the next whitespace-
// delimited word, or to the end of the text if no further word exists.
func (m *Model) WordRight() {
	n := len(m.text)
	if m.cursor >= n {
		return
	}
	i := m.cursor
	for i < n && !isWordSep(m.text[i]) {
		i++
	}
	for i < n && isWordSep(m.text[i]) {
		i++
	}
	m.cursor = i
}

// KillWordLeft deletes from the cursor to the start of the previous
// whitespace-delimited word. Returns true if any characters were
// removed.
func (m *Model) KillWordLeft() bool {
	if m.cursor == 0 {
		return false
	}
	end := m.cursor
	m.WordLeft()
	if m.cursor == end {
		return false
	}
	m.text = m.text[:m.cursor] + m.text[end:]
	return true
}

// KillWordRight deletes from the cursor to the start of the next
// whitespace-delimited word. Returns true if any characters were
// removed.
func (m *Model) KillWordRight() bool {
	if m.cursor >= len(m.text) {
		return false
	}
	start := m.cursor
	end := m.cursor
	n := len(m.text)
	for end < n && !isWordSep(m.text[end]) {
		end++
	}
	for end < n && isWordSep(m.text[end]) {
		end++
	}
	if end == start {
		return false
	}
	m.text = m.text[:start] + m.text[end:]
	return true
}

// isWordSep reports whether b separates words for cursor-by-word
// movement and word-kill operations. Whitespace is the only separator;
// punctuation (`:`, `,`, `>`, `=`, etc.) is part of a word so that a
// query token like `state:open` moves and kills as a unit.
func isWordSep(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// HandleKeyMsg dispatches a tea.KeyPressMsg to the appropriate editing operation.
// It returns true if the key was handled (i.e. a text mutation or cursor move).
func (m *Model) HandleKeyMsg(msg tea.KeyPressMsg) bool {
	// Alt+key (Meta) combinations: word-level movement and editing.
	if msg.Mod.Contains(tea.ModAlt) {
		switch msg.Code {
		case 'b':
			m.WordLeft()
			return true
		case 'f':
			m.WordRight()
			return true
		case 'd':
			return m.KillWordRight()
		case tea.KeyBackspace:
			return m.KillWordLeft()
		}
	}

	// Ctrl+key combinations: readline-style line editing.
	if msg.Mod.Contains(tea.ModCtrl) {
		switch msg.Code {
		case 'a':
			m.Home()
			return true
		case 'e':
			m.End()
			return true
		case 'b':
			m.Left()
			return true
		case 'f':
			m.Right()
			return true
		case 'd':
			return m.Delete()
		case 'h':
			return m.Backspace()
		case 'k':
			return m.KillToEnd()
		case 'u':
			return m.KillToStart()
		case 'w':
			return m.KillWordLeft()
		}
	}

	switch msg.Code {
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
	case tea.KeyHome:
		m.Home()
		return true
	case tea.KeyEnd:
		m.End()
		return true
	case tea.KeySpace:
		m.Insert(" ")
		return true
	default:
		if len(msg.Text) > 0 {
			m.Insert(msg.Text)
			return true
		}
		return false
	}
}

// ANSI escape sequences for reverse video and faint (dim) text.
const (
	reverseOn  = "\x1b[7m"
	reverseOff = "\x1b[27m"
	faintOn    = "\x1b[2m"
	faintOff   = "\x1b[22m"
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

	cursorRune := ' '
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

// RenderLineDim is like RenderLine but additionally renders runes in
// the half-open range [dimStart, dimEnd) with faint (dim) styling.
// Useful for previewing tab-completion suggestions: the "added"
// portion of the text appears visually distinct from what the user
// typed. Pass dimStart >= dimEnd to disable the dim styling
// (equivalent to RenderLine).
func (m *Model) RenderLineDim(width int, suffix string, dimStart, dimEnd int) string {
	if width <= 0 {
		return ""
	}
	if dimStart >= dimEnd {
		return m.RenderLine(width, suffix)
	}

	suffixRunes := []rune(suffix)
	viewWidth := width - len(suffixRunes)
	if viewWidth < 1 {
		viewWidth = 1
	}
	runes := []rune(m.text)

	start := 0
	if m.cursor >= viewWidth {
		start = m.cursor - viewWidth + 1
	}
	end := start + viewWidth
	if end > len(runes)+1 {
		end = len(runes) + 1
	}

	var b strings.Builder
	inDim := false
	for i := start; i < end; i++ {
		var ch rune
		if i < len(runes) {
			ch = runes[i]
		} else {
			ch = ' '
		}

		isCursor := i == m.cursor
		shouldDim := i >= dimStart && i < dimEnd

		if shouldDim && !inDim {
			b.WriteString(faintOn)
			inDim = true
		} else if !shouldDim && inDim {
			b.WriteString(faintOff)
			inDim = false
		}

		if isCursor {
			b.WriteString(reverseOn)
			b.WriteRune(ch)
			b.WriteString(reverseOff)
		} else {
			b.WriteRune(ch)
		}
	}
	if inDim {
		b.WriteString(faintOff)
	}

	visible := end - start
	if visible < viewWidth {
		b.WriteString(strings.Repeat(" ", viewWidth-visible))
	}
	b.WriteString(suffix)
	return b.String()
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
