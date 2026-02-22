package grid

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/grouping"
)

// View renders the grid as a string.
func (m Model[T]) View() string {
	m.recomputeDisplayRows()

	if m.width == 0 || m.height == 0 {
		return ""
	}

	var sections []string

	// Quick filter bar
	if m.quickFilterActive {
		sections = append(sections, m.renderQuickFilter())
	}

	// Column filter editor
	if m.filterEditColIdx >= 0 {
		sections = append(sections, m.renderFilterEditor())
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

	// Pad or truncate to fill the grid dimensions
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, m.width, "")
	}
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

// renderFilterEditor renders the column filter editor above the header.
func (m Model[T]) renderFilterEditor() string {
	if m.filterEditColIdx < 0 || m.filterEditColIdx >= len(m.cols) {
		return ""
	}

	col := m.cols[m.filterEditColIdx]
	if col.Filter == nil {
		return ""
	}

	var lines []string

	// Title line
	title := "Filter: " + col.HeaderName
	lines = append(lines, m.styles.FilterInput.Width(m.width).Render(title))

	// Filter view lines
	view := col.Filter.View()
	if view != "" {
		for _, vl := range strings.Split(view, "\n") {
			lines = append(lines, m.styles.FilterInput.Width(m.width).Render(vl))
		}
	}

	return strings.Join(lines, "\n")
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
			for _, child := range group.Columns {
				for _, idx := range colIndices {
					if m.cols[idx].ColumnID == child.ColumnID {
						width += m.colWidths[idx]
					}
				}
			}
			if width > 0 {
				if m.styles.BorderColumn && len(group.Columns) > 1 {
					width += len(group.Columns) - 1
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
	visibleCenter := m.visibleCenterCols(center)
	cells = append(cells, renderGroupRegion(visibleCenter))
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

	leftSep, rightSep := m.scrollSeparators(center)

	if len(left) > 0 {
		parts = append(parts, m.renderHeaderCells(left))
		parts = append(parts, leftSep)
	} else if leftSep != m.styles.PinSeparator {
		parts = append(parts, leftSep)
	}

	// Only render visible center columns
	visibleCenter := m.visibleCenterCols(center)
	parts = append(parts, m.renderHeaderCells(visibleCenter))

	if len(right) > 0 {
		parts = append(parts, rightSep)
		parts = append(parts, m.renderHeaderCells(right))
	} else if rightSep != m.styles.PinSeparator {
		parts = append(parts, rightSep)
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

		// Add filter active indicator
		if col.Filter != nil && col.Filter.Active() {
			header += " " + m.styles.FilterActive
		}

		// Add sort indicator
		dir := m.sortModel.DirectionFor(col.ColumnID)
		switch dir {
		case data.SortAsc:
			header += " " + m.styles.SortAsc
		case data.SortDesc:
			header += " " + m.styles.SortDesc
		}

		// Apply header style
		style := m.styles.HeaderCell
		if m.focusedCell.Row == -1 && m.focusedCell.Col == idx && m.focused {
			style = m.styles.CellFocused
		}

		// Truncate header to content width (column width minus style padding/borders)
		contentWidth := w - style.GetHorizontalFrameSize()
		if contentWidth < 1 {
			contentWidth = 1
		}
		header = data.TruncateOrPad(header, contentWidth)

		cells = append(cells, style.Width(w).MaxWidth(w).Render(header))
	}
	return strings.Join(cells, m.colSeparator())
}

// renderHeaderBorder renders the border below the header.
func (m Model[T]) renderHeaderBorder() string {
	return strings.Repeat(m.styles.Border.Bottom, m.width)
}

// renderRow renders a single row.
func (m Model[T]) renderRow(rn *data.RowNode[T], displayIndex int, isPinned bool) string {
	if rn.IsGroup {
		return m.renderGroupRow(rn, displayIndex)
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

	return strings.Join(parts, "")
}

// renderCells renders cells for a row across specified column indices.
func (m Model[T]) renderCells(rn *data.RowNode[T], colIndices []int, displayIndex int, isPinned bool) string {
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
		if col.ColumnSpan != nil {
			span = col.ColumnSpan(rn.Data)
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
		colLo, colHi := m.colSelectionRange()
		isColSelected := colLo >= 0 && idx >= colLo && idx <= colHi
		isSelected := !col.NoSelect && (m.sel.IsSelected(rn.ID) || isColSelected)
		isFocused := m.focused && m.focusedCell.Row == displayIndex && m.focusedCell.Col == idx

		// Select base style based on row context
		style := m.styles.Cell
		if isPinned {
			style = m.styles.CellPinned
		} else if displayIndex >= 0 && displayIndex%2 != 0 {
			style = m.styles.CellOddRow
		} else if displayIndex >= 0 {
			style = m.styles.CellEvenRow
		}

		// Override for focus/selection
		if isFocused {
			style = m.styles.CellFocused
		} else if isSelected {
			style = m.styles.CellSelected
		}

		if m.styles.StyleFunc != nil {
			style = applyCustomStyle(m.styles.StyleFunc(displayIndex, idx, rn.Data), style, isFocused, isSelected)
		}
		if col.CellStyle != nil {
			style = applyCustomStyle(col.CellStyle(val, rn.Data), style, isFocused, isSelected)
		}

		// Compute content width (column width minus style padding/borders)
		contentWidth := w - style.GetHorizontalFrameSize()
		if contentWidth < 1 {
			contentWidth = 1
		}

		// Use custom renderer if available, otherwise default to formatted text
		cellContent := formatted
		ctx := data.CellContext[T]{
			Value:          val,
			FormattedValue: formatted,
			Data:           rn.Data,
			RowNode:        rn,
			Column:         &col,
			ColumnIndex:       idx,
			RowIndex:       displayIndex,
			IsSelected:     isSelected,
			IsFocused:      isFocused,
			Width:          contentWidth,
			Height:         1,
		}

		var renderer data.CellRenderer[T]
		if col.CellRendererSelector != nil {
			renderer = col.CellRendererSelector(rn.Data)
		}
		if renderer == nil {
			renderer = col.CellRenderer
		}
		if renderer != nil {
			cellContent = renderer.Render(ctx)
		} else {
			cellContent = ansi.Truncate(cellContent, contentWidth, "…")
		}

		cells = append(cells, style.Width(w).MaxWidth(w).Render(cellContent))
	}

	return strings.Join(cells, m.colSeparator())
}

// renderGroupRow renders a synthetic group row spanning all columns,
// followed by an aggregation row if any columns define aggregation functions.
func (m Model[T]) renderGroupRow(rn *data.RowNode[T], displayIndex int) string {
	indent := strings.Repeat(" ", rn.GroupLevel*m.styles.GroupIndent)

	indicator := m.styles.GroupCollapsed
	if rn.Expanded {
		indicator = m.styles.GroupExpanded
	}

	childCount := len(rn.Children)
	label := fmt.Sprintf("%s%s %s (%d)", indent, indicator, rn.GroupKey, childCount)

	isFocused := m.focused && m.focusedCell.Row == displayIndex

	style := m.styles.GroupRow
	if isFocused {
		style = m.styles.CellFocused
	}

	contentWidth := m.width - style.GetHorizontalFrameSize()
	if contentWidth < 1 {
		contentWidth = 1
	}

	cellContent := data.TruncateOrPad(label, contentWidth)
	labelLine := style.Width(m.width).MaxWidth(m.width).Render(cellContent)

	// Check if any columns have aggregation
	hasAgg := false
	for i := range m.cols {
		if m.cols[i].AggFuncCustom != nil || m.cols[i].AggFunc != "" {
			hasAgg = true
			break
		}
	}
	if !hasAgg {
		return labelLine
	}

	// Render aggregation row aligned with columns
	aggLine := m.renderAggRow(rn)
	return labelLine + "\n" + aggLine
}

// renderAggRow renders an aggregation row with values aligned to their columns.
func (m Model[T]) renderAggRow(rn *data.RowNode[T]) string {
	left, center, right := m.visibleColIndices()

	var parts []string

	if len(left) > 0 {
		parts = append(parts, m.renderAggCells(rn, left))
		parts = append(parts, m.styles.PinSeparator)
	}

	visibleCenter := m.visibleCenterCols(center)
	parts = append(parts, m.renderAggCells(rn, visibleCenter))

	if len(right) > 0 {
		parts = append(parts, m.styles.PinSeparator)
		parts = append(parts, m.renderAggCells(rn, right))
	}

	return strings.Join(parts, "")
}

// renderAggCells renders aggregation cells for specified column indices.
// Columns with an aggregation function show the computed value; others are blank.
func (m Model[T]) renderAggCells(rn *data.RowNode[T], colIndices []int) string {
	var cells []string
	style := m.styles.GroupRow

	for _, idx := range colIndices {
		col := m.cols[idx]
		w := m.colWidths[idx]

		contentWidth := w - style.GetHorizontalFrameSize()
		if contentWidth < 1 {
			contentWidth = 1
		}

		var cellContent string
		if col.AggFuncCustom != nil || col.AggFunc != "" {
			values := collectLeafValues(rn, &col)
			var aggResult any
			if col.AggFuncCustom != nil {
				aggResult = col.AggFuncCustom(values)
			} else {
				aggResult = grouping.Aggregate(values, col.AggFunc)
			}
			formatted := fmt.Sprintf("%v", aggResult)
			if col.ValueFormatter != nil {
				var zero T
				formatted = col.ValueFormatter(aggResult, zero)
			}
			cellContent = data.TruncateOrPad(formatted, contentWidth)
		} else {
			cellContent = data.TruncateOrPad("", contentWidth)
		}

		cells = append(cells, style.Width(w).MaxWidth(w).Render(cellContent))
	}

	return strings.Join(cells, m.colSeparator())
}

// collectLeafValues recursively walks a group node's children to collect
// all leaf-row values for a given column.
func collectLeafValues[T any](node *data.RowNode[T], col *data.Column[T]) []any {
	var values []any
	for _, child := range node.Children {
		if child.IsGroup {
			values = append(values, collectLeafValues(child, col)...)
		} else if col.ValueGetter != nil {
			values = append(values, col.ValueGetter(child.Data))
		}
	}
	return values
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

// applyCustomStyle merges a custom cell style with the current state style.
// Inherit handles colors, alignment, and text decorations. Padding is not
// inherited by lipgloss, so we carry it over manually when the custom style
// has none. When the cell is focused or selected, the colors from the state
// style are forced so the highlight remains visible.
func applyCustomStyle(custom, state lipgloss.Style, isFocused, isSelected bool) lipgloss.Style {
	result := custom.Inherit(state)

	// lipgloss.Inherit skips padding. If custom didn't set any,
	// preserve the base style's padding so content width and visual
	// spacing remain correct.
	t, r, b, l := result.GetPadding()
	if t == 0 && r == 0 && b == 0 && l == 0 {
		t, r, b, l = state.GetPadding()
		if t > 0 || r > 0 || b > 0 || l > 0 {
			result = result.Padding(t, r, b, l)
		}
	}

	if isFocused || isSelected {
		result = result.
			Foreground(state.GetForeground()).
			Background(state.GetBackground())
	}
	return result
}

// scrollSeparators computes the left and right separator strings for the center
// region edges. When columns are hidden off-screen in a given direction, the
// corresponding scroll indicator is returned instead of the pin separator.
func (m Model[T]) scrollSeparators(center []int) (left, right string) {
	left = m.styles.PinSeparator
	if m.vp.leftCol > 0 {
		left = m.styles.ScrollLeft
	}

	right = m.styles.PinSeparator
	if m.vp.leftCol+m.vp.visibleCols < len(center) {
		right = m.styles.ScrollRight
	}

	return left, right
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
