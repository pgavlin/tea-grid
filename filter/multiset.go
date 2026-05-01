package filter

import (
	tea "charm.land/bubbletea/v2"
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

// View, Update implemented in subsequent tasks.
func (f *MultiSetFilter) View() string                         { return "" }
func (f *MultiSetFilter) Update(msg tea.Msg) (Filter, tea.Cmd) { return f, nil }

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
