# Auto-Fit Column Widths Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship content-based auto-fit column widths for tea-grid: a declarative `Column.AutoFit` flag that measures raw rows, an imperative `AutoSizeColumn(s)` API that measures displayed rows with sticky overrides, and default `w` / `W` keybindings.

**Architecture:** Extend `Column[T]` with `AutoFit bool`. Add a `NaturalWidthRenderer[T]` opt-in interface so renderers can declare their natural width (built-in `BarRenderer` / `ProgressRenderer` / `SparklineRenderer` adopt it). Measurement lives in a single helper in `grid/grid.go` used by both declarative and imperative paths. Imperative results are stored in `Model.manualWidths map[string]int` and take highest precedence in `computeColWidths`. New `KeyMap.AutoSizeColumn` / `AutoSizeColumns` dispatch from `handleKeyMsg`.

**Tech Stack:** Go 1.25, `charm.land/lipgloss/v2` for display-width measurement (`lipgloss.Width` — strips ANSI, handles CJK/emoji), `charm.land/bubbles/v2/key` for bindings. TDD with `go test`.

**Spec:** `docs/superpowers/specs/2026-04-20-autofit-column-widths-design.md`

**Run tests:** `go test ./grid/... ./data/...`
**Run one test:** `go test ./grid/ -run TestName -v`
**Race detector:** `go test -race ./...`

---

## File Structure

**Create:**
- (none — all changes are additive to existing files)

**Modify:**
- `data/cell.go` — add `NaturalWidthRenderer[T]` interface.
- `data/column.go` — add `AutoFit bool` field to `Column[T]`.
- `data/cell_builtin.go` — add `PreferredWidth` field and `NaturalWidth` method to `BarRenderer` and `ProgressRenderer`; add `NaturalWidth` method to `SparklineRenderer`.
- `grid/grid.go` — add `hasAutoFit bool` and `manualWidths map[string]int` fields to `Model`; add measurement helper; update `computeColWidths` for overrides and AutoFit; add `AutoSizeColumn(s)` and `ResetColumnWidth(s)` methods; trigger `computeColWidths` from data-mutating setters when `hasAutoFit`; prune `manualWidths` in `SetColumns`.
- `grid/keymap.go` — add `AutoSizeColumn` / `AutoSizeColumns` fields and defaults.
- `grid/update.go` — dispatch `w` / `W` in `handleKeyMsg`.
- `CLAUDE.md` — document `AutoFit`, the four methods, and the key bindings.

**Tests:** Add to `grid/grid_test.go` and `data/cell_builtin_test.go`. Existing helpers (`testCols`, `testData`, `newTestGrid`) are reused.

---

## Task 1: Add `NaturalWidthRenderer[T]` interface

**Files:**
- Modify: `data/cell.go`
- Test: `data/cell_test.go`

- [ ] **Step 1: Write the failing test**

Append to `data/cell_test.go`:

```go
// naturalWidthTestRenderer is a test renderer that implements
// both CellRenderer and NaturalWidthRenderer.
type naturalWidthTestRenderer struct {
	natural int
}

func (r naturalWidthTestRenderer) Render(ctx CellContext[int]) string {
	return "x"
}

func (r naturalWidthTestRenderer) NaturalWidth(ctx CellContext[int]) int {
	return r.natural
}

func TestNaturalWidthRenderer_InterfaceSatisfied(t *testing.T) {
	var r CellRenderer[int] = naturalWidthTestRenderer{natural: 7}
	nw, ok := r.(NaturalWidthRenderer[int])
	if !ok {
		t.Fatal("expected renderer to satisfy NaturalWidthRenderer[int]")
	}
	if got := nw.NaturalWidth(CellContext[int]{}); got != 7 {
		t.Errorf("NaturalWidth = %d, want 7", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./data/ -run TestNaturalWidthRenderer_InterfaceSatisfied -v`
Expected: compile error — `undefined: NaturalWidthRenderer`.

- [ ] **Step 3: Add the interface**

In `data/cell.go`, after the `CellRendererFunc` block (around line 32), append:

```go
// NaturalWidthRenderer reports a renderer's preferred display width,
// independent of ctx.Width. Renderers that want their output to drive
// AutoFit should implement this. Renderers that produce width-independent
// output (e.g. plain text) can skip it; the column's AutoFit path falls
// back to Column.Text / ValueFormatter / Value for measurement.
type NaturalWidthRenderer[T any] interface {
	CellRenderer[T]
	NaturalWidth(ctx CellContext[T]) int
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./data/ -run TestNaturalWidthRenderer_InterfaceSatisfied -v`
Expected: PASS.

- [ ] **Step 5: Run the full data-package tests**

Run: `go test ./data/...`
Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add data/cell.go data/cell_test.go
git commit -m "data: add NaturalWidthRenderer interface for AutoFit measurement"
```

---

## Task 2: Add `AutoFit bool` field to `Column[T]`

**Files:**
- Modify: `data/column.go:48-53`

This is a purely additive field change with no behavior yet. No test — behavior lands in Task 3+.

- [ ] **Step 1: Add the field**

In `data/column.go`, find the Sizing block (lines 48-53) and add `AutoFit`:

```go
	// Sizing
	Width    int  // Fixed width in terminal columns. 0 = auto.
	MinWidth int  // Minimum width (default: 4).
	MaxWidth int  // Maximum width. 0 = unconstrained.
	Flex     int  // Flex weight for distributing remaining space. 0 = no flex.
	AutoFit  bool // Size to widest rendered content (header + rows), clamped to [MinWidth, MaxWidth]. Ignored when Width > 0. Takes precedence over Flex. Overridden by AutoSizeColumn(s).
```

- [ ] **Step 2: Verify compile**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Verify existing tests still pass**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add data/column.go
git commit -m "data: add AutoFit bool field to Column[T]"
```

---

## Task 3: Add measurement helper `measureColumnWidth`

**Files:**
- Modify: `grid/grid.go` (add helper function near `computeColWidths`)
- Test: `grid/grid_test.go`

Build a pure helper that measures a column's content width against a provided row set, using `NaturalWidthRenderer` when available and the Text/ValueFormatter/Value fallback otherwise. Clamp to `[MinWidth, MaxWidth]`.

- [ ] **Step 1: Write the failing test**

Append to `grid/grid_test.go`:

```go
func TestMeasureColumnWidth_HeaderOnly(t *testing.T) {
	col := data.Column[TestRow]{
		ColumnID:   "Name",
		HeaderName: "Name",
		Value:      func(r TestRow) any { return r.Name },
		MinWidth:   4,
	}
	m := newTestGrid()
	got := m.measureColumnWidth(col, 0, nil)
	if got != 4 {
		t.Errorf("empty rows + short header should clamp to MinWidth=4, got %d", got)
	}
}

func TestMeasureColumnWidth_RowsDriveWidth(t *testing.T) {
	col := data.Column[TestRow]{
		ColumnID:   "Name",
		HeaderName: "N",
		Value:      func(r TestRow) any { return r.Name },
		MinWidth:   1,
	}
	rows := []*data.RowNode[TestRow]{
		{Data: TestRow{Name: "Alice"}},
		{Data: TestRow{Name: "Bartholomew"}},
	}
	m := newTestGrid()
	got := m.measureColumnWidth(col, 0, rows)
	if got != len("Bartholomew") {
		t.Errorf("expected width %d, got %d", len("Bartholomew"), got)
	}
}

func TestMeasureColumnWidth_ClampsToMaxWidth(t *testing.T) {
	col := data.Column[TestRow]{
		ColumnID:   "Name",
		HeaderName: "N",
		Value:      func(r TestRow) any { return r.Name },
		MinWidth:   1,
		MaxWidth:   5,
	}
	rows := []*data.RowNode[TestRow]{{Data: TestRow{Name: "Bartholomew"}}}
	m := newTestGrid()
	got := m.measureColumnWidth(col, 0, rows)
	if got != 5 {
		t.Errorf("expected width clamped to MaxWidth=5, got %d", got)
	}
}

func TestMeasureColumnWidth_ClampsToMinWidth(t *testing.T) {
	col := data.Column[TestRow]{
		ColumnID:   "Name",
		HeaderName: "N",
		Value:      func(r TestRow) any { return r.Name },
		MinWidth:   10,
	}
	rows := []*data.RowNode[TestRow]{{Data: TestRow{Name: "Al"}}}
	m := newTestGrid()
	got := m.measureColumnWidth(col, 0, rows)
	if got != 10 {
		t.Errorf("expected width clamped to MinWidth=10, got %d", got)
	}
}

func TestMeasureColumnWidth_UsesTextFallback(t *testing.T) {
	col := data.Column[TestRow]{
		ColumnID:   "Name",
		HeaderName: "N",
		Text:       func(r *TestRow) string { return "[" + r.Name + "]" },
		Value:      func(r TestRow) any { return r.Name },
		MinWidth:   1,
	}
	rows := []*data.RowNode[TestRow]{{Data: TestRow{Name: "Al"}}}
	m := newTestGrid()
	got := m.measureColumnWidth(col, 0, rows)
	if got != 4 {
		t.Errorf("Text() should produce \"[Al]\" (4 chars), got width %d", got)
	}
}

type fixedNaturalRenderer struct {
	width int
}

func (r fixedNaturalRenderer) Render(ctx data.CellContext[TestRow]) string { return "" }
func (r fixedNaturalRenderer) NaturalWidth(ctx data.CellContext[TestRow]) int {
	return r.width
}

func TestMeasureColumnWidth_UsesNaturalWidthRenderer(t *testing.T) {
	col := data.Column[TestRow]{
		ColumnID:     "Bar",
		HeaderName:   "B",
		CellRenderer: fixedNaturalRenderer{width: 7},
		Value:        func(r TestRow) any { return r.Salary },
		MinWidth:     1,
	}
	rows := []*data.RowNode[TestRow]{
		{Data: TestRow{Salary: 1}},
		{Data: TestRow{Salary: 2}},
	}
	m := newTestGrid()
	got := m.measureColumnWidth(col, 0, rows)
	if got != 7 {
		t.Errorf("expected NaturalWidth=7, got %d", got)
	}
}

type ignoredRenderer struct{}

func (ignoredRenderer) Render(ctx data.CellContext[TestRow]) string { return "XXXXXXXX" }

func TestMeasureColumnWidth_RendererWithoutNaturalWidthFallsThroughToText(t *testing.T) {
	col := data.Column[TestRow]{
		ColumnID:     "Name",
		HeaderName:   "N",
		CellRenderer: ignoredRenderer{},
		Value:        func(r TestRow) any { return r.Name },
		MinWidth:     1,
	}
	rows := []*data.RowNode[TestRow]{{Data: TestRow{Name: "Al"}}}
	m := newTestGrid()
	got := m.measureColumnWidth(col, 0, rows)
	if got != 2 {
		t.Errorf("renderer without NaturalWidth should be ignored for measurement; expected 2 (from Value \"Al\"), got %d", got)
	}
}

func TestMeasureColumnWidth_SkipsSpanningCells(t *testing.T) {
	col := data.Column[TestRow]{
		ColumnID:   "Name",
		HeaderName: "N",
		Value:      func(r TestRow) any { return r.Name },
		ColumnSpan: func(r TestRow) int { return 3 },
		MinWidth:   1,
	}
	rows := []*data.RowNode[TestRow]{{Data: TestRow{Name: "Bartholomew"}}}
	m := newTestGrid()
	got := m.measureColumnWidth(col, 0, rows)
	if got != 1 {
		t.Errorf("spanning cells should be skipped; expected MinWidth=1, got %d", got)
	}
}

func TestMeasureColumnWidth_WideRunes(t *testing.T) {
	col := data.Column[TestRow]{
		ColumnID:   "Name",
		HeaderName: "N",
		Value:      func(r TestRow) any { return r.Name },
		MinWidth:   1,
	}
	// "日本語" is 3 runes but 6 display columns.
	rows := []*data.RowNode[TestRow]{{Data: TestRow{Name: "日本語"}}}
	m := newTestGrid()
	got := m.measureColumnWidth(col, 0, rows)
	if got != 6 {
		t.Errorf("expected display width 6 for CJK, got %d", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./grid/ -run TestMeasureColumnWidth -v`
Expected: compile error — `m.measureColumnWidth undefined`.

- [ ] **Step 3: Add the measurement helper**

In `grid/grid.go`, just before `computeColWidths` (before line 1146 — the existing `// computeColWidths runs the column sizing algorithm.` comment), add:

```go
// measureColumnWidth returns the content-fit width for col measured against
// the given rows. It takes the max of the header width and each row's
// rendered display width (via NaturalWidthRenderer or the Text/Value chain),
// then clamps to [MinWidth (default 4), MaxWidth (0 = unbounded)].
// Cells with ColumnSpan > 1 are skipped — they lie about per-column width
// by design.
func (m *Model[T]) measureColumnWidth(col data.Column[T], colIdx int, rows []*data.RowNode[T]) int {
	minW := col.MinWidth
	if minW == 0 {
		minW = 4
	}

	w := lipgloss.Width(col.HeaderName)

	var natural data.NaturalWidthRenderer[T]
	if col.CellRenderer != nil {
		if nw, ok := col.CellRenderer.(data.NaturalWidthRenderer[T]); ok {
			natural = nw
		}
	}

	for i, rn := range rows {
		if col.ColumnSpan != nil && col.ColumnSpan(rn.Data) > 1 {
			continue
		}
		var cellW int
		if natural != nil {
			ctx := m.buildMeasureContext(col, colIdx, i, rn)
			cellW = natural.NaturalWidth(ctx)
		} else {
			cellW = lipgloss.Width(measureCellText(col, rn.Data))
		}
		if cellW > w {
			w = cellW
		}
	}

	if w < minW {
		w = minW
	}
	if col.MaxWidth > 0 && w > col.MaxWidth {
		w = col.MaxWidth
	}
	return w
}

// measureCellText produces the string a column would render for row via the
// Text / ValueFormatter / Value fallback chain (no CellRenderer invocation).
func measureCellText[T any](col data.Column[T], row T) string {
	if col.Text != nil {
		return col.Text(&row)
	}
	var v any
	if col.Value != nil {
		v = col.Value(row)
	}
	if col.ValueFormatter != nil {
		return col.ValueFormatter(v, row)
	}
	return conv.SprintValue(v)
}

// buildMeasureContext constructs a CellContext suitable for a NaturalWidth
// query. Width = MaxWidth if set, else 0 (signals "no target width").
func (m *Model[T]) buildMeasureContext(col data.Column[T], colIdx, rowIdx int, rn *data.RowNode[T]) data.CellContext[T] {
	var val any
	var formatted string
	if col.Text != nil {
		formatted = col.Text(&rn.Data)
	} else {
		if col.Value != nil {
			val = col.Value(rn.Data)
		}
		formatted = conv.SprintValue(val)
		if col.ValueFormatter != nil {
			formatted = col.ValueFormatter(val, rn.Data)
		}
	}
	w := col.MaxWidth
	return data.CellContext[T]{
		Value:          val,
		FormattedValue: formatted,
		Data:           rn.Data,
		RowNode:        rn,
		Column:         &col,
		ColumnIndex:    colIdx,
		RowIndex:       rowIdx,
		Width:          w,
		Height:         1,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./grid/ -run TestMeasureColumnWidth -v`
Expected: all PASS.

- [ ] **Step 5: Run the full test suite**

Run: `go test ./...`
Expected: all PASS (no regressions).

- [ ] **Step 6: Commit**

```bash
git add grid/grid.go grid/grid_test.go
git commit -m "grid: add measureColumnWidth helper for AutoFit"
```

---

## Task 4: Integrate AutoFit into `computeColWidths`

**Files:**
- Modify: `grid/grid.go:1146-1236` (`computeColWidths`)
- Test: `grid/grid_test.go`

Extend `computeColWidths` so AutoFit columns are measured after fixed-width columns, before flex distribution. Measurement uses `m.rows` (the raw row set).

- [ ] **Step 1: Write the failing tests**

Append to `grid/grid_test.go`:

```go
func TestComputeColWidths_AutoFit_Basic(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "Name", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, AutoFit: true},
		{ColumnID: "Dept", HeaderName: "Dept", Value: func(r TestRow) any { return r.Department }, MinWidth: 1, Flex: 1},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := newTestGrid(WithColumns[TestRow](cols), WithStyles[TestRow](s))
	// testData: longest Name is "Carol" (5). Header "Name" (4). Max = 5.
	if m.colWidths[0] != 5 {
		t.Errorf("AutoFit column width = %d, want 5", m.colWidths[0])
	}
}

func TestComputeColWidths_AutoFit_ClampedByMaxWidth(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "Name", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, MaxWidth: 3, AutoFit: true},
		{ColumnID: "Dept", HeaderName: "Dept", Value: func(r TestRow) any { return r.Department }, MinWidth: 1, Flex: 1},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := newTestGrid(WithColumns[TestRow](cols), WithStyles[TestRow](s))
	if m.colWidths[0] != 3 {
		t.Errorf("AutoFit column clamped to MaxWidth=3, got %d", m.colWidths[0])
	}
}

func TestComputeColWidths_AutoFit_ClampedByMinWidth(t *testing.T) {
	// No rows → empty measurement → clamp to MinWidth
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "N", Value: func(r TestRow) any { return r.Name }, MinWidth: 10, AutoFit: true},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := New(
		WithColumns[TestRow](cols),
		WithWidth[TestRow](80),
		WithStyles[TestRow](s),
	)
	if m.colWidths[0] != 10 {
		t.Errorf("AutoFit column with no rows clamped to MinWidth=10, got %d", m.colWidths[0])
	}
}

func TestComputeColWidths_AutoFit_HeaderIsMeasured(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "X", HeaderName: "LongHeaderText", Value: func(r TestRow) any { return "a" }, MinWidth: 1, AutoFit: true},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := newTestGrid(WithColumns[TestRow](cols), WithStyles[TestRow](s))
	if m.colWidths[0] != len("LongHeaderText") {
		t.Errorf("expected width %d (header), got %d", len("LongHeaderText"), m.colWidths[0])
	}
}

func TestComputeColWidths_AutoFit_IgnoredWhenWidthSet(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "Name", Value: func(r TestRow) any { return r.Name }, Width: 20, AutoFit: true},
		{ColumnID: "Dept", HeaderName: "Dept", Value: func(r TestRow) any { return r.Department }, MinWidth: 1, Flex: 1},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := newTestGrid(WithColumns[TestRow](cols), WithStyles[TestRow](s))
	if m.colWidths[0] != 20 {
		t.Errorf("Width=20 should win over AutoFit, got %d", m.colWidths[0])
	}
}

func TestComputeColWidths_AutoFit_TakesPrecedenceOverFlex(t *testing.T) {
	// AutoFit column should be sized to content, not share the flex pool.
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "Name", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, AutoFit: true, Flex: 3},
		{ColumnID: "Dept", HeaderName: "Dept", Value: func(r TestRow) any { return r.Department }, MinWidth: 1, Flex: 1},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := newTestGrid(WithColumns[TestRow](cols), WithStyles[TestRow](s))
	// Longest Name is "Carol" = 5
	if m.colWidths[0] != 5 {
		t.Errorf("AutoFit+Flex: AutoFit wins, expected 5, got %d", m.colWidths[0])
	}
	// Dept column should absorb the rest
	if m.colWidths[1] != 80-5 {
		t.Errorf("Dept column should absorb remaining width, expected %d, got %d", 80-5, m.colWidths[1])
	}
}

func TestComputeColWidths_AutoFit_SpanningCellsIgnored(t *testing.T) {
	cols := []data.Column[TestRow]{
		{
			ColumnID:   "Name",
			HeaderName: "N",
			Value:      func(r TestRow) any { return r.Name },
			ColumnSpan: func(r TestRow) int { return 2 },
			MinWidth:   1,
			AutoFit:    true,
		},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := New(
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{Name: "Bartholomew"}}),
		WithWidth[TestRow](80),
		WithStyles[TestRow](s),
	)
	// Spanning cells are skipped; only the header "N" contributes → clamp to MinWidth=1.
	if m.colWidths[0] != 1 {
		t.Errorf("spanning cells skipped, expected MinWidth=1, got %d", m.colWidths[0])
	}
}

func TestComputeColWidths_AutoFit_HiddenColumn(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "Name", Value: func(r TestRow) any { return r.Name }, Hide: true, AutoFit: true},
		{ColumnID: "Dept", HeaderName: "Dept", Value: func(r TestRow) any { return r.Department }, MinWidth: 1, Flex: 1},
	}
	m := newTestGrid(WithColumns[TestRow](cols))
	if m.colWidths[0] != 0 {
		t.Errorf("hidden AutoFit column width should be 0, got %d", m.colWidths[0])
	}
}

func TestComputeColWidths_AutoFit_StableUnderFilter(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "N", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, AutoFit: true, Filter: filter.NewTextFilter()},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := newTestGrid(WithColumns[TestRow](cols), WithStyles[TestRow](s))
	before := m.colWidths[0]
	// Apply a filter that narrows to only "Alice" rows
	tf := filter.NewTextFilter()
	tf.SetText("Alice")
	m.SetColumnFilter("Name", tf)
	after := m.colWidths[0]
	if before != after {
		t.Errorf("AutoFit should be stable under filter: %d -> %d", before, after)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./grid/ -run TestComputeColWidths_AutoFit -v`
Expected: multiple failures — AutoFit columns fall through to the flex path and get wider-than-content widths.

- [ ] **Step 3: Extend `computeColWidths` for AutoFit**

Replace the body of `computeColWidths` in `grid/grid.go:1146-1236` with:

```go
// computeColWidths runs the column sizing algorithm.
func (m *Model[T]) computeColWidths() {
	if len(m.cols) == 0 || m.width == 0 {
		return
	}

	visibleCols := m.visibleCols()
	n := len(visibleCols)
	if n == 0 {
		return
	}

	m.colWidths = make([]int, len(m.cols))
	remaining := m.width

	// Account for column separators
	if m.styles.BorderColumn {
		remaining -= n - 1
	}

	// Track which columns still need sizing as flex.
	type colInfo struct {
		idx  int
		flex int
		min  int
		max  int
	}

	var flexCols []colInfo
	for _, idx := range visibleCols {
		col := m.cols[idx]
		switch {
		case col.Width > 0:
			m.colWidths[idx] = col.Width
			remaining -= col.Width
		case col.AutoFit:
			w := m.measureColumnWidth(col, idx, m.rows)
			m.colWidths[idx] = w
			remaining -= w
		default:
			minW := col.MinWidth
			if minW == 0 {
				minW = 4
			}
			flex := col.Flex
			if flex == 0 {
				flex = 1
			}
			flexCols = append(flexCols, colInfo{
				idx:  idx,
				flex: flex,
				min:  minW,
				max:  col.MaxWidth,
			})
		}
	}

	// Allocate minimums to flex columns
	for _, fc := range flexCols {
		m.colWidths[fc.idx] = fc.min
		remaining -= fc.min
	}

	// Distribute remaining space by flex weight.
	// Use cumulative calculation to avoid losing pixels to integer division.
	if remaining > 0 && len(flexCols) > 0 {
		totalFlex := 0
		for _, fc := range flexCols {
			totalFlex += fc.flex
		}

		distributed := 0
		cumulativeFlex := 0
		for i := range flexCols {
			cumulativeFlex += flexCols[i].flex
			target := remaining * cumulativeFlex / totalFlex
			extra := target - distributed
			distributed = target

			newWidth := m.colWidths[flexCols[i].idx] + extra

			if flexCols[i].max > 0 && newWidth > flexCols[i].max {
				newWidth = flexCols[i].max
			}

			m.colWidths[flexCols[i].idx] = newWidth
		}
	}

	// Rebuild cached column indices, styles, and visible count
	m.rebuildVisibleColIndices()
	m.updateVisibleColCount()
	m.rebuildColStyles()
}
```

- [ ] **Step 4: Run AutoFit tests**

Run: `go test ./grid/ -run TestComputeColWidths_AutoFit -v`
Expected: all PASS.

- [ ] **Step 5: Run the full test suite**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add grid/grid.go grid/grid_test.go
git commit -m "grid: wire AutoFit columns into computeColWidths"
```

---

## Task 5: Trigger `computeColWidths` on data mutations for AutoFit

**Files:**
- Modify: `grid/grid.go` (`Model` struct, `New`, `SetRows`, `SetColumns`, `InsertRow`, `RemoveRow`, `UpdateRow`)
- Test: `grid/grid_test.go`

Add `hasAutoFit bool` on `Model`. Recompute it whenever columns change. Use it to gate calls to `computeColWidths` from data-mutating setters so non-AutoFit grids pay no extra cost.

- [ ] **Step 1: Write the failing tests**

Append to `grid/grid_test.go`:

```go
func TestComputeColWidths_AutoFit_RecomputesOnSetRows(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "N", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, AutoFit: true},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := New(
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{Name: "Al"}}),
		WithWidth[TestRow](80),
		WithStyles[TestRow](s),
	)
	if got := m.colWidths[0]; got != 2 {
		t.Fatalf("initial width = %d, want 2", got)
	}
	m.SetRows([]TestRow{{Name: "Bartholomew"}})
	if got := m.colWidths[0]; got != len("Bartholomew") {
		t.Errorf("after SetRows width = %d, want %d", got, len("Bartholomew"))
	}
}

func TestComputeColWidths_AutoFit_RecomputesOnInsertRow(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "N", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, AutoFit: true},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := New(
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{Name: "Al"}}),
		WithWidth[TestRow](80),
		WithStyles[TestRow](s),
	)
	m.InsertRow(0, TestRow{Name: "Bartholomew"})
	if got := m.colWidths[0]; got != len("Bartholomew") {
		t.Errorf("after InsertRow width = %d, want %d", got, len("Bartholomew"))
	}
}

func TestComputeColWidths_AutoFit_RecomputesOnUpdateRow(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "N", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, AutoFit: true},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := New(
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{Name: "Al"}}),
		WithWidth[TestRow](80),
		WithStyles[TestRow](s),
	)
	// Find the ID of the existing row
	rowID := m.rows[0].ID
	m.UpdateRow(rowID, TestRow{Name: "Bartholomew"})
	if got := m.colWidths[0]; got != len("Bartholomew") {
		t.Errorf("after UpdateRow width = %d, want %d", got, len("Bartholomew"))
	}
}

func TestComputeColWidths_AutoFit_RecomputesOnRemoveRow(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "N", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, AutoFit: true},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := New(
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{Name: "Al"}, {Name: "Bartholomew"}}),
		WithWidth[TestRow](80),
		WithStyles[TestRow](s),
	)
	if got := m.colWidths[0]; got != len("Bartholomew") {
		t.Fatalf("initial width = %d, want %d", got, len("Bartholomew"))
	}
	// Remove the long one
	longID := m.rows[1].ID
	m.RemoveRow(longID)
	if got := m.colWidths[0]; got != 2 {
		t.Errorf("after RemoveRow width = %d, want 2", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./grid/ -run "TestComputeColWidths_AutoFit_RecomputesOn" -v`
Expected: each test fails because data mutations don't currently call `computeColWidths`.

- [ ] **Step 3: Add `hasAutoFit` field and update helper**

In `grid/grid.go`, add the field to the `Model` struct (after the `colWidths` line, around line 56):

```go
	// Column widths (computed)
	colWidths []int

	// True if any column has AutoFit = true. Used to gate data-triggered
	// layout recomputes on SetRows / InsertRow / RemoveRow / UpdateRow.
	hasAutoFit bool

	// Sticky width overrides set by AutoSizeColumn(s). Keyed by ColumnID.
	manualWidths map[string]int
```

Next, add a small helper near `computeColWidths`:

```go
// refreshHasAutoFit recomputes the hasAutoFit cache from the current
// column set.
func (m *Model[T]) refreshHasAutoFit() {
	m.hasAutoFit = false
	for i := range m.cols {
		if m.cols[i].AutoFit {
			m.hasAutoFit = true
			return
		}
	}
}
```

- [ ] **Step 4: Wire `hasAutoFit` into column-change sites**

In `New` (`grid/grid.go:193-196`), add `m.refreshHasAutoFit()` before the layout call:

```go
	// Build initial display rows and compute layout
	m.recomputeDisplayRows()
	m.refreshHasAutoFit()
	m.computeColWidths()
	m.updateViewportSize()
```

In `SetColumns` (around line 227-233), add the refresh:

```go
func (m *Model[T]) SetColumns(cols []data.Column[T]) {
	m.cols = cols
	m.dirty = true
	m.filterDirty = true
	m.refreshHasAutoFit()
	m.recomputeDisplayRows()
	m.computeColWidths()
}
```

- [ ] **Step 5: Trigger `computeColWidths` in data mutators when `hasAutoFit`**

Update `SetRows` (around line 210):

```go
func (m *Model[T]) SetRows(rows []T) {
	m.setRowData(rows)
	m.pruneSelection()
	m.dirty = true
	m.recomputeDisplayRows()
	if m.hasAutoFit {
		m.computeColWidths()
	}
}
```

Update `UpdateRow` (around line 241):

```go
func (m *Model[T]) UpdateRow(id string, d T) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].Data = d
			m.dirty = true
			m.filterDirty = true
			m.recomputeDisplayRows()
			if m.hasAutoFit {
				m.computeColWidths()
			}
			return
		}
	}
}
```

Update `InsertRow` (around line 254):

```go
func (m *Model[T]) InsertRow(index int, d T) {
	rn := m.makeRowNode(d)
	if index >= len(m.rows) {
		m.rows = append(m.rows, rn)
	} else {
		m.rows = append(m.rows[:index+1], m.rows[index:]...)
		m.rows[index] = rn
	}
	m.dirty = true
	m.filterDirty = true
	m.recomputeDisplayRows()
	if m.hasAutoFit {
		m.computeColWidths()
	}
}
```

Update `RemoveRow` (around line 268):

```go
func (m *Model[T]) RemoveRow(id string) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			m.sel.Clear()
			m.dirty = true
			m.filterDirty = true
			m.recomputeDisplayRows()
			if m.hasAutoFit {
				m.computeColWidths()
			}
			return
		}
	}
}
```

- [ ] **Step 6: Run the new tests**

Run: `go test ./grid/ -run "TestComputeColWidths_AutoFit_RecomputesOn" -v`
Expected: all PASS.

- [ ] **Step 7: Run the full test suite**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
git add grid/grid.go grid/grid_test.go
git commit -m "grid: trigger computeColWidths on data mutation when AutoFit is in use"
```

---

## Task 6: Add `manualWidths` override precedence in `computeColWidths`

**Files:**
- Modify: `grid/grid.go` (`computeColWidths`)
- Test: `grid/grid_test.go`

The field is already declared in Task 5. Now make `computeColWidths` consult it *before* anything else.

- [ ] **Step 1: Write the failing tests**

Append to `grid/grid_test.go`:

```go
func TestComputeColWidths_ManualOverride_WinsOverWidth(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "Name", Value: func(r TestRow) any { return r.Name }, Width: 50},
		{ColumnID: "Dept", HeaderName: "Dept", Value: func(r TestRow) any { return r.Department }, MinWidth: 1, Flex: 1},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := newTestGrid(WithColumns[TestRow](cols), WithStyles[TestRow](s))
	m.manualWidths = map[string]int{"Name": 7}
	m.computeColWidths()
	if m.colWidths[0] != 7 {
		t.Errorf("override width = %d, want 7 (override should win over Width=50)", m.colWidths[0])
	}
}

func TestComputeColWidths_ManualOverride_WinsOverAutoFit(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "Name", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, AutoFit: true},
		{ColumnID: "Dept", HeaderName: "Dept", Value: func(r TestRow) any { return r.Department }, MinWidth: 1, Flex: 1},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := newTestGrid(WithColumns[TestRow](cols), WithStyles[TestRow](s))
	m.manualWidths = map[string]int{"Name": 9}
	m.computeColWidths()
	if m.colWidths[0] != 9 {
		t.Errorf("override width = %d, want 9 (override should win over AutoFit)", m.colWidths[0])
	}
}

func TestComputeColWidths_ManualOverride_WinsOverFlex(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "Name", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, Flex: 3},
		{ColumnID: "Dept", HeaderName: "Dept", Value: func(r TestRow) any { return r.Department }, MinWidth: 1, Flex: 1},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := newTestGrid(WithColumns[TestRow](cols), WithStyles[TestRow](s))
	m.manualWidths = map[string]int{"Name": 6}
	m.computeColWidths()
	if m.colWidths[0] != 6 {
		t.Errorf("override width = %d, want 6", m.colWidths[0])
	}
	// Dept should now absorb what Name would have taken
	if m.colWidths[1] != 80-6 {
		t.Errorf("Dept column = %d, want %d", m.colWidths[1], 80-6)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./grid/ -run TestComputeColWidths_ManualOverride -v`
Expected: failures — overrides not yet honored.

- [ ] **Step 3: Add override precedence to `computeColWidths`**

In `grid/grid.go`, update the column-partition switch inside `computeColWidths` so overrides win first:

```go
	for _, idx := range visibleCols {
		col := m.cols[idx]
		if w, ok := m.manualWidths[col.ColumnID]; ok {
			m.colWidths[idx] = w
			remaining -= w
			continue
		}
		switch {
		case col.Width > 0:
			m.colWidths[idx] = col.Width
			remaining -= col.Width
		case col.AutoFit:
			w := m.measureColumnWidth(col, idx, m.rows)
			m.colWidths[idx] = w
			remaining -= w
		default:
			minW := col.MinWidth
			if minW == 0 {
				minW = 4
			}
			flex := col.Flex
			if flex == 0 {
				flex = 1
			}
			flexCols = append(flexCols, colInfo{
				idx:  idx,
				flex: flex,
				min:  minW,
				max:  col.MaxWidth,
			})
		}
	}
```

- [ ] **Step 4: Run override tests**

Run: `go test ./grid/ -run TestComputeColWidths_ManualOverride -v`
Expected: all PASS.

- [ ] **Step 5: Run the full test suite**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add grid/grid.go grid/grid_test.go
git commit -m "grid: honor manualWidths override as highest precedence in computeColWidths"
```

---

## Task 7: Implement `AutoSizeColumn(s)` and `ResetColumnWidth(s)`

**Files:**
- Modify: `grid/grid.go`
- Test: `grid/grid_test.go`

Add the four public methods. `AutoSize*` measures `m.displayRows` and writes to `manualWidths`. `Reset*` removes entries.

- [ ] **Step 1: Write the failing tests**

Append to `grid/grid_test.go`:

```go
func TestAutoSizeColumns_MeasuresDisplayedRows(t *testing.T) {
	cols := []data.Column[TestRow]{
		{
			ColumnID:   "Name",
			HeaderName: "N",
			Value:      func(r TestRow) any { return r.Name },
			Filter:     filter.NewTextFilter(),
			MinWidth:   1,
			Flex:       1,
		},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := New(
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{Name: "Al"}, {Name: "Bartholomew"}}),
		WithWidth[TestRow](80),
		WithStyles[TestRow](s),
	)
	// Filter to only short names.
	tf := filter.NewTextFilter()
	tf.SetText("Al")
	m.SetColumnFilter("Name", tf)
	m.AutoSizeColumns()
	if got := m.colWidths[0]; got != 2 {
		t.Errorf("AutoSizeColumns measured against displayed rows, expected 2, got %d", got)
	}
}

func TestAutoSizeColumn_Single(t *testing.T) {
	cols := testCols()
	s := DefaultStyles()
	s.BorderColumn = false
	m := newTestGrid(WithColumns[TestRow](cols), WithStyles[TestRow](s))
	deptBefore := m.colWidths[1]
	m.AutoSizeColumn("Name")
	if _, ok := m.manualWidths["Name"]; !ok {
		t.Error("expected manualWidths[\"Name\"] to be set")
	}
	if _, ok := m.manualWidths["Department"]; ok {
		t.Error("only Name should have an override")
	}
	// Dept column should still be non-zero (flex takes whatever's left)
	if m.colWidths[1] == 0 {
		t.Errorf("Dept column should retain flex width, got %d (before: %d)", m.colWidths[1], deptBefore)
	}
}

func TestAutoSizeColumn_UnknownID_IsNoop(t *testing.T) {
	m := newTestGrid()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic for unknown ID, got: %v", r)
		}
	}()
	m.AutoSizeColumn("not-a-real-column")
	if len(m.manualWidths) != 0 {
		t.Errorf("expected no overrides, got %v", m.manualWidths)
	}
}

func TestResetColumnWidth_RevertsToDeclared(t *testing.T) {
	cols := testCols()
	s := DefaultStyles()
	s.BorderColumn = false
	m := newTestGrid(WithColumns[TestRow](cols), WithStyles[TestRow](s))
	m.AutoSizeColumn("Name")
	before := m.colWidths[0]
	m.ResetColumnWidth("Name")
	if _, ok := m.manualWidths["Name"]; ok {
		t.Error("expected override cleared")
	}
	if m.colWidths[0] == before {
		t.Errorf("expected width to change after reset (flex should kick back in); got %d (same as override)", m.colWidths[0])
	}
}

func TestResetColumnWidths_ClearsAll(t *testing.T) {
	m := newTestGrid()
	m.AutoSizeColumn("Name")
	m.AutoSizeColumn("Department")
	if len(m.manualWidths) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(m.manualWidths))
	}
	m.ResetColumnWidths()
	if len(m.manualWidths) != 0 {
		t.Errorf("expected 0 overrides, got %d", len(m.manualWidths))
	}
}

func TestAutoSizeColumns_StickyAcrossSetRows(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "N", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, Flex: 1},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := New(
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{Name: "Al"}}),
		WithWidth[TestRow](80),
		WithStyles[TestRow](s),
	)
	m.AutoSizeColumns()
	before := m.colWidths[0]
	m.SetRows([]TestRow{{Name: "Bartholomew"}})
	if m.colWidths[0] != before {
		t.Errorf("override should persist across SetRows: %d -> %d", before, m.colWidths[0])
	}
}

func TestAutoSizeColumns_StickyAcrossFilter(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "N", Value: func(r TestRow) any { return r.Name }, Filter: filter.NewTextFilter(), MinWidth: 1, Flex: 1},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := New(
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{Name: "Alice"}, {Name: "Bartholomew"}}),
		WithWidth[TestRow](80),
		WithStyles[TestRow](s),
	)
	m.AutoSizeColumns()
	before := m.colWidths[0]
	tf := filter.NewTextFilter()
	tf.SetText("Alice")
	m.SetColumnFilter("Name", tf)
	if m.colWidths[0] != before {
		t.Errorf("override should persist across filter: %d -> %d", before, m.colWidths[0])
	}
}

func TestAutoSizeColumns_RespectsMinMaxWidth(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "N", Value: func(r TestRow) any { return r.Name }, MinWidth: 10, MaxWidth: 20, Flex: 1},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := New(
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{Name: "Al"}}),
		WithWidth[TestRow](80),
		WithStyles[TestRow](s),
	)
	m.AutoSizeColumns()
	if got := m.manualWidths["Name"]; got != 10 {
		t.Errorf("override clamped to MinWidth=10, got %d", got)
	}
}

func TestAutoSizeColumns_NoRows(t *testing.T) {
	cols := []data.Column[TestRow]{
		{ColumnID: "Name", HeaderName: "LongerHeader", Value: func(r TestRow) any { return r.Name }, MinWidth: 1, Flex: 1},
	}
	s := DefaultStyles()
	s.BorderColumn = false
	m := New(
		WithColumns[TestRow](cols),
		WithWidth[TestRow](80),
		WithStyles[TestRow](s),
	)
	m.AutoSizeColumns()
	if got := m.manualWidths["Name"]; got != len("LongerHeader") {
		t.Errorf("no-rows AutoSize should equal header width %d, got %d", len("LongerHeader"), got)
	}
}

func TestAutoSizeColumns_SkipsHidden(t *testing.T) {
	cols := testCols()
	cols[0].Hide = true
	m := newTestGrid(WithColumns[TestRow](cols))
	m.AutoSizeColumns()
	if _, ok := m.manualWidths["Name"]; ok {
		t.Error("hidden column should not be measured by AutoSizeColumns")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./grid/ -run "TestAutoSize|TestResetColumnWidth" -v`
Expected: compile error — methods undefined.

- [ ] **Step 3: Implement the four methods**

In `grid/grid.go`, add after the existing data setters (e.g. after `RemoveRow`, around line 280) — but before `computeColWidths` is fine too. A good home is a new `// --- Column widths ---` section near the top-level data methods. Add:

```go
// --- Column widths ---

// AutoSizeColumns re-measures every non-hidden column against the currently
// displayed rows (post-filter, post-sort) and stores the clamped widths as
// sticky overrides. Overrides win over Width, AutoFit, and Flex until
// cleared by ResetColumnWidth(s).
func (m *Model[T]) AutoSizeColumns() {
	if m.manualWidths == nil {
		m.manualWidths = make(map[string]int)
	}
	for i := range m.cols {
		if m.cols[i].Hide {
			continue
		}
		m.manualWidths[m.cols[i].ColumnID] = m.measureColumnWidth(m.cols[i], i, m.displayRows)
	}
	m.computeColWidths()
}

// AutoSizeColumn re-measures a single column against displayed rows and
// records a sticky override. A colID that doesn't match any column is
// silently ignored.
func (m *Model[T]) AutoSizeColumn(colID string) {
	for i := range m.cols {
		if m.cols[i].ColumnID != colID {
			continue
		}
		if m.cols[i].Hide {
			return
		}
		if m.manualWidths == nil {
			m.manualWidths = make(map[string]int)
		}
		m.manualWidths[colID] = m.measureColumnWidth(m.cols[i], i, m.displayRows)
		m.computeColWidths()
		return
	}
}

// ResetColumnWidths clears all sticky width overrides. Columns revert to
// their declared sizing (Width / AutoFit / Flex).
func (m *Model[T]) ResetColumnWidths() {
	if len(m.manualWidths) == 0 {
		return
	}
	m.manualWidths = nil
	m.computeColWidths()
}

// ResetColumnWidth clears the sticky override for one column.
func (m *Model[T]) ResetColumnWidth(colID string) {
	if _, ok := m.manualWidths[colID]; !ok {
		return
	}
	delete(m.manualWidths, colID)
	m.computeColWidths()
}
```

- [ ] **Step 4: Run new tests**

Run: `go test ./grid/ -run "TestAutoSize|TestResetColumnWidth" -v`
Expected: all PASS.

- [ ] **Step 5: Run the full test suite**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add grid/grid.go grid/grid_test.go
git commit -m "grid: add AutoSizeColumn(s) and ResetColumnWidth(s) methods"
```

---

## Task 8: Prune `manualWidths` in `SetColumns`

**Files:**
- Modify: `grid/grid.go:227-233` (`SetColumns`)
- Test: `grid/grid_test.go`

When columns are replaced, drop override entries whose `ColumnID` is no longer present. Keep entries whose ID is still around.

- [ ] **Step 1: Write the failing tests**

Append to `grid/grid_test.go`:

```go
func TestSetColumns_PrunesRemovedOverrides(t *testing.T) {
	m := newTestGrid()
	m.AutoSizeColumn("Name")
	m.AutoSizeColumn("Department")
	// Replace with a set that drops "Name"
	newCols := []data.Column[TestRow]{
		{ColumnID: "Department", HeaderName: "Dept", Value: func(r TestRow) any { return r.Department }, MinWidth: 1, Flex: 1},
	}
	m.SetColumns(newCols)
	if _, ok := m.manualWidths["Name"]; ok {
		t.Error("expected override for removed column \"Name\" to be pruned")
	}
	if _, ok := m.manualWidths["Department"]; !ok {
		t.Error("expected override for surviving column \"Department\" to persist")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./grid/ -run TestSetColumns_PrunesRemovedOverrides -v`
Expected: FAIL — `Name` is still in the map.

- [ ] **Step 3: Update `SetColumns`**

In `grid/grid.go:227-233`, modify `SetColumns`:

```go
func (m *Model[T]) SetColumns(cols []data.Column[T]) {
	m.cols = cols
	m.dirty = true
	m.filterDirty = true

	// Prune overrides for columns that no longer exist.
	if len(m.manualWidths) > 0 {
		present := make(map[string]struct{}, len(cols))
		for i := range cols {
			present[cols[i].ColumnID] = struct{}{}
		}
		for id := range m.manualWidths {
			if _, ok := present[id]; !ok {
				delete(m.manualWidths, id)
			}
		}
	}

	m.refreshHasAutoFit()
	m.recomputeDisplayRows()
	m.computeColWidths()
}
```

- [ ] **Step 4: Run the test**

Run: `go test ./grid/ -run TestSetColumns_PrunesRemovedOverrides -v`
Expected: PASS.

- [ ] **Step 5: Run the full test suite**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add grid/grid.go grid/grid_test.go
git commit -m "grid: prune manualWidths for removed columns in SetColumns"
```

---

## Task 9: Add `NaturalWidth` to width-adaptive built-in renderers

**Files:**
- Modify: `data/cell_builtin.go`
- Test: `data/cell_builtin_test.go`

`BarRenderer` and `ProgressRenderer` gain a `PreferredWidth int` field defaulting to 10. `SparklineRenderer` returns `len(values)` when value is `[]float64`.

- [ ] **Step 1: Write failing tests**

Append to `data/cell_builtin_test.go`:

```go
func TestBarRenderer_NaturalWidth_Default(t *testing.T) {
	r := BarRenderer[int]{MaxValue: 100}
	var _ NaturalWidthRenderer[int] = r
	if got := r.NaturalWidth(CellContext[int]{}); got != 10 {
		t.Errorf("default PreferredWidth = %d, want 10", got)
	}
}

func TestBarRenderer_NaturalWidth_Configured(t *testing.T) {
	r := BarRenderer[int]{MaxValue: 100, PreferredWidth: 25}
	if got := r.NaturalWidth(CellContext[int]{}); got != 25 {
		t.Errorf("NaturalWidth = %d, want 25", got)
	}
}

func TestProgressRenderer_NaturalWidth_Default(t *testing.T) {
	r := ProgressRenderer[int]{MaxValue: 100}
	var _ NaturalWidthRenderer[int] = r
	if got := r.NaturalWidth(CellContext[int]{}); got != 10 {
		t.Errorf("default PreferredWidth = %d, want 10", got)
	}
}

func TestProgressRenderer_NaturalWidth_Configured(t *testing.T) {
	r := ProgressRenderer[int]{MaxValue: 100, PreferredWidth: 15}
	if got := r.NaturalWidth(CellContext[int]{}); got != 15 {
		t.Errorf("NaturalWidth = %d, want 15", got)
	}
}

func TestSparklineRenderer_NaturalWidth_FromSlice(t *testing.T) {
	r := SparklineRenderer[int]{}
	var _ NaturalWidthRenderer[int] = r
	ctx := CellContext[int]{Value: []float64{1, 2, 3, 4, 5}}
	if got := r.NaturalWidth(ctx); got != 5 {
		t.Errorf("NaturalWidth = %d, want 5", got)
	}
}

func TestSparklineRenderer_NaturalWidth_NonSlice(t *testing.T) {
	r := SparklineRenderer[int]{}
	ctx := CellContext[int]{Value: 42}
	if got := r.NaturalWidth(ctx); got != 10 {
		t.Errorf("non-slice default = %d, want 10", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./data/ -run "TestBarRenderer_NaturalWidth|TestProgressRenderer_NaturalWidth|TestSparklineRenderer_NaturalWidth" -v`
Expected: compile error — `PreferredWidth` undefined and `NaturalWidth` method missing.

- [ ] **Step 3: Add `PreferredWidth` and `NaturalWidth` to `BarRenderer`**

In `data/cell_builtin.go`, find the `BarRenderer` struct (around line 68-73) and add the field, plus a method after `Render`:

```go
// BarRenderer renders a horizontal bar proportional to value.
type BarRenderer[T any] struct {
	MaxValue       float64
	BarChar        string
	Style          lipgloss.Style
	PreferredWidth int // Natural width for AutoFit. Default: 10.
}
```

After the `BarRenderer.Render` method (around line 100), add:

```go
// NaturalWidth reports the preferred width for AutoFit.
func (r BarRenderer[T]) NaturalWidth(ctx CellContext[T]) int {
	if r.PreferredWidth > 0 {
		return r.PreferredWidth
	}
	return 10
}
```

- [ ] **Step 4: Add `PreferredWidth` and `NaturalWidth` to `ProgressRenderer`**

In `data/cell_builtin.go`, update `ProgressRenderer` (around line 167-172):

```go
// ProgressRenderer renders a mini progress bar within the cell.
type ProgressRenderer[T any] struct {
	MaxValue       float64
	FilledChar     string
	EmptyChar      string
	PreferredWidth int // Natural width for AutoFit. Default: 10.
}
```

After the `ProgressRenderer.Render` method (around line 200), add:

```go
// NaturalWidth reports the preferred width for AutoFit.
func (r ProgressRenderer[T]) NaturalWidth(ctx CellContext[T]) int {
	if r.PreferredWidth > 0 {
		return r.PreferredWidth
	}
	return 10
}
```

- [ ] **Step 5: Add `NaturalWidth` to `SparklineRenderer`**

After the `SparklineRenderer.Render` method (around line 140), add:

```go
// NaturalWidth reports the number of points in the series, falling back to
// 10 when the value is not a []float64.
func (r SparklineRenderer[T]) NaturalWidth(ctx CellContext[T]) int {
	if values, ok := ctx.Value.([]float64); ok {
		return len(values)
	}
	return 10
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./data/ -run "TestBarRenderer_NaturalWidth|TestProgressRenderer_NaturalWidth|TestSparklineRenderer_NaturalWidth" -v`
Expected: all PASS.

- [ ] **Step 7: Run the full test suite**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
git add data/cell_builtin.go data/cell_builtin_test.go
git commit -m "data: implement NaturalWidthRenderer on Bar/Progress/Sparkline"
```

---

## Task 10: Add `w` / `W` keybindings and dispatch

**Files:**
- Modify: `grid/keymap.go`, `grid/update.go`
- Test: `grid/grid_test.go`

Add two new `KeyMap` fields with defaults `w` / `W`. Dispatch them from `handleKeyMsg`. Mode gating is automatic — `handleKeyMsg` isn't reached in edit / filter-edit / quick-filter modes (see `Update` in `update.go:32-44`).

- [ ] **Step 1: Write the failing tests**

Append to `grid/grid_test.go`:

```go
func TestKeyMap_AutoSizeColumn_FitsFocusedColumn(t *testing.T) {
	m := newTestGrid()
	// Focus column index 0 ("Name")
	m.focusedCell = CellPosition{Row: 0, Col: 0}
	m = sendKey(m, tea.KeyPressMsg{Code: 'w', Text: "w"})
	if _, ok := m.manualWidths["Name"]; !ok {
		t.Error("expected 'w' to set override on focused column")
	}
	if _, ok := m.manualWidths["Department"]; ok {
		t.Error("expected 'w' to not affect other columns")
	}
}

func TestKeyMap_AutoSizeColumns_FitsAll(t *testing.T) {
	m := newTestGrid()
	m = sendKey(m, tea.KeyPressMsg{Code: 'W', Text: "W"})
	// All non-hidden columns should have overrides
	for _, c := range m.cols {
		if c.Hide {
			continue
		}
		if _, ok := m.manualWidths[c.ColumnID]; !ok {
			t.Errorf("expected override for column %q", c.ColumnID)
		}
	}
}

func TestKeyMap_AutoSize_NotDispatchedInEditMode(t *testing.T) {
	cols := testCols()
	cols[0].Editable = true
	cols[0].ValueSetter = func(r *TestRow, v any) { r.Name = v.(string) }
	m := newTestGrid(WithColumns[TestRow](cols), WithEditable[TestRow](true))
	m.focusedCell = CellPosition{Row: 0, Col: 0}
	// Enter edit mode
	m = sendKey(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.editState == nil {
		t.Fatal("expected edit mode after Enter")
	}
	// 'w' should be consumed by editor, not trigger AutoSize
	m = sendKey(m, tea.KeyPressMsg{Code: 'w', Text: "w"})
	if _, ok := m.manualWidths["Name"]; ok {
		t.Error("'w' in edit mode must not trigger AutoSize")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./grid/ -run "TestKeyMap_AutoSize" -v`
Expected: `m.manualWidths["Name"]` isn't set because there's no dispatch yet.

- [ ] **Step 3: Add KeyMap fields**

In `grid/keymap.go`, find the `// General` / `Help` block at the end of the `KeyMap` struct (around lines 63-64). Replace it with a new `Column sizing` block followed by the existing `General` block:

```go
	// Column sizing
	AutoSizeColumn  key.Binding // Fit focused column to content.
	AutoSizeColumns key.Binding // Fit all visible columns to content.

	// General
	Help key.Binding
```

`Help` must remain the last field in the struct.

- [ ] **Step 4: Add defaults**

In `DefaultKeyMap()` (`grid/keymap.go:69`), add bindings before the `Help` binding at the end of the returned struct:

```go
		AutoSizeColumn: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "fit column width"),
		),
		AutoSizeColumns: key.NewBinding(
			key.WithKeys("W"),
			key.WithHelp("W", "fit all column widths"),
		),
```

- [ ] **Step 5: Dispatch in `handleKeyMsg`**

In `grid/update.go`, locate the `SortColumn` / `MultiSortColumn` block (around lines 109-133). Add new cases *after* `MultiSortColumn`:

```go
	case key.Matches(msg, m.KeyMap.AutoSizeColumn):
		if m.focusedCell.Col >= 0 && m.focusedCell.Col < len(m.cols) {
			m.AutoSizeColumn(m.cols[m.focusedCell.Col].ColumnID)
		}
		return m, nil

	case key.Matches(msg, m.KeyMap.AutoSizeColumns):
		m.AutoSizeColumns()
		return m, nil
```

- [ ] **Step 6: Run the keybinding tests**

Run: `go test ./grid/ -run "TestKeyMap_AutoSize" -v`
Expected: all PASS.

- [ ] **Step 7: Run the full test suite**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
git add grid/keymap.go grid/update.go grid/grid_test.go
git commit -m "grid: add w/W keybindings for AutoSizeColumn(s)"
```

---

## Task 11: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

Add the new sizing modes, API, and keybindings to the architecture notes.

- [ ] **Step 1: Read the sizing section**

Run: open `CLAUDE.md` and locate the "Column Width Algorithm" section (around line 58-62 in the current file).

- [ ] **Step 2: Replace the section**

Replace the "Column Width Algorithm" section with:

```markdown
### Column Width Algorithm

In `grid.go:computeColWidths`, column sizing precedence is:

1. **Sticky override** set by `AutoSizeColumn(s)` (stored in `Model.manualWidths`), cleared by `ResetColumnWidth(s)`.
2. `Column.Width > 0` — fixed width.
3. `Column.AutoFit = true` — content-measured against raw rows (`m.rows`), clamped to `[MinWidth, MaxWidth]`. Recomputed on `New`/`SetColumns`/`SetRows`/`InsertRow`/`RemoveRow`/`UpdateRow`/`SetWidth`/pin changes; stable under filter/sort/scroll/selection/edit. Cached via `Model.hasAutoFit` to avoid measurement cost on grids with no AutoFit columns.
4. `Column.Flex` — remaining space distributed by weight (cumulative division to avoid pixel loss).

Auto-fit measurement uses a `NaturalWidthRenderer[T]` sub-interface on `CellRenderer[T]`. Renderers that scale to `ctx.Width` (`BarRenderer`, `ProgressRenderer`, `SparklineRenderer`) implement it; text-ish renderers fall back to the `Text → ValueFormatter(Value) → SprintValue(Value)` chain.

Imperative methods on `Model[T]` (measured against `m.displayRows`, sticky until cleared):
- `AutoSizeColumns()` — fit every visible column.
- `AutoSizeColumn(colID)` — fit one column.
- `ResetColumnWidths()` / `ResetColumnWidth(colID)` — clear overrides.

Default keybindings: `w` (fit focused column) and `W` (fit all). `ResetColumnWidth(s)` have no default binding.

MinWidth (default 4) and MaxWidth are respected in all paths. Hidden columns get width 0. Border separators subtract from available space when `BorderColumn=true`.
```

- [ ] **Step 3: Verify the file parses**

Run: `go test ./...` (sanity) — expected all PASS.

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: document AutoFit, AutoSizeColumn(s), and w/W keybindings"
```

---

## Final Verification

- [ ] **Step 1: Run all tests with race detector**

Run: `go test -race ./...`
Expected: all PASS.

- [ ] **Step 2: Run linters**

Run: `go vet ./...`
Expected: no errors.

If `golangci-lint` is available:
Run: `golangci-lint run`
Expected: clean.

- [ ] **Step 3: Run gofumpt**

Run: `gofumpt -l .`
Expected: no output (all files are formatted).

If there are changes:
Run: `gofumpt -w .`
Then: `git add -A && git commit -m "style: gofumpt"`

- [ ] **Step 4: Manual smoke test via an example**

Run: `go run ./examples/basic/`
Check that the grid still renders correctly (no visual regressions).

Press `w` and `W` to verify the new bindings work interactively.

Exit with `q` or `ctrl+c`.

- [ ] **Step 5: Final commit sweep**

Run: `git log --oneline main..HEAD`
Expected: ~11 commits, each focused on one task.
