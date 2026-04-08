# Performance Fixes Implementation Plan (#6–#10)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix five performance issues in the display row pipeline, targeting redundant computation, excessive allocations, and unnecessary copies at 100K+ row scale.

**Architecture:** Each fix is independent and committed separately. They proceed from simplest to most complex: remove redundant call (#6), pre-compute active filters (#10), pool builder in quick filter (#8), switch to pointer slices (#9), cache filtered results (#7). Existing benchmarks in `grid/bench_test.go`, `grouping/bench_test.go`, and `filter/bench_test.go` validate each improvement.

**Tech Stack:** Go 1.25, Bubble Tea v2 (charm.land)

---

### Task 1: Remove redundant first recomputeDisplayRows() call in Update() (#6)

**Files:**
- Modify: `grid/update.go:16`

The first call at line 16 of `Update()` is always a no-op: `dirty` is false at entry because every public setter eagerly calls `recomputeDisplayRows()` which clears `dirty`. The second call at line 34 handles any state changes made during message dispatch. Removing the first call eliminates one function call overhead per Update.

- [ ] **Step 1: Remove the first recomputeDisplayRows() call**

In `grid/update.go`, delete line 16:

```go
// Before:
func (m Model[T]) Update(msg tea.Msg) (Model[T], tea.Cmd) {
	m.recomputeDisplayRows()

	var cmd tea.Cmd
	switch msg := msg.(type) {

// After:
func (m Model[T]) Update(msg tea.Msg) (Model[T], tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
```

- [ ] **Step 2: Run tests and benchmarks**

```bash
go test ./grid/ -count=1
go test ./grid/ -bench=BenchmarkUpdate_Navigation -benchmem -count=3
```

Expected: All tests pass. Navigation benchmarks unchanged (both calls were no-ops for navigation).

- [ ] **Step 3: Commit**

```bash
git add grid/update.go
git commit -m "perf: remove redundant first recomputeDisplayRows() call in Update()

Fixes #6. The first call was always a no-op since dirty=false at
Update() entry — every public setter eagerly recomputes."
```

---

### Task 2: Pre-compute active column filter list (#10)

**Files:**
- Modify: `grid/grid.go` (Model struct, recomputeDisplayRows, passesColumnFilters)

`passesColumnFilters()` iterates all columns and calls `Filter.Active()` per column per row. With 100K rows and 8 columns, that's 800K `Active()` calls even when no filters are set. Pre-compute a slice of columns with active filters; short-circuit when empty.

- [ ] **Step 1: Add activeFilters field and rebuild function**

In `grid/grid.go`, add a field to the Model struct (near the other filter fields around line 68):

```go
// Cached list of columns with active filters (rebuilt when dirty)
activeFilters []int // indices into m.cols
```

Add a method to rebuild the cache:

```go
// rebuildActiveFilters updates the cached list of column indices with active filters.
func (m *Model[T]) rebuildActiveFilters() {
	m.activeFilters = m.activeFilters[:0]
	for i, col := range m.cols {
		if i == m.filterEditColIdx {
			continue
		}
		if col.Filter != nil && col.Filter.Active() {
			m.activeFilters = append(m.activeFilters, i)
		}
	}
}
```

- [ ] **Step 2: Call rebuildActiveFilters at the top of recomputeDisplayRows**

In `recomputeDisplayRows()`, add the call after the dirty check (around line 807):

```go
func (m *Model[T]) recomputeDisplayRows() {
	if !m.dirty && m.displayRows != nil {
		return
	}

	m.rebuildActiveFilters()

	// Start with all rows
	filtered := make([]data.RowNode[T], 0, len(m.rows))
```

- [ ] **Step 3: Rewrite passesColumnFilters to use the cached list**

Replace the current implementation:

```go
func (m *Model[T]) passesColumnFilters(data T) bool {
	for _, i := range m.activeFilters {
		col := m.cols[i]
		val := col.ValueGetter(data)
		if !col.Filter.Matches(val) {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run tests and benchmarks**

```bash
go test ./grid/ -count=1
go test ./grid/ -bench=BenchmarkRecomputeDisplayRows_ColumnFilters -benchmem -count=3
```

Expected: All tests pass. `ColumnFilters_NoneActive` benchmarks show significant improvement (no longer iterating all columns per row). `ColumnFilters_OneActive` and `ColumnFilters_AllActive` should show modest improvement from avoiding repeated `Active()` calls.

- [ ] **Step 5: Commit**

```bash
git add grid/grid.go
git commit -m "perf: pre-compute active column filter list to avoid per-row iteration

Fixes #10. passesColumnFilters() now iterates only columns with active
filters. When no filters are active (common case), it returns true
immediately without touching any columns."
```

---

### Task 3: Pool strings.Builder and cache word split in passesQuickFilter (#8)

**Files:**
- Modify: `grid/grid.go` (Model struct, recomputeDisplayRows, passesQuickFilter)

`passesQuickFilter()` allocates a new `strings.Builder` per row and recomputes `strings.Fields(strings.ToLower(m.quickFilterText))` per row. With 100K rows, this is 100K builder allocations and 100K redundant word splits.

- [ ] **Step 1: Add quickFilterBuilder field to Model**

In `grid/grid.go`, add a field to the Model struct (near the other filter fields around line 68):

```go
quickFilterBuilder strings.Builder // reused across rows in passesQuickFilter
```

- [ ] **Step 2: Pre-compute words and pass to passesQuickFilter**

In `recomputeDisplayRows()`, compute the words once before the filter loop (around line 808, after the `rebuildActiveFilters()` call):

```go
	m.rebuildActiveFilters()

	// Pre-compute quick filter words once (not per row).
	var quickFilterWords []string
	if m.quickFilterText != "" {
		quickFilterWords = strings.Fields(strings.ToLower(m.quickFilterText))
	}
```

Update the quick filter call in the loop (around line 851):

```go
		// Apply quick filter
		if len(quickFilterWords) > 0 && !m.passesQuickFilter(rn.Data, quickFilterWords) {
			continue
		}
```

- [ ] **Step 3: Rewrite passesQuickFilter to reuse builder and accept pre-split words**

```go
func (m *Model[T]) passesQuickFilter(data T, words []string) bool {
	m.quickFilterBuilder.Reset()
	for _, col := range m.cols {
		if col.ValueGetter == nil {
			continue
		}
		val := col.ValueGetter(data)
		fmt.Fprintf(&m.quickFilterBuilder, " %v", val)
	}
	lower := strings.ToLower(m.quickFilterBuilder.String())

	for _, word := range words {
		if !strings.Contains(lower, word) {
			return false
		}
	}
	return true
}
```

The builder's internal buffer grows on first use and is reused for subsequent rows via `Reset()`, avoiding per-row allocation. The `words` slice is computed once per recompute pass.

- [ ] **Step 4: Run tests and benchmarks**

```bash
go test ./grid/ -count=1
go test ./grid/ -bench=BenchmarkRecomputeDisplayRows_WithQuickFilter -benchmem -count=3
go test ./grid/ -bench=BenchmarkRecomputeDisplayRows_QuickFilter_MultiWord -benchmem -count=3
```

Expected: All tests pass. Quick filter benchmarks show reduced allocations (from ~12 allocs/row down to ~10 allocs/row — the `fmt.Fprintf` and `strings.ToLower` still allocate, but the builder allocation is eliminated).

- [ ] **Step 5: Commit**

```bash
git add grid/grid.go
git commit -m "perf: reuse strings.Builder and cache word split in passesQuickFilter

Fixes #8. The builder is now a field on Model, reset per row instead
of allocated. The quick filter words are split once per recompute pass
instead of per row."
```

---

### Task 4: Change displayRows to pointer slice and fix FlattenGroups value copies (#9)

**Files:**
- Modify: `grid/grid.go` (Model struct, recomputeDisplayRows, sortRows, precomputeAggValues, rowHeightFunc, selectedRowNodes, SelectedRowNodes, SelectedRows, FocusedRowData, and multiple accessor methods)
- Modify: `grid/update.go` (all displayRows accesses)
- Modify: `grid/render.go` (renderRow calls, collectLeafValues)
- Modify: `grid/options.go` (WithPostSort signature)
- Modify: `grid/messages.go` (SelectionChangedMsg.Selected field)
- Modify: `grid/viewport.go` (if displayRows is used)
- Modify: `grouping/grouping.go` (BuildGroups input, FlattenGroups return type)
- Modify: `grid/grid_test.go` (all displayRows assertions)
- Modify: `grid/bench_test.go` (dirty flag access)
- Modify: `grid/render_golden_test.go` (if displayRows is accessed)
- Modify: `grouping/grouping_test.go` (FlattenGroups assertions)

This is the largest change. `FlattenGroups()` currently copies every `RowNode[T]` by value into the result slice (`result = append(result, *g)`). At 100K rows, this copies ~184MB. Switching to pointer slices eliminates these copies.

#### Sub-task 4a: Change grouping package

- [ ] **Step 1: Change BuildGroups to accept `[]*data.RowNode[T]`**

In `grouping/grouping.go`, change the `BuildGroups` signature and update internals:

```go
func BuildGroups[T any](
	rows []*data.RowNode[T],
	cols []data.Column[T],
	groupCols []string,
	expanded map[string]bool,
	defaultExpanded int,
) []*data.RowNode[T] {
	if len(groupCols) == 0 {
		result := make([]*data.RowNode[T], len(rows))
		copy(result, rows)
		return result
	}

	// Find the column definition for the first group column
	var groupCol *data.Column[T]
	for i := range cols {
		if cols[i].ColumnID == groupCols[0] {
			groupCol = &cols[i]
			break
		}
	}
	if groupCol == nil {
		result := make([]*data.RowNode[T], len(rows))
		copy(result, rows)
		return result
	}

	// Group rows by the column value
	groupMap := make(map[string][]*data.RowNode[T])
	var groupOrder []string
	for _, row := range rows {
		val := groupCol.ValueGetter(row.Data)
		key := fmt.Sprintf("%v", val)
		if _, exists := groupMap[key]; !exists {
			groupOrder = append(groupOrder, key)
		}
		groupMap[key] = append(groupMap[key], row)
	}

	// Create group nodes
	var groups []*data.RowNode[T]
	for _, key := range groupOrder {
		children := groupMap[key]
		groupNode := &data.RowNode[T]{
			IsGroup:    true,
			GroupKey:   key,
			GroupLevel: 0,
			Children:   children,
		}

		// Set expanded state
		if exp, exists := expanded[key]; exists {
			groupNode.Expanded = exp
		} else {
			groupNode.Expanded = defaultExpanded == -1 || defaultExpanded > 0
		}

		// Set parent references
		for _, child := range children {
			child.Parent = groupNode
		}

		// Recursively group children if there are more group columns
		if len(groupCols) > 1 {
			subGroups := BuildGroups(children, cols, groupCols[1:], expanded, defaultExpanded-1)
			groupNode.Children = subGroups
			for _, sg := range subGroups {
				sg.Parent = groupNode
				sg.GroupLevel = 1
			}
		}

		groups = append(groups, groupNode)
	}

	return groups
}
```

Key changes: the input `rows` is now `[]*data.RowNode[T]`, which eliminates the `childRows` copy in the recursive case (previously lines 139-142). Rows are directly passed through.

- [ ] **Step 2: Change FlattenGroups to return `[]*data.RowNode[T]`**

```go
func FlattenGroups[T any](groups []*data.RowNode[T]) []*data.RowNode[T] {
	var result []*data.RowNode[T]
	for _, g := range groups {
		result = append(result, g)
		if g.IsGroup && g.Expanded {
			result = append(result, FlattenGroups(g.Children)...)
		}
	}
	return result
}
```

No more `*g` dereference. Just append the pointer.

- [ ] **Step 3: Update grouping tests**

In `grouping/grouping_test.go`, update all assertions that access `FlattenGroups` results. The results are now pointers, so instead of `result[i].GroupKey` it stays the same (Go auto-dereferences). But if any test compares `result[i]` by value, update to compare fields. Also update `testRows()` to return `[]*data.RowNode[testRow]`:

```go
func testRows() []*data.RowNode[testRow] {
	rows := []testRow{
		{"Alice", "Eng", 100},
		{"Bob", "Eng", 200},
		{"Carol", "Eng", 150},
		{"Dave", "Sales", 120},
		{"Eve", "Sales", 180},
		{"Frank", "Sales", 160},
	}
	nodes := make([]*data.RowNode[testRow], len(rows))
	for i, d := range rows {
		nodes[i] = &data.RowNode[testRow]{Data: d, ID: d.Name}
	}
	return nodes
}
```

Update all test functions to work with the new pointer-based signatures. The `ExpandAll`/`CollapseAll` methods already take `[]*data.RowNode[T]` so those tests are unchanged.

- [ ] **Step 4: Update grouping benchmarks**

In `grouping/bench_test.go`, update `makeBenchRows` to return `[]*data.RowNode[benchRow]`:

```go
func makeBenchRows(n int) []*data.RowNode[benchRow] {
	nodes := make([]*data.RowNode[benchRow], n)
	for i := range nodes {
		nodes[i] = &data.RowNode[benchRow]{
			Data: benchRow{
				Name:       fmt.Sprintf("Person_%d", i),
				Department: benchDepartments[i%len(benchDepartments)],
				City:       benchCities[i%len(benchCities)],
				Salary:     50000 + float64(i%100)*1000,
			},
			ID: fmt.Sprintf("row_%d", i),
		}
	}
	return nodes
}
```

- [ ] **Step 5: Run grouping tests**

```bash
go test ./grouping/ -count=1 -v
```

Expected: All tests pass.

#### Sub-task 4b: Change grid package internal types

- [ ] **Step 6: Change Model field types**

In `grid/grid.go`, change the field declarations:

```go
// Row data
rows      []*data.RowNode[T] // All row nodes
pinnedTop []*data.RowNode[T] // Rows pinned to top
pinnedBot []*data.RowNode[T] // Rows pinned to bottom
```

```go
// Computed display rows (cached)
displayRows []*data.RowNode[T]
```

```go
// Pinning
staticPinnedTopNodes []*data.RowNode[T]
staticPinnedBotNodes []*data.RowNode[T]
```

Change the postSort field and option:

```go
// In grid.go Model struct:
postSort  func([]*data.RowNode[T]) []*data.RowNode[T]
```

```go
// In options.go WithPostSort:
func WithPostSort[T any](fn func([]*data.RowNode[T]) []*data.RowNode[T]) Option[T] {
	return func(m *Model[T]) {
		m.postSort = fn
	}
}
```

Change `SelectionChangedMsg.Selected` in `grid/messages.go`:

```go
type SelectionChangedMsg[T any] struct {
	Regions  []SelectionRegion      // All current selection regions.
	Selected []*data.RowNode[T]     // Rows covered by any selection (convenience).
}
```

- [ ] **Step 7: Update setRowData to produce pointer slice**

In `grid/grid.go`, update `setRowData`:

```go
func (m *Model[T]) setRowData(rows []T) {
	oldRows := m.rows
	m.rows = make([]*data.RowNode[T], len(rows))
	for i, d := range rows {
		rn := &data.RowNode[T]{
			Data:      d,
			RowHeight: m.defaultRowHeight,
		}
		if m.dynamicRowHeight != nil {
			rn.RowHeight = m.dynamicRowHeight(d)
		}
		if m.rowIDFunc != nil {
			rn.ID = m.rowIDFunc(d)
		} else if i < len(oldRows) {
			rn.ID = oldRows[i].ID
		} else {
			rn.ID = fmt.Sprintf("row-%d", m.nextRowID)
			m.nextRowID++
		}
		m.rows[i] = rn
	}
	m.dirty = true
}
```

- [ ] **Step 8: Update recomputeDisplayRows**

Update the filtering loop and grouping/sort paths to use pointer slices:

```go
func (m *Model[T]) recomputeDisplayRows() {
	if !m.dirty && m.displayRows != nil {
		return
	}

	m.rebuildActiveFilters()

	var quickFilterWords []string
	if m.quickFilterText != "" {
		quickFilterWords = strings.Fields(strings.ToLower(m.quickFilterText))
	}

	filtered := make([]*data.RowNode[T], 0, len(m.rows))
	m.pinnedTop = nil
	m.pinnedBot = nil

	for _, rn := range m.rows {
		if rn.Pinned == data.PinTop {
			m.pinnedTop = append(m.pinnedTop, rn)
			continue
		}
		if rn.Pinned == data.PinBottom {
			m.pinnedBot = append(m.pinnedBot, rn)
			continue
		}
		if m.pinnedTopFunc != nil && m.pinnedTopFunc(rn.Data) {
			rn.Pinned = data.PinTop
			m.pinnedTop = append(m.pinnedTop, rn)
			continue
		}
		if m.pinnedBotFunc != nil && m.pinnedBotFunc(rn.Data) {
			rn.Pinned = data.PinBottom
			m.pinnedBot = append(m.pinnedBot, rn)
			continue
		}
		if m.externalFilter != nil && !m.externalFilter(rn.Data) {
			continue
		}
		if !m.passesColumnFilters(rn.Data) {
			continue
		}
		if len(quickFilterWords) > 0 && !m.passesQuickFilter(rn.Data, quickFilterWords) {
			continue
		}
		filtered = append(filtered, rn)
	}

	m.pinnedTop = append(m.pinnedTop, m.staticPinnedTopNodes...)
	m.pinnedBot = append(m.pinnedBot, m.staticPinnedBotNodes...)

	if len(m.groupModel.GroupColumns) > 0 {
		groups := grouping.BuildGroups(filtered, m.cols, m.groupModel.GroupColumns,
			m.groupModel.Expanded, m.groupModel.DefaultExpanded)
		m.sortGroups(groups)
		m.displayRows = grouping.FlattenGroups(groups)
	} else {
		m.sortRows(filtered)
		m.displayRows = filtered
	}

	if m.postSort != nil {
		m.displayRows = m.postSort(m.displayRows)
	}

	for i := range m.displayRows {
		m.displayRows[i].RowIndex = i
	}

	m.precomputeAggValues()
	m.dirty = false
	m.updateViewportSize()
}
```

- [ ] **Step 9: Update sortRows and sortGroupsAtLevel**

```go
func (m *Model[T]) sortRows(rows []*data.RowNode[T]) {
	if len(m.sortModel.SortOrder) == 0 {
		return
	}

	sort.SliceStable(rows, func(i, j int) bool {
		for _, sc := range m.sortModel.SortOrder {
			col := m.findCol(sc.ColumnID)
			if col == nil || col.ValueGetter == nil {
				continue
			}
			a := col.ValueGetter(rows[i].Data)
			b := col.ValueGetter(rows[j].Data)

			var cmp int
			if col.Comparator != nil {
				cmp = col.Comparator(a, b)
			} else {
				cmp = defaultCompare(a, b)
			}

			if sc.Direction == data.SortDesc {
				cmp = -cmp
			}
			if cmp != 0 {
				return cmp < 0
			}
		}
		return false
	})
}
```

Update `sortGroupsAtLevel` — the leaf row sort no longer needs to copy:

```go
	// In sortGroupsAtLevel, replace the leaf row sorting (lines 1038-1046):
	} else {
		// Sort leaf rows directly — no copy needed with pointer slices
		m.sortRows(g.Children)
	}
```

- [ ] **Step 10: Update remaining Model methods**

Update all methods that access `displayRows` elements. Key changes:
- `Rows()`: dereference `rn.Data` → `rn.Data` (no change, Go auto-dereferences pointers)
- `SelectedRows()`: range over pointer slice, access `.Data` (auto-deref)
- `SelectedRowNodes()`: return `m.displayRows[i]` directly (already a pointer)
- `selectedRowNodes()`: return `[]*data.RowNode[T]` (change return type)
- `FocusedRowData()`: `rn := m.displayRows[...]` is now `*data.RowNode[T]`
- `rowHeightFunc()`: `m.displayRows[i].RowHeight` auto-dereferences
- `precomputeAggValues()`: `rn := m.displayRows[i]` is already a pointer
- `buildStaticPinnedNodes()`: produce `[]*data.RowNode[T]`
- `SetRowSelection`, `ToggleRowSelection`, `ScrollToRowByID`: range over pointer slice

- [ ] **Step 11: Update render.go**

In `View()`, pointer slice means `&m.displayRows[i]` becomes `m.displayRows[i]`:

```go
	// Pinned top rows
	for _, rn := range m.pinnedTop {
		sections = append(sections, m.renderRow(rn, -1, true))
	}

	// Body rows
	for i := start; i < end; i++ {
		sections = append(sections, m.renderRow(m.displayRows[i], i, false))
	}

	// Pinned bottom rows
	for _, rn := range m.pinnedBot {
		sections = append(sections, m.renderRow(rn, -1, true))
	}
```

- [ ] **Step 12: Update update.go**

All `m.displayRows[i]` accesses now yield `*data.RowNode[T]` — update accordingly:
- `handleKeyMsg` line 229: `rn := m.displayRows[...]` is now a pointer (no `&` needed)
- `handleEditKeyMsg` line 316: `rn := m.displayRows[pos.Row]` is now a pointer
- `startEditing` line 585: `rn := m.displayRows[pos.Row]` is now a pointer
- `expandCurrentGroup`, `collapseCurrentGroup`, `toggleCurrentGroup`: same

- [ ] **Step 13: Update grid tests**

This is the most tedious part. In `grid/grid_test.go`:
- All `m.displayRows[i].Field` accesses auto-deref, so most assertions work unchanged
- Any `m.rows[i]` access now returns a pointer — update as needed
- `m.pinnedTop[i]`, `m.pinnedBot[i]` now return pointers
- `SelectionChangedMsg.Selected` is now `[]*data.RowNode[T]`

Update `grid/bench_test.go` if needed (the dirty flag access `m.dirty = true` is on Model, not displayRows, so it should be fine).

- [ ] **Step 14: Run all tests**

```bash
go test ./... -count=1
```

Expected: All tests pass.

- [ ] **Step 15: Run benchmarks to verify improvement**

```bash
go test ./grouping/ -bench=BenchmarkFlattenGroups -benchmem -count=3
go test ./grid/ -bench=BenchmarkRecomputeDisplayRows_GroupFlatten -benchmem -count=3
go test ./grid/ -bench=BenchmarkRecomputeDisplayRows_WithGrouping -benchmem -count=3
```

Expected: `FlattenGroups` at 100K rows drops from ~184MB to <1MB allocations. Grid grouping benchmarks show similar improvements.

- [ ] **Step 16: Commit**

```bash
git add grid/ grouping/
git commit -m "perf: switch displayRows to pointer slice, eliminate FlattenGroups value copies

Fixes #9. displayRows, pinnedTop, pinnedBot are now []*data.RowNode[T].
FlattenGroups and BuildGroups operate on pointer slices, eliminating
per-row struct copies. sortGroupsAtLevel no longer copies leaf rows
for sorting. At 100K rows, FlattenGroups drops from ~184MB to <1MB."
```

---

### Task 5: Cache filtered results across recompute calls (#7)

**Files:**
- Modify: `grid/grid.go` (Model struct, recomputeDisplayRows, all filter state mutators)
- Modify: `grid/update.go` (filter state changes set filterDirty)

The display pipeline runs filter → group → sort → flatten. When only sort/group/expand state changes, the filter pass is wasted. Add a `filterDirty` flag to skip refiltering when filter state hasn't changed.

- [ ] **Step 1: Add filter cache fields to Model**

In `grid/grid.go`, add fields to the Model struct:

```go
// Filter result cache
filterDirty    bool
cachedFiltered []*data.RowNode[T]
```

- [ ] **Step 2: Mark filterDirty at every filter state change**

Every place that sets `m.dirty = true` because of a filter change must ALSO set `m.filterDirty = true`. These are:

In `grid/grid.go`:
- `SetRows()` / `setRowData()`: `m.filterDirty = true` (row data changed, filters must re-run)
- `SetColumns()`: `m.filterDirty = true` (columns changed, filter list may differ)
- `UpdateRow()`: `m.filterDirty = true`
- `InsertRow()`: `m.filterDirty = true`
- `RemoveRow()`: `m.filterDirty = true`
- `SetQuickFilter()`: `m.filterDirty = true`
- `SetColumnFilter()`: `m.filterDirty = true`
- `ClearFilters()`: `m.filterDirty = true`
- `PinRow()`: `m.filterDirty = true` (pin state affects filtering)

In `grid/update.go`:
- `handleKeyMsg` quick filter toggle (line 145): `m.filterDirty = true`
- `handleQuickFilterKeyMsg` Escape/Backspace/Space/char (lines 367, 377, 384, 396): `m.filterDirty = true`
- `handleFilterEditKeyMsg` Enter/Escape (lines 419, 431): `m.filterDirty = true`
- `handleEditKeyMsg` confirm edit (line 328): `m.filterDirty = true` (edited data may affect filter match)

Places that set `m.dirty = true` but NOT `m.filterDirty`:
- `handleKeyMsg` sort toggle/multi-sort (lines 96, 109, 202, 215): sort change only
- `handleKeyMsg` group column toggle (line 122): grouping change only
- `expandCurrentGroup` / `collapseCurrentGroup` (lines 644, 662): expand/collapse only

Initialize `filterDirty = true` in `New()` (alongside `dirty: true`).

- [ ] **Step 3: Update recomputeDisplayRows to use the cache**

```go
func (m *Model[T]) recomputeDisplayRows() {
	if !m.dirty && m.displayRows != nil {
		return
	}

	m.rebuildActiveFilters()

	var filtered []*data.RowNode[T]

	if !m.filterDirty && m.cachedFiltered != nil {
		// Reuse cached filter results — only sort/group/expand changed.
		filtered = m.cachedFiltered
		// Rebuild pinned rows from cache since they depend on row state.
		m.pinnedTop = nil
		m.pinnedBot = nil
		for _, rn := range m.rows {
			if rn.Pinned == data.PinTop {
				m.pinnedTop = append(m.pinnedTop, rn)
			} else if rn.Pinned == data.PinBottom {
				m.pinnedBot = append(m.pinnedBot, rn)
			}
		}
		m.pinnedTop = append(m.pinnedTop, m.staticPinnedTopNodes...)
		m.pinnedBot = append(m.pinnedBot, m.staticPinnedBotNodes...)
	} else {
		// Full filter pass.
		var quickFilterWords []string
		if m.quickFilterText != "" {
			quickFilterWords = strings.Fields(strings.ToLower(m.quickFilterText))
		}

		filtered = make([]*data.RowNode[T], 0, len(m.rows))
		m.pinnedTop = nil
		m.pinnedBot = nil

		for _, rn := range m.rows {
			if rn.Pinned == data.PinTop {
				m.pinnedTop = append(m.pinnedTop, rn)
				continue
			}
			if rn.Pinned == data.PinBottom {
				m.pinnedBot = append(m.pinnedBot, rn)
				continue
			}
			if m.pinnedTopFunc != nil && m.pinnedTopFunc(rn.Data) {
				rn.Pinned = data.PinTop
				m.pinnedTop = append(m.pinnedTop, rn)
				continue
			}
			if m.pinnedBotFunc != nil && m.pinnedBotFunc(rn.Data) {
				rn.Pinned = data.PinBottom
				m.pinnedBot = append(m.pinnedBot, rn)
				continue
			}
			if m.externalFilter != nil && !m.externalFilter(rn.Data) {
				continue
			}
			if !m.passesColumnFilters(rn.Data) {
				continue
			}
			if len(quickFilterWords) > 0 && !m.passesQuickFilter(rn.Data, quickFilterWords) {
				continue
			}
			filtered = append(filtered, rn)
		}

		m.pinnedTop = append(m.pinnedTop, m.staticPinnedTopNodes...)
		m.pinnedBot = append(m.pinnedBot, m.staticPinnedBotNodes...)

		m.cachedFiltered = filtered
		m.filterDirty = false
	}

	// Group / sort / flatten (always runs when dirty)
	if len(m.groupModel.GroupColumns) > 0 {
		groups := grouping.BuildGroups(filtered, m.cols, m.groupModel.GroupColumns,
			m.groupModel.Expanded, m.groupModel.DefaultExpanded)
		m.sortGroups(groups)
		m.displayRows = grouping.FlattenGroups(groups)
	} else {
		m.sortRows(filtered)
		m.displayRows = filtered
	}

	if m.postSort != nil {
		m.displayRows = m.postSort(m.displayRows)
	}

	for i := range m.displayRows {
		m.displayRows[i].RowIndex = i
	}

	m.precomputeAggValues()
	m.dirty = false
	m.updateViewportSize()
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./grid/ -count=1 -v
```

Expected: All tests pass. The cache is transparent to consumers.

- [ ] **Step 5: Run benchmarks to verify improvement**

```bash
go test ./grid/ -bench=BenchmarkRecomputeDisplayRows_WithSort -benchmem -count=3
go test ./grid/ -bench=BenchmarkRecomputeDisplayRows_WithGrouping -benchmem -count=3
```

To properly measure the cache benefit, we need a benchmark that changes sort/group state without changing filters. Add a targeted benchmark:

In `grid/bench_test.go`, add:

```go
// BenchmarkRecomputeDisplayRows_SortChangeOnly measures the cost of recompute
// when only sort state changes (filter results should be cached).
func BenchmarkRecomputeDisplayRows_SortChangeOnly(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			tf := filter.NewTextFilter()
			tf.SetText("eng")
			m := newBenchGrid(rows,
				WithColumnFilter[benchRow]("Department", tf),
				WithQuickFilterText[benchRow]("person"),
				WithDefaultSort[benchRow]([]gridsort.SortCriterion{
					{ColumnID: "Salary", Direction: data.SortAsc},
				}),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				// Simulate sort-only change: dirty=true but filterDirty=false.
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}
```

```bash
go test ./grid/ -bench=BenchmarkRecomputeDisplayRows_SortChangeOnly -benchmem -count=3
```

Expected: Significant reduction in time and allocations compared to `BenchmarkRecomputeDisplayRows_Full` (filter pass is skipped).

- [ ] **Step 6: Commit**

```bash
git add grid/grid.go grid/update.go grid/bench_test.go
git commit -m "perf: cache filtered results across recompute calls

Fixes #7. Adds a filterDirty flag tracked independently from dirty.
When only sort/group/expand state changes, the filter pass is skipped
and cached results are reused. For 100K rows with active filters,
sort-only changes skip 100K+ row filter evaluations."
```

---

### Final verification

- [ ] **Run full test suite with race detector**

```bash
go test -race ./... -count=1
```

- [ ] **Run full benchmark suite and compare to baseline**

```bash
go test ./grid/ ./grouping/ ./filter/ -bench=. -benchmem -count=3
```

Compare against the baseline numbers from the initial commit (f61fce4).
