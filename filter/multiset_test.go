package filter

import "testing"

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
