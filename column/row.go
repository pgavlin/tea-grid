package column

// RowNode wraps a user-supplied row value with runtime metadata.
type RowNode[T any] struct {
	// Data is the user-supplied row value.
	Data T

	// Runtime state (managed by the grid)
	ID         string // Unique row ID.
	RowIndex   int    // Current display index (post sort/filter/group).
	Selected   bool
	Expanded   bool // For group rows.
	RowHeight  int  // In terminal lines. Default: 1.
	Pinned     Pin  // Top, Bottom, or None.
	IsGroup    bool // True if this is a synthetic group row.
	GroupKey   string      // The value this group represents.
	GroupLevel int         // Nesting depth (0 = top).
	Children   []*RowNode[T]
	Parent     *RowNode[T]
}
