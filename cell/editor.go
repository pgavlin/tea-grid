package cell

import (
	tea "github.com/charmbracelet/bubbletea"
)

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
