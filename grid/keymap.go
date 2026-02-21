package grid

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines the key bindings for the grid.
type KeyMap struct {
	// Cell navigation
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Home         key.Binding
	End          key.Binding
	LineStart    key.Binding
	LineEnd      key.Binding

	// Header navigation
	GoToHeader key.Binding

	// Selection
	Select    key.Binding
	SelectAll key.Binding

	// Sorting (when header is focused)
	ToggleSort      key.Binding
	ToggleMultiSort key.Binding

	// Sorting (from any row)
	SortColumn      key.Binding
	MultiSortColumn key.Binding

	// Editing
	StartEdit   key.Binding
	ConfirmEdit key.Binding
	CancelEdit  key.Binding

	// Filtering
	QuickFilter  key.Binding
	ColumnFilter key.Binding

	// Grouping
	ToggleGroupColumn key.Binding
	ExpandGroup       key.Binding
	CollapseGroup     key.Binding
	ExpandAll         key.Binding
	CollapseAll       key.Binding

	// General
	Help key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "half page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "first row"),
		),
		End: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "last row"),
		),
		LineStart: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "first column"),
		),
		LineEnd: key.NewBinding(
			key.WithKeys("$"),
			key.WithHelp("$", "last column"),
		),
		GoToHeader: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "go to header"),
		),
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "select"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "select all"),
		),
		ToggleSort: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "sort"),
		),
		ToggleMultiSort: key.NewBinding(
			key.WithKeys("shift+enter"),
			key.WithHelp("shift+enter", "multi-sort"),
		),
		SortColumn: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort column"),
		),
		MultiSortColumn: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "multi-sort column"),
		),
		StartEdit: key.NewBinding(
			key.WithKeys("enter", "f2"),
			key.WithHelp("enter/F2", "edit"),
		),
		ConfirmEdit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		CancelEdit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		QuickFilter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "quick filter"),
		),
		ColumnFilter: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "column filter"),
		),
		ToggleGroupColumn: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "toggle grouping"),
		),
		ExpandGroup: key.NewBinding(
			key.WithKeys("enter", "right"),
			key.WithHelp("enter/→", "expand"),
		),
		CollapseGroup: key.NewBinding(
			key.WithKeys("left", "backspace"),
			key.WithHelp("←/bksp", "collapse"),
		),
		ExpandAll: key.NewBinding(
			key.WithKeys("shift+right"),
			key.WithHelp("shift+→", "expand all"),
		),
		CollapseAll: key.NewBinding(
			key.WithKeys("shift+left"),
			key.WithHelp("shift+←", "collapse all"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}
