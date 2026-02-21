// Package grouping provides row grouping and aggregation for the tea-grid component.
package grouping

import (
	"fmt"
	"math"

	"github.com/pgavlin/tea-grid/data"
)

// Model holds the current grouping state.
type Model[T any] struct {
	GroupColumns    []string        // ColumnIDs of columns being grouped, in order.
	Expanded        map[string]bool // GroupKey -> expanded state.
	DefaultExpanded int             // Number of levels expanded by default. -1 = all.
}

// New creates a new grouping model.
func New[T any](groupCols []string, defaultExpanded int) Model[T] {
	return Model[T]{
		GroupColumns:    groupCols,
		Expanded:        make(map[string]bool),
		DefaultExpanded: defaultExpanded,
	}
}

// IsExpanded returns whether a group is expanded.
func (m *Model[T]) IsExpanded(groupKey string) bool {
	expanded, exists := m.Expanded[groupKey]
	return exists && expanded
}

// SetExpanded sets the expanded state of a group.
func (m *Model[T]) SetExpanded(groupKey string, expanded bool) {
	m.Expanded[groupKey] = expanded
}

// ExpandAll expands all groups.
func (m *Model[T]) ExpandAll(groups []*data.RowNode[T]) {
	for _, g := range groups {
		if g.IsGroup {
			m.Expanded[g.GroupKey] = true
			m.ExpandAll(g.Children)
		}
	}
}

// ToggleGroupColumn adds colID to GroupColumns if absent, or removes it if present.
func (m *Model[T]) ToggleGroupColumn(colID string) {
	for i, id := range m.GroupColumns {
		if id == colID {
			m.GroupColumns = append(m.GroupColumns[:i], m.GroupColumns[i+1:]...)
			return
		}
	}
	m.GroupColumns = append(m.GroupColumns, colID)
}

// CollapseAll collapses all groups.
func (m *Model[T]) CollapseAll(groups []*data.RowNode[T]) {
	for _, g := range groups {
		if g.IsGroup {
			m.Expanded[g.GroupKey] = false
			m.CollapseAll(g.Children)
		}
	}
}

// BuildGroups organizes rows into a group tree based on GroupColumns.
// Returns the top-level group nodes.
func BuildGroups[T any](
	rows []data.RowNode[T],
	cols []data.Column[T],
	groupCols []string,
	expanded map[string]bool,
	defaultExpanded int,
) []*data.RowNode[T] {
	if len(groupCols) == 0 {
		result := make([]*data.RowNode[T], len(rows))
		for i := range rows {
			result[i] = &rows[i]
		}
		return result
	}

	// Find the column definition for the first group column
	var groupCol *data.Column[T]
	for i := range cols {
		if cols[i].ColumnID == groupCols[0] {
			groupCol = &cols[i]
			break
		}
	}
	if groupCol == nil {
		result := make([]*data.RowNode[T], len(rows))
		for i := range rows {
			result[i] = &rows[i]
		}
		return result
	}

	// Group rows by the column value
	groupMap := make(map[string][]*data.RowNode[T])
	var groupOrder []string
	for i := range rows {
		val := groupCol.ValueGetter(rows[i].Data)
		key := fmt.Sprintf("%v", val)
		if _, exists := groupMap[key]; !exists {
			groupOrder = append(groupOrder, key)
		}
		groupMap[key] = append(groupMap[key], &rows[i])
	}

	// Create group nodes
	var groups []*data.RowNode[T]
	for _, key := range groupOrder {
		children := groupMap[key]
		groupNode := &data.RowNode[T]{
			IsGroup:    true,
			GroupKey:   key,
			GroupLevel: 0,
			Children:   children,
		}

		// Set expanded state
		if exp, exists := expanded[key]; exists {
			groupNode.Expanded = exp
		} else {
			groupNode.Expanded = defaultExpanded == -1 || defaultExpanded > 0
		}

		// Set parent references
		for _, child := range children {
			child.Parent = groupNode
		}

		// Recursively group children if there are more group columns
		if len(groupCols) > 1 {
			childRows := make([]data.RowNode[T], len(children))
			for i, c := range children {
				childRows[i] = *c
			}
			subGroups := BuildGroups(childRows, cols, groupCols[1:], expanded, defaultExpanded-1)
			groupNode.Children = subGroups
			for _, sg := range subGroups {
				sg.Parent = groupNode
				sg.GroupLevel = 1
			}
		}

		groups = append(groups, groupNode)
	}

	return groups
}

// FlattenGroups flattens the group tree into a display list, respecting expanded state.
func FlattenGroups[T any](groups []*data.RowNode[T]) []data.RowNode[T] {
	var result []data.RowNode[T]
	for _, g := range groups {
		if g.IsGroup {
			result = append(result, *g)
			if g.Expanded {
				result = append(result, FlattenGroups(g.Children)...)
			}
		} else {
			result = append(result, *g)
		}
	}
	return result
}

// Aggregate computes aggregated values for a group node using the specified function.
func Aggregate(values []any, funcName string) any {
	switch funcName {
	case "sum":
		return aggSum(values)
	case "avg":
		return aggAvg(values)
	case "count":
		return len(values)
	case "min":
		return aggMin(values)
	case "max":
		return aggMax(values)
	case "first":
		if len(values) > 0 {
			return values[0]
		}
		return nil
	case "last":
		if len(values) > 0 {
			return values[len(values)-1]
		}
		return nil
	default:
		return nil
	}
}

func aggSum(values []any) any {
	var sum float64
	for _, v := range values {
		sum += toFloat64(v)
	}
	return sum
}

func aggAvg(values []any) any {
	if len(values) == 0 {
		return 0.0
	}
	sum := toFloat64(aggSum(values))
	return sum / float64(len(values))
}

func aggMin(values []any) any {
	if len(values) == 0 {
		return nil
	}
	min := math.MaxFloat64
	for _, v := range values {
		f := toFloat64(v)
		if f < min {
			min = f
		}
	}
	return min
}

func aggMax(values []any) any {
	if len(values) == 0 {
		return nil
	}
	max := -math.MaxFloat64
	for _, v := range values {
		f := toFloat64(v)
		if f > max {
			max = f
		}
	}
	return max
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	default:
		return 0
	}
}
