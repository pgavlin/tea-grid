package grid

import (
	"time"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/internal/querybar"
	"github.com/pgavlin/tea-grid/searchquery"
	"github.com/pgavlin/tea-grid/selection"
	"github.com/pgavlin/tea-grid/sort"
)

// Option is a functional option for configuring a grid.
type Option[T any] func(*Model[T])

// WithColumns sets the column definitions.
func WithColumns[T any](cols []data.Column[T]) Option[T] {
	return func(m *Model[T]) {
		m.cols = cols
	}
}

// WithColumnGroups sets the column groups for grouped headers.
func WithColumnGroups[T any](groups []data.ColumnGroup[T]) Option[T] {
	return func(m *Model[T]) {
		m.colGroups = groups
	}
}

// WithRows sets the row data. The data is stored and applied after all options
// have been processed, ensuring that WithRowID and other options are available.
func WithRows[T any](rows []T) Option[T] {
	return func(m *Model[T]) {
		m.pendingRows = rows
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

// WithEditable enables/disables cell editing globally.
func WithEditable[T any](enabled bool) Option[T] {
	return func(m *Model[T]) {
		m.editable = enabled
	}
}

// WithQuickFilterDebounce sets the delay before recomputing after a quick
// filter keystroke. Default is 100ms. Set to 0 to disable debouncing.
//
// Reserved for the live-bare-prefix follow-up; inert in the v1
// Enter-only query bar.
func WithQuickFilterDebounce[T any](d time.Duration) Option[T] {
	return func(m *Model[T]) {
		m.quickFilterDebounceDelay = d
	}
}

// WithColumnFilter sets the filter for the column with the given ID.
// This can be called multiple times for different columns.
// Filters are applied after all options, so ordering with WithColumns doesn't matter.
func WithColumnFilter[T any](colID string, f filter.Filter) Option[T] {
	return func(m *Model[T]) {
		if m.pendingColumnFilters == nil {
			m.pendingColumnFilters = make(map[string]filter.Filter)
		}
		m.pendingColumnFilters[colID] = f
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
func WithPostSort[T any](fn func([]*data.RowNode[T]) []*data.RowNode[T]) Option[T] {
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

// WithColumnPin pins the column with the given ID to the specified direction.
// This can be called multiple times for different columns.
// Pins are applied after all options, so ordering with WithColumns doesn't matter.
func WithColumnPin[T any](colID string, dir data.Pin) Option[T] {
	return func(m *Model[T]) {
		if m.pendingColumnPins == nil {
			m.pendingColumnPins = make(map[string]data.Pin)
		}
		m.pendingColumnPins[colID] = dir
	}
}

// WithQueryBar enables the GitHub-style query bar. The bar is rendered
// above the headers and serves as both editing surface and canonical
// view of current filter state. Field/value clauses route to per-column
// filters via the RoundTrippable interface; bare terms feed the
// existing quick-filter substring matcher.
func WithQueryBar[T any]() Option[T] {
	return func(m *Model[T]) {
		m.queryBar = querybar.New(nil)
		m.queryBar.Enable()
	}
}

// WithQueryBarText sets the initial bar text; parsed and applied on
// grid setup against the (possibly auto-derived) vocabulary.
func WithQueryBarText[T any](text string) Option[T] {
	return func(m *Model[T]) {
		if m.queryBar == nil {
			m.queryBar = querybar.New(nil)
			m.queryBar.Enable()
		}
		m.queryBar.SetText(text)
		m.queryBar.Editor().SetText(text)
	}
}

// WithQueryBarVocabulary supplies an explicit vocabulary that shadows
// the column-derived one. Required for parse-time rewrites (is:open →
// state:open) and for queryable fields that do not correspond to a
// single column.
func WithQueryBarVocabulary[T any](v *searchquery.Vocabulary) Option[T] {
	return func(m *Model[T]) {
		if m.queryBar == nil {
			m.queryBar = querybar.New(nil)
			m.queryBar.Enable()
		}
		m.queryBar.SetVocabulary(v)
	}
}

// WithFocusedCell sets the initial focused cell position.
func WithFocusedCell[T any](pos CellPosition) Option[T] {
	return func(m *Model[T]) {
		m.focusedCell = pos
	}
}
