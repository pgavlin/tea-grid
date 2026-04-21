// Package grid provides the core tea-grid component for Bubble Tea.
package grid

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pgavlin/stain"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grouping"
	"github.com/pgavlin/tea-grid/internal/conv"
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
	rows      []*data.RowNode[T] // All row nodes
	pinnedTop []*data.RowNode[T] // Rows pinned to top
	pinnedBot []*data.RowNode[T] // Rows pinned to bottom

	// Row ID function
	rowIDFunc func(T) string
	nextRowID int

	// Computed display rows (cached)
	displayRows []*data.RowNode[T]
	dirty       bool // true if display rows need recomputing

	// Column widths (computed)
	colWidths []int

	// True if any column has AutoFit = true. Used to gate data-triggered
	// layout recomputes on SetRows / InsertRow / RemoveRow / UpdateRow.
	hasAutoFit bool

	// Sticky width overrides set by AutoSizeColumn(s). Keyed by ColumnID.
	manualWidths map[string]int

	// Viewport and scrolling
	vp viewport

	// Selection
	sel selection.Model

	// Sorting
	sortModel gridsort.Model[T]
	postSort  func([]*data.RowNode[T]) []*data.RowNode[T]

	// Filtering
	quickFilterEnabled       bool
	quickFilterText          string
	quickFilterActive        bool
	quickFilterWords         []string      // cached split of quickFilterText, updated on text change
	quickFilterSeq           uint64        // bumped on each keystroke, used to discard stale debounce ticks
	quickFilterDebounceDelay time.Duration // delay before recomputing after keystroke (default 100ms, 0 = immediate)
	filterEditColIdx    int             // -1 = no filter editor active
	externalFilter      func(T) bool
	// Cached list of column indices with active filters (rebuilt at start of each recompute)
	activeFilters []int
	// Filter result cache
	filterDirty           bool
	cachedFiltered        []*data.RowNode[T]
	cachedFilterEditColIdx int // filterEditColIdx when cachedFiltered was built

	// Grouping
	groupModel grouping.Model[T]

	// Editing
	editable  bool
	editState *editState[T]

	// Pending row data (set by WithRows, applied in New after all options)
	pendingRows []T

	// Pending column filters (set by WithColumnFilter, applied in New after all options)
	pendingColumnFilters map[string]filter.Filter

	// Pending column pins (set by WithColumnPin, applied in New after all options)
	pendingColumnPins map[string]data.Pin

	// Pinning functions
	pinnedTopFunc        func(T) bool
	pinnedBotFunc        func(T) bool
	staticPinnedTop      []T
	staticPinnedBot      []T
	staticPinnedTopNodes []*data.RowNode[T]
	staticPinnedBotNodes []*data.RowNode[T]

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
	styles    Styles
	colStyles []colCellStyles // pre-computed per-column styles with Width/MaxWidth/Height applied
	cellBlock stain.Block     // reusable block for cell rendering (avoids buffer growth allocs)

	// Cached visible column indices, rebuilt on column/pin changes.
	cachedLeft, cachedCenter, cachedRight []int
}

// colCellStyles holds pre-computed lipgloss styles for a single column,
// with Width, MaxWidth, and Height already applied. This avoids copying
// the 648-byte Style struct 3 times per cell per frame.
type colCellStyles struct {
	cell     *stain.Style
	evenRow  *stain.Style
	oddRow   *stain.Style
	focused  *stain.Style
	selected *stain.Style
	pinned   *stain.Style
	header   *stain.Style
	// contentWidth is the usable cell width (column width minus horizontal
	// frame size). Pre-computed to avoid calling GetHorizontalFrameSize()
	// per cell, which copies the 648-byte Style struct.
	contentWidth int
}

// New creates a new grid model with the given options.
func New[T any](opts ...Option[T]) Model[T] {
	m := Model[T]{
		KeyMap:                   DefaultKeyMap(),
		Help:                     help.New(),
		styles:                   DefaultStyles(),
		vp:                       newViewport(),
		sel:                      selection.New(selection.SelectNone),
		groupModel:               grouping.Model[T]{Expanded: make(map[string]bool), DefaultExpanded: -1},
		defaultRowHeight:         1,
		dirty:                    true,
		filterDirty:              true,
		focusedCell:              CellPosition{Row: 0, Col: 0},
		filterEditColIdx:         -1,
		cachedFilterEditColIdx:   -1,
		quickFilterDebounceDelay: 100 * time.Millisecond,
	}

	for _, opt := range opts {
		opt(&m)
	}

	// Apply deferred row data (after all options so rowIDFunc etc. are set)
	if m.pendingRows != nil {
		m.setRowData(m.pendingRows)
		m.pendingRows = nil
	}

	// Apply deferred column filters (after all options so columns are set)
	for i := range m.cols {
		if f, ok := m.pendingColumnFilters[m.cols[i].ColumnID]; ok {
			m.cols[i].Filter = f
		}
	}
	m.pendingColumnFilters = nil

	// Apply deferred column pins (after all options so columns are set)
	for i := range m.cols {
		if dir, ok := m.pendingColumnPins[m.cols[i].ColumnID]; ok {
			m.cols[i].Pinned = dir
		}
	}
	m.pendingColumnPins = nil

	// Build static pinned row nodes once (so IDs are stable)
	m.buildStaticPinnedNodes()

	// Build initial display rows and compute layout
	m.recomputeDisplayRows()
	m.refreshHasAutoFit()
	m.computeColWidths()
	m.updateViewportSize()

	return m
}

// Init is a no-op. The model is always in a consistent state because every
// public setter eagerly recomputes display rows after mutating state.
func (m Model[T]) Init() tea.Cmd {
	return nil
}

// --- Data ---

// SetRows replaces the row data.
func (m *Model[T]) SetRows(rows []T) {
	m.setRowData(rows)
	m.pruneSelection()
	m.dirty = true
	m.recomputeDisplayRows()
	if m.hasAutoFit {
		m.computeColWidths()
	}
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
	m.filterDirty = true
	m.refreshHasAutoFit()
	m.recomputeDisplayRows()
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
			m.filterDirty = true
			m.recomputeDisplayRows()
			if m.hasAutoFit {
				m.computeColWidths()
			}
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
	m.filterDirty = true
	m.recomputeDisplayRows()
	if m.hasAutoFit {
		m.computeColWidths()
	}
}

// RemoveRow removes the row with the given ID.
func (m *Model[T]) RemoveRow(id string) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			m.sel.Clear()
			m.dirty = true
			m.filterDirty = true
			m.recomputeDisplayRows()
			if m.hasAutoFit {
				m.computeColWidths()
			}
			return
		}
	}
}

// --- Dimensions ---

// SetWidth sets the grid width in terminal columns and recomputes column widths.
func (m *Model[T]) SetWidth(w int) {
	m.width = w
	m.computeColWidths()
}

// SetHeight sets the grid height in terminal lines and recomputes the viewport.
func (m *Model[T]) SetHeight(h int) {
	m.height = h
	m.updateViewportSize()
}

// Width returns the current grid width.
func (m Model[T]) Width() int { return m.width }

// Height returns the current grid height.
func (m Model[T]) Height() int { return m.height }

// --- Focus ---

// Focus sets the grid as focused, enabling keyboard input.
func (m *Model[T]) Focus() { m.focused = true }

// Blur removes focus from the grid, disabling keyboard input.
func (m *Model[T]) Blur() { m.focused = false }

// Focused returns whether the grid is currently focused.
func (m Model[T]) Focused() bool { return m.focused }

// SetFocusedCell moves the focus to the given cell position.
func (m *Model[T]) SetFocusedCell(pos CellPosition) { m.focusedCell = pos }

// FocusedCell returns the currently focused cell position.
func (m Model[T]) FocusedCell() CellPosition { return m.focusedCell }

// --- Selection ---

// SetRowSelection replaces the selection with a single full-row rect for the given row ID.
func (m *Model[T]) SetRowSelection(id string) {
	for i, rn := range m.displayRows {
		if rn.ID == id {
			m.sel.Replace(selection.Rect{
				Kind:   selection.KindFullRow,
				Anchor: selection.Position{Row: i, Col: 0},
				Cursor: selection.Position{Row: i, Col: 0},
			})
			return
		}
	}
}

// ToggleRowSelection toggles a KindFullRow rect for the given row ID.
// Non-KindFullRow rects are cleared first (same semantics as the Space key).
func (m *Model[T]) ToggleRowSelection(id string) {
	for i, rn := range m.displayRows {
		if rn.ID == id {
			m.sel.ToggleFullRow(i)
			return
		}
	}
}

// SetColumnSelection replaces the selection with a single full-column rect.
func (m *Model[T]) SetColumnSelection(colIdx int) {
	m.sel.Replace(selection.Rect{
		Kind:   selection.KindFullCol,
		Anchor: selection.Position{Row: 0, Col: colIdx},
		Cursor: selection.Position{Row: len(m.displayRows) - 1, Col: colIdx},
	})
}

// SetRectSelection replaces the selection with a single rectangular region.
func (m *Model[T]) SetRectSelection(anchor, cursor CellPosition) {
	m.sel.Replace(selection.Rect{
		Kind:   selection.KindRect,
		Anchor: selection.Position{Row: anchor.Row, Col: anchor.Col},
		Cursor: selection.Position{Row: cursor.Row, Col: cursor.Col},
	})
}

// SelectAllRows replaces the selection with a KindFullRow rect spanning all display rows.
func (m *Model[T]) SelectAllRows() {
	if len(m.displayRows) == 0 {
		return
	}
	m.sel.Replace(selection.Rect{
		Kind:   selection.KindFullRow,
		Anchor: selection.Position{Row: 0, Col: 0},
		Cursor: selection.Position{Row: len(m.displayRows) - 1, Col: 0},
	})
}

// ClearSelection removes all selection.
func (m *Model[T]) ClearSelection() {
	m.sel.Clear()
}

// Selection returns all current selection regions.
func (m Model[T]) Selection() []SelectionRegion {
	if len(m.sel.Rects) == 0 {
		return nil
	}
	regions := make([]SelectionRegion, len(m.sel.Rects))
	for i, r := range m.sel.Rects {
		regions[i] = SelectionRegion{
			Kind:   r.Kind,
			Anchor: CellPosition{Row: r.Anchor.Row, Col: r.Anchor.Col},
			Cursor: CellPosition{Row: r.Cursor.Row, Col: r.Cursor.Col},
		}
	}
	return regions
}

// HasSelection returns true if there is at least one selection region.
func (m Model[T]) HasSelection() bool {
	return m.sel.Active()
}

// IsCellSelected returns true if the cell at (row, col) is within any selection region.
func (m Model[T]) IsCellSelected(row, col int) bool {
	return m.sel.ContainsCell(row, col)
}

// IsRowSelected returns true if the given row ID is covered by any KindFullRow rect.
func (m Model[T]) IsRowSelected(id string) bool {
	ranges := m.sel.FullRowRanges()
	for _, r := range ranges {
		for i := r[0]; i <= r[1]; i++ {
			if i >= 0 && i < len(m.displayRows) && m.displayRows[i].ID == id {
				return true
			}
		}
	}
	return false
}

// IsColumnSelected returns true if the given column index is within a KindFullCol rect.
func (m Model[T]) IsColumnSelected(colIdx int) bool {
	for _, r := range m.sel.Rects {
		if r.Kind == selection.KindFullCol {
			cLo, cHi := r.ColRange()
			if colIdx >= cLo && colIdx <= cHi {
				return true
			}
		}
	}
	return false
}

// isRowInSelection returns true if the display row index is covered by any selection rect.
func (m Model[T]) isRowInSelection(rowIdx int) bool {
	for _, r := range m.sel.Rects {
		switch r.Kind {
		case selection.KindFullRow, selection.KindRect:
			lo, hi := r.RowRange()
			if rowIdx >= lo && rowIdx <= hi {
				return true
			}
		}
	}
	return false
}

// SelectedRows returns the data for all rows covered by the current selection.
// For KindFullRow and KindRect, rows in the row range are included.
// For KindFullCol, no rows are returned (column selection is about columns).
func (m Model[T]) SelectedRows() []T {
	if !m.sel.Active() {
		return nil
	}
	var result []T
	for _, rn := range m.displayRows {
		if rn.IsGroup {
			continue
		}
		if m.isRowInSelection(rn.RowIndex) {
			result = append(result, rn.Data)
		}
	}
	return result
}

// SelectedRowNodes returns the row nodes for all rows covered by the current selection.
// For KindFullRow and KindRect, rows in the row range are included.
// For KindFullCol, no rows are returned.
//
// The returned pointers reference live grid-internal state; fields like RowIndex, Pinned,
// Parent, and AggValues are rewritten on every recompute. Do not retain these pointers
// across Update calls. Use SelectedRows() instead if you only need the row data.
func (m Model[T]) SelectedRowNodes() []*data.RowNode[T] {
	if !m.sel.Active() {
		return nil
	}
	var result []*data.RowNode[T]
	for _, rn := range m.displayRows {
		if rn.IsGroup {
			continue
		}
		if m.isRowInSelection(rn.RowIndex) {
			result = append(result, rn)
		}
	}
	return result
}

// SelectionBounds returns the bounding rectangle of the selection, or (-1,-1,-1,-1) if inactive.
// Row values are display row indices; column values are grid column indices.
func (m Model[T]) SelectionBounds() (rowLo, rowHi, colLo, colHi int) {
	return m.sel.BoundingRect()
}

// --- Sorting ---

// SetSort replaces the sort criteria and marks display rows as dirty.
func (m *Model[T]) SetSort(criteria []gridsort.SortCriterion) {
	m.sortModel.SortOrder = criteria
	m.dirty = true
	m.recomputeDisplayRows()
}

// SortOrder returns the current sort criteria.
func (m Model[T]) SortOrder() []gridsort.SortCriterion {
	return m.sortModel.SortOrder
}

// --- Filtering ---

// SetQuickFilter sets the quick filter text and marks display rows as dirty.
func (m *Model[T]) SetQuickFilter(text string) {
	m.quickFilterText = text
	m.updateQuickFilterWords()
	m.dirty = true
	m.filterDirty = true
	m.recomputeDisplayRows()
}

// updateQuickFilterWords recomputes the cached word list from quickFilterText.
func (m *Model[T]) updateQuickFilterWords() {
	if m.quickFilterText == "" {
		m.quickFilterWords = nil
	} else {
		m.quickFilterWords = strings.Fields(strings.ToLower(m.quickFilterText))
	}
}

// SetColumnFilter sets the filter for the column with the given ID.
func (m *Model[T]) SetColumnFilter(colID string, f filter.Filter) {
	for i := range m.cols {
		if m.cols[i].ColumnID == colID {
			m.cols[i].Filter = f
			m.dirty = true
			m.filterDirty = true
			m.recomputeDisplayRows()
			return
		}
	}
}

// ClearFilters removes all quick and column filters.
func (m *Model[T]) ClearFilters() {
	m.quickFilterText = ""
	m.updateQuickFilterWords()
	for i := range m.cols {
		m.cols[i].Filter = nil
	}
	m.dirty = true
	m.filterDirty = true
	m.recomputeDisplayRows()
}

// --- Grouping ---

// ExpandGroup expands the group with the given key.
func (m *Model[T]) ExpandGroup(groupKey string) {
	m.groupModel.SetExpanded(groupKey, true)
	m.dirty = true
	m.recomputeDisplayRows()
}

// CollapseGroup collapses the group with the given key.
func (m *Model[T]) CollapseGroup(groupKey string) {
	m.groupModel.SetExpanded(groupKey, false)
	m.dirty = true
	m.recomputeDisplayRows()
}

// ExpandAll expands all groups at all levels.
func (m *Model[T]) ExpandAll() {
	groups := grouping.BuildGroups(m.rows, m.cols, m.groupModel.GroupColumns,
		m.groupModel.Expanded, m.groupModel.DefaultExpanded)
	m.groupModel.ExpandAll(groups)
	m.dirty = true
	m.recomputeDisplayRows()
}

// CollapseAll collapses all groups at all levels.
func (m *Model[T]) CollapseAll() {
	groups := grouping.BuildGroups(m.rows, m.cols, m.groupModel.GroupColumns,
		m.groupModel.Expanded, m.groupModel.DefaultExpanded)
	m.groupModel.CollapseAll(groups)
	m.dirty = true
	m.recomputeDisplayRows()
}

// --- Row Access ---

// FocusedRowData returns the data of the currently focused row.
// Returns the zero value and false if the focus is on the header, a group row,
// or if the focus position is out of range.
func (m Model[T]) FocusedRowData() (T, bool) {
	if m.focusedCell.Row < 0 || m.focusedCell.Row >= len(m.displayRows) {
		var zero T
		return zero, false
	}
	rn := m.displayRows[m.focusedCell.Row]
	if rn.IsGroup {
		var zero T
		return zero, false
	}
	return rn.Data, true
}

// Filtering reports whether the grid is currently in a filter editing mode
// (column filter editor or quick filter). When true, the grid is consuming
// keys like Escape internally, and parent models should not handle them.
func (m Model[T]) Filtering() bool {
	return m.filterEditColIdx >= 0 || m.quickFilterActive
}

// ScrollToRowByID scrolls to and focuses the row with the given ID.
// Returns true if the row was found, false otherwise.
func (m *Model[T]) ScrollToRowByID(id string) bool {
	m.recomputeDisplayRows()
	for i, rn := range m.displayRows {
		if rn.ID == id {
			m.focusedCell = CellPosition{Row: i, Col: m.focusedCell.Col}
			m.vp.ensureRowVisible(i, len(m.displayRows), m.rowHeightFunc())
			return true
		}
	}
	return false
}

// --- Scrolling ---

// ScrollToRow scrolls the viewport to ensure the given display row index is visible.
func (m *Model[T]) ScrollToRow(index int) {
	m.vp.ensureRowVisible(index, len(m.displayRows), m.rowHeightFunc())
}

// ScrollToTop scrolls to the first row.
func (m *Model[T]) ScrollToTop() { m.vp.scrollToTop() }

// ScrollToBottom scrolls to the last page.
func (m *Model[T]) ScrollToBottom() { m.vp.scrollToBottom(len(m.displayRows), m.rowHeightFunc()) }

// --- Pinning ---

// PinColumn pins the column with the given ID to the specified direction.
func (m *Model[T]) PinColumn(colID string, dir data.Pin) {
	for i := range m.cols {
		if m.cols[i].ColumnID == colID {
			m.cols[i].Pinned = dir
			m.computeColWidths()
			return
		}
	}
}

// UnpinColumn removes the pin from the column with the given ID.
func (m *Model[T]) UnpinColumn(colID string) {
	m.PinColumn(colID, data.PinNone)
}

// PinRow pins the row with the given ID to the specified position.
func (m *Model[T]) PinRow(id string, pos data.Pin) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].Pinned = pos
			m.dirty = true
			m.filterDirty = true
			m.recomputeDisplayRows()
			return
		}
	}
}

// UnpinRow removes the pin from the row with the given ID.
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

// rowHeightFunc returns a function that maps a display row index to its height in terminal lines.
func (m *Model[T]) rowHeightFunc() func(int) int {
	return func(i int) int {
		if i >= 0 && i < len(m.displayRows) {
			if h := m.displayRows[i].RowHeight; h > 0 {
				return h
			}
		}
		return 1
	}
}

func (m *Model[T]) setRowData(rows []T) {
	oldRows := m.rows
	m.rows = make([]*data.RowNode[T], len(rows))
	for i, d := range rows {
		rn := &data.RowNode[T]{
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
			rn.ID = "row-" + strconv.Itoa(m.nextRowID)
			m.nextRowID++
		}
		m.rows[i] = rn
	}
	m.dirty = true
	m.filterDirty = true
}

func (m *Model[T]) pruneSelection() {
	// Index-based selection is invalidated by data changes (sort/filter/SetRows).
	m.sel.Clear()
}

func (m *Model[T]) makeRowNode(d T) *data.RowNode[T] {
	rn := &data.RowNode[T]{
		Data:      d,
		RowHeight: m.defaultRowHeight,
	}
	if m.dynamicRowHeight != nil {
		rn.RowHeight = m.dynamicRowHeight(d)
	}
	if m.rowIDFunc != nil {
		rn.ID = m.rowIDFunc(d)
	} else {
		rn.ID = "row-" + strconv.Itoa(m.nextRowID)
		m.nextRowID++
	}
	return rn
}

func (m *Model[T]) buildStaticPinnedNodes() {
	m.staticPinnedTopNodes = nil
	for i, d := range m.staticPinnedTop {
		rn := &data.RowNode[T]{
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
		rn := &data.RowNode[T]{
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

	pinnedTopHeight := 0
	for _, rn := range m.pinnedTop {
		h := rn.RowHeight
		if h < 1 {
			h = 1
		}
		pinnedTopHeight += h
	}
	pinnedBotHeight := 0
	for _, rn := range m.pinnedBot {
		h := rn.RowHeight
		if h < 1 {
			h = 1
		}
		pinnedBotHeight += h
	}

	m.vp.visibleLines = m.height - headerHeight - filterHeight - pinnedTopHeight - pinnedBotHeight
	if m.vp.visibleLines < 1 {
		m.vp.visibleLines = 1
	}
}

// recomputeDisplayRows recomputes the sorted, filtered, grouped display row list.
func (m *Model[T]) recomputeDisplayRows() {
	if !m.dirty && m.displayRows != nil {
		return
	}

	m.rebuildActiveFilters()

	var filtered []*data.RowNode[T]

	if !m.filterDirty && m.cachedFiltered != nil && m.filterEditColIdx == m.cachedFilterEditColIdx {
		// Filter state unchanged — reuse cached results.
		// Copy the slice to avoid sortRows mutating the cache in-place.
		filtered = append([]*data.RowNode[T](nil), m.cachedFiltered...)
		// Rebuild pinned rows from source since pin state may have changed.
		m.pinnedTop = nil
		m.pinnedBot = nil
		for _, rn := range m.rows {
			if rn.Pinned == data.PinTop {
				m.pinnedTop = append(m.pinnedTop, rn)
			} else if rn.Pinned == data.PinBottom {
				m.pinnedBot = append(m.pinnedBot, rn)
			}
		}
		m.pinnedTop = append(m.pinnedTop, m.staticPinnedTopNodes...)
		m.pinnedBot = append(m.pinnedBot, m.staticPinnedBotNodes...)
	} else {
		// Full filter pass.
		filtered = make([]*data.RowNode[T], 0, len(m.rows))
		m.pinnedTop = nil
		m.pinnedBot = nil

		for _, rn := range m.rows {
			if rn.Pinned == data.PinTop {
				m.pinnedTop = append(m.pinnedTop, rn)
				continue
			}
			if rn.Pinned == data.PinBottom {
				m.pinnedBot = append(m.pinnedBot, rn)
				continue
			}
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
			if m.externalFilter != nil && !m.externalFilter(rn.Data) {
				continue
			}
			if !m.passesColumnFilters(rn.Data) {
				continue
			}
			if len(m.quickFilterWords) > 0 && !m.passesQuickFilter(rn, m.quickFilterWords) {
				continue
			}
			filtered = append(filtered, rn)
		}

		m.pinnedTop = append(m.pinnedTop, m.staticPinnedTopNodes...)
		m.pinnedBot = append(m.pinnedBot, m.staticPinnedBotNodes...)

		m.cachedFiltered = filtered
		m.cachedFilterEditColIdx = m.filterEditColIdx
		m.filterDirty = false
	}

	// Group / sort / flatten (always runs when dirty)
	if len(m.groupModel.GroupColumns) > 0 {
		groups := grouping.BuildGroups(filtered, m.cols, m.groupModel.GroupColumns,
			m.groupModel.Expanded, m.groupModel.DefaultExpanded)
		m.sortGroups(groups)
		m.displayRows = grouping.FlattenGroups(groups)
	} else {
		m.sortRows(filtered)
		m.displayRows = filtered
	}

	if m.postSort != nil {
		m.displayRows = m.postSort(m.displayRows)
	}

	for i := range m.displayRows {
		m.displayRows[i].RowIndex = i
	}

	m.precomputeAggValues()
	m.dirty = false
	m.updateViewportSize()
}

// precomputeAggValues computes and caches aggregation values on group RowNodes
// so that renderAggCells doesn't need to walk the tree every frame.
func (m *Model[T]) precomputeAggValues() {
	for i := range m.displayRows {
		rn := m.displayRows[i]
		if !rn.IsGroup {
			continue
		}
		rn.AggValues = make(map[string]any, len(m.cols))
		for _, col := range m.cols {
			if col.AggFuncCustom == nil && col.AggFunc == "" {
				continue
			}
			values := collectLeafValues(rn, &col)
			if col.AggFuncCustom != nil {
				rn.AggValues[col.ColumnID] = col.AggFuncCustom(values)
			} else {
				rn.AggValues[col.ColumnID] = grouping.Aggregate(values, col.AggFunc)
			}
		}
	}
}

// rebuildActiveFilters updates the cached list of column indices with active filters.
func (m *Model[T]) rebuildActiveFilters() {
	m.activeFilters = m.activeFilters[:0]
	for i, col := range m.cols {
		if i == m.filterEditColIdx {
			continue
		}
		if col.Filter != nil && col.Filter.Active() {
			m.activeFilters = append(m.activeFilters, i)
		}
	}
}

func (m *Model[T]) passesColumnFilters(data T) bool {
	for _, i := range m.activeFilters {
		col := m.cols[i]
		val := col.Value(data)
		if !col.Filter.Matches(val) {
			return false
		}
	}
	return true
}

func (m *Model[T]) passesQuickFilter(rn *data.RowNode[T], words []string) bool {
	for _, word := range words {
		found := false
		for _, col := range m.cols {
			if col.QuickFilterMatch != nil {
				if col.QuickFilterMatch(&rn.Data, word) {
					found = true
					break
				}
				continue
			}
			var text string
			if col.Text != nil {
				text = col.Text(&rn.Data)
			} else if col.Value != nil {
				text = conv.SprintValue(col.Value(rn.Data))
			} else {
				continue
			}
			if containsFold(text, word) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// containsFold reports whether s contains substr under Unicode case-folding.
func containsFold(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if strings.EqualFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func (m *Model[T]) sortRows(rows []*data.RowNode[T]) {
	if len(m.sortModel.SortOrder) == 0 {
		return
	}

	sort.SliceStable(rows, func(i, j int) bool {
		for _, sc := range m.sortModel.SortOrder {
			col := m.findCol(sc.ColumnID)
			if col == nil {
				continue
			}

			var cmp int
			if col.Compare != nil {
				cmp = col.Compare(&rows[i].Data, &rows[j].Data)
			} else if col.Value != nil {
				a := col.Value(rows[i].Data)
				b := col.Value(rows[j].Data)
				if col.Comparator != nil {
					cmp = col.Comparator(a, b)
				} else {
					cmp = defaultCompare(a, b)
				}
			} else {
				continue
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
			if col != nil && col.Value != nil {
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
			// Sort leaf rows directly — no copy needed with pointer slices
			m.sortRows(g.Children)
		}
	}
}

// firstLeafValue walks down to the first leaf row and returns the column value.
func (m *Model[T]) firstLeafValue(node *data.RowNode[T], col *data.Column[T]) any {
	if !node.IsGroup {
		return col.Value(node.Data)
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

// refreshHasAutoFit recomputes the hasAutoFit cache from the current
// column set.
func (m *Model[T]) refreshHasAutoFit() {
	m.hasAutoFit = false
	for i := range m.cols {
		if m.cols[i].AutoFit {
			m.hasAutoFit = true
			return
		}
	}
}

// measureColumnWidth returns the content-fit width for col measured against
// the given rows. It takes the max of the header width and each row's
// rendered display width (via NaturalWidthRenderer or the Text/Value chain),
// then clamps to [MinWidth (default 4), MaxWidth (0 = unbounded)].
// Cells with ColumnSpan > 1 are skipped — they lie about per-column width
// by design.
func (m *Model[T]) measureColumnWidth(col data.Column[T], colIdx int, rows []*data.RowNode[T]) int {
	minW := col.MinWidth
	if minW == 0 {
		minW = 4
	}

	w := lipgloss.Width(col.HeaderName)

	var natural data.NaturalWidthRenderer[T]
	if col.CellRenderer != nil {
		if nw, ok := col.CellRenderer.(data.NaturalWidthRenderer[T]); ok {
			natural = nw
		}
	}

	for i, rn := range rows {
		if col.ColumnSpan != nil && col.ColumnSpan(rn.Data) > 1 {
			continue
		}
		var cellW int
		if natural != nil {
			ctx := m.buildMeasureContext(col, colIdx, i, rn)
			cellW = natural.NaturalWidth(ctx)
		} else {
			cellW = lipgloss.Width(measureCellText(col, rn.Data))
		}
		if cellW > w {
			w = cellW
		}
	}

	if w < minW {
		w = minW
	}
	if col.MaxWidth > 0 && w > col.MaxWidth {
		w = col.MaxWidth
	}
	return w
}

// measureCellText produces the string a column would render for row via the
// Text / ValueFormatter / Value fallback chain (no CellRenderer invocation).
func measureCellText[T any](col data.Column[T], row T) string {
	if col.Text != nil {
		return col.Text(&row)
	}
	var v any
	if col.Value != nil {
		v = col.Value(row)
	}
	if col.ValueFormatter != nil {
		return col.ValueFormatter(v, row)
	}
	return conv.SprintValue(v)
}

// buildMeasureContext constructs a CellContext suitable for a NaturalWidth
// query. Width = MaxWidth if set, else 0 (signals "no target width").
func (m *Model[T]) buildMeasureContext(col data.Column[T], colIdx, rowIdx int, rn *data.RowNode[T]) data.CellContext[T] {
	var val any
	var formatted string
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
	w := col.MaxWidth
	return data.CellContext[T]{
		Value:          val,
		FormattedValue: formatted,
		Data:           rn.Data,
		RowNode:        rn,
		Column:         &col,
		ColumnIndex:    colIdx,
		RowIndex:       rowIdx,
		Width:          w,
		Height:         1,
	}
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
		idx  int
		flex int
		min  int
		max  int
	}

	var flexCols []colInfo
	for _, idx := range visibleCols {
		col := m.cols[idx]
		if w, ok := m.manualWidths[col.ColumnID]; ok {
			m.colWidths[idx] = w
			remaining -= w
			continue
		}
		switch {
		case col.Width > 0:
			m.colWidths[idx] = col.Width
			remaining -= col.Width
		case col.AutoFit:
			w := m.measureColumnWidth(col, idx, m.rows)
			m.colWidths[idx] = w
			remaining -= w
		default:
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

	// Rebuild cached column indices, styles, and visible count
	m.rebuildVisibleColIndices()
	m.updateVisibleColCount()
	m.rebuildColStyles()
}

// rebuildColStyles pre-computes lipgloss styles for each column with Width,
// MaxWidth, and Height already applied. This avoids copying the 648-byte Style
// struct multiple times per cell per frame in the render loop.
func (m *Model[T]) rebuildColStyles() {
	m.colStyles = make([]colCellStyles, len(m.cols))
	h := m.defaultRowHeight
	if h < 1 {
		h = 1
	}
	profile := stain.ANSI256
	for i, w := range m.colWidths {
		apply := func(s lipgloss.Style) *stain.Style {
			return stain.Compile(s.Width(w).MaxWidth(w).Height(h), profile)
		}
		cell := m.styles.Cell.Width(w).MaxWidth(w).Height(h)
		contentWidth := w - cell.GetHorizontalFrameSize()
		if contentWidth < 1 {
			contentWidth = 1
		}
		m.colStyles[i] = colCellStyles{
			cell:         stain.Compile(cell, profile),
			evenRow:      apply(m.styles.CellEvenRow),
			oddRow:       apply(m.styles.CellOddRow),
			focused:      apply(m.styles.CellFocused),
			selected:     apply(m.styles.CellSelected),
			pinned:       apply(m.styles.CellPinned),
			header:       stain.Compile(m.styles.HeaderCell.Width(w).MaxWidth(w), profile),
			contentWidth: contentWidth,
		}
	}
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

// visibleColIndices returns cached column indices partitioned by pin direction.
func (m *Model[T]) visibleColIndices() (left, center, right []int) {
	return m.cachedLeft, m.cachedCenter, m.cachedRight
}

// rebuildVisibleColIndices recomputes the cached left/center/right column
// index slices. Called when columns or pin state changes.
func (m *Model[T]) rebuildVisibleColIndices() {
	m.cachedLeft = m.cachedLeft[:0]
	m.cachedCenter = m.cachedCenter[:0]
	m.cachedRight = m.cachedRight[:0]
	for i, col := range m.cols {
		if col.Hide {
			continue
		}
		switch col.Pinned {
		case data.PinLeft:
			m.cachedLeft = append(m.cachedLeft, i)
		case data.PinRight:
			m.cachedRight = append(m.cachedRight, i)
		default:
			m.cachedCenter = append(m.cachedCenter, i)
		}
	}
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
	case time.Time:
		if bv, ok := b.(time.Time); ok {
			if av.Before(bv) {
				return -1
			}
			if av.After(bv) {
				return 1
			}
			return 0
		}
	}

	// Fallback: compare string representations
	as := fmt.Sprintf("%v", a)
	bs := fmt.Sprintf("%v", b)
	return strings.Compare(as, bs)
}
