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

func TestSetFilter_ClauseEmpty(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	if _, _, ok := f.Clause(); ok {
		t.Errorf("Clause() ok=true for fully-included filter")
	}
}

func TestSetFilter_ClauseSmallIncludeSet(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("a")
	f.Exclude("b")
	values, negate, ok := f.Clause()
	if !ok || negate || len(values) != 1 || values[0] != "c" {
		t.Errorf("Clause() = (%v, negate=%v, ok=%v), want ([c], false, true)", values, negate, ok)
	}
}

func TestSetFilter_ClauseSmallExcludeSet(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("a")
	values, negate, ok := f.Clause()
	if !ok || !negate || len(values) != 1 || values[0] != "a" {
		t.Errorf("Clause() = (%v, negate=%v, ok=%v), want ([a], true, true)", values, negate, ok)
	}
}

func TestSetFilter_SetClauseInclude(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	if err := f.SetClause([]string{"a", "b"}, false); err != nil {
		t.Fatalf("SetClause: %v", err)
	}
	if !f.Matches("a") || !f.Matches("b") || f.Matches("c") {
		t.Errorf("after SetClause([a,b], false): a=%v b=%v c=%v, want true,true,false",
			f.Matches("a"), f.Matches("b"), f.Matches("c"))
	}
}

func TestSetFilter_SetClauseExclude(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	if err := f.SetClause([]string{"a"}, true); err != nil {
		t.Fatalf("SetClause: %v", err)
	}
	if f.Matches("a") || !f.Matches("b") || !f.Matches("c") {
		t.Errorf("after SetClause([a], true): a=%v b=%v c=%v, want false,true,true",
			f.Matches("a"), f.Matches("b"), f.Matches("c"))
	}
}

func TestSetFilter_SetClauseRoundTrip(t *testing.T) {
	cases := []struct {
		all    []string
		values []string
		negate bool
	}{
		{[]string{"a", "b", "c"}, []string{"a"}, true},
		{[]string{"a", "b", "c"}, []string{"c"}, false},
		{[]string{"a", "b", "c", "d"}, []string{"a", "b"}, false},
	}
	for _, tc := range cases {
		f := NewSetFilter(tc.all...)
		if err := f.SetClause(tc.values, tc.negate); err != nil {
			t.Errorf("SetClause(%v, %v): %v", tc.values, tc.negate, err)
			continue
		}
		gotValues, gotNegate, ok := f.Clause()
		if !ok || gotNegate != tc.negate || !equalStringSetsRT(gotValues, tc.values) {
			t.Errorf("round trip %v negate=%v: got values=%v negate=%v ok=%v",
				tc.values, tc.negate, gotValues, gotNegate, ok)
		}
	}
}

func equalStringSetsRT(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]bool, len(a))
	for _, s := range a {
		m[s] = true
	}
	for _, s := range b {
		if !m[s] {
			return false
		}
	}
	return true
}

func TestBoolFilter_ClauseEmpty(t *testing.T) {
	f := NewBoolFilter()
	if _, _, ok := f.Clause(); ok {
		t.Errorf("Clause() ok=true for any-state filter")
	}
}

func TestBoolFilter_ClauseTrueOnly(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle()
	values, _, ok := f.Clause()
	if !ok || len(values) != 1 || values[0] != "true" {
		t.Errorf("Clause() = (%v, ok=%v), want ([true], true)", values, ok)
	}
}

func TestBoolFilter_ClauseFalseOnly(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle()
	f.Toggle()
	values, _, ok := f.Clause()
	if !ok || len(values) != 1 || values[0] != "false" {
		t.Errorf("Clause() = (%v, ok=%v), want ([false], true)", values, ok)
	}
}

func TestBoolFilter_SetClauseRoundTrip(t *testing.T) {
	for _, in := range []string{"true", "false", "1", "0"} {
		f := NewBoolFilter()
		if err := f.SetClause([]string{in}, false); err != nil {
			t.Errorf("%s: SetClause: %v", in, err)
			continue
		}
		if !f.Active() {
			t.Errorf("%s: Active()=false after SetClause", in)
		}
	}
}

func TestBoolFilter_SetClauseRejectsBad(t *testing.T) {
	f := NewBoolFilter()
	if err := f.SetClause([]string{"maybe"}, false); err == nil {
		t.Errorf("SetClause(maybe): err=nil, want error")
	}
}

func TestTimeFilter_ClauseEmpty(t *testing.T) {
	f := NewTimeFilter()
	if _, _, ok := f.Clause(); ok {
		t.Errorf("Clause() ok=true for empty filter")
	}
}

func TestTimeFilter_ClauseDate(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2026-04-01")
	values, _, ok := f.Clause()
	if !ok || values[0] != "2026-04-01" {
		t.Errorf("Clause() = (%v, ok=%v), want ([2026-04-01], true)", values, ok)
	}
}

func TestTimeFilter_ClauseRange(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2026-01-01..2026-12-31")
	values, _, ok := f.Clause()
	if !ok || values[0] != "2026-01-01..2026-12-31" {
		t.Errorf("Clause() values=%v ok=%v, want [2026-01-01..2026-12-31] true", values, ok)
	}
}

func TestTimeFilter_SetClauseRoundTrip(t *testing.T) {
	cases := []string{"2026-04-01", "2026-01-01..2026-12-31"}
	for _, expr := range cases {
		f := NewTimeFilter()
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

func TestTimeFilter_SetClauseRejectsBad(t *testing.T) {
	f := NewTimeFilter()
	if err := f.SetClause([]string{"not-a-date"}, false); err == nil {
		t.Errorf("SetClause(not-a-date): err=nil, want error")
	}
}
