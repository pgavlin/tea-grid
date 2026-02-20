// Package selection provides row selection state for the tea-grid component.
package selection

// Mode defines how row selection behaves.
type Mode int

const (
	SelectNone   Mode = iota // No selection.
	SelectSingle             // At most one row selected.
	SelectMulti              // Multiple rows via Space, Shift+arrows, Ctrl+A.
)

// Model holds the current selection state.
type Model struct {
	Mode     Mode
	selected map[string]bool // RowNode.ID -> selected
	anchor   int             // For shift-selection range
}

// New creates a new selection model with the given mode.
func New(mode Mode) Model {
	return Model{
		Mode:     mode,
		selected: make(map[string]bool),
		anchor:   -1,
	}
}

// IsSelected returns true if the row with the given ID is selected.
func (m *Model) IsSelected(id string) bool {
	return m.selected[id]
}

// Toggle toggles the selection state of a row.
// In SelectSingle mode, selecting a new row deselects the previous one.
func (m *Model) Toggle(id string) {
	if m.Mode == SelectNone {
		return
	}
	if m.selected[id] {
		delete(m.selected, id)
	} else {
		if m.Mode == SelectSingle {
			m.selected = make(map[string]bool)
		}
		m.selected[id] = true
	}
}

// Select selects a row by ID.
func (m *Model) Select(id string) {
	if m.Mode == SelectNone {
		return
	}
	if m.Mode == SelectSingle {
		m.selected = make(map[string]bool)
	}
	m.selected[id] = true
}

// Deselect deselects a row by ID.
func (m *Model) Deselect(id string) {
	delete(m.selected, id)
}

// SelectAll marks all given IDs as selected.
func (m *Model) SelectAll(ids []string) {
	if m.Mode != SelectMulti {
		return
	}
	for _, id := range ids {
		m.selected[id] = true
	}
}

// DeselectAll clears all selections.
func (m *Model) DeselectAll() {
	m.selected = make(map[string]bool)
}

// SelectedIDs returns all selected row IDs.
func (m *Model) SelectedIDs() []string {
	ids := make([]string, 0, len(m.selected))
	for id := range m.selected {
		ids = append(ids, id)
	}
	return ids
}

// Count returns the number of selected rows.
func (m *Model) Count() int {
	return len(m.selected)
}

// SetAnchor sets the anchor index for range selection.
func (m *Model) SetAnchor(index int) {
	m.anchor = index
}

// Anchor returns the current anchor index.
func (m *Model) Anchor() int {
	return m.anchor
}
