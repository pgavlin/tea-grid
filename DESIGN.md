# tea-grid: AG Grid-Inspired Table Component for Bubble Tea

## Design Document

### 1. Overview

`tea-grid` is a high-performance, feature-rich data grid component for
[Bubble Tea v2](https://charm.land/bubbletea). It brings the power and
flexibility of [AG Grid](https://www.ag-grid.com/) to terminal user interfaces,
providing sorting, filtering, selection, cell editing, column/row pinning, grouping,
virtual scrolling, and extensible cell rendering -- all within the Elm Architecture.

The existing `bubbles/table` component is intentionally minimal: columns are
`{Title, Width}`, rows are `[]string`, and the only interaction is cursor movement.
`tea-grid` aims to be the "batteries included" option for applications that need a
real data grid.

---

### 2. Goals and Non-Goals

**Goals**

- Type-safe row data via Go generics.
- AG Grid-caliber feature set adapted for terminal constraints.
- First-class keyboard navigation; optional mouse support.
- Virtual scrolling for datasets in the tens of thousands of rows.
- Composable: individual features (sorting, filtering, editing, ...) can be
  enabled/disabled independently.
- Idiomatic Bubble Tea: the grid is a `tea.Model` that composes cleanly with other
  Bubble Tea components.
- Styling via lipgloss, consistent with the Charm ecosystem.

**Non-Goals**

- Server-side row models or infinite scrolling backed by a remote data source (may be
  added later as a separate data source adapter).
- Chart integration.
- Clipboard integration beyond what the terminal provides.
- Drag-and-drop column reordering (no reliable cross-terminal mouse-drag support).

---

### 3. Architecture

#### 3.1 Elm Architecture Integration

```
            ┌─────────────┐
  Msg ────▶ │   Update    │ ────▶ Cmd
            │ (update.go) │
            └──────┬──────┘
                   │
                   ▼
            ┌─────────────┐
            │    Model    │  (state: rows, columns, selection, sort, filter, ...)
            └──────┬──────┘
                   │
                   ▼
            ┌─────────────┐
            │    View     │ ────▶ string
            │ (render.go) │
            └─────────────┘
```

The grid is a single `grid.Model[T]` that implements `tea.Model`. All state
mutations flow through `Update`; all rendering flows through `View`. There are no
side-channels or shared mutable state.

#### 3.2 Package Layout

```
github.com/pgavlin/tea-grid/
├── grid/                # Core grid model, update, view
│   ├── grid.go          # Model[T], New(), data CRUD, display pipeline, column sizing
│   ├── update.go        # Update() message routing, key handlers
│   ├── render.go        # View() rendering pipeline
│   ├── viewport.go      # Virtual scrolling
│   ├── options.go       # All With* functional options
│   ├── keymap.go        # KeyMap and default bindings
│   ├── styles.go        # Styles and DefaultStyles()
│   └── messages.go      # All tea.Msg types emitted by the grid
├── data/                # Column, row, and cell types
│   ├── column.go        # Column[T], ColumnGroup[T], Pin, SortDirection, FromType, FromRows
│   ├── row.go           # RowNode[T]
│   ├── cell.go          # CellContext[T], CellRenderer[T], CellEditor[T]
│   └── cell_builtin.go  # Built-in renderers and editors
├── filter/              # Filtering subsystem
│   ├── filter.go        # Filter interface, message types
│   └── builtin.go       # Text, Number, Set, Bool, Time filters
├── sort/                # Sorting subsystem
│   └── sort.go          # Model[T], SortCriterion
├── selection/           # Selection subsystem
│   └── selection.go     # Model, Mode, Kind, Rect, Position
├── grouping/            # Row grouping subsystem
│   └── grouping.go      # Model[T], BuildGroups, FlattenGroups, Aggregate
├── internal/
│   └── lineedit/        # Line editing widget for filter/editor input
│       └── lineedit.go  # Model with cursor, text editing, key handling
└── examples/            # Example programs
    ├── basic/           # Reflection-based columns, quick filter
    ├── columns/         # Manual column defs, filters, grouping, aggregation
    ├── csv/             # CSV file viewer with type inference
    ├── jsonl/           # JSONL file viewer with map[string]any rows
    ├── hscroll/         # Horizontal scrolling with pinned columns
    ├── anyrow/          # Heterogeneous row types using []any
    ├── selection/       # Multi-selection with status bar
    └── spreadsheet/     # Full spreadsheet with formulas, formatting, file I/O
```

#### 3.3 Package Dependency Graph

```
grid      →  data, filter, sort, selection, grouping, internal/lineedit
data      →  filter, internal/lineedit
filter    →  internal/lineedit
sort      →  data (uses SortDirection)
grouping  →  data (uses RowNode, Column)
selection →  (standalone)
internal/lineedit → (standalone)
```

`grid/` is the integration package that composes all others. The sub-packages are
independently usable and have no dependency on `grid/`.

---

### 4. Core Types

#### 4.1 Model

```go
package grid

// Model is the top-level grid component. T is the type of each row's data.
type Model[T any] struct {
    // Public fields following bubbles convention
    KeyMap   KeyMap
    Help     help.Model

    // Internal state (unexported)
    cols        []data.Column[T]
    colGroups   []data.ColumnGroup[T]
    rows        []data.RowNode[T]
    pinnedTop   []data.RowNode[T]
    pinnedBot   []data.RowNode[T]
    rowIDFunc   func(T) string

    displayRows []data.RowNode[T]  // cached display rows
    dirty       bool               // triggers recomputation
    colWidths   []int

    vp          viewport           // virtual scrolling state
    sel         selection.Model    // selection state
    sortModel   sort.Model[T]     // sort state
    groupModel  grouping.Model[T]  // grouping state
    editState   *editState[T]      // nil when not editing

    // Filtering state (inline, no separate filter model)
    quickFilterEnabled bool
    quickFilterText    string
    quickFilterActive  bool
    filterEditColIdx   int
    externalFilter     func(T) bool

    // Pinning
    pinnedTopFunc   func(T) bool
    pinnedBotFunc   func(T) bool
    staticPinnedTop []T
    staticPinnedBot []T

    // Row height
    defaultRowHeight int
    dynamicRowHeight func(T) int

    // Layout
    width       int
    height      int
    focused     bool
    focusedCell CellPosition
    styles      Styles
    editable    bool
    postSort    func([]data.RowNode[T]) []data.RowNode[T]
}

type CellPosition struct {
    Row int
    Col int
}

func New[T any](opts ...Option[T]) Model[T]
```

#### 4.2 Column Definition

Columns are the heart of the grid's configuration. Each `Column` describes how a
single column is sourced, displayed, sorted, filtered, and edited.

```go
package data

// Column defines a single column in the grid.
type Column[T any] struct {
    // Identity
    ColumnID   string  // Unique identifier. Required.
    HeaderName string  // Display name in the header row.

    // Data access
    ValueGetter    func(T) any                    // Extracts the cell value from the row data. Required.
    ValueFormatter func(value any, data T) string // Format the value for display.

    // Sizing
    Width    int  // Fixed width in terminal columns. 0 = auto.
    MinWidth int  // Minimum width (default: 4).
    MaxWidth int  // Maximum width. 0 = unconstrained.
    Flex     int  // Flex weight for distributing remaining space. 0 = no flex.

    // Sorting
    Sortable   bool                // Default: true.
    Comparator func(a, b any) int // Custom comparator.

    // Filtering
    Filterable bool          // Default: true.
    Filter     filter.Filter // Column filter (Text, Number, Set, Bool, Time, or custom).

    // Pinning
    Pinned     Pin  // PinLeft, PinRight, or PinNone.
    LockPinned bool // Prevent user from changing pin state.

    // Cell rendering
    CellRenderer         CellRenderer[T]            // Custom renderer.
    CellRendererSelector func(T) CellRenderer[T]    // Dynamic renderer per row.
    CellStyle            func(value any, data T) lipgloss.Style // Per-cell styling.

    // Cell editing
    Editable    bool                      // Default: false.
    CellEditor  CellEditor[T]            // Custom editor.
    ValueSetter func(data *T, value any) // Write the edited value back.

    // Column spanning
    ColumnSpan func(data T) int // Number of columns this cell spans. Default: 1.

    // Aggregation
    AggFunc       string                 // "sum", "avg", "count", "min", "max".
    AggFuncCustom func(values []any) any // Custom aggregation.

    // Visibility
    Hide     bool // If true, column is not rendered.
    NoSelect bool // If true, column is skipped during selection navigation.
}

// Pin is used for both column pinning (Left/Right) and row pinning (Top/Bottom).
type Pin int

const (
    PinNone   Pin = iota
    PinLeft        // Column: pinned to left edge.
    PinRight       // Column: pinned to right edge.
    PinTop         // Row: pinned to top.
    PinBottom      // Row: pinned to bottom.
)

type SortDirection int

const (
    SortNone SortDirection = iota
    SortAsc
    SortDesc
)
```

**Column Inference**

The `data` package provides two convenience functions for deriving columns
automatically:

```go
// FromType derives columns from T's exported struct fields via reflection.
// Each field becomes a Column with ColumnID and HeaderName set to the field name,
// and a pre-built ValueGetter closure.
func FromType[T any]() []Column[T]

// FromRows infers columns from sample data. Supports map[string]any, []any,
// and struct types. Automatically assigns appropriate filters (TextFilter,
// NumberFilter, BoolFilter) based on inferred value types.
func FromRows[T any](rows []T) []Column[T]
```

**Column Groups**

Column groups produce a single level of grouped headers. Each group spans its
child columns with a shared header label:

```go
type ColumnGroup[T any] struct {
    HeaderName string
    Columns    []Column[T]   // Leaf columns in this group.
}
```

#### 4.3 Row Node

Each row of user data is wrapped in a `RowNode` that carries runtime metadata:

```go
package data

type RowNode[T any] struct {
    // Data is the user-supplied row value.
    Data T

    // Runtime state (managed by the grid)
    ID         string          // Unique row ID. Auto-generated if not set via RowID option.
    RowIndex   int             // Current display index (post sort/filter/group).
    Expanded   bool            // For group rows.
    RowHeight  int             // In terminal lines. Default: 1.
    Pinned     Pin             // PinTop, PinBottom, or PinNone.
    IsGroup    bool            // True if this is a synthetic group row.
    GroupKey   string          // The value this group represents.
    GroupLevel int             // Nesting depth (0 = top).
    Children   []*RowNode[T]
    Parent     *RowNode[T]
    AggValues  map[string]any  // Computed aggregate values (for group rows).
}
```

---

### 5. Feature Design

#### 5.1 Virtual Scrolling

The grid virtualizes both rows and columns. Only cells within the visible viewport
are rendered. This is critical for large data sets.

```go
type viewport struct {
    topRow      int // Index of the first visible row.
    leftCol     int // Index of the first visible (unpinned) column.
    visibleLines int // Number of terminal lines in the viewport.
    visibleCols int  // Number of columns that fit in the viewport width.
}
```

**Rendering pipeline:**

1. Compute the sorted, filtered, grouped row list (the "display rows").
2. Slice display rows to `[topRow, topRow + visibleLines]`, accounting for variable
   row heights.
3. For each visible row, render only columns in the visible column range (accounting
   for pinned columns which are always rendered).
4. Assemble the final output string by joining pinned-left columns, viewport columns,
   and pinned-right columns with border separators.

Scrolling is triggered by cursor movement, page up/down, and half-page (Ctrl+U/D)
navigation. The viewport supports variable row heights, walking rows to determine
visibility.

#### 5.2 Column Sizing

Columns are sized in a multi-pass algorithm inspired by CSS flexbox:

1. **Hidden columns**: Columns with `Hide: true` get width 0.
2. **Border accounting**: If `BorderColumn` is enabled, `n-1` separator characters
   are subtracted from available width.
3. **Fixed columns**: Columns with an explicit `Width` are allocated exactly that many
   terminal columns.
4. **Minimum pass**: All remaining (flex) columns are allocated at least `MinWidth`
   columns (default: 4).
5. **Flex pass**: Remaining space is distributed proportionally to each column's `Flex`
   weight. Columns without a `Flex` value receive `Flex = 1` by default. A cumulative
   division algorithm prevents pixel loss from integer rounding.
6. **Max clamp**: Any column exceeding `MaxWidth` is clamped.

When the terminal is resized (`tea.WindowSizeMsg`), the entire sizing pass is re-run.

#### 5.3 Sorting

Sorting is a first-class feature with full multi-column support.

**State:**

```go
package sort

type Model[T any] struct {
    SortOrder []SortCriterion // Ordered list of active sorts.
    MultiSort bool            // Whether multi-column sort is enabled.
}

type SortCriterion struct {
    ColumnID  string
    Direction data.SortDirection // Asc or Desc.
}
```

**Methods:**

```go
func (m *Model[T]) ToggleSort(colID string)                    // Cycles asc -> desc -> none.
func (m *Model[T]) AddSort(colID string)                       // Adds to multi-sort or toggles if present.
func (m *Model[T]) Clear()                                     // Removes all sort criteria.
func (m *Model[T]) DirectionFor(colID string) data.SortDirection
func (m *Model[T]) IndexFor(colID string) int                  // Returns 0-based sort index or -1.
```

**Behavior:**

| Action                 | Effect                                           |
|------------------------|--------------------------------------------------|
| Enter on header cell   | Toggle sort on that column (asc -> desc -> none) |
| Shift+Enter on header  | Add column to multi-sort                         |
| `s` from any row       | Sort by current column                           |
| `S` from any row       | Add current column to multi-sort                 |
| Programmatic API       | `SetSort([]SortCriterion)`                       |

**Custom comparators** receive two values to compare:

```go
Comparator func(a, b any) int
```

The built-in `defaultCompare` handles `string`, `int`, `int64`, `float64`, `bool`,
and `time.Time` with type-specific fast paths, falling back to `fmt.Sprintf`
comparison.

**Post-sort hook** allows reordering after the grid's sort completes:

```go
WithPostSort(func(rows []data.RowNode[T]) []data.RowNode[T])
```

After any sort change, the display row list is recomputed and the viewport is
adjusted to keep the previously focused row visible.

#### 5.4 Filtering

The grid supports three filtering mechanisms that compose together (all filters must
pass for a row to be visible):

**5.4.1 Column Filters**

Each column can have an attached filter. Built-in filter types:

```go
package filter

// Filter is the interface that column filters implement.
type Filter interface {
    // Matches returns true if the value passes the filter.
    Matches(value any) bool

    // View renders the filter's UI (e.g., a text input in the header).
    View() string

    // Update processes messages for the filter's UI.
    Update(msg tea.Msg) (Filter, tea.Cmd)

    // Active returns true if the filter is currently constraining results.
    Active() bool

    // Clear resets the filter to its default (non-active) state.
    Clear()
}
```

**Message types for filter focus management:**

```go
type FilterFocusMsg struct {
    Width    int
    MaxLines int
}
type FilterBlurMsg struct{}
```

Built-in filters:

| Filter           | Description                                                          | UI              |
|------------------|----------------------------------------------------------------------|-----------------|
| `TextFilter`     | Substring / regex match on string values.                            | Text input      |
| `NumberFilter`   | Comparison operators (=, !=, <, >, <=, >=) or range (e.g. `10..50`).| Text input      |
| `SetFilter`      | Include/exclude from a set of distinct values. Interactive checkbox list with search. | Checkbox list |
| `BoolFilter`     | Cycles through any / true-only / false-only.                         | Toggle          |
| `TimeFilter`     | Date/time range filter. Supports `start..end` or single date. Parses common formats. | Text input |

**5.4.2 Quick Filter**

A single text input that searches across all columns:

```go
WithQuickFilter(enabled bool)
WithQuickFilterText(text string) // Set initial filter text.
```

When active, the quick filter renders above the grid. Each word in the input is
matched independently (all words must match somewhere in the row, case-insensitive).

**5.4.3 External Filter**

An application-supplied predicate that runs on every row:

```go
WithExternalFilter(func(data T) bool)
```

This is useful for filters driven by external UI elements outside the grid.

#### 5.5 Selection

The selection model uses a **rectangular selection** approach, supporting cell ranges,
full rows, and full columns.

```go
package selection

type Mode int

const (
    SelectNone   Mode = iota // No selection.
    SelectSingle             // At most one selection rect.
    SelectMulti              // Multiple selection rects.
)

type Kind int

const (
    KindNone    Kind = iota
    KindRect         // Arbitrary rectangular selection (Shift+nav).
    KindFullRow      // Full-row selection.
    KindFullCol      // Full-column selection.
)

type Position struct {
    Row, Col int
}

type Rect struct {
    Kind   Kind
    Anchor Position // Starting corner of selection.
    Cursor Position // Current corner of selection.
}

type Model struct {
    Mode  Mode
    Rects []Rect
}
```

**Methods:**

```go
func New(mode Mode) Model
func (m *Model) Active() bool
func (m *Model) Clear()
func (m *Model) Replace(r Rect)
func (m *Model) ToggleFullRow(row int)
func (m *Model) ContainsCell(row, col int) bool
func (m *Model) FullRowRanges() [][2]int
func (m *Model) BoundingRect() (rowLo, rowHi, colLo, colHi int)
```

**Key bindings:**

| Key             | Action                                    |
|-----------------|-------------------------------------------|
| Space           | Toggle current row (full-row selection)   |
| `R`             | Select full row                           |
| `C`             | Select full column                        |
| Shift+arrows    | Extend rectangular selection              |
| Ctrl+A          | Select all rows                           |
| Escape          | Deselect all                              |

#### 5.6 Cell Rendering

Cells are rendered through a pipeline:

```
ValueGetter(data) -> raw value
      │
      ▼
ValueFormatter(value, data) -> display string
      │
      ▼
CellRenderer.Render(CellContext) -> styled string (lipgloss)
      │
      ▼
CellStyle(value, data) -> final style applied
```

**CellRenderer interface:**

```go
package data

// CellContext provides all information a renderer needs.
type CellContext[T any] struct {
    Value          any              // The raw cell value.
    FormattedValue string           // After ValueFormatter.
    Data           T                // The full row data.
    RowNode        *RowNode[T]
    Column         *Column[T]
    ColumnIndex    int
    RowIndex       int
    IsSelected     bool
    IsFocused      bool
    Width          int              // Available width in terminal columns.
    Height         int              // Available height in terminal lines.
}

// CellRenderer renders a cell's content.
type CellRenderer[T any] interface {
    Render(ctx CellContext[T]) string
}

// CellRendererFunc is a convenience adapter.
type CellRendererFunc[T any] func(ctx CellContext[T]) string

func (f CellRendererFunc[T]) Render(ctx CellContext[T]) string {
    return f(ctx)
}
```

**Built-in renderers:**

| Renderer              | Description                                                      |
|-----------------------|------------------------------------------------------------------|
| `TextRenderer`        | Default. Truncates/pads text to fit width.                       |
| `NumberRenderer`      | Right-aligned, optional thousands separator (`ThousandsSep`).    |
| `TimeRenderer`        | Renders `time.Time` values. Configurable `Format` string (default: `2006-01-02 15:04`). Supports `Relative` display (e.g. "2h ago"). |
| `BarRenderer`         | Horizontal bar proportional to `MaxValue`.                       |
| `SparklineRenderer`   | Inline sparkline for `[]float64` series using block characters.  |
| `BoolRenderer`        | Renders `✓` / `✗` or custom `TrueGlyph`/`FalseGlyph`.          |
| `ProgressRenderer`    | Mini progress bar within the cell (`FilledChar`/`EmptyChar`).    |

#### 5.7 Cell Editing

When a cell is editable, the user can enter edit mode. The grid transitions the
focused cell from display mode to edit mode, swapping the renderer for an editor.

```go
package data

// CellEditor handles inline editing of a cell value.
type CellEditor[T any] interface {
    // Init is called when editing begins. Returns initial command.
    Init(ctx CellContext[T]) tea.Cmd

    // Update handles messages while editing.
    Update(msg tea.Msg) (CellEditor[T], tea.Cmd)

    // View renders the editor UI.
    View() string

    // Value returns the current edited value.
    Value() any

    // Validate returns an error string if the value is invalid, or "".
    Validate() string
}
```

**Built-in editors:**

| Editor            | Constructor                          | Description                              |
|-------------------|--------------------------------------|------------------------------------------|
| `TextEditor`      | `NewTextEditor[T]()`                 | Single-line text input.                  |
| `NumberEditor`    | `NewNumberEditor[T]()`               | Numeric input with `WithMin`/`WithMax`/`WithStep` builder methods. |
| `SelectEditor`    | `NewSelectEditor[T](options []string)` | Cycle through a list of options.       |
| `BoolEditor`      | `NewBoolEditor[T]()`                 | Toggle true/false.                       |
| `TimeEditor`      | `NewTimeEditor[T]()`                 | Text input for `time.Time`. Accepts multiple human-readable formats. |

**Editing lifecycle:**

1. User presses Enter (or F2) on an editable cell -> `CellEditingStartedMsg`
2. Grid replaces the cell renderer with the cell editor.
3. Keystrokes are routed to the editor's `Update` method.
4. User presses Enter to confirm or Escape to cancel.
5. On confirm: `Validate()` is called. If valid, `ValueSetter` writes the value back
   to the row data and `CellValueChangedMsg` is emitted. If invalid, the editor
   remains active with the validation error displayed.
6. On cancel: `CellEditingCancelledMsg` is emitted, original value restored.

#### 5.8 Column Pinning

Columns with `Pinned: PinLeft` are always rendered at the left edge. Columns with
`Pinned: PinRight` are always rendered at the right edge. The center viewport scrolls
horizontally between them.

```
┌──────────┬────────────────────────────────┬─────────┐
│  Pinned  │     Scrollable Viewport        │ Pinned  │
│  Left    │  ◄──── horizontal scroll ────► │ Right   │
│          │                                │         │
└──────────┴────────────────────────────────┴─────────┘
```

**Rendering**: The view is assembled from three independently rendered column regions:

1. Pinned-left columns (fixed, always visible).
2. Center viewport columns (scrollable).
3. Pinned-right columns (fixed, always visible).

Scroll indicators (left/right arrows) are shown when columns are off-screen.

Column pinning can be set declaratively at construction time:

```go
WithColumnPin[T](colID string, dir data.Pin)
```

Or programmatically at runtime:

```go
func (m *Model[T]) PinColumn(colID string, dir data.Pin)
func (m *Model[T]) UnpinColumn(colID string)
```

#### 5.9 Row Pinning

Rows can be pinned to the top or bottom of the grid. Pinned rows are always visible
regardless of scroll position.

```go
// Dynamic pinning via predicate:
WithPinnedTopRows(func(data T) bool)
WithPinnedBottomRows(func(data T) bool)

// Static pinned data:
WithStaticPinnedTop(rows []T)
WithStaticPinnedBottom(rows []T)
```

**Rendering**: The viewport is divided vertically:

```
┌────────────────────────────────────────┐
│ Pinned Top Rows                        │ ← always visible
├────────────────────────────────────────┤
│                                        │
│ Scrollable Row Viewport                │ ← virtual-scrolled
│                                        │
├────────────────────────────────────────┤
│ Pinned Bottom Rows                     │ ← always visible
└────────────────────────────────────────┘
```

Pinned rows participate in column alignment and styling but are excluded from sorting
and scrolling.

Runtime pinning API:

```go
func (m *Model[T]) PinRow(id string, pos data.Pin)
func (m *Model[T]) UnpinRow(id string)
```

#### 5.10 Row Grouping

Grouping is configured via the `WithGrouping` option, specifying column IDs to group
by:

```go
package grouping

type Model[T any] struct {
    GroupColumns    []string        // ColumnIDs of columns being grouped, in order.
    Expanded        map[string]bool // GroupKey -> expanded state.
    DefaultExpanded int             // Number of levels expanded by default. -1 = all.
}
```

**Methods:**

```go
func New[T any](groupCols []string, defaultExpanded int) Model[T]
func (m *Model[T]) IsExpanded(groupKey string) bool
func (m *Model[T]) SetExpanded(groupKey string, expanded bool)
func (m *Model[T]) ExpandAll(groups []*data.RowNode[T])
func (m *Model[T]) CollapseAll(groups []*data.RowNode[T])
func (m *Model[T]) ToggleGroupColumn(colID string)
```

**Package-level functions:**

```go
func BuildGroups[T any](rows []data.RowNode[T], cols []data.Column[T], groupCols []string,
    expanded map[string]bool, defaultExpanded int) []*data.RowNode[T]
func FlattenGroups[T any](groups []*data.RowNode[T]) []data.RowNode[T]
func Aggregate(values []any, funcName string) any
```

**Display**: Group rows are synthetic rows inserted into the display list. They show
the group value and aggregated data. Child rows are indented and only visible when
the group is expanded.

**Aggregation**: Group rows display aggregated values for columns. Built-in
aggregation functions:

| Function | Description                   |
|----------|-------------------------------|
| `sum`    | Sum of child values.          |
| `avg`    | Average of child values.      |
| `count`  | Number of children.           |
| `min`    | Minimum child value.          |
| `max`    | Maximum child value.          |
| `first`  | First child's value.          |
| `last`   | Last child's value.           |

Custom aggregation functions can be registered via `Column.AggFuncCustom`.

**Interaction:**

| Key                  | Action                                    |
|----------------------|-------------------------------------------|
| Enter on group row   | Toggle group expansion                    |
| Right on group row   | Expand focused group                      |
| Left on group row    | Collapse focused group                    |
| Ctrl+Shift+Right     | Expand all groups                         |
| Ctrl+Shift+Left      | Collapse all groups                       |
| `G`                  | Toggle group-by for current column        |

#### 5.11 Column Spanning

A column can declare `ColumnSpan func(data T) int` to allow its cells to span multiple
columns. This is useful for "full-width" detail rows or report-style layouts.

When a cell spans multiple columns, the spanned columns are skipped in rendering.
Column spanning is confined to within a single pin region (left, center, or right).

#### 5.12 Keyboard Navigation

Navigation follows AG Grid's model adapted for TUI conventions:

```go
type KeyMap struct {
    // Cell navigation
    Up, Down, Left, Right       key.Binding
    PageUp, PageDown            key.Binding
    HalfPageUp, HalfPageDown    key.Binding
    Home, End                   key.Binding
    LineStart, LineEnd          key.Binding
    GoToHeader                  key.Binding

    // Selection
    Select, SelectAll, DeselectAll key.Binding
    SelectRow, SelectColumn        key.Binding
    ShiftUp, ShiftDown, ShiftLeft, ShiftRight key.Binding

    // Sorting
    ToggleSort, ToggleMultiSort key.Binding
    SortColumn, MultiSortColumn key.Binding

    // Editing
    StartEdit, ConfirmEdit, CancelEdit key.Binding

    // Filtering
    QuickFilter, ColumnFilter key.Binding

    // Grouping
    ToggleGroupColumn, ToggleGroup key.Binding
    ExpandGroup, CollapseGroup     key.Binding
    ExpandAll, CollapseAll         key.Binding

    // General
    Help key.Binding
}
```

**Default key bindings:**

| Action           | Keys                  |
|------------------|-----------------------|
| Up               | `↑`, `k`              |
| Down             | `↓`, `j`              |
| Left             | `←`, `h`              |
| Right            | `→`, `l`              |
| PageUp           | `PgUp`                |
| PageDown         | `PgDn`                |
| HalfPageUp       | `Ctrl+U`              |
| HalfPageDown     | `Ctrl+D`              |
| Home             | `Home`                |
| End              | `End`                 |
| LineStart        | `0`                   |
| LineEnd          | `$`                   |
| GoToHeader       | `g`                   |
| Select           | `Space`               |
| SelectAll        | `Ctrl+A`              |
| DeselectAll      | `Esc`                 |
| SelectRow        | `R`                   |
| SelectColumn     | `C`                   |
| ShiftUp          | `Shift+↑`, `K`        |
| ShiftDown        | `Shift+↓`, `J`        |
| ShiftLeft        | `Shift+←`, `H`        |
| ShiftRight       | `Shift+→`, `L`        |
| ToggleSort       | `Enter` (on header)   |
| ToggleMultiSort  | `Shift+Enter` (header)|
| SortColumn       | `s` (from any row)    |
| MultiSortColumn  | `S` (from any row)    |
| StartEdit        | `Enter`, `F2`         |
| ConfirmEdit      | `Enter`               |
| CancelEdit       | `Esc`                 |
| QuickFilter      | `/`                   |
| ColumnFilter     | `Ctrl+F`              |
| ToggleGroupColumn| `G`                   |
| ToggleGroup      | `Enter` (on group)    |
| ExpandGroup      | `→` (on group row)    |
| CollapseGroup    | `←` (on group row)    |
| ExpandAll        | `Ctrl+Shift+→`        |
| CollapseAll      | `Ctrl+Shift+←`        |
| Help             | `?`                   |

**Focus model**: The grid has a "focused cell" (`CellPosition{Row, Col}`) that acts
as a cursor. Arrow keys move the focus. The focused cell is visually highlighted.
When the focus moves beyond the visible viewport, the viewport scrolls to keep it
visible.

**Header mode**: When the cursor is moved above the first row (Up from row 0), focus
shifts to the header row. In header mode, Left/Right navigates between column
headers, and Enter toggles sort.

**Update routing**: `Update()` dispatches `tea.KeyPressMsg` based on mode:

1. **Not focused**: No-op.
2. **Editing** (`editState != nil`): Enter confirms (validates, applies via
   ValueSetter), Esc cancels, else routes to editor's `Update`.
3. **Filter editing** (`filterEditColIdx >= 0`): Enter applies filter and closes,
   Esc clears filter and closes, else routes to filter's `Update`.
4. **Quick filter active**: Esc clears, Backspace/Space/Enter/runes modify text.
5. **Normal**: Full navigation, sorting, selection, editing, filtering, grouping.

---

### 6. Styling

Styling uses lipgloss throughout and is fully customizable.

```go
type Styles struct {
    // Table-level
    Table       lipgloss.Style // Outer container.

    // Header
    Header      lipgloss.Style // Header row.
    HeaderCell  lipgloss.Style // Individual header cell.
    SortAsc     string         // Ascending sort indicator (default: "▲").
    SortDesc    string         // Descending sort indicator (default: "▼").

    // Cells
    Cell         lipgloss.Style // Default cell style.
    CellFocused  lipgloss.Style // Focused cell highlight.
    CellSelected lipgloss.Style // Selected cell/row highlight.
    CellEvenRow  lipgloss.Style // Base cell style for even-indexed rows.
    CellOddRow   lipgloss.Style // Base cell style for odd-indexed rows.
    CellPinned   lipgloss.Style // Base cell style for pinned rows.

    // Pinning
    PinnedLeft    lipgloss.Style // Pinned-left region.
    PinnedRight   lipgloss.Style // Pinned-right region.
    PinSeparator  string         // Vertical separator between pinned and scrollable (default: "|").
    ScrollLeft    string         // Left scroll indicator (default: "◀").
    ScrollRight   string         // Right scroll indicator (default: "▶").

    // Grouping
    GroupRow       lipgloss.Style
    GroupExpanded  string // Expanded indicator (default: "▼").
    GroupCollapsed string // Collapsed indicator (default: "▶").
    GroupIndent    int    // Indentation per level (default: 2).

    // Borders
    Border       lipgloss.Border // Border style (e.g., lipgloss.RoundedBorder()).
    BorderHeader bool            // Show border below header (default: true).
    BorderRow    bool            // Show border between rows.
    BorderColumn bool            // Show border between columns.

    // Filtering
    FilterInput  lipgloss.Style // Filter editor input.
    FilterMatch  lipgloss.Style // Highlighted matching text.
    FilterActive string         // Filter-active indicator in header.

    // Editing
    EditorInput lipgloss.Style // Cell editor input.
    EditorError lipgloss.Style // Validation error.

    // Per-cell styling callback (overrides the above for fine-grained control)
    StyleFunc func(row, col int, data any) lipgloss.Style
}

func DefaultStyles() Styles
```

The `StyleFunc` callback mirrors lipgloss/table's `StyleFunc` and provides maximum
flexibility. Row index `-1` represents the header row.

---

### 7. Options API

The grid is configured via functional options at construction time. All options can
also be changed at runtime via setter methods on the model.

```go
// Construction
func New[T any](opts ...Option[T]) Model[T]

// Core options
func WithColumns[T any](cols []data.Column[T]) Option[T]
func WithColumnGroups[T any](groups []data.ColumnGroup[T]) Option[T]
func WithRows[T any](rows []T) Option[T]
func WithRowID[T any](fn func(T) string) Option[T]
func WithWidth[T any](w int) Option[T]
func WithHeight[T any](h int) Option[T]
func WithStyles[T any](s Styles) Option[T]
func WithKeyMap[T any](km KeyMap) Option[T]
func WithFocused[T any](f bool) Option[T]
func WithFocusedCell[T any](pos CellPosition) Option[T]

// Feature toggles
func WithSelection[T any](mode selection.Mode) Option[T]
func WithEditable[T any](enabled bool) Option[T]
func WithQuickFilter[T any](enabled bool) Option[T]
func WithQuickFilterText[T any](text string) Option[T]
func WithExternalFilter[T any](fn func(T) bool) Option[T]
func WithColumnFilter[T any](colID string, f filter.Filter) Option[T]

// Sorting
func WithDefaultSort[T any](criteria []sort.SortCriterion) Option[T]
func WithMultiSort[T any](enabled bool) Option[T]
func WithPostSort[T any](fn func([]data.RowNode[T]) []data.RowNode[T]) Option[T]

// Grouping
func WithGrouping[T any](cols ...string) Option[T]
func WithGroupDefaultExpanded[T any](levels int) Option[T]

// Pinning
func WithColumnPin[T any](colID string, dir data.Pin) Option[T]
func WithPinnedTopRows[T any](fn func(T) bool) Option[T]
func WithPinnedBottomRows[T any](fn func(T) bool) Option[T]
func WithStaticPinnedTop[T any](rows []T) Option[T]
func WithStaticPinnedBottom[T any](rows []T) Option[T]

// Row configuration
func WithRowHeight[T any](height int) Option[T]
func WithDynamicRowHeight[T any](fn func(T) int) Option[T]
```

Note: `WithRows` and `WithColumnFilter` are deferred -- rows are set and filters
applied after all other options, so column definitions and settings are available
when rows are processed.

---

### 8. Messages (Msg Types)

The grid communicates with the parent application via Bubble Tea messages:

```go
// Navigation
type FocusChangedMsg struct {
    Position CellPosition
    Previous CellPosition
}

// Selection
type SelectionChangedMsg[T any] struct {
    Regions  []SelectionRegion
    Selected []data.RowNode[T]
}

type SelectionRegion struct {
    Kind   SelectionKind        // SelectionNone, SelectionRect, SelectionFullRow, SelectionFullCol
    Anchor CellPosition
    Cursor CellPosition
}

// Sorting
type SortChangedMsg struct {
    SortOrder []sort.SortCriterion
}

// Filtering
type FilterChangedMsg struct {
    ColumnID string
    Active   bool
}

type QuickFilterChangedMsg struct {
    Text string
}

// Editing
type CellEditingStartedMsg   struct { Position CellPosition }
type CellEditingConfirmedMsg struct { Position CellPosition }
type CellValueChangedMsg[T any] struct {
    Position CellPosition
    OldValue any
    NewValue any
    Data     T
}
type CellEditingCancelledMsg struct { Position CellPosition }

// Grouping
type GroupExpandedMsg  struct { GroupKey string; Level int }
type GroupCollapsedMsg struct { GroupKey string; Level int }
type GroupColumnsChangedMsg struct { GroupColumns []string }
```

---

### 9. Public API (Methods on Model)

```go
// --- Elm Architecture ---
func (m Model[T]) Init() tea.Cmd
func (m Model[T]) Update(msg tea.Msg) (Model[T], tea.Cmd)
func (m Model[T]) View() string

// --- Data ---
func (m *Model[T]) SetRows(rows []T)
func (m Model[T]) Rows() []T
func (m *Model[T]) SetColumns(cols []data.Column[T])
func (m Model[T]) Columns() []data.Column[T]
func (m *Model[T]) UpdateRow(id string, d T)
func (m *Model[T]) InsertRow(index int, d T)
func (m *Model[T]) RemoveRow(id string)

// --- Dimensions ---
func (m *Model[T]) SetWidth(w int)
func (m *Model[T]) SetHeight(h int)
func (m Model[T]) Width() int
func (m Model[T]) Height() int

// --- Focus ---
func (m *Model[T]) Focus()
func (m *Model[T]) Blur()
func (m Model[T]) Focused() bool
func (m *Model[T]) SetFocusedCell(pos CellPosition)
func (m Model[T]) FocusedCell() CellPosition
func (m Model[T]) FocusedRowData() (T, bool)
func (m Model[T]) Filtering() bool

// --- Selection ---
func (m *Model[T]) SetRowSelection(id string)
func (m *Model[T]) ToggleRowSelection(id string)
func (m *Model[T]) SetColumnSelection(colIdx int)
func (m *Model[T]) SetRectSelection(anchor, cursor CellPosition)
func (m *Model[T]) SelectAllRows()
func (m *Model[T]) ClearSelection()
func (m Model[T]) Selection() []SelectionRegion
func (m Model[T]) HasSelection() bool
func (m Model[T]) IsCellSelected(row, col int) bool
func (m Model[T]) IsRowSelected(id string) bool
func (m Model[T]) IsColumnSelected(colIdx int) bool
func (m Model[T]) SelectedRows() []T
func (m Model[T]) SelectedRowNodes() []*data.RowNode[T]
func (m Model[T]) SelectionBounds() (rowLo, rowHi, colLo, colHi int)

// --- Sorting ---
func (m *Model[T]) SetSort(criteria []sort.SortCriterion)
func (m Model[T]) SortOrder() []sort.SortCriterion

// --- Filtering ---
func (m *Model[T]) SetQuickFilter(text string)
func (m *Model[T]) SetColumnFilter(colID string, f filter.Filter)
func (m *Model[T]) ClearFilters()

// --- Grouping ---
func (m *Model[T]) ExpandGroup(groupKey string)
func (m *Model[T]) CollapseGroup(groupKey string)
func (m *Model[T]) ExpandAll()
func (m *Model[T]) CollapseAll()

// --- Scrolling ---
func (m *Model[T]) ScrollToRow(index int)
func (m *Model[T]) ScrollToRowByID(id string) bool
func (m *Model[T]) ScrollToTop()
func (m *Model[T]) ScrollToBottom()

// --- Pinning ---
func (m *Model[T]) PinColumn(colID string, dir data.Pin)
func (m *Model[T]) UnpinColumn(colID string)
func (m *Model[T]) PinRow(id string, pos data.Pin)
func (m *Model[T]) UnpinRow(id string)

// --- Help ---
func (m Model[T]) HelpView() string
func (m Model[T]) ShortHelp() []key.Binding
func (m Model[T]) FullHelp() [][]key.Binding
```

---

### 10. Rendering Pipeline

The `View()` method follows this pipeline:

```
 1. Quick filter bar (if active)
         │
         ▼
 2. Column filter editor (if active)
         │
         ▼
 3. Column group headers (if ColumnGroups defined)
         │
         ▼
 4. Column headers with sort indicators and filter-active indicators
         │
         ▼
 5. Header border (if BorderHeader enabled)
         │
         ▼
 6. Pinned top rows
         │
         ▼
 7. Separator
         │
         ▼
 8. Visible body rows (from virtual scroll window)
    For each row:
    ├─ If group row: render group row with indent + expand indicator + agg values
    └─ If data row:
       For each visible column:
       ├─ Get value: ValueGetter(data)
       ├─ Format: ValueFormatter
       ├─ Render: CellRenderer or default TextRenderer
       ├─ Apply style: CellStyle, then Styles.Cell/Selected/Focused/Even/Odd
       └─ Pad/truncate to column width
         │
         ▼
 9. Separator
         │
         ▼
10. Pinned bottom rows
         │
         ▼
11. Pad or truncate to grid dimensions
         │
         ▼
12. Apply table-level style
```

Each row is assembled from three column regions: pinned-left, center viewport, and
pinned-right, joined with pin separators. Scroll indicators appear when columns are
off-screen.

---

### 11. Data Flow: Display Row Pipeline

When any of sorting, filtering, or grouping state changes, the display row list is
recomputed via `recomputeDisplayRows()`:

```
Source rows ([]RowNode[T])
    │
    ▼
[1] Pin separation: extract rows with Pinned == PinTop/PinBottom
    │                (dynamic predicates or per-row Pinned field)
    │
    ▼
[2] External filter: WithExternalFilter(fn)
    │
    ▼
[3] Column filters: each active Filter.Matches()
    │                (skips column being actively edited)
    │
    ▼
[4] Quick filter: all words must appear in concatenated column values (case-insensitive)
    │
    ▼
[5] Append static pinned rows (WithStaticPinnedTop/Bottom)
    │
    ▼
[6] Grouping (if GroupColumns non-empty):
    │  BuildGroups() → sort within groups → FlattenGroups()
    │
    ▼
[7] Sorting (if no grouping): sort using SortOrder + column comparators
    │
    ▼
[8] Post-sort hook (if set)
    │
    ▼
[9] Update RowIndex on each display row
    │
    ▼
[10] Compute AggValues for group nodes
    │
    ▼
Display rows (used by viewport and rendering)
```

This pipeline is executed lazily: the result is cached and only recomputed when the
`dirty` flag is set (by data changes, filter state, sort state, or group state
changes). Public setters eagerly recompute; `Init()` is a no-op.

---

### 12. Example Usage

```go
package main

import (
    "fmt"
    "os"

    tea "charm.land/bubbletea/v2"
    "github.com/pgavlin/tea-grid/data"
    "github.com/pgavlin/tea-grid/filter"
    "github.com/pgavlin/tea-grid/grid"
    "github.com/pgavlin/tea-grid/selection"
)

type Employee struct {
    Name       string
    Department string
    Salary     float64
    Active     bool
}

func main() {
    cols := []data.Column[Employee]{
        {
            ColumnID:    "name",
            HeaderName:  "Employee Name",
            ValueGetter: func(e Employee) any { return e.Name },
            Pinned:      data.PinLeft,
            MinWidth:    20,
            Flex:        2,
            Filterable:  true,
            Filter:      filter.NewTextFilter(),
        },
        {
            ColumnID:    "department",
            HeaderName:  "Dept",
            ValueGetter: func(e Employee) any { return e.Department },
            Width:       15,
            Sortable:    true,
            Filter:      filter.NewSetFilter(),
        },
        {
            ColumnID:    "salary",
            HeaderName:  "Salary",
            ValueGetter: func(e Employee) any { return e.Salary },
            Width:       12,
            Sortable:    true,
            ValueFormatter: func(v any, _ Employee) string {
                return fmt.Sprintf("$%.0f", v.(float64))
            },
            AggFunc: "sum",
        },
        {
            ColumnID:    "active",
            HeaderName:  "Active",
            ValueGetter: func(e Employee) any { return e.Active },
            Width:       8,
            CellRenderer: data.BoolRenderer[Employee]{
                TrueGlyph:  "✓",
                FalseGlyph: "✗",
            },
            Editable:    true,
            CellEditor:  data.NewBoolEditor[Employee](),
            ValueSetter: func(e *Employee, v any) { e.Active = v.(bool) },
        },
    }

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
        grid.WithSelection[Employee](selection.SelectMulti),
        grid.WithQuickFilter[Employee](true),
        grid.WithGrouping[Employee]("department"),
        grid.WithGroupDefaultExpanded[Employee](-1),
        grid.WithFocused[Employee](true),
    )

    p := tea.NewProgram(g, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

---

### 13. Performance Considerations

| Concern | Approach |
|---------|----------|
| Large datasets (10k+ rows) | Virtual scrolling; only render visible rows. |
| Sort/filter on large data | Performed on the Go side; O(n log n) sort, O(n) filter. Cache results via `dirty` flag. |
| Frequent re-renders | Minimize string allocations in `View()`. Pre-compute column widths. Use `strings.Builder`. |
| Wide tables (many columns) | Column virtualization: only render columns in the visible horizontal range. |
| Grouping overhead | Build group tree on recompute; flatten to display list. |
| Resize events | Re-layout column widths on `tea.WindowSizeMsg`. |
| Reflection | `data.FromType[T]()` uses reflection once at init to generate `ValueGetter` closures. No reflection in hot paths. |

---

### 14. Accessibility & Terminal Compatibility

- All features are keyboard-accessible. Mouse is an optional enhancement.
- Renders correctly in 80-column terminals (graceful truncation).
- Supports `NO_COLOR` via lipgloss's color profile detection.
- Works with screen readers via sequential, meaningful text rendering in `View()`.

---

### 15. Future Extensions (Out of Scope for v1)

These features are explicitly deferred but the architecture accommodates them:

- **Server-side data source**: A `DataSource` interface that returns pages of rows
  asynchronously, enabling infinite scroll against remote APIs.
- **Master-detail rows**: Expandable rows that show a nested sub-grid or custom
  detail view.
- **Column reordering**: Via keyboard commands (move column left/right).
- **Copy/export**: Copy selected rows to clipboard as TSV; export as CSV.
- **Undo/redo**: Transaction log for cell edits with Ctrl+Z / Ctrl+Y.

---

### 16. Design Decisions

1. **Generics constraint**: `T` is constrained to `any`. Row identity is handled via
   `WithRowID(func(T) string)` rather than requiring `comparable`. This keeps the API
   flexible for users whose row types contain slices, maps, or other non-comparable
   fields.

2. **No reflection for data access**: `ValueGetter` is required on every `Column`.
   The convenience functions `FromType[T]()` and `FromRows[T](rows)` use reflection
   once at init time to generate column lists with pre-built `ValueGetter` closures,
   so users who want the simple struct-mirroring behavior get it without per-access
   reflection cost.

3. **Unified Pin type**: A single `Pin` enum (`PinNone`, `PinLeft`, `PinRight`,
   `PinTop`, `PinBottom`) is used for both column pinning and row pinning, simplifying
   the type hierarchy.

4. **Rectangular selection model**: Rather than a simple row-based selection, the
   selection package uses rectangular regions (`Rect` with `Anchor` and `Cursor`
   positions). This enables cell-range selection, full-row selection, and full-column
   selection through a unified model.

5. **Single-level column groups**: For v1, `ColumnGroup` supports only one level of
   grouping (a group header spanning its child columns). Nested column groups are
   deferred to a future version. This simplifies header rendering and keyboard
   navigation in the header region.

6. **Custom border rendering**: The grid implements its own border rendering rather
   than delegating to `lipgloss/table`. This avoids the overhead of reconstructing a
   `lipgloss/table.Table` on every frame and gives full control over border drawing
   in the presence of pinned regions, column spanning, and virtual scrolling.

7. **Consolidated data package**: Column definitions, row nodes, cell contexts, cell
   renderers, and cell editors all live in the `data` package rather than separate
   `column/`, `row/`, and `cell/` packages. This reduces import boilerplate and
   reflects their tight coupling.

8. **Bubble Tea v2**: The grid targets Bubble Tea v2 (`charm.land/bubbletea/v2`) and
   Lipgloss v2 (`charm.land/lipgloss/v2`), using the `charm.land` module paths.
   `Update()` returns `(Model[T], tea.Cmd)` (concrete type) rather than
   `(tea.Model, tea.Cmd)`.
