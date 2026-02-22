package grid

// viewport manages virtual scrolling state.
type viewport struct {
	topRow       int // Index of the first visible row.
	leftCol      int // Index of the first visible (unpinned) column.
	visibleLines int // Number of terminal lines that fit in the viewport height.
	visibleCols  int // Number of columns that fit in the viewport width.
}

// newViewport creates a new viewport with default settings.
func newViewport() viewport {
	return viewport{}
}

// ensureRowVisible adjusts the scroll position to ensure the given row is visible.
// rowHeight returns the height in terminal lines for a given row index.
func (v *viewport) ensureRowVisible(row int, totalRows int, rowHeight func(int) int) {
	if row < v.topRow {
		v.topRow = row
	} else {
		// Check if row is already visible by walking from topRow
		usedLines := 0
		visible := false
		for i := v.topRow; i < totalRows; i++ {
			h := rowHeight(i)
			if i == row && usedLines+h <= v.visibleLines {
				visible = true
				break
			}
			usedLines += h
			if usedLines >= v.visibleLines {
				break
			}
		}
		if !visible {
			// row is below the visible area — scroll down
			// Walk backward from row to find the new topRow
			usedLines := rowHeight(row)
			v.topRow = row
			for i := row - 1; i >= 0; i-- {
				h := rowHeight(i)
				if usedLines+h > v.visibleLines {
					break
				}
				usedLines += h
				v.topRow = i
			}
		}
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
// rowHeight returns the height in terminal lines for a given row index.
func (v *viewport) visibleRowRange(totalRows int, rowHeight func(int) int) (start, end int) {
	start = v.topRow
	if start < 0 {
		start = 0
	}
	usedLines := 0
	end = start
	for end < totalRows {
		h := rowHeight(end)
		if usedLines+h > v.visibleLines {
			break
		}
		usedLines += h
		end++
	}
	// Ensure at least one row is visible
	if end == start && start < totalRows {
		end = start + 1
	}
	return
}

// scrollToTop scrolls to the first row.
func (v *viewport) scrollToTop() {
	v.topRow = 0
}

// scrollToBottom scrolls to the last page.
// rowHeight returns the height in terminal lines for a given row index.
func (v *viewport) scrollToBottom(totalRows int, rowHeight func(int) int) {
	usedLines := 0
	v.topRow = totalRows
	for i := totalRows - 1; i >= 0; i-- {
		h := rowHeight(i)
		if usedLines+h > v.visibleLines {
			break
		}
		usedLines += h
		v.topRow = i
	}
	if v.topRow < 0 {
		v.topRow = 0
	}
}
