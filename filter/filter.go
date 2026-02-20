// Package filter provides filtering interfaces and built-in filters for tea-grid.
package filter

import (
	tea "github.com/charmbracelet/bubbletea"
)

// FilterFocusMsg is sent to a filter when it begins interactive editing.
type FilterFocusMsg struct {
	Width    int // Available render width.
	MaxLines int // Maximum lines for the editor (up to 10).
}

// FilterBlurMsg is sent to a filter when interactive editing ends.
type FilterBlurMsg struct{}

// Filter is the interface that column filters implement.
type Filter interface {
	// Matches returns true if the value passes the filter.
	Matches(value any) bool

	// View renders the filter's UI (e.g., a text input in the header).
	// Returns empty string if the filter has no inline UI.
	View() string

	// Update processes messages for the filter's UI.
	Update(msg tea.Msg) (Filter, tea.Cmd)

	// Active returns true if the filter is currently constraining results.
	Active() bool

	// Clear resets the filter to its default (non-filtering) state.
	Clear()
}
