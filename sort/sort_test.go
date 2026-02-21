package sort

import (
	"testing"

	"github.com/pgavlin/tea-grid/data"
)

func TestToggleSortAddNew(t *testing.T) {
	m := Model[string]{}
	m.ToggleSort("col1")
	if len(m.SortOrder) != 1 {
		t.Fatalf("expected 1 sort, got %d", len(m.SortOrder))
	}
	if m.SortOrder[0].ColID != "col1" || m.SortOrder[0].Direction != data.SortAsc {
		t.Errorf("expected col1 Asc, got %+v", m.SortOrder[0])
	}
}

func TestToggleSortAscToDesc(t *testing.T) {
	m := Model[string]{SortOrder: []SortCriterion{{ColID: "col1", Direction: data.SortAsc}}}
	m.ToggleSort("col1")
	if len(m.SortOrder) != 1 {
		t.Fatalf("expected 1 sort, got %d", len(m.SortOrder))
	}
	if m.SortOrder[0].Direction != data.SortDesc {
		t.Errorf("expected Desc, got %v", m.SortOrder[0].Direction)
	}
}

func TestToggleSortDescToRemoved(t *testing.T) {
	m := Model[string]{SortOrder: []SortCriterion{{ColID: "col1", Direction: data.SortDesc}}}
	m.ToggleSort("col1")
	if len(m.SortOrder) != 0 {
		t.Fatalf("expected 0 sorts after desc toggle, got %d", len(m.SortOrder))
	}
}

func TestToggleSortReplaceOnNewColumn(t *testing.T) {
	m := Model[string]{SortOrder: []SortCriterion{{ColID: "col1", Direction: data.SortAsc}}}
	m.ToggleSort("col2")
	if len(m.SortOrder) != 1 {
		t.Fatalf("expected 1 sort, got %d", len(m.SortOrder))
	}
	if m.SortOrder[0].ColID != "col2" {
		t.Errorf("expected col2, got %s", m.SortOrder[0].ColID)
	}
}

func TestAddSortAppend(t *testing.T) {
	m := Model[string]{SortOrder: []SortCriterion{{ColID: "col1", Direction: data.SortAsc}}}
	m.AddSort("col2")
	if len(m.SortOrder) != 2 {
		t.Fatalf("expected 2 sorts, got %d", len(m.SortOrder))
	}
	if m.SortOrder[1].ColID != "col2" || m.SortOrder[1].Direction != data.SortAsc {
		t.Errorf("expected col2 Asc, got %+v", m.SortOrder[1])
	}
}

func TestAddSortCycleExisting(t *testing.T) {
	m := Model[string]{SortOrder: []SortCriterion{
		{ColID: "col1", Direction: data.SortAsc},
		{ColID: "col2", Direction: data.SortAsc},
	}}
	m.AddSort("col2")
	if len(m.SortOrder) != 2 {
		t.Fatalf("expected 2 sorts, got %d", len(m.SortOrder))
	}
	if m.SortOrder[1].Direction != data.SortDesc {
		t.Errorf("expected col2 Desc, got %v", m.SortOrder[1].Direction)
	}
}

func TestAddSortRemoveOnThirdToggle(t *testing.T) {
	m := Model[string]{SortOrder: []SortCriterion{
		{ColID: "col1", Direction: data.SortAsc},
		{ColID: "col2", Direction: data.SortDesc},
	}}
	m.AddSort("col2")
	if len(m.SortOrder) != 1 {
		t.Fatalf("expected 1 sort after removal, got %d", len(m.SortOrder))
	}
	if m.SortOrder[0].ColID != "col1" {
		t.Errorf("remaining sort should be col1, got %s", m.SortOrder[0].ColID)
	}
}

func TestClear(t *testing.T) {
	m := Model[string]{SortOrder: []SortCriterion{
		{ColID: "col1", Direction: data.SortAsc},
		{ColID: "col2", Direction: data.SortDesc},
	}}
	m.Clear()
	if len(m.SortOrder) != 0 {
		t.Fatalf("expected empty after Clear, got %d", len(m.SortOrder))
	}
}

func TestDirectionFor(t *testing.T) {
	m := Model[string]{SortOrder: []SortCriterion{
		{ColID: "col1", Direction: data.SortAsc},
	}}
	if d := m.DirectionFor("col1"); d != data.SortAsc {
		t.Errorf("expected SortAsc, got %v", d)
	}
	if d := m.DirectionFor("unknown"); d != data.SortNone {
		t.Errorf("expected SortNone, got %v", d)
	}
}

func TestIndexFor(t *testing.T) {
	m := Model[string]{SortOrder: []SortCriterion{
		{ColID: "col1", Direction: data.SortAsc},
		{ColID: "col2", Direction: data.SortDesc},
	}}
	if idx := m.IndexFor("col1"); idx != 0 {
		t.Errorf("expected 0, got %d", idx)
	}
	if idx := m.IndexFor("col2"); idx != 1 {
		t.Errorf("expected 1, got %d", idx)
	}
	if idx := m.IndexFor("unknown"); idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}
