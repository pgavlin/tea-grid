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

func TestNumberFilter_ClauseEmpty(t *testing.T) {
	f := NewNumberFilter()
	if _, _, ok := f.Clause(); ok {
		t.Errorf("Clause() ok=true for empty filter")
	}
}

func TestNumberFilter_ClauseGreaterThan(t *testing.T) {
	f := NewNumberFilter()
	f.SetText(">10")
	values, negate, ok := f.Clause()
	if !ok || negate || len(values) != 1 || values[0] != ">10" {
		t.Errorf("Clause() = (%v, %v, %v), want ([>10], false, true)", values, negate, ok)
	}
}

func TestNumberFilter_ClauseRange(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("5..20")
	values, _, ok := f.Clause()
	if !ok || values[0] != "5..20" {
		t.Errorf("Clause() values=%v ok=%v, want [5..20] true", values, ok)
	}
}

func TestNumberFilter_SetClauseRoundTrip(t *testing.T) {
	cases := []string{">10", "<=5", "5..20", "=42", "!=7"}
	for _, expr := range cases {
		f := NewNumberFilter()
		if err := f.SetClause([]string{expr}, false); err != nil {
			t.Errorf("%s: SetClause: %v", expr, err)
			continue
		}
		values, _, ok := f.Clause()
		if !ok || values[0] != expr {
			t.Errorf("round trip %s: got values=%v ok=%v", expr, values, ok)
		}
	}
}

func TestNumberFilter_SetClauseRejectsBadInput(t *testing.T) {
	f := NewNumberFilter()
	if err := f.SetClause([]string{"not-a-number"}, false); err == nil {
		t.Errorf("SetClause with bad input: err=nil, want error")
	}
}
