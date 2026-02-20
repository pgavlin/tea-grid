// Package cell provides cell rendering and editing for the tea-grid component.
package cell

import (
	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/row"
)

// CellContext provides all information a renderer needs.
type CellContext[T any] struct {
	Value          any              // The raw cell value.
	FormattedValue string           // After ValueFormatter.
	Data           T                // The full row data.
	RowNode        *row.RowNode[T]
	ColDef         *column.ColDef[T]
	ColIndex       int
	RowIndex       int
	IsSelected     bool
	IsFocused      bool
	Width          int // Available width in terminal columns.
	Height         int // Available height in terminal lines.
}

// CellRenderer renders a cell's content.
type CellRenderer[T any] interface {
	Render(ctx CellContext[T]) string
}

// CellRendererFunc is a convenience adapter.
type CellRendererFunc[T any] func(ctx CellContext[T]) string

func (f CellRendererFunc[T]) Render(ctx CellContext[T]) string {
	return f(ctx)
}
