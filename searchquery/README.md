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

Full ISO timestamps accepted; the time portion is stripped. The same
shape applies to numeric ranges — see the open follow-up below.

## Open follow-ups

- **Generic `Range[T]`.** `filter.NumberFilter` and this package's
  `ParseTimeRange` parse the same shape against different value types.
  Parameterize once, drop into both.
- **In-memory binder.** Walk an `AST` against a single row, given a
  `Vocabulary` + per-field accessor/comparator functions. Mirror of the
  SQL binders that live in caller-side code.
- **Query-bar widget + round-tripping.** A textinput-backed UI that
  parses on every keystroke, `SetClause`s each column's filter, and
  re-renders the bar from the active filter set when those filters
  change inline. Round-trip-safety requires every UI-expressible filter
  state to be representable as query syntax (or marked unsuited for the
  bar — e.g. regex-mode TextFilter).
- **Multi-clause-on-same-field.** GitHub-style label semantics —
  repeated clauses AND together — needs a filter type that accepts a
  list of constraints. Either extend `SetFilter` with a "match-all"
  mode or introduce a sibling.

## License

Same as tea-grid (root `LICENSE`).
