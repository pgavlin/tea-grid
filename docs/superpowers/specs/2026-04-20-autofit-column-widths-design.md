# Auto-fit column widths

**Status:** Draft
**Date:** 2026-04-20

## Problem

`data.Column` supports `Width` (fixed), `MinWidth`, `MaxWidth`, and `Flex`. There is no way to size a column to the widest rendered cell content, bounded by a maximum.

Consumers work around this by either hand-tuning `Flex` weights per column/dataset combination, or by precomputing content widths application-side. Both approaches duplicate the layout pass the grid already does. Every consumer rederives widths on every model rebuild, and each one tends to get filter/sort interactions subtly wrong.

This is standard grid behavior and should live in the library.

## Proposal

Two additions, one declarative and one imperative:

**Declarative.** A `AutoFit bool` field on `Column[T]`. When set, the grid measures the widest rendered content across the *raw* row set plus header, clamps to `[MinWidth, MaxWidth]`, and uses that as the column's width. Recomputed on data-set changes; stable under filter/sort/scroll.

**Imperative.** Four methods on `Model[T]` — `AutoSizeColumns()`, `AutoSizeColumn(colID)`, `ResetColumnWidths()`, `ResetColumnWidth(colID)` — that measure the currently *displayed* rows (post-filter, post-sort) and set widths as sticky per-column overrides. Overrides persist across subsequent data mutations until explicitly cleared.

```go
type Column[T any] struct {
    // ...existing fields...

    // AutoFit sizes the column to the widest rendered content across raw rows,
    // clamped to [MinWidth, MaxWidth]. Ignored when Width > 0. Takes precedence
    // over Flex. Overridden by any width set via AutoSizeColumn(s).
    AutoFit bool
}
```

### Precedence

When more than one sizing path applies, precedence is:

1. **Sticky override** (set by `AutoSizeColumn(s)`, cleared by `ResetColumnWidth(s)`) — wins over everything.
2. `Width > 0` → fixed width, ignore `AutoFit` and `Flex`.
3. `AutoFit = true` → content-measured against raw rows, ignore `Flex`.
4. `Flex > 0` → share of remaining space as today.
5. Otherwise fall through to the current default (flex = 1 in `computeColWidths`).

## Measurement

For each `AutoFit` column, the grid computes the max of:

- The header (`HeaderName`) display width.
- The rendered display width of each row's cell, using the fallback chain below.

Rendered width is computed via `lipgloss.Width`, which strips ANSI and handles wide characters (CJK, emoji) correctly.

### Why not invoke `Render` directly

A plausible approach is: call `Render(ctx)` with a large sentinel `Width` and measure the result with `lipgloss.Width`. This doesn't work because every built-in renderer (and most user renderers) ends with `TruncateOrPad(content, ctx.Width)` — they pad to the requested width. Calling `Render` with `Width = 16384` produces a string 16384 columns wide regardless of content, so the measurement is always the sentinel. Trimming trailing whitespace would work for simple cases but is unsafe in general (padding may be styled with a background color).

Instead the grid never calls `Render` for measurement. Renderers that want to participate in auto-fit declare their natural width via an optional sub-interface; renderers that don't implement it are treated as transparent, and the column falls back to the text/value chain.

### NaturalWidthRenderer interface

```go
// NaturalWidthRenderer reports a renderer's preferred display width,
// independent of ctx.Width. Renderers that want their output to drive
// AutoFit should implement this.
type NaturalWidthRenderer[T any] interface {
    CellRenderer[T]
    NaturalWidth(ctx CellContext[T]) int
}
```

### Fallback chain

For each row, produce a display width:

1. If `Column.CellRenderer` is set and implements `NaturalWidthRenderer[T]`, use `NaturalWidth(ctx)`.
2. Else ignore the renderer and measure a string via the chain: `Text(&row)` → `ValueFormatter(Value(row), row)` → `conv.SprintValue(Value(row))`, then `lipgloss.Width`.

This works correctly for text-ish built-in renderers (`TextRenderer`, `NumberRenderer`, `TimeRenderer`, `BoolRenderer`) without them implementing `NaturalWidthRenderer`, because their rendered content is `ctx.FormattedValue` + padding — and the padding is what we want to strip for measurement. The text fallback produces the same pre-padding content.

The tradeoff: a custom renderer that decorates its input (e.g. wraps a value in brackets or appends a glyph) will be *under-measured* by the text fallback. Users writing such renderers implement `NaturalWidthRenderer` to get accurate measurement; until they do, the column sizes to the decorated content's *undecorated* text, which is a mild but documented under-size.

### Measurement context

When invoking `NaturalWidth` for measurement, the grid constructs a `CellContext[T]` with:

- `Value`, `FormattedValue`, `Data`, `RowNode`, `Column`, `ColumnIndex`, `RowIndex` populated as they are in render.
- `IsSelected`, `IsFocused` = `false`.
- `Width` = `MaxWidth` if set, else 0 (a signal that no target width is imposed; the renderer should compute its natural width from content, not from `ctx.Width`).
- `Height` = `1`.

### Built-in renderer updates

- `BarRenderer`, `ProgressRenderer`: implement `NaturalWidth` returning a configurable preferred width. Add a `PreferredWidth int` field; default to 10 when zero.
- `SparklineRenderer`: implement `NaturalWidth` returning `len(values)` when `ctx.Value` is `[]float64`, else a reasonable default.
- `TextRenderer`, `NumberRenderer`, `TimeRenderer`, `BoolRenderer`: no change. The text fallback produces the correct measurement via `ctx.FormattedValue`.

### What is measured

- **Included:** visible, non-spanning cells in the raw row set (`m.rows`), before any filter or sort. Plus the header.
- **Excluded:**
  - Cells where `ColumnSpan(row) > 1` (they lie about per-column width by design).
  - Group summary rows (dynamic, aggregation-dependent; may change as the user expands/collapses groups).
  - Pinned rows (`PinTop` / `PinBottom`): included — they are part of `m.rows`.
  - Hidden columns: not measured (width stays 0).

### Clamping

The measured width is clamped to `[max(MinWidth, 4), MaxWidth]`. `MinWidth = 0` defaults to 4 as in the current code. `MaxWidth = 0` means unconstrained.

If a cell's rendered content exceeds the clamped column width at draw time, the existing truncation/ellipsis behavior applies.

## When measurement runs

### Declarative (`AutoFit`)

Measurement runs from `computeColWidths` whenever it is called. Today that's `New()`, `SetColumns()`, `SetWidth()`, and column-pin changes. Column widths don't currently depend on data, so `SetRows`, `InsertRow`, `RemoveRow`, and `UpdateRow` only call `recomputeDisplayRows`, not `computeColWidths`.

For `AutoFit`, data-mutating setters must also trigger `computeColWidths`:

- `SetRows` → call `computeColWidths` after `recomputeDisplayRows`.
- `InsertRow`, `RemoveRow`, `UpdateRow` → same.

To avoid paying the measurement cost on every row mutation when no column is auto-fit, gate the data-triggered call on a model-level flag `hasAutoFit bool`, computed once in `New`/`SetColumns` by scanning `m.cols`.

Measurement does **not** run on filter, sort, scroll, selection, or cell edit. Those operations leave the column width stable, matching AG Grid's "don't churn during interaction" behavior.

Rationale: filtering to narrow widths would cause columns to jitter as the user types in a filter box. Measuring against the raw row set yields a stable upper bound bounded by `MaxWidth`, which is what users expect from "fit to content."

### Imperative (`AutoSizeColumn(s)`)

Runs only when the consumer calls the method. Each call:

1. Measures the target column(s) against `m.displayRows` (post-filter, post-sort, including expanded group rows).
2. Uses the same NaturalWidth / text-fallback chain as declarative `AutoFit`.
3. Writes the clamped result into `m.manualWidths[colID]`.
4. Triggers `computeColWidths` to repaint.

`AutoSizeColumns()` measures every non-hidden column; `AutoSizeColumn(colID)` measures one. A `colID` that doesn't match any column is silently ignored (following existing pattern in `PinColumn`).

Subsequent filter/sort/scroll/`SetRows`/`InsertRow`/`RemoveRow`/`UpdateRow`/`SetWidth` calls preserve the override. Only `ResetColumnWidth(s)` or `SetColumns` (for removed IDs) clear it.

Measurement against displayed rows rather than raw rows is the entire point of the imperative path — the user is asking "fit to what's on screen right now."

## Public API additions

```go
// In package data:
type NaturalWidthRenderer[T any] interface {
    CellRenderer[T]
    NaturalWidth(ctx CellContext[T]) int
}

// In Column[T]:
AutoFit bool

// In package grid, on Model[T]:

// AutoSizeColumns re-measures every non-hidden column against the currently
// displayed rows (post-filter, post-sort) and stores the clamped widths as
// sticky overrides. Overrides win over Width, AutoFit, and Flex until cleared
// by ResetColumnWidth(s).
func (m *Model[T]) AutoSizeColumns()

// AutoSizeColumn re-measures a single column the same way. A colID that
// doesn't match any column is ignored.
func (m *Model[T]) AutoSizeColumn(colID string)

// ResetColumnWidths clears all sticky width overrides. Columns revert to
// their declared sizing (Width / AutoFit / Flex).
func (m *Model[T]) ResetColumnWidths()

// ResetColumnWidth clears the sticky override for one column.
func (m *Model[T]) ResetColumnWidth(colID string)
```

Built-in renderer changes (`data/cell_builtin.go`):
- `BarRenderer` and `ProgressRenderer` gain a `PreferredWidth int` field and a `NaturalWidth` method.
- `SparklineRenderer` gains a `NaturalWidth` method.

Internal model state: `manualWidths map[string]int` on `Model[T]`. Keyed by `ColumnID`. Written by `AutoSizeColumn(s)`; read by `computeColWidths` as the first rung of precedence; pruned by `SetColumns` to drop entries whose `ColumnID` is no longer present.

### KeyMap additions

Two new fields on `KeyMap` in `grid/keymap.go`, with defaults matching the existing lowercase-current / shift-broader pattern (e.g. `s`/`S` for sort, `R`/`C` for row/column select):

```go
type KeyMap struct {
    // ...existing fields...

    AutoSizeColumn  key.Binding // Fit the focused column to content.
    AutoSizeColumns key.Binding // Fit all visible columns to content.
}
```

Defaults in `DefaultKeyMap()`:

- `AutoSizeColumn`: `w` (help: "fit column width")
- `AutoSizeColumns`: `W` (help: "fit all column widths")

`ResetColumnWidth(s)` have no default binding. Consumers that want a reset key wire their own; an accidental `w` is cheap to re-press.

Dispatch in `grid/update.go:handleKeyMsg`: call `m.AutoSizeColumn(col.ColumnID)` for the focused column on `AutoSizeColumn`; call `m.AutoSizeColumns()` on `AutoSizeColumns`. Neither runs in edit / filter-edit / quick-filter mode (standard gating).

## Layout algorithm integration

In `grid.go:computeColWidths`, the current partition is:

1. Fixed-width columns (`Width > 0`) — allocated first, subtract from `remaining`.
2. Flex columns — given `MinWidth`, then distributed the remainder by `Flex` weight.

The new partition:

1. **Overridden columns** (in `manualWidths`) — use the stored width, subtract from `remaining`.
2. Fixed-width columns (`Width > 0`).
3. **Auto-fit columns (`AutoFit = true`)** — measured against raw rows, clamped, subtract from `remaining`.
4. Flex columns — current behavior on what's left.

If remaining space goes negative after step 3, auto-fit and overridden columns keep their widths; flex columns get their `MinWidth` and the grid overflows horizontally as it does today. No special crowding-out logic.

## Performance

**Declarative `AutoFit`:** O(N × M_autofit) per re-layout, where N is raw row count and M_autofit is the number of auto-fit columns. Each cell measurement is one function call (renderer or text fallback) plus one `lipgloss.Width`. Amortized across the existing `recomputeDisplayRows` cost (which is already O(N × M) for filter/sort), the overhead is modest.

**Imperative `AutoSizeColumn(s)`:** O(D × M_target) where D is the displayed row count and M_target is 1 (single column) or the visible column count (`AutoSizeColumns`). Called only on explicit user action, so cost is bounded by user frequency rather than data scale.

No `MaxMeasureRows` cap, no row sampling. YAGNI until a profile shows it matters.

## Documentation

- `Column[T].AutoFit`: godoc describing precedence, measurement chain, and `NaturalWidthRenderer` opt-in.
- `NaturalWidthRenderer[T]`: godoc explaining when to implement it (any renderer that consumes `ctx.Width` to size its output, or that decorates its input so the text-fallback chain would under-measure it).
- Built-in renderers: note in godoc which ones implement `NaturalWidthRenderer`.
- `Model[T].AutoSizeColumn(s)` and `ResetColumnWidth(s)`: godoc describing the sticky-override semantics, that measurement uses displayed rows, and how to combine with declarative `AutoFit`.
- `KeyMap.AutoSizeColumn` / `AutoSizeColumns`: godoc noting the defaults (`w` / `W`) and that reset is intentionally unbound.
- README/CLAUDE.md sizing section: add `AutoFit`, the four imperative methods, and the key bindings, with precedence order.

## Testing

- `TestComputeColWidths_AutoFit_Basic`: column with `AutoFit = true`, measure against a set of short strings; column width equals max(header, max(cell)).
- `TestComputeColWidths_AutoFit_ClampedByMaxWidth`: content exceeds `MaxWidth`; column renders at `MaxWidth` and cells truncate.
- `TestComputeColWidths_AutoFit_ClampedByMinWidth`: content narrower than `MinWidth`; column renders at `MinWidth`.
- `TestComputeColWidths_AutoFit_WithNaturalWidthRenderer`: column with a renderer that implements `NaturalWidthRenderer`; measurement uses `NaturalWidth` rather than the text fallback.
- `TestComputeColWidths_AutoFit_RendererWithoutNaturalWidthUsesTextFallback`: column with a `CellRenderer` that does not implement `NaturalWidthRenderer`; measurement uses Text/ValueFormatter/Value rather than calling `Render`.
- `TestComputeColWidths_AutoFit_HeaderIsMeasured`: rows all short, header long; width matches header.
- `TestComputeColWidths_AutoFit_WideRunes`: cells contain CJK or emoji; measured width is display width, not rune count.
- `TestComputeColWidths_AutoFit_StableUnderFilter`: apply a filter that narrows rows; column width does not change.
- `TestComputeColWidths_AutoFit_RecomputesOnSetRows`: call `SetRows` with wider content; column grows.
- `TestComputeColWidths_AutoFit_IgnoredWhenWidthSet`: `Width = 20` and `AutoFit = true`; column is 20.
- `TestComputeColWidths_AutoFit_TakesPrecedenceOverFlex`: `AutoFit = true` and `Flex = 3`; column is auto-fit, remaining space distributed among other flex columns.
- `TestComputeColWidths_AutoFit_SpanningCellsIgnored`: cell with `ColumnSpan > 1` does not contribute to its column's measurement.
- `TestComputeColWidths_AutoFit_HiddenColumn`: `Hide = true` + `AutoFit = true` → width 0, not measured.

### Imperative API

- `TestAutoSizeColumns_MeasuresDisplayedRows`: apply a filter that narrows rows, call `AutoSizeColumns`; widths match filtered content, not the raw row set.
- `TestAutoSizeColumn_Single`: call `AutoSizeColumn("foo")`; only `foo` is resized, other columns unchanged.
- `TestAutoSizeColumn_UnknownID`: call `AutoSizeColumn("nope")`; no-op, no panic.
- `TestAutoSizeColumns_OverridesAutoFit`: column with `AutoFit = true`; after `AutoSizeColumns`, subsequent `SetRows` with wider content does **not** re-measure (override is sticky).
- `TestAutoSizeColumns_OverridesWidth`: column with `Width = 50`; after `AutoSizeColumns`, column renders at measured width, not 50.
- `TestAutoSizeColumns_OverridesFlex`: column with `Flex = 3`; after `AutoSizeColumns`, column stops participating in flex distribution.
- `TestResetColumnWidth_RevertsToDeclared`: call `AutoSizeColumn("foo")` then `ResetColumnWidth("foo")`; column returns to its declared `Width`/`AutoFit`/`Flex` behavior on next layout.
- `TestResetColumnWidths_ClearsAll`: multiple overrides set; `ResetColumnWidths` clears them all.
- `TestAutoSize_StickyAcrossSetRows`: call `AutoSizeColumns`, then `SetRows` with different data; override widths persist.
- `TestAutoSize_StickyAcrossFilterSort`: call `AutoSizeColumns`, filter, sort; override widths persist.
- `TestSetColumns_PrunesOverrides`: set override on column "foo", then `SetColumns` to a set without "foo"; `manualWidths["foo"]` is dropped.
- `TestAutoSizeColumns_RespectsMinMaxWidth`: override is clamped to `[MinWidth, MaxWidth]` just like declarative `AutoFit`.
- `TestAutoSizeColumns_NoRows`: call with no rows; widths equal header widths (clamped).
- `TestKeyMap_AutoSizeColumn_FitsFocusedColumn`: focus column "foo", press `w`; only `foo`'s width changes.
- `TestKeyMap_AutoSizeColumns_FitsAll`: press `W`; every visible column's width updates.
- `TestKeyMap_AutoSize_GatedByMode`: in edit mode / filter-edit mode / quick-filter mode, `w` and `W` are not dispatched (they behave as ordinary input or are swallowed by the active mode, matching the existing gating pattern).

## Out of scope

- Three-way mode (`off | row-sample | all-rows`). YAGNI.
- `MaxMeasureRows` cap. YAGNI.
- Default key binding for `ResetColumnWidth(s)` (consumers wire their own; the reset API ships but is unbound by default).
- Per-column auto-fit that accounts for group summary rows. Dynamic and rare; revisit if asked.

## Alternatives considered

1. **Do it in the consumer.** Every downstream caller ends up writing the same ~30-line measure-and-cap helper and rederives widths on every model rebuild.
2. **Overload `Width: -1` to mean "auto."** Hides intent; hurts discoverability.
3. **A separate `AutoColumn[T any]` factory type.** Doubles the column API surface for a single bit of behavior.
4. **AG Grid–style explicit trigger only (no declarative flag).** Forces policy back onto the consumer; loses the set-and-forget ergonomics. The final design ships both, since they serve different use cases (stable declarative widths vs. "fit to what I'm looking at").
5. **Escape-hatch `Measure func(T) string` on `Column`.** A second extension point for the same job as `NaturalWidthRenderer`, and in the wrong place: measurement knowledge belongs with the renderer, not bolted onto the column alongside the renderer that would duplicate it.
6. **Measure only virtually-rendered rows (AG Grid's model).** Smaller `O()` but produces columns whose widths change when the user scrolls to data that was outside the viewport at measurement time. Confusing; rejected.
