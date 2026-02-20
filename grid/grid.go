// Package grid provides the core tea-grid component for Bubble Tea.
package grid

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/cell"
	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grouping"
	"github.com/pgavlin/tea-grid/row"
	"github.com/pgavlin/tea-grid/selection"
	gridsort "github.com/pgavlin/tea-grid/sort"
)

// editState holds the state for an active cell edit.
type editState[T any] struct {
	position CellPosition
	editor   cell.CellEditor[T]
	oldValue any
}

// Model is the top-level grid component. T is the type of each row's data.
type Model[T any] struct {
	// Public fields following bubbles convention
	KeyMap KeyMap
	Help   help.Model

	// Column definitions
	cols      []column.ColDef[T]
	colGroups []column.ColGroup[T]

	// Row data
	rows      []row.RowNode[T] // All row nodes
	pinnedTop []row.RowNode[T] // Rows pinned to top
	pinnedBot []row.RowNode[T] // Rows pinned to bottom

	// Row ID function
	rowIDFunc func(T) string
	nextRowID int

	// Computed display rows (cached)
	displayRows []row.RowNode[T]
	dirty       bool // true if display rows need recomputing

	// Column widths (computed)
	colWidths []int

	// Viewport and scrolling
	vp viewport

	// Selection
	sel              selection.Model
	showSelectionCol bool

	// Sorting
	sortModel gridsort.Model[T]
	postSort  func([]row.RowNode[T]) []row.RowNode[T]

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

	// Pinning functions
	pinnedTopFunc  func(T) bool
	pinnedBotFunc  func(T) bool
	staticPinnedTop []T
	staticPinnedBot []T

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
	ready  bool // true after first WindowSizeMsg

	// Callbacks
	onSelectionChanged func([]T)
	onCellValueChanged func(CellValueChangedMsg[T])
	onSortChanged      func([]gridsort.SortCriterion)
}

// New creates a new grid model with the given options.
func New[T any](opts ...Option[T]) Model[T] {
	m := Model[T]{
		KeyMap:           DefaultKeyMap(),
		Help:             help.New(),
		styles:           DefaultStyles(),
		vp:               newViewport(),
		sel:              selection.New(selection.SelectNone),
		groupModel:       grouping.Model[T]{Expanded: make(map[string]bool), DefaultExpanded: -1},
		defaultRowHeight: 1,
		dirty:            true,
		focusedCell:      CellPosition{Row: 0, Col: 0},
		filterEditColIdx: -1,
	}

	for _, opt := range opts {
		opt(&m)
	}

	// Build initial display rows
	m.recomputeDisplayRows()

	return m
}

// Init implements tea.Model.
func (m Model[T]) Init() tea.Cmd {
	return nil
}

// --- Data ---

// SetRows replaces the row data.
func (m *Model[T]) SetRows(rows []T) {
	m.setRowData(rows)
	m.dirty = true
	m.recomputeDisplayRows()
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
func (m *Model[T]) SetColumns(cols []column.ColDef[T]) {
	m.cols = cols
	m.dirty = true
	m.computeColWidths()
	m.recomputeDisplayRows()
}

// Columns returns the column definitions.
func (m Model[T]) Columns() []column.ColDef[T] {
	return m.cols
}

// UpdateRow updates the data for a row with the given ID.
func (m *Model[T]) UpdateRow(id string, data T) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].Data = data
			m.dirty = true
			m.recomputeDisplayRows()
			return
		}
	}
}

// InsertRow inserts a row at the given index.
func (m *Model[T]) InsertRow(index int, data T) {
	rn := m.makeRowNode(data)
	if index >= len(m.rows) {
		m.rows = append(m.rows, rn)
	} else {
		m.rows = append(m.rows[:index+1], m.rows[index:]...)
		m.rows[index] = rn
	}
	m.dirty = true
	m.recomputeDisplayRows()
}

// RemoveRow removes the row with the given ID.
func (m *Model[T]) RemoveRow(id string) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			m.dirty = true
			m.recomputeDisplayRows()
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
	var result []T
	for _, rn := range m.rows {
		for _, id := range ids {
			if rn.ID == id {
				result = append(result, rn.Data)
			}
		}
	}
	return result
}

func (m Model[T]) SelectedRowNodes() []*row.RowNode[T] {
	ids := m.sel.SelectedIDs()
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result []*row.RowNode[T]
	for i := range m.rows {
		if idSet[m.rows[i].ID] {
			result = append(result, &m.rows[i])
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
func (m *Model[T]) DeselectAll()          { m.sel.DeselectAll() }
func (m Model[T]) IsSelected(id string) bool { return m.sel.IsSelected(id) }

// --- Sorting ---

func (m *Model[T]) SetSort(criteria []gridsort.SortCriterion) {
	m.sortModel.SortOrder = criteria
	m.dirty = true
	m.recomputeDisplayRows()
}

func (m Model[T]) SortOrder() []gridsort.SortCriterion {
	return m.sortModel.SortOrder
}

// --- Filtering ---

func (m *Model[T]) SetQuickFilter(text string) {
	m.quickFilterText = text
	m.dirty = true
	m.recomputeDisplayRows()
}

func (m *Model[T]) SetColumnFilter(colID string, f filter.Filter) {
	for i := range m.cols {
		if m.cols[i].ColID == colID {
			m.cols[i].Filter = f
			m.dirty = true
			m.recomputeDisplayRows()
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
	m.recomputeDisplayRows()
}

// --- Grouping ---

func (m *Model[T]) ExpandGroup(groupKey string) {
	m.groupModel.SetExpanded(groupKey, true)
	m.dirty = true
	m.recomputeDisplayRows()
}

func (m *Model[T]) CollapseGroup(groupKey string) {
	m.groupModel.SetExpanded(groupKey, false)
	m.dirty = true
	m.recomputeDisplayRows()
}

func (m *Model[T]) ExpandAll() {
	groups := grouping.BuildGroups(m.rows, m.cols, m.groupModel.GroupColumns,
		m.groupModel.Expanded, m.groupModel.DefaultExpanded)
	m.groupModel.ExpandAll(groups)
	m.dirty = true
	m.recomputeDisplayRows()
}

func (m *Model[T]) CollapseAll() {
	groups := grouping.BuildGroups(m.rows, m.cols, m.groupModel.GroupColumns,
		m.groupModel.Expanded, m.groupModel.DefaultExpanded)
	m.groupModel.CollapseAll(groups)
	m.dirty = true
	m.recomputeDisplayRows()
}

// --- Scrolling ---

func (m *Model[T]) ScrollToRow(index int) { m.vp.ensureRowVisible(index) }
func (m *Model[T]) ScrollToTop()          { m.vp.scrollToTop() }
func (m *Model[T]) ScrollToBottom()       { m.vp.scrollToBottom(len(m.displayRows)) }

// --- Pinning ---

func (m *Model[T]) PinColumn(colID string, dir column.PinDirection) {
	for i := range m.cols {
		if m.cols[i].ColID == colID {
			m.cols[i].Pinned = dir
			m.computeColWidths()
			return
		}
	}
}

func (m *Model[T]) UnpinColumn(colID string) {
	m.PinColumn(colID, column.PinNone)
}

func (m *Model[T]) PinRow(id string, pos row.PinPosition) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].Pinned = pos
			m.dirty = true
			m.recomputeDisplayRows()
			return
		}
	}
}

func (m *Model[T]) UnpinRow(id string) {
	m.PinRow(id, row.PinNone)
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
		m.KeyMap.Select, m.KeyMap.Help, m.KeyMap.Quit,
	}
}

// FullHelp implements help.KeyMap.
func (m Model[T]) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.KeyMap.Up, m.KeyMap.Down, m.KeyMap.Left, m.KeyMap.Right},
		{m.KeyMap.PageUp, m.KeyMap.PageDown, m.KeyMap.Home, m.KeyMap.End},
		{m.KeyMap.Select, m.KeyMap.SelectAll, m.KeyMap.QuickFilter},
		{m.KeyMap.Help, m.KeyMap.Quit},
	}
}

// --- Internal ---

func (m *Model[T]) setRowData(rows []T) {
	m.rows = make([]row.RowNode[T], len(rows))
	for i, data := range rows {
		m.rows[i] = m.makeRowNode(data)
	}
	m.dirty = true
}

func (m *Model[T]) makeRowNode(data T) row.RowNode[T] {
	rn := row.RowNode[T]{
		Data:      data,
		RowHeight: m.defaultRowHeight,
	}
	if m.dynamicRowHeight != nil {
		rn.RowHeight = m.dynamicRowHeight(data)
	}
	if m.rowIDFunc != nil {
		rn.ID = m.rowIDFunc(data)
	} else {
		rn.ID = fmt.Sprintf("row-%d", m.nextRowID)
		m.nextRowID++
	}
	return rn
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
	filtered := make([]row.RowNode[T], 0, len(m.rows))

	// Separate pinned rows
	m.pinnedTop = nil
	m.pinnedBot = nil

	for i := range m.rows {
		rn := m.rows[i]

		// Check static pinning
		if rn.Pinned == row.PinTop {
			m.pinnedTop = append(m.pinnedTop, rn)
			continue
		}
		if rn.Pinned == row.PinBottom {
			m.pinnedBot = append(m.pinnedBot, rn)
			continue
		}

		// Check dynamic pinning
		if m.pinnedTopFunc != nil && m.pinnedTopFunc(rn.Data) {
			rn.Pinned = row.PinTop
			m.pinnedTop = append(m.pinnedTop, rn)
			continue
		}
		if m.pinnedBotFunc != nil && m.pinnedBotFunc(rn.Data) {
			rn.Pinned = row.PinBottom
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

	// Add static pinned rows
	for _, data := range m.staticPinnedTop {
		rn := m.makeRowNode(data)
		rn.Pinned = row.PinTop
		m.pinnedTop = append(m.pinnedTop, rn)
	}
	for _, data := range m.staticPinnedBot {
		rn := m.makeRowNode(data)
		rn.Pinned = row.PinBottom
		m.pinnedBot = append(m.pinnedBot, rn)
	}

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

func (m *Model[T]) sortRows(rows []row.RowNode[T]) {
	if len(m.sortModel.SortOrder) == 0 {
		return
	}

	sort.SliceStable(rows, func(i, j int) bool {
		for _, sc := range m.sortModel.SortOrder {
			col := m.findCol(sc.ColID)
			if col == nil || col.ValueGetter == nil {
				continue
			}
			a := col.ValueGetter(rows[i].Data)
			b := col.ValueGetter(rows[j].Data)

			var cmp int
			if col.Comparator != nil {
				cmp = col.Comparator(a, b, sc.Direction == column.SortDesc)
			} else {
				cmp = defaultCompare(a, b)
			}

			if sc.Direction == column.SortDesc {
				cmp = -cmp
			}
			if cmp != 0 {
				return cmp < 0
			}
		}
		return false
	})
}

func (m *Model[T]) sortGroups(groups []*row.RowNode[T]) {
	for _, g := range groups {
		if g.IsGroup && g.Children != nil {
			// Sort children
			childRows := make([]row.RowNode[T], len(g.Children))
			for i, c := range g.Children {
				childRows[i] = *c
			}
			m.sortRows(childRows)
			for i := range childRows {
				g.Children[i] = &childRows[i]
			}
			// Recurse into sub-groups
			m.sortGroups(g.Children)
		}
	}
}

func (m *Model[T]) findCol(colID string) *column.ColDef[T] {
	for i := range m.cols {
		if m.cols[i].ColID == colID {
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

	// Distribute remaining space by flex weight
	if remaining > 0 && len(flexCols) > 0 {
		totalFlex := 0
		for _, fc := range flexCols {
			totalFlex += fc.flex
		}

		for i := range flexCols {
			extra := remaining * flexCols[i].flex / totalFlex
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
		case column.PinLeft:
			left = append(left, i)
		case column.PinRight:
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

	// Calculate how many center columns fit
	available := m.width
	// Subtract pinned column widths
	for i, col := range m.cols {
		if !col.Hide && (col.Pinned == column.PinLeft || col.Pinned == column.PinRight) {
			available -= m.colWidths[i]
			if m.styles.BorderColumn {
				available--
			}
		}
	}

	count := 0
	used := 0
	for _, idx := range center {
		w := m.colWidths[idx]
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
