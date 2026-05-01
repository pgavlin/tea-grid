package filter

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pgavlin/tea-grid/internal/lineedit"
)

// MultiSetFilter matches a value against a set of constraints with AND
// semantics: the value passes only if every constraint is satisfied.
// Useful for slice-valued columns ([]string labels, []Tag, etc.).
//
// Constraints are added via AddConstraint or by repeated SetClause
// calls from the query bar. The popup UI is view-and-remove only;
// new constraints come from the query bar.
type MultiSetFilter struct {
	constraints []string
	matcher     func(rowValue any, constraint string) bool

	// Editing state for the minimal popup.
	editing  bool
	width    int
	maxLines int
	focused  int
}

// MultiSetOption configures a MultiSetFilter at construction time.
type MultiSetOption func(*MultiSetFilter)

// WithMultiSetMatcher overrides the default matcher (which expects
// rowValue to be []string and matches by element equality).
func WithMultiSetMatcher(m func(rowValue any, constraint string) bool) MultiSetOption {
	return func(f *MultiSetFilter) {
		f.matcher = m
	}
}

// NewMultiSetFilter returns a MultiSetFilter with no constraints.
func NewMultiSetFilter(opts ...MultiSetOption) *MultiSetFilter {
	f := &MultiSetFilter{
		matcher: defaultMultiSetMatcher,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// AddConstraint appends a constraint. Duplicates are ignored.
func (f *MultiSetFilter) AddConstraint(c string) {
	for _, existing := range f.constraints {
		if existing == c {
			return
		}
	}
	f.constraints = append(f.constraints, c)
}

// RemoveConstraint removes the first occurrence of c. No-op if absent.
func (f *MultiSetFilter) RemoveConstraint(c string) {
	for i, existing := range f.constraints {
		if existing == c {
			f.constraints = append(f.constraints[:i], f.constraints[i+1:]...)
			if f.focused >= len(f.constraints) && f.focused > 0 {
				f.focused--
			}
			return
		}
	}
}

// Constraints returns the current constraint list (a copy).
func (f *MultiSetFilter) Constraints() []string {
	out := make([]string, len(f.constraints))
	copy(out, f.constraints)
	return out
}

// Clear removes all constraints.
func (f *MultiSetFilter) Clear() {
	f.constraints = nil
	f.focused = 0
}

// Matches reports whether rowValue satisfies every constraint.
func (f *MultiSetFilter) Matches(value any) bool {
	for _, c := range f.constraints {
		if !f.matcher(value, c) {
			return false
		}
	}
	return true
}

// Active reports whether the filter has any constraints.
func (f *MultiSetFilter) Active() bool {
	return len(f.constraints) > 0
}

// View renders the constraint list with row focus and a footer hint.
// Empty when the popup is not in editing mode.
func (f *MultiSetFilter) View() string {
	if !f.editing {
		return ""
	}
	var lines []string
	for i, c := range f.constraints {
		entry := "× " + c
		if f.width > 0 {
			entry = lineedit.TruncateOrPad(entry, f.width)
		}
		if i == f.focused {
			entry = lipgloss.NewStyle().Reverse(true).Render(entry)
		}
		lines = append(lines, entry)
	}
	footer := "d delete · esc close · / edit"
	if f.width > 0 {
		footer = lineedit.TruncateOrPad(footer, f.width)
	}
	lines = append(lines, footer)
	if f.maxLines > 0 && len(lines) > f.maxLines {
		lines = lines[:f.maxLines]
	}
	return strings.Join(lines, "\n")
}

// Update handles popup interactions: focus navigation and constraint
// deletion.
func (f *MultiSetFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	switch msg := msg.(type) {
	case FilterFocusMsg:
		f.editing = true
		f.width = msg.Width
		f.maxLines = msg.MaxLines
		f.focused = 0
		return f, nil
	case FilterBlurMsg:
		f.editing = false
		return f, nil
	case tea.KeyPressMsg:
		if !f.editing {
			return f, nil
		}
		switch msg.Code {
		case tea.KeyUp:
			if f.focused > 0 {
				f.focused--
			}
		case tea.KeyDown:
			if f.focused < len(f.constraints)-1 {
				f.focused++
			}
		case tea.KeyBackspace:
			f.deleteFocused()
		default:
			if msg.Text == "d" {
				f.deleteFocused()
			}
		}
	}
	return f, nil
}

func (f *MultiSetFilter) deleteFocused() {
	if f.focused < 0 || f.focused >= len(f.constraints) {
		return
	}
	f.constraints = append(f.constraints[:f.focused], f.constraints[f.focused+1:]...)
	if f.focused >= len(f.constraints) && f.focused > 0 {
		f.focused--
	}
}

// defaultMultiSetMatcher handles the common case: rowValue is []string,
// constraint matches when the slice contains the constraint.
func defaultMultiSetMatcher(rowValue any, constraint string) bool {
	switch v := rowValue.(type) {
	case []string:
		for _, s := range v {
			if s == constraint {
				return true
			}
		}
		return false
	case string:
		return v == constraint
	default:
		return false
	}
}
