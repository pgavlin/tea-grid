package grid

import (
	"fmt"
	"strings"

	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/row"
)

// View renders the grid as a string. Implements tea.Model.
func (m Model[T]) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var sections []string

	// Quick filter bar
	if m.quickFilterActive {
		sections = append(sections, m.renderQuickFilter())
	}

	// Column group headers (if any)
	if len(m.colGroups) > 0 {
		sections = append(sections, m.renderGroupHeaders())
	}

	// Column headers
	sections = append(sections, m.renderHeader())

	// Header border
	if m.styles.BorderHeader {
		sections = append(sections, m.renderHeaderBorder())
	}

	// Pinned top rows
	for _, rn := range m.pinnedTop {
		sections = append(sections, m.renderRow(&rn, -1, true))
	}
	if len(m.pinnedTop) > 0 {
		sections = append(sections, m.renderSeparator())
	}

	// Body rows (virtual scrolled)
	start, end := m.vp.visibleRowRange(len(m.displayRows))
	for i := start; i < end; i++ {
		sections = append(sections, m.renderRow(&m.displayRows[i], i, false))
		if m.styles.BorderRow && i < end-1 {
			sections = append(sections, m.renderRowBorder())
		}
	}

	// Pinned bottom rows
	if len(m.pinnedBot) > 0 {
		sections = append(sections, m.renderSeparator())
	}
	for _, rn := range m.pinnedBot {
		sections = append(sections, m.renderRow(&rn, -1, true))
	}

	content := strings.Join(sections, "\n")

	// Pad or truncate to fill the grid height
	lines := strings.Split(content, "\n")
	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}
	if len(lines) > m.height {
		lines = lines[:m.height]
	}

	result := strings.Join(lines, "\n")

	// Apply table-level style
	if m.styles.Table.Value() != "" {
		result = m.styles.Table.Render(result)
	}

	return result
}

// renderQuickFilter renders the quick filter input bar.
func (m Model[T]) renderQuickFilter() string {
	label := "Filter: "
	input := m.quickFilterText
	if input == "" {
		input = "Type to filter..."
	}
	line := m.styles.FilterInput.Width(m.width).Render(label + input)
	return line
}

// renderGroupHeaders renders the column group header row.
func (m Model[T]) renderGroupHeaders() string {
	left, center, right := m.visibleColIndices()

	var cells []string

	// Render group spans
	renderGroupRegion := func(colIndices []int) string {
		var parts []string
		for _, group := range m.colGroups {
			width := 0
			for _, child := range group.Children {
				for _, idx := range colIndices {
					if m.cols[idx].ColID == child.ColID {
						width += m.colWidths[idx]
					}
				}
			}
			if width > 0 {
				if m.styles.BorderColumn && len(group.Children) > 1 {
					width += len(group.Children) - 1
				}
				parts = append(parts, m.styles.HeaderCell.Width(width).Render(group.HeaderName))
			}
		}
		return strings.Join(parts, m.colSeparator())
	}

	if len(left) > 0 {
		cells = append(cells, renderGroupRegion(left))
		cells = append(cells, m.styles.PinSeparator)
	}
	cells = append(cells, renderGroupRegion(center))
	if len(right) > 0 {
		cells = append(cells, m.styles.PinSeparator)
		cells = append(cells, renderGroupRegion(right))
	}

	return strings.Join(cells, "")
}

// renderHeader renders the column header row.
func (m Model[T]) renderHeader() string {
	left, center, right := m.visibleColIndices()

	var parts []string

	if len(left) > 0 {
		parts = append(parts, m.renderHeaderCells(left))
		parts = append(parts, m.styles.PinSeparator)
	}

	// Only render visible center columns
	visibleCenter := m.visibleCenterCols(center)
	parts = append(parts, m.renderHeaderCells(visibleCenter))

	if len(right) > 0 {
		parts = append(parts, m.styles.PinSeparator)
		parts = append(parts, m.renderHeaderCells(right))
	}

	return strings.Join(parts, "")
}

// renderHeaderCells renders header cells for a set of column indices.
func (m Model[T]) renderHeaderCells(colIndices []int) string {
	var cells []string
	for _, idx := range colIndices {
		col := m.cols[idx]
		w := m.colWidths[idx]

		header := col.HeaderName

		// Add sort indicator
		dir := m.sortModel.DirectionFor(col.ColID)
		switch dir {
		case column.SortAsc:
			header += " " + m.styles.SortAsc
		case column.SortDesc:
			header += " " + m.styles.SortDesc
		}

		// Apply header style
		style := m.styles.HeaderCell
		if m.focusedCell.Row == -1 && m.focusedCell.Col == idx && m.focused {
			style = m.styles.CellFocused
		}

		cells = append(cells, style.Width(w).MaxWidth(w).Render(header))
	}
	return strings.Join(cells, m.colSeparator())
}

// renderHeaderBorder renders the border below the header.
func (m Model[T]) renderHeaderBorder() string {
	return strings.Repeat(m.styles.Border.Bottom, m.width)
}

// renderRow renders a single row.
func (m Model[T]) renderRow(rn *row.RowNode[T], displayIndex int, isPinned bool) string {
	if rn.IsGroup {
		return m.renderGroupRow(rn)
	}

	left, center, right := m.visibleColIndices()

	var parts []string

	if len(left) > 0 {
		parts = append(parts, m.renderCells(rn, left, displayIndex, isPinned))
		parts = append(parts, m.styles.PinSeparator)
	}

	visibleCenter := m.visibleCenterCols(center)
	parts = append(parts, m.renderCells(rn, visibleCenter, displayIndex, isPinned))

	if len(right) > 0 {
		parts = append(parts, m.styles.PinSeparator)
		parts = append(parts, m.renderCells(rn, right, displayIndex, isPinned))
	}

	line := strings.Join(parts, "")

	// Apply row-level styles
	if isPinned {
		line = m.styles.PinnedRow.Render(line)
	} else if displayIndex >= 0 {
		if displayIndex%2 == 0 {
			if m.styles.EvenRow.Value() != "" {
				line = m.styles.EvenRow.Render(line)
			}
		} else {
			if m.styles.OddRow.Value() != "" {
				line = m.styles.OddRow.Render(line)
			}
		}
	}

	return line
}

// renderCells renders cells for a row across specified column indices.
func (m Model[T]) renderCells(rn *row.RowNode[T], colIndices []int, displayIndex int, isPinned bool) string {
	var cells []string

	skipUntil := -1
	for _, idx := range colIndices {
		if idx < skipUntil {
			continue
		}

		col := m.cols[idx]
		w := m.colWidths[idx]

		// Handle column spanning
		span := 1
		if col.ColSpan != nil {
			span = col.ColSpan(rn.Data)
		}
		if span > 1 {
			// Add widths of spanned columns
			for s := 1; s < span; s++ {
				nextIdx := idx + s
				if nextIdx < len(m.cols) {
					w += m.colWidths[nextIdx]
					if m.styles.BorderColumn {
						w++ // account for separator
					}
				}
			}
			skipUntil = idx + span
		}

		// Get value
		var val any
		if col.ValueGetter != nil {
			val = col.ValueGetter(rn.Data)
		}

		// Format
		formatted := fmt.Sprintf("%v", val)
		if col.ValueFormatter != nil {
			formatted = col.ValueFormatter(val, rn.Data)
		}

		// Check if this cell is being edited
		if m.editState != nil && m.editState.position.Row == displayIndex && m.editState.position.Col == idx {
			editorView := m.editState.editor.View()
			cells = append(cells, m.styles.EditorInput.Width(w).MaxWidth(w).Render(editorView))
			continue
		}

		// Determine style
		isSelected := m.sel.IsSelected(rn.ID)
		isFocused := m.focused && m.focusedCell.Row == displayIndex && m.focusedCell.Col == idx

		style := m.styles.Cell
		if isFocused {
			style = m.styles.CellFocused
		} else if isSelected {
			style = m.styles.CellSelected
		}

		if m.styles.StyleFunc != nil {
			style = m.styles.StyleFunc(displayIndex, idx, rn.Data)
		}
		if col.CellStyle != nil {
			style = col.CellStyle(val, rn.Data)
		}

		// Use custom renderer if available or default text rendering
		cellContent := formatted
		cells = append(cells, style.Width(w).MaxWidth(w).Render(cellContent))
	}

	return strings.Join(cells, m.colSeparator())
}

// renderGroupRow renders a synthetic group row.
func (m Model[T]) renderGroupRow(rn *row.RowNode[T]) string {
	indent := strings.Repeat(" ", rn.GroupLevel*m.styles.GroupIndent)

	indicator := m.styles.GroupCollapsed
	if rn.Expanded {
		indicator = m.styles.GroupExpanded
	}

	childCount := len(rn.Children)
	label := fmt.Sprintf("%s%s %s (%d)", indent, indicator, rn.GroupKey, childCount)

	isFocused := m.focused && m.focusedCell.Row >= 0 && m.focusedCell.Row < len(m.displayRows) &&
		m.displayRows[m.focusedCell.Row].IsGroup && m.displayRows[m.focusedCell.Row].GroupKey == rn.GroupKey

	style := m.styles.GroupRow
	if isFocused {
		style = m.styles.CellFocused
	}

	return style.Width(m.width).Render(label)
}

// renderSeparator renders a horizontal separator between pinned and unpinned regions.
func (m Model[T]) renderSeparator() string {
	return strings.Repeat("─", m.width)
}

// renderRowBorder renders a border between rows.
func (m Model[T]) renderRowBorder() string {
	return strings.Repeat(m.styles.Border.Middle, m.width)
}

// colSeparator returns the column separator string.
func (m Model[T]) colSeparator() string {
	if m.styles.BorderColumn {
		return "│"
	}
	return ""
}

// visibleCenterCols returns the subset of center columns that are visible in the viewport.
func (m Model[T]) visibleCenterCols(center []int) []int {
	if len(center) == 0 {
		return nil
	}

	start := m.vp.leftCol
	if start >= len(center) {
		start = len(center) - 1
	}
	if start < 0 {
		start = 0
	}

	end := start + m.vp.visibleCols
	if end > len(center) {
		end = len(center)
	}

	return center[start:end]
}
