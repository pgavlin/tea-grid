# GitHub-style query bar as a default tea-grid feature

**Status:** Draft
**Date:** 2026-05-01

## Problem

The `searchquery` package on this branch ships a parser, AST, and `Vocabulary` registry for a GitHub-style query language — content-agnostic, no backend assumed. Tea-grid has per-column filters (`TextFilter`, `NumberFilter`, `SetFilter`, `BoolFilter`, `TimeFilter`) and a quick-filter mechanism (bare-word substring match across columns), but no single textual surface that ties them together. A user filtering on `state`, `assignee`, and a free-text term has to open three popups.

The README's open follow-ups sketch the missing pieces: an in-memory binder that walks an AST against per-column filters, a `RoundTrippable` interface so filter state can be rendered back as query text, and a query-bar widget that drives both. This spec lands those pieces as a default tea-grid feature.

Round-trippability is the load-bearing requirement. The bar and the column filter UIs both edit the same state; whichever the user touches, the other has to stay in sync. Without round-trip, the feature is a one-way "type queries here" affordance and the column popups are second-class.

## Scope

- A new `RoundTrippable` interface in `filter/`, implemented by every built-in filter.
- A new `MultiSetFilter` for AND-of-includes semantics on slice-valued columns.
- An in-memory binder in `internal/querybar/` that maps `searchquery.AST` to filter mutations and back.
- A query-bar widget integrated into `grid.Model[T]` as an opt-in sub-model, gated by `WithQueryBar(...)`.
- Auto-derivation of `searchquery.Vocabulary` from columns, with full override.
- Replacement of the existing `WithQuickFilter` / `KeyMap.QuickFilter` surface — bare terms become the bar's residual term portion.
- Documentation: per-package READMEs, godoc on every new exported identifier, an `examples/querybar/` walkthrough.

Out of scope:

- Generic `Range[T]` (README follow-up). Orthogonal refactor; revisit after the round-trip surface settles.
- Live re-parse on every keystroke. Bar submits on Enter; bare-prefix-while-typing is a follow-up.
- Tab-completion / vocabulary-driven autocomplete. Reserved key, no implementation in v1.
- Saved views, history, or any persistence of query strings.
- Slice-valued column extraction beyond what `MultiSetFilter`'s default `matcher` provides (`[]string` `contains`).

## Proposal

Three layers, composed top-to-bottom by `grid/`.

### 1. `filter.RoundTrippable`

```go
// filter/roundtrip.go

// RoundTrippable is implemented by filters that can serialize their
// state to and from a query-bar clause. Filters that do not implement
// it are treated as opaque by the query bar — annotated, not edited.
type RoundTrippable interface {
    SetClause(values []string, negate bool) error
    Clause() (values []string, negate bool, ok bool)
}
```

Per-filter behavior:

| Filter            | `Clause()` shape                                   | `SetClause` behavior                                      | Lossy state                                        |
|-------------------|----------------------------------------------------|-----------------------------------------------------------|----------------------------------------------------|
| `TextFilter`      | one value, the substring                           | one value, sets text                                      | regex mode → `ok=false`                            |
| `NumberFilter`    | one value, the existing `>5`/`5..10`/`=5` text     | one value, parsed via existing `parseText`                | none                                               |
| `SetFilter`       | comma list (the included subset; flips to `negate=true` form when more than half included) | comma list; with `negate=true`, excludes the listed set | none in v1 (every state has a textual form)        |
| `BoolFilter`      | `true` / `false`                                   | `true`/`false`/`1`/`0`                                    | none                                               |
| `TimeFilter`      | one value, the existing range text                 | one value, parsed via existing `parseTime`                | none                                               |
| `MultiSetFilter`  | one value per constraint (repeated clauses)        | called once per repeated clause; appends one constraint   | none                                               |

`SetFilter`'s negation flip keeps the bar text small. `Clause()` returns the included subset normally; once more than half of `allValues` are included, it returns the excluded subset with `negate=true`. Symmetric on `SetClause`.

Note that `SetFilter` operates on a known `allValues` set. When new values appear in row data after a clause has been applied (e.g. a newly-loaded row introduces a label that was not in `allValues` at parse time), the new value is treated as "not included" — consistent with current `Include`/`SetValues` semantics. Documented in `SetClause`'s godoc.

### 2. `MultiSetFilter`

A new filter type. AND-of-includes against a typically slice-valued row value:

```go
// filter/multiset.go

type MultiSetFilter struct {
    constraints []string
    matcher     func(rowValue any, constraint string) bool
}

func NewMultiSetFilter(opts ...MultiSetOption) *MultiSetFilter
func (f *MultiSetFilter) AddConstraint(c string)
func (f *MultiSetFilter) RemoveConstraint(c string)
func (f *MultiSetFilter) Constraints() []string
func (f *MultiSetFilter) Clear()

// Filter
func (f *MultiSetFilter) Matches(value any) bool
func (f *MultiSetFilter) View() string
func (f *MultiSetFilter) Update(msg tea.Msg) (Filter, tea.Cmd)
func (f *MultiSetFilter) Active() bool

// RoundTrippable
func (f *MultiSetFilter) Clause() (values []string, negate bool, ok bool)
func (f *MultiSetFilter) SetClause(values []string, negate bool) error
```

Default `matcher` handles `[]string` row values via element equality. Callers supply their own matcher for richer slice element types.

The popup is intentionally minimal:

```
┌─ labels (must match all) ──────────┐
│  × bug                             │
│  × urgent                          │
│  × good first issue                │
│                                    │
│  d delete · esc close · / edit     │
└────────────────────────────────────┘
```

Arrow keys move row focus; `d` (or backspace) removes the focused constraint. No add-affordance in the popup. The footer reminds the user that new constraints come from the query bar. When the last constraint is removed the filter goes inactive; the popup closes on next render.

That is, `MultiSetFilter` is "view and remove in the popup, add in the bar". The asymmetry is acknowledged in the footer text rather than hidden.

For v1, comma-list values inside a `MultiSetFilter` clause (`label:a,b label:c`) reject as a parse-time warning — each constraint is exactly one value. Users who want OR-within-AND behavior register the column with a `SetFilter` instead and accept losing AND across constraints. Documented as a known limitation; revisit when usage demands it.

### 3. `internal/querybar`

Hidden package. Re-exports of consumer-visible types live in `grid/`.

```
internal/querybar/
  state.go     State[T]: textinput, parse cache, lossy-annotation set.
  bind.go      AST → filter mutations (Apply); filter state → AST → text (Rerender).
  vocab.go     AutoVocabulary[T]: derive Vocabulary from columns; option for override.
```

`State[T]` holds the textinput, the most recent parse (for error display), and the set of column IDs currently in lossy state. `Apply` and `Rerender` are pure functions over `(state, columns)`; the grid invokes them from `Update` paths.

#### Bar → filters (Enter pressed)

1. Parse `state.text` against the vocabulary.
2. Group `ast.Clauses` by canonical `Field`.
3. For each field, look up the column and type-assert the filter to `RoundTrippable`.
   - `*SetFilter`: merge values across clauses, single `SetClause` call.
   - `*MultiSetFilter`: clear prior constraints, then one `SetClause` per clause.
   - Scalar filters: expect a single clause, single value. More than one → use the last, surface a warning.
4. Columns with a previous active filter not mentioned in the AST → `Clear()`.
5. Set `m.quickFilterText = ast.Terms`.
6. Recompute display rows.

Per-clause errors do not fail the whole submit. `created:notadate` leaves the prior `created` filter intact and surfaces `1 error: created:notadate — invalid date` in the bar's status line. Already-good clauses apply.

#### Filters → bar (any mutation path)

1. Walk columns. For each filter that is `RoundTrippable` and `Active()`:
   - `clause, ok := f.Clause()`
   - `ok == true`  → emit clause text.
   - `ok == false` → add column ID to lossy-annotation set.
2. Append any non-`RoundTrippable` `Active()` filter to lossy-annotation set.
3. Append `m.quickFilterText` (bare terms) at the end.
4. `state.text = joined`; `state.lossy = set`.

The filter-changes-bar sync runs synchronously inside `grid.Update` at the existing dirty-marking sites (`SetRows`, `InsertRow`, `RemoveRow`, `UpdateRow`, `Clear*Filter*`, column-filter `Update` returning a new filter, programmatic `SetFilter`). No observers, no messages, no polling — the grid owns both sides of the state.

### 4. Grid integration

`grid.Model[T]` grows one new field, gated by an enable flag set from `WithQueryBar`:

```go
type Model[T any] struct {
    // ... existing fields ...
    queryBar *querybar.State[T]   // nil unless WithQueryBar applied
}
```

When `WithQueryBar` is applied, the bar is always rendered above the headers — both as an editing surface and as the canonical view of current filter state. That is, even with no active filter the bar is visible (showing an empty input); pressing `/` puts focus into it for editing. This differs from the current quick filter, which only renders when active.

The bar replaces the quick-filter user-facing entry point:

| Today                                     | After                                                                                          |
|-------------------------------------------|------------------------------------------------------------------------------------------------|
| `quickFilterEnabled bool`                 | Removed. `WithQueryBar(...)` enables the bar.                                                  |
| `WithQuickFilter()` option                | Removed. Use `WithQueryBar(opts...)`.                                                          |
| `WithQuickFilterText(s)` option           | Replaced with `WithQueryBarText(s)`.                                                           |
| `KeyMap.QuickFilter` (`/`)                | Renamed `KeyMap.QueryBar`, same default key, help text "query".                                |
| `quickFilterActive`                       | Renamed `queryBarActive`.                                                                      |
| `handleQuickFilterKeyMsg`                 | Replaced by `handleQueryBarKeyMsg`.                                                            |
| `Column.QuickFilterMatch`                 | Kept as-is. Bare-term matching is unchanged.                                                   |
| `passesQuickFilter`, `quickFilterText/Words` | Kept. Bare terms still drive substring matching.                                            |
| `quickFilterDebounceDelay`                | Kept (inert in v1's Enter-only model; reserved for the live-prefix follow-up).                 |

#### Bar key handling

| Key                       | Behavior                                                                                                |
|---------------------------|---------------------------------------------------------------------------------------------------------|
| `/`                       | Open bar (focus textinput, copy current rendered text in for editing).                                  |
| Printable rune / backspace| Edit textinput. No re-parse on each keystroke.                                                          |
| Enter                     | Submit (the full bar→filters flow).                                                                     |
| Esc                       | Cancel: discard textinput edits, re-render from canonical filter state, exit edit mode.                 |
| Tab                       | Reserved for autocomplete. Out of scope for v1; falls through.                                          |

The repo is pre-1.0; no migration shim for the removed `WithQuickFilter*` options. CHANGELOG entry covers the rename with a one-line migration.

### 5. Lossy mode handling

When a filter is `Active()` but `Clause()` returns `ok=false` (or the filter does not implement `RoundTrippable`), the bar can not represent it textually. Render:

```
state:open  bug    [+2 hidden filters: regex on title, custom on labels]
└─ round-trippable + bare terms ──┘  └─ greyed annotation ──────────────┘
```

The annotation is rendered with a separator style (`Styles.QueryBarLossy`) and lists each lossy column by ID.

Submit replaces only round-trippable clauses (and the bare-term portion). Lossy filters are left alone. That is, the bar is "edit what you can see" — the annotation is honest about what is not being touched.

There is no per-bar "clear all" key. The existing `KeyMap.ClearFilters` (Esc outside edit mode) already wipes every column filter and the quick-filter text. To surface the option exactly when the user needs it, the lossy annotation reads `[+1 hidden filter: regex on title — esc to clear all]` when at least one lossy filter is present.

#### Clauses targeting a lossy column

If `title` is in regex mode and the user types `title:foo` and submits, `SetClause` resets regex mode and applies the new substring. The rule is uniform: submitting a clause for a column overrides whatever state that filter was in, lossy or not. No special locked mode. Documented in the bar's godoc.

### 6. Vocabulary derivation

Auto-derived from columns by default; full override available. From `internal/querybar/vocab.go`:

- Every visible column with a non-empty `ColumnID` and a `RoundTrippable` filter becomes a queryable field. Field name = `ColumnID`.
- Field type inferred from filter type: `TextFilter` → `FieldString`; `NumberFilter` → `FieldString` (parses ranges itself); `SetFilter` → `FieldString` with `AcceptsList=true`; `MultiSetFilter` → `FieldString` with `AcceptsList=false` (one value per clause); `TimeFilter` → `FieldTime`; `BoolFilter` → `FieldBool`.
- Aliases come from a new optional `Column.QueryAliases []string`.
- Columns with no filter, or a filter that does not implement `RoundTrippable`, are skipped — they remain queryable in the AST sense (parser keeps them) but the binder ignores them.

Override is via `WithQueryBarVocabulary(v *searchquery.Vocabulary)`. Required for parse-time rewrites (`is:open` → `state:open`) — those can not be inferred from columns — and for cases where the queryable surface and the column set do not align (computed fields, virtual fields). Pure aliases on a column-derived field do not require the override; `Column.QueryAliases` covers them.

Auto-derivation runs once at `WithQueryBar` setup and again whenever `SetColumns` is called. The vocabulary is rebuilt from the current column set; outstanding bar text is re-parsed on the next render against the new vocabulary. Field/value clauses for fields that disappear from the vocabulary are kept verbatim in the AST (the parser is permissive) but the binder ignores them — same behavior as unknown fields.

## Tests

Per-package unit tests:

- `filter/`:
  - `TestRoundTrippable_*` per filter type — `Clause()` after state transitions, `SetClause` round-trip identity, `ok=false` on lossy modes (regex), error returns on invalid input.
  - `TestSetFilter_NegationFlip` — when more than half included, `Clause()` returns the negated form.
  - `TestMultiSetFilter_*` — Add/Remove/Clear/Matches; popup focus and deletion; `Clause()` returns one value per constraint.

- `internal/querybar/`:
  - `TestBindApply_*` — full AST → filter mutations across all built-in filter types and combinations (OR, AND, scalar, lossy).
  - `TestBindRerender_*` — filter state → bar text including the lossy annotation.
  - `TestRoundTripIdentity` — table-driven: for every `(vocab, query)` pair, parse → apply → rerender produces the same canonical text.
  - `TestAutoVocabulary_*` — column → field derivation, type inference, override behavior.

- `grid/`:
  - `TestQueryBar_KeyHandling` — `/` opens, Enter submits, Esc cancels.
  - `TestQueryBar_FilterChangeRerender` — parameterized over every mutation site, verifies bar stays in sync.
  - `TestQueryBar_ClearFiltersClearsLossy` — Esc-out-of-bar invokes `ClearFilters` and wipes lossy filters too.
  - `TestQueryBar_LossyAnnotationContents` — render output includes lossy column IDs.

Integration:

- `examples/querybar/` exercises text, number, set, bool, time, and multiset filters against a single dataset. Doubles as the documentation walkthrough. Not part of automation; referenced from the README.

## Documentation

- `searchquery/README.md` — update the "Open follow-ups" section, mark the binder, query-bar, and round-trip pieces as landed; add a usage section pointing at `WithQueryBar`.
- `filter/README.md` — new. Cover the `Filter` interface, `RoundTrippable`, every built-in filter, and `MultiSetFilter`'s asymmetric editing model.
- `grid/README.md` — extend with a query-bar section: enabling the bar, vocabulary auto-derivation and override, lossy annotation, key bindings.
- Godoc on every new exported identifier. Stdlib-style `TypeName verb-phrase.` openers throughout.
- `examples/querybar/` — annotated example, callable via `go run ./examples/querybar/`.

## Out of scope

- `Range[T]` generic refactor of `NumberFilter` + `TimeFilter` parsing internals.
- Live re-parse on every keystroke (the "live bare prefix" path).
- Tab-completion / vocabulary-driven autocomplete.
- Saved views; query-history persistence.
- Comma-list-inside-AND-clause semantics for `MultiSetFilter`.
- Slice extraction infrastructure beyond `MultiSetFilter`'s default matcher.

## Open questions

None blocking. Two items deferred to follow-up specs:

1. _Range[T]_ — the duplication between `NumberFilter` and `TimeFilter` parsing has not yet caused real pain. Refactor when a third caller appears or when the existing two start drifting.
2. _Live bare-prefix while typing_ — would let bare terms apply incrementally during a query edit, matching today's quick-filter UX. Adds complexity (split parse, debounce coordination); revisit if Enter-only feels heavy in practice.
