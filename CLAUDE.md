# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

tea-grid is an AG Grid-inspired data grid component for [Bubble Tea v2](https://charm.land/bubbletea). It provides sorting, filtering, selection, cell editing, column/row pinning, grouping, virtual scrolling, variable row heights, and extensible cell rendering — all within the Elm Architecture. The core type is `grid.Model[T]` which implements `tea.Model`. Uses `charm.land` module paths (Bubble Tea v2, Lipgloss v2, Bubbles v2).

## Build & Test Commands

```bash
# Run all tests
go test ./...

# Run tests for a single package
go test ./grid/

# Run a single test
go test ./grid/ -run TestNav_ArrowKeys

# Verbose output
go test ./... -v -count=1

# Race detector
go test -race ./...

# Run an example
go run ./examples/basic/
```

### CI & Tooling

- **GitHub Actions** (`.github/workflows/ci.yml`): `test` (go test -race), `lint` (golangci-lint v7), `format` (gofumpt check) — all on Go 1.25
- **`.golangci.yml`**: govet, errcheck, staticcheck, unused; gofumpt formatter
- **`.mise.toml`**: Go 1.25.1, golangci-lint 2.9.0, gofumpt 0.9.2

## Architecture

### Elm Architecture Flow

All state lives in `grid.Model[T]`. Messages flow through `Update` (in `grid/update.go`), state is read-only in `View` (in `grid/render.go`). No side-channels or shared mutable state.

### Package Dependency Graph

```
grid      →  data, filter, sort, selection, grouping, internal/lineedit
data      →  filter (Column has Filter field; also defines CellRenderer/CellEditor/RowNode)
filter    →  (standalone, defines Filter interface)
sort      →  data (uses SortDirection)
grouping  →  data (uses RowNode, Column)
selection →  (standalone, rectangular selection model)
internal/lineedit →  (standalone, line editing widget for filter input)
```

`grid/` is the integration package that composes all others. The sub-packages are independently usable and have no dependency on `grid/`.

### Key Design Patterns

- **Generics**: `Model[T]` is parameterized on the row data type. `Column[T]` uses `ValueGetter func(T) any` for type-safe data extraction.
- **Functional options**: `grid.New[T](opts...)` using `Option[T] func(*Model[T])`.
- **Interface-based extension**: `data.CellRenderer[T]`, `data.CellEditor[T]`, and `filter.Filter` are the main extension points.
- **Display row pipeline** (`grid.go:recomputeDisplayRows`): raw rows → pin separation → external filter → column filters → quick filter → grouping → sorting → post-sort hook → flat display list. Results are cached; the `dirty` flag triggers recomputation. Public setters eagerly recompute; `Init()` is a no-op.

### Grid File Responsibilities

| File | Role |
|------|------|
| `grid/grid.go` | Model struct, New(), data CRUD, display pipeline, column sizing, defaultCompare |
| `grid/update.go` | Update() message routing, key handlers for normal/edit/filter modes |
| `grid/render.go` | View() rendering: headers, rows, groups, pinned regions, filters |
| `grid/viewport.go` | Virtual scroll state (topRow/leftCol), visibility range |
| `grid/options.go` | All `With*` functional options |
| `grid/styles.go` | Styles struct, DefaultStyles() |
| `grid/keymap.go` | KeyMap struct, DefaultKeyMap() — vim-style keys |
| `grid/messages.go` | All tea.Msg types emitted by the grid |

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

### Update Routing

`Update()` dispatches `KeyPressMsg` (Bubble Tea v2) based on mode:
1. **Editing** → `handleEditKeyMsg` (Enter confirms, Esc cancels, else routes to editor)
2. **Filter editing** → `handleFilterEditKeyMsg` (Enter applies, Esc clears)
3. **Quick filter active** → `handleQuickFilterKeyMsg` (runes append, Esc clears)
4. **Normal** → `handleKeyMsg` (navigation, selection, sort, group, edit start)

### Selection Model

The `selection` package uses a **rectangular selection model**. Key types:
- `Mode`: `SelectNone`, `SelectSingle`, `SelectMulti`
- `Kind`: `KindNone`, `KindRect`, `KindFullRow`, `KindFullCol`
- `Rect`: anchor + cursor positions defining a selection rectangle
- `Model`: holds a slice of `Rect`s; supports shift+navigation expansion

### Reflection Usage

`data.Columns[T]()` uses reflection once at init to derive `Column` slice from struct fields. In hot paths, everything uses the `ValueGetter` function closures — no reflection at runtime.
