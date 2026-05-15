# tea-grid

An AG Grid-inspired data grid component for [Bubble Tea v2](https://charm.land/bubbletea). Provides sorting, filtering, selection, cell editing, column/row pinning, grouping, virtual scrolling, and extensible cell rendering within the Elm Architecture.

![demo](demo.gif)

## Features

- **Sorting** -- single and multi-column, ascending/descending, custom comparators
- **Filtering** -- per-column filters (text, number, set, bool, time, multiset) and a GitHub-style query bar with round-trip between bar text and column filter state
- **Selection** -- single/multi row, column, and rectangular cell selection
- **Cell editing** -- built-in text, number, bool, select, and time editors
- **Column/row pinning** -- pin columns left/right, rows top/bottom
- **Grouping** -- hierarchical row grouping with aggregation (sum, avg, count, min, max)
- **Virtual scrolling** -- renders only visible rows for large datasets
- **Variable row heights** -- per-row height with height-aware viewport
- **Extensible rendering** -- custom `CellRenderer` and `CellEditor` interfaces
- **Vim-style keybindings** -- hjkl navigation, customizable via `KeyMap`

## Installation

```
go get github.com/pgavlin/tea-grid
```

## Quick Start

```go
package main

import (
    "fmt"
    "os"

    tea "charm.land/bubbletea/v2"

    "github.com/pgavlin/tea-grid/data"
    "github.com/pgavlin/tea-grid/grid"
)

type Employee struct {
    Name       string
    Department string
    Salary     float64
    Active     bool
}

type model struct {
    grid grid.Model[Employee]
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.grid.SetWidth(msg.Width)
        m.grid.SetHeight(msg.Height)
    case tea.KeyPressMsg:
        if msg.String() == "q" || msg.String() == "ctrl+c" {
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
    cols := data.FromType[Employee]()
    rows := []Employee{
        {"Alice Johnson", "Engineering", 145000, true},
        {"Bob Smith", "Engineering", 130000, true},
        {"Carol Davis", "Marketing", 95000, false},
        {"Dave Wilson", "Marketing", 105000, true},
        {"Eve Brown", "Sales", 88000, true},
    }

    g := grid.New(
        grid.WithColumns(cols),
        grid.WithRows(rows),
        grid.WithRowID(func(e Employee) string { return e.Name }),
        grid.WithFocused[Employee](true),
    )

    p := tea.NewProgram(model{grid: g})
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

## Architecture

The core type is `grid.Model[T]`. All state lives in the model; messages flow through `Update`, and rendering happens in `View`.

### Package Structure

```
grid/        Main integration package: Model, Update, View, options, styles, keymap
data/        Column, RowNode, CellContext, CellRenderer, CellEditor, built-in renderers/editors
filter/      Filter interface and built-in filters (text, number, set, bool, time, multiset)
sort/        Sort state and multi-column sort logic
selection/   Rectangle-based selection model (row, column, rect)
grouping/    Row grouping, tree flattening, and aggregation functions
searchquery/ Query bar parser and vocabulary (powers WithQueryBar)
```

### Display Pipeline

Raw rows pass through a pipeline in `recomputeDisplayRows` (triggered when the `dirty` flag is set):

1. Pin separation (top/bottom pinned rows extracted)
2. External filter
3. Per-column filters
4. Quick filter
5. Grouping (hierarchical tree construction)
6. Sorting (within groups or flat)
7. Post-sort hook
8. Flattened display list with cached aggregation values

### Column Definition

Columns can be derived automatically from struct fields or defined manually:

```go
// Automatic: derives columns from struct fields via reflection
cols := data.FromType[Employee]()

// Manual: full control over each column
cols := []data.Column[Employee]{
    {
        ColumnID:   "name",
        HeaderName: "Name",
        Value:      func(e Employee) any { return e.Name },
        Sortable:   true,
        Filterable: true,
        Flex:       1,
    },
}
```

### Functional Options

The grid is configured via functional options passed to `grid.New`:

```go
g := grid.New(
    grid.WithColumns(cols),
    grid.WithRows(rows),
    grid.WithWidth(120),
    grid.WithHeight(40),
    grid.WithFocused[Employee](true),
    grid.WithSelection[Employee](selection.SelectMulti),
    grid.WithQueryBar[Employee](),
    grid.WithEditable[Employee](true),
    grid.WithGrouping[Employee]("Department"),
    grid.WithMultiSort[Employee](true),
)
```

## Examples

See the `examples/` directory:

- **basic** -- minimal grid with struct rows
- **spreadsheet** -- editable grid with multiple column types
- **selection** -- row, column, and rect selection
- **hscroll** -- horizontal scrolling with pinned columns
- **querybar** -- GitHub-style query bar with column-aware syntax
- **csv** -- load and display CSV files
- **jsonl** -- load and display JSONL files
- **anyrow** -- map-based rows with automatic column inference
- **columns** -- manual column definition with custom renderers

Run an example:

```
go run ./examples/basic/
```

## Key Bindings

Default bindings (vim-style, configurable via `grid.WithKeyMap`):

| Key | Action |
|-----|--------|
| `j`/`k` or arrows | Navigate rows |
| `h`/`l` or arrows | Navigate columns |
| `pgup`/`pgdn` | Page up/down |
| `ctrl+u`/`ctrl+d` | Half page up/down |
| `home`/`end` | First/last row |
| `0`/`$` | First/last column |
| `g` | Go to header |
| `space` | Toggle cell selection |
| `R`/`C` | Select row / column |
| `ctrl+a` | Select all |
| `esc` | Deselect / clear filters / cancel edit |
| `shift+arrows` or `H`/`J`/`K`/`L` | Expand rectangular selection |
| `s`/`S` | Sort / multi-sort by focused column |
| `enter`/`shift+enter` | Sort / multi-sort (when header focused) |
| `/` | Open query bar |
| `ctrl+f` | Column filter |
| `enter`/`F2` | Edit cell (in a data row) |
| `G` | Toggle grouping by focused column |
| `enter` | Toggle group expansion (on a group row) |
| `ctrl+shift+left`/`right` | Collapse / expand all groups |
| `w`/`W` | Auto-fit focused column / all columns |
