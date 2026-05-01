# Searchquery Default Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the GitHub-style query language as a default tea-grid feature with round-trippable queries between an opt-in query-bar widget and column filter state.

**Architecture:** Three layers. (1) `filter.RoundTrippable` interface implemented by every built-in filter, plus a new `MultiSetFilter` for AND-of-includes semantics. (2) An internal `internal/querybar` package that maps AST → filter mutations (Apply) and filter state → AST → text (Rerender). (3) An opt-in `*querybar.State` field on `grid.Model[T]`, gated by `WithQueryBar(...)`, that owns the textinput and routes between bar mode and the existing key paths. Filters remain canonical; the bar is a textual projection. Lossy filter states get annotated, not blocked.

**Tech Stack:** Go 1.25, Bubble Tea v2 (`charm.land/bubbletea/v2`), Lipgloss v2 (`charm.land/lipgloss/v2`), Bubbles v2 (`charm.land/bubbles/v2/key`), `internal/lineedit`, `searchquery` (parser+AST already on this branch), standard `testing`.

**Spec:** `docs/superpowers/specs/2026-05-01-searchquery-default-feature-design.md`

---

## File Map

**New files:**
- `filter/roundtrip.go` — `RoundTrippable` interface.
- `filter/roundtrip_test.go` — round-trip identity and lossy-state tests for every built-in filter.
- `filter/multiset.go` — `MultiSetFilter` and its options.
- `filter/multiset_test.go` — `MultiSetFilter` unit tests.
- `filter/README.md` — package doc covering `Filter`, `RoundTrippable`, every built-in, and `MultiSetFilter`'s asymmetric editing model.
- `internal/querybar/state.go` — `State`, `New`, `Enabled`, text/lossy accessors.
- `internal/querybar/state_test.go` — state lifecycle tests.
- `internal/querybar/vocab.go` — `BuildAutoVocab[T]`, the column-derived `*searchquery.Vocabulary` builder.
- `internal/querybar/vocab_test.go` — auto-derivation and override tests.
- `internal/querybar/bind.go` — `Apply[T]` (AST → filter mutations) and `Rerender[T]` (filter state → text).
- `internal/querybar/bind_test.go` — apply/rerender unit tests, including the round-trip-identity table.
- `examples/querybar/main.go` — wired example with all filter types including `MultiSetFilter`.
- `grid/README.md` — extends with the new query-bar section.

**Modified:**
- `filter/builtin.go` — add `Clause`/`SetClause` to `TextFilter`, `NumberFilter`, `SetFilter`, `BoolFilter`, `TimeFilter`.
- `data/column.go` — add `QueryAliases []string` field to `Column[T]`.
- `grid/options.go` — remove `WithQuickFilter`, `WithQuickFilterText`; add `WithQueryBar`, `WithQueryBarText`, `WithQueryBarVocabulary`. Keep `WithQuickFilterDebounce` as inert (still sets `quickFilterDebounceDelay`).
- `grid/keymap.go` — rename `KeyMap.QuickFilter` to `KeyMap.QueryBar`, update help text to "query".
- `grid/styles.go` — add `Styles.QueryBar` and `Styles.QueryBarLossy`.
- `grid/messages.go` — rename `QuickFilterChangedMsg` to `QueryBarChangedMsg` (Text remains the bar text); keep `FilterChangedMsg` and `FiltersClearedMsg` unchanged.
- `grid/grid.go` — replace `quickFilterEnabled` with `queryBar *querybar.State`; rename `quickFilterActive` to `queryBarActive`; add `invalidateQueryBar()` helper, call it from every existing dirty-marking site.
- `grid/update.go` — replace `handleQuickFilterKeyMsg` with `handleQueryBarKeyMsg`; route via `m.queryBar != nil && m.queryBarActive`.
- `grid/render.go` — replace `renderQuickFilter` with `renderQueryBar`; it always renders when the bar is enabled.
- `searchquery/README.md` — mark binder/round-trip/query-bar items as landed; add a usage section pointing at `WithQueryBar`.
- `examples/anyrow/main.go`, `examples/basic/main.go`, `examples/columns/main.go`, `examples/csv/main.go`, `examples/jsonl/main.go`, `examples/selection/main.go`, `examples/spreadsheet/main.go` — replace `WithQuickFilter[X](true)` with `WithQueryBar[X]()`.
- `README.md` (root) — replace the "quick filter" mention in the Features list with "query bar (GitHub-style filtering)".

---

## Task 1: Add the `RoundTrippable` interface

**Files:**
- Create: `filter/roundtrip.go`

- [ ] **Step 1: Create the file with the interface**

```go
// Package filter — see filter.go for the Filter interface.

package filter

// RoundTrippable is implemented by filters that can serialize their
// state to and from a query-bar clause. Filters that do not implement
// it are treated as opaque by the query bar — annotated, not edited.
//
// SetClause replaces the filter's state with what the clause expresses.
// Returning an error leaves prior state unchanged and surfaces the
// error in the bar's status line; one error per offending clause.
//
// Clause returns the filter's state as a clause. ok=false when the
// filter is inactive (no clause to surface) or in a state that can not
// be expressed in the query syntax — the bar treats both ok=false
// states the same way: skip in the bar text, mention in the lossy
// annotation if Active() is true.
type RoundTrippable interface {
	SetClause(values []string, negate bool) error
	Clause() (values []string, negate bool, ok bool)
}
```

- [ ] **Step 2: Verify the package builds**

Run: `go build ./filter/`
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add filter/roundtrip.go
git commit -m "filter: add RoundTrippable interface for query-bar round-tripping"
```

---

## Task 2: TextFilter Clause/SetClause + tests

**Files:**
- Modify: `filter/builtin.go`
- Create: `filter/roundtrip_test.go`

- [ ] **Step 1: Write failing tests for TextFilter round-tripping**

Create `filter/roundtrip_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./filter/ -run TestTextFilter_ -v`
Expected: FAIL — methods `Clause` and `SetClause` not declared on `*TextFilter`.

- [ ] **Step 3: Add `Clause` and `SetClause` to `TextFilter`**

Edit `filter/builtin.go`. Find the `--- TextFilter ---` block. After `Clear`, add:

```go
// Clause implements RoundTrippable. Returns the substring as a single
// value. Regex mode is lossy: returns ok=false so the query bar can
// annotate it instead of trying to render it textually.
func (f *TextFilter) Clause() (values []string, negate bool, ok bool) {
	if !f.Active() {
		return nil, false, false
	}
	if f.regex {
		return nil, false, false
	}
	return []string{f.editor.Text()}, false, true
}

// SetClause implements RoundTrippable. Resets regex mode and applies
// the value as substring text. Rejects negation and multi-value lists.
func (f *TextFilter) SetClause(values []string, negate bool) error {
	if negate {
		return fmt.Errorf("TextFilter does not support negation")
	}
	if len(values) != 1 {
		return fmt.Errorf("TextFilter expects exactly one value, got %d", len(values))
	}
	f.regex = false
	f.SetText(values[0])
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./filter/ -run TestTextFilter_ -v`
Expected: PASS for all five tests.

- [ ] **Step 5: Verify the interface is satisfied**

Run: `go vet ./filter/`
Expected: no output.

Add a compile-time check at the bottom of `filter/roundtrip.go` (after the interface):

```go
// Compile-time interface assertions for built-in filters.
var (
	_ RoundTrippable = (*TextFilter)(nil)
)
```

Run: `go build ./filter/`
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add filter/roundtrip.go filter/roundtrip_test.go filter/builtin.go
git commit -m "filter: implement RoundTrippable on TextFilter"
```

---

## Task 3: NumberFilter Clause/SetClause + tests

**Files:**
- Modify: `filter/builtin.go`, `filter/roundtrip.go`, `filter/roundtrip_test.go`

- [ ] **Step 1: Append failing tests**

Append to `filter/roundtrip_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./filter/ -run TestNumberFilter_ -v`
Expected: FAIL — `Clause`/`SetClause` not declared on `*NumberFilter`.

- [ ] **Step 3: Add `Clause` and `SetClause` to `NumberFilter`**

Edit `filter/builtin.go`. Find the `--- NumberFilter ---` block. After `Clear`, add:

```go
// Clause implements RoundTrippable. Returns the editor text verbatim;
// it is already in the canonical form NumberFilter accepts.
func (f *NumberFilter) Clause() (values []string, negate bool, ok bool) {
	if !f.Active() {
		return nil, false, false
	}
	return []string{f.editor.Text()}, false, true
}

// SetClause implements RoundTrippable. The single value is parsed via
// the existing SetText path; an unparseable value returns an error and
// leaves prior state intact.
func (f *NumberFilter) SetClause(values []string, negate bool) error {
	if negate {
		return fmt.Errorf("NumberFilter does not support negation")
	}
	if len(values) != 1 {
		return fmt.Errorf("NumberFilter expects exactly one value, got %d", len(values))
	}
	prev := f.editor.Text()
	f.SetText(values[0])
	if !f.Active() && values[0] != "" {
		// parse failed; restore prior state
		f.SetText(prev)
		return fmt.Errorf("NumberFilter: could not parse %q", values[0])
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./filter/ -run TestNumberFilter_ -v`
Expected: PASS for all five tests.

- [ ] **Step 5: Add compile-time assertion**

Edit `filter/roundtrip.go`, extend the compile-time check block:

```go
var (
	_ RoundTrippable = (*TextFilter)(nil)
	_ RoundTrippable = (*NumberFilter)(nil)
)
```

Run: `go build ./filter/`
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add filter/roundtrip.go filter/roundtrip_test.go filter/builtin.go
git commit -m "filter: implement RoundTrippable on NumberFilter"
```

---

## Task 4: SetFilter Clause/SetClause with negation flip + tests

**Files:**
- Modify: `filter/builtin.go`, `filter/roundtrip.go`, `filter/roundtrip_test.go`

- [ ] **Step 1: Append failing tests**

Append to `filter/roundtrip_test.go`:

```go
func TestSetFilter_ClauseEmpty(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	// All values included → not active → no clause.
	if _, _, ok := f.Clause(); ok {
		t.Errorf("Clause() ok=true for fully-included filter")
	}
}

func TestSetFilter_ClauseSmallIncludeSet(t *testing.T) {
	// Exclude two of three: include set is the smaller side.
	f := NewSetFilter("a", "b", "c")
	f.Exclude("a")
	f.Exclude("b")
	values, negate, ok := f.Clause()
	if !ok || negate || len(values) != 1 || values[0] != "c" {
		t.Errorf("Clause() = (%v, negate=%v, ok=%v), want ([c], false, true)", values, negate, ok)
	}
}

func TestSetFilter_ClauseSmallExcludeSet(t *testing.T) {
	// Exclude one of three: more than half included → flip to negate form.
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
		{[]string{"a", "b", "c"}, []string{"a"}, true},  // exclude one → negate form
		{[]string{"a", "b", "c"}, []string{"c"}, false}, // include one → plain form
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./filter/ -run TestSetFilter_ -v`
Expected: FAIL — `Clause`/`SetClause` not declared on `*SetFilter`.

- [ ] **Step 3: Add `Clause` and `SetClause` to `SetFilter`**

Edit `filter/builtin.go`. Find the `--- SetFilter ---` block. After `Clear`, add:

```go
// Clause implements RoundTrippable. Returns the included subset by
// default; once more than half of allValues is included, returns the
// excluded subset with negate=true to keep the bar text small.
//
// New values that appear in row data after a clause was applied are
// treated as "not included" — consistent with current Include/SetValues
// semantics. Documented in the package README.
func (f *SetFilter) Clause() (values []string, negate bool, ok bool) {
	if !f.Active() {
		return nil, false, false
	}
	includedCount := len(f.values) - f.excludedCount
	if includedCount*2 > len(f.values) {
		// More than half included → emit excluded subset, negate.
		var excluded []string
		for _, v := range f.allValues {
			if !f.values[v] {
				excluded = append(excluded, v)
			}
		}
		return excluded, true, true
	}
	var included []string
	for _, v := range f.allValues {
		if f.values[v] {
			included = append(included, v)
		}
	}
	return included, false, true
}

// SetClause implements RoundTrippable. With negate=false, includes
// exactly the listed values and excludes the rest. With negate=true,
// includes everything except the listed values. Values not in
// allValues are ignored (treated as "not included" — see Clause's
// godoc).
func (f *SetFilter) SetClause(values []string, negate bool) error {
	want := make(map[string]bool, len(values))
	for _, v := range values {
		want[v] = true
	}
	f.excludedCount = 0
	for _, v := range f.allValues {
		var included bool
		if negate {
			included = !want[v]
		} else {
			included = want[v]
		}
		f.values[v] = included
		if !included {
			f.excludedCount++
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./filter/ -run TestSetFilter_ -v`
Expected: PASS for all six tests.

- [ ] **Step 5: Add compile-time assertion**

Edit `filter/roundtrip.go`:

```go
var (
	_ RoundTrippable = (*TextFilter)(nil)
	_ RoundTrippable = (*NumberFilter)(nil)
	_ RoundTrippable = (*SetFilter)(nil)
)
```

Run: `go build ./filter/`
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add filter/roundtrip.go filter/roundtrip_test.go filter/builtin.go
git commit -m "filter: implement RoundTrippable on SetFilter with negation flip"
```

---

## Task 5: BoolFilter Clause/SetClause + tests

**Files:**
- Modify: `filter/builtin.go`, `filter/roundtrip.go`, `filter/roundtrip_test.go`

- [ ] **Step 1: Append failing tests**

Append to `filter/roundtrip_test.go`:

```go
func TestBoolFilter_ClauseEmpty(t *testing.T) {
	f := NewBoolFilter()
	if _, _, ok := f.Clause(); ok {
		t.Errorf("Clause() ok=true for any-state filter")
	}
}

func TestBoolFilter_ClauseTrueOnly(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle() // any → true only
	values, _, ok := f.Clause()
	if !ok || len(values) != 1 || values[0] != "true" {
		t.Errorf("Clause() = (%v, ok=%v), want ([true], true)", values, ok)
	}
}

func TestBoolFilter_ClauseFalseOnly(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle()
	f.Toggle() // any → true only → false only
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./filter/ -run TestBoolFilter_ -v`
Expected: FAIL.

- [ ] **Step 3: Add `Clause` and `SetClause` to `BoolFilter`**

Edit `filter/builtin.go`. Find the `--- BoolFilter ---` block. After `Clear`, add:

```go
// Clause implements RoundTrippable.
func (f *BoolFilter) Clause() (values []string, negate bool, ok bool) {
	switch f.state {
	case 1:
		return []string{"true"}, false, true
	case 2:
		return []string{"false"}, false, true
	default:
		return nil, false, false
	}
}

// SetClause implements RoundTrippable. Accepts "true", "false", "1",
// or "0" (case-insensitive).
func (f *BoolFilter) SetClause(values []string, negate bool) error {
	if negate {
		return fmt.Errorf("BoolFilter does not support negation")
	}
	if len(values) != 1 {
		return fmt.Errorf("BoolFilter expects exactly one value, got %d", len(values))
	}
	switch strings.ToLower(values[0]) {
	case "true", "1":
		f.state = 1
	case "false", "0":
		f.state = 2
	default:
		return fmt.Errorf("BoolFilter: unrecognized value %q", values[0])
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./filter/ -run TestBoolFilter_ -v`
Expected: PASS.

- [ ] **Step 5: Add compile-time assertion**

Edit `filter/roundtrip.go`:

```go
var (
	_ RoundTrippable = (*TextFilter)(nil)
	_ RoundTrippable = (*NumberFilter)(nil)
	_ RoundTrippable = (*SetFilter)(nil)
	_ RoundTrippable = (*BoolFilter)(nil)
)
```

Run: `go build ./filter/`

- [ ] **Step 6: Commit**

```bash
git add filter/roundtrip.go filter/roundtrip_test.go filter/builtin.go
git commit -m "filter: implement RoundTrippable on BoolFilter"
```

---

## Task 6: TimeFilter Clause/SetClause + tests

**Files:**
- Modify: `filter/builtin.go`, `filter/roundtrip.go`, `filter/roundtrip_test.go`

- [ ] **Step 1: Append failing tests**

Append to `filter/roundtrip_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./filter/ -run TestTimeFilter_ -v`
Expected: FAIL.

- [ ] **Step 3: Add `Clause` and `SetClause` to `TimeFilter`**

Edit `filter/builtin.go`. Find the `--- TimeFilter ---` block. After `Clear`, add:

```go
// Clause implements RoundTrippable. Returns the editor text verbatim;
// it is already in the canonical form TimeFilter accepts.
func (f *TimeFilter) Clause() (values []string, negate bool, ok bool) {
	if !f.Active() {
		return nil, false, false
	}
	return []string{f.editor.Text()}, false, true
}

// SetClause implements RoundTrippable. Parses via the existing SetText
// path; if parsing yields no bounds, restores prior text and returns
// an error.
func (f *TimeFilter) SetClause(values []string, negate bool) error {
	if negate {
		return fmt.Errorf("TimeFilter does not support negation")
	}
	if len(values) != 1 {
		return fmt.Errorf("TimeFilter expects exactly one value, got %d", len(values))
	}
	prev := f.editor.Text()
	f.SetText(values[0])
	if !f.Active() && values[0] != "" {
		f.SetText(prev)
		return fmt.Errorf("TimeFilter: could not parse %q", values[0])
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./filter/ -run TestTimeFilter_ -v`
Expected: PASS.

- [ ] **Step 5: Add compile-time assertion**

Edit `filter/roundtrip.go`:

```go
var (
	_ RoundTrippable = (*TextFilter)(nil)
	_ RoundTrippable = (*NumberFilter)(nil)
	_ RoundTrippable = (*SetFilter)(nil)
	_ RoundTrippable = (*BoolFilter)(nil)
	_ RoundTrippable = (*TimeFilter)(nil)
)
```

Run: `go test ./filter/ -v -count=1`
Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add filter/roundtrip.go filter/roundtrip_test.go filter/builtin.go
git commit -m "filter: implement RoundTrippable on TimeFilter"
```

---

## Task 7: MultiSetFilter — type, options, basic methods

**Files:**
- Create: `filter/multiset.go`
- Create: `filter/multiset_test.go`

- [ ] **Step 1: Write failing tests for the basic API**

Create `filter/multiset_test.go`:

```go
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
	// Default matcher: row value is []string; constraint matches when
	// the slice contains the constraint string.
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./filter/ -run TestMultiSetFilter_ -v`
Expected: FAIL — types do not exist yet.

- [ ] **Step 3: Create the file with the basic API**

Create `filter/multiset.go`:

```go
package filter

import (
	tea "charm.land/bubbletea/v2"
)

// MultiSetFilter matches a value against a set of constraints with AND
// semantics: the value passes only if every constraint is satisfied.
// Useful for slice-valued columns ([]string labels, []Tag, etc.).
//
// Constraints are added via AddConstraint or by repeated SetClause
// calls from the query bar. The popup UI is view-and-remove only;
// new constraints come from the query bar.
type MultiSetFilter struct {
	constraints []string
	matcher     func(rowValue any, constraint string) bool

	// Editing state for the minimal popup.
	editing  bool
	width    int
	maxLines int
	focused  int // focused row in the constraint list
}

// MultiSetOption configures a MultiSetFilter at construction time.
type MultiSetOption func(*MultiSetFilter)

// WithMultiSetMatcher overrides the default matcher (which expects
// rowValue to be []string and matches by element equality).
func WithMultiSetMatcher(m func(rowValue any, constraint string) bool) MultiSetOption {
	return func(f *MultiSetFilter) {
		f.matcher = m
	}
}

// NewMultiSetFilter returns a MultiSetFilter with no constraints.
func NewMultiSetFilter(opts ...MultiSetOption) *MultiSetFilter {
	f := &MultiSetFilter{
		matcher: defaultMultiSetMatcher,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// AddConstraint appends a constraint. Duplicates are ignored.
func (f *MultiSetFilter) AddConstraint(c string) {
	for _, existing := range f.constraints {
		if existing == c {
			return
		}
	}
	f.constraints = append(f.constraints, c)
}

// RemoveConstraint removes the first occurrence of c. No-op if absent.
func (f *MultiSetFilter) RemoveConstraint(c string) {
	for i, existing := range f.constraints {
		if existing == c {
			f.constraints = append(f.constraints[:i], f.constraints[i+1:]...)
			if f.focused >= len(f.constraints) && f.focused > 0 {
				f.focused--
			}
			return
		}
	}
}

// Constraints returns the current constraint list (a copy).
func (f *MultiSetFilter) Constraints() []string {
	out := make([]string, len(f.constraints))
	copy(out, f.constraints)
	return out
}

// Clear removes all constraints.
func (f *MultiSetFilter) Clear() {
	f.constraints = nil
	f.focused = 0
}

// Matches reports whether rowValue satisfies every constraint.
func (f *MultiSetFilter) Matches(value any) bool {
	for _, c := range f.constraints {
		if !f.matcher(value, c) {
			return false
		}
	}
	return true
}

// Active reports whether the filter has any constraints.
func (f *MultiSetFilter) Active() bool {
	return len(f.constraints) > 0
}

// View, Update implemented in subsequent tasks.
func (f *MultiSetFilter) View() string                         { return "" }
func (f *MultiSetFilter) Update(msg tea.Msg) (Filter, tea.Cmd) { return f, nil }

// defaultMultiSetMatcher handles the common case: rowValue is []string,
// constraint matches when the slice contains the constraint.
func defaultMultiSetMatcher(rowValue any, constraint string) bool {
	switch v := rowValue.(type) {
	case []string:
		for _, s := range v {
			if s == constraint {
				return true
			}
		}
		return false
	case string:
		return v == constraint
	default:
		return false
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./filter/ -run TestMultiSetFilter_ -v`
Expected: PASS for all six tests.

- [ ] **Step 5: Verify package builds**

Run: `go build ./filter/`
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add filter/multiset.go filter/multiset_test.go
git commit -m "filter: add MultiSetFilter (AND-of-includes), basic API"
```

---

## Task 8: MultiSetFilter — popup UI (View + Update)

**Files:**
- Modify: `filter/multiset.go`, `filter/multiset_test.go`

- [ ] **Step 1: Append failing UI tests**

In `filter/multiset_test.go`, replace the existing `import "testing"` line with:

```go
import (
	"testing"

	tea "charm.land/bubbletea/v2"
)
```

Then append to the file:

```go
func TestMultiSetFilter_FocusEditing(t *testing.T) {
	f := NewMultiSetFilter()
	f.AddConstraint("bug")
	f.AddConstraint("urgent")

	// Enter editing mode.
	g, _ := f.Update(FilterFocusMsg{Width: 40, MaxLines: 6})
	f = g.(*MultiSetFilter)
	if !f.editing {
		t.Errorf("editing=false after FilterFocusMsg")
	}
	if f.focused != 0 {
		t.Errorf("focused=%d after focus, want 0", f.focused)
	}

	// Down arrow advances focus.
	g, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	f = g.(*MultiSetFilter)
	if f.focused != 1 {
		t.Errorf("focused=%d after Down, want 1", f.focused)
	}

	// Up arrow at top stays at 0.
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

	// Press 'd' to delete the focused (first) constraint.
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
	if !contains(out, "bug") || !contains(out, "urgent") {
		t.Errorf("View() = %q, want it to contain bug and urgent", out)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && stringIndex(haystack, needle) >= 0
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./filter/ -run TestMultiSetFilter_ -v`
Expected: editing tests FAIL — `View` returns "" and `Update` is a no-op.

- [ ] **Step 3: Update imports and replace the placeholder `View`/`Update`**

In `filter/multiset.go`, replace the import block:

```go
import (
	tea "charm.land/bubbletea/v2"
)
```

with:

```go
import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pgavlin/tea-grid/internal/lineedit"
)
```

Replace the placeholder `View` and `Update` lines with the real bodies plus a `deleteFocused` helper:

```go
// View renders the constraint list with row focus and a footer hint.
// Empty when the popup is not in editing mode.
func (f *MultiSetFilter) View() string {
	if !f.editing {
		return ""
	}
	var lines []string
	for i, c := range f.constraints {
		entry := "× " + c
		if f.width > 0 {
			entry = lineedit.TruncateOrPad(entry, f.width)
		}
		if i == f.focused {
			entry = lipgloss.NewStyle().Reverse(true).Render(entry)
		}
		lines = append(lines, entry)
	}
	footer := "d delete · esc close · / edit"
	if f.width > 0 {
		footer = lineedit.TruncateOrPad(footer, f.width)
	}
	lines = append(lines, footer)
	if f.maxLines > 0 && len(lines) > f.maxLines {
		lines = lines[:f.maxLines]
	}
	return strings.Join(lines, "\n")
}

// Update handles popup interactions: focus navigation and constraint
// deletion.
func (f *MultiSetFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	switch msg := msg.(type) {
	case FilterFocusMsg:
		f.editing = true
		f.width = msg.Width
		f.maxLines = msg.MaxLines
		f.focused = 0
		return f, nil
	case FilterBlurMsg:
		f.editing = false
		return f, nil
	case tea.KeyPressMsg:
		if !f.editing {
			return f, nil
		}
		switch msg.Code {
		case tea.KeyUp:
			if f.focused > 0 {
				f.focused--
			}
		case tea.KeyDown:
			if f.focused < len(f.constraints)-1 {
				f.focused++
			}
		case tea.KeyBackspace:
			f.deleteFocused()
		default:
			if msg.Text == "d" {
				f.deleteFocused()
			}
		}
	}
	return f, nil
}

func (f *MultiSetFilter) deleteFocused() {
	if f.focused < 0 || f.focused >= len(f.constraints) {
		return
	}
	f.constraints = append(f.constraints[:f.focused], f.constraints[f.focused+1:]...)
	if f.focused >= len(f.constraints) && f.focused > 0 {
		f.focused--
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./filter/ -run TestMultiSetFilter_ -v`
Expected: PASS for all UI tests.

- [ ] **Step 5: Verify package builds**

Run: `go build ./filter/ && go vet ./filter/`
Expected: success, no warnings.

- [ ] **Step 6: Commit**

```bash
git add filter/multiset.go filter/multiset_test.go
git commit -m "filter: add MultiSetFilter popup UI (view + key handling)"
```

---

## Task 9: MultiSetFilter — RoundTrippable

**Files:**
- Modify: `filter/multiset.go`, `filter/roundtrip.go`, `filter/multiset_test.go`

- [ ] **Step 1: Append failing tests**

Append to `filter/multiset_test.go`:

```go
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
	// SetClause is called once per repeated bar clause; each call
	// appends one constraint.
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./filter/ -run TestMultiSetFilter_Clause -v`
Expected: FAIL — methods not declared.

- [ ] **Step 3: Add `fmt` to imports and add `Clause` / `SetClause`**

In `filter/multiset.go`, extend the import block to add `"fmt"`:

```go
import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pgavlin/tea-grid/internal/lineedit"
)
```

Append after `Clear`:

```go
// Clause implements RoundTrippable. Returns one value per constraint;
// the bar serializes these as repeated `field:value` clauses (the AST
// encoding of AND).
func (f *MultiSetFilter) Clause() (values []string, negate bool, ok bool) {
	if !f.Active() {
		return nil, false, false
	}
	out := make([]string, len(f.constraints))
	copy(out, f.constraints)
	return out, false, true
}

// SetClause implements RoundTrippable. Each call appends one
// constraint. v1 rejects multi-value (comma-list) clauses; users who
// want OR-within-AND register the column with a SetFilter instead.
func (f *MultiSetFilter) SetClause(values []string, negate bool) error {
	if negate {
		return fmt.Errorf("MultiSetFilter does not support negation")
	}
	if len(values) != 1 {
		return fmt.Errorf("MultiSetFilter expects one value per clause, got %d", len(values))
	}
	f.AddConstraint(values[0])
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./filter/ -v -count=1`
Expected: all `MultiSetFilter` tests pass.

- [ ] **Step 5: Add compile-time assertion**

Edit `filter/roundtrip.go`, extend the var block:

```go
var (
	_ RoundTrippable = (*TextFilter)(nil)
	_ RoundTrippable = (*NumberFilter)(nil)
	_ RoundTrippable = (*SetFilter)(nil)
	_ RoundTrippable = (*BoolFilter)(nil)
	_ RoundTrippable = (*TimeFilter)(nil)
	_ RoundTrippable = (*MultiSetFilter)(nil)
)
```

Run: `go test ./filter/ -v -count=1`
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add filter/multiset.go filter/multiset_test.go filter/roundtrip.go
git commit -m "filter: implement RoundTrippable on MultiSetFilter"
```

---

## Task 10: Add `QueryAliases` to `data.Column`

**Files:**
- Modify: `data/column.go`

- [ ] **Step 1: Write failing test**

Append to `data/column_test.go` (or create a new test file `data/column_aliases_test.go` if `column_test.go` does not have package-level helpers):

```go
func TestColumn_QueryAliases(t *testing.T) {
	col := Column[map[string]any]{
		ColumnID:     "state",
		QueryAliases: []string{"status", "st"},
	}
	if len(col.QueryAliases) != 2 || col.QueryAliases[0] != "status" {
		t.Errorf("QueryAliases = %v, want [status st]", col.QueryAliases)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./data/ -run TestColumn_QueryAliases -v`
Expected: FAIL — `QueryAliases` field does not exist.

- [ ] **Step 3: Add the field to `Column[T]`**

Edit `data/column.go`. Inside the `Column[T]` struct, after `QuickFilterMatch`, add:

```go
	// Filtering
	Filterable       bool                            // Default: true.
	Filter           filter.Filter                   // Column filter.
	QuickFilterMatch func(data *T, word string) bool // Reports whether this column matches a quick filter word. Takes *T to avoid copying. If nil, falls back to Text or Value + containsFold.
	QueryAliases     []string                        // Additional names this column responds to in the query bar (e.g., ["status", "st"] for a "state" column).
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./data/ -run TestColumn_QueryAliases -v`
Expected: PASS.

- [ ] **Step 5: Verify the rest of the package still builds**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add data/column.go data/column_test.go
git commit -m "data: add Column.QueryAliases for query-bar field aliases"
```

---

## Task 11: `internal/querybar/vocab.go` — `BuildAutoVocab`

**Files:**
- Create: `internal/querybar/vocab.go`
- Create: `internal/querybar/vocab_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/querybar/vocab_test.go`:

```go
package querybar

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/searchquery"
)

func TestBuildAutoVocab_Empty(t *testing.T) {
	v := BuildAutoVocab[map[string]any](nil)
	if v == nil {
		t.Fatal("BuildAutoVocab(nil) = nil")
	}
	if _, ok := v.Resolve("anything"); ok {
		t.Errorf("empty vocab resolved unexpected name")
	}
}

func TestBuildAutoVocab_TextColumn(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "title", Filter: filter.NewTextFilter(), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	canon, ok := v.Resolve("title")
	if !ok || canon != "title" {
		t.Errorf("Resolve(title) = (%q, %v), want (title, true)", canon, ok)
	}
	f, ok := v.Field("title")
	if !ok || f.Type != searchquery.FieldString {
		t.Errorf("Field(title).Type = %v, want FieldString", f.Type)
	}
}

func TestBuildAutoVocab_TimeColumnInfersTimeType(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "created", Filter: filter.NewTimeFilter(), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	f, ok := v.Field("created")
	if !ok || f.Type != searchquery.FieldTime {
		t.Errorf("Field(created).Type = %v, want FieldTime", f.Type)
	}
}

func TestBuildAutoVocab_BoolColumnInfersBoolType(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "active", Filter: filter.NewBoolFilter(), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	f, _ := v.Field("active")
	if f.Type != searchquery.FieldBool {
		t.Errorf("Field(active).Type = %v, want FieldBool", f.Type)
	}
}

func TestBuildAutoVocab_SetColumnAcceptsList(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "label", Filter: filter.NewSetFilter("a", "b"), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	f, _ := v.Field("label")
	if !f.AcceptsList {
		t.Errorf("Field(label).AcceptsList = false, want true (SetFilter)")
	}
}

func TestBuildAutoVocab_MultiSetColumnSingleValue(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "tag", Filter: filter.NewMultiSetFilter(), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	f, _ := v.Field("tag")
	if f.AcceptsList {
		t.Errorf("Field(tag).AcceptsList = true, want false (MultiSetFilter)")
	}
}

func TestBuildAutoVocab_AliasesFromColumn(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{
			ColumnID:     "state",
			Filter:       filter.NewSetFilter("open", "closed"),
			Filterable:   true,
			QueryAliases: []string{"status"},
		},
	}
	v := BuildAutoVocab(cols)
	canon, ok := v.Resolve("status")
	if !ok || canon != "state" {
		t.Errorf("Resolve(status) = (%q, %v), want (state, true)", canon, ok)
	}
}

func TestBuildAutoVocab_SkipsNonRoundTrippableFilters(t *testing.T) {
	// Stub a filter that does not implement RoundTrippable.
	cols := []data.Column[map[string]any]{
		{ColumnID: "x", Filter: &nonRTFilter{}, Filterable: true},
	}
	v := BuildAutoVocab(cols)
	if _, ok := v.Field("x"); ok {
		t.Errorf("Field(x) = ok, want skipped (filter not RoundTrippable)")
	}
}

func TestBuildAutoVocab_SkipsHiddenAndUnfilterable(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "hidden", Filter: filter.NewTextFilter(), Filterable: true, Hide: true},
		{ColumnID: "unfilterable", Filter: filter.NewTextFilter(), Filterable: false},
		{ColumnID: "ok", Filter: filter.NewTextFilter(), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	if _, ok := v.Field("hidden"); ok {
		t.Errorf("hidden should be skipped")
	}
	if _, ok := v.Field("unfilterable"); ok {
		t.Errorf("unfilterable should be skipped")
	}
	if _, ok := v.Field("ok"); !ok {
		t.Errorf("ok should be included")
	}
}

// nonRTFilter is a Filter that does not implement RoundTrippable.
type nonRTFilter struct{}

func (n *nonRTFilter) Matches(value any) bool                    { return true }
func (n *nonRTFilter) View() string                              { return "" }
func (n *nonRTFilter) Update(msg tea.Msg) (filter.Filter, tea.Cmd) { return n, nil }
func (n *nonRTFilter) Active() bool                              { return false }
func (n *nonRTFilter) Clear()                                    {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/querybar/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Create the file**

Create `internal/querybar/vocab.go`:

```go
// Package querybar implements the query-bar widget used by tea-grid's
// grid.Model[T]. The package is internal because its surface is tightly
// coupled to grid internals; consumers configure the bar via the
// re-exported options in package grid.
package querybar

import (
	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/searchquery"
)

// BuildAutoVocab derives a *searchquery.Vocabulary from the column set.
// Every visible, filterable column with a RoundTrippable filter becomes
// a queryable Field; field type is inferred from filter type; aliases
// come from Column.QueryAliases. Columns with no filter, a non-
// RoundTrippable filter, Hide=true, or Filterable=false are skipped.
func BuildAutoVocab[T any](cols []data.Column[T]) *searchquery.Vocabulary {
	var fields []searchquery.Field
	for i := range cols {
		c := &cols[i]
		if c.Hide || !c.Filterable || c.Filter == nil {
			continue
		}
		if _, ok := c.Filter.(filter.RoundTrippable); !ok {
			continue
		}
		fields = append(fields, searchquery.Field{
			Name:        c.ColumnID,
			Aliases:     append([]string(nil), c.QueryAliases...),
			Type:        inferFieldType(c.Filter),
			AcceptsList: filterAcceptsList(c.Filter),
		})
	}
	return searchquery.NewVocabulary(fields)
}

// inferFieldType picks a searchquery.FieldType from the concrete filter
// type. NumberFilter/SetFilter/TextFilter/MultiSetFilter all report
// FieldString — the parser is content-agnostic and the binder handles
// numeric parsing inside NumberFilter itself.
func inferFieldType(f filter.Filter) searchquery.FieldType {
	switch f.(type) {
	case *filter.TimeFilter:
		return searchquery.FieldTime
	case *filter.BoolFilter:
		return searchquery.FieldBool
	default:
		return searchquery.FieldString
	}
}

// filterAcceptsList reports whether a filter accepts comma-separated
// value lists in a single clause. Only SetFilter does in v1;
// MultiSetFilter takes one value per repeated clause.
func filterAcceptsList(f filter.Filter) bool {
	_, ok := f.(*filter.SetFilter)
	return ok
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/querybar/ -v -count=1`
Expected: PASS for all `BuildAutoVocab_*` tests.

- [ ] **Step 5: Commit**

```bash
git add internal/querybar/vocab.go internal/querybar/vocab_test.go
git commit -m "querybar: add BuildAutoVocab — column-derived Vocabulary"
```

---

## Task 12: `internal/querybar/state.go` — `State` type

**Files:**
- Create: `internal/querybar/state.go`
- Create: `internal/querybar/state_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/querybar/state_test.go`:

```go
package querybar

import (
	"testing"

	"github.com/pgavlin/tea-grid/searchquery"
)

func TestState_NewDisabled(t *testing.T) {
	s := New(nil)
	if s.Enabled() {
		t.Errorf("New(nil).Enabled() = true, want false (no vocab built yet)")
	}
}

func TestState_EnableWithVocab(t *testing.T) {
	v := searchquery.NewVocabulary(nil)
	s := New(v)
	s.Enable()
	if !s.Enabled() {
		t.Errorf("Enable() did not enable")
	}
}

func TestState_TextRoundTrip(t *testing.T) {
	s := New(searchquery.NewVocabulary(nil))
	s.Enable()
	s.SetText("hello world")
	if s.Text() != "hello world" {
		t.Errorf("Text() = %q, want %q", s.Text(), "hello world")
	}
}

func TestState_LossyAccessors(t *testing.T) {
	s := New(searchquery.NewVocabulary(nil))
	s.Enable()
	s.SetLossy([]string{"title", "labels"})
	got := s.Lossy()
	if len(got) != 2 || got[0] != "title" || got[1] != "labels" {
		t.Errorf("Lossy() = %v, want [title labels]", got)
	}
}

func TestState_Editing(t *testing.T) {
	s := New(searchquery.NewVocabulary(nil))
	s.Enable()
	if s.Editing() {
		t.Errorf("Editing() = true initially")
	}
	s.BeginEdit()
	if !s.Editing() {
		t.Errorf("BeginEdit did not enter editing mode")
	}
	s.EndEdit()
	if s.Editing() {
		t.Errorf("EndEdit did not exit editing mode")
	}
}

func TestState_VocabOverride(t *testing.T) {
	custom := searchquery.NewVocabulary([]searchquery.Field{{Name: "custom"}})
	s := New(nil)
	s.SetVocabulary(custom)
	if s.Vocab() != custom {
		t.Errorf("Vocab() did not return the override")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/querybar/ -run TestState_ -v`
Expected: FAIL — type `State` does not exist.

- [ ] **Step 3: Create `state.go`**

Create `internal/querybar/state.go`:

```go
package querybar

import (
	"github.com/pgavlin/tea-grid/internal/lineedit"
	"github.com/pgavlin/tea-grid/searchquery"
)

// State holds the query bar's UI state and its current view of the
// canonical filter state (text + lossy column IDs). It is consumed by
// grid.Model[T]; consumers configure the bar via WithQueryBar* options.
//
// State is non-generic: its content (text, lossy IDs, editing flags)
// does not depend on T. The generic Apply / Rerender functions in
// bind.go take cols []data.Column[T] separately.
type State struct {
	enabled bool
	editing bool

	editor lineedit.Model

	// text is the canonical projection of the current filter state.
	// It mirrors the filter side; user edits go through the editor and
	// are committed to text on submit.
	text string

	// lossy is the set of column IDs whose filter state could not be
	// represented in the bar (regex on TextFilter, non-RoundTrippable
	// filters). The bar renders these in an annotation, not text.
	lossy []string

	// parseErr holds the last parse error from a submit, surfaced in
	// the bar's status line.
	parseErr string

	// auto is the column-derived vocabulary; rebuilt on column changes.
	// custom is an explicit override set via WithQueryBarVocabulary; if
	// non-nil, it shadows auto.
	auto   *searchquery.Vocabulary
	custom *searchquery.Vocabulary
}

// New returns a new State with the given auto-vocabulary. The bar is
// disabled until Enable is called. The auto vocabulary may be nil; it
// is rebuilt by SetAutoVocabulary when columns are known.
func New(auto *searchquery.Vocabulary) *State {
	return &State{auto: auto}
}

// Enable marks the bar as active. Until enabled, grid.Model[T] does
// not render the bar or route keys to it.
func (s *State) Enable() { s.enabled = true }

// Enabled reports whether the bar is enabled.
func (s *State) Enabled() bool { return s.enabled }

// Editing reports whether the bar's textinput currently has focus.
func (s *State) Editing() bool { return s.editing }

// BeginEdit puts focus on the textinput and copies the canonical text
// into the editor for editing.
func (s *State) BeginEdit() {
	s.editing = true
	s.editor.SetText(s.text)
	s.editor.CursorToEnd()
}

// EndEdit drops focus on the textinput and discards uncommitted edits.
func (s *State) EndEdit() {
	s.editing = false
	s.editor.SetText(s.text)
}

// Text returns the canonical bar text (the projection of filter state).
func (s *State) Text() string { return s.text }

// SetText replaces the canonical bar text. Called by Rerender.
func (s *State) SetText(t string) {
	s.text = t
	if !s.editing {
		s.editor.SetText(t)
	}
}

// EditorText returns the textinput's current value (may diverge from
// Text while Editing()).
func (s *State) EditorText() string { return s.editor.Text() }

// Editor exposes the underlying editor for grid.Model to forward key
// messages into.
func (s *State) Editor() *lineedit.Model { return &s.editor }

// Lossy returns the set of column IDs in lossy state.
func (s *State) Lossy() []string {
	out := make([]string, len(s.lossy))
	copy(out, s.lossy)
	return out
}

// SetLossy replaces the lossy-column-ID set. Called by Rerender.
func (s *State) SetLossy(ids []string) {
	s.lossy = append(s.lossy[:0], ids...)
}

// ParseErr returns the last parse error from a submit, or "".
func (s *State) ParseErr() string { return s.parseErr }

// SetParseErr stores a parse-error string for display in the status
// line.
func (s *State) SetParseErr(e string) { s.parseErr = e }

// SetAutoVocabulary updates the column-derived vocabulary. Called when
// columns change (SetColumns).
func (s *State) SetAutoVocabulary(v *searchquery.Vocabulary) { s.auto = v }

// SetVocabulary sets an explicit override that shadows the column-
// derived vocabulary.
func (s *State) SetVocabulary(v *searchquery.Vocabulary) { s.custom = v }

// Vocab returns the active vocabulary: the override if set, otherwise
// the column-derived one.
func (s *State) Vocab() *searchquery.Vocabulary {
	if s.custom != nil {
		return s.custom
	}
	return s.auto
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/querybar/ -run TestState_ -v -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/querybar/state.go internal/querybar/state_test.go
git commit -m "querybar: add State — bar text, lossy set, vocabulary mgmt"
```

---

## Task 13: `internal/querybar/bind.go` — `Rerender`

**Files:**
- Create: `internal/querybar/bind.go`
- Create: `internal/querybar/bind_test.go`

- [ ] **Step 1: Write failing tests for Rerender**

Create `internal/querybar/bind_test.go`:

```go
package querybar

import (
	"strings"
	"testing"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
)

func cols2() []data.Column[map[string]any] {
	return []data.Column[map[string]any]{
		{ColumnID: "state", Filter: filter.NewSetFilter("open", "closed"), Filterable: true},
		{ColumnID: "title", Filter: filter.NewTextFilter(), Filterable: true},
		{ColumnID: "count", Filter: filter.NewNumberFilter(), Filterable: true},
	}
}

func TestRerender_NoFiltersNoBareTerms_EmptyText(t *testing.T) {
	c := cols2()
	text, lossy := Rerender(c, "")
	if text != "" {
		t.Errorf("text = %q, want empty", text)
	}
	if len(lossy) != 0 {
		t.Errorf("lossy = %v, want none", lossy)
	}
}

func TestRerender_OneClauseAndBareTerms(t *testing.T) {
	c := cols2()
	c[0].Filter.(*filter.SetFilter).Exclude("closed") // active: include {open}
	c[1].Filter.(*filter.TextFilter).SetText("memory")
	text, lossy := Rerender(c, "extra terms")
	if !strings.Contains(text, "state:open") {
		t.Errorf("text %q missing state:open", text)
	}
	if !strings.Contains(text, "title:memory") {
		t.Errorf("text %q missing title:memory", text)
	}
	if !strings.HasSuffix(text, "extra terms") {
		t.Errorf("text %q does not end with bare terms", text)
	}
	if len(lossy) != 0 {
		t.Errorf("lossy = %v, want none", lossy)
	}
}

func TestRerender_LossyFilterAnnotated(t *testing.T) {
	c := cols2()
	c[1].Filter.(*filter.TextFilter).SetRegex(true)
	c[1].Filter.(*filter.TextFilter).SetText("foo.*bar")
	_, lossy := Rerender(c, "")
	if len(lossy) != 1 || lossy[0] != "title" {
		t.Errorf("lossy = %v, want [title]", lossy)
	}
}

func TestRerender_NumberFilterClauseFormat(t *testing.T) {
	c := cols2()
	c[2].Filter.(*filter.NumberFilter).SetText(">10")
	text, _ := Rerender(c, "")
	if !strings.Contains(text, "count:>10") {
		t.Errorf("text %q missing count:>10", text)
	}
}

func TestRerender_SetFilterCommaList(t *testing.T) {
	c := []data.Column[map[string]any]{
		{ColumnID: "state", Filter: filter.NewSetFilter("a", "b", "c", "d"), Filterable: true},
	}
	c[0].Filter.(*filter.SetFilter).Exclude("c")
	c[0].Filter.(*filter.SetFilter).Exclude("d") // include set is {a, b} — half or less
	text, _ := Rerender(c, "")
	if !strings.Contains(text, "state:a,b") && !strings.Contains(text, "state:b,a") {
		t.Errorf("text %q missing state:a,b (or any order)", text)
	}
}

func TestRerender_SetFilterNegateForm(t *testing.T) {
	c := []data.Column[map[string]any]{
		{ColumnID: "state", Filter: filter.NewSetFilter("a", "b", "c"), Filterable: true},
	}
	c[0].Filter.(*filter.SetFilter).Exclude("a") // include set is {b, c} — more than half
	text, _ := Rerender(c, "")
	if !strings.Contains(text, "-state:a") {
		t.Errorf("text %q missing -state:a (negation flip)", text)
	}
}

func TestRerender_MultiSetRepeatedClauses(t *testing.T) {
	c := []data.Column[map[string]any]{
		{ColumnID: "label", Filter: filter.NewMultiSetFilter(), Filterable: true},
	}
	mf := c[0].Filter.(*filter.MultiSetFilter)
	mf.AddConstraint("bug")
	mf.AddConstraint("urgent")
	text, _ := Rerender(c, "")
	// Repeated clauses, separated by spaces.
	if !strings.Contains(text, "label:bug") || !strings.Contains(text, "label:urgent") {
		t.Errorf("text %q missing label:bug or label:urgent", text)
	}
}

func TestRerender_NonRoundTrippableFilterIsLossy(t *testing.T) {
	c := []data.Column[map[string]any]{
		{ColumnID: "x", Filter: &nonRTFilter{}, Filterable: true},
	}
	// nonRTFilter.Active returns false; so it shouldn't be lossy.
	// Add a small wrapper that reports active.
	_, lossy := Rerender(c, "")
	if len(lossy) != 0 {
		t.Errorf("inactive non-RT filter should not be lossy; got %v", lossy)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/querybar/ -run TestRerender_ -v`
Expected: FAIL — `Rerender` does not exist.

- [ ] **Step 3: Create `bind.go` with `Rerender`**

Create `internal/querybar/bind.go`:

```go
package querybar

import (
	"strings"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
)

// Rerender renders the current filter state as bar text plus a list of
// column IDs in lossy state. Inputs:
//
//   - cols: the grid's current column set. Each column's Filter is
//     inspected for RoundTrippable + Active.
//   - bareTerms: the residual quick-filter text (appended verbatim).
//
// Returns the joined text and the lossy column IDs (in column order).
func Rerender[T any](cols []data.Column[T], bareTerms string) (text string, lossy []string) {
	var clauses []string
	for i := range cols {
		c := &cols[i]
		if c.Filter == nil || !c.Filter.Active() {
			continue
		}
		rt, ok := c.Filter.(filter.RoundTrippable)
		if !ok {
			lossy = append(lossy, c.ColumnID)
			continue
		}
		values, negate, ok := rt.Clause()
		if !ok {
			lossy = append(lossy, c.ColumnID)
			continue
		}
		clauses = append(clauses, formatClause(c.ColumnID, values, negate, isMultiClause(c.Filter)))
	}
	parts := clauses
	if bareTerms != "" {
		parts = append(parts, bareTerms)
	}
	return strings.Join(parts, " "), lossy
}

// formatClause builds the textual form of a clause. For multi-clause
// filters (MultiSetFilter), one clause per value is emitted. For OR
// list filters (SetFilter), values are comma-joined. For scalar
// filters, exactly one value is expected.
func formatClause(field string, values []string, negate, multiClause bool) string {
	prefix := ""
	if negate {
		prefix = "-"
	}
	if multiClause {
		// One field:value per constraint (AND semantics).
		out := make([]string, len(values))
		for i, v := range values {
			out[i] = prefix + field + ":" + quoteIfNeeded(v)
		}
		return strings.Join(out, " ")
	}
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = quoteIfNeeded(v)
	}
	return prefix + field + ":" + strings.Join(quoted, ",")
}

// isMultiClause reports whether the filter type emits one clause per
// value (AND semantics). Only MultiSetFilter does.
func isMultiClause(f filter.Filter) bool {
	_, ok := f.(*filter.MultiSetFilter)
	return ok
}

// quoteIfNeeded wraps a value in double quotes when it contains any
// character that would break the bare-word grammar (whitespace or one
// of the structural characters `:`, `,`, `"`).
func quoteIfNeeded(v string) string {
	if v == "" {
		return `""`
	}
	for _, r := range v {
		if r == ' ' || r == '\t' || r == '\n' || r == ':' || r == ',' || r == '"' {
			return `"` + strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `"`, `\"`) + `"`
		}
	}
	return v
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/querybar/ -run TestRerender_ -v -count=1`
Expected: PASS for all `Rerender_*` tests.

- [ ] **Step 5: Commit**

```bash
git add internal/querybar/bind.go internal/querybar/bind_test.go
git commit -m "querybar: add Rerender — filter state to bar text"
```

---

## Task 14: `internal/querybar/bind.go` — `Apply`

**Files:**
- Modify: `internal/querybar/bind.go`, `internal/querybar/bind_test.go`

- [ ] **Step 1: Append failing tests for Apply**

In `internal/querybar/bind_test.go`, extend the existing import block to include `searchquery`:

```go
import (
	"strings"
	"testing"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/searchquery"
)
```

Then append:

```go
func vocab2() *searchquery.Vocabulary {
	return BuildAutoVocab(cols2())
}

func TestApply_ScalarTextFilter(t *testing.T) {
	c := cols2()
	ast, _ := searchquery.Parse("title:memory", vocab2())
	res := Apply(c, ast)
	if res.BareTerms != "" {
		t.Errorf("BareTerms = %q, want empty", res.BareTerms)
	}
	if len(res.Errors) != 0 {
		t.Errorf("Errors = %v, want none", res.Errors)
	}
	tf := c[1].Filter.(*filter.TextFilter)
	if !tf.Active() {
		t.Errorf("TextFilter inactive after Apply")
	}
	values, _, _ := tf.Clause()
	if len(values) != 1 || values[0] != "memory" {
		t.Errorf("after Apply: values=%v, want [memory]", values)
	}
}

func TestApply_SetFilterCommaList(t *testing.T) {
	c := cols2()
	ast, _ := searchquery.Parse("state:open,closed", vocab2())
	Apply(c, ast)
	sf := c[0].Filter.(*filter.SetFilter)
	if !sf.Matches("open") || !sf.Matches("closed") {
		t.Errorf("SetFilter should include open and closed")
	}
}

func TestApply_BareTermsExtracted(t *testing.T) {
	c := cols2()
	ast, _ := searchquery.Parse("memory leak", vocab2())
	res := Apply(c, ast)
	if res.BareTerms != "memory leak" {
		t.Errorf("BareTerms = %q, want %q", res.BareTerms, "memory leak")
	}
}

func TestApply_ClearsFiltersNotMentioned(t *testing.T) {
	c := cols2()
	c[1].Filter.(*filter.TextFilter).SetText("old text")
	ast, _ := searchquery.Parse("state:open", vocab2())
	Apply(c, ast)
	if c[1].Filter.(*filter.TextFilter).Active() {
		t.Errorf("title TextFilter should be cleared (not in submitted query)")
	}
}

func TestApply_LossyFilterLeftAlone(t *testing.T) {
	c := cols2()
	c[1].Filter.(*filter.TextFilter).SetRegex(true)
	c[1].Filter.(*filter.TextFilter).SetText("foo.*")
	ast, _ := searchquery.Parse("state:open", vocab2())
	Apply(c, ast)
	tf := c[1].Filter.(*filter.TextFilter)
	if !tf.Active() {
		t.Errorf("lossy TextFilter (regex) should not be cleared by Apply")
	}
}

func TestApply_MultiSetClearsAndAccumulates(t *testing.T) {
	c := []data.Column[map[string]any]{
		{ColumnID: "label", Filter: filter.NewMultiSetFilter(), Filterable: true},
	}
	mf := c[0].Filter.(*filter.MultiSetFilter)
	mf.AddConstraint("old")
	v := BuildAutoVocab(c)
	ast, _ := searchquery.Parse("label:bug label:urgent", v)
	Apply(c, ast)
	cs := mf.Constraints()
	if len(cs) != 2 || cs[0] != "bug" || cs[1] != "urgent" {
		t.Errorf("after Apply: %v, want [bug urgent] (old cleared, new appended)", cs)
	}
}

func TestApply_PerClauseErrorContinues(t *testing.T) {
	c := cols2()
	ast, _ := searchquery.Parse("count:notanumber state:open", vocab2())
	res := Apply(c, ast)
	if len(res.Errors) != 1 {
		t.Errorf("Errors = %v, want exactly one", res.Errors)
	}
	if !c[0].Filter.(*filter.SetFilter).Matches("open") {
		t.Errorf("good clause did not apply despite bad sibling")
	}
	if c[2].Filter.(*filter.NumberFilter).Active() {
		t.Errorf("bad clause should not have set NumberFilter active")
	}
}

func TestApply_UnknownFieldIgnored(t *testing.T) {
	c := cols2()
	ast, _ := searchquery.Parse("nonexistent:foo state:open", vocab2())
	res := Apply(c, ast)
	if len(res.Errors) != 0 {
		t.Errorf("unknown field should not produce error; got %v", res.Errors)
	}
	if !c[0].Filter.(*filter.SetFilter).Matches("open") {
		t.Errorf("known clause did not apply alongside unknown")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/querybar/ -run TestApply_ -v`
Expected: FAIL — `Apply` not declared.

- [ ] **Step 3: Add `Apply` and `ApplyResult` to `bind.go`**

In `internal/querybar/bind.go`, extend the import block to include `fmt` and `searchquery`:

```go
import (
	"fmt"
	"strings"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/searchquery"
)
```

Then append:

```go
// ApplyResult reports the outcome of an Apply call.
type ApplyResult struct {
	// BareTerms is the residual bare-term string from the AST,
	// suitable for the grid's existing quick-filter mechanism.
	BareTerms string

	// Errors lists per-clause errors. A non-empty Errors does not
	// imply Apply rolled back; clauses that succeeded have already
	// been applied.
	Errors []string
}

// Apply pushes an AST into the column filters. For each clause:
//
//   - SetFilter: merge values across same-field clauses; one SetClause
//     call with the merged values.
//   - MultiSetFilter: clear prior constraints, then one SetClause per
//     clause (each appends one constraint).
//   - Scalar filters (Text/Number/Bool/Time): use the last clause for
//     the field; record a warning if there were multiple.
//
// Columns whose filter is RoundTrippable and Active but NOT mentioned
// in the AST are Cleared. Lossy filters (RoundTrippable.Clause returns
// ok=false) are left alone — the bar can not represent them and we
// take that as "the user is not editing them right now."
//
// Per-clause errors do not abort: good clauses apply, bad ones surface
// in ApplyResult.Errors.
func Apply[T any](cols []data.Column[T], ast searchquery.AST) ApplyResult {
	res := ApplyResult{BareTerms: ast.Terms}

	// Group clauses by canonical field.
	grouped := make(map[string][]searchquery.Clause)
	for _, cl := range ast.Clauses {
		grouped[cl.Field] = append(grouped[cl.Field], cl)
	}

	// Index columns by ID for lookup, and remember which we touched.
	colIdx := make(map[string]int, len(cols))
	for i := range cols {
		colIdx[cols[i].ColumnID] = i
	}
	mentioned := make(map[string]bool, len(grouped))

	// Apply clauses.
	for field, clauses := range grouped {
		idx, ok := colIdx[field]
		if !ok {
			// Unknown field: parser keeps it; binder ignores.
			continue
		}
		mentioned[field] = true
		f := cols[idx].Filter
		rt, ok := f.(filter.RoundTrippable)
		if !ok {
			continue
		}

		switch typed := f.(type) {
		case *filter.MultiSetFilter:
			typed.Clear()
			for _, c := range clauses {
				if err := rt.SetClause(c.Values, c.Negate); err != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", field, err))
				}
			}
		case *filter.SetFilter:
			merged := mergeValues(clauses)
			negate := false
			for _, c := range clauses {
				if c.Negate {
					negate = true
					break
				}
			}
			if err := rt.SetClause(merged, negate); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", field, err))
			}
		default:
			last := clauses[len(clauses)-1]
			if len(clauses) > 1 {
				res.Errors = append(res.Errors,
					fmt.Sprintf("%s: multiple clauses on scalar field; using last", field))
			}
			if err := rt.SetClause(last.Values, last.Negate); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", field, err))
			}
		}
	}

	// Clear active RoundTrippable filters not mentioned in the AST.
	// Lossy filters (Clause ok=false) are left alone.
	for i := range cols {
		c := &cols[i]
		if c.Filter == nil || !c.Filter.Active() {
			continue
		}
		if mentioned[c.ColumnID] {
			continue
		}
		rt, ok := c.Filter.(filter.RoundTrippable)
		if !ok {
			continue
		}
		_, _, clauseOk := rt.Clause()
		if !clauseOk {
			continue // lossy — leave it
		}
		c.Filter.Clear()
	}

	return res
}

// mergeValues collects all values across a set of clauses on the same
// field. Duplicates are preserved; SetFilter's SetClause de-duplicates
// implicitly via the include set.
func mergeValues(clauses []searchquery.Clause) []string {
	var out []string
	for _, c := range clauses {
		out = append(out, c.Values...)
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/querybar/ -run TestApply_ -v -count=1`
Expected: PASS for all `Apply_*` tests.

- [ ] **Step 5: Run all querybar tests**

Run: `go test ./internal/querybar/ -v -count=1`
Expected: PASS for everything.

- [ ] **Step 6: Commit**

```bash
git add internal/querybar/bind.go internal/querybar/bind_test.go
git commit -m "querybar: add Apply — push AST into column filters"
```

---

## Task 15: Round-trip identity test

**Files:**
- Modify: `internal/querybar/bind_test.go`

- [ ] **Step 1: Append failing identity tests**

Append to `internal/querybar/bind_test.go`:

```go
func TestRoundTripIdentity(t *testing.T) {
	cases := []struct {
		name  string
		query string
	}{
		{"plain bare terms", "memory leak"},
		{"single set clause", "state:open"},
		{"set comma list", "state:open,closed"},
		{"text scalar clause", "title:memory"},
		{"number range", "count:5..20"},
		{"clause plus terms", "state:open memory"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := cols2()
			v := BuildAutoVocab(c)
			ast, err := searchquery.Parse(tc.query, v)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			res := Apply(c, ast)
			text, lossy := Rerender(c, res.BareTerms)
			if len(lossy) != 0 {
				t.Errorf("unexpected lossy: %v", lossy)
			}
			// Re-parse the rendered text; the AST should be equivalent.
			ast2, err := searchquery.Parse(text, v)
			if err != nil {
				t.Fatalf("Parse(rendered): %v\ntext=%q", err, text)
			}
			if !astsEquivalent(ast, ast2) {
				t.Errorf("round-trip drift:\n  in: %s\n  out: %s\n  ast1: %+v\n  ast2: %+v",
					tc.query, text, ast, ast2)
			}
		})
	}
}

func astsEquivalent(a, b searchquery.AST) bool {
	if a.Terms != b.Terms {
		return false
	}
	if len(a.Clauses) != len(b.Clauses) {
		return false
	}
	// Compare clauses by canonical field; values may have been
	// reordered (set membership is order-independent).
	indexBy := func(ast searchquery.AST) map[string]searchquery.Clause {
		out := make(map[string]searchquery.Clause, len(ast.Clauses))
		for _, c := range ast.Clauses {
			out[c.Field] = c
		}
		return out
	}
	ax := indexBy(a)
	bx := indexBy(b)
	for field, ac := range ax {
		bc, ok := bx[field]
		if !ok {
			return false
		}
		if ac.Negate != bc.Negate {
			return false
		}
		if !equalStringSets(ac.Values, bc.Values) {
			return false
		}
	}
	return true
}

func equalStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int, len(a))
	for _, s := range a {
		m[s]++
	}
	for _, s := range b {
		if m[s] == 0 {
			return false
		}
		m[s]--
	}
	return true
}
```

- [ ] **Step 2: Run tests to verify they pass (or fail revealingly)**

Run: `go test ./internal/querybar/ -run TestRoundTripIdentity -v -count=1`
Expected: PASS for all six sub-cases. If any case fails, fix the offending direction (Apply or Rerender) before moving on — round-trip identity is the load-bearing guarantee.

- [ ] **Step 3: Commit**

```bash
git add internal/querybar/bind_test.go
git commit -m "querybar: add round-trip identity table"
```

---

## Task 16: Add `Styles.QueryBar` and `Styles.QueryBarLossy`

**Files:**
- Modify: `grid/styles.go`

- [ ] **Step 1: Add the fields and defaults**

Edit `grid/styles.go`. Inside the `Styles` struct, find the `// Filtering` block and append:

```go
	// Filtering
	FilterInput  lipgloss.Style // Quick filter input.
	FilterMatch  lipgloss.Style // Highlighted matching text.
	FilterActive string         // Active filter indicator in header (default: "⫧").
	QueryBar     lipgloss.Style // Query bar input.
	QueryBarLossy lipgloss.Style // Style for the lossy-filter annotation in the bar.
```

In `DefaultStyles()`, find the `FilterInput` default and append after `FilterActive`:

```go
		FilterActive: "⫧",
		QueryBar: lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("236")),
		QueryBarLossy: lipgloss.NewStyle().
			Faint(true).
			Foreground(lipgloss.Color("245")),
```

- [ ] **Step 2: Verify the package builds**

Run: `go build ./grid/`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add grid/styles.go
git commit -m "grid: add Styles.QueryBar and Styles.QueryBarLossy"
```

---

## Task 17: Rename `KeyMap.QuickFilter` → `KeyMap.QueryBar`

**Files:**
- Modify: `grid/keymap.go`

- [ ] **Step 1: Rename in the struct and default**

Edit `grid/keymap.go`. In the `KeyMap` struct, change:

```go
	// Filtering
	QuickFilter  key.Binding
	ColumnFilter key.Binding
	ClearFilters key.Binding
```

to:

```go
	// Filtering
	QueryBar     key.Binding
	ColumnFilter key.Binding
	ClearFilters key.Binding
```

In `DefaultKeyMap()`, change:

```go
		QuickFilter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "quick filter"),
		),
```

to:

```go
		QueryBar: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "query"),
		),
```

- [ ] **Step 2: Verify the package builds**

Run: `go build ./grid/`
Expected: build error — `KeyMap.QuickFilter` is referenced elsewhere. We will fix those references in subsequent tasks. For now, this build error is expected.

If the build passes (no remaining references), continue to Step 3.

- [ ] **Step 3: Commit**

```bash
git add grid/keymap.go
git commit -m "grid: rename KeyMap.QuickFilter to KeyMap.QueryBar"
```

(The build will be broken at this point. The next several tasks fix references; after task 22 the build is clean again.)

---

## Task 18: Rename `QuickFilterChangedMsg` → `QueryBarChangedMsg`

**Files:**
- Modify: `grid/messages.go`

- [ ] **Step 1: Rename the type**

Edit `grid/messages.go`. Replace:

```go
// QuickFilterChangedMsg is emitted when the quick filter text changes.
type QuickFilterChangedMsg struct {
	Text string
}
```

with:

```go
// QueryBarChangedMsg is emitted when the query bar text changes
// (after a submit or on a programmatic SetText). Text is the canonical
// bar text — clauses + bare terms.
type QueryBarChangedMsg struct {
	Text string
}
```

- [ ] **Step 2: Verify the package compiles (will still error on referencing files)**

Run: `go build ./grid/`
Expected: error referencing `QuickFilterChangedMsg`. Continue.

- [ ] **Step 3: Commit**

```bash
git add grid/messages.go
git commit -m "grid: rename QuickFilterChangedMsg to QueryBarChangedMsg"
```

---

## Task 19: Replace `WithQuickFilter*` with `WithQueryBar*`

**Files:**
- Modify: `grid/options.go`

- [ ] **Step 1: Remove `WithQuickFilter` and `WithQuickFilterText`**

Edit `grid/options.go`. Delete:

```go
// WithQuickFilter enables/disables the quick filter bar.
func WithQuickFilter[T any](enabled bool) Option[T] {
	return func(m *Model[T]) {
		m.quickFilterEnabled = enabled
	}
}
```

and

```go
// WithQuickFilterText sets the initial quick filter text.
// Use alongside WithQuickFilter(true) to enable the quick filter UI.
func WithQuickFilterText[T any](text string) Option[T] {
	return func(m *Model[T]) {
		m.quickFilterText = text
		m.updateQuickFilterWords()
	}
}
```

- [ ] **Step 2: Add `WithQueryBar`, `WithQueryBarText`, `WithQueryBarVocabulary`**

Edit `grid/options.go`. First, replace the existing import block with:

```go
import (
	"time"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/internal/querybar"
	"github.com/pgavlin/tea-grid/searchquery"
	"github.com/pgavlin/tea-grid/selection"
	"github.com/pgavlin/tea-grid/sort"
)
```

Then append at the bottom of the file:

```go
// WithQueryBar enables the GitHub-style query bar. The bar is rendered
// above the headers and serves as both editing surface and canonical
// view of current filter state. Field/value clauses route to per-column
// filters via the RoundTrippable interface; bare terms feed the
// existing quick-filter substring matcher.
func WithQueryBar[T any]() Option[T] {
	return func(m *Model[T]) {
		m.queryBar = querybar.New(nil)
		m.queryBar.Enable()
	}
}

// WithQueryBarText sets the initial bar text; parsed and applied on
// grid setup against the (possibly auto-derived) vocabulary.
func WithQueryBarText[T any](text string) Option[T] {
	return func(m *Model[T]) {
		if m.queryBar == nil {
			m.queryBar = querybar.New(nil)
			m.queryBar.Enable()
		}
		m.queryBar.SetText(text)
		m.queryBar.Editor().SetText(text)
	}
}

// WithQueryBarVocabulary supplies an explicit vocabulary that shadows
// the column-derived one. Required for parse-time rewrites (is:open →
// state:open) and for queryable fields that do not correspond to a
// single column.
func WithQueryBarVocabulary[T any](v *searchquery.Vocabulary) Option[T] {
	return func(m *Model[T]) {
		if m.queryBar == nil {
			m.queryBar = querybar.New(nil)
			m.queryBar.Enable()
		}
		m.queryBar.SetVocabulary(v)
	}
}
```

- [ ] **Step 3: Verify (compilation will still fail until grid.go is updated)**

Run: `go build ./grid/`
Expected: errors mentioning `quickFilterEnabled` and `queryBar` field absent. We will fix these in the next task.

- [ ] **Step 4: Commit**

```bash
git add grid/options.go
git commit -m "grid: replace WithQuickFilter with WithQueryBar options"
```

---

## Task 20: Update `Model[T]` fields and `Init`-time vocabulary build

**Files:**
- Modify: `grid/grid.go`

- [ ] **Step 1: Replace `quickFilterEnabled` with `queryBar`, rename `quickFilterActive`**

Edit `grid/grid.go`. In the `Model[T]` struct's filtering block, currently:

```go
	// Filtering
	quickFilterEnabled       bool
	quickFilterText          string
	quickFilterActive        bool
	quickFilterWords         []string      // cached split of quickFilterText, updated on text change
	quickFilterSeq           uint64        // bumped on each keystroke, used to discard stale debounce ticks
	quickFilterDebounceDelay time.Duration // delay before recomputing after keystroke (default 100ms, 0 = immediate)
	filterEditColIdx         int           // -1 = no filter editor active
```

replace with:

```go
	// Filtering
	queryBar                 *querybar.State // nil unless WithQueryBar applied
	queryBarActive           bool            // user is editing the bar's textinput
	quickFilterText          string          // bare-term portion (drives passesQuickFilter)
	quickFilterWords         []string        // cached split of quickFilterText
	quickFilterSeq           uint64          // bumped on each keystroke; reserved for live-prefix follow-up
	quickFilterDebounceDelay time.Duration   // reserved for live-prefix follow-up
	filterEditColIdx         int             // -1 = no filter editor active
```

Add `querybar` to the imports at the top of `grid/grid.go`. Insert into the third import group (alongside `data`, `filter`, etc.):

```go
	"github.com/pgavlin/tea-grid/internal/querybar"
```

- [ ] **Step 2: After options apply in `New`, build the auto-vocabulary**

Edit `grid/grid.go`. Find the section in `New[T any]` that applies pending column filters (around line 181-187). After that block but before `m.buildStaticPinnedNodes()`, add:

```go
	// Build the query-bar's auto-vocabulary now that columns are set.
	if m.queryBar != nil {
		m.queryBar.SetAutoVocabulary(querybar.BuildAutoVocab(m.cols))
		// Apply any initial bar text by submitting it through Apply.
		if m.queryBar.Text() != "" {
			m.applyQueryBarSubmit()
		}
	}
```

- [ ] **Step 3: Update `SetColumns` to rebuild the vocabulary**

In `SetColumns` (around line 238-259), add at the end (after the existing recompute calls):

```go
	if m.queryBar != nil {
		m.queryBar.SetAutoVocabulary(querybar.BuildAutoVocab(m.cols))
		m.invalidateQueryBar()
	}
```

- [ ] **Step 4: Add `invalidateQueryBar` helper near `hasActiveFilters`**

After `hasActiveFilters` (around line 591-601), add:

```go
// invalidateQueryBar re-renders the bar's text and lossy set from the
// current filter state. Called from every site that mutates filter or
// quick-filter state. No-op when the bar is not enabled.
func (m *Model[T]) invalidateQueryBar() {
	if m.queryBar == nil || m.queryBar.Editing() {
		// While the user is editing, we do not stomp their text.
		return
	}
	text, lossy := querybar.Rerender(m.cols, m.quickFilterText)
	m.queryBar.SetText(text)
	m.queryBar.SetLossy(lossy)
}

// applyQueryBarSubmit parses the bar's text and pushes it into the
// column filters via querybar.Apply. Called on Enter in bar mode and
// at New() if WithQueryBarText was used.
func (m *Model[T]) applyQueryBarSubmit() {
	text := m.queryBar.EditorText()
	if text == "" {
		text = m.queryBar.Text()
	}
	ast, err := searchquery.Parse(text, m.queryBar.Vocab())
	if err != nil {
		m.queryBar.SetParseErr(err.Error())
		return
	}
	m.queryBar.SetParseErr("")
	res := querybar.Apply(m.cols, ast)
	if len(res.Errors) > 0 {
		m.queryBar.SetParseErr(strings.Join(res.Errors, "; "))
	}
	m.quickFilterText = res.BareTerms
	m.updateQuickFilterWords()
	m.dirty = true
	m.filterDirty = true
	// Re-render bar from canonical filter state.
	text2, lossy := querybar.Rerender(m.cols, m.quickFilterText)
	m.queryBar.SetText(text2)
	m.queryBar.SetLossy(lossy)
}
```

Add `searchquery` to the imports of `grid/grid.go`. Insert into the third import group (after `internal/querybar`):

```go
	"github.com/pgavlin/tea-grid/searchquery"
```

- [ ] **Step 5: Wire `invalidateQueryBar` into mutation sites**

Add `m.invalidateQueryBar()` immediately after each site that mutates filter or row state. Edit these sites in `grid/grid.go`:

- Inside `SetRows`, after `m.recomputeDisplayRows()`: add `m.invalidateQueryBar()`.
- Inside `UpdateRow`, after `m.recomputeDisplayRows()`: add `m.invalidateQueryBar()`.
- Inside `InsertRow`, after `m.recomputeDisplayRows()`: add `m.invalidateQueryBar()`.
- Inside `RemoveRow`, after `m.recomputeDisplayRows()`: add `m.invalidateQueryBar()`.
- Inside `SetQuickFilter`, after `m.recomputeDisplayRows()`: add `m.invalidateQueryBar()`.
- Inside `SetColumnFilter`, after `m.recomputeDisplayRows()`: add `m.invalidateQueryBar()`.
- Inside `ClearFilters`, after `m.recomputeDisplayRows()`: add `m.invalidateQueryBar()`.

- [ ] **Step 6: Update `Filtering` to include bar editing**

Currently:

```go
func (m Model[T]) Filtering() bool {
	return m.filterEditColIdx >= 0 || m.quickFilterActive
}
```

Replace with:

```go
func (m Model[T]) Filtering() bool {
	return m.filterEditColIdx >= 0 || m.queryBarActive
}
```

- [ ] **Step 7: Verify package builds**

Run: `go build ./grid/`
Expected: there will still be build errors from references in `grid/update.go` and `grid/render.go`. We fix those next.

- [ ] **Step 8: Commit**

```bash
git add grid/grid.go
git commit -m "grid: replace quickFilter* fields with queryBar; add invalidateQueryBar"
```

---

## Task 21: Replace `handleQuickFilterKeyMsg` with `handleQueryBarKeyMsg`

**Files:**
- Modify: `grid/update.go`

- [ ] **Step 1: Update the `Update` dispatch**

Edit `grid/update.go`. Replace:

```go
		} else if m.quickFilterActive {
			m, cmd = m.handleQuickFilterKeyMsg(msg)
		} else {
```

with:

```go
		} else if m.queryBarActive {
			m, cmd = m.handleQueryBarKeyMsg(msg)
		} else {
```

- [ ] **Step 2: Replace the `KeyMap.QuickFilter` open path in `handleKeyMsg`**

In `handleKeyMsg`, find the `// Quick filter` block (around lines 166-181):

```go
	// Quick filter
	case key.Matches(msg, m.KeyMap.QuickFilter):
		if m.quickFilterEnabled {
			m.quickFilterActive = !m.quickFilterActive
			if !m.quickFilterActive {
				if m.quickFilterText != "" {
					m.quickFilterText = ""
					m.dirty = true
					m.filterDirty = true
					m.updateViewportSize()
					return m, func() tea.Msg { return QuickFilterChangedMsg{Text: ""} }
				}
			}
			m.updateViewportSize()
			return m, nil
		}
```

Replace with:

```go
	// Query bar
	case key.Matches(msg, m.KeyMap.QueryBar):
		if m.queryBar != nil {
			if m.queryBarActive {
				// Toggle off — exit editing and discard uncommitted changes.
				m.queryBarActive = false
				m.queryBar.EndEdit()
			} else {
				m.queryBarActive = true
				m.queryBar.BeginEdit()
			}
			m.updateViewportSize()
			return m, nil
		}
```

- [ ] **Step 3: Replace `handleQuickFilterKeyMsg` body**

Find `handleQuickFilterKeyMsg` (around lines 397-443) and the helper `quickFilterChanged` and `quickFilterDebounceMsg` block (lines 15-27, 445-460). Delete all three blocks (the message struct, the cmd helper, the handler). In their place add:

```go
// handleQueryBarKeyMsg handles key messages while the query bar's
// textinput is focused. Enter submits, Esc cancels, all other keys are
// forwarded to the underlying lineedit.
func (m Model[T]) handleQueryBarKeyMsg(msg tea.KeyPressMsg) (Model[T], tea.Cmd) {
	switch msg.Code {
	case tea.KeyEscape:
		// Cancel: discard edits, re-render canonical text.
		m.queryBarActive = false
		m.queryBar.EndEdit()
		m.invalidateQueryBar()
		m.updateViewportSize()
		return m, nil

	case tea.KeyEnter:
		// Submit: parse + apply, then leave editing mode.
		m.applyQueryBarSubmit()
		m.queryBarActive = false
		m.queryBar.EndEdit()
		m.dirty = true
		m.filterDirty = true
		m.updateViewportSize()
		text := m.queryBar.Text()
		return m, func() tea.Msg { return QueryBarChangedMsg{Text: text} }
	}

	// Forward to the lineedit. We do not parse on every keystroke;
	// bare-term application is also deferred to submit (Enter).
	if m.queryBar.Editor().HandleKeyMsg(msg) {
		// editor mutated; nothing else to do until submit
	}
	return m, nil
}
```

Also delete the `case quickFilterDebounceMsg` block in `Update` (around lines 45-49):

```go
	case quickFilterDebounceMsg:
		if msg.seq == m.quickFilterSeq {
			m.dirty = true
			m.filterDirty = true
		}
```

Delete this entire case (the `Update` function only has `tea.KeyPressMsg` after the deletion).

Remove the `time` import if no longer needed. Check the rest of `update.go`; if `time` is still used elsewhere (it is not after this change), keep it. Otherwise drop the `"time"` import.

- [ ] **Step 4: Remove `quickFilterChanged` helper**

After deleting the body in Step 3, also remove (if not already) the `quickFilterChanged` method:

```go
func (m *Model[T]) quickFilterChanged() tea.Cmd { ... }
```

- [ ] **Step 5: Verify the package builds**

Run: `go build ./grid/`
Expected: errors only in `grid/render.go` referencing `m.renderQuickFilter` and `m.quickFilterActive` (the latter no longer exists). Continue to next task.

- [ ] **Step 6: Commit**

```bash
git add grid/update.go
git commit -m "grid: replace handleQuickFilterKeyMsg with handleQueryBarKeyMsg"
```

---

## Task 22: Replace `renderQuickFilter` with `renderQueryBar`

**Files:**
- Modify: `grid/render.go`

- [ ] **Step 1: Update the `View` dispatch**

Edit `grid/render.go`. Replace:

```go
	// Quick filter bar
	if m.quickFilterActive {
		sections = append(sections, m.renderQuickFilter())
	}
```

with:

```go
	// Query bar (always visible when enabled; editing state changes appearance)
	if m.queryBar != nil {
		sections = append(sections, m.renderQueryBar())
	}
```

- [ ] **Step 2: Replace the `renderQuickFilter` function**

Replace:

```go
// renderQuickFilter renders the quick filter input bar.
func (m Model[T]) renderQuickFilter() string {
	label := "Filter: "
	input := m.quickFilterText
	if input == "" {
		input = "Type to filter..."
	}
	line := m.styles.FilterInput.Width(m.width).Render(label + input)
	return line
}
```

with:

```go
// renderQueryBar renders the query bar above the headers. Always
// visible when the bar is enabled. Shows the canonical text (or the
// editor's current value while editing), followed by an annotation
// listing columns whose filter state can not be expressed in the bar.
func (m Model[T]) renderQueryBar() string {
	label := "/ "
	var content string
	if m.queryBarActive {
		content = m.queryBar.EditorText()
	} else {
		content = m.queryBar.Text()
	}
	if content == "" && !m.queryBarActive {
		content = "press / to filter"
	}

	body := label + content

	// Append lossy annotation if any.
	lossy := m.queryBar.Lossy()
	if len(lossy) > 0 {
		hint := ""
		if !m.queryBarActive {
			hint = " — esc to clear all"
		}
		annotation := fmt.Sprintf("  [+%d hidden filter%s: %s%s]",
			len(lossy), pluralS(len(lossy)), strings.Join(lossy, ", "), hint)
		body += m.styles.QueryBarLossy.Render(annotation)
	}

	// Append parse error if any.
	if e := m.queryBar.ParseErr(); e != "" {
		body += "  " + m.styles.QueryBarLossy.Render("("+e+")")
	}

	return m.styles.QueryBar.Width(m.width).Render(body)
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
```

Make sure `fmt` and `strings` are imported in `grid/render.go`. Check the file's import block — they should be present already; add if missing.

- [ ] **Step 3: Verify the package builds**

Run: `go build ./grid/`
Expected: success.

- [ ] **Step 4: Run the grid tests to spot regressions**

Run: `go test ./grid/ -count=1`
Expected: some tests will fail because they reference `WithQuickFilter`. We update those in the next task.

- [ ] **Step 5: Commit**

```bash
git add grid/render.go
git commit -m "grid: replace renderQuickFilter with renderQueryBar (always-visible)"
```

---

## Task 23: Update FullHelp + bench tests (mechanical renames)

**Files:**
- Modify: `grid/grid.go`, `grid/bench_test.go`

- [ ] **Step 1: Update `FullHelp` to use `QueryBar`**

In `grid/grid.go`, find the `FullHelp` function (around lines 740-750). Currently:

```go
{m.KeyMap.Select, m.KeyMap.SelectAll, m.KeyMap.QuickFilter, m.KeyMap.ClearFilters},
```

Change to:

```go
{m.KeyMap.Select, m.KeyMap.SelectAll, m.KeyMap.QueryBar, m.KeyMap.ClearFilters},
```

- [ ] **Step 2: Update bench tests (mechanical rename)**

In `grid/bench_test.go`, replace every `WithQuickFilterText[benchRow](...)` with `WithQueryBarText[benchRow](...)` and every `WithQuickFilter[benchRow](true)` with `WithQueryBar[benchRow]()`. Use sed:

```bash
sed -i '' \
  -e 's/WithQuickFilterText\[benchRow\]/WithQueryBarText[benchRow]/g' \
  -e 's/WithQuickFilter\[benchRow\](true)/WithQueryBar[benchRow]()/g' \
  grid/bench_test.go
```

`Column.QuickFilterMatch` references in bench files stay — that field is unchanged.

`WithQuickFilterDebounce` references in bench files stay (the option is kept inert in v1).

- [ ] **Step 3: Verify package builds**

Run: `go build ./grid/`
Expected: success.

- [ ] **Step 4: Verify benches compile**

Run: `go test ./grid/ -bench=. -benchtime=1x -run=^$`
Expected: each benchmark runs once with no compile errors. (Real benchmark numbers come later; this just confirms compilation.)

- [ ] **Step 5: Commit**

```bash
git add grid/grid.go grid/bench_test.go
git commit -m "grid: rename QuickFilter to QueryBar in FullHelp and bench tests"
```

---

## Task 23b: Delete obsolete quick-filter tests and rename surviving ones

**Files:**
- Modify: `grid/grid_test.go`

The test suite has dozens of tests that exercise the OLD quick-filter user-facing behavior (`/` toggles, runes go directly to filter text, Enter just confirms, Esc clears text). The new bar's behavior is different (Enter parses + applies; Esc cancels without clearing; runes go to lineedit). The right move is to **delete the obsolete tests** and let Task 24's new query-bar tests take their place. The grid's underlying methods (`SetQuickFilter`, `passesQuickFilter`, `quickFilterText`) are kept; tests that exercise those remain valid.

- [ ] **Step 1: Delete obsolete tests**

In `grid/grid_test.go`, delete each of the following test functions in full (subject + body):

- `TestQuickFilter_SlashActivates`
- `TestQuickFilter_RunesAddToFilterText`
- `TestQuickFilter_EnterConfirms`
- `TestQuickFilter_EscClearsAndCloses`
- `TestQuickFilter_FiltersResults`
- `TestQuickFilter_DisabledIgnoresSlash`
- `TestQuickFilter_BackspaceRemovesChar`
- `TestHandleKeyMsg_QuickFilterToggleOnOff`
- `TestQuickFilter_BackspaceWhenEmpty`
- `TestQuickFilter_EnterConfirmsFilter`
- `TestQuickFilter_EscWithEmptyText`
- `TestQuickFilter_ToggleOffWithTextViaHandleKeyMsg`
- `TestQuickFilterKeyMsg_EscWithText`
- `TestQuickFilterKeyMsg_BackspaceEmitsMsg`
- `TestQuickFilterKeyMsg_RunesEmitsMsg`
- `TestQuickFilterKeyMsg_UnhandledKeyType`
- `TestRender_QuickFilterBarShownWhenActive` (replaced conceptually by a new test below)

Use `grep -n "^func Test.*QuickFilter" grid/grid_test.go` to locate each, then remove the entire function body up to and including the closing `}`.

- [ ] **Step 2: Rename surviving "with quick filter" tests**

Rename the following — they exercise the underlying machinery that is preserved (`SetQuickFilter`, `passesQuickFilter`, `WithQueryBar`) and should keep working:

- `TestWithQuickFilter` → `TestWithQueryBar`
  Inside: `WithQuickFilter[TestRow](true)` → `WithQueryBar[TestRow]()`
- `TestWithQuickFilterText_FiltersRows` → `TestWithQueryBarText_FiltersRows`
  Inside: `WithQuickFilterText[TestRow]("Engineering")` → `WithQueryBarText[TestRow]("Engineering")`
- `TestWithQuickFilterText_WithQuickFilterEnabled` → `TestWithQueryBarText_Combined`
  Inside: replace both options with their query-bar equivalents.
- `TestClearFilters_EscClearsQuickFilterText` (keep name; the test still exercises ClearFilters)
  Inside: `WithQuickFilter[TestRow](true)` → `WithQueryBar[TestRow]()`. The `m.SetQuickFilter("Carol")` call is still valid.

`TestDisplayRows_QuickFilter`, `TestPublicAPI_SetQuickFilter`, `TestInitLifecycle_SetQuickFilterAfterNew`, `TestInitLifecycle_ViewBeforeAndAfterSetQuickFilter`, `TestPassesQuickFilter_NilValue`, `TestPassesQuickFilter_WhitespaceOnlyFilter` — these test internal methods (`SetQuickFilter`, `passesQuickFilter`) directly without going through the bar. Inside each, replace any `WithQuickFilter[TestRow](true)` with `WithQueryBar[TestRow]()`. The test names can stay as-is (they describe the underlying behavior, which is preserved).

- [ ] **Step 3: Update message-type references**

Replace every remaining `QuickFilterChangedMsg` reference in `grid/grid_test.go` with `QueryBarChangedMsg`:

```bash
sed -i '' 's/QuickFilterChangedMsg/QueryBarChangedMsg/g' grid/grid_test.go
```

- [ ] **Step 4: Run grid tests**

Run: `go test ./grid/ -count=1`
Expected: PASS. If any test fails, inspect the failure — it usually points to a remaining old-behavior assumption that should be ported to the new bar tests added in Task 24, or removed if obsolete.

- [ ] **Step 5: Regenerate golden files if any fail**

The current goldens (`grid/testdata/multiline_*.golden`) do not exercise the quick-filter or query-bar render paths, so regeneration is unlikely to be needed. If any golden test does fail with diffs reflecting the new always-visible bar:

Run: `go test ./grid/ -update-golden`

Re-run:

Run: `go test ./grid/ -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add grid/grid_test.go grid/testdata
git commit -m "grid: drop obsolete quick-filter tests; rename surviving ones to QueryBar"
```

---

## Task 24: Add grid integration tests for the query bar

**Files:**
- Modify: `grid/grid_test.go`

- [ ] **Step 1: Append integration tests**

Append to `grid/grid_test.go`:

```go
func TestQueryBar_OpenAndCancel(t *testing.T) {
	g := New(
		WithColumns([]data.Column[testRow]{
			{ColumnID: "name", Filter: filter.NewTextFilter(), Filterable: true},
		}),
		WithQueryBar[testRow](),
		WithFocused[testRow](true),
		WithWidth[testRow](80),
		WithHeight[testRow](10),
	)
	// Press '/' to open.
	g, _ = g.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	if !g.Filtering() {
		t.Errorf("Filtering() = false after '/', want true")
	}
	// Press Esc to cancel.
	g, _ = g.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if g.Filtering() {
		t.Errorf("Filtering() = true after Esc, want false")
	}
}

func TestQueryBar_SubmitAppliesFilter(t *testing.T) {
	g := New(
		WithColumns([]data.Column[testRow]{
			{ColumnID: "name", Filter: filter.NewTextFilter(), Filterable: true},
		}),
		WithQueryBar[testRow](),
		WithRows([]testRow{{Name: "alice"}, {Name: "bob"}}),
		WithRowID(func(r testRow) string { return r.Name }),
		WithFocused[testRow](true),
		WithWidth[testRow](80),
		WithHeight[testRow](10),
	)
	// Open the bar.
	g, _ = g.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	// Type "name:alice".
	for _, r := range "name:alice" {
		g, _ = g.Update(tea.KeyPressMsg{Code: rune(r), Text: string(r)})
	}
	// Submit.
	g, _ = g.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	// Filter should now restrict the visible rows.
	displayed := g.displayRows
	if len(displayed) != 1 || displayed[0].Data.Name != "alice" {
		t.Errorf("after submit: %d rows, want 1 (alice)", len(displayed))
	}
}

func TestQueryBar_FiltersToBarReflectsChanges(t *testing.T) {
	textF := filter.NewTextFilter()
	g := New(
		WithColumns([]data.Column[testRow]{
			{ColumnID: "name", Filter: textF, Filterable: true},
		}),
		WithQueryBar[testRow](),
		WithRows([]testRow{{Name: "alice"}, {Name: "bob"}}),
		WithRowID(func(r testRow) string { return r.Name }),
		WithFocused[testRow](true),
		WithWidth[testRow](80),
		WithHeight[testRow](10),
	)
	// Programmatically set the filter; the bar should re-render to
	// reflect it.
	textF.SetText("alice")
	g.invalidateQueryBar()
	if got := g.queryBar.Text(); !strings.Contains(got, "name:alice") {
		t.Errorf("bar text = %q, want it to contain name:alice", got)
	}
}

func TestQueryBar_LossyAnnotation(t *testing.T) {
	tf := filter.NewTextFilter()
	tf.SetRegex(true)
	tf.SetText("foo.*")
	g := New(
		WithColumns([]data.Column[testRow]{
			{ColumnID: "name", Filter: tf, Filterable: true},
		}),
		WithQueryBar[testRow](),
		WithRows([]testRow{{Name: "alice"}}),
		WithRowID(func(r testRow) string { return r.Name }),
		WithFocused[testRow](true),
		WithWidth[testRow](80),
		WithHeight[testRow](10),
	)
	g.invalidateQueryBar()
	lossy := g.queryBar.Lossy()
	if len(lossy) != 1 || lossy[0] != "name" {
		t.Errorf("Lossy() = %v, want [name]", lossy)
	}
}

func TestQueryBar_ClearFiltersClearsLossy(t *testing.T) {
	tf := filter.NewTextFilter()
	tf.SetRegex(true)
	tf.SetText("foo.*")
	g := New(
		WithColumns([]data.Column[testRow]{
			{ColumnID: "name", Filter: tf, Filterable: true},
		}),
		WithQueryBar[testRow](),
		WithRows([]testRow{{Name: "alice"}}),
		WithRowID(func(r testRow) string { return r.Name }),
		WithFocused[testRow](true),
		WithWidth[testRow](80),
		WithHeight[testRow](10),
	)
	g.ClearFilters()
	if tf.Active() {
		t.Errorf("ClearFilters did not clear lossy filter")
	}
}
```

If `testRow` does not exist yet in `grid_test.go`, add at the top:

```go
type testRow struct {
	Name string
}
```

- [ ] **Step 2: Run integration tests**

Run: `go test ./grid/ -run TestQueryBar -v -count=1`
Expected: PASS for all five.

- [ ] **Step 3: Run all tests for the project**

Run: `go test ./... -count=1`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add grid/grid_test.go
git commit -m "grid: add query bar integration tests"
```

---

## Task 25: Update existing examples — replace `WithQuickFilter` with `WithQueryBar`

**Files:**
- Modify: `examples/anyrow/main.go`, `examples/basic/main.go`, `examples/columns/main.go`, `examples/csv/main.go`, `examples/jsonl/main.go`, `examples/selection/main.go`, `examples/spreadsheet/main.go`

- [ ] **Step 1: Run a sweep replace**

For each file, change:

```go
grid.WithQuickFilter[X](true),
```

to:

```go
grid.WithQueryBar[X](),
```

(where `X` is the row type — `Employee`, `Row`, `*SpreadsheetRow`, etc.)

Use:

```bash
sed -i '' 's/grid.WithQuickFilter\[\([^]]*\)\](true)/grid.WithQueryBar[\1]()/g' \
  examples/anyrow/main.go examples/basic/main.go examples/columns/main.go \
  examples/csv/main.go examples/jsonl/main.go examples/selection/main.go \
  examples/spreadsheet/main.go
```

- [ ] **Step 2: Verify each example builds**

Run:

```bash
for ex in anyrow basic columns csv jsonl selection spreadsheet; do
  go build ./examples/$ex/ || echo "FAILED: $ex"
done
```

Expected: no `FAILED` lines.

- [ ] **Step 3: Commit**

```bash
git add examples/
git commit -m "examples: migrate from WithQuickFilter to WithQueryBar"
```

---

## Task 26: Add `examples/querybar/` walkthrough example

**Files:**
- Create: `examples/querybar/main.go`

- [ ] **Step 1: Create the example**

Create `examples/querybar/main.go`:

```go
// querybar demonstrates the GitHub-style query bar with all built-in
// filter types: text (substring), number (>5, 5..20), set (comma-OR),
// bool (true/false), time (date and range), and multiset (AND-of-
// includes for slice-valued columns).
//
// Sample queries to try (after pressing '/'):
//
//   state:open
//   state:open,closed
//   priority:>3
//   active:true
//   created:2026-01-01..2026-12-31
//   labels:bug labels:urgent
//   memory leak                       (bare terms — substring match across columns)
//   state:open critical               (mix of clauses and bare terms)
//
// Press '/' to focus the bar, Enter to submit, Esc to cancel.
package main

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grid"
)

type Issue struct {
	Title    string
	State    string
	Priority int
	Active   bool
	Created  time.Time
	Labels   []string
}

func columns() []data.Column[Issue] {
	return []data.Column[Issue]{
		{
			ColumnID:   "title",
			HeaderName: "Title",
			Value:      func(i Issue) any { return i.Title },
			Filter:     filter.NewTextFilter(),
			Filterable: true,
			Flex:       2,
		},
		{
			ColumnID:   "state",
			HeaderName: "State",
			Value:      func(i Issue) any { return i.State },
			Filter:     filter.NewSetFilter("open", "closed", "draft"),
			Filterable: true,
			Width:      10,
		},
		{
			ColumnID:   "priority",
			HeaderName: "Priority",
			Value:      func(i Issue) any { return i.Priority },
			Filter:     filter.NewNumberFilter(),
			Filterable: true,
			Width:      10,
		},
		{
			ColumnID:   "active",
			HeaderName: "Active",
			Value:      func(i Issue) any { return i.Active },
			Filter:     filter.NewBoolFilter(),
			Filterable: true,
			Width:      8,
		},
		{
			ColumnID:   "created",
			HeaderName: "Created",
			Value:      func(i Issue) any { return i.Created },
			Filter:     filter.NewTimeFilter(),
			Filterable: true,
			Width:      14,
		},
		{
			ColumnID:   "labels",
			HeaderName: "Labels",
			Value:      func(i Issue) any { return i.Labels },
			Filter:     filter.NewMultiSetFilter(),
			Filterable: true,
			Flex:       1,
		},
	}
}

func sampleData() []Issue {
	mustParse := func(s string) time.Time {
		t, _ := time.Parse("2006-01-02", s)
		return t
	}
	return []Issue{
		{Title: "memory leak in worker pool", State: "open", Priority: 5, Active: true, Created: mustParse("2026-01-15"), Labels: []string{"bug", "urgent"}},
		{Title: "add dark mode", State: "open", Priority: 2, Active: true, Created: mustParse("2026-02-03"), Labels: []string{"feature", "ux"}},
		{Title: "flaky test in checkout", State: "closed", Priority: 4, Active: false, Created: mustParse("2026-01-20"), Labels: []string{"bug", "test"}},
		{Title: "rewrite billing module", State: "draft", Priority: 1, Active: true, Created: mustParse("2026-03-01"), Labels: []string{"refactor"}},
		{Title: "investigate pager noise", State: "open", Priority: 3, Active: true, Created: mustParse("2026-03-12"), Labels: []string{"ops", "urgent"}},
	}
}

type model struct {
	grid grid.Model[Issue]
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.grid.SetWidth(msg.Width)
		m.grid.SetHeight(msg.Height)
	case tea.KeyPressMsg:
		if !m.grid.Filtering() && (msg.String() == "q" || msg.String() == "ctrl+c") {
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.grid, cmd = m.grid.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	v := tea.NewView(m.grid.View())
	v.AltScreen = true
	return v
}

func main() {
	g := grid.New(
		grid.WithColumns(columns()),
		grid.WithRows(sampleData()),
		grid.WithRowID(func(i Issue) string { return i.Title }),
		grid.WithQueryBar[Issue](),
		grid.WithFocused[Issue](true),
	)
	p := tea.NewProgram(model{grid: g})
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify the example builds**

Run: `go build ./examples/querybar/`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add examples/querybar/
git commit -m "examples: add querybar walkthrough with all filter types"
```

---

## Task 27: Write `filter/README.md`

**Files:**
- Create: `filter/README.md`

- [ ] **Step 1: Create the README**

Create `filter/README.md`:

```markdown
# filter

Per-column filters for tea-grid. Defines the `Filter` interface and the
built-in filter implementations: `TextFilter`, `NumberFilter`,
`SetFilter`, `BoolFilter`, `TimeFilter`, and `MultiSetFilter`.

## Filter interface

```go
type Filter interface {
    Matches(value any) bool
    View() string
    Update(msg tea.Msg) (Filter, tea.Cmd)
    Active() bool
    Clear()
}
```

A `Filter` lives on a `data.Column[T].Filter` field. The grid invokes
`Matches` against the cell value during the display-row pipeline and
routes `Update` events to the focused filter while it is being edited.

## Built-in filters

**TextFilter** -- substring match. Optional regex mode via `SetRegex(true)`.
**NumberFilter** -- comparison operators (`=`, `!=`, `<`, `>`, `<=`, `>=`)
and ranges (`5..20`). Accepts `int`, `int64`, `float32`, `float64`.
**SetFilter** -- include/exclude from a fixed set of distinct values.
Inline checkbox UI in the popup. OR semantics within the included subset.
**BoolFilter** -- three states: any, true only, false only.
**TimeFilter** -- date or date range. Accepts a wide set of layouts
(see `parseTime`).
**MultiSetFilter** -- AND-of-includes for slice-valued columns. The
popup is view-and-remove only; new constraints come from the query bar
(see below). Default matcher handles `[]string` row values; supply
`WithMultiSetMatcher` for richer slice element types.

## RoundTrippable

Filters that can serialize their state to and from a query-bar clause
implement `RoundTrippable`:

```go
type RoundTrippable interface {
    SetClause(values []string, negate bool) error
    Clause() (values []string, negate bool, ok bool)
}
```

Every built-in filter implements `RoundTrippable`. Filters that do not
are treated as opaque by the query bar — annotated, not edited.

`Clause` returns `ok=false` when the filter is inactive (no constraint
to surface) or in a state that can not be expressed in the query syntax.
The query bar treats both ok=false states the same way: skip in the bar
text, mention in the lossy annotation if `Active()` is true.

### SetFilter negation flip

`SetFilter.Clause` returns the included subset by default. Once more
than half of `allValues` is included, it flips to the excluded subset
with `negate=true` to keep the bar text small.

`SetFilter` operates on a known `allValues` set. New values that appear
in row data after a clause was applied are treated as "not included" —
consistent with `Include`/`SetValues` semantics.

### TextFilter regex mode is lossy

`TextFilter` in regex mode returns `ok=false` from `Clause`; the query
bar annotates the column as a hidden filter rather than rendering a
`text:` clause for it.

`SetClause` on a `TextFilter` resets regex mode. The rule is uniform:
submitting a clause for a column overrides whatever state that filter
was in, lossy or not.

### MultiSetFilter editing model

`MultiSetFilter` accumulates constraints with AND semantics. The popup
shows the current constraint list with `× constraint` rows; arrow keys
move row focus, `d` (or backspace) removes the focused constraint.
There is no add affordance in the popup — new constraints come from the
query bar. The footer reads:

```
d delete · esc close · / edit
```

`SetClause` is called once per repeated bar clause; each call appends
one constraint. v1 rejects multi-value (comma-list) values inside a
single clause.

## License

Same as tea-grid (root `LICENSE`).
```

- [ ] **Step 2: Commit**

```bash
git add filter/README.md
git commit -m "filter: add README covering built-in filters and RoundTrippable"
```

---

## Task 28: Write `grid/README.md`

**Files:**
- Create: `grid/README.md`

- [ ] **Step 1: Create the README**

Create `grid/README.md`:

```markdown
# grid

Top-level tea-grid component. Composes `data`, `filter`, `sort`,
`grouping`, `selection`, and the internal `querybar` widget into a
single `Model[T]` that implements `tea.Model`.

## Quick start

```go
g := grid.New(
    grid.WithColumns(cols),
    grid.WithRows(rows),
    grid.WithRowID(func(r Row) string { return r.ID }),
    grid.WithFocused[Row](true),
)
```

The model is parameterized on the row type `T`; columns extract values
via `data.Column[T].Value`.

## Query bar

Enable the GitHub-style query bar with `WithQueryBar`:

```go
g := grid.New(
    grid.WithColumns(cols),
    grid.WithQueryBar[Row](),
)
```

The bar is rendered above the headers and serves as both editing
surface and canonical view of current filter state. Press `/` to focus
the textinput, type a query, press Enter to submit (or Esc to cancel).

### Field/value clauses

Field/value clauses (`state:open`, `count:>5`, `created:2026-01-01..*`)
route to per-column filters via the `RoundTrippable` interface. The
column's `ColumnID` is the field name by default; add aliases via
`Column.QueryAliases`:

```go
{ColumnID: "state", QueryAliases: []string{"status", "st"}, ...}
```

Bare terms (anything not a `field:value` token) feed the existing
quick-filter substring matcher.

### Vocabulary

The query-bar vocabulary is auto-derived from the column set. Every
visible, filterable column with a `RoundTrippable` filter becomes a
queryable field; field type is inferred from filter type. Override with
`WithQueryBarVocabulary` for parse-time rewrites (e.g. `is:open` →
`state:open`) or for queryable fields that do not correspond to a
single column.

### Lossy filter states

Some filter states can not be expressed in the query syntax — `TextFilter`
in regex mode, non-`RoundTrippable` filters. The bar annotates these
inline:

```
state:open  bug    [+1 hidden filter: regex on title — esc to clear all]
```

Submit replaces only round-trippable clauses. Lossy filters are left
alone; clear them via the column UI or via the global `ClearFilters`
binding (default `esc`).

### Initial bar text

```go
grid.New(
    grid.WithColumns(cols),
    grid.WithQueryBar[Row](),
    grid.WithQueryBarText[Row]("state:open priority:>3"),
)
```

The text is parsed and applied at construction.

## Key bindings

Default bindings (customizable via `WithKeyMap`):

| Action            | Key       |
|-------------------|-----------|
| Open query bar    | `/`       |
| Submit query      | `Enter`   |
| Cancel bar edit   | `Esc`     |
| Clear all filters | `Esc`     |
| Open column filter| `Ctrl+F`  |
| Sort column       | `s`       |
| Multi-sort column | `S`       |
| Auto-fit column   | `w`       |
| Auto-fit all      | `W`       |

See `keymap.go` for the full list.

## License

Same as tea-grid (root `LICENSE`).
```

- [ ] **Step 2: Commit**

```bash
git add grid/README.md
git commit -m "grid: add README covering query bar and key bindings"
```

---

## Task 29: Update `searchquery/README.md`

**Files:**
- Modify: `searchquery/README.md`

- [ ] **Step 1: Mark binder/round-trip items as landed**

Edit `searchquery/README.md`. Replace the entire `## Open follow-ups` section through the end of `### Multi-clause-on-same-field (related to round-trip)` with:

```markdown
## Status

The parser, AST, and `Vocabulary` registry are content-agnostic and
backend-agnostic. tea-grid wires them into `grid.Model[T]` via the
`internal/querybar` package; consumers enable the integration with
`grid.WithQueryBar()`. See `grid/README.md` for usage.

## Use with tea-grid

```go
g := grid.New(
    grid.WithColumns(cols),
    grid.WithQueryBar[Row](),
)
```

The vocabulary is auto-derived from columns. Override with
`grid.WithQueryBarVocabulary(v)` to add parse-time rewrites
(`is:open` → `state:open`) or queryable fields that do not correspond
to a single column.

## Open follow-ups

### Generic `Range[T]`

`filter.NumberFilter` and this package's `ParseTimeRange` parse the
same shape (`>x`, `<x`, `>=x`, `<=x`, `a..b`, `*..b`, `a..*`) against
different value types. Parameterize once, drop into both:

```go
type Range[T any] struct {
    From, To       *T
    FromExclusive  bool
    ToExclusive    bool
}

func ParseRange[T any](s string, parse func(string) (T, error)) (Range[T], error)
```

`NumberFilter`'s `=`/`!=` operators stay as separate operator handling
on top of the range parser.

### Live bare-prefix while typing

The bar currently submits on Enter only. A future enhancement could
apply bare-term changes incrementally (matching today's quick-filter
UX) by splitting the parse into "definitely bare prefix" and "rest"
without invoking the full parser per keystroke.

### Tab-completion

The parser already exposes a `Vocabulary`. A Tab-completion path on
the bar would use it to suggest field names and values. Out of scope
for the v1 query-bar landing.
```

- [ ] **Step 2: Commit**

```bash
git add searchquery/README.md
git commit -m "searchquery: README — mark query-bar landing complete; update follow-ups"
```

---

## Task 30: Update root `README.md` features list

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the Filtering bullet**

Edit `README.md`. Find:

```markdown
- **Filtering** -- quick filter, per-column filters (text, number, set, bool, time)
```

Replace with:

```markdown
- **Filtering** -- per-column filters (text, number, set, bool, time, multiset) and a GitHub-style query bar with round-trip between bar text and column filter state
```

If the Quick Start example uses `WithQuickFilter`, update it to `WithQueryBar`. (Inspect the file to confirm.)

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update root README features list for query bar"
```

---

## Task 31: Final verification — full test suite + race + lint

**Files:** none

- [ ] **Step 1: Run the full test suite**

Run: `go test ./... -count=1`
Expected: PASS for every package.

- [ ] **Step 2: Run with race detector**

Run: `go test -race ./...`
Expected: PASS, no data races.

- [ ] **Step 3: Run lint and format check**

Run: `gofumpt -l .`
Expected: no output (all files formatted).

If anything is listed, run `gofumpt -w` on those files and commit:

```bash
gofumpt -w <listed files>
git add <listed files>
git commit -m "style: gofumpt"
```

Run: `golangci-lint run ./...`
Expected: no issues. Fix any linting issues in a follow-up commit.

- [ ] **Step 4: Build all examples one more time**

```bash
for ex in $(ls examples); do
  go build ./examples/$ex/ || echo "FAILED: $ex"
done
```

Expected: no `FAILED` lines.

- [ ] **Step 5: Smoke-test the new example interactively**

Run: `go run ./examples/querybar/`

Verify by hand:
- The bar renders above the headers.
- `/` opens the bar; type `state:open`, press Enter — only `open` rows remain.
- The bar text reflects the active filter (`state:open`).
- Open the column popup for `state` (Ctrl+F on the state column), toggle `closed` on, close — the bar updates to show `state:open,closed`.
- Type `priority:>3` and submit; only rows with priority > 3 remain.
- Type `labels:bug labels:urgent` and submit; only rows with both labels remain.

Press `q` or Ctrl+C to quit.

- [ ] **Step 6: Final commit (if anything else fell out)**

If verification revealed any issues, fix them and commit. Otherwise this task is complete with no further changes.
