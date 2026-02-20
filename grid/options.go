package grid

import (
	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/row"
	"github.com/pgavlin/tea-grid/selection"
	"github.com/pgavlin/tea-grid/sort"
)

// Option is a functional option for configuring a grid.
type Option[T any] func(*Model[T])

// WithColumns sets the column definitions.
func WithColumns[T any](cols []column.ColDef[T]) Option[T] {
	return func(m *Model[T]) {
		m.cols = cols
	}
}

// WithColumnGroups sets the column groups for grouped headers.
func WithColumnGroups[T any](groups []column.ColGroup[T]) Option[T] {
	return func(m *Model[T]) {
		m.colGroups = groups
	}
}

// WithRows sets the row data.
func WithRows[T any](rows []T) Option[T] {
	return func(m *Model[T]) {
		m.setRowData(rows)
	}
}

// WithRowID sets the function used to derive unique row IDs.
func WithRowID[T any](fn func(T) string) Option[T] {
	return func(m *Model[T]) {
		m.rowIDFunc = fn
	}
}

// WithWidth sets the grid width in terminal columns.
func WithWidth[T any](w int) Option[T] {
	return func(m *Model[T]) {
		m.width = w
	}
}

// WithHeight sets the grid height in terminal lines.
func WithHeight[T any](h int) Option[T] {
	return func(m *Model[T]) {
		m.height = h
	}
}

// WithStyles sets the grid styles.
func WithStyles[T any](s Styles) Option[T] {
	return func(m *Model[T]) {
		m.styles = s
	}
}

// WithKeyMap sets the key bindings.
func WithKeyMap[T any](km KeyMap) Option[T] {
	return func(m *Model[T]) {
		m.KeyMap = km
	}
}

// WithFocused sets whether the grid starts focused.
func WithFocused[T any](f bool) Option[T] {
	return func(m *Model[T]) {
		m.focused = f
	}
}

// WithSelection sets the selection mode.
func WithSelection[T any](mode selection.Mode) Option[T] {
	return func(m *Model[T]) {
		m.sel = selection.New(mode)
	}
}

// WithSelectionColumn enables/disables the checkbox column for multi-select.
func WithSelectionColumn[T any](enabled bool) Option[T] {
	return func(m *Model[T]) {
		m.showSelectionCol = enabled
	}
}

// WithEditable enables/disables cell editing globally.
func WithEditable[T any](enabled bool) Option[T] {
	return func(m *Model[T]) {
		m.editable = enabled
	}
}

// WithQuickFilter enables/disables the quick filter bar.
func WithQuickFilter[T any](enabled bool) Option[T] {
	return func(m *Model[T]) {
		m.quickFilterEnabled = enabled
	}
}

// WithExternalFilter sets an external filter predicate.
func WithExternalFilter[T any](fn func(T) bool) Option[T] {
	return func(m *Model[T]) {
		m.externalFilter = fn
	}
}

// WithGrouping enables grouping by the specified column IDs.
func WithGrouping[T any](cols ...string) Option[T] {
	return func(m *Model[T]) {
		m.groupModel.GroupColumns = cols
	}
}

// WithGroupDefaultExpanded sets the number of levels expanded by default (-1 = all).
func WithGroupDefaultExpanded[T any](levels int) Option[T] {
	return func(m *Model[T]) {
		m.groupModel.DefaultExpanded = levels
	}
}

// WithDefaultSort sets the initial sort criteria.
func WithDefaultSort[T any](criteria []sort.SortCriterion) Option[T] {
	return func(m *Model[T]) {
		m.sortModel.SortOrder = criteria
	}
}

// WithMultiSort enables/disables multi-column sorting.
func WithMultiSort[T any](enabled bool) Option[T] {
	return func(m *Model[T]) {
		m.sortModel.MultiSort = enabled
	}
}

// WithPostSort sets a post-sort transformation function.
func WithPostSort[T any](fn func([]row.RowNode[T]) []row.RowNode[T]) Option[T] {
	return func(m *Model[T]) {
		m.postSort = fn
	}
}

// WithPinnedTopRows sets a predicate for dynamically pinning rows to the top.
func WithPinnedTopRows[T any](fn func(T) bool) Option[T] {
	return func(m *Model[T]) {
		m.pinnedTopFunc = fn
	}
}

// WithPinnedBottomRows sets a predicate for dynamically pinning rows to the bottom.
func WithPinnedBottomRows[T any](fn func(T) bool) Option[T] {
	return func(m *Model[T]) {
		m.pinnedBotFunc = fn
	}
}

// WithStaticPinnedTop sets static rows pinned to the top.
func WithStaticPinnedTop[T any](rows []T) Option[T] {
	return func(m *Model[T]) {
		m.staticPinnedTop = rows
	}
}

// WithStaticPinnedBottom sets static rows pinned to the bottom.
func WithStaticPinnedBottom[T any](rows []T) Option[T] {
	return func(m *Model[T]) {
		m.staticPinnedBot = rows
	}
}

// WithRowHeight sets the default row height.
func WithRowHeight[T any](height int) Option[T] {
	return func(m *Model[T]) {
		m.defaultRowHeight = height
	}
}

// WithDynamicRowHeight sets a function to compute row height per row.
func WithDynamicRowHeight[T any](fn func(T) int) Option[T] {
	return func(m *Model[T]) {
		m.dynamicRowHeight = fn
	}
}

// WithOnSelectionChanged sets a callback for selection changes.
func WithOnSelectionChanged[T any](fn func([]T)) Option[T] {
	return func(m *Model[T]) {
		m.onSelectionChanged = fn
	}
}

// WithOnCellValueChanged sets a callback for cell value changes.
func WithOnCellValueChanged[T any](fn func(CellValueChangedMsg[T])) Option[T] {
	return func(m *Model[T]) {
		m.onCellValueChanged = fn
	}
}

// WithOnSortChanged sets a callback for sort changes.
func WithOnSortChanged[T any](fn func([]sort.SortCriterion)) Option[T] {
	return func(m *Model[T]) {
		m.onSortChanged = fn
	}
}
