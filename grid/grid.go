// Package grid provides the core tea-grid component for Bubble Tea.
package grid

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grouping"
	"github.com/pgavlin/tea-grid/selection"
	gridsort "github.com/pgavlin/tea-grid/sort"
)

// editState holds the state for an active cell edit.
type editState[T any] struct {
	position CellPosition
	editor   data.CellEditor[T]
	oldValue any
}

// Model is the top-level grid component. T is the type of each row's data.
type Model[T any] struct {
	// Public fields following bubbles convention
	KeyMap KeyMap
	Help   help.Model

	// Column definitions
	cols      []data.Column[T]
	colGroups []data.ColumnGroup[T]

	// Row data
	rows      []data.RowNode[T] // All row nodes
	pinnedTop []data.RowNode[T] // Rows pinned to top
	pinnedBot []data.RowNode[T] // Rows pinned to bottom

	// Row ID function
	rowIDFunc func(T) string
	nextRowID int

	// Computed display rows (cached)
	displayRows []data.RowNode[T]
	dirty       bool // true if display rows need recomputing

	// Column widths (computed)
	colWidths []int

	// Viewport and scrolling
	vp viewport

	// Selection
	sel             selection.Model
	colSelectAnchor int          // -1 = no column selection (C key)
	colSelectCursor int          // -1 = no column selection (C key)
	rectAnchor      CellPosition // Row=-1 means no rectangular selection (shift+nav)

	// Sorting
	sortModel gridsort.Model[T]
	postSort  func([]data.RowNode[T]) []data.RowNode[T]

	// Filtering
	quickFilterEnabled bool
	quickFilterText    string
	quickFilterActive  bool
	filterEditColIdx   int // -1 = no filter editor active
	externalFilter     func(T) bool

	// Grouping
	groupModel grouping.Model[T]

	// Editing
	editable  bool
	editState *editState[T]

	// Pending row data (set by WithRows, applied in New after all options)
	pendingRows []T

	// Pinning functions
	pinnedTopFunc        func(T) bool
	pinnedBotFunc        func(T) bool
	staticPinnedTop      []T
	staticPinnedBot      []T
	staticPinnedTopNodes []data.RowNode[T]
	staticPinnedBotNodes []data.RowNode[T]

	// Row height
	defaultRowHeight int
	dynamicRowHeight func(T) int

	// Dimensions
	width   int
	height  int
	focused bool

	// Focus position: Row == -1 means header
	focusedCell CellPosition

	// Styles
	styles Styles

}

// New creates a new grid model with the given options.
func New[T any](opts ...Option[T]) Model[T] {
	m := Model[T]{
		KeyMap:           DefaultKeyMap(),
		Help:             help.New(),
		styles:           DefaultStyles(),
		vp:               newViewport(),
		sel:              selection.New(selection.SelectNone),
		colSelectAnchor:  -1,
		colSelectCursor:  -1,
		rectAnchor:       CellPosition{Row: -1, Col: -1},
		groupModel:       grouping.Model[T]{Expanded: make(map[string]bool), DefaultExpanded: -1},
		defaultRowHeight: 1,
		dirty:            true,
		focusedCell:      CellPosition{Row: 0, Col: 0},
		filterEditColIdx: -1,
	}

	for _, opt := range opts {
		opt(&m)
	}

	// Apply deferred row data (after all options so rowIDFunc etc. are set)
	if m.pendingRows != nil {
		m.setRowData(m.pendingRows)
		m.pendingRows = nil
	}

	// Build static pinned row nodes once (so IDs are stable)
	m.buildStaticPinnedNodes()

	// Build initial display rows and compute layout
	m.recomputeDisplayRows()
	m.computeColWidths()
	m.updateViewportSize()

	return m
}

// Init returns an initial command. Currently always returns nil.
func (m Model[T]) Init() tea.Cmd {
	return nil
}

// --- Data ---

// SetRows replaces the row data.
func (m *Model[T]) SetRows(rows []T) {
	m.setRowData(rows)
	m.pruneSelection()
	m.dirty = true
}

// Rows returns the raw row data.
func (m Model[T]) Rows() []T {
	result := make([]T, len(m.rows))
	for i, rn := range m.rows {
		result[i] = rn.Data
	}
	return result
}

// SetColumns replaces the column definitions.
func (m *Model[T]) SetColumns(cols []data.Column[T]) {
	m.cols = cols
	m.dirty = true
	m.computeColWidths()
}

// Columns returns the column definitions.
func (m Model[T]) Columns() []data.Column[T] {
	return m.cols
}

// UpdateRow updates the data for a row with the given ID.
func (m *Model[T]) UpdateRow(id string, d T) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].Data = d
			m.dirty = true
			return
		}
	}
}

// InsertRow inserts a row at the given index.
func (m *Model[T]) InsertRow(index int, d T) {
	rn := m.makeRowNode(d)
	if index >= len(m.rows) {
		m.rows = append(m.rows, rn)
	} else {
		m.rows = append(m.rows[:index+1], m.rows[index:]...)
		m.rows[index] = rn
	}
	m.dirty = true
}

// RemoveRow removes the row with the given ID.
func (m *Model[T]) RemoveRow(id string) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			m.sel.Deselect(id)
			m.dirty = true
			return
		}
	}
}

// --- Dimensions ---

func (m *Model[T]) SetWidth(w int) {
	m.width = w
	m.computeColWidths()
}

func (m *Model[T]) SetHeight(h int) {
	m.height = h
	m.updateViewportSize()
}

func (m Model[T]) Width() int  { return m.width }
func (m Model[T]) Height() int { return m.height }

// --- Focus ---

func (m *Model[T]) Focus()                          { m.focused = true }
func (m *Model[T]) Blur()                           { m.focused = false }
func (m Model[T]) Focused() bool                    { return m.focused }
func (m *Model[T]) SetFocusedCell(pos CellPosition) { m.focusedCell = pos }
func (m Model[T]) FocusedCell() CellPosition        { return m.focusedCell }

// --- Selection ---

func (m Model[T]) SelectedRows() []T {
	ids := m.sel.SelectedIDs()
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result []T
	for _, rn := range m.rows {
		if idSet[rn.ID] {
			result = append(result, rn.Data)
		}
	}
	return result
}

func (m Model[T]) SelectedRowNodes() []*data.RowNode[T] {
	ids := m.sel.SelectedIDs()
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result []*data.RowNode[T]
	for i := range m.rows {
		if idSet[m.rows[i].ID] {
			result = append(result, &m.rows[i])
		}
	}
	return result
}

// selectedRowNodes returns row nodes for all selected IDs (used for emitting SelectionChangedMsg).
func (m Model[T]) selectedRowNodes() []data.RowNode[T] {
	ids := m.sel.SelectedIDs()
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result []data.RowNode[T]
	for _, rn := range m.rows {
		if idSet[rn.ID] {
			result = append(result, rn)
		}
	}
	return result
}

func (m *Model[T]) SelectRow(id string)   { m.sel.Select(id) }
func (m *Model[T]) DeselectRow(id string) { m.sel.Deselect(id) }
func (m *Model[T]) SelectAll() {
	ids := make([]string, len(m.displayRows))
	for i, rn := range m.displayRows {
		ids[i] = rn.ID
	}
	m.sel.SelectAll(ids)
}
func (m *Model[T]) DeselectAll() {
	m.sel.DeselectAll()
	m.sel.SetAnchor(-1)
	m.colSelectAnchor = -1
	m.colSelectCursor = -1
	m.rectAnchor = CellPosition{Row: -1, Col: -1}
}
func (m Model[T]) IsSelected(id string) bool { return m.sel.IsSelected(id) }

// SelectColumn selects all cells in the given column, clearing row selection.
func (m *Model[T]) SelectColumn(colIdx int) {
	m.sel.DeselectAll()
	m.colSelectAnchor = colIdx
	m.colSelectCursor = colIdx
}

// DeselectColumns clears column selection.
func (m *Model[T]) DeselectColumns() {
	m.colSelectAnchor = -1
	m.colSelectCursor = -1
}

// IsColumnSelected returns true if the given column index is selected.
func (m Model[T]) IsColumnSelected(colIdx int) bool {
	lo, hi := m.colSelectionRange()
	return lo >= 0 && colIdx >= lo && colIdx <= hi
}

// colSelectionRange returns the ordered range of selected columns, or (-1, -1) if none.
func (m Model[T]) colSelectionRange() (lo, hi int) {
	if m.colSelectAnchor < 0 || m.colSelectCursor < 0 {
		return -1, -1
	}
	lo, hi = m.colSelectAnchor, m.colSelectCursor
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi
}

// SelectionRect returns the rectangular selection bounds, or (-1,-1,-1,-1) if inactive.
// Row values are display row indices; column values are grid column indices.
func (m Model[T]) SelectionRect() (rowLo, rowHi, colLo, colHi int) {
	return m.selectionRect()
}

// selectionRect returns the rectangular selection bounds, or (-1,-1,-1,-1) if inactive.
func (m Model[T]) selectionRect() (rowLo, rowHi, colLo, colHi int) {
	if m.rectAnchor.Row < 0 {
		return -1, -1, -1, -1
	}
	rowLo, rowHi = m.rectAnchor.Row, m.focusedCell.Row
	if rowLo > rowHi {
		rowLo, rowHi = rowHi, rowLo
	}
	colLo, colHi = m.rectAnchor.Col, m.focusedCell.Col
	if colLo > colHi {
		colLo, colHi = colHi, colLo
	}
	return rowLo, rowHi, colLo, colHi
}

// --- Sorting ---

func (m *Model[T]) SetSort(criteria []gridsort.SortCriterion) {
	m.sortModel.SortOrder = criteria
	m.dirty = true
}

func (m Model[T]) SortOrder() []gridsort.SortCriterion {
	return m.sortModel.SortOrder
}

// --- Filtering ---

func (m *Model[T]) SetQuickFilter(text string) {
	m.quickFilterText = text
	m.dirty = true
}

func (m *Model[T]) SetColumnFilter(colID string, f filter.Filter) {
	for i := range m.cols {
		if m.cols[i].ColumnID == colID {
			m.cols[i].Filter = f
			m.dirty = true
			return
		}
	}
}

func (m *Model[T]) ClearFilters() {
	m.quickFilterText = ""
	for i := range m.cols {
		m.cols[i].Filter = nil
	}
	m.dirty = true
}

// --- Grouping ---

func (m *Model[T]) ExpandGroup(groupKey string) {
	m.groupModel.SetExpanded(groupKey, true)
	m.dirty = true
}

func (m *Model[T]) CollapseGroup(groupKey string) {
	m.groupModel.SetExpanded(groupKey, false)
	m.dirty = true
}

func (m *Model[T]) ExpandAll() {
	groups := grouping.BuildGroups(m.rows, m.cols, m.groupModel.GroupColumns,
		m.groupModel.Expanded, m.groupModel.DefaultExpanded)
	m.groupModel.ExpandAll(groups)
	m.dirty = true
}

func (m *Model[T]) CollapseAll() {
	groups := grouping.BuildGroups(m.rows, m.cols, m.groupModel.GroupColumns,
		m.groupModel.Expanded, m.groupModel.DefaultExpanded)
	m.groupModel.CollapseAll(groups)
	m.dirty = true
}

// --- Scrolling ---

func (m *Model[T]) ScrollToRow(index int) { m.vp.ensureRowVisible(index) }
func (m *Model[T]) ScrollToTop()          { m.vp.scrollToTop() }
func (m *Model[T]) ScrollToBottom()       { m.vp.scrollToBottom(len(m.displayRows)) }

// --- Pinning ---

func (m *Model[T]) PinColumn(colID string, dir data.Pin) {
	for i := range m.cols {
		if m.cols[i].ColumnID == colID {
			m.cols[i].Pinned = dir
			m.computeColWidths()
			return
		}
	}
}

func (m *Model[T]) UnpinColumn(colID string) {
	m.PinColumn(colID, data.PinNone)
}

func (m *Model[T]) PinRow(id string, pos data.Pin) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].Pinned = pos
			m.dirty = true
			return
		}
	}
}

func (m *Model[T]) UnpinRow(id string) {
	m.PinRow(id, data.PinNone)
}

// --- Help ---

// HelpView renders the help text.
func (m Model[T]) HelpView() string {
	return m.Help.View(m)
}

// ShortHelp implements help.KeyMap.
func (m Model[T]) ShortHelp() []key.Binding {
	return []key.Binding{
		m.KeyMap.Up, m.KeyMap.Down, m.KeyMap.Left, m.KeyMap.Right,
		m.KeyMap.Select, m.KeyMap.Help,
	}
}

// FullHelp implements help.KeyMap.
func (m Model[T]) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.KeyMap.Up, m.KeyMap.Down, m.KeyMap.Left, m.KeyMap.Right},
		{m.KeyMap.PageUp, m.KeyMap.PageDown, m.KeyMap.Home, m.KeyMap.End},
		{m.KeyMap.Select, m.KeyMap.SelectAll, m.KeyMap.QuickFilter},
		{m.KeyMap.Help},
	}
}

// --- Internal ---

func (m *Model[T]) setRowData(rows []T) {
	oldRows := m.rows
	m.rows = make([]data.RowNode[T], len(rows))
	for i, d := range rows {
		rn := data.RowNode[T]{
			Data:      d,
			RowHeight: m.defaultRowHeight,
		}
		if m.dynamicRowHeight != nil {
			rn.RowHeight = m.dynamicRowHeight(d)
		}
		if m.rowIDFunc != nil {
			rn.ID = m.rowIDFunc(d)
		} else if i < len(oldRows) {
			// Reuse the existing ID for stable identity
			rn.ID = oldRows[i].ID
		} else {
			rn.ID = fmt.Sprintf("row-%d", m.nextRowID)
			m.nextRowID++
		}
		m.rows[i] = rn
	}
	m.dirty = true
}

func (m *Model[T]) pruneSelection() {
	validIDs := make(map[string]bool, len(m.rows))
	for _, rn := range m.rows {
		validIDs[rn.ID] = true
	}
	m.sel.Retain(validIDs)
}

func (m *Model[T]) makeRowNode(d T) data.RowNode[T] {
	rn := data.RowNode[T]{
		Data:      d,
		RowHeight: m.defaultRowHeight,
	}
	if m.dynamicRowHeight != nil {
		rn.RowHeight = m.dynamicRowHeight(d)
	}
	if m.rowIDFunc != nil {
		rn.ID = m.rowIDFunc(d)
	} else {
		rn.ID = fmt.Sprintf("row-%d", m.nextRowID)
		m.nextRowID++
	}
	return rn
}

func (m *Model[T]) buildStaticPinnedNodes() {
	m.staticPinnedTopNodes = nil
	for i, d := range m.staticPinnedTop {
		rn := data.RowNode[T]{
			Data:      d,
			RowHeight: m.defaultRowHeight,
			Pinned:    data.PinTop,
		}
		if m.dynamicRowHeight != nil {
			rn.RowHeight = m.dynamicRowHeight(d)
		}
		if m.rowIDFunc != nil {
			rn.ID = m.rowIDFunc(d)
		} else {
			rn.ID = fmt.Sprintf("pinned-top-%d", i)
		}
		m.staticPinnedTopNodes = append(m.staticPinnedTopNodes, rn)
	}
	m.staticPinnedBotNodes = nil
	for i, d := range m.staticPinnedBot {
		rn := data.RowNode[T]{
			Data:      d,
			RowHeight: m.defaultRowHeight,
			Pinned:    data.PinBottom,
		}
		if m.dynamicRowHeight != nil {
			rn.RowHeight = m.dynamicRowHeight(d)
		}
		if m.rowIDFunc != nil {
			rn.ID = m.rowIDFunc(d)
		} else {
			rn.ID = fmt.Sprintf("pinned-bot-%d", i)
		}
		m.staticPinnedBotNodes = append(m.staticPinnedBotNodes, rn)
	}
}

func (m *Model[T]) updateViewportSize() {
	headerHeight := 1
	if len(m.colGroups) > 0 {
		headerHeight = 2
	}
	if m.styles.BorderHeader {
		headerHeight++
	}

	filterHeight := 0
	if m.quickFilterActive {
		filterHeight = 1
	}
	if m.filterEditColIdx >= 0 && m.filterEditColIdx < len(m.cols) {
		col := m.cols[m.filterEditColIdx]
		if col.Filter != nil {
			view := col.Filter.View()
			filterHeight += 1 // title line
			if view != "" {
				filterHeight += strings.Count(view, "\n") + 1
			}
		}
	}

	pinnedTopHeight := len(m.pinnedTop)
	pinnedBotHeight := len(m.pinnedBot)

	m.vp.visibleRows = m.height - headerHeight - filterHeight - pinnedTopHeight - pinnedBotHeight
	if m.vp.visibleRows < 1 {
		m.vp.visibleRows = 1
	}
}

// recomputeDisplayRows recomputes the sorted, filtered, grouped display row list.
func (m *Model[T]) recomputeDisplayRows() {
	if !m.dirty && m.displayRows != nil {
		return
	}

	// Start with all rows
	filtered := make([]data.RowNode[T], 0, len(m.rows))

	// Separate pinned rows
	m.pinnedTop = nil
	m.pinnedBot = nil

	for i := range m.rows {
		rn := m.rows[i]

		// Check static pinning
		if rn.Pinned == data.PinTop {
			m.pinnedTop = append(m.pinnedTop, rn)
			continue
		}
		if rn.Pinned == data.PinBottom {
			m.pinnedBot = append(m.pinnedBot, rn)
			continue
		}

		// Check dynamic pinning
		if m.pinnedTopFunc != nil && m.pinnedTopFunc(rn.Data) {
			rn.Pinned = data.PinTop
			m.pinnedTop = append(m.pinnedTop, rn)
			continue
		}
		if m.pinnedBotFunc != nil && m.pinnedBotFunc(rn.Data) {
			rn.Pinned = data.PinBottom
			m.pinnedBot = append(m.pinnedBot, rn)
			continue
		}

		// Apply external filter
		if m.externalFilter != nil && !m.externalFilter(rn.Data) {
			continue
		}

		// Apply column filters
		if !m.passesColumnFilters(rn.Data) {
			continue
		}

		// Apply quick filter
		if m.quickFilterText != "" && !m.passesQuickFilter(rn.Data) {
			continue
		}

		filtered = append(filtered, rn)
	}

	// Add static pinned rows (using cached nodes for stable IDs)
	m.pinnedTop = append(m.pinnedTop, m.staticPinnedTopNodes...)
	m.pinnedBot = append(m.pinnedBot, m.staticPinnedBotNodes...)

	// Grouping
	if len(m.groupModel.GroupColumns) > 0 {
		groups := grouping.BuildGroups(filtered, m.cols, m.groupModel.GroupColumns,
			m.groupModel.Expanded, m.groupModel.DefaultExpanded)

		// Sort within groups
		m.sortGroups(groups)

		// Flatten
		m.displayRows = grouping.FlattenGroups(groups)
	} else {
		// Sort
		m.sortRows(filtered)
		m.displayRows = filtered
	}

	// Apply post-sort hook
	if m.postSort != nil {
		m.displayRows = m.postSort(m.displayRows)
	}

	// Update row indices
	for i := range m.displayRows {
		m.displayRows[i].RowIndex = i
	}

	m.dirty = false
	m.updateViewportSize()
}

func (m *Model[T]) passesColumnFilters(data T) bool {
	for i, col := range m.cols {
		if i == m.filterEditColIdx {
			continue // skip the column being edited
		}
		if col.Filter == nil || !col.Filter.Active() {
			continue
		}
		val := col.ValueGetter(data)
		if !col.Filter.Matches(val) {
			return false
		}
	}
	return true
}

func (m *Model[T]) passesQuickFilter(data T) bool {
	words := strings.Fields(strings.ToLower(m.quickFilterText))
	if len(words) == 0 {
		return true
	}

	// Build a string of all column values
	var rowText strings.Builder
	for _, col := range m.cols {
		if col.ValueGetter == nil {
			continue
		}
		val := col.ValueGetter(data)
		fmt.Fprintf(&rowText, " %v", val)
	}
	lower := strings.ToLower(rowText.String())

	for _, word := range words {
		if !strings.Contains(lower, word) {
			return false
		}
	}
	return true
}

func (m *Model[T]) sortRows(rows []data.RowNode[T]) {
	if len(m.sortModel.SortOrder) == 0 {
		return
	}

	sort.SliceStable(rows, func(i, j int) bool {
		for _, sc := range m.sortModel.SortOrder {
			col := m.findCol(sc.ColumnID)
			if col == nil || col.ValueGetter == nil {
				continue
			}
			a := col.ValueGetter(rows[i].Data)
			b := col.ValueGetter(rows[j].Data)

			var cmp int
			if col.Comparator != nil {
				cmp = col.Comparator(a, b)
			} else {
				cmp = defaultCompare(a, b)
			}

			if sc.Direction == data.SortDesc {
				cmp = -cmp
			}
			if cmp != 0 {
				return cmp < 0
			}
		}
		return false
	})
}

func (m *Model[T]) sortGroups(groups []*data.RowNode[T]) {
	m.sortGroupsAtLevel(groups, 0)
}

func (m *Model[T]) sortGroupsAtLevel(groups []*data.RowNode[T], level int) {
	// Sort the group nodes themselves by the grouping column's sort criteria
	if level < len(m.groupModel.GroupColumns) {
		groupColumnID := m.groupModel.GroupColumns[level]
		var sortDir data.SortDirection
		for _, sc := range m.sortModel.SortOrder {
			if sc.ColumnID == groupColumnID {
				sortDir = sc.Direction
				break
			}
		}
		if sortDir != data.SortNone {
			col := m.findCol(groupColumnID)
			if col != nil && col.ValueGetter != nil {
				sort.SliceStable(groups, func(i, j int) bool {
					aVal := m.firstLeafValue(groups[i], col)
					bVal := m.firstLeafValue(groups[j], col)

					var cmp int
					if col.Comparator != nil {
						cmp = col.Comparator(aVal, bVal)
					} else {
						cmp = defaultCompare(aVal, bVal)
					}
					if sortDir == data.SortDesc {
						cmp = -cmp
					}
					return cmp < 0
				})
			}
		}
	}

	// Process children of each group
	for _, g := range groups {
		if !g.IsGroup || g.Children == nil {
			continue
		}

		// Check if children are sub-groups or leaf rows
		if len(g.Children) > 0 && g.Children[0].IsGroup {
			m.sortGroupsAtLevel(g.Children, level+1)
		} else {
			// Sort leaf rows
			childRows := make([]data.RowNode[T], len(g.Children))
			for i, c := range g.Children {
				childRows[i] = *c
			}
			m.sortRows(childRows)
			for i := range childRows {
				g.Children[i] = &childRows[i]
			}
		}
	}
}

// firstLeafValue walks down to the first leaf row and returns the column value.
func (m *Model[T]) firstLeafValue(node *data.RowNode[T], col *data.Column[T]) any {
	if !node.IsGroup {
		return col.ValueGetter(node.Data)
	}
	if len(node.Children) > 0 {
		return m.firstLeafValue(node.Children[0], col)
	}
	return nil
}

func (m *Model[T]) findCol(colID string) *data.Column[T] {
	for i := range m.cols {
		if m.cols[i].ColumnID == colID {
			return &m.cols[i]
		}
	}
	return nil
}

// computeColWidths runs the column sizing algorithm.
func (m *Model[T]) computeColWidths() {
	if len(m.cols) == 0 || m.width == 0 {
		return
	}

	visibleCols := m.visibleCols()
	n := len(visibleCols)
	if n == 0 {
		return
	}

	m.colWidths = make([]int, len(m.cols))
	remaining := m.width

	// Account for column separators
	if m.styles.BorderColumn {
		remaining -= n - 1
	}

	// Track which columns still need sizing
	type colInfo struct {
		idx   int
		flex  int
		min   int
		max   int
		fixed bool
	}

	var flexCols []colInfo
	for _, idx := range visibleCols {
		col := m.cols[idx]
		if col.Width > 0 {
			// Fixed width
			m.colWidths[idx] = col.Width
			remaining -= col.Width
		} else {
			minW := col.MinWidth
			if minW == 0 {
				minW = 4
			}
			flex := col.Flex
			if flex == 0 {
				flex = 1
			}
			flexCols = append(flexCols, colInfo{
				idx:  idx,
				flex: flex,
				min:  minW,
				max:  col.MaxWidth,
			})
		}
	}

	// Allocate minimums
	for _, fc := range flexCols {
		m.colWidths[fc.idx] = fc.min
		remaining -= fc.min
	}

	// Distribute remaining space by flex weight.
	// Use cumulative calculation to avoid losing pixels to integer division.
	if remaining > 0 && len(flexCols) > 0 {
		totalFlex := 0
		for _, fc := range flexCols {
			totalFlex += fc.flex
		}

		distributed := 0
		cumulativeFlex := 0
		for i := range flexCols {
			cumulativeFlex += flexCols[i].flex
			target := remaining * cumulativeFlex / totalFlex
			extra := target - distributed
			distributed = target

			newWidth := m.colWidths[flexCols[i].idx] + extra

			// Clamp to max
			if flexCols[i].max > 0 && newWidth > flexCols[i].max {
				newWidth = flexCols[i].max
			}

			m.colWidths[flexCols[i].idx] = newWidth
		}
	}

	// Update visible columns count
	m.updateVisibleColCount()
}

// visibleCols returns indices of non-hidden columns.
func (m *Model[T]) visibleCols() []int {
	var result []int
	for i, col := range m.cols {
		if !col.Hide {
			result = append(result, i)
		}
	}
	return result
}

// visibleColIndices returns column indices partitioned by pin direction.
func (m *Model[T]) visibleColIndices() (left, center, right []int) {
	for i, col := range m.cols {
		if col.Hide {
			continue
		}
		switch col.Pinned {
		case data.PinLeft:
			left = append(left, i)
		case data.PinRight:
			right = append(right, i)
		default:
			center = append(center, i)
		}
	}
	return
}

func (m *Model[T]) updateVisibleColCount() {
	_, center, _ := m.visibleColIndices()
	if len(center) == 0 {
		m.vp.visibleCols = 0
		return
	}

	// Calculate available space for center columns
	available := m.width
	// Subtract pinned column widths
	for i, col := range m.cols {
		if !col.Hide && (col.Pinned == data.PinLeft || col.Pinned == data.PinRight) {
			available -= m.colWidths[i]
			if m.styles.BorderColumn {
				available--
			}
		}
	}

	// Count from leftCol, not from 0
	start := m.vp.leftCol
	if start >= len(center) {
		start = len(center) - 1
	}
	if start < 0 {
		start = 0
	}

	count := 0
	used := 0
	for i := start; i < len(center); i++ {
		w := m.colWidths[center[i]]
		if m.styles.BorderColumn && count > 0 {
			w++
		}
		if used+w > available {
			break
		}
		used += w
		count++
	}
	m.vp.visibleCols = count
	if m.vp.visibleCols < 1 && len(center) > 0 {
		m.vp.visibleCols = 1
	}
}

// defaultCompare compares two values of common types.
func defaultCompare(a, b any) int {
	// Same-type fast paths
	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			return strings.Compare(av, bv)
		}
	case int:
		if bv, ok := b.(int); ok {
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
			return 0
		}
	case int64:
		if bv, ok := b.(int64); ok {
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
			return 0
		}
	case float64:
		if bv, ok := b.(float64); ok {
			if av < bv {
				return -1
			}
			if av > bv {
				return 1
			}
			return 0
		}
	case bool:
		if bv, ok := b.(bool); ok {
			if av == bv {
				return 0
			}
			if !av {
				return -1
			}
			return 1
		}
	}

	// Fallback: compare string representations
	as := fmt.Sprintf("%v", a)
	bs := fmt.Sprintf("%v", b)
	return strings.Compare(as, bs)
}
