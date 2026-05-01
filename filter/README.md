# filter

Per-column filters for tea-grid. Defines the `Filter` interface and the
built-in filter implementations: `TextFilter`, `NumberFilter`,
`SetFilter`, `BoolFilter`, `TimeFilter`, and `MultiSetFilter`.

## Filter interface

```go
type Filter interface {
    Matches(value any) bool
    View() string
    Update(msg tea.Msg) (Filter, tea.Cmd)
    Active() bool
    Clear()
}
```

A `Filter` lives on a `data.Column[T].Filter` field. The grid invokes
`Matches` against the cell value during the display-row pipeline and
routes `Update` events to the focused filter while it is being edited.

## Built-in filters

- **TextFilter** -- substring match. Optional regex mode via `SetRegex(true)`.
- **NumberFilter** -- comparison operators (`=`, `!=`, `<`, `>`, `<=`, `>=`)
  and ranges (`5..20`). Accepts `int`, `int64`, `float32`, `float64`.
- **SetFilter** -- include/exclude from a fixed set of distinct values.
  Inline checkbox UI in the popup. OR semantics within the included subset.
- **BoolFilter** -- three states: any, true only, false only.
- **TimeFilter** -- date or date range. Accepts a wide set of layouts
  (see `parseTime`).
- **MultiSetFilter** -- AND-of-includes for slice-valued columns. The
  popup is view-and-remove only; new constraints come from the query bar
  (see below). Default matcher handles `[]string` row values; supply
  `WithMultiSetMatcher` for richer slice element types.

## RoundTrippable

Filters that can serialize their state to and from a query-bar clause
implement `RoundTrippable`:

```go
type RoundTrippable interface {
    SetClause(values []string, negate bool) error
    Clause() (values []string, negate bool, ok bool)
}
```

Every built-in filter implements `RoundTrippable`. Filters that do not
are treated as opaque by the query bar — annotated, not edited.

`Clause` returns `ok=false` when the filter is inactive (no constraint
to surface) or in a state that can not be expressed in the query syntax.
The query bar treats both `ok=false` states the same way: skip in the
bar text, mention in the lossy annotation if `Active()` is true.

### SetFilter negation flip

`SetFilter.Clause` returns the included subset by default. Once more
than half of `allValues` is included, it flips to the excluded subset
with `negate=true` to keep the bar text small.

`SetFilter` operates on a known `allValues` set. New values that appear
in row data after a clause was applied are treated as "not included" --
consistent with `Include`/`SetValues` semantics.

### TextFilter regex mode is lossy

`TextFilter` in regex mode returns `ok=false` from `Clause`; the query
bar annotates the column as a hidden filter rather than rendering a
`text:` clause for it.

`SetClause` on a `TextFilter` resets regex mode. The rule is uniform:
submitting a clause for a column overrides whatever state that filter
was in, lossy or not.

### MultiSetFilter editing model

`MultiSetFilter` accumulates constraints with AND semantics. The popup
shows the current constraint list with `× constraint` rows; arrow keys
move row focus, `d` (or backspace) removes the focused constraint.
There is no add affordance in the popup -- new constraints come from
the query bar. The footer reads:

```
d delete · esc close · / edit
```

`SetClause` is called once per repeated bar clause; each call appends
one constraint. v1 rejects multi-value (comma-list) values inside a
single clause.

## License

Same as tea-grid (root `LICENSE`).
