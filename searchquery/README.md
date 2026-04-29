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

### Generic `Range[T]`

`filter.NumberFilter` and this package's `ParseTimeRange` parse the same
shape (`>x`, `<x`, `>=x`, `<=x`, `a..b`, `*..b`, `a..*`) against
different value types. Parameterize once, drop into both:

```go
type Range[T any] struct {
    From, To       *T
    FromExclusive  bool
    ToExclusive    bool
}

func ParseRange[T any](s string, parse func(string) (T, error)) (Range[T], error)
```

`NumberFilter`'s `=`/`!=` operators stay as separate operator handling
on top of the range parser.

### In-memory binder

Walk an `AST` against a single row, given a `Vocabulary` + per-field
accessor/comparator functions. Mirror of the SQL binders that live in
caller-side code. Tea-grid columns already carry `ColumnID`, value
extractor, and a quick-filter hook — the column set is essentially the
`Vocabulary`. Bare terms feed the grid's existing quick-filter mechanism
(whitespace-AND across columns, substring match per word) directly.

### Query-bar widget + round-tripping

A textinput-backed UI that drives all column filters from a single
GitHub-style query string. Round-tripping in **both** directions: the
query bar can set filter state and the active filter state can be
rendered back as a query string, so users can edit either UI freely.

#### Proposed opt-in interface

Filters that opt into round-tripping implement:

```go
type RoundTrippable interface {
    // SetClause applies a parsed clause to the filter's state.
    SetClause(values []string, negate bool) error

    // Clause returns the filter's state as a clause. ok=false when
    // the filter is inactive (no constraint to surface in the bar).
    Clause() (values []string, negate bool, ok bool)
}
```

`TextFilter.SetText` and `NumberFilter.SetText` are already half of
this — formalize and add the reverse. Filters that don't implement
`RoundTrippable` are query-only-write or UI-only — round-trip just
skips them.

The bar then becomes:

```
query → filters:  for each ast.Clause, cols[c.Field].Filter.(RoundTrippable).SetClause(c.Values, c.Negate)
filters → query:  walk active column filters; each one that is RoundTrippable contributes one clause; concat with bare terms from the grid's quick filter text
```

#### Source of truth

Filters are canonical; the query bar is a view that re-renders on any
filter state change. User edits in the bar → `Parse` → `SetClause` per
column → filters update → bar re-renders from filters. Idempotent. Open
question: how often / by what mechanism the bar listens for filter
changes (poll on tick, explicit observer, Update message).

#### Lossy filter modes don't round-trip

Some filter states can't be expressed in the query syntax. The bar
should mark these as round-trip-blocking — typically by holding
read-only when the column's active filter has no `Clause()` and
showing a small "(custom filter)" annotation in the bar, or by
auto-expanding the syntax (rare; consider before adopting).

Known cases:

- **TextFilter regex mode** (`/foo.*bar/`) — syntax has no regex sigil.
  Either extend (`label:/foo.*bar/`) or mark blocking.
- **SetFilter exclusion-of-known-set** (excluded={a,b,c} of N total) —
  `-column:a,b,c` covers it only when the user thinks of it as
  "exclude these"; partial-include with mixed selections doesn't have a
  clean form.
- **SetFilter on dynamic value sets** — if new values appear (e.g. a
  newly-loaded row introduces a new label), an old `column:a,b,c`
  re-applied via the bar gets the new value as "not selected" —
  defensible but worth being explicit about.
- **Combined include + exclude on the same column** — uncommon enough
  that round-trip can punt; mark blocking.

#### Field name drift

Aliases canonicalize at parse time (`status:open` → `state:open`), so
the bar may "fix" the user's spelling on next render. Probably fine,
maybe surprising. A reasonable middle: preserve user input verbatim
until the next user edit, then re-render from the canonical state.

### Multi-clause-on-same-field (related to round-trip)

GitHub-style label semantics — repeated clauses AND together — needs a
filter type that accepts a list of constraints. Either extend
`SetFilter` with a "match-all" mode or introduce a sibling. This
intersects with round-tripping: such a filter must surface its
constraint list back through `Clause()` as a list of values (where the
field's "repeats AND" semantics let the bar choose between
`label:a,b,c` (OR) and `label:a label:b label:c` (AND) — and the parser
hands back exactly those distinct ASTs, so it's a binder-side decision).

## License

Same as tea-grid (root `LICENSE`).
