# Auto-fit column widths

**Status:** Draft
**Date:** 2026-04-20

## Problem

`data.Column` supports `Width` (fixed), `MinWidth`, `MaxWidth`, and `Flex`. There is no way to size a column to the widest rendered cell content, bounded by a maximum.

Consumers work around this by either hand-tuning `Flex` weights per column/dataset combination, or by precomputing content widths application-side. Both approaches duplicate the layout pass the grid already does. Every consumer rederives widths on every model rebuild, and each one tends to get filter/sort interactions subtly wrong.

This is standard grid behavior and should live in the library.

## Proposal

Add a declarative `AutoFit bool` field on `Column[T]`. When set, the grid measures the widest rendered content across the raw row set plus header, clamps to `[MinWidth, MaxWidth]`, and uses that as the column's width.

```go
type Column[T any] struct {
    // ...existing fields...

    // AutoFit sizes the column to the widest rendered content, clamped to
    // [MinWidth, MaxWidth]. Ignored when Width > 0. Takes precedence over Flex.
    AutoFit bool
}
```

### Precedence

When more than one sizing field is set, precedence is:

1. `Width > 0` → fixed width, ignore `AutoFit` and `Flex`.
2. `AutoFit = true` → content-measured, ignore `Flex`.
3. `Flex > 0` → share of remaining space as today.
4. Otherwise fall through to the current default (flex = 1 in `computeColWidths`).

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

Measurement runs from `computeColWidths` whenever it is called. Today that's `New()`, `SetColumns()`, `SetWidth()`, and column-pin changes. Column widths don't currently depend on data, so `SetRows`, `InsertRow`, `RemoveRow`, and `UpdateRow` only call `recomputeDisplayRows`, not `computeColWidths`.

For AutoFit, data-mutating setters must also trigger `computeColWidths`:

- `SetRows` → call `computeColWidths` after `recomputeDisplayRows`.
- `InsertRow`, `RemoveRow`, `UpdateRow` → same.

To avoid paying the measurement cost on every row mutation when no column is auto-fit, gate the data-triggered call on a model-level flag `hasAutoFit bool`, computed once in `New`/`SetColumns` by scanning `m.cols`.

Measurement does **not** run on filter, sort, scroll, selection, or cell edit. Those operations leave the column width stable, matching AG Grid's "don't churn during interaction" behavior.

Rationale: filtering to narrow widths would cause columns to jitter as the user types in a filter box. Measuring against the raw row set yields a stable upper bound bounded by `MaxWidth`, which is what users expect from "fit to content."

## Public API additions

```go
// In package data:
type NaturalWidthRenderer[T any] interface {
    CellRenderer[T]
    NaturalWidth(ctx CellContext[T]) int
}

// In Column[T]:
AutoFit bool
```

Built-in renderer changes (`data/cell_builtin.go`):
- `BarRenderer` and `ProgressRenderer` gain a `PreferredWidth int` field and a `NaturalWidth` method.
- `SparklineRenderer` gains a `NaturalWidth` method.

No new grid-level options, methods, or messages. Everything hangs off the existing `computeColWidths` call sites.

## Layout algorithm integration

In `grid.go:computeColWidths`, the current partition is:

1. Fixed-width columns (`Width > 0`) — allocated first, subtract from `remaining`.
2. Flex columns — given `MinWidth`, then distributed the remainder by `Flex` weight.

The new partition:

1. Fixed-width columns (`Width > 0`).
2. **Auto-fit columns (`AutoFit = true`)** — measured, clamped, subtract from `remaining`.
3. Flex columns — current behavior on what's left.

If remaining space goes negative after step 2, auto-fit columns keep their measured widths; flex columns get their `MinWidth` and the grid overflows horizontally as it does today. No special crowding-out logic.

## Performance

Measurement is O(N × M_autofit) where N is raw row count and M_autofit is the number of auto-fit columns. Each cell measurement is one function call (renderer or text fallback) plus one `lipgloss.Width`. Amortized across the existing `recomputeDisplayRows` cost (which is already O(N × M) for filter/sort), the overhead is modest.

No `MaxMeasureRows` cap, no row sampling. YAGNI until a profile shows it matters.

## Documentation

- `Column[T].AutoFit`: godoc describing precedence, measurement chain, and `NaturalWidthRenderer` opt-in.
- `NaturalWidthRenderer[T]`: godoc explaining when to implement it (any renderer that consumes `ctx.Width` to size its output, or that decorates its input so the text-fallback chain would under-measure it).
- Built-in renderers: note in godoc which ones implement `NaturalWidthRenderer`.
- README/CLAUDE.md sizing section: add `AutoFit` to the list of width modes with precedence order.

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

## Out of scope

- Three-way mode (`off | row-sample | all-rows`). YAGNI.
- `MaxMeasureRows` cap. YAGNI.
- User-driven auto-fit trigger (AG Grid–style double-click header separator or an `AutoSizeColumns()` API). Can be added later without breaking the declarative flag.
- Per-column auto-fit that accounts for group summary rows. Dynamic and rare; revisit if asked.

## Alternatives considered

1. **Do it in the consumer.** Every downstream caller ends up writing the same ~30-line measure-and-cap helper and rederives widths on every model rebuild.
2. **Overload `Width: -1` to mean "auto."** Hides intent; hurts discoverability.
3. **A separate `AutoColumn[T any]` factory type.** Doubles the column API surface for a single bit of behavior.
4. **AG Grid–style explicit trigger only (no declarative flag).** Forces policy back onto the consumer; loses the set-and-forget ergonomics that are the point of this feature. Can coexist with the flag later if someone wants it.
5. **Escape-hatch `Measure func(T) string` on `Column`.** A second extension point for the same job as `NaturalWidthRenderer`, and in the wrong place: measurement knowledge belongs with the renderer, not bolted onto the column alongside the renderer that would duplicate it.
6. **Measure only virtually-rendered rows (AG Grid's model).** Smaller `O()` but produces columns whose widths change when the user scrolls to data that was outside the viewport at measurement time. Confusing; rejected.
