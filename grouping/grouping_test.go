package grouping

import (
	"testing"

	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/row"
)

type testRow struct {
	Name       string
	Department string
	Salary     float64
}

func testCols() []column.ColDef[testRow] {
	return []column.ColDef[testRow]{
		{
			ColID:       "Name",
			HeaderName:  "Name",
			ValueGetter: func(r testRow) any { return r.Name },
		},
		{
			ColID:       "Department",
			HeaderName:  "Department",
			ValueGetter: func(r testRow) any { return r.Department },
		},
		{
			ColID:       "Salary",
			HeaderName:  "Salary",
			ValueGetter: func(r testRow) any { return r.Salary },
		},
	}
}

func testRows() []row.RowNode[testRow] {
	data := []testRow{
		{"Alice", "Eng", 100},
		{"Bob", "Eng", 200},
		{"Carol", "Eng", 150},
		{"Dave", "Sales", 120},
		{"Eve", "Sales", 180},
		{"Frank", "Sales", 160},
	}
	nodes := make([]row.RowNode[testRow], len(data))
	for i, d := range data {
		nodes[i] = row.RowNode[testRow]{Data: d, ID: d.Name}
	}
	return nodes
}

// --- BuildGroups ---

func TestBuildGroupsSingleLevel(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{}
	groups := BuildGroups(rows, cols, []string{"Department"}, expanded, -1)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}

func TestBuildGroupsNodeProperties(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{}
	groups := BuildGroups(rows, cols, []string{"Department"}, expanded, -1)

	for _, g := range groups {
		if !g.IsGroup {
			t.Error("group node should have IsGroup=true")
		}
		if g.GroupKey != "Eng" && g.GroupKey != "Sales" {
			t.Errorf("unexpected GroupKey: %q", g.GroupKey)
		}
		if g.GroupLevel != 0 {
			t.Errorf("top-level group should have GroupLevel=0, got %d", g.GroupLevel)
		}
		if len(g.Children) != 3 {
			t.Errorf("group %q: expected 3 children, got %d", g.GroupKey, len(g.Children))
		}
	}
}

func TestBuildGroupsChildParentPointers(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{}
	groups := BuildGroups(rows, cols, []string{"Department"}, expanded, -1)

	for _, g := range groups {
		for _, child := range g.Children {
			if child.Parent != g {
				t.Errorf("child %q: Parent should point to group %q", child.ID, g.GroupKey)
			}
		}
	}
}

func TestBuildGroupsMultiLevel(t *testing.T) {
	// Create rows with two groupable dimensions
	rows := []row.RowNode[testRow]{
		{Data: testRow{"Alice", "Eng", 100}, ID: "1"},
		{Data: testRow{"Bob", "Eng", 200}, ID: "2"},
		{Data: testRow{"Carol", "Sales", 150}, ID: "3"},
		{Data: testRow{"Dave", "Sales", 120}, ID: "4"},
	}
	cols := []column.ColDef[testRow]{
		{ColID: "Department", ValueGetter: func(r testRow) any { return r.Department }},
		{ColID: "Name", ValueGetter: func(r testRow) any { return r.Name }},
	}
	expanded := map[string]bool{}
	groups := BuildGroups(rows, cols, []string{"Department", "Name"}, expanded, -1)

	// Should have 2 top-level groups
	if len(groups) != 2 {
		t.Fatalf("expected 2 top groups, got %d", len(groups))
	}

	// Each should have sub-groups
	for _, g := range groups {
		if len(g.Children) == 0 {
			t.Errorf("group %q should have children", g.GroupKey)
		}
		for _, child := range g.Children {
			if !child.IsGroup {
				t.Errorf("second-level children should be groups")
			}
			if child.GroupLevel != 1 {
				t.Errorf("second-level group should have GroupLevel=1, got %d", child.GroupLevel)
			}
		}
	}
}

func TestBuildGroupsNoGroupColumns(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{}
	result := BuildGroups(rows, cols, nil, expanded, -1)

	if len(result) != len(rows) {
		t.Fatalf("no group cols: expected %d rows, got %d", len(rows), len(result))
	}
	for _, rn := range result {
		if rn.IsGroup {
			t.Error("should not create group nodes")
		}
	}
}

func TestBuildGroupsMissingColumn(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{}
	result := BuildGroups(rows, cols, []string{"NonExistent"}, expanded, -1)

	if len(result) != len(rows) {
		t.Fatalf("missing col: expected %d rows, got %d", len(rows), len(result))
	}
}

func TestBuildGroupsEmpty(t *testing.T) {
	cols := testCols()
	expanded := map[string]bool{}
	result := BuildGroups([]row.RowNode[testRow]{}, cols, []string{"Department"}, expanded, -1)

	if len(result) != 0 {
		t.Fatalf("empty rows: expected 0, got %d", len(result))
	}
}

func TestBuildGroupsPreservesOrder(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{}
	groups := BuildGroups(rows, cols, []string{"Department"}, expanded, -1)

	// First group should be "Eng" (appears first in input)
	if groups[0].GroupKey != "Eng" {
		t.Errorf("first group should be Eng, got %q", groups[0].GroupKey)
	}

	// Children within Eng should be Alice, Bob, Carol (input order)
	engChildren := groups[0].Children
	if engChildren[0].Data.Name != "Alice" {
		t.Errorf("first Eng child should be Alice, got %s", engChildren[0].Data.Name)
	}
}

// --- FlattenGroups ---

func TestFlattenGroupsAllExpanded(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{}
	groups := BuildGroups(rows, cols, []string{"Department"}, expanded, -1)

	flat := FlattenGroups(groups)
	// 2 group nodes + 6 children = 8
	if len(flat) != 8 {
		t.Fatalf("all expanded: expected 8 rows, got %d", len(flat))
	}
}

func TestFlattenGroupsAllCollapsed(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{}
	groups := BuildGroups(rows, cols, []string{"Department"}, expanded, 0)

	flat := FlattenGroups(groups)
	// Only 2 group nodes (collapsed)
	if len(flat) != 2 {
		t.Fatalf("all collapsed: expected 2 rows, got %d", len(flat))
	}
}

func TestFlattenGroupsMixed(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{"Eng": true}
	groups := BuildGroups(rows, cols, []string{"Department"}, expanded, 0)

	flat := FlattenGroups(groups)
	// Eng expanded: 1 group + 3 children = 4. Sales collapsed: 1 group = 1. Total = 5
	if len(flat) != 5 {
		t.Fatalf("mixed: expected 5 rows, got %d", len(flat))
	}
}

func TestFlattenGroupsNoGroups(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{}
	result := BuildGroups(rows, cols, nil, expanded, -1)

	flat := FlattenGroups(result)
	if len(flat) != len(rows) {
		t.Fatalf("no groups: expected %d, got %d", len(rows), len(flat))
	}
}

// --- Expand/Collapse state ---

func TestDefaultExpandedAll(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{}
	groups := BuildGroups(rows, cols, []string{"Department"}, expanded, -1)

	for _, g := range groups {
		if !g.Expanded {
			t.Errorf("DefaultExpanded=-1: group %q should be expanded", g.GroupKey)
		}
	}
}

func TestDefaultExpandedNone(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{}
	groups := BuildGroups(rows, cols, []string{"Department"}, expanded, 0)

	for _, g := range groups {
		if g.Expanded {
			t.Errorf("DefaultExpanded=0: group %q should be collapsed", g.GroupKey)
		}
	}
}

func TestExplicitExpandedOverridesDefault(t *testing.T) {
	rows := testRows()
	cols := testCols()
	expanded := map[string]bool{"Eng": true, "Sales": false}
	groups := BuildGroups(rows, cols, []string{"Department"}, expanded, 0)

	for _, g := range groups {
		switch g.GroupKey {
		case "Eng":
			if !g.Expanded {
				t.Error("Eng should be expanded (explicit override)")
			}
		case "Sales":
			if g.Expanded {
				t.Error("Sales should be collapsed (explicit override)")
			}
		}
	}
}

func TestSetExpandedAndIsExpanded(t *testing.T) {
	m := New[testRow](nil, -1)
	m.SetExpanded("group1", true)
	if !m.IsExpanded("group1") {
		t.Error("should be expanded")
	}
	m.SetExpanded("group1", false)
	if m.IsExpanded("group1") {
		t.Error("should be collapsed")
	}
}

func TestExpandAllCollapseAll(t *testing.T) {
	rows := testRows()
	cols := testCols()
	m := New[testRow]([]string{"Department"}, -1)
	groups := BuildGroups(rows, cols, m.GroupColumns, m.Expanded, m.DefaultExpanded)

	m.CollapseAll(groups)
	for key := range m.Expanded {
		if m.Expanded[key] {
			t.Errorf("CollapseAll: %q should be collapsed", key)
		}
	}

	m.ExpandAll(groups)
	for key := range m.Expanded {
		if !m.Expanded[key] {
			t.Errorf("ExpandAll: %q should be expanded", key)
		}
	}
}

// --- ToggleGroupColumn ---

func TestToggleGroupColumnAdd(t *testing.T) {
	m := New[testRow](nil, -1)
	m.ToggleGroupColumn("col1")
	if len(m.GroupColumns) != 1 || m.GroupColumns[0] != "col1" {
		t.Errorf("expected [col1], got %v", m.GroupColumns)
	}
}

func TestToggleGroupColumnRemove(t *testing.T) {
	m := New[testRow]([]string{"col1"}, -1)
	m.ToggleGroupColumn("col1")
	if len(m.GroupColumns) != 0 {
		t.Errorf("expected empty, got %v", m.GroupColumns)
	}
}

func TestToggleGroupColumnAddSecond(t *testing.T) {
	m := New[testRow]([]string{"col1"}, -1)
	m.ToggleGroupColumn("col2")
	if len(m.GroupColumns) != 2 {
		t.Fatalf("expected 2, got %d", len(m.GroupColumns))
	}
	if m.GroupColumns[0] != "col1" || m.GroupColumns[1] != "col2" {
		t.Errorf("expected [col1 col2], got %v", m.GroupColumns)
	}
}

// --- Aggregate ---

func TestAggregateSum(t *testing.T) {
	result := Aggregate([]any{1, 2, 3}, "sum")
	if result != 6.0 {
		t.Errorf("sum: expected 6.0, got %v", result)
	}
}

func TestAggregateAvg(t *testing.T) {
	result := Aggregate([]any{2, 4, 6}, "avg")
	if result != 4.0 {
		t.Errorf("avg: expected 4.0, got %v", result)
	}
}

func TestAggregateCount(t *testing.T) {
	result := Aggregate([]any{"a", "b", "c"}, "count")
	if result != 3 {
		t.Errorf("count: expected 3, got %v", result)
	}
}

func TestAggregateMin(t *testing.T) {
	result := Aggregate([]any{3, 1, 2}, "min")
	if result != 1.0 {
		t.Errorf("min: expected 1.0, got %v", result)
	}
}

func TestAggregateMax(t *testing.T) {
	result := Aggregate([]any{1, 3, 2}, "max")
	if result != 3.0 {
		t.Errorf("max: expected 3.0, got %v", result)
	}
}

func TestAggregateFirst(t *testing.T) {
	result := Aggregate([]any{"a", "b", "c"}, "first")
	if result != "a" {
		t.Errorf("first: expected a, got %v", result)
	}
}

func TestAggregateLast(t *testing.T) {
	result := Aggregate([]any{"a", "b", "c"}, "last")
	if result != "c" {
		t.Errorf("last: expected c, got %v", result)
	}
}

func TestAggregateUnknown(t *testing.T) {
	result := Aggregate([]any{1, 2}, "unknown")
	if result != nil {
		t.Errorf("unknown func: expected nil, got %v", result)
	}
}

func TestAggregateSumEmpty(t *testing.T) {
	result := Aggregate([]any{}, "sum")
	if result != 0.0 {
		t.Errorf("sum empty: expected 0.0, got %v", result)
	}
}

func TestAggregateAvgEmpty(t *testing.T) {
	result := Aggregate([]any{}, "avg")
	if result != 0.0 {
		t.Errorf("avg empty: expected 0.0, got %v", result)
	}
}

func TestAggregateMinEmpty(t *testing.T) {
	result := Aggregate([]any{}, "min")
	if result != nil {
		t.Errorf("min empty: expected nil, got %v", result)
	}
}

func TestAggregateMaxEmpty(t *testing.T) {
	result := Aggregate([]any{}, "max")
	if result != nil {
		t.Errorf("max empty: expected nil, got %v", result)
	}
}

func TestAggregateFirstEmpty(t *testing.T) {
	result := Aggregate([]any{}, "first")
	if result != nil {
		t.Errorf("first empty: expected nil, got %v", result)
	}
}

func TestAggregateLastEmpty(t *testing.T) {
	result := Aggregate([]any{}, "last")
	if result != nil {
		t.Errorf("last empty: expected nil, got %v", result)
	}
}

func TestAggregateMixedNumericTypes(t *testing.T) {
	result := Aggregate([]any{int(1), float64(2.5), int64(3)}, "sum")
	if result != 6.5 {
		t.Errorf("mixed types sum: expected 6.5, got %v", result)
	}
}
