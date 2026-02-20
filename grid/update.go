package grid

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/cell"
	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/filter"
)

// Update handles messages. Implements tea.Model.
func (m Model[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.computeColWidths()
		m.updateViewportSize()
		return m, nil

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

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles key messages in normal (non-editing) mode.
func (m Model[T]) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visibleCols := m.visibleCols()
	totalRows := len(m.displayRows)

	switch {
	// Quit
	case key.Matches(msg, m.KeyMap.Quit):
		return m, tea.Quit

	// Help toggle
	case key.Matches(msg, m.KeyMap.Help):
		m.Help.ShowAll = !m.Help.ShowAll
		return m, nil

	// Navigation
	case key.Matches(msg, m.KeyMap.Up):
		return m.moveFocus(m.focusedCell.Row-1, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.Down):
		return m.moveFocus(m.focusedCell.Row+1, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.Left):
		if m.focusedCell.Row >= 0 && m.focusedCell.Row < totalRows {
			rn := m.displayRows[m.focusedCell.Row]
			if rn.IsGroup && rn.Expanded {
				return m.collapseCurrentGroup()
			}
		}
		return m.moveFocus(m.focusedCell.Row, m.focusedCell.Col-1)

	case key.Matches(msg, m.KeyMap.Right):
		if m.focusedCell.Row >= 0 && m.focusedCell.Row < totalRows {
			rn := m.displayRows[m.focusedCell.Row]
			if rn.IsGroup && !rn.Expanded {
				return m.expandCurrentGroup()
			}
		}
		return m.moveFocus(m.focusedCell.Row, m.focusedCell.Col+1)

	case key.Matches(msg, m.KeyMap.PageUp):
		return m.moveFocus(m.focusedCell.Row-m.vp.visibleRows, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.PageDown):
		return m.moveFocus(m.focusedCell.Row+m.vp.visibleRows, m.focusedCell.Col)

	case key.Matches(msg, m.KeyMap.HalfPageUp):
		return m.moveFocus(m.focusedCell.Row-m.vp.visibleRows/2, m.focusedCell.Col)

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

	// Selection
	case key.Matches(msg, m.KeyMap.Select):
		if m.focusedCell.Row >= 0 && m.focusedCell.Row < totalRows {
			rn := m.displayRows[m.focusedCell.Row]
			m.sel.Toggle(rn.ID)
			return m, nil
		}

	case key.Matches(msg, m.KeyMap.SelectAll):
		m.SelectAll()
		return m, nil

	// Sorting (header mode)
	case key.Matches(msg, m.KeyMap.ToggleSort):
		if m.focusedCell.Row == -1 {
			// Header mode: toggle sort
			if m.focusedCell.Col >= 0 && m.focusedCell.Col < len(m.cols) {
				col := m.cols[m.focusedCell.Col]
				if col.Sortable {
					m.sortModel.ToggleSort(col.ColID)
					m.dirty = true
					m.recomputeDisplayRows()
					if m.onSortChanged != nil {
						m.onSortChanged(m.sortModel.SortOrder)
					}
				}
			}
			return m, nil
		}
		// Data row: start editing or expand group
		if m.focusedCell.Row >= 0 && m.focusedCell.Row < totalRows {
			rn := m.displayRows[m.focusedCell.Row]
			if rn.IsGroup {
				return m.toggleCurrentGroup()
			}
			return m.startEditing()
		}

	case key.Matches(msg, m.KeyMap.ToggleMultiSort):
		if m.focusedCell.Row == -1 && m.sortModel.MultiSort {
			if m.focusedCell.Col >= 0 && m.focusedCell.Col < len(m.cols) {
				col := m.cols[m.focusedCell.Col]
				if col.Sortable {
					m.sortModel.AddSort(col.ColID)
					m.dirty = true
					m.recomputeDisplayRows()
					if m.onSortChanged != nil {
						m.onSortChanged(m.sortModel.SortOrder)
					}
				}
			}
			return m, nil
		}

	// Quick filter
	case key.Matches(msg, m.KeyMap.QuickFilter):
		if m.quickFilterEnabled {
			m.quickFilterActive = !m.quickFilterActive
			if !m.quickFilterActive {
				m.quickFilterText = ""
				m.dirty = true
				m.recomputeDisplayRows()
			}
			m.updateViewportSize()
			return m, nil
		}

	// Column filter
	case key.Matches(msg, m.KeyMap.ColumnFilter):
		return m.startFilterEdit()

	// Cancel/Escape
	case key.Matches(msg, m.KeyMap.CancelEdit):
		// Deselect all on escape
		m.sel.DeselectAll()
		return m, nil

	// Expand/collapse all
	case key.Matches(msg, m.KeyMap.ExpandAll):
		if len(m.groupModel.GroupColumns) > 0 {
			m.ExpandAll()
			return m, nil
		}

	case key.Matches(msg, m.KeyMap.CollapseAll):
		if len(m.groupModel.GroupColumns) > 0 {
			m.CollapseAll()
			return m, nil
		}
	}

	return m, nil
}

// handleEditKeyMsg handles key messages while editing a cell.
func (m Model[T]) handleEditKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

		if m.onCellValueChanged != nil {
			m.onCellValueChanged(CellValueChangedMsg[T]{
				Position: pos,
				OldValue: oldValue,
				NewValue: newValue,
				Data:     m.displayRows[pos.Row].Data,
			})
		}

		return m, nil

	case key.Matches(msg, m.KeyMap.CancelEdit):
		m.editState = nil
		return m, nil

	default:
		// Route to editor
		editor, cmd := m.editState.editor.Update(msg)
		m.editState.editor = editor
		return m, cmd
	}
}

// handleQuickFilterKeyMsg handles key messages while the quick filter is active.
func (m Model[T]) handleQuickFilterKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.quickFilterActive = false
		m.quickFilterText = ""
		m.dirty = true
		m.recomputeDisplayRows()
		m.updateViewportSize()
		return m, nil

	case tea.KeyBackspace:
		if len(m.quickFilterText) > 0 {
			m.quickFilterText = m.quickFilterText[:len(m.quickFilterText)-1]
			m.dirty = true
			m.recomputeDisplayRows()
		}
		return m, nil

	case tea.KeyRunes:
		m.quickFilterText += string(msg.Runes)
		m.dirty = true
		m.recomputeDisplayRows()
		return m, nil

	case tea.KeyEnter:
		// Confirm filter and return to normal mode
		m.quickFilterActive = false
		m.updateViewportSize()
		return m, nil
	}

	return m, nil
}

// handleFilterEditKeyMsg handles key messages while the column filter editor is active.
func (m Model[T]) handleFilterEditKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		m.recomputeDisplayRows()
		m.updateViewportSize()
		return m, func() tea.Msg {
			return FilterChangedMsg{ColID: col.ColID, Active: col.Filter.Active()}
		}

	case key.Matches(msg, m.KeyMap.CancelEdit):
		// Clear the filter and close editor
		switch f := col.Filter.(type) {
		case *filter.TextFilter:
			f.SetText("")
		case *filter.NumberFilter:
			f.SetText("")
		case *filter.TimeFilter:
			f.SetText("")
		case *filter.SetFilter:
			f.IncludeAll()
		case *filter.BoolFilter:
			for f.Active() {
				f.Toggle()
			}
		}
		col.Filter.Update(filter.FilterBlurMsg{})
		m.cols[colIdx] = col
		m.filterEditColIdx = -1
		m.dirty = true
		m.recomputeDisplayRows()
		m.updateViewportSize()
		return m, func() tea.Msg {
			return FilterChangedMsg{ColID: col.ColID, Active: false}
		}

	default:
		// Route to filter
		newFilter, cmd := col.Filter.Update(msg)
		m.cols[colIdx].Filter = newFilter
		return m, cmd
	}
}

// startFilterEdit begins editing the column filter for the focused column.
func (m Model[T]) startFilterEdit() (tea.Model, tea.Cmd) {
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

	// Compute column width
	colWidth := m.width
	if colIdx < len(m.colWidths) {
		colWidth = m.colWidths[colIdx]
	}

	// Send focus message to the filter
	newFilter, cmd := col.Filter.Update(filter.FilterFocusMsg{
		Width:    colWidth,
		MaxLines: maxLines,
	})
	m.cols[colIdx].Filter = newFilter
	m.filterEditColIdx = colIdx
	m.updateViewportSize()

	return m, cmd
}

// moveFocus moves the focus to a new position, clamping to valid bounds.
func (m Model[T]) moveFocus(newRow, newCol int) (tea.Model, tea.Cmd) {
	prev := m.focusedCell

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
func (m Model[T]) startEditing() (tea.Model, tea.Cmd) {
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

	ctx := cell.CellContext[T]{
		Value:          val,
		FormattedValue: formatted,
		Data:           rn.Data,
		RowNode:        &rn,
		ColDef:         &col,
		ColIndex:       pos.Col,
		RowIndex:       pos.Row,
		Width:          m.colWidths[pos.Col],
	}

	// Use column's editor or default text editor
	var editor cell.CellEditor[T]
	if col.CellEditor != nil {
		if ce, ok := col.CellEditor.(cell.CellEditor[T]); ok {
			editor = ce
		}
	}
	if editor == nil {
		editor = cell.NewTextEditor[T]()
	}
	cmd := editor.Init(ctx)

	m.editState = &editState[T]{
		position: pos,
		editor:   editor,
		oldValue: val,
	}

	_ = cmd

	return m, func() tea.Msg {
		return CellEditingStartedMsg{Position: pos}
	}
}

// expandCurrentGroup expands the currently focused group.
func (m Model[T]) expandCurrentGroup() (tea.Model, tea.Cmd) {
	if m.focusedCell.Row < 0 || m.focusedCell.Row >= len(m.displayRows) {
		return m, nil
	}
	rn := m.displayRows[m.focusedCell.Row]
	if !rn.IsGroup {
		return m, nil
	}

	m.groupModel.SetExpanded(rn.GroupKey, true)
	m.dirty = true
	m.recomputeDisplayRows()

	return m, func() tea.Msg {
		return GroupExpandedMsg{GroupKey: rn.GroupKey, Level: rn.GroupLevel}
	}
}

// collapseCurrentGroup collapses the currently focused group.
func (m Model[T]) collapseCurrentGroup() (tea.Model, tea.Cmd) {
	if m.focusedCell.Row < 0 || m.focusedCell.Row >= len(m.displayRows) {
		return m, nil
	}
	rn := m.displayRows[m.focusedCell.Row]
	if !rn.IsGroup {
		return m, nil
	}

	m.groupModel.SetExpanded(rn.GroupKey, false)
	m.dirty = true
	m.recomputeDisplayRows()

	return m, func() tea.Msg {
		return GroupCollapsedMsg{GroupKey: rn.GroupKey, Level: rn.GroupLevel}
	}
}

// toggleCurrentGroup toggles the expand/collapse state of the focused group.
func (m Model[T]) toggleCurrentGroup() (tea.Model, tea.Cmd) {
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

// ensure imports are used
var _ = column.SortAsc
