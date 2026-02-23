// Package selection provides rectangle-based selection state for the tea-grid component.
package selection

// Mode defines how selection behaves.
type Mode int

const (
	SelectNone   Mode = iota // No selection.
	SelectSingle             // At most one rect selected.
	SelectMulti              // Multiple rects via Space, Shift+arrows, Ctrl+A.
)

// Kind identifies the type of a selection rectangle.
type Kind int

const (
	KindNone    Kind = iota
	KindRect                // Arbitrary rectangle (Shift+nav)
	KindFullRow             // Full-row selection (R, Space, Ctrl+A) — all columns
	KindFullCol             // Full-column selection (C key) — all rows
)

// Position identifies a cell by row and column index.
type Position struct {
	Row, Col int
}

// Rect represents a selection rectangle with a fixed anchor and a moving cursor.
type Rect struct {
	Kind   Kind
	Anchor Position // Fixed corner
	Cursor Position // Moving corner
}

// RowRange returns the ordered row range of the rectangle.
func (r Rect) RowRange() (lo, hi int) {
	lo, hi = r.Anchor.Row, r.Cursor.Row
	if lo > hi {
		lo, hi = hi, lo
	}
	return
}

// ColRange returns the ordered column range of the rectangle.
func (r Rect) ColRange() (lo, hi int) {
	lo, hi = r.Anchor.Col, r.Cursor.Col
	if lo > hi {
		lo, hi = hi, lo
	}
	return
}

// Model holds the current selection state as a list of rectangles.
type Model struct {
	Mode  Mode
	Rects []Rect
}

// New creates a new selection model with the given mode.
func New(mode Mode) Model {
	return Model{Mode: mode}
}

// Active returns true if there is at least one selection rectangle.
func (m *Model) Active() bool {
	return len(m.Rects) > 0
}

// Clear removes all selection rectangles.
func (m *Model) Clear() {
	m.Rects = nil
}

// Replace sets the selection to exactly one rectangle.
// In SelectNone mode this is a no-op.
func (m *Model) Replace(r Rect) {
	if m.Mode == SelectNone {
		return
	}
	m.Rects = []Rect{r}
}

// ToggleFullRow adds or removes a KindFullRow rect for the given row.
// Non-KindFullRow rects are cleared first. If an existing KindFullRow rect
// matches exactly (row, row), it is removed; otherwise a new one is added.
// In SelectSingle mode, the list is cleared before adding.
func (m *Model) ToggleFullRow(row int) {
	if m.Mode == SelectNone {
		return
	}

	// Remove any non-KindFullRow rects
	filtered := m.Rects[:0]
	for _, r := range m.Rects {
		if r.Kind == KindFullRow {
			filtered = append(filtered, r)
		}
	}
	m.Rects = filtered

	// Search for an existing single-row KindFullRow matching exactly this row
	for i, r := range m.Rects {
		rLo, rHi := r.RowRange()
		if rLo == row && rHi == row {
			// Remove it (toggle off)
			m.Rects = append(m.Rects[:i], m.Rects[i+1:]...)
			return
		}
	}

	// Add new rect
	newRect := Rect{
		Kind:   KindFullRow,
		Anchor: Position{Row: row, Col: 0},
		Cursor: Position{Row: row, Col: 0},
	}

	if m.Mode == SelectSingle {
		m.Rects = []Rect{newRect}
	} else {
		m.Rects = append(m.Rects, newRect)
	}
}

// ContainsCell returns true if the cell at (row, col) is within any selection rectangle.
func (m *Model) ContainsCell(row, col int) bool {
	if len(m.Rects) == 0 {
		return false
	}
	for _, r := range m.Rects {
		switch r.Kind {
		case KindFullRow:
			rLo, rHi := r.RowRange()
			if row >= rLo && row <= rHi {
				return true
			}
		case KindFullCol:
			cLo, cHi := r.ColRange()
			if col >= cLo && col <= cHi {
				return true
			}
		case KindRect:
			rLo, rHi := r.RowRange()
			cLo, cHi := r.ColRange()
			if row >= rLo && row <= rHi && col >= cLo && col <= cHi {
				return true
			}
		}
	}
	return false
}

// FullRowRanges returns sorted (lo, hi) pairs for all KindFullRow rects.
func (m *Model) FullRowRanges() [][2]int {
	var ranges [][2]int
	for _, r := range m.Rects {
		if r.Kind == KindFullRow {
			lo, hi := r.RowRange()
			ranges = append(ranges, [2]int{lo, hi})
		}
	}
	return ranges
}

// BoundingRect returns the union bounding box of all rects,
// or (-1, -1, -1, -1) if there are no rects.
func (m *Model) BoundingRect() (rowLo, rowHi, colLo, colHi int) {
	if len(m.Rects) == 0 {
		return -1, -1, -1, -1
	}

	rowLo, rowHi = m.Rects[0].RowRange()
	colLo, colHi = m.Rects[0].ColRange()

	for _, r := range m.Rects[1:] {
		rLo, rHi := r.RowRange()
		cLo, cHi := r.ColRange()
		if rLo < rowLo {
			rowLo = rLo
		}
		if rHi > rowHi {
			rowHi = rHi
		}
		if cLo < colLo {
			colLo = cLo
		}
		if cHi > colHi {
			colHi = cHi
		}
	}
	return
}
