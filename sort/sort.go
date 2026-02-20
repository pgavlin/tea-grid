// Package sort provides sorting state and logic for the tea-grid component.
package sort

import (
	"github.com/pgavlin/tea-grid/column"
)

// SortCriterion describes a single sort column and direction.
type SortCriterion struct {
	ColID     string
	Direction column.SortDirection // Asc or Desc.
}

// Model holds the current sort state.
type Model[T any] struct {
	SortOrder  []SortCriterion // Ordered list of active sorts.
	MultiSort  bool            // Whether multi-column sort is enabled.
}

// ToggleSort cycles a column through asc -> desc -> none.
// If the column is not currently sorted, it becomes the primary sort (asc).
// Returns the updated sort order.
func (m *Model[T]) ToggleSort(colID string) {
	for i, sc := range m.SortOrder {
		if sc.ColID == colID {
			switch sc.Direction {
			case column.SortAsc:
				m.SortOrder[i].Direction = column.SortDesc
			case column.SortDesc:
				// Remove from sort order
				m.SortOrder = append(m.SortOrder[:i], m.SortOrder[i+1:]...)
			}
			return
		}
	}
	// Not currently sorted — set as primary sort (asc), replacing existing
	m.SortOrder = []SortCriterion{{ColID: colID, Direction: column.SortAsc}}
}

// AddSort adds a column to multi-sort or toggles it if already present.
func (m *Model[T]) AddSort(colID string) {
	for i, sc := range m.SortOrder {
		if sc.ColID == colID {
			switch sc.Direction {
			case column.SortAsc:
				m.SortOrder[i].Direction = column.SortDesc
			case column.SortDesc:
				m.SortOrder = append(m.SortOrder[:i], m.SortOrder[i+1:]...)
			}
			return
		}
	}
	m.SortOrder = append(m.SortOrder, SortCriterion{ColID: colID, Direction: column.SortAsc})
}

// Clear removes all sort criteria.
func (m *Model[T]) Clear() {
	m.SortOrder = nil
}

// DirectionFor returns the sort direction for the given column, or SortNone.
func (m *Model[T]) DirectionFor(colID string) column.SortDirection {
	for _, sc := range m.SortOrder {
		if sc.ColID == colID {
			return sc.Direction
		}
	}
	return column.SortNone
}

// IndexFor returns the sort index (0-based) for the given column, or -1.
func (m *Model[T]) IndexFor(colID string) int {
	for i, sc := range m.SortOrder {
		if sc.ColID == colID {
			return i
		}
	}
	return -1
}
