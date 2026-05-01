package filter

import "testing"

func TestTextFilter_ClauseEmpty(t *testing.T) {
	f := NewTextFilter()
	if _, _, ok := f.Clause(); ok {
		t.Errorf("Clause() ok=true for empty filter, want false")
	}
}

func TestTextFilter_ClauseSubstring(t *testing.T) {
	f := NewTextFilter()
	f.SetText("hello")
	values, negate, ok := f.Clause()
	if !ok || negate || len(values) != 1 || values[0] != "hello" {
		t.Errorf("Clause() = (%v, %v, %v), want ([hello], false, true)", values, negate, ok)
	}
}

func TestTextFilter_ClauseRegexLossy(t *testing.T) {
	f := NewTextFilter()
	f.SetRegex(true)
	f.SetText("foo.*bar")
	if _, _, ok := f.Clause(); ok {
		t.Errorf("Clause() ok=true in regex mode, want false (lossy)")
	}
	if !f.Active() {
		t.Errorf("Active() = false, want true (regex filter is active)")
	}
}

func TestTextFilter_SetClauseRoundTrip(t *testing.T) {
	f := NewTextFilter()
	if err := f.SetClause([]string{"hello"}, false); err != nil {
		t.Fatalf("SetClause: %v", err)
	}
	values, _, _ := f.Clause()
	if len(values) != 1 || values[0] != "hello" {
		t.Errorf("round trip: got %v, want [hello]", values)
	}
}

func TestTextFilter_SetClauseResetsRegex(t *testing.T) {
	f := NewTextFilter()
	f.SetRegex(true)
	f.SetText("foo.*bar")
	if err := f.SetClause([]string{"hello"}, false); err != nil {
		t.Fatalf("SetClause: %v", err)
	}
	values, _, ok := f.Clause()
	if !ok || len(values) != 1 || values[0] != "hello" {
		t.Errorf("Clause() = (%v, ok=%v), want ([hello], ok=true) — regex should be reset", values, ok)
	}
}

func TestTextFilter_SetClauseRejectsMultipleValues(t *testing.T) {
	f := NewTextFilter()
	if err := f.SetClause([]string{"a", "b"}, false); err == nil {
		t.Errorf("SetClause with 2 values: err=nil, want error")
	}
}
