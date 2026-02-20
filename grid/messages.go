package grid

import (
	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/row"
	"github.com/pgavlin/tea-grid/sort"
)

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

// RowSelectedMsg is emitted when a row is selected or deselected.
type RowSelectedMsg[T any] struct {
	Row      row.RowNode[T]
	Selected bool
}

// SelectionChangedMsg is emitted when the selection set changes.
type SelectionChangedMsg[T any] struct {
	Selected []row.RowNode[T]
}

// SortChangedMsg is emitted when the sort order changes.
type SortChangedMsg struct {
	SortOrder []sort.SortCriterion
}

// FilterChangedMsg is emitted when a column filter changes.
type FilterChangedMsg struct {
	ColID  string
	Active bool
}

// QuickFilterChangedMsg is emitted when the quick filter text changes.
type QuickFilterChangedMsg struct {
	Text string
}

// CellEditingStartedMsg is emitted when cell editing begins.
type CellEditingStartedMsg struct {
	Position CellPosition
}

// CellEditingStoppedMsg is emitted when cell editing ends (confirm or cancel).
type CellEditingStoppedMsg struct {
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

// RowsSetMsg is emitted when the row data is replaced.
type RowsSetMsg[T any] struct {
	Rows []T
}

// ColumnsSetMsg is emitted when the column definitions are replaced.
type ColumnsSetMsg[T any] struct {
	Cols []column.ColDef[T]
}
