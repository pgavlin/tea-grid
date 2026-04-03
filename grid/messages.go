package grid

import (
	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/selection"
	"github.com/pgavlin/tea-grid/sort"
)

// SelectionKind is a type alias for selection.Kind so users don't need to import the selection package.
type SelectionKind = selection.Kind

const (
	SelectionNone    = selection.KindNone    // No selection.
	SelectionRect    = selection.KindRect    // Rectangular cell selection.
	SelectionFullRow = selection.KindFullRow // Full-row selection.
	SelectionFullCol = selection.KindFullCol // Full-column selection.
)

// SelectionRegion describes one contiguous selected region.
type SelectionRegion struct {
	Kind   SelectionKind
	Anchor CellPosition
	Cursor CellPosition
}

// RowRange returns the ordered row range of the region.
func (r SelectionRegion) RowRange() (lo, hi int) {
	lo, hi = r.Anchor.Row, r.Cursor.Row
	if lo > hi {
		lo, hi = hi, lo
	}
	return
}

// ColRange returns the ordered column range of the region.
func (r SelectionRegion) ColRange() (lo, hi int) {
	lo, hi = r.Anchor.Col, r.Cursor.Col
	if lo > hi {
		lo, hi = hi, lo
	}
	return
}

// CellPosition identifies a cell by row and column index.
type CellPosition struct {
	Row int
	Col int
}

// FocusChangedMsg is emitted when the focused cell changes.
type FocusChangedMsg struct {
	Position CellPosition
	Previous CellPosition
}

// SelectionChangedMsg is emitted when the selection set changes.
type SelectionChangedMsg[T any] struct {
	Regions  []SelectionRegion   // All current selection regions.
	Selected []*data.RowNode[T]  // Rows covered by any selection (convenience).
}

// SortChangedMsg is emitted when the sort order changes.
type SortChangedMsg struct {
	SortOrder []sort.SortCriterion
}

// FilterChangedMsg is emitted when a column filter changes.
type FilterChangedMsg struct {
	ColumnID string
	Active   bool
}

// QuickFilterChangedMsg is emitted when the quick filter text changes.
type QuickFilterChangedMsg struct {
	Text string
}

// CellEditingStartedMsg is emitted when cell editing begins.
type CellEditingStartedMsg struct {
	Position CellPosition
}

// CellEditingConfirmedMsg is emitted when a cell edit is confirmed.
type CellEditingConfirmedMsg struct {
	Position CellPosition
}

// CellValueChangedMsg is emitted when a cell value is changed via editing.
type CellValueChangedMsg[T any] struct {
	Position CellPosition
	OldValue any
	NewValue any
	Data     T
}

// CellEditingCancelledMsg is emitted when cell editing is cancelled.
type CellEditingCancelledMsg struct {
	Position CellPosition
}

// GroupExpandedMsg is emitted when a group is expanded.
type GroupExpandedMsg struct {
	GroupKey string
	Level    int
}

// GroupCollapsedMsg is emitted when a group is collapsed.
type GroupCollapsedMsg struct {
	GroupKey string
	Level    int
}

// GroupColumnsChangedMsg is emitted when the set of grouped columns changes.
type GroupColumnsChangedMsg struct {
	GroupColumns []string
}
