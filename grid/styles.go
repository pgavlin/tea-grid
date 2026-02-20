package grid

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles defines all visual styling for the grid.
type Styles struct {
	// Table-level
	Table lipgloss.Style // Outer container.

	// Header
	Header     lipgloss.Style // Header row.
	HeaderCell lipgloss.Style // Individual header cell.
	SortAsc    string         // Ascending sort indicator (default: "▲").
	SortDesc   string         // Descending sort indicator (default: "▼").

	// Cells
	Cell         lipgloss.Style // Default cell style.
	CellFocused  lipgloss.Style // Focused cell highlight.
	CellSelected lipgloss.Style // Selected row highlight.

	// Pinning
	PinnedLeft   lipgloss.Style // Pinned-left region.
	PinnedRight  lipgloss.Style // Pinned-right region.
	PinnedRow    lipgloss.Style // Pinned row (top/bottom).
	PinSeparator string         // Vertical separator between pinned and scrollable.

	// Grouping
	GroupRow       lipgloss.Style
	GroupExpanded  string // Expanded indicator (default: "▼").
	GroupCollapsed string // Collapsed indicator (default: "▶").
	GroupIndent    int    // Indentation per level (default: 2).

	// Borders
	Border       lipgloss.Border // Border style.
	BorderHeader bool            // Show border below header.
	BorderRow    bool            // Show border between rows.
	BorderColumn bool            // Show border between columns.

	// Filtering
	FilterInput  lipgloss.Style // Quick filter input.
	FilterMatch  lipgloss.Style // Highlighted matching text.
	FilterActive string         // Active filter indicator in header (default: "⫧").

	// Editing
	EditorInput lipgloss.Style // Cell editor input.
	EditorError lipgloss.Style // Validation error.

	// Scrollbar
	Scrollbar      lipgloss.Style
	ScrollbarThumb lipgloss.Style

	// Alternating rows
	EvenRow lipgloss.Style
	OddRow  lipgloss.Style

	// Per-cell styling callback
	StyleFunc func(row, col int, data any) lipgloss.Style
}

// DefaultStyles returns the default grid styles.
func DefaultStyles() Styles {
	return Styles{
		Table: lipgloss.NewStyle(),

		Header: lipgloss.NewStyle().
			Bold(true).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()),
		HeaderCell: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1),

		SortAsc:  "▲",
		SortDesc: "▼",

		Cell: lipgloss.NewStyle().
			Padding(0, 1),
		CellFocused: lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("230")),
		CellSelected: lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("236")),

		PinnedLeft:   lipgloss.NewStyle(),
		PinnedRight:  lipgloss.NewStyle(),
		PinnedRow:    lipgloss.NewStyle().Bold(true),
		PinSeparator: "│",

		GroupRow: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1),
		GroupExpanded:  "▼",
		GroupCollapsed: "▶",
		GroupIndent:    2,

		Border:       lipgloss.NormalBorder(),
		BorderHeader: true,
		BorderRow:    false,
		BorderColumn: true,

		FilterInput: lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("236")),
		FilterMatch: lipgloss.NewStyle().
			Background(lipgloss.Color("178")).
			Foreground(lipgloss.Color("0")),
		FilterActive: "⫧",

		EditorInput: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")),
		EditorError: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),

		Scrollbar:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		ScrollbarThumb: lipgloss.NewStyle().Foreground(lipgloss.Color("248")),

		EvenRow: lipgloss.NewStyle(),
		OddRow:  lipgloss.NewStyle().Background(lipgloss.Color("235")),
	}
}
