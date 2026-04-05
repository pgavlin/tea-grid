# tea-grid Performance Report

**Platform:** Apple M4 Max, darwin/arm64, Go 1.25  
**Benchmark configuration:** `-benchtime=5s -count=3`  
**Baseline:** commit f61fce4 (benchmarks only, before any fixes)  
**Current:** commit e151831 (all five fixes applied)

---

## Summary

Five targeted fixes reduced the display pipeline geomean by **93.9%** in time and **97.1%** in memory. The filter cache is the single largest contributor (63–407× speedup on filter-heavy workloads); pointer slices eliminate the dominant memory cost in grouping and sorting. These improvements shift the performance bottleneck from the data pipeline to the render path.

---

## 1. Display Pipeline — Cached Path

These benchmarks measure `recomputeDisplayRows()` with `dirty=true` but `filterDirty=false` (the common case: navigation, sort changes, group expand/collapse). The filter cache is warm after the first iteration.

### Time (100K rows)

| Benchmark | Before | After | Factor |
|---|---|---|---|
| NoFilter | 4,721µs | 404µs | **11.7×** |
| WithColumnFilter | 8,885µs | 120µs | **74×** |
| WithQuickFilter | 59,401µs | 120µs | **495×** |
| WithSort | 83,193µs | 4,147µs | **20×** |
| WithGrouping | 26,048µs | 7,998µs | **3.3×** |
| Full pipeline | 21,489µs | 3,256µs | **6.6×** |

### Memory (100K rows)

| Benchmark | Before | After | Factor |
|---|---|---|---|
| NoFilter | 21 MB | 784 KB | **27×** |
| WithColumnFilter | 24 MB | 80 KB | **300×** |
| WithQuickFilter | 50 MB | 80 KB | **625×** |
| WithSort | 32.7 MB | 2.66 MB | **12×** |
| WithGrouping | 239 MB | 12.6 MB | **19×** |
| Full pipeline | 41 MB | 2.3 MB | **18×** |

### What's driving each result

**Filter-heavy benchmarks (WithColumnFilter, WithQuickFilter):** The 74–495× speedups are almost entirely from the filter cache (#7). Once warm, these benchmarks skip the entire filter pass and simply copy a pre-built pointer slice (~80KB for 10% selectivity at 100K rows). The number of active filters no longer matters in the cached path — `ColumnFilters_OneActive` and `ColumnFilters_AllActive` both land at ~120µs at 100K rows.

**NoFilter:** The 27× memory reduction comes from the pointer slice change (#9) — previously, `FlattenGroups` copied 100K `RowNode` structs by value (21MB); now it copies 100K 8-byte pointers (784KB). The time improvement combines pointer slices (#9) with the filter cache (#7): the cache is trivially valid (no filter to re-evaluate), so the cached path simply copies the pointer slice into `displayRows` without iterating the source rows.

**WithSort:** The 20× speedup comes primarily from pointer slices (#9). `sort.SliceStable` previously swapped 200-byte `RowNode` structs; now it swaps 8-byte pointers. The filter cache (#7) avoids re-running the filter on unchanged data. The combination eliminates the two dominant costs.

**WithGrouping:** The 3.3× speedup and 19× memory reduction come from pointer slices (#9). `BuildGroups` and `FlattenGroups` no longer copy `RowNode` values — they thread pointers through the tree. The grouping traversal itself (tree construction, per-node allocations) remains the bottleneck.

---

## 2. Display Pipeline — Cold Path

These benchmarks force `filterDirty=true` each iteration, bypassing the cache. They measure the true cost of filter evaluation when filter state changes.

### Column filter cold path (100K rows)

| Rows | Time | Memory | Allocs |
|---|---|---|---|
| 1K | 77µs | 44 KB | 3,001 |
| 10K | 783µs | 440 KB | 30,002 |
| 100K | 7.6ms | 4.37 MB | 300,002 |

**3 allocs per row** (filter evaluation overhead). Scales linearly. A cold evaluation at 100K rows is 63× slower than a cache hit (120µs cached vs 7.6ms cold).

### Quick filter cold path (100K rows)

| Rows | Time | Memory | Allocs |
|---|---|---|---|
| 1K | 490µs | 196 KB | 7,992 |
| 10K | 4.98ms | 1.98 MB | 79,906 |
| 100K | 48.9ms | 19.8 MB | 799,008 |

**8 allocs per row** — significantly worse than column filter. The hot path in `passesQuickFilter` calls `fmt.Appendf` per column value and then `strings.ToLower(string(m.quickFilterScratch))`, which allocates two strings per row regardless. At 100K rows, a cold quick filter takes 48.9ms — 407× slower than the cache hit path (120µs). The scratch buffer optimization (#8) eliminated the `strings.Builder` copy-guard hazard and the `strings.Fields` allocation per row, but string conversion remains the dominant allocator.

**Implication for the cache:** The filter cache's value grows with dataset size. At 10K rows, the difference between cold (5ms) and cached (8µs) is already 625×. At 100K rows it is 407× for quick filter and 63× for column filter. Any interaction that does not change filter state benefits immediately.

### Full pipeline cold (100K rows): 15.5ms, 8.4MB, 559K allocs

Sort and grouping dominate here (the filter eliminates ~90% of rows early). The cold vs cached ratio is only ~4.7× because sort and group costs are shared.

---

## 3. Rendering — `View()`

`View()` had no benchmark before this work. It is the primary hot path: called on every frame regardless of whether data changed.

| Configuration | Time | Memory | Allocs |
|---|---|---|---|
| Plain | 1,024µs | 387 KB | 10,191 |
| WithSort | 986µs | 394 KB | 10,307 |
| WithGrouping | 1,005µs | 391 KB | 10,249 |
| WithSelection | 1,140µs | 490 KB | 11,951 |

### Key findings

**View() is the dominant cost at steady state.** At 1ms per frame, rendering is 2.5× more expensive than the display pipeline cache-hit path at 100K rows (404µs). For a typical 60fps TUI, the frame budget is 16.7ms — View() alone consumes 6% of it. With Bubble Tea's synchronous rendering model, any frame that triggers a View() call spends at least 1ms in the renderer.

**10,191 allocations per frame.** With 38 visible rows × 8 columns = 304 cells plus header cells, this is approximately 32 allocations per cell. These come from lipgloss style application: each `Render()` call allocates strings for ANSI escape sequences, padding bytes, and styled output. This is a significant allocation rate for a per-frame operation.

**Sort and grouping have no measurable effect on render cost.** `View_Plain` (1,024µs) and `View_WithGrouping` (1,005µs) are statistically identical — the renderer works only on the visible viewport, and the group header rendering path has the same allocation structure as data row rendering.

**Selection adds meaningful overhead: +116µs (+11%), +103KB (+27%), +1,760 allocs (+17%).** The selection membership check itself (`sel.ContainsCell(row, col int)`) does not allocate — it is a simple range check on plain ints. The extra allocs come from **lipgloss ANSI color rendering**: the `CellSelected` style has `Background(lipgloss.Color("236"))` whereas the default even/odd row styles have no explicit background. Memory profiling shows the increase concentrated in `ansi.Style.Styled` (+618/frame), `ansi.Style.BackgroundColor` (+309/frame), and `ansi.backgroundColorString` (+158/frame) — the cost of generating and wrapping `\x1b[48;5;236m...\x1b[0m` escape sequences around each selected cell's content.

**Render cost is viewport-bound, not dataset-bound.** All View benchmarks use 10,000 rows, but rendering 10 rows or 10,000,000 rows would produce identical numbers since `View()` only processes the visible range determined by the virtual scroll position.

---

## 4. Selection Materialization

These benchmarks measure `SelectedRows()` and `SelectedRowNodes()` at 100% selection (worst case for allocation).

| Method | Rows | Time | Memory | Allocs |
|---|---|---|---|---|
| SelectedRows | 1K | 322µs | 231 KB | 11 |
| SelectedRows | 10K | 3.69ms | 4.17 MB | 19 |
| SelectedRows | 100K | 35.5ms | 48.3 MB | 29 |
| SelectedRowNodes | 1K | 413µs | 17.1 KB | 11 |
| SelectedRowNodes | 10K | 3.64ms | 303 KB | 19 |
| SelectedRowNodes | 100K | 37.2ms | 4.3 MB | 28 |

### Key findings

**Time is nearly identical between the two methods.** `SelectedRows` (35.5ms) and `SelectedRowNodes` (37.2ms) take essentially the same time at 100K rows. The bottleneck is not copying data — it is the O(n) loop over `displayRows` with pointer dereferences and selection membership checks. Each of the 100K `RowNode` pointers points to a separately heap-allocated struct; traversing them incurs cache misses as the working set (100K × ~200 bytes = 20MB) far exceeds L1/L2 cache.

**Memory differs by 11×.** `SelectedRows` at 100K produces 48.3MB (100K × ~96-byte `benchRow` structs plus amortized slice growth) vs `SelectedRowNodes` at 4.3MB (100K × 8-byte pointers). For workloads where the caller needs to retain the selection result, `SelectedRowNodes` is 11× more memory-efficient — but the pointers reference live grid-internal state, requiring awareness of the semantics documented on the method.

**Allocation count is O(log n), not O(n).** Only 11–29 allocations regardless of how many rows are selected. The `append`-based growth strategy means the result slice is reallocated as it grows — Go's growth factor tapers from 2× to ~1.25× for large slices. At 100K elements, this accounts for all ~29 measured allocations; neither `SelectedRows()` nor `SelectedRowNodes()` allocate outside of the result slice growth.

**100K fully-selected rows takes 35ms.** This is not a hot path (it fires on user-initiated selection actions like Ctrl+A, not on every frame), but it establishes that materializing a full selection is expensive relative to the pipeline costs. For interactive applications, callers should avoid calling `SelectedRows()` on every `Update()` — only materialize when actually needed.

---

## 5. Grouping

The grouping package benchmarks isolate `BuildGroups` and `FlattenGroups` in isolation.

### FlattenGroups improvement (pointer slices, #9)

| Benchmark | Before | After | Factor |
|---|---|---|---|
| SingleLevel/100K time | 17.2ms | 946µs | **18×** |
| SingleLevel/100K memory | 175.9 MB | 6.3 MB | **28×** |
| TwoLevels/100K memory | 72.5 MB | 11.4 MB | **6.4×** |

**BuildGroups is unchanged** — its allocation structure is governed by the tree nodes it creates, not by the row type. The single-level improvement is entirely from eliminating the `[]RowNode[T]` value-copy output from `FlattenGroups`.

### Two-level grouping is 1.7× slower than single-level

At 100K rows: single-level 8.4ms vs two-level 14.9ms. Two-level grouping creates O(n/group_size) inner group nodes per outer group, requiring more recursive tree traversal during both build and flatten phases. The allocation count doubles (~200K → 400K allocs).

---

## 6. Filter Package

The filter package itself is fast — the costs are in the grid's pipeline, not in filter evaluation.

| Benchmark | Time |
|---|---|
| TextFilter.Active (active or inactive) | 0.25ns |
| TextFilter.Matches (1K rows) | 674µs |
| TextFilter.Matches regex (1K rows) | 458µs |
| NumberFilter.Matches (1K rows) | 15µs |
| SetFilter.Active (any size) | 0.23ns |
| SetFilter.Matches (1K rows) | 337µs |

**All `Active()` methods are O(1).** `TextFilter.Active()` compares a string to empty; `SetFilter.Active()` checks `excludedCount > 0`; others are similar flag checks. The benchmarks confirm 0.23–0.25ns regardless of filter size. Issue #10 fixed the *grid's* usage: `passesColumnFilters()` previously called `Active()` on all M columns for every row (O(N×M) calls). The fix pre-computes `activeFilters []int` once per recompute, iterating only columns with active filters.

---

## 7. Navigation

`Update()` for pure navigation keystrokes (arrow keys, no data change):

| Configuration | Time | Allocs |
|---|---|---|
| Navigation, 1K rows | ~12µs | 8 |
| Navigation, 100K rows | ~10µs | 8 |
| NavigationWithSort, 100K rows | ~10µs | 8 |

Navigation is **dataset-size independent** — the display pipeline is not invoked because `dirty` is never set by keystrokes. The ~10µs and 8 allocations reflect key matching, focus position update, and the Elm Architecture model value-copy (each `Update()` call receives and returns `Model[T]` by value). The 38KB per call is the shallow size of the model struct being copied.

**Removing the redundant `recomputeDisplayRows()` call at the top of `Update()` (fix #6)** had no measurable effect on navigation timing because the old call was a no-op (`dirty` was always false at that point). Its value was eliminating a dead code path that would have become expensive if `dirty` were ever set by a future change.

---

## 8. Findings Requiring Attention

### High priority

**View() allocates ~32 objects per cell.** 10,191 allocs per frame for 304 visible cells is high for a per-frame hot path. The source is lipgloss's `Render()` call for each cell — each styled render allocates ANSI escape sequence strings, padding, and the output string. A future optimization would be to pre-render or cache cell styles, or to use a `strings.Builder` passed through the render chain to amortize allocations.

**Selection styling allocates more than non-selection styling.** The extra 1,760 allocs when selection is active (+17%) come from lipgloss applying ANSI background color escape sequences. The `CellSelected` style has `Background(lipgloss.Color("236"))` while default even/odd row styles have no explicit background. Memory profiling confirms the increase is in `ansi.Style.Styled`, `ansi.Style.BackgroundColor`, and `ansi.backgroundColorString`. The selection membership check itself (`sel.ContainsCell`) does not allocate. If the default non-selected row styles also used explicit background colors, the alloc gap would shrink or disappear.

### Medium priority

**`passesQuickFilter` allocates 8 objects per row on the cold path.** The scratch buffer optimization (#8) addressed the `strings.Builder` copy-guard issue and eliminated per-row `strings.Fields` allocation. However, `strings.ToLower(string(m.quickFilterScratch))` still allocates two strings per row. Using `bytes.ToLower` in-place and `bytes.Contains` for matching would eliminate these, bringing quick filter allocs down from 8/row to ~0/row (same as column filter).

**`SelectedRows()` linear scan is cache-unfriendly at large N.** At 100K rows, 35ms for a full selection materialization reflects poor spatial locality — RowNode pointers are scattered in memory. An optimization would be to store row data contiguously (a `[]T` parallel array alongside the `[]*RowNode[T]`) so that `SelectedRows()` can do a contiguous memory scan rather than chasing pointers.

### Informational

**The filter cache is highly effective but fragile at the edges.** The `startFilterEdit` function sets `filterDirty=true` when the user opens the column filter editor, even if they immediately press Escape. This discards a valid cache unnecessarily. The `handleFilterEditKeyMsg` cancel branch also sets `filterDirty=true`, making the `startFilterEdit` flag redundant. This is conservative (correct) but wastes one recompute on Escape-cancel of the filter editor.

**Dynamic pinning predicates (`pinnedTopFunc`/`pinnedBotFunc`) are only evaluated on full filter passes.** On cache-hit recomputes, the cached `rn.Pinned` field value is used directly. Since predicates write `rn.Pinned` in-place, the pin state established on the first full filter pass persists permanently. This is correct for static predicates but would silently fail for predicates whose answer can change over time without the row data changing. Worth documenting.

---

## 9. Profiled Hotspots (April 2026)

CPU and memory profiles collected from targeted benchmarks at 100K rows.

### Sort: ValueGetter boxing dominates

**Profile:** `Sort_Cold/100K` — 34.9ms, 12.9MB, 1.59M allocs

88% of allocations come from a single source: `runtime.convT64` boxing
`float64` → `any` in the ValueGetter return. The sort comparator calls
`col.ValueGetter(rows[i].Data)` twice per comparison. With 100K rows
and O(n log n) stable sort, that's 3.4M+ boxing allocations per sort.
`defaultCompare` itself is fast (7% CPU, 0 allocs for same-type
comparisons). `findCol` is cheap (2.3% CPU, inlined).

Multi-key sort (3 keys) scales roughly linearly with key count: 124ms,
72MB, 5.5M allocs at 100K rows — each key adds its own ValueGetter
calls per comparison.

**Fix path:** A typed comparator like `Compare func(*T, *T) int` on
`Column` would eliminate both the `any` boxing (no interface return)
and the row struct copy (`*T` points to existing heap data). Same
pattern as `QuickFilterMatch`. The existing `Comparator func(a, b any)
int` field could remain as a fallback for columns that don't set the
typed variant.

### View: lipgloss Style struct copying is 33% of CPU

**Profile:** `View_Plain` — 1.03ms, 385KB, 10,190 allocs

`runtime.duffcopy` (Go's optimized memcpy) accounts for 33% of CPU.
The source is the lipgloss method chain
`style.Width(w).MaxWidth(w).Height(rowHeight).Render(cellContent)`:
each chained method returns a new copy of the `Style` struct, which is
large in lipgloss v2. Three intermediate copies per cell × 304 visible
cells = ~912 struct copies per frame.

`Style.isBorderStyleSetWithoutSides` at 12% of CPU checks border
properties on every cell. `runtime.mallocgc` at 13% reflects the
10K allocs from string building in the render pipeline.

Render scales linearly with column count: 4 cols → 581µs, 8 → 1.04ms,
16 → 2.01ms, 32 → 3.64ms.

**Fix path:** Pre-compute the fully-configured style per column during
`computeColWidths` (call `Width`/`MaxWidth`/`Height` once, store the
result) instead of rebuilding the style chain for every cell every
frame. This cannot eliminate lipgloss's internal allocations but would
remove the 912 intermediate struct copies. A more aggressive approach
would cache rendered cells when the underlying data and column width
haven't changed.

### Quick filter: 31ms per keystroke at 100K rows

**Profile:** `Update_QuickFilterKeystroke/100K` — 31.2ms, 10.5MB, 979K
allocs

Every character typed triggers a full cold filter re-evaluation across
all 100K rows. At 31ms per keystroke, a user typing at 60 WPM (~5
chars/sec) generates ~155ms of filter work per second — manageable, but
fast typists or large datasets will feel lag.

**Fix path:** Debounce the recompute. On each keystroke, update the
filter text and set dirty/filterDirty, but delay the actual recompute
by 50–100ms. If another keystroke arrives within the debounce window,
reset the timer. This avoids wasting work on intermediate filter states
that the user will type past.

## Appendix: Raw Numbers at 100K Rows

| Benchmark | Time | Memory | Allocs |
|---|---|---|---|
| Update_Navigation | 9.8µs | 37.5 KB | 8 |
| Update_SortToggle | 6.5µs | 19 KB | 2 |
| Update_QuickFilterKeystroke | 31,200µs | 10.5 MB | 979K |
| New (construction) | 9,234µs | 25.1 MB | 300K |
| SetRows | 6,000µs | 22.9 MB | 100K |
| RecomputeDisplayRows_NoFilter (cached) | 404µs | 784 KB | 1 |
| RecomputeDisplayRows_WithColumnFilter (cached) | 120µs | 80 KB | 1 |
| RecomputeDisplayRows_WithQuickFilter (cached) | 120µs | 80 KB | 1 |
| RecomputeDisplayRows_WithSort (cached) | 4,147µs | 2.66 MB | 240K |
| RecomputeDisplayRows_WithGrouping (cached) | 7,998µs | 12.6 MB | 200K |
| RecomputeDisplayRows_Full (cached) | 3,256µs | 2.3 MB | 180K |
| RecomputeDisplayRows_SortChangeOnly (cached) | 467µs | 270 KB | 24K |
| RecomputeDisplayRows_Sort_Cold | 34,900µs | 12.9 MB | 1.59M |
| RecomputeDisplayRows_Sort_MultiKey (3 keys) | 124,000µs | 72 MB | 5.55M |
| RecomputeDisplayRows_ColumnFilter_Cold | 7,630µs | 4.37 MB | 300K |
| RecomputeDisplayRows_QuickFilter_Cold | 29,800µs | 10.0 MB | 920K |
| RecomputeDisplayRows_QuickFilter_Cold_WithMatch | 17,200µs | 3.5 MB | 360K |
| RecomputeDisplayRows_Full_Cold | 15,523µs | 8.37 MB | 559K |
| View_Plain | 1,024µs | 387 KB | 10,191 |
| View_WithSelection | 1,140µs | 490 KB | 11,951 |
| View_ColumnCount (4 cols) | 581µs | 217 KB | 5,505 |
| View_ColumnCount (16 cols) | 2,010µs | 729 KB | 19,618 |
| View_ColumnCount (32 cols) | 3,639µs | 1.24 MB | 34,238 |
| SelectedRows (100% selected) | 35,500µs | 48.3 MB | 29 |
| SelectedRowNodes (100% selected) | 37,200µs | 4.3 MB | 28 |
