package grid

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pgavlin/stain"
	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/grouping"
	"github.com/pgavlin/tea-grid/internal/conv"
)

// View renders the grid as a string.
func (m Model[T]) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var sections []string

	// Query bar (visible when enabled and either editing, has text, has lossy
	// filters, or has a parse error — otherwise collapses to nothing).
	if m.queryBarVisible() {
		sections = append(sections, m.renderQueryBar())
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
		sections = append(sections, m.renderRow(rn, -1, true))
	}
	if len(m.pinnedTop) > 0 {
		sections = append(sections, m.renderSeparator())
	}

	// Body rows (virtual scrolled)
	start, end := m.vp.visibleRowRange(len(m.displayRows), m.rowHeightFunc())
	for i := start; i < end; i++ {
		sections = append(sections, m.renderRow(m.displayRows[i], i, false))
		if m.styles.BorderRow && i < end-1 {
			sections = append(sections, m.renderRowBorder())
		}
	}

	// Pinned bottom rows
	if len(m.pinnedBot) > 0 {
		sections = append(sections, m.renderSeparator())
	}
	for _, rn := range m.pinnedBot {
		sections = append(sections, m.renderRow(rn, -1, true))
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

// renderQueryBar renders the query bar above the headers. Visible
// when the bar is enabled and there is something to show — otherwise
// queryBarVisible() returns false and View skips this section. While
// editing, the bar renders the lineedit (cursor visible, supports
// arrow keys, mid-string insertion, etc.); otherwise it shows the
// canonical text. Lossy annotations and parse errors are appended
// after the body in both modes.
func (m Model[T]) renderQueryBar() string {
	label := m.styles.QueryBarLossy.Render("⌕ ")
	labelWidth := lipgloss.Width(label)
	var body string
	if m.queryBarActive {
		editorWidth := m.width - labelWidth - m.styles.QueryBar.GetHorizontalFrameSize()
		if editorWidth < 1 {
			editorWidth = 1
		}
		dimStart, dimEnd, _ := m.queryBar.CompletionRange()
		body = label + m.queryBar.Editor().RenderLineDim(editorWidth, "", dimStart, dimEnd)
	} else {
		body = label + m.queryBar.Text()
	}

	lossy := m.queryBar.Lossy()
	if len(lossy) > 0 {
		hint := ""
		if !m.queryBarActive {
			hint = " — esc to clear all"
		}
		annotation := fmt.Sprintf("  [+%d hidden filter%s: %s%s]",
			len(lossy), pluralS(len(lossy)), strings.Join(lossy, ", "), hint)
		body += m.styles.QueryBarLossy.Render(annotation)
	}

	if e := m.queryBar.ParseErr(); e != "" {
		body += "  " + m.styles.QueryBarLossy.Render("("+e+")")
	}

	return m.styles.QueryBar.Width(m.width).Render(body)
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
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
	sep := m.colSeparator()
	var b strings.Builder
	b.Grow(m.width * 3)
	first := true
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

		if !first {
			b.WriteString(sep)
		}
		first = false

		// Apply header style — use pre-computed stain styles when possible
		if idx < len(m.colStyles) {
			var ss *stain.Style
			if m.focusedCell.Row == -1 && m.focusedCell.Col == idx && m.focused {
				ss = m.colStyles[idx].focused
			} else {
				ss = m.colStyles[idx].header
			}
			contentWidth := m.colStyles[idx].contentWidth
			header = data.TruncateOrPad(header, contentWidth)
			ss.RenderTo(&m.cellBlock, header)
			b.Write(m.cellBlock.Bytes())
		} else {
			style := m.styles.HeaderCell
			if m.focusedCell.Row == -1 && m.focusedCell.Col == idx && m.focused {
				style = m.styles.CellFocused
			}
			style = style.Width(w).MaxWidth(w)
			contentWidth := w - style.GetHorizontalFrameSize()
			if contentWidth < 1 {
				contentWidth = 1
			}
			header = data.TruncateOrPad(header, contentWidth)
			b.WriteString(style.Render(header))
		}
	}
	return b.String()
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
	rowHeight := rn.RowHeight
	if rowHeight < 1 {
		rowHeight = 1
	}

	sep := m.colSeparator()

	// Fast path: when all cells use pre-computed stain styles, write directly
	// into a strings.Builder via RenderTo with a reusable Block, avoiding
	// per-cell String() copies and the []string intermediate slice.
	useAllPrecomputed := m.styles.StyleFunc == nil && m.editState == nil
	if useAllPrecomputed {
		for _, idx := range colIndices {
			col := m.cols[idx]
			if col.CellStyle != nil || col.ColumnSpan != nil || idx >= len(m.colStyles) {
				useAllPrecomputed = false
				break
			}
		}
	}
	if useAllPrecomputed && rowHeight == m.defaultRowHeight {
		var b strings.Builder
		b.Grow(m.width * 3) // pre-size for ANSI sequences + content
		first := true
		for _, idx := range colIndices {
			col := m.cols[idx]

			// Get display text — use typed Text path when available
			var formatted string
			var val any
			if col.Text != nil {
				formatted = col.Text(&rn.Data)
			} else {
				if col.Value != nil {
					val = col.Value(rn.Data)
				}
				formatted = conv.SprintValue(val)
				if col.ValueFormatter != nil {
					formatted = col.ValueFormatter(val, rn.Data)
				}
			}

			// Select stain style
			isSelected := !col.NoSelect && m.sel.ContainsCell(displayIndex, idx)
			isFocused := m.focused && m.focusedCell.Row == displayIndex && m.focusedCell.Col == idx

			cs := &m.colStyles[idx]
			var ss *stain.Style
			if isFocused {
				ss = cs.focused
			} else if isSelected {
				ss = cs.selected
			} else if isPinned {
				ss = cs.pinned
			} else if displayIndex >= 0 && displayIndex%2 != 0 {
				ss = cs.oddRow
			} else if displayIndex >= 0 {
				ss = cs.evenRow
			} else {
				ss = cs.cell
			}

			// Render content
			contentWidth := cs.contentWidth
			cellContent := formatted
			var renderer data.CellRenderer[T]
			if col.CellRendererSelector != nil {
				renderer = col.CellRendererSelector(rn.Data)
			}
			if renderer == nil {
				renderer = col.CellRenderer
			}
			if renderer != nil {
				ctx := data.CellContext[T]{
					Value: val, FormattedValue: formatted,
					Data: rn.Data, RowNode: rn, Column: &col,
					ColumnIndex: idx, RowIndex: displayIndex,
					IsSelected: isSelected, IsFocused: isFocused,
					Width: contentWidth, Height: rowHeight,
				}
				cellContent = renderer.Render(ctx)
			} else {
				cellContent = ansi.Truncate(cellContent, contentWidth, "…")
			}

			// Write separator + rendered cell bytes directly into builder
			if !first {
				b.WriteString(sep)
			}
			first = false
			ss.RenderTo(&m.cellBlock, cellContent)
			rendered := m.cellBlock.Bytes()
			// If any cell produces multi-line output, fall through to the
			// slow path which handles cross-cell line alignment.
			for _, c := range rendered {
				if c == '\n' {
					goto slowPath
				}
			}
			b.Write(rendered)
		}
		return b.String()
	}

slowPath:

	// Slow path: column spanning, editing, custom StyleFunc/CellStyle, or
	// non-default row height. Collect []string and join via joinCellLines.
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
			for s := 1; s < span; s++ {
				nextIdx := idx + s
				if nextIdx < len(m.cols) {
					w += m.colWidths[nextIdx]
					if m.styles.BorderColumn {
						w++
					}
				}
			}
			skipUntil = idx + span
		}

		var val any
		if col.Value != nil {
			val = col.Value(rn.Data)
		}
		formatted := conv.SprintValue(val)
		if col.ValueFormatter != nil {
			formatted = col.ValueFormatter(val, rn.Data)
		}

		if m.editState != nil && m.editState.position.Row == displayIndex && m.editState.position.Col == idx {
			editorView := m.editState.editor.View()
			cells = append(cells, m.styles.EditorInput.Width(w).MaxWidth(w).Height(rowHeight).Render(editorView))
			continue
		}

		isSelected := !col.NoSelect && m.sel.ContainsCell(displayIndex, idx)
		isFocused := m.focused && m.focusedCell.Row == displayIndex && m.focusedCell.Col == idx

		style := m.styles.Cell
		if isPinned {
			style = m.styles.CellPinned
		} else if displayIndex >= 0 && displayIndex%2 != 0 {
			style = m.styles.CellOddRow
		} else if displayIndex >= 0 {
			style = m.styles.CellEvenRow
		}
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
		style = style.Width(w).MaxWidth(w).Height(rowHeight)

		contentWidth := w - style.GetHorizontalFrameSize()
		if contentWidth < 1 {
			contentWidth = 1
		}
		cellContent := formatted
		ctx := data.CellContext[T]{
			Value: val, FormattedValue: formatted,
			Data: rn.Data, RowNode: rn, Column: &col,
			ColumnIndex: idx, RowIndex: displayIndex,
			IsSelected: isSelected, IsFocused: isFocused,
			Width: contentWidth, Height: rowHeight,
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
		cells = append(cells, style.Render(cellContent))
	}

	return m.joinCellLines(cells, colIndices)
}

// joinCellLines joins rendered cells with column separators, handling
// multi-line cells by aligning each line across all columns. For multi-line
// cells, each line gets its own separator and shorter cells are padded to
// match the tallest cell.
func (m Model[T]) joinCellLines(cells []string, colIndices []int) string {
	sep := m.colSeparator()

	// Fast path: if no cell contains a newline, join directly.
	multiLine := false
	for _, c := range cells {
		if strings.Contains(c, "\n") {
			multiLine = true
			break
		}
	}
	if !multiLine {
		return strings.Join(cells, sep)
	}

	// Split each cell into lines and find the max line count.
	cellLines := make([][]string, len(cells))
	maxLines := 0
	for i, c := range cells {
		cellLines[i] = strings.Split(c, "\n")
		if len(cellLines[i]) > maxLines {
			maxLines = len(cellLines[i])
		}
	}

	// Join corresponding lines across all cells.
	result := make([]string, maxLines)
	for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
		parts := make([]string, len(cells))
		for cellIdx := range cells {
			if lineIdx < len(cellLines[cellIdx]) {
				parts[cellIdx] = cellLines[cellIdx][lineIdx]
			} else {
				// Pad with spaces to the column width.
				w := m.colWidths[colIndices[cellIdx]]
				parts[cellIdx] = strings.Repeat(" ", w)
			}
		}
		result[lineIdx] = strings.Join(parts, sep)
	}
	return strings.Join(result, "\n")
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
			var aggResult any
			if rn.AggValues != nil {
				aggResult = rn.AggValues[col.ColumnID]
			}
			if aggResult == nil {
				// Fallback: compute on the fly if not cached
				values := collectLeafValues(rn, &col)
				if col.AggFuncCustom != nil {
					aggResult = col.AggFuncCustom(values)
				} else {
					aggResult = grouping.Aggregate(values, col.AggFunc)
				}
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

		sized := style.Width(w).MaxWidth(w)
		cells = append(cells, sized.Render(cellContent))
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
		} else if col.Value != nil {
			values = append(values, col.Value(child.Data))
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
