// Package column defines column types for the tea-grid component.
package column

import (
	"reflect"

	"github.com/charmbracelet/lipgloss"

	"github.com/pgavlin/tea-grid/filter"
)

// PinDirection indicates whether a column is pinned to the left or right edge.
type PinDirection int

const (
	PinNone  PinDirection = iota
	PinLeft
	PinRight
)

// SortDirection indicates the sort direction for a column.
type SortDirection int

const (
	SortNone SortDirection = iota
	SortAsc
	SortDesc
)

// ColDef defines a single column in the grid.
type ColDef[T any] struct {
	// Identity
	ColID      string // Unique identifier. Required.
	HeaderName string // Display name in the header row.

	// Data access
	ValueGetter    func(T) any                    // Extracts the cell value from the row data. Required.
	ValueFormatter func(value any, data T) string // Format the value for display.

	// Sizing
	Width    int // Fixed width in terminal columns. 0 = auto.
	MinWidth int // Minimum width (default: 4).
	MaxWidth int // Maximum width. 0 = unconstrained.
	Flex     int // Flex weight for distributing remaining space. 0 = no flex.

	// Sorting
	Sortable   bool                                           // Default: true.
	Comparator func(a, b any, isDesc bool) int                // Custom sort.
	SortIndex  int                                            // Initial sort priority (0 = primary). -1 = not sorted.
	SortDir    SortDirection                                  // Asc, Desc, or None.

	// Filtering
	Filterable bool // Default: true.
	Filter     filter.Filter // Column filter.

	// Pinning
	Pinned     PinDirection // Left, Right, or None.
	LockPinned bool         // Prevent user from changing pin state.

	// Cell rendering
	CellRenderer         any                                   // Custom renderer (implements cell.CellRenderer[T]).
	CellRendererSelector func(T) any                           // Dynamic renderer per row (returns cell.CellRenderer[T]).
	CellStyle            func(value any, data T) lipgloss.Style // Per-cell styling.

	// Cell editing
	Editable    bool                      // Default: false.
	CellEditor  any                       // Custom editor (implements cell.CellEditor[T]).
	ValueSetter func(data *T, value any)  // Write the edited value back.

	// Column spanning
	ColSpan func(data T) int // Number of columns this cell spans. Default: 1.

	// Grouping
	RowGroup      bool               // If true, rows are grouped by this column's values.
	AggFunc       string             // Aggregation function name: "sum", "avg", "count", "min", "max".
	AggFuncCustom func(values []any) any // Custom aggregation.

	// Visibility
	Hide bool // If true, column is not rendered.
}

// ColGroup produces a single level of grouped headers.
type ColGroup[T any] struct {
	HeaderName string
	Children   []ColDef[T] // Leaf columns in this group.
}

// Columns returns a []ColDef[T] derived from T's exported struct fields.
// For each exported field, it produces a ColDef with:
//   - ColID and HeaderName set to the field name
//   - ValueGetter set to a function that retrieves the field's value from T
//
// Panics if T is not a struct type.
func Columns[T any]() []ColDef[T] {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic("column.Columns: T must be a struct type")
	}

	var cols []ColDef[T]
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		idx := i // capture for closure
		cols = append(cols, ColDef[T]{
			ColID:      field.Name,
			HeaderName: field.Name,
			ValueGetter: func(data T) any {
				v := reflect.ValueOf(data)
				if v.Kind() == reflect.Ptr {
					if v.IsNil() {
						return nil
					}
					v = v.Elem()
				}
				return v.Field(idx).Interface()
			},
			Sortable:   true,
			Filterable: true,
			MinWidth:   4,
		})
	}
	return cols
}
