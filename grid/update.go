package grid

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
)

// Update handles messages and returns the updated model.
func (m Model[T]) Update(msg tea.Msg) (Model[T], tea.Cmd) {
	m.recomputeDisplayRows()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		// If editing, route to editor
		if m.editState != nil {
			return m.handleEditKeyMsg(msg)
		}

		// If filter editor is active, route there
		if m.filterEditColIdx >= 0 {
			return m.handleFilterEditKeyMsg(msg)
		}

		// If quick filter is active, route there
		if m.quickFilterActive {
			return m.handleQuickFilterKeyMsg(msg)
		}

		return m.handleKeyMsg(msg)
	}

	return m, nil
}

// handleKeyMsg handles key messages in normal (non-editing) mode.
func (m Model[T]) handleKeyMsg(msg tea.KeyMsg) (Model[T], tea.Cmd) {
	visibleCols := m.visibleCols()
	totalRows := len(m.displayRows)

	switch {
	// Navigation
	case key.Matches(msg, m.KeyMap.Up):
		return m.moveFocus(m.focusedCell.Row-1, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.Down):
		return m.moveFocus(m.focusedCell.Row+1, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.PageUp):
		newRow := m.focusedCell.Row - m.vp.visibleRows
		if newRow < 0 {
			newRow = 0
		}
		return m.moveFocus(newRow, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.PageDown):
		return m.moveFocus(m.focusedCell.Row+m.vp.visibleRows, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.HalfPageUp):
		newRow := m.focusedCell.Row - m.vp.visibleRows/2
		if newRow < 0 {
			newRow = 0
		}
		return m.moveFocus(newRow, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.HalfPageDown):
		return m.moveFocus(m.focusedCell.Row+m.vp.visibleRows/2, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.Home):
		return m.moveFocus(0, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.End):
		return m.moveFocus(totalRows-1, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.LineStart):
		if len(visibleCols) > 0 {
			return m.moveFocus(m.focusedCell.Row, visibleCols[0])
		}

	case key.Matches(msg, m.KeyMap.LineEnd):
		if len(visibleCols) > 0 {
			return m.moveFocus(m.focusedCell.Row, visibleCols[len(visibleCols)-1])
		}

	case key.Matches(msg, m.KeyMap.GoToHeader):
		return m.moveFocus(-1, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.SelectAll):
		m.SelectAll()
		return m, func() tea.Msg {
			return SelectionChangedMsg[T]{Selected: m.selectedRowNodes()}
		}

	case key.Matches(msg, m.KeyMap.DeselectAll):
		m.DeselectAll()
		return m, nil

	// Sort column from any row
	case key.Matches(msg, m.KeyMap.SortColumn):
		if m.focusedCell.Col >= 0 && m.focusedCell.Col < len(m.cols) {
			col := m.cols[m.focusedCell.Col]
			if col.Sortable {
				m.sortModel.ToggleSort(col.ColumnID)
				m.dirty = true
				return m, func() tea.Msg {
					return SortChangedMsg{SortOrder: m.sortModel.SortOrder}
				}
			}
		}
		return m, nil

	case key.Matches(msg, m.KeyMap.MultiSortColumn):
		if m.sortModel.MultiSort && m.focusedCell.Col >= 0 && m.focusedCell.Col < len(m.cols) {
			col := m.cols[m.focusedCell.Col]
			if col.Sortable {
				m.sortModel.AddSort(col.ColumnID)
				m.dirty = true
				return m, func() tea.Msg {
					return SortChangedMsg{SortOrder: m.sortModel.SortOrder}
				}
			}
		}
		return m, nil

	// Toggle group column from any row
	case key.Matches(msg, m.KeyMap.ToggleGroupColumn):
		if m.focusedCell.Col >= 0 && m.focusedCell.Col < len(m.cols) {
			col := m.cols[m.focusedCell.Col]
			m.groupModel.ToggleGroupColumn(col.ColumnID)
			m.dirty = true
			return m, func() tea.Msg {
				return GroupColumnsChangedMsg{GroupColumns: m.groupModel.GroupColumns}
			}
		}
		return m, nil

	// Quick filter
	case key.Matches(msg, m.KeyMap.QuickFilter):
		if m.quickFilterEnabled {
			m.quickFilterActive = !m.quickFilterActive
			if !m.quickFilterActive {
				if m.quickFilterText != "" {
					m.quickFilterText = ""
					m.dirty = true
					m.updateViewportSize()
					return m, func() tea.Msg { return QuickFilterChangedMsg{Text: ""} }
				}
			}
			m.updateViewportSize()
			return m, nil
		}

	// Column filter
	case key.Matches(msg, m.KeyMap.ColumnFilter):
		return m.startFilterEdit()

	// Shift+nav selection expansion
	case key.Matches(msg, m.KeyMap.ShiftUp):
		return m.shiftMoveFocus(m.focusedCell.Row-1, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.ShiftDown):
		return m.shiftMoveFocus(m.focusedCell.Row+1, m.focusedCell.Col)

	// Expand/collapse all overlap with shift+left/right.
	// When groups exist, grouping takes priority; otherwise shift+nav handles it.
	case key.Matches(msg, m.KeyMap.CollapseAll, m.KeyMap.ShiftLeft):
		if key.Matches(msg, m.KeyMap.CollapseAll) && len(m.groupModel.GroupColumns) > 0 {
			m.CollapseAll()
			return m, nil
		}
		return m.shiftMoveFocus(m.focusedCell.Row, m.focusedCell.Col-1)

	case key.Matches(msg, m.KeyMap.ExpandAll, m.KeyMap.ShiftRight):
		if key.Matches(msg, m.KeyMap.ExpandAll) && len(m.groupModel.GroupColumns) > 0 {
			m.ExpandAll()
			return m, nil
		}
		return m.shiftMoveFocus(m.focusedCell.Row, m.focusedCell.Col+1)
	}

	// Early out if the focus is beyond the total rows for any reason.
	if m.focusedCell.Row >= totalRows {
		return m, nil
	}

	// Header row keybindings
	if isHeader := m.focusedCell.Row == -1; isHeader {
		switch {
		case key.Matches(msg, m.KeyMap.Left):
			return m.moveFocus(m.focusedCell.Row, m.focusedCell.Col-1)

		case key.Matches(msg, m.KeyMap.Right):
			return m.moveFocus(m.focusedCell.Row, m.focusedCell.Col+1)

		case key.Matches(msg, m.KeyMap.ToggleSort):
			if m.focusedCell.Col >= 0 && m.focusedCell.Col < len(m.cols) {
				col := m.cols[m.focusedCell.Col]
				if col.Sortable {
					m.sortModel.ToggleSort(col.ColumnID)
					m.dirty = true
					return m, func() tea.Msg {
						return SortChangedMsg{SortOrder: m.sortModel.SortOrder}
					}
				}
			}
			return m, nil

		case key.Matches(msg, m.KeyMap.ToggleMultiSort):
			if m.sortModel.MultiSort && m.focusedCell.Col >= 0 && m.focusedCell.Col < len(m.cols) {
				col := m.cols[m.focusedCell.Col]
				if col.Sortable {
					m.sortModel.AddSort(col.ColumnID)
					m.dirty = true
					return m, func() tea.Msg {
						return SortChangedMsg{SortOrder: m.sortModel.SortOrder}
					}
				}
			}
			return m, nil

		case key.Matches(msg, m.KeyMap.ShiftLeft):
			return m.shiftMoveFocus(m.focusedCell.Row, m.focusedCell.Col-1)

		case key.Matches(msg, m.KeyMap.ShiftRight):
			return m.shiftMoveFocus(m.focusedCell.Row, m.focusedCell.Col+1)
		}

		return m, nil
	}

	// Group row keybindings
	rn := m.displayRows[m.focusedCell.Row]
	if rn.IsGroup {
		switch {
		// Group toggle
		case key.Matches(msg, m.KeyMap.ToggleGroup):
			return m.toggleCurrentGroup()

		// Group expansion
		case key.Matches(msg, m.KeyMap.ExpandGroup):
			if !rn.Expanded {
				return m.expandCurrentGroup()
			}

		// Group collapse
		case key.Matches(msg, m.KeyMap.CollapseGroup):
			if rn.Expanded {
				return m.collapseCurrentGroup()
			}
		}
		return m, nil
	}

	// Data row keybindings
	switch {
	case key.Matches(msg, m.KeyMap.Left):
		return m.moveFocus(m.focusedCell.Row, m.focusedCell.Col-1)

	case key.Matches(msg, m.KeyMap.Right):
		return m.moveFocus(m.focusedCell.Row, m.focusedCell.Col+1)

	// Selection
	case key.Matches(msg, m.KeyMap.SelectRow):
		m.sel.DeselectAll()
		m.sel.Select(rn.ID)
		m.sel.SetAnchor(m.focusedCell.Row)
		m.colSelectAnchor = -1
		m.colSelectCursor = -1
		m.rectAnchor = CellPosition{Row: -1, Col: -1}
		return m, func() tea.Msg {
			return SelectionChangedMsg[T]{Selected: m.selectedRowNodes()}
		}

	case key.Matches(msg, m.KeyMap.SelectColumn):
		colIdx := m.focusedCell.Col
		if colIdx >= 0 && colIdx < len(m.cols) && !m.cols[colIdx].NoSelect {
			m.sel.DeselectAll()
			m.sel.SetAnchor(-1)
			m.colSelectAnchor = colIdx
			m.colSelectCursor = colIdx
			m.rectAnchor = CellPosition{Row: -1, Col: -1}
			return m, func() tea.Msg {
				return SelectionChangedMsg[T]{Selected: nil}
			}
		}
		return m, nil

	case key.Matches(msg, m.KeyMap.Select):
		m.sel.Toggle(rn.ID)
		m.sel.SetAnchor(-1)
		m.colSelectAnchor = -1
		m.colSelectCursor = -1
		m.rectAnchor = CellPosition{Row: -1, Col: -1}
		return m, func() tea.Msg {
			return SelectionChangedMsg[T]{Selected: m.selectedRowNodes()}
		}

	// Editing
	case key.Matches(msg, m.KeyMap.StartEdit):
		return m.startEditing()
	}

	return m, nil
}

// handleEditKeyMsg handles key messages while editing a cell.
func (m Model[T]) handleEditKeyMsg(msg tea.KeyMsg) (Model[T], tea.Cmd) {
	switch {
	case key.Matches(msg, m.KeyMap.ConfirmEdit):
		// Validate and confirm
		errMsg := m.editState.editor.Validate()
		if errMsg != "" {
			// Stay in edit mode with error
			return m, nil
		}

		pos := m.editState.position
		newValue := m.editState.editor.Value()
		oldValue := m.editState.oldValue

		// Apply value
		col := m.cols[pos.Col]
		if col.ValueSetter != nil {
			rn := &m.displayRows[pos.Row]
			col.ValueSetter(&rn.Data, newValue)
			// Also update in the source rows
			for i := range m.rows {
				if m.rows[i].ID == rn.ID {
					col.ValueSetter(&m.rows[i].Data, newValue)
					break
				}
			}
		}

		m.editState = nil
		data := m.displayRows[pos.Row].Data

		return m, tea.Batch(
			func() tea.Msg {
				return CellEditingConfirmedMsg{Position: pos}
			},
			func() tea.Msg {
				return CellValueChangedMsg[T]{
					Position: pos,
					OldValue: oldValue,
					NewValue: newValue,
					Data:     data,
				}
			},
		)

	case key.Matches(msg, m.KeyMap.CancelEdit):
		pos := m.editState.position
		m.editState = nil
		return m, func() tea.Msg {
			return CellEditingCancelledMsg{Position: pos}
		}

	default:
		// Route to editor
		editor, cmd := m.editState.editor.Update(msg)
		m.editState.editor = editor
		return m, cmd
	}
}

// handleQuickFilterKeyMsg handles key messages while the quick filter is active.
func (m Model[T]) handleQuickFilterKeyMsg(msg tea.KeyMsg) (Model[T], tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.quickFilterActive = false
		if m.quickFilterText != "" {
			m.quickFilterText = ""
			m.dirty = true
			m.updateViewportSize()
			return m, func() tea.Msg { return QuickFilterChangedMsg{Text: ""} }
		}
		m.updateViewportSize()
		return m, nil

	case tea.KeyBackspace:
		if len(m.quickFilterText) > 0 {
			m.quickFilterText = m.quickFilterText[:len(m.quickFilterText)-1]
			m.dirty = true
			return m, func() tea.Msg { return QuickFilterChangedMsg{Text: m.quickFilterText} }
		}
		return m, nil

	case tea.KeyRunes:
		m.quickFilterText += string(msg.Runes)
		m.dirty = true
		return m, func() tea.Msg { return QuickFilterChangedMsg{Text: m.quickFilterText} }

	case tea.KeyEnter:
		// Confirm filter and return to normal mode
		m.quickFilterActive = false
		m.updateViewportSize()
		return m, nil
	}

	return m, nil
}

// handleFilterEditKeyMsg handles key messages while the column filter editor is active.
func (m Model[T]) handleFilterEditKeyMsg(msg tea.KeyMsg) (Model[T], tea.Cmd) {
	colIdx := m.filterEditColIdx
	if colIdx < 0 || colIdx >= len(m.cols) {
		m.filterEditColIdx = -1
		return m, nil
	}
	col := m.cols[colIdx]

	switch {
	case key.Matches(msg, m.KeyMap.ConfirmEdit):
		// Apply the filter and close editor
		col.Filter.Update(filter.FilterBlurMsg{})
		m.cols[colIdx] = col
		m.filterEditColIdx = -1
		m.dirty = true
		m.updateViewportSize()
		return m, func() tea.Msg {
			return FilterChangedMsg{ColumnID: col.ColumnID, Active: col.Filter.Active()}
		}

	case key.Matches(msg, m.KeyMap.CancelEdit):
		// Clear the filter and close editor
		col.Filter.Clear()
		col.Filter.Update(filter.FilterBlurMsg{})
		m.cols[colIdx] = col
		m.filterEditColIdx = -1
		m.dirty = true
		m.updateViewportSize()
		return m, func() tea.Msg {
			return FilterChangedMsg{ColumnID: col.ColumnID, Active: false}
		}

	default:
		// Route to filter
		newFilter, cmd := col.Filter.Update(msg)
		m.cols[colIdx].Filter = newFilter
		return m, cmd
	}
}

// startFilterEdit begins editing the column filter for the focused data.
func (m Model[T]) startFilterEdit() (Model[T], tea.Cmd) {
	colIdx := m.focusedCell.Col
	if colIdx < 0 || colIdx >= len(m.cols) {
		return m, nil
	}

	col := m.cols[colIdx]
	if !col.Filterable || col.Filter == nil {
		return m, nil
	}

	// Compute available maxLines
	headerHeight := 1
	if len(m.colGroups) > 0 {
		headerHeight = 2
	}
	if m.styles.BorderHeader {
		headerHeight++
	}
	maxLines := m.height - headerHeight - 3
	if maxLines > 10 {
		maxLines = 10
	}
	if maxLines < 2 {
		maxLines = 2
	}

	// Send focus message to the filter
	newFilter, cmd := col.Filter.Update(filter.FilterFocusMsg{
		Width:    m.width,
		MaxLines: maxLines,
	})
	m.cols[colIdx].Filter = newFilter
	m.filterEditColIdx = colIdx
	m.updateViewportSize()

	return m, cmd
}

// moveFocus moves the focus to a new position, clamping to valid bounds.
// It also resets both row and column selection anchors.
func (m Model[T]) moveFocus(newRow, newCol int) (Model[T], tea.Cmd) {
	prev := m.focusedCell

	// Clear selection anchors on plain navigation
	m.sel.SetAnchor(-1)
	m.colSelectAnchor = -1
	m.colSelectCursor = -1
	m.rectAnchor = CellPosition{Row: -1, Col: -1}

	// Clamp row: -1 = header, 0..len-1 = data rows
	totalRows := len(m.displayRows)
	if newRow < -1 {
		newRow = -1
	}
	if newRow >= totalRows {
		newRow = totalRows - 1
	}
	if newRow < -1 {
		newRow = 0
	}

	// Clamp column
	visibleCols := m.visibleCols()
	if len(visibleCols) == 0 {
		return m, nil
	}

	// Find the current visible col position
	colPos := -1
	for i, idx := range visibleCols {
		if idx == newCol {
			colPos = i
			break
		}
	}
	if colPos == -1 {
		// Find nearest
		if newCol < visibleCols[0] {
			newCol = visibleCols[0]
		} else if newCol > visibleCols[len(visibleCols)-1] {
			newCol = visibleCols[len(visibleCols)-1]
		} else {
			// Find closest
			best := visibleCols[0]
			for _, idx := range visibleCols {
				if idx <= newCol {
					best = idx
				}
			}
			newCol = best
		}
	}

	m.focusedCell = CellPosition{Row: newRow, Col: newCol}

	// Scroll viewport to keep focus visible
	if newRow >= 0 {
		m.vp.ensureRowVisible(newRow)
	}

	// Handle horizontal scrolling for center columns
	_, center, _ := m.visibleColIndices()
	for i, idx := range center {
		if idx == newCol {
			m.vp.ensureColVisible(i)
			m.updateVisibleColCount()
			break
		}
	}

	if m.focusedCell != prev {
		return m, func() tea.Msg {
			return FocusChangedMsg{Position: m.focusedCell, Previous: prev}
		}
	}

	return m, nil
}

// startEditing begins editing the currently focused cell.
func (m Model[T]) startEditing() (Model[T], tea.Cmd) {
	if !m.editable {
		return m, nil
	}

	pos := m.focusedCell
	if pos.Row < 0 || pos.Row >= len(m.displayRows) {
		return m, nil
	}
	if pos.Col < 0 || pos.Col >= len(m.cols) {
		return m, nil
	}

	col := m.cols[pos.Col]
	if !col.Editable {
		return m, nil
	}

	rn := m.displayRows[pos.Row]
	if rn.IsGroup {
		return m, nil
	}

	// Get current value
	var val any
	if col.ValueGetter != nil {
		val = col.ValueGetter(rn.Data)
	}

	// Format value for context
	formatted := fmt.Sprintf("%v", val)
	if col.ValueFormatter != nil {
		formatted = col.ValueFormatter(val, rn.Data)
	}

	ctx := data.CellContext[T]{
		Value:          val,
		FormattedValue: formatted,
		Data:           rn.Data,
		RowNode:        &rn,
		Column:         &col,
		ColumnIndex:    pos.Col,
		RowIndex:       pos.Row,
		Width:          m.colWidths[pos.Col],
	}

	// Use column's editor or default text editor
	editor := col.CellEditor
	if editor == nil {
		editor = data.NewTextEditor[T]()
	}
	initCmd := editor.Init(ctx)

	m.editState = &editState[T]{
		position: pos,
		editor:   editor,
		oldValue: val,
	}

	startCmd := func() tea.Msg {
		return CellEditingStartedMsg{Position: pos}
	}

	return m, tea.Batch(initCmd, startCmd)
}

// expandCurrentGroup expands the currently focused group.
func (m Model[T]) expandCurrentGroup() (Model[T], tea.Cmd) {
	if m.focusedCell.Row < 0 || m.focusedCell.Row >= len(m.displayRows) {
		return m, nil
	}
	rn := m.displayRows[m.focusedCell.Row]
	if !rn.IsGroup {
		return m, nil
	}

	m.groupModel.SetExpanded(rn.GroupKey, true)
	m.dirty = true

	return m, func() tea.Msg {
		return GroupExpandedMsg{GroupKey: rn.GroupKey, Level: rn.GroupLevel}
	}
}

// collapseCurrentGroup collapses the currently focused group.
func (m Model[T]) collapseCurrentGroup() (Model[T], tea.Cmd) {
	if m.focusedCell.Row < 0 || m.focusedCell.Row >= len(m.displayRows) {
		return m, nil
	}
	rn := m.displayRows[m.focusedCell.Row]
	if !rn.IsGroup {
		return m, nil
	}

	m.groupModel.SetExpanded(rn.GroupKey, false)
	m.dirty = true

	return m, func() tea.Msg {
		return GroupCollapsedMsg{GroupKey: rn.GroupKey, Level: rn.GroupLevel}
	}
}

// toggleCurrentGroup toggles the expand/collapse state of the focused group.
func (m Model[T]) toggleCurrentGroup() (Model[T], tea.Cmd) {
	if m.focusedCell.Row < 0 || m.focusedCell.Row >= len(m.displayRows) {
		return m, nil
	}
	rn := m.displayRows[m.focusedCell.Row]
	if !rn.IsGroup {
		return m, nil
	}

	if rn.Expanded {
		return m.collapseCurrentGroup()
	}
	return m.expandCurrentGroup()
}

// shiftMoveFocus expands the rectangular selection toward (newRow, newCol).
// The anchor stays fixed at the starting cell; the cursor (focusedCell) moves.
func (m Model[T]) shiftMoveFocus(newRow, newCol int) (Model[T], tea.Cmd) {
	// Initialize rectangle anchor from current focus if not set
	anchor := m.rectAnchor
	if anchor.Row < 0 {
		anchor = m.focusedCell
	}

	// Clear row and column selection (shift+nav is mutually exclusive)
	m.sel.DeselectAll()
	m.sel.SetAnchor(-1)
	m.colSelectAnchor = -1
	m.colSelectCursor = -1

	// Move focus (this clears all anchors including rectAnchor)
	m, cmd := m.moveFocus(newRow, newCol)

	// Restore rectangle anchor
	m.rectAnchor = anchor

	return m, tea.Batch(cmd, func() tea.Msg {
		return SelectionChangedMsg[T]{Selected: nil}
	})
}
