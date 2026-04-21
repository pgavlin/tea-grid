# Clear-filters Keybinding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `ClearFilters` key binding (default `esc`) that clears all column filters and the quick filter, with priority below `DeselectAll` so the first `esc` clears the selection and the next clears the filters.

**Architecture:** New `KeyMap.ClearFilters` field, new `FiltersClearedMsg` message type, new private `Model[T].hasActiveFilters()` helper, and a single combined `case` in `handleKeyMsg` that matches either binding and dispatches by priority (selection first, then filters). The existing public `ClearFilters()` method is reused; no API changes to it.

**Tech Stack:** Go 1.25, Bubble Tea v2 (`charm.land/bubbletea/v2`), Bubbles v2 (`charm.land/bubbles/v2/key`), standard `testing` package.

**Spec:** `docs/superpowers/specs/2026-04-21-clear-filters-keybinding-design.md`

---

## File Map

- **Modify** `grid/keymap.go` — add `ClearFilters` field to `KeyMap` struct + default in `DefaultKeyMap()`.
- **Modify** `grid/messages.go` — add `FiltersClearedMsg` type.
- **Modify** `grid/grid.go` — add private `hasActiveFilters()` method on `Model[T]`; add `ClearFilters` to `FullHelp`.
- **Modify** `grid/update.go` — replace standalone `DeselectAll` `case` in `handleKeyMsg` with a combined case that handles both bindings by priority.
- **Modify** `grid/grid_test.go` — add tests covering all four behavioral branches plus the `FullHelp` entry.

No new files.

---

## Task 1: Add the `ClearFilters` field to `KeyMap`

**Files:**
- Modify: `grid/keymap.go`

- [ ] **Step 1: Add the field to the `KeyMap` struct**

In `grid/keymap.go`, find the `// Filtering` block (currently lines 52-54) and add a third field:

```go
	// Filtering
	QuickFilter  key.Binding
	ColumnFilter key.Binding
	ClearFilters key.Binding
```

- [ ] **Step 2: Add the default binding in `DefaultKeyMap`**

In `grid/keymap.go`, find the `ColumnFilter` default (currently lines 195-198) and add a `ClearFilters` default immediately after it:

```go
		ColumnFilter: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "column filter"),
		),
		ClearFilters: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filters"),
		),
```

- [ ] **Step 3: Verify the package still builds**

Run: `go build ./grid/`
Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add grid/keymap.go
git commit -m "grid: add ClearFilters key binding (default esc)"
```

---

## Task 2: Add `FiltersClearedMsg`

**Files:**
- Modify: `grid/messages.go`

- [ ] **Step 1: Add the message type**

In `grid/messages.go`, append after `QuickFilterChangedMsg` (currently ends at line 75):

```go
// FiltersClearedMsg is emitted when the user clears all column filters and
// the quick filter via the ClearFilters key binding.
type FiltersClearedMsg struct{}
```

- [ ] **Step 2: Verify the package still builds**

Run: `go build ./grid/`
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add grid/messages.go
git commit -m "grid: add FiltersClearedMsg"
```

---

## Task 3: Add `hasActiveFilters` helper

**Files:**
- Modify: `grid/grid.go`

- [ ] **Step 1: Write a failing test for the helper**

In `grid/grid_test.go`, add this test at the end of the file (after the last existing test):

```go
func TestHasActiveFilters(t *testing.T) {
	cols := testCols()
	cols[0].Filter = filter.NewTextFilter()
	m := newTestGrid(WithColumns[TestRow](cols))

	if m.hasActiveFilters() {
		t.Error("expected hasActiveFilters=false on fresh grid")
	}

	// Quick filter text alone counts as active.
	m.SetQuickFilter("hello")
	if !m.hasActiveFilters() {
		t.Error("expected hasActiveFilters=true with quick filter text")
	}
	m.SetQuickFilter("")

	// Inactive column filter (no text set) does not count.
	if m.hasActiveFilters() {
		t.Error("expected hasActiveFilters=false with empty column filter")
	}

	// Active column filter counts.
	tf := filter.NewTextFilter()
	tf.SetText("Carol")
	m.SetColumnFilter("Name", tf)
	if !m.hasActiveFilters() {
		t.Error("expected hasActiveFilters=true with active column filter")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./grid/ -run TestHasActiveFilters -v`
Expected: FAIL — compile error: `m.hasActiveFilters undefined`.

- [ ] **Step 3: Implement the helper**

In `grid/grid.go`, immediately after the existing `ClearFilters` method (currently lines 576-586), add:

```go
// hasActiveFilters reports whether any column filter is active or the quick
// filter text is non-empty. Used by the ClearFilters key binding to skip the
// recompute when there is nothing to clear.
func (m *Model[T]) hasActiveFilters() bool {
	if m.quickFilterText != "" {
		return true
	}
	for i := range m.cols {
		if m.cols[i].Filter != nil && m.cols[i].Filter.Active() {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./grid/ -run TestHasActiveFilters -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add grid/grid.go grid/grid_test.go
git commit -m "grid: add hasActiveFilters helper"
```

---

## Task 4: Wire the binding into `handleKeyMsg`

This is the behavior change. Five sub-tests, then implementation, then verify all pass.

**Files:**
- Modify: `grid/update.go`
- Modify: `grid/grid_test.go`

- [ ] **Step 1: Write failing test — selection-active esc clears only selection**

In `grid/grid_test.go`, append at the end of the file:

```go
func TestClearFilters_EscClearsSelectionFirst(t *testing.T) {
	cols := testCols()
	cols[0].Filter = filter.NewTextFilter()
	m := newTestGrid(
		WithColumns[TestRow](cols),
		WithSelection[TestRow](selection.SelectMulti),
	)

	// Set up: active filter + active selection.
	tf := filter.NewTextFilter()
	tf.SetText("Carol")
	m.SetColumnFilter("Name", tf)
	m.SelectAllRows()
	if !m.sel.Active() {
		t.Fatal("precondition: expected selection to be active")
	}
	if !m.hasActiveFilters() {
		t.Fatal("precondition: expected filter to be active")
	}

	m = sendKey(m, tea.KeyPressMsg{Code: tea.KeyEscape})

	if m.sel.Active() {
		t.Error("expected selection cleared by first esc")
	}
	if !m.hasActiveFilters() {
		t.Error("expected filter to remain active after first esc")
	}
}
```

- [ ] **Step 2: Write failing test — no-selection esc with active column filter clears it**

Append:

```go
func TestClearFilters_EscClearsColumnFilter(t *testing.T) {
	cols := testCols()
	cols[0].Filter = filter.NewTextFilter()
	m := newTestGrid(WithColumns[TestRow](cols))

	tf := filter.NewTextFilter()
	tf.SetText("Carol")
	m.SetColumnFilter("Name", tf)
	if !m.hasActiveFilters() {
		t.Fatal("precondition: expected filter to be active")
	}

	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if m.hasActiveFilters() {
		t.Error("expected filters cleared by esc")
	}
	if cmd == nil {
		t.Fatal("expected FiltersClearedMsg command")
	}
	if _, ok := cmd().(FiltersClearedMsg); !ok {
		t.Errorf("expected FiltersClearedMsg, got %T", cmd())
	}
}
```

- [ ] **Step 3: Write failing test — no-selection esc with persisted quick filter text clears it**

Append:

```go
func TestClearFilters_EscClearsQuickFilterText(t *testing.T) {
	m := newTestGrid(WithQuickFilter[TestRow](true))

	// Confirmed quick filter: text persists, mode is no longer active.
	m.SetQuickFilter("Carol")
	if m.quickFilterActive {
		t.Fatal("precondition: quickFilterActive must be false (confirmed text)")
	}
	if !m.hasActiveFilters() {
		t.Fatal("precondition: expected quick filter text to count as active")
	}

	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if m.quickFilterText != "" {
		t.Errorf("expected quickFilterText cleared, got %q", m.quickFilterText)
	}
	if cmd == nil {
		t.Fatal("expected FiltersClearedMsg command")
	}
	if _, ok := cmd().(FiltersClearedMsg); !ok {
		t.Errorf("expected FiltersClearedMsg, got %T", cmd())
	}
}
```

- [ ] **Step 4: Write failing test — esc with no selection and no filters is a no-op**

Append:

```go
func TestClearFilters_EscNoopWhenNothingActive(t *testing.T) {
	m := newTestGrid()

	if m.sel.Active() {
		t.Fatal("precondition: selection must be inactive")
	}
	if m.hasActiveFilters() {
		t.Fatal("precondition: filters must be inactive")
	}

	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if cmd != nil {
		t.Errorf("expected nil command for no-op esc, got %T", cmd())
	}
	if m.hasActiveFilters() {
		t.Error("expected filters still inactive")
	}
	if m.sel.Active() {
		t.Error("expected selection still inactive")
	}
}
```

- [ ] **Step 5: Run all four new tests to verify they fail**

Run: `go test ./grid/ -run 'TestClearFilters_Esc' -v`
Expected: all FAIL — the third test (`TestClearFilters_EscClearsQuickFilterText`) and fourth (`TestClearFilters_EscNoopWhenNothingActive`) likely fail because the current `DeselectAll` case calls `m.ClearSelection()` and returns `nil` regardless of filter state; the second test (`TestClearFilters_EscClearsColumnFilter`) fails because `cmd` is `nil`. The first test (`TestClearFilters_EscClearsSelectionFirst`) may already pass (current behavior: esc clears selection, filters untouched) — that's fine.

- [ ] **Step 6: Replace the `DeselectAll` case in `handleKeyMsg`**

In `grid/update.go`, find the existing `DeselectAll` case (currently lines 194-196):

```go
		case key.Matches(msg, m.KeyMap.DeselectAll):
			m.ClearSelection()
			return m, nil
```

Replace it with a combined case that handles both bindings by priority:

```go
		case key.Matches(msg, m.KeyMap.DeselectAll), key.Matches(msg, m.KeyMap.ClearFilters):
			if m.sel.Active() {
				m.ClearSelection()
				return m, nil
			}
			if m.hasActiveFilters() {
				m.ClearFilters()
				return m, func() tea.Msg { return FiltersClearedMsg{} }
			}
			return m, nil
```

Note the comma between the two `key.Matches` calls — Go's `case` allows multiple comma-separated expressions.

- [ ] **Step 7: Run the four new tests to verify they pass**

Run: `go test ./grid/ -run 'TestClearFilters_Esc' -v`
Expected: all PASS.

- [ ] **Step 8: Run the full grid test suite to verify no regressions**

Run: `go test ./grid/ -count=1`
Expected: PASS. In particular, the existing `TestFilterEdit_EscCancelsAndClearsFilter` (line 3296) and quick-filter esc tests must still pass — they exercise the per-mode handlers that route before normal mode, so they should be unaffected.

- [ ] **Step 9: Commit**

```bash
git add grid/update.go grid/grid_test.go
git commit -m "grid: bind esc to ClearFilters with priority below DeselectAll"
```

---

## Task 5: Add `ClearFilters` to `FullHelp`

**Files:**
- Modify: `grid/grid.go`
- Modify: `grid/grid_test.go`

- [ ] **Step 1: Write a failing test for the help entry**

In `grid/grid_test.go`, append at the end of the file:

```go
func TestFullHelp_IncludesClearFilters(t *testing.T) {
	m := newTestGrid()
	groups := m.FullHelp()

	found := false
	for _, group := range groups {
		for _, b := range group {
			if key.Matches(tea.KeyPressMsg{Code: tea.KeyEscape}, b) &&
				b.Help().Desc == "clear filters" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected FullHelp to include ClearFilters (esc) binding")
	}
}
```

The `Desc == "clear filters"` check disambiguates from `DeselectAll`, which also matches `esc`.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./grid/ -run TestFullHelp_IncludesClearFilters -v`
Expected: FAIL — `expected FullHelp to include ClearFilters (esc) binding`.

- [ ] **Step 3: Add `ClearFilters` to `FullHelp`**

In `grid/grid.go`, find `FullHelp` (currently lines 725-734) and modify the `QuickFilter` row:

```go
// FullHelp implements help.KeyMap.
func (m Model[T]) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.KeyMap.Up, m.KeyMap.Down, m.KeyMap.Left, m.KeyMap.Right},
		{m.KeyMap.PageUp, m.KeyMap.PageDown, m.KeyMap.Home, m.KeyMap.End},
		{m.KeyMap.Select, m.KeyMap.SelectAll, m.KeyMap.QuickFilter, m.KeyMap.ClearFilters},
		{m.KeyMap.AutoSizeColumn, m.KeyMap.AutoSizeColumns},
		{m.KeyMap.Help},
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./grid/ -run TestFullHelp_IncludesClearFilters -v`
Expected: PASS.

- [ ] **Step 5: Run the full grid test suite**

Run: `go test ./grid/ -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add grid/grid.go grid/grid_test.go
git commit -m "grid: list ClearFilters in FullHelp"
```

---

## Task 6: Final verification

**Files:** none (no changes).

- [ ] **Step 1: Run the full test suite with the race detector**

Run: `go test -race ./...`
Expected: PASS.

- [ ] **Step 2: Run the linter**

Run: `golangci-lint run`
Expected: no issues.

- [ ] **Step 3: Verify formatting**

Run: `gofumpt -l .`
Expected: no output (no files need formatting).

If any files are listed, run `gofumpt -w .` and amend the appropriate commit (or, if the offending file is one you didn't touch, leave it alone — it predates this work).

---

## Self-Review Notes

- **Spec coverage:** every numbered item in the spec maps to a task. KeyMap addition → Task 1. `FiltersClearedMsg` → Task 2. `hasActiveFilters` helper → Task 3. Combined `handleKeyMsg` case → Task 4. `FullHelp` update → Task 5. All six test cases listed in the spec are covered: selection-priority (Task 4 step 1), column-filter-clear (Task 4 step 2), quick-filter-text-clear (Task 4 step 3 — the called-out edge case), no-op (Task 4 step 4), `FullHelp` (Task 5 step 1), existing per-mode esc tests preserved (Task 4 step 8 regression check).
- **Type consistency:** `ClearFilters` (binding name), `FiltersClearedMsg` (message type), `hasActiveFilters` (helper) used consistently throughout. The existing public method `ClearFilters()` and binding `KeyMap.ClearFilters` share a name — by Go's rules they don't collide (different namespaces) and the spec calls this out as intentional reuse of the existing method.
- **Out-of-scope discipline:** no changes to per-mode esc handlers, no changes to public `ClearFilters()` semantics, no new "clear quick filter only" or "clear column filters only" bindings.
