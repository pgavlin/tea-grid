# Clear-filters keybinding

**Status:** Draft
**Date:** 2026-04-21

## Problem

`Model[T]` already exposes `ClearFilters()` (clears the quick filter and every column filter), but there is no key binding to invoke it. Consumers either wire one up themselves or call the method imperatively.

`esc` is the obvious key for this — it is the conventional "back out / clear transient state" key — but in normal mode `esc` is already bound to `DeselectAll`. The existing per-mode handlers (edit, filter-edit, quick-filter) consume `esc` before it ever reaches normal-mode dispatch, so this design only affects normal-mode behavior.

## Proposal

Add a `ClearFilters` binding to `KeyMap` defaulting to `esc`, and in normal mode resolve the conflict with `DeselectAll` by priority: the first `esc` clears the selection (if any), the next clears the filters (if any). Each `esc` undoes one layer of transient state, in the order most users will want it undone.

### KeyMap addition

```go
ClearFilters key.Binding
// default: esc / "clear filters"
```

### Normal-mode handler

In `grid/update.go:handleKeyMsg`, replace the standalone `DeselectAll` case with a combined case that matches either binding:

```go
case key.Matches(msg, m.KeyMap.DeselectAll), key.Matches(msg, m.KeyMap.ClearFilters):
    if m.sel.Active() {
        m.ClearSelection()
        return m, nil
    }
    if m.hasActiveFilters() {
        m.ClearFilters()
        return m, func() tea.Msg { return FiltersClearedMsg{} }
    }
    return m, nil
```

This keeps the two bindings independently rebindable: a consumer who wants `esc` to *only* clear filters can `key.NewBinding(key.WithKeys())` on `DeselectAll`, and the priority logic falls through to the filter clear.

### `hasActiveFilters` helper

Private method on `Model[T]`:

```go
func (m *Model[T]) hasActiveFilters() bool {
    if m.quickFilterText != "" {
        return true
    }
    for i := range m.cols {
        if m.cols[i].Filter != nil && m.cols[i].Filter.Active() {
            return true
        }
    }
    return false
}
```

Used only by the keybinding handler. Avoids spending the work of `ClearFilters()` (which dirties the model and recomputes display rows) when there is nothing to clear.

### `FiltersClearedMsg`

New message type in `grid/messages.go`:

```go
// FiltersClearedMsg is emitted when the user clears all filters via the
// ClearFilters key binding.
type FiltersClearedMsg struct{}
```

A single coarse-grained notification rather than N per-column `FilterChangedMsg` + a `QuickFilterChangedMsg`. The user's intent was a bulk clear; signaling it as one event matches that.

The public `ClearFilters()` method continues to emit nothing — same convention as other public setters (`SetRows`, `SetColumns`, etc.). The message belongs to the keybinding handler, which is the user-driven path.

### `FullHelp`

Add `m.KeyMap.ClearFilters` to the row in `Model.FullHelp` that already contains `QuickFilter`.

## Tests

- Selection active + filters active → `esc` clears only selection; filters intact; no `FiltersClearedMsg`.
- No selection + filters active (column filter) → `esc` clears filters; emits `FiltersClearedMsg`.
- No selection + quick filter text set (but quick-filter input not active) → `esc` clears the quick filter text; emits `FiltersClearedMsg`. *(Edge case worth covering: the quick filter mode handler only runs while `quickFilterActive` is true; once confirmed, the text persists and normal-mode `esc` should clear it.)*
- No selection + no filters → `esc` is a no-op; no message.
- `FullHelp` includes the `ClearFilters` binding.
- Existing edit / filter-edit / quick-filter `esc` tests remain green (those modes route before normal mode).

## Out of scope

- Changing the public `ClearFilters()` API or its message behavior.
- Per-mode `esc` semantics (edit cancel, filter-edit clear, quick-filter clear) — already implemented and unchanged.
- A "clear only column filters" or "clear only quick filter" binding — not requested; the bulk clear is sufficient.
