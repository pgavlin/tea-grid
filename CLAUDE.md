# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

tea-grid is an AG Grid-inspired data grid component for [Bubble Tea](https://github.com/charmbracelet/bubbletea). It provides sorting, filtering, selection, cell editing, column/row pinning, grouping, virtual scrolling, and extensible cell rendering — all within the Elm Architecture. The core type is `grid.Model[T]` which implements `tea.Model`.

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

No Makefile, linter config, or CI pipeline exists. Standard `go build`/`go test` tooling only.

## Architecture

### Elm Architecture Flow

All state lives in `grid.Model[T]`. Messages flow through `Update` (in `grid/update.go`), state is read-only in `View` (in `grid/render.go`). No side-channels or shared mutable state.

### Package Dependency Graph

```
grid      →  data, filter, sort, selection, grouping
data      →  filter (ColDef has Filter field; also defines CellRenderer/CellEditor/RowNode)
filter    →  (standalone, defines Filter interface)
sort      →  data (uses SortDirection)
grouping  →  data (uses RowNode, ColDef)
selection →  (standalone)
```

`grid/` is the integration package that composes all others. The sub-packages are independently usable and have no dependency on `grid/`.

### Key Design Patterns

- **Generics**: `Model[T]` is parameterized on the row data type. `ColDef[T]` uses `ValueGetter func(T) any` for type-safe data extraction.
- **Functional options**: `grid.New[T](opts...)` using `Option[T] func(*Model[T])`.
- **Interface-based extension**: `data.CellRenderer[T]`, `data.CellEditor[T]`, and `filter.Filter` are the main extension points.
- **Display row pipeline** (`grid.go:recomputeDisplayRows`): raw rows → pin separation → external filter → column filters → quick filter → grouping → sorting → post-sort hook → flat display list. Results are cached; the `dirty` flag triggers recomputation.

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

In `grid.go:computeColWidths`: fixed-width columns allocated first, then remaining space distributed to flex columns by weight (cumulative division to avoid pixel loss). MinWidth (default 4) and MaxWidth are respected. Hidden columns get width 0. Border separators subtract from available space when `BorderColumn=true`.

### Update Routing

`Update()` dispatches `KeyMsg` based on mode:
1. **Editing** → `handleEditKeyMsg` (Enter confirms, Esc cancels, else routes to editor)
2. **Filter editing** → `handleFilterEditKeyMsg` (Enter applies, Esc clears)
3. **Quick filter active** → `handleQuickFilterKeyMsg` (runes append, Esc clears)
4. **Normal** → `handleKeyMsg` (navigation, selection, sort, group, edit start)

### Reflection Usage

`data.Columns[T]()` uses reflection once at init to derive `ColDef` slice from struct fields. In hot paths, everything uses the `ValueGetter` function closures — no reflection at runtime.
