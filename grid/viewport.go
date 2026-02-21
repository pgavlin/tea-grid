package grid

// viewport manages virtual scrolling state.
type viewport struct {
	topRow      int // Index of the first visible row.
	leftCol     int // Index of the first visible (unpinned) column.
	visibleRows int // Number of rows that fit in the viewport height.
	visibleCols int // Number of columns that fit in the viewport width.
}

// newViewport creates a new viewport with default settings.
func newViewport() viewport {
	return viewport{}
}

// ensureVisible adjusts the scroll position to ensure the given row is visible.
func (v *viewport) ensureRowVisible(row int) {
	if row < v.topRow {
		v.topRow = row
	} else if row >= v.topRow+v.visibleRows {
		v.topRow = row - v.visibleRows + 1
	}
	if v.topRow < 0 {
		v.topRow = 0
	}
}

// ensureColVisible adjusts the scroll position to ensure the given column is visible.
func (v *viewport) ensureColVisible(col int) {
	if col < v.leftCol {
		v.leftCol = col
	} else if col >= v.leftCol+v.visibleCols {
		v.leftCol = col - v.visibleCols + 1
	}
	if v.leftCol < 0 {
		v.leftCol = 0
	}
}

// visibleRowRange returns the start and end indices of visible rows.
func (v *viewport) visibleRowRange(totalRows int) (start, end int) {
	start = v.topRow
	if start < 0 {
		start = 0
	}
	end = v.topRow + v.visibleRows
	if end > totalRows {
		end = totalRows
	}
	return start, end
}

// scrollToTop scrolls to the first row.
func (v *viewport) scrollToTop() {
	v.topRow = 0
}

// scrollToBottom scrolls to the last page.
func (v *viewport) scrollToBottom(totalRows int) {
	v.topRow = totalRows - v.visibleRows
	if v.topRow < 0 {
		v.topRow = 0
	}
}
