package filter

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestMultiSetFilter_EmptyInactive(t *testing.T) {
	f := NewMultiSetFilter()
	if f.Active() {
		t.Errorf("Active() = true for empty filter, want false")
	}
}

func TestMultiSetFilter_AddRemoveConstraint(t *testing.T) {
	f := NewMultiSetFilter()
	f.AddConstraint("bug")
	f.AddConstraint("urgent")
	if !f.Active() {
		t.Errorf("Active() = false after AddConstraint, want true")
	}
	cs := f.Constraints()
	if len(cs) != 2 || cs[0] != "bug" || cs[1] != "urgent" {
		t.Errorf("Constraints = %v, want [bug urgent]", cs)
	}
	f.RemoveConstraint("bug")
	if cs := f.Constraints(); len(cs) != 1 || cs[0] != "urgent" {
		t.Errorf("after Remove: %v, want [urgent]", cs)
	}
}

func TestMultiSetFilter_Clear(t *testing.T) {
	f := NewMultiSetFilter()
	f.AddConstraint("bug")
	f.Clear()
	if f.Active() {
		t.Errorf("Active() = true after Clear, want false")
	}
}

func TestMultiSetFilter_AddConstraintIgnoresDuplicates(t *testing.T) {
	f := NewMultiSetFilter()
	f.AddConstraint("bug")
	f.AddConstraint("bug")
	if cs := f.Constraints(); len(cs) != 1 {
		t.Errorf("Constraints = %v, want [bug] (no duplicates)", cs)
	}
}

func TestMultiSetFilter_MatchesDefault(t *testing.T) {
	f := NewMultiSetFilter()
	f.AddConstraint("bug")
	f.AddConstraint("urgent")
	if !f.Matches([]string{"bug", "urgent", "ux"}) {
		t.Errorf("Matches([bug urgent ux]) = false, want true")
	}
	if f.Matches([]string{"bug"}) {
		t.Errorf("Matches([bug]) = true, want false (urgent missing)")
	}
}

func TestMultiSetFilter_MatchesCustom(t *testing.T) {
	f := NewMultiSetFilter(WithMultiSetMatcher(func(rowValue any, c string) bool {
		s, _ := rowValue.(string)
		return s == c
	}))
	f.AddConstraint("foo")
	if !f.Matches("foo") {
		t.Errorf("Matches(foo) = false")
	}
	if f.Matches("bar") {
		t.Errorf("Matches(bar) = true, want false")
	}
}

func TestMultiSetFilter_FocusEditing(t *testing.T) {
	f := NewMultiSetFilter()
	f.AddConstraint("bug")
	f.AddConstraint("urgent")

	g, _ := f.Update(FilterFocusMsg{Width: 40, MaxLines: 6})
	f = g.(*MultiSetFilter)
	if !f.editing {
		t.Errorf("editing=false after FilterFocusMsg")
	}
	if f.focused != 0 {
		t.Errorf("focused=%d after focus, want 0", f.focused)
	}

	g, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	f = g.(*MultiSetFilter)
	if f.focused != 1 {
		t.Errorf("focused=%d after Down, want 1", f.focused)
	}

	g, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	f = g.(*MultiSetFilter)
	g, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	f = g.(*MultiSetFilter)
	if f.focused != 0 {
		t.Errorf("focused=%d after Up Up, want 0", f.focused)
	}
}

func TestMultiSetFilter_DeleteFocusedConstraint(t *testing.T) {
	f := NewMultiSetFilter()
	f.AddConstraint("bug")
	f.AddConstraint("urgent")
	g, _ := f.Update(FilterFocusMsg{Width: 40, MaxLines: 6})
	f = g.(*MultiSetFilter)

	g, _ = f.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	f = g.(*MultiSetFilter)
	cs := f.Constraints()
	if len(cs) != 1 || cs[0] != "urgent" {
		t.Errorf("after d on focus 0: %v, want [urgent]", cs)
	}
}

func TestMultiSetFilter_ViewRendersConstraints(t *testing.T) {
	f := NewMultiSetFilter()
	f.AddConstraint("bug")
	f.AddConstraint("urgent")
	g, _ := f.Update(FilterFocusMsg{Width: 40, MaxLines: 6})
	f = g.(*MultiSetFilter)
	out := f.View()
	if !strings.Contains(out, "bug") || !strings.Contains(out, "urgent") {
		t.Errorf("View() = %q, want it to contain bug and urgent", out)
	}
}

func TestMultiSetFilter_ClauseInactive(t *testing.T) {
	f := NewMultiSetFilter()
	if _, _, ok := f.Clause(); ok {
		t.Errorf("Clause() ok=true for empty filter")
	}
}

func TestMultiSetFilter_ClauseOneValuePerConstraint(t *testing.T) {
	f := NewMultiSetFilter()
	f.AddConstraint("bug")
	f.AddConstraint("urgent")
	values, negate, ok := f.Clause()
	if !ok || negate || len(values) != 2 {
		t.Errorf("Clause() = (%v, negate=%v, ok=%v), want ([bug urgent], false, true)", values, negate, ok)
	}
	if values[0] != "bug" || values[1] != "urgent" {
		t.Errorf("Clause values = %v, want [bug urgent]", values)
	}
}

func TestMultiSetFilter_SetClauseAppends(t *testing.T) {
	f := NewMultiSetFilter()
	if err := f.SetClause([]string{"bug"}, false); err != nil {
		t.Fatalf("SetClause: %v", err)
	}
	if err := f.SetClause([]string{"urgent"}, false); err != nil {
		t.Fatalf("SetClause: %v", err)
	}
	cs := f.Constraints()
	if len(cs) != 2 || cs[0] != "bug" || cs[1] != "urgent" {
		t.Errorf("after two SetClause calls: %v, want [bug urgent]", cs)
	}
}

func TestMultiSetFilter_SetClauseRejectsCommaList(t *testing.T) {
	f := NewMultiSetFilter()
	if err := f.SetClause([]string{"a", "b"}, false); err == nil {
		t.Errorf("SetClause(2 values): err=nil, want error")
	}
}
