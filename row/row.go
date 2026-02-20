// Package row defines row node types for the tea-grid component.
package row

// PinPosition indicates whether a row is pinned to the top or bottom.
type PinPosition int

const (
	PinNone PinPosition = iota
	PinTop
	PinBottom
)

// RowNode wraps a user-supplied row value with runtime metadata.
type RowNode[T any] struct {
	// Data is the user-supplied row value.
	Data T

	// Runtime state (managed by the grid)
	ID         string // Unique row ID.
	RowIndex   int    // Current display index (post sort/filter/group).
	Selected   bool
	Expanded   bool        // For group rows.
	RowHeight  int         // In terminal lines. Default: 1.
	Pinned     PinPosition // Top, Bottom, or None.
	IsGroup    bool        // True if this is a synthetic group row.
	GroupKey   string      // The value this group represents.
	GroupLevel int         // Nesting depth (0 = top).
	Children   []*RowNode[T]
	Parent     *RowNode[T]
}
