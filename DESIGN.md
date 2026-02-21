# tea-grid: AG Grid-Inspired Table Component for Bubble Tea

## Design Document

### 1. Overview

`tea-grid` is a high-performance, feature-rich data grid component for
[Bubble Tea](https://github.com/charmbracelet/bubbletea). It brings the power and
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
            │  (grid.go)  │
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
github.com/charmbracelet/tea-grid/
├── grid/            # Core grid model, update, view
│   ├── grid.go      # Model[T], New(), Update(), View()
│   ├── options.go   # Option functions
│   ├── render.go    # View rendering pipeline
│   ├── viewport.go  # Virtual scrolling
│   ├── keymap.go    # KeyMap and default bindings
│   └── styles.go    # Styles and DefaultStyles()
├── column/          # Column definition types
│   └── column.go    # ColDef[T], ColGroup[T], PinDirection
├── row/             # Row node types
│   └── row.go       # RowNode[T], PinPosition
├── cell/            # Cell rendering and editing
│   ├── renderer.go  # CellRenderer interface
│   ├── editor.go    # CellEditor interface
│   └── builtin.go   # Built-in renderers/editors
├── filter/          # Filtering subsystem
│   ├── filter.go    # Filter interface
│   └── builtin.go   # Text, Number, Set, Bool, Time filters
├── sort/            # Sorting subsystem
│   └── sort.go      # SortModel, Comparator
├── selection/       # Selection subsystem
│   └── selection.go # SelectionModel, modes
├── grouping/        # Row grouping subsystem
│   └── grouping.go  # GroupModel, aggregation
└── examples/        # Example programs
    ├── basic/
    ├── editable/
    ├── grouping/
    └── kitchen-sink/
```

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
    cols        []column.ColDef[T]
    colGroups   []column.ColGroup[T]
    rows        []row.RowNode[T]
    pinnedTop   []row.RowNode[T]
    pinnedBot   []row.RowNode[T]

    viewport    viewport          // virtual scrolling state
    selection   selection.Model   // selection state
    sortModel   sort.Model[T]     // sort state
    filterModel filter.Model[T]   // filter state
    groupModel  grouping.Model[T] // grouping state
    editState   *editState[T]     // nil when not editing

    width       int
    height      int
    focused     bool
    focusedCell CellPosition      // {Row, Col} of the focused cell

    styles      Styles
    ready       bool              // true after first WindowSizeMsg
}

type CellPosition struct {
    Row int
    Col int
}

func New[T any](opts ...Option[T]) Model[T]
```

#### 4.2 Column Definition

Columns are the heart of the grid's configuration. Each `ColDef` describes how a
single column is sourced, displayed, sorted, filtered, and edited.

```go
package column

// ColDef defines a single column in the grid.
type ColDef[T any] struct {
    // Identity
    ColID      string  // Unique identifier. Required.
    HeaderName string  // Display name in the header row.

    // Data access
    ValueGetter func(T) any         // Extracts the cell value from the row data. Required.
    ValueFormatter func(value any, data T) string // Format the value for display.

    // Sizing
    Width    int  // Fixed width in terminal columns. 0 = auto.
    MinWidth int  // Minimum width (default: 4).
    MaxWidth int  // Maximum width. 0 = unconstrained.
    Flex     int  // Flex weight for distributing remaining space. 0 = no flex.

    // Sorting
    Sortable   bool                                      // Default: true.
    Comparator func(a, b any, nodeA, nodeB *RowNode[T], isDesc bool) int // Custom sort.
    SortIndex  int  // Initial sort priority (0 = primary). -1 = not sorted.
    SortDir    SortDirection // Asc, Desc, or None.

    // Filtering
    Filterable bool             // Default: true.
    Filter     filter.Filter    // Column filter (Text, Number, Set, or custom).

    // Pinning
    Pinned PinDirection // Left, Right, or None.
    LockPinned bool     // Prevent user from changing pin state.

    // Cell rendering
    CellRenderer         CellRenderer[T]            // Custom renderer.
    CellRendererSelector func(T) CellRenderer[T]    // Dynamic renderer per row.
    CellStyle            func(value any, data T) lipgloss.Style // Per-cell styling.

    // Cell editing
    Editable   bool                       // Default: false.
    CellEditor CellEditor[T]              // Custom editor.
    ValueSetter func(data *T, value any)  // Write the edited value back.

    // Column spanning
    ColSpan func(data T) int // Number of columns this cell spans. Default: 1.

    // Grouping
    RowGroup    bool   // If true, rows are grouped by this column's values.
    AggFunc     string // Aggregation function name: "sum", "avg", "count", "min", "max".
    AggFuncCustom func(values []any) any // Custom aggregation.

    // Visibility
    Hide bool // If true, column is not rendered.
}

// Columns returns a []ColDef[T] derived from T's exported struct fields.
// For each exported field, it produces a ColDef with:
//   - ColID and HeaderName set to the field name
//   - ValueGetter set to a function that retrieves the field's value from T
// This is a convenience for the common case where the column list mirrors the
// struct layout. Callers can modify the returned slice to customize individual
// columns (e.g., override HeaderName, set Width, attach a Filter, etc.).
//
// Panics if T is not a struct type.
func Columns[T any]() []ColDef[T]

type PinDirection int

const (
    PinNone  PinDirection = iota
    PinLeft
    PinRight
)

type SortDirection int

const (
    SortNone SortDirection = iota
    SortAsc
    SortDesc
)
```

**Column Groups**

Column groups produce a single level of grouped headers. Each group spans its
child columns with a shared header label:

```go
type ColGroup[T any] struct {
    HeaderName string
    Children   []ColDef[T]   // Leaf columns in this group.
}
```

#### 4.3 Row Node

Each row of user data is wrapped in a `RowNode` that carries runtime metadata:

```go
package row

type RowNode[T any] struct {
    // Data is the user-supplied row value.
    Data T

    // Runtime state (managed by the grid)
    ID         string       // Unique row ID. Auto-generated if not set via RowID option.
    RowIndex   int          // Current display index (post sort/filter/group).
    Selected   bool
    Expanded   bool         // For group rows.
    RowHeight  int          // In terminal lines. Default: 1.
    Pinned     PinPosition  // Top, Bottom, or None.
    IsGroup    bool         // True if this is a synthetic group row.
    GroupKey   string       // The value this group represents.
    GroupLevel int          // Nesting depth (0 = top).
    Children   []*RowNode[T]
    Parent     *RowNode[T]
}

type PinPosition int

const (
    PinNone   PinPosition = iota
    PinTop
    PinBottom
)
```

---

### 5. Feature Design

#### 5.1 Virtual Scrolling

The grid virtualizes both rows and columns. Only cells within the visible viewport
(plus a configurable buffer) are rendered. This is critical for large data sets.

```go
type viewport struct {
    topRow      int // Index of the first visible row.
    leftCol     int // Index of the first visible (unpinned) column.
    visibleRows int // Number of rows that fit in the viewport height.
    visibleCols int // Number of columns that fit in the viewport width.
    rowBuffer   int // Extra rows to render above/below viewport (default: 5).
}
```

**Rendering pipeline:**

1. Compute the sorted, filtered, grouped row list (the "display rows").
2. Slice display rows to `[topRow - rowBuffer, topRow + visibleRows + rowBuffer]`.
3. For each visible row, render only columns in the visible column range (accounting
   for pinned columns which are always rendered).
4. Assemble the final output string by joining pinned-left columns, viewport columns,
   and pinned-right columns with border separators.

Scrolling is triggered by cursor movement, page up/down, and mouse wheel events.

#### 5.2 Column Sizing

Columns are sized in a multi-pass algorithm inspired by CSS flexbox:

1. **Fixed columns**: Columns with an explicit `Width` are allocated exactly that many
   terminal columns.
2. **Minimum pass**: All remaining columns are allocated at least `MinWidth` columns.
3. **Flex pass**: Remaining space is distributed proportionally to each column's `Flex`
   weight. Columns without a `Flex` value receive `Flex = 1` by default.
4. **Max clamp**: Any column exceeding `MaxWidth` is clamped, and the freed space is
   redistributed to columns that can still grow.

When the terminal is resized (`tea.WindowSizeMsg`), the entire sizing pass is re-run.

#### 5.3 Sorting

Sorting is a first-class feature with full multi-column support.

**State:**

```go
package sort

type Model[T any] struct {
    SortOrder []SortCriterion // Ordered list of active sorts.
}

type SortCriterion struct {
    ColID     string
    Direction SortDirection // Asc or Desc.
}
```

**Behavior:**

| Action                 | Effect                                           |
|------------------------|--------------------------------------------------|
| Enter on header cell   | Toggle sort on that column (asc -> desc -> none) |
| Shift+Enter on header  | Add column to multi-sort                         |
| Programmatic API       | `SetSort([]SortCriterion)`                       |

**Cycle order** is configurable per column:

```go
SortingOrder []SortDirection // e.g., {SortAsc, SortDesc} to skip "none"
```

**Custom comparators** receive the full row node for context:

```go
Comparator func(a, b any, nodeA, nodeB *RowNode[T], isDesc bool) int
```

**Post-sort hook** allows reordering after the grid's sort completes:

```go
WithPostSort(func(rows []RowNode[T]) []RowNode[T])
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
    // Returns empty string if the filter has no inline UI.
    View() string

    // Update processes messages for the filter's UI.
    Update(msg tea.Msg) (Filter, tea.Cmd)

    // Active returns true if the filter is currently constraining results.
    Active() bool
}
```

Built-in filters:

| Filter           | Description                                                          | UI              |
|------------------|----------------------------------------------------------------------|-----------------|
| `TextFilter`     | Substring / regex match on string values.                            | Text input      |
| `NumberFilter`   | Comparison operators (=, !=, <, >, <=, >=) or range (e.g. `10..50`).| Text input      |
| `SetFilter`      | Include/exclude from a set of distinct values.                       | Checkbox list   |
| `BoolFilter`     | True / false / any.                                                  | Toggle          |
| `TimeFilter`     | Date/time range filter. Accepts a start..end range as text input. Parses dates in common human-readable formats (e.g. `2024-01-01`, `Jan 2 2024`) as well as RFC1123Z. Either bound may be omitted for an open-ended range (e.g. `2024-01-01..` or `..2024-06-30`). | Text input |

**5.4.2 Quick Filter**

A single text input that searches across all columns:

```go
WithQuickFilter(enabled bool)
```

When active, the quick filter renders above the grid. Each word in the input is
matched independently (all words must match somewhere in the row). Matching cells
are highlighted.

**5.4.3 External Filter**

An application-supplied predicate that runs on every row:

```go
WithExternalFilter(func(data T) bool)
```

This is useful for filters driven by external UI elements outside the grid.

#### 5.5 Row Selection

```go
package selection

type Mode int

const (
    SelectNone   Mode = iota // No selection.
    SelectSingle             // At most one row selected.
    SelectMulti              // Multiple rows via Space, Shift+arrows, Ctrl+A.
)

type Model struct {
    Mode     Mode
    selected map[string]bool // RowNode.ID -> selected
    anchor   int             // For shift-selection range
}
```

**Key bindings:**

| Key             | SelectSingle          | SelectMulti                     |
|-----------------|-----------------------|---------------------------------|
| Space           | Toggle current row    | Toggle current row              |
| Shift+Up/Down   | --                    | Extend selection range          |
| Ctrl+A          | --                    | Select all (visible) rows       |
| Escape          | Deselect              | Deselect all                    |

**Checkbox column**: When `SelectMulti` is enabled, an optional checkbox column can
be prepended automatically:

```go
WithSelectionColumn(enabled bool)
```

**API:**

```go
func (m Model[T]) SelectedRows() []T
func (m Model[T]) SelectedRowNodes() []*RowNode[T]
func (m Model[T]) IsSelected(id string) bool
```

**Messages emitted:**

```go
type RowSelectedMsg[T any] struct {
    Row   RowNode[T]
    Selected bool
}

type SelectionChangedMsg[T any] struct {
    Selected []RowNode[T]
}
```

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
package cell

// CellContext provides all information a renderer needs.
type CellContext[T any] struct {
    Value          any            // The raw cell value.
    FormattedValue string         // After ValueFormatter.
    Data           T              // The full row data.
    RowNode        *row.RowNode[T]
    ColDef         *column.ColDef[T]
    ColIndex       int
    RowIndex       int
    IsSelected     bool
    IsFocused      bool
    Width          int            // Available width in terminal columns.
    Height         int            // Available height in terminal lines.
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
| `NumberRenderer`      | Right-aligned, optional thousands separator.                     |
| `TimeRenderer`        | Renders `time.Time` values. Configurable format string (default: `2006-01-02 15:04`). Supports relative display (e.g. "2h ago") via an optional `Relative bool` flag. |
| `BarRenderer`         | Renders a horizontal bar proportional to value.                  |
| `SparklineRenderer`   | Inline sparkline for numeric series.                             |
| `BoolRenderer`        | Renders `✓` / `✗` or custom true/false glyphs.                  |
| `ProgressRenderer`    | Mini progress bar within the cell.                               |

#### 5.7 Cell Editing

When a cell is editable, the user can enter edit mode. The grid transitions the
focused cell from display mode to edit mode, swapping the renderer for an editor.

```go
package cell

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

| Editor            | Description                                          |
|-------------------|------------------------------------------------------|
| `TextEditor`      | Single-line text input (wraps `textinput.Model`).    |
| `NumberEditor`    | Numeric input with optional min/max/step.            |
| `SelectEditor`    | Cycle through a list of options.                     |
| `BoolEditor`      | Toggle true/false.                                   |
| `TimeEditor`      | Text input for `time.Time` values. Accepts human-readable formats (e.g. `Jan 2 2024 3:04pm`, `2024-01-02 15:04`) and RFC1123Z (`Mon, 02 Jan 2006 15:04:05 -0700`). Validates on confirm and displays a parse error if the input is not recognized. |

**Editing lifecycle:**

1. User presses Enter (or F2) on an editable cell -> `CellEditingStartedMsg`
2. Grid replaces the cell renderer with the cell editor.
3. Keystrokes are routed to the editor's `Update` method.
4. User presses Enter to confirm or Escape to cancel.
5. On confirm: `Validate()` is called. If valid, `ValueSetter` writes the value back
   to the row data and `CellValueChangedMsg` is emitted. If invalid, the editor
   remains active with the validation error displayed.
6. On cancel: `CellEditingCancelledMsg` is emitted, original value restored.

**Messages:**

```go
type CellEditingStartedMsg struct { Position CellPosition }
type CellEditingStoppedMsg struct { Position CellPosition }
type CellValueChangedMsg[T any] struct {
    Position CellPosition
    OldValue any
    NewValue any
    Data     T
}
type CellEditingCancelledMsg struct { Position CellPosition }
```

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

These are joined horizontally using `lipgloss.JoinHorizontal`. A vertical border
separator distinguishes pinned regions from the scrollable center.

**Constraint**: Combined pinned column width must not exceed `gridWidth - 10` to
ensure the center viewport remains usable. If it does, an error message is displayed
or columns are auto-unpinned.

#### 5.9 Row Pinning

Rows can be pinned to the top or bottom of the grid. Pinned rows are always visible
regardless of scroll position.

```go
WithPinnedTopRows(func(data T) bool)
WithPinnedBottomRows(func(data T) bool)

// Or supply static pinned data:
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
and scrolling. Selection of pinned rows is optional (`WithPinnedRowSelection(bool)`).

#### 5.10 Row Grouping

When one or more columns have `RowGroup: true`, the grid organizes rows into a
tree of groups.

```go
package grouping

type Model[T any] struct {
    GroupColumns []string // ColIDs of columns being grouped, in order.
    Expanded     map[string]bool // GroupKey -> expanded state.
    DefaultExpanded int // Number of levels expanded by default. -1 = all.
}
```

**Display**: Group rows are synthetic rows inserted into the display list. They show
the group value and aggregated data. Child rows are indented and only visible when
the group is expanded.

```
▶ Country: United States (3)    $1,200,000    142
  ├─ Alice Johnson               $450,000      52
  ├─ Bob Smith                   $380,000      45
  └─ Carol Davis                 $370,000      45
▼ Country: United Kingdom (2)    $820,000      89
  ├─ ...
```

**Aggregation**: Group rows display aggregated values for numeric columns. Built-in
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

Custom aggregation functions can be registered via `ColDef.AggFuncCustom`.

**Interaction:**

| Key         | Action                                    |
|-------------|-------------------------------------------|
| Enter/Right | Expand focused group                      |
| Left        | Collapse focused group                    |
| Shift+Right | Expand all groups at the current level    |
| Shift+Left  | Collapse all groups at the current level  |

#### 5.11 Column Spanning

A column can declare `ColSpan func(data T) int` to allow its cells to span multiple
columns. This is useful for "full-width" detail rows or report-style layouts.

```go
ColDef[T]{
    Field:      "name",
    HeaderName: "Name",
    ColSpan: func(data T) int {
        if data.IsHeader {
            return 5 // Span across 5 columns
        }
        return 1
    },
}
```

When a cell spans multiple columns, the spanned columns are skipped in rendering.
Column spanning is confined to within a single pin region (left, center, or right).

#### 5.12 Keyboard Navigation

Navigation follows AG Grid's model adapted for TUI conventions:

```go
type KeyMap struct {
    // Cell navigation
    Up           key.Binding // ↑ or k
    Down         key.Binding // ↓ or j
    Left         key.Binding // ← or h
    Right        key.Binding // → or l
    PageUp       key.Binding
    PageDown     key.Binding
    HalfPageUp   key.Binding // Ctrl+U
    HalfPageDown key.Binding // Ctrl+D
    Home         key.Binding // Go to first row
    End          key.Binding // Go to last row
    LineStart    key.Binding // Go to first column
    LineEnd      key.Binding // Go to last column

    // Header navigation
    GoToHeader   key.Binding // g then h (chord)

    // Selection
    Select       key.Binding // Space
    SelectRange  key.Binding // Shift+Up / Shift+Down
    SelectAll    key.Binding // Ctrl+A

    // Sorting (when header is focused)
    ToggleSort      key.Binding // Enter
    ToggleMultiSort key.Binding // Shift+Enter

    // Editing
    StartEdit    key.Binding // Enter or F2
    ConfirmEdit  key.Binding // Enter
    CancelEdit   key.Binding // Escape

    // Filtering
    OpenFilter      key.Binding // / (slash)
    CloseFilter     key.Binding // Escape
    QuickFilter     key.Binding // Ctrl+F

    // Grouping
    ExpandGroup     key.Binding // Enter or →
    CollapseGroup   key.Binding // ← or Backspace
    ExpandAll       key.Binding // Shift+→
    CollapseAll     key.Binding // Shift+←

    // General
    Quit         key.Binding // q (if enabled)
    Help         key.Binding // ?
}
```

**Focus model**: The grid has a "focused cell" (`CellPosition{Row, Col}`) that acts
as a cursor. Arrow keys move the focus. The focused cell is visually highlighted.
When the focus moves beyond the visible viewport, the viewport scrolls to keep it
visible.

**Header mode**: When the cursor is moved above the first row (Up from row 0), focus
shifts to the header row. In header mode, Left/Right navigates between column
headers, and Enter toggles sort.

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
    CellSelected lipgloss.Style // Selected row highlight.
    CellEvenRow  lipgloss.Style // Base cell style for even-indexed rows.
    CellOddRow   lipgloss.Style // Base cell style for odd-indexed rows.
    CellPinned   lipgloss.Style // Base cell style for pinned rows.

    // Pinning
    PinnedLeft   lipgloss.Style // Pinned-left region.
    PinnedRight  lipgloss.Style // Pinned-right region.
    PinSeparator string         // Vertical separator between pinned and scrollable.

    // Grouping
    GroupRow       lipgloss.Style
    GroupExpanded  string // Expanded indicator (default: "▼").
    GroupCollapsed string // Collapsed indicator (default: "▶").
    GroupIndent    int    // Indentation per level (default: 2).

    // Borders
    Border       lipgloss.Border // Border style (e.g., lipgloss.RoundedBorder()).
    BorderHeader bool            // Show border below header.
    BorderRow    bool            // Show border between rows.
    BorderColumn bool            // Show border between columns.

    // Filtering
    FilterInput lipgloss.Style  // Quick filter input.
    FilterMatch lipgloss.Style  // Highlighted matching text.

    // Editing
    EditorInput lipgloss.Style  // Cell editor input.
    EditorError lipgloss.Style  // Validation error.

    // Scrollbar
    Scrollbar         lipgloss.Style
    ScrollbarThumb    lipgloss.Style

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
func WithColumns[T any](cols []ColDef[T]) Option[T]
func WithColumnGroups[T any](groups []ColGroup[T]) Option[T]
func WithRows[T any](rows []T) Option[T]
func WithRowID[T any](fn func(T) string) Option[T]
func WithWidth[T any](w int) Option[T]
func WithHeight[T any](h int) Option[T]
func WithStyles[T any](s Styles) Option[T]
func WithKeyMap[T any](km KeyMap) Option[T]
func WithFocused[T any](f bool) Option[T]

// Feature toggles
func WithSelection[T any](mode SelectionMode) Option[T]
func WithSelectionColumn[T any](enabled bool) Option[T]
func WithEditable[T any](enabled bool) Option[T]
func WithQuickFilter[T any](enabled bool) Option[T]
func WithExternalFilter[T any](fn func(T) bool) Option[T]
func WithGrouping[T any](cols ...string) Option[T]
func WithGroupDefaultExpanded[T any](levels int) Option[T]

// Sorting
func WithDefaultSort[T any](criteria []SortCriterion) Option[T]
func WithMultiSort[T any](enabled bool) Option[T]
func WithPostSort[T any](fn func([]RowNode[T]) []RowNode[T]) Option[T]

// Pinning
func WithPinnedTopRows[T any](fn func(T) bool) Option[T]
func WithPinnedBottomRows[T any](fn func(T) bool) Option[T]
func WithStaticPinnedTop[T any](rows []T) Option[T]
func WithStaticPinnedBottom[T any](rows []T) Option[T]

// Row configuration
func WithRowHeight[T any](height int) Option[T]
func WithDynamicRowHeight[T any](fn func(T) int) Option[T]

// Scrolling
func WithRowBuffer[T any](n int) Option[T]

// Callbacks
func WithOnSelectionChanged[T any](fn func([]T)) Option[T]
func WithOnCellValueChanged[T any](fn func(CellValueChangedMsg[T])) Option[T]
func WithOnSortChanged[T any](fn func([]SortCriterion)) Option[T]
```

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
type RowSelectedMsg[T any]       struct { Row RowNode[T]; Selected bool }
type SelectionChangedMsg[T any]  struct { Selected []RowNode[T] }

// Sorting
type SortChangedMsg struct { SortOrder []SortCriterion }

// Filtering
type FilterChangedMsg struct { ColID string; Active bool }
type QuickFilterChangedMsg struct { Text string }

// Editing
type CellEditingStartedMsg    struct { Position CellPosition }
type CellEditingStoppedMsg    struct { Position CellPosition }
type CellValueChangedMsg[T any] struct {
    Position CellPosition
    OldValue any
    NewValue any
    Data     T
}
type CellEditingCancelledMsg  struct { Position CellPosition }

// Grouping
type GroupExpandedMsg  struct { GroupKey string; Level int }
type GroupCollapsedMsg struct { GroupKey string; Level int }

// Data
type RowsSetMsg[T any]    struct { Rows []T }
type ColumnsSetMsg[T any] struct { Cols []ColDef[T] }
```

---

### 9. Public API (Methods on Model)

```go
// --- Data ---
func (m *Model[T]) SetRows(rows []T)
func (m Model[T]) Rows() []T
func (m *Model[T]) SetColumns(cols []ColDef[T])
func (m Model[T]) Columns() []ColDef[T]
func (m *Model[T]) UpdateRow(id string, data T)
func (m *Model[T]) InsertRow(index int, data T)
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

// --- Selection ---
func (m Model[T]) SelectedRows() []T
func (m Model[T]) SelectedRowNodes() []*RowNode[T]
func (m *Model[T]) SelectRow(id string)
func (m *Model[T]) DeselectRow(id string)
func (m *Model[T]) SelectAll()
func (m *Model[T]) DeselectAll()
func (m Model[T]) IsSelected(id string) bool

// --- Sorting ---
func (m *Model[T]) SetSort(criteria []SortCriterion)
func (m Model[T]) SortOrder() []SortCriterion

// --- Filtering ---
func (m *Model[T]) SetQuickFilter(text string)
func (m *Model[T]) SetColumnFilter(colID string, filter filter.Filter)
func (m *Model[T]) ClearFilters()

// --- Grouping ---
func (m *Model[T]) ExpandGroup(groupKey string)
func (m *Model[T]) CollapseGroup(groupKey string)
func (m *Model[T]) ExpandAll()
func (m *Model[T]) CollapseAll()

// --- Scrolling ---
func (m *Model[T]) ScrollToRow(index int)
func (m *Model[T]) ScrollToTop()
func (m *Model[T]) ScrollToBottom()

// --- Pinning ---
func (m *Model[T]) PinColumn(colID string, dir PinDirection)
func (m *Model[T]) UnpinColumn(colID string)
func (m *Model[T]) PinRow(id string, pos PinPosition)
func (m *Model[T]) UnpinRow(id string)

// --- Bubble Tea interface ---
func (m Model[T]) Init() tea.Cmd
func (m Model[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model[T]) View() string
func (m Model[T]) HelpView() string
```

---

### 10. Rendering Pipeline

The `View()` method follows this pipeline:

```
1. Compute column widths (flex layout)
         │
         ▼
2. Partition columns: [pinned-left] [center] [pinned-right]
         │
         ▼
3. Render header row(s)
   ├─ Group headers (if ColGroups defined)
   └─ Column headers with sort indicators
         │
         ▼
4. Render pinned-top rows
         │
         ▼
5. Render visible body rows (from virtual scroll window)
   For each row:
   ├─ If group row: render group row with indent + expand indicator
   └─ If data row:
      For each visible column:
      ├─ Get value: ValueGetter(data)
      ├─ Format: ValueFormatter
      ├─ Render: CellRenderer or default TextRenderer
      ├─ Apply style: CellStyle, then Styles.Cell/Selected/Focused
      └─ Pad/truncate to column width
         │
         ▼
6. Render pinned-bottom rows
         │
         ▼
7. Render scrollbar (vertical, and horizontal if needed)
         │
         ▼
8. Render quick filter bar (if active)
         │
         ▼
9. Assemble:
   lipgloss.JoinVertical(
       quickFilterBar,
       headerRow,
       pinnedTopRows,
       lipgloss.JoinHorizontal(pinnedLeft, centerViewport, pinnedRight),
       pinnedBottomRows,
       statusBar,
   )
```

Each cell is rendered as a fixed-width string. Lipgloss styles are applied to
individual cells, then cells are joined with column separators. Borders are drawn
using the configured `lipgloss.Border` style.

---

### 11. Data Flow for Complex Operations

#### Sort + Filter + Group Interaction

When any of sorting, filtering, or grouping state changes, the display row list is
recomputed in this order:

```
Source rows ([]T)
    │
    ▼
[1] External filter: WithExternalFilter(fn)
    │
    ▼
[2] Column filters: each active Filter.Matches()
    │
    ▼
[3] Quick filter: substring match across all columns
    │
    ▼
Filtered rows
    │
    ▼
[4] Grouping: build group tree from GroupColumns
    │
    ▼
[5] Aggregation: compute aggregate values for each group node
    │
    ▼
[6] Sorting: sort within each group (or globally if ungrouped)
    │
    ▼
[7] Flatten: expand group tree to a flat display list
    │  (respecting collapsed state)
    │
    ▼
Display rows (used by viewport and rendering)
```

This pipeline is executed lazily: the result is cached and only recomputed when the
underlying data, filter state, sort state, or group state changes.

---

### 12. Example Usage

```go
package main

import (
    "fmt"
    "os"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/tea-grid/column"
    "github.com/charmbracelet/tea-grid/grid"
    "github.com/charmbracelet/tea-grid/filter"
)

type Employee struct {
    Name       string
    Department string
    Salary     float64
    Active     bool
}

func main() {
    cols := []column.ColDef[Employee]{
        {
            ColID:       "name",
            HeaderName:  "Employee Name",
            ValueGetter: func(e Employee) any { return e.Name },
            Pinned:      column.PinLeft,
            MinWidth:    20,
            Flex:        2,
            Filterable:  true,
            Filter:      filter.NewTextFilter(),
        },
        {
            ColID:       "department",
            HeaderName:  "Dept",
            ValueGetter: func(e Employee) any { return e.Department },
            Width:       15,
            Sortable:    true,
            RowGroup:    true,
            Filter:      filter.NewSetFilter(),
        },
        {
            ColID:       "salary",
            HeaderName:  "Salary",
            ValueGetter: func(e Employee) any { return e.Salary },
            Width:       12,
            Sortable:    true,
            ValueFormatter: func(v any, _ Employee) string {
                return fmt.Sprintf("$%,.0f", v.(float64))
            },
            AggFunc: "sum",
        },
        {
            ColID:       "active",
            HeaderName:  "Active",
            ValueGetter: func(e Employee) any { return e.Active },
            Width:       8,
            CellRenderer: cell.BoolRenderer[Employee]{
                TrueGlyph:  "✓",
                FalseGlyph: "✗",
            },
            Editable:   true,
            CellEditor:  cell.NewBoolEditor[Employee](),
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
        grid.WithSelectionColumn[Employee](true),
        grid.WithQuickFilter[Employee](true),
        grid.WithGrouping[Employee]("Department"),
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
| Large datasets (10k+ rows) | Virtual scrolling; only render visible rows + buffer. |
| Sort/filter on large data | Performed on the Go side; O(n log n) sort, O(n) filter. Cache results. |
| Frequent re-renders | Minimize string allocations in `View()`. Pre-compute column widths. Use `strings.Builder`. |
| Wide tables (many columns) | Column virtualization: only render columns in the visible horizontal range. |
| Grouping overhead | Build group tree once; update incrementally on data change. |
| Resize events | Debounce `WindowSizeMsg` handling; re-layout only after a brief pause. |

---

### 14. Accessibility & Terminal Compatibility

- All features are keyboard-accessible. Mouse is an optional enhancement.
- Renders correctly in 80-column terminals (graceful truncation).
- Supports `NO_COLOR` via lipgloss's color profile detection.
- Works with screen readers via sequential, meaningful text rendering in `View()`.
- Tested against: iTerm2, Terminal.app, Alacritty, Windows Terminal, tmux, screen.

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
- **Formula cells**: Computed columns with simple expression evaluation.
- **Frozen rows** (row spanning): A single row whose cells span the full width for
  section headers.

---

### 16. Design Decisions

1. **Generics constraint**: `T` is constrained to `any`. Row identity is handled via
   `WithRowID(func(T) string)` rather than requiring `comparable`. This keeps the API
   flexible for users whose row types contain slices, maps, or other non-comparable
   fields.

2. **No reflection for data access**: The `Field` string path has been dropped.
   `ValueGetter` is required on every `ColDef`. The convenience function
   `Columns[T]()` uses reflection once at init time to generate a default column list
   with pre-built `ValueGetter` functions, so users who want the simple struct-mirroring
   behavior get it without per-access reflection cost.

3. **Single-level column groups**: For v1, `ColGroup` supports only one level of
   grouping (a group header spanning its child columns). Nested column groups are
   deferred to a future version. This simplifies header rendering and keyboard
   navigation in the header region.

4. **Custom border rendering**: The grid implements its own border rendering rather
   than delegating to `lipgloss/table`. This avoids the overhead of reconstructing a
   `lipgloss/table.Table` on every frame and gives full control over border drawing
   in the presence of pinned regions, column spanning, and virtual scrolling.
