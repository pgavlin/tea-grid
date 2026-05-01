# grid

Top-level tea-grid component. Composes `data`, `filter`, `sort`,
`grouping`, `selection`, and the internal `querybar` widget into a
single `Model[T]` that implements `tea.Model`.

## Quick start

```go
g := grid.New(
    grid.WithColumns(cols),
    grid.WithRows(rows),
    grid.WithRowID(func(r Row) string { return r.ID }),
    grid.WithFocused[Row](true),
)
```

The model is parameterized on the row type `T`; columns extract values
via `data.Column[T].Value`.

## Query bar

Enable the GitHub-style query bar with `WithQueryBar`:

```go
g := grid.New(
    grid.WithColumns(cols),
    grid.WithQueryBar[Row](),
)
```

The bar is rendered above the headers and serves as both editing
surface and canonical view of current filter state. Press `/` to focus
the textinput, type a query, press Enter to submit (or Esc to cancel).

### Field/value clauses

Field/value clauses (`state:open`, `count:>5`, `created:2026-01-01..*`)
route to per-column filters via the `RoundTrippable` interface. The
column's `ColumnID` is the field name by default; add aliases via
`Column.QueryAliases`:

```go
{ColumnID: "state", QueryAliases: []string{"status", "st"}, ...}
```

Bare terms (anything not a `field:value` token) feed the existing
quick-filter substring matcher.

### Vocabulary

The query-bar vocabulary is auto-derived from the column set. Every
visible, filterable column with a `RoundTrippable` filter becomes a
queryable field; field type is inferred from filter type. Override
with `WithQueryBarVocabulary` for parse-time rewrites (e.g. `is:open`
→ `state:open`) or for queryable fields that do not correspond to a
single column.

### Lossy filter states

Some filter states can not be expressed in the query syntax --
`TextFilter` in regex mode, non-`RoundTrippable` filters. The bar
annotates these inline:

```
state:open  bug    [+1 hidden filter: regex on title — esc to clear all]
```

Submit replaces only round-trippable clauses. Lossy filters are left
alone; clear them via the column UI or via the global `ClearFilters`
binding (default `esc`).

### Initial bar text

```go
grid.New(
    grid.WithColumns(cols),
    grid.WithQueryBar[Row](),
    grid.WithQueryBarText[Row]("state:open priority:>3"),
)
```

The text is parsed and applied at construction.

## Key bindings

Default bindings (customizable via `WithKeyMap`):

| Action            | Key       |
|-------------------|-----------|
| Open query bar    | `/`       |
| Submit query      | `Enter`   |
| Cancel bar edit   | `Esc`     |
| Clear all filters | `Esc`     |
| Open column filter| `Ctrl+F`  |
| Sort column       | `s`       |
| Multi-sort column | `S`       |
| Auto-fit column   | `w`       |
| Auto-fit all      | `W`       |

See `keymap.go` for the full list.

## License

Same as tea-grid (root `LICENSE`).
