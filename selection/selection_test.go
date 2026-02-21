package selection

import (
	"sort"
	"testing"
)

// --- SelectNone ---

func TestSelectNoneToggle(t *testing.T) {
	m := New(SelectNone)
	m.Toggle("a")
	if m.Count() != 0 {
		t.Error("SelectNone: Toggle should be no-op")
	}
}

func TestSelectNoneSelect(t *testing.T) {
	m := New(SelectNone)
	m.Select("a")
	if m.Count() != 0 {
		t.Error("SelectNone: Select should be no-op")
	}
}

func TestSelectNoneSelectAll(t *testing.T) {
	m := New(SelectNone)
	m.SelectAll([]string{"a", "b"})
	if m.Count() != 0 {
		t.Error("SelectNone: SelectAll should be no-op")
	}
}

func TestSelectNoneSelectedIDs(t *testing.T) {
	m := New(SelectNone)
	if len(m.SelectedIDs()) != 0 {
		t.Error("SelectNone: SelectedIDs should be empty")
	}
}

// --- SelectSingle ---

func TestSelectSingleToggle(t *testing.T) {
	m := New(SelectSingle)
	m.Toggle("a")
	if !m.IsSelected("a") {
		t.Error("should select a")
	}
	if m.Count() != 1 {
		t.Errorf("count should be 1, got %d", m.Count())
	}
}

func TestSelectSingleToggleDeselect(t *testing.T) {
	m := New(SelectSingle)
	m.Toggle("a")
	m.Toggle("a")
	if m.IsSelected("a") {
		t.Error("second toggle should deselect a")
	}
	if m.Count() != 0 {
		t.Errorf("count should be 0, got %d", m.Count())
	}
}

func TestSelectSingleOnlyOne(t *testing.T) {
	m := New(SelectSingle)
	m.Toggle("a")
	m.Toggle("b")
	if m.IsSelected("a") {
		t.Error("a should be deselected when b is toggled")
	}
	if !m.IsSelected("b") {
		t.Error("b should be selected")
	}
	if m.Count() != 1 {
		t.Errorf("count should be 1, got %d", m.Count())
	}
}

func TestSelectSingleSelectReplaces(t *testing.T) {
	m := New(SelectSingle)
	m.Select("a")
	m.Select("b")
	if m.IsSelected("a") {
		t.Error("a should be replaced by b")
	}
	if !m.IsSelected("b") {
		t.Error("b should be selected")
	}
}

func TestSelectSingleSelectAllNoop(t *testing.T) {
	m := New(SelectSingle)
	m.SelectAll([]string{"a", "b", "c"})
	if m.Count() != 0 {
		t.Error("SelectAll should be no-op in single mode")
	}
}

func TestSelectSingleDeselectAll(t *testing.T) {
	m := New(SelectSingle)
	m.Select("a")
	m.DeselectAll()
	if m.Count() != 0 {
		t.Error("DeselectAll should clear selection")
	}
}

// --- SelectMulti ---

func TestSelectMultiToggleAccumulates(t *testing.T) {
	m := New(SelectMulti)
	m.Toggle("a")
	m.Toggle("b")
	if !m.IsSelected("a") || !m.IsSelected("b") {
		t.Error("both a and b should be selected")
	}
	if m.Count() != 2 {
		t.Errorf("count should be 2, got %d", m.Count())
	}
}

func TestSelectMultiToggleDeselects(t *testing.T) {
	m := New(SelectMulti)
	m.Toggle("a")
	m.Toggle("b")
	m.Toggle("a")
	if m.IsSelected("a") {
		t.Error("a should be deselected")
	}
	if !m.IsSelected("b") {
		t.Error("b should still be selected")
	}
	if m.Count() != 1 {
		t.Errorf("count should be 1, got %d", m.Count())
	}
}

func TestSelectMultiSelectAdds(t *testing.T) {
	m := New(SelectMulti)
	m.Select("a")
	m.Select("b")
	if !m.IsSelected("a") || !m.IsSelected("b") {
		t.Error("both should be selected")
	}
}

func TestSelectMultiSelectAll(t *testing.T) {
	m := New(SelectMulti)
	m.SelectAll([]string{"a", "b", "c"})
	if m.Count() != 3 {
		t.Errorf("count should be 3, got %d", m.Count())
	}
}

func TestSelectMultiDeselectAll(t *testing.T) {
	m := New(SelectMulti)
	m.SelectAll([]string{"a", "b", "c"})
	m.DeselectAll()
	if m.Count() != 0 {
		t.Error("DeselectAll should clear all")
	}
}

func TestSelectMultiDeselect(t *testing.T) {
	m := New(SelectMulti)
	m.SelectAll([]string{"a", "b", "c"})
	m.Deselect("b")
	if m.IsSelected("b") {
		t.Error("b should be deselected")
	}
	if m.Count() != 2 {
		t.Errorf("count should be 2, got %d", m.Count())
	}
}

// --- Count and SelectedIDs ---

func TestSelectedIDsContent(t *testing.T) {
	m := New(SelectMulti)
	m.Select("x")
	m.Select("y")
	m.Select("z")

	ids := m.SelectedIDs()
	sort.Strings(ids)
	if len(ids) != 3 || ids[0] != "x" || ids[1] != "y" || ids[2] != "z" {
		t.Errorf("unexpected IDs: %v", ids)
	}
}

// --- Retain ---

func TestRetain(t *testing.T) {
	m := New(SelectMulti)
	m.SelectAll([]string{"a", "b", "c", "d"})
	if m.Count() != 4 {
		t.Fatalf("expected 4 selected, got %d", m.Count())
	}

	// Retain only "b" and "d"
	m.Retain(map[string]bool{"b": true, "d": true})
	if m.Count() != 2 {
		t.Errorf("expected 2 after Retain, got %d", m.Count())
	}
	if !m.IsSelected("b") {
		t.Error("expected b to be retained")
	}
	if !m.IsSelected("d") {
		t.Error("expected d to be retained")
	}
	if m.IsSelected("a") {
		t.Error("expected a to be pruned")
	}
	if m.IsSelected("c") {
		t.Error("expected c to be pruned")
	}
}

func TestRetainEmptyValidSet(t *testing.T) {
	m := New(SelectMulti)
	m.SelectAll([]string{"a", "b"})
	m.Retain(map[string]bool{})
	if m.Count() != 0 {
		t.Errorf("expected 0 after Retain with empty set, got %d", m.Count())
	}
}

// --- Anchor ---

func TestAnchorDefault(t *testing.T) {
	m := New(SelectMulti)
	if m.Anchor() != -1 {
		t.Errorf("default anchor should be -1, got %d", m.Anchor())
	}
}

func TestAnchorSetGet(t *testing.T) {
	m := New(SelectMulti)
	m.SetAnchor(5)
	if m.Anchor() != 5 {
		t.Errorf("anchor should be 5, got %d", m.Anchor())
	}
}
