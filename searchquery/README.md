# searchquery

A small GitHub-style query language for filtering tabular data. Lives
alongside (and complements) [`filter/`](../filter/) and the grid's
quick-filter mechanism.

## Status

**Starting point.** Extracted from the
[vector](https://github.com/pgavlin/vector) project, where it ships with
SQL binders against issue/PR/external-record FTS tables. This package is
the parser + AST + `Vocabulary` registry — content-agnostic, no backend
assumed. The natural next step is a tea-grid binder that turns an AST
into per-column `filter.Filter` state and grid quick-filter text, with
round-tripping in both directions.

## Grammar

```
query    := token (WS+ token)*
token    := "-"? clause | bareword | quoted
clause   := ident ":" value ("," value)*
value    := bareword | quoted
```

- Bare terms accumulate in `AST.Terms` (space-joined). Drive the grid's
  quick-filter with these — same word-AND, substring-across-columns
  semantics.
- `field:value` clauses become typed `Clause` entries. Comma-OR within a
  value list; repeats on the same field default to AND across the AST
  (binders choose the exact semantics per field).
- Leading `-` negates the clause: `-state:closed`.
- Quoting wraps multi-word values: `label:"good first issue"`. Standard
  `\"` and `\\` escapes inside.
- Aliases (`Field.Aliases`) and parse-time rewrites (`Vocabulary.AddRewrite`)
  let users type sugar that canonicalizes before binders see it.

## Time grammar

`ParseTimeRange` handles GitHub's date filter syntax:

```
YYYY-MM-DD                exact date (From == To, both inclusive)
>YYYY-MM-DD               after that date (exclusive)
>=YYYY-MM-DD              on or after
<YYYY-MM-DD               before
<=YYYY-MM-DD              on or before
YYYY-MM-DD..YYYY-MM-DD    inclusive range
YYYY-MM-DD..*             open-ended upper
*..YYYY-MM-DD             open-ended lower
```

Full ISO timestamps accepted; the time portion is stripped.

`ParseTimeRange` is a one-liner over the generic `ParseRange[T]`,
which parses the same grammar against any value type. The caller
supplies the value parser:

```go
type Range[T any] struct {
    From, To                   *T
    FromExclusive, ToExclusive bool
}

func ParseRange[T any](s string, parse func(string) (T, error)) (Range[T], error)
```

`TimeRange` is a type alias for `Range[time.Time]`. `filter.NumberFilter`
uses `ParseRange[float64]` for its comparator/range handling; `=` and
`!=` stay as NumberFilter-specific operators on top.

## Use with tea-grid

The parser, AST, and `Vocabulary` registry are content-agnostic and
backend-agnostic. tea-grid wires them into `grid.Model[T]` via the
`internal/querybar` package; consumers enable the integration with
`grid.WithQueryBar()`. See `grid/README.md` for usage.

```go
g := grid.New(
    grid.WithColumns(cols),
    grid.WithQueryBar[Row](),
)
```

The vocabulary is auto-derived from columns. Override with
`grid.WithQueryBarVocabulary(v)` to add parse-time rewrites
(`is:open` → `state:open`) or queryable fields that do not correspond
to a single column.

## Open follow-ups

### Live bare-prefix while typing

The bar currently submits on Enter only. A future enhancement could
apply bare-term changes incrementally (matching today's quick-filter
UX) by splitting the parse into "definitely bare prefix" and "rest"
without invoking the full parser per keystroke.

### Tab-completion

The parser already exposes a `Vocabulary`. A Tab-completion path on
the bar would use it to suggest field names and values. Out of scope
for the v1 query-bar landing.

## License

Same as tea-grid (root `LICENSE`).
