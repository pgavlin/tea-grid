package data

import (
	tea "charm.land/bubbletea/v2"
)

// CellContext provides all information a renderer needs.
type CellContext[T any] struct {
	Value          any    // The raw cell value.
	FormattedValue string // After ValueFormatter.
	Data           T      // The full row data.
	RowNode        *RowNode[T]
	Column         *Column[T]
	ColumnIndex    int
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

// NaturalWidthRenderer reports a renderer's preferred display width,
// independent of ctx.Width. Renderers that want their output to drive
// AutoFit should implement this. Renderers that produce width-independent
// output (e.g. plain text) can skip it; the column's AutoFit path falls
// back to Column.Text / ValueFormatter / Value for measurement.
type NaturalWidthRenderer[T any] interface {
	CellRenderer[T]
	NaturalWidth(ctx CellContext[T]) int
}

// CellEditor handles inline editing of a cell value.
type CellEditor[T any] interface {
	// Init is called when editing begins. Returns initial command.
	Init(ctx CellContext[T]) tea.Cmd

	// Update handles messages while editing.
	Update(msg tea.Msg) (CellEditor[T], tea.Cmd)

	// View renders the editor UI.
	View() string

	// Value returns the current edited value.
	Value() any

	// Validate returns an error string if the value is invalid, or "".
	Validate() string
}
