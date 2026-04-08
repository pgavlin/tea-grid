# tea-grid Datasource Row Model

## Motivation

tea-grid's current row model requires all row data in memory via `WithRows([]T)`.
The grid owns sorting, filtering, and grouping over this slice. This works well
for datasets up to ~100K rows, but breaks down for larger data:

- A log viewer over a 10 GB file can't parse and hold millions of `LogRecord`
  values on the Go heap.
- A database-backed grid can't (and shouldn't) fetch every row upfront.
- An application tailing a stream has no finite row set at all.

AG Grid solves this with multiple row models. The most relevant is the
**Server-Side Row Model**, where the grid delegates sort/filter/grouping to a
datasource and only requests the rows it needs to render. We propose an analogous
model for tea-grid.

## Design Principles

1. **The grid drives the interaction.** The grid owns the UI state for sorting,
   filtering, grouping, and scrolling. When state changes, the grid tells the
   datasource what it needs. The datasource never pushes unsolicited data.

2. **Elm architecture.** Requests are issued as `tea.Cmd`s. Responses arrive as
   `tea.Msg`s. No shared mutable state, no channels, no callbacks.

3. **The datasource owns the computation.** The grid does not sort, filter, or
   group in datasource mode. It passes the full state to the datasource and
   renders whatever comes back. The datasource can implement these operations
   however it wants — linear scan, index lookup, SQL query.

4. **Client-side model is unchanged.** `WithRows([]T)` continues to work exactly
   as it does today. The datasource model is opt-in via `WithDatasource`.

## API

### Datasource Interface

```go
// Datasource provides row data on demand for datasets too large to hold in
// memory. The grid calls Request when it needs rows, passing the full
// sort/filter/group state. The datasource returns a tea.Cmd that produces
// a DatasourceResponseMsg[T] when executed.
type Datasource[T any] interface {
    Request(req DatasourceRequest[T]) tea.Cmd
}
```

### Request

```go
type DatasourceRequest[T any] struct {
    // Sequence is a monotonically increasing counter that tracks the current
    // sort/filter/group state. The datasource must echo it back in the
    // response. The grid uses it to discard stale responses (e.g. a response
    // for a filter that has since changed).
    //
    // Sequence is only bumped when sort, filter, or group state changes — not
    // on scroll. Multiple in-flight requests for different row windows may
    // share the same sequence.
    Sequence uint64

    // Row window. The grid requests rows [StartRow, EndRow).
    StartRow int
    EndRow   int

    // Sort criteria, in priority order. Empty means natural/insertion order.
    Sort []sort.SortCriterion

    // Column filters. Each entry carries the column ID and the filter instance.
    // The datasource type-asserts to concrete filter types (TextFilter,
    // SetFilter, etc.) to extract state for query translation.
    Filters []ColumnFilter

    // Quick filter text. Empty means no quick filter.
    QuickFilter string

    // Grouping columns, in nesting order. Empty means no grouping.
    GroupCols []string

    // Group keys. When the user expands a group, the grid sends the key path
    // from root to the expanded node. Empty means top-level rows/groups.
    // Example: GroupCols=["country","year"], GroupKeys=["Argentina"] means
    // "give me the year-level groups (or leaf rows) under Argentina."
    GroupKeys []string
}

type ColumnFilter struct {
    ColumnID string
    Filter   filter.Filter
}
```

> **Design note — filter representation.** The request passes `filter.Filter`
> instances directly. The datasource type-asserts to concrete types
> (`*TextFilter`, `*SetFilter`, etc.) to extract filter state for query
> translation. This is an intentional coupling: the datasource inherently
> must understand filter semantics to translate them into queries, and for
> custom filter types the datasource author typically owns both the filter
> implementation and the query translation.
>
> If a decoupled state-extraction mechanism is needed in the future (e.g.
> a generic datasource that works with any filter type), additional methods
> can be added to the `filter.Filter` interface without breaking existing
> implementations.

### Response

```go
// DatasourceResponseMsg is the tea.Msg produced by the Cmd returned from
// Datasource.Request.
type DatasourceResponseMsg[T any] struct {
    // Sequence echoed from the request.
    Sequence uint64

    // Rows for the requested window. May be shorter than EndRow-StartRow if
    // the window extends past the end of the data.
    Rows []T

    // RowIDs are the stable identifiers for each row, parallel to Rows.
    // Used for selection stability, row tracking, and deduplication. If nil,
    // the grid falls back to WithRowID(func(T) string) if configured, or
    // uses display indices (unstable across data changes).
    RowIDs []string

    // RowCount is the total number of rows matching the current
    // sort/filter/group state.
    //   >0 : exact count (grid renders a scrollbar of this height)
    //    0 : no matching rows
    //   -1 : unknown (grid allows infinite scroll; datasource signals the end
    //         by returning fewer rows than requested)
    RowCount int

    // Error, if non-nil, indicates that the request failed. The grid displays
    // an error state and does not update the cache. A nil error with zero Rows
    // means "no matching data" (not an error).
    Error error
}
```

### Grid Option

```go
func WithDatasource[T any](ds Datasource[T]) Option[T] {
    return func(m *Model[T]) {
        m.datasource = ds
    }
}
```

`WithDatasource` and `WithRows` are mutually exclusive. Passing both is a
programming error (panic at construction time is fine).

## Grid Behavior in Datasource Mode

### State

```go
// Added to grid.Model[T]:
datasource  Datasource[T]
dsStateSeq  uint64                // bumped on sort/filter/group change only
dsCache     map[blockKey][]T      // raw T values from responses
dsTotalRows int                   // from most recent response (-1 = unknown)
dsLoading   bool                  // true while a request is in flight
dsStale     []*data.RowNode[T]    // previous displayRows, shown during loading
dsError     error                 // most recent error, cleared on next success
```

> **Design note — cache stores `[]T`, not `[]*RowNode[T]`.** The cache's job
> is to avoid re-fetching. Grid metadata like `RowIndex` changes on every
> viewport shift and should not be cached. The grid wraps `T` values in
> `*RowNode[T]` when building `displayRows` from the cache.

### State Machine

**Sort/filter/group change:**
1. Bump `dsStateSeq`.
2. Clear `dsCache`.
3. Save current `displayRows` as `dsStale`.
4. Set `dsLoading = true`.
5. Issue `Request` for the current visible window.
6. Render `dsStale` with a loading indicator.

**Scroll (cache miss):**
1. Set `dsLoading = true`.
2. Issue `Request` for the new visible range (same `dsStateSeq`).
3. Render cached rows where available; gaps show placeholder rows.

> **Design note — scroll does not bump `dsStateSeq`.** Only sort/filter/group
> changes invalidate the cache. Scroll requests use the current sequence
> because the data at a given row index is still valid — only the viewport
> position has changed. This means an in-flight request for rows [200, 300)
> is not discarded when the user scrolls to [400, 500): both responses are
> cached at their respective block keys. Sequence comparison only gates
> whether the sort/filter/group state has changed, not the viewport position.

**Scroll (cache hit):**
1. No request issued. Build `displayRows` from cache. This is the fast path.

**Response received:**
1. If `msg.Sequence != m.dsStateSeq`, discard (stale — sort/filter/group state
   has changed since this request was issued).
2. If `msg.Error != nil`, set `dsError = msg.Error`, set `dsLoading = false`.
   Do not update cache. The grid renders the error state.
3. Otherwise: clear `dsError`, populate `dsCache` with the response block.
4. Set `dsLoading = false`.
5. Build `displayRows` from cache for the current visible range.
6. Update `dsTotalRows` from `msg.RowCount`.
7. Optionally issue a pre-fetch `Request` for the next block in the scroll
   direction.

### Rendering During Loading

While `dsLoading` is true, the grid renders `dsStale` (the previous display
rows) in place. This avoids flicker on filter changes. A configurable loading
style (e.g. dimmed rows, a status line indicator) signals that the data is
stale. If no stale data exists (first load), the grid renders empty rows or a
"Loading..." placeholder.

When `dsError` is non-nil, the grid renders an error indicator. The specific
rendering is controlled by a configurable style/callback. On the next
successful response, the error is cleared.

### Block Cache

The cache is keyed by `(startRow, endRow)` within a single state-sequence
epoch. Any `dsStateSeq` bump (sort/filter/group change) invalidates the entire
cache. Scroll does not invalidate the cache.

Configuration:

```go
// WithDatasourceBlockSize sets the number of rows per cache block.
// Default: 100.
func WithDatasourceBlockSize[T any](n int) Option[T]

// WithDatasourceMaxBlocks sets the maximum number of cached blocks.
// Default: 10. When exceeded, the block furthest from the viewport is evicted.
func WithDatasourceMaxBlocks[T any](n int) Option[T]
```

When the grid needs rows [200, 250) and blockSize=100, it requests block
[200, 300). The extra rows are cached for subsequent scrolling.

### Pre-fetching

After receiving a response, the grid issues a speculative `Request` for the
adjacent block in the user's scroll direction (determined by comparing the
new viewport position to the previous one). This keeps one block ahead
of the viewport so smooth scrolling rarely hits a cache miss.

Pre-fetch requests use the same `dsStateSeq` as the triggering response.
If a pre-fetch response arrives after a state change has bumped the sequence,
it is discarded like any other stale response.

### Features Disabled in Datasource Mode

These features require in-memory access to all rows and are disabled when a
datasource is active. The grid remains a single `Model[T]` type — these
methods panic at runtime with a clear message rather than using separate
types with different method sets. This keeps the API surface simple and avoids
the complexity of Go's type system fighting value semantics and virtual
dispatch for a 7-method difference.

**Methods that panic in datasource mode (7):**

- `SetRows`, `Rows` — datasource owns the data
- `UpdateRow`, `InsertRow`, `RemoveRow` — mutations go through the datasource
- `PinRow`, `UnpinRow` — modifies rows we don't own

**Options that panic at construction time if combined with `WithDatasource` (4):**

- `WithExternalFilter` — client-side predicate, can't be delegated
- `WithPostSort` — client-side hook over the full row set
- `WithPinnedTopRows`, `WithPinnedBottomRows` — predicates over all rows

**Methods that work but dispatch differently (6):**

- `SetSort`, `SetQuickFilter`, `SetColumnFilter`, `ClearFilters` — bump
  `dsStateSeq` and issue a datasource request instead of calling
  `recomputeDisplayRows()`
- `ExpandGroup`, `CollapseGroup` — update `GroupKeys` and issue a request
  (grouping support is deferred; these are no-ops in the flat-only v1)

**Other disabled features:**

- **Cell editing** — `CellEditingConfirmedMsg` fires but the grid cannot write
  the new value back (it does not own the data). Callers who need editing in
  datasource mode should handle `CellEditingConfirmedMsg` in their own
  `Update()` and push the change to the datasource, then trigger a refresh.

Features that continue to work unchanged (~55 methods):

- **Viewport and scrolling** — the grid still owns the viewport.
- **Selection** — by display index, works as before.
- **Focus and keyboard navigation** — unchanged.
- **Column sizing, pinning, reordering** — purely visual, no data dependency.
- **Cell rendering** — the grid calls `ValueGetter`/`ValueFormatter` on
  whatever `T` the datasource returns.
- **Static pinned rows** — `WithStaticPinnedTop`/`Bottom` still work; these
  rows are provided directly, not through the datasource.
- **Row height** — `WithRowHeight(func(T) int)` applies to datasource rows
  when building `RowNode` wrappers from cached `T` values.

### Selection Stability

Selection operates on display indices. If the user selects row 250, scrolls
away (block evicted), and scrolls back (re-fetched), the selection at index
250 is preserved — assuming sort/filter state has not changed. If the
underlying data has changed between fetches (e.g. new rows inserted), index
250 may map to a different logical row. This is inherent to index-based
selection and matches AG Grid's behavior.

Future work could track selection by row ID (derived via `WithRowID`) for
stability across data changes, but this is out of scope for the initial design.

## Example: hustle Log Viewer

hustle views log files that can be multiple gigabytes. With the datasource
model:

```go
type logDatasource struct {
    data    []byte   // mmap'd file
    offsets []int64  // byte offset of each line
    format  log.Format

    mu       sync.Mutex
    index    []int32  // filtered+sorted line indices
    indexSeq uint64   // sequence when index was last built
}

func (ds *logDatasource) Request(req grid.DatasourceRequest[log.LogRecord]) tea.Cmd {
    return func() tea.Msg {
        ds.mu.Lock()
        defer ds.mu.Unlock()

        // Rebuild filtered/sorted index if state changed
        if ds.indexSeq != req.Sequence {
            ds.rebuildIndex(req)
            ds.indexSeq = req.Sequence
        }

        // Parse only the requested window
        end := req.EndRow
        if end > len(ds.index) {
            end = len(ds.index)
        }
        rows := make([]log.LogRecord, 0, end-req.StartRow)
        for i := req.StartRow; i < end; i++ {
            line := ds.lineAt(ds.index[i])
            if rec, err := ds.format.ParseRecord(line); err == nil {
                rows = append(rows, rec)
            }
        }

        return grid.DatasourceResponseMsg[log.LogRecord]{
            Sequence: req.Sequence,
            Rows:     rows,
            RowCount: len(ds.index),
        }
    }
}
```

Memory profile for a 10 GB file with 50M lines:

| Data                         | Size     |
|------------------------------|----------|
| mmap'd file (virtual)        | 10 GB    |
| Resident pages (OS-managed)  | ~100 MB  |
| Line offset index            | 400 MB   |
| Filtered index (worst case)  | 200 MB   |
| Parsed records (visible)     | ~50 KB   |
| **Go heap total**            | **~600 MB** |

Compared to loading all records: ~10-25 GB of Go heap.

## AG Grid Precedent

AG Grid provides four row models: Client-Side, Infinite, Server-Side, and
Viewport. The Server-Side model is the closest analogue. Key similarities:

- Grid sends sort/filter/group state with every request.
- Datasource returns a row window and total count.
- Grid caches blocks and evicts on state change.
- Grouping is lazy: expanding a group sends the group key path, datasource
  returns children.

Key differences from our design:

- AG Grid's `getRows` is callback-based (JavaScript); ours uses `tea.Cmd`/
  `tea.Msg` to fit the Elm architecture.
- AG Grid has a separate Infinite model (flat only, no grouping); we don't
  need this — our single datasource interface handles both cases via the
  optional `GroupCols`/`GroupKeys` fields.
- AG Grid's Viewport model (server pushes data) has no analogue here. If
  needed in the future, it could be a second datasource interface.

## Interaction with Performance Work

The datasource model sidesteps most of what the client-side performance
optimizations address:

- **No `recomputeDisplayRows` pipeline** — the datasource handles
  filter/sort/group. The grid's filter cache, active-filter pre-computation,
  and quick-filter scratch buffer are unused.
- **Block cache replaces filter cache** as the primary optimization for
  avoiding redundant work.
- **Pointer slices still matter** — `displayRows` is built from cached blocks
  as `[]*data.RowNode[T]`, so the rendering path benefits from the same
  pointer-based display list.
- **`View()` cost is unchanged** (~1ms, ~10K allocs per frame) and remains the
  steady-state bottleneck. Rendering depends on the visible viewport, not the
  data source.

## Resolved Questions

1. **Row IDs in datasource mode.** The datasource provides IDs via `RowIDs
   []string` in the response, because `T` is often a projection that does not
   contain the natural key. `WithRowID(func(T) string)` serves as a fallback
   when `RowIDs` is nil. This gives datasources full control over ID
   derivation (e.g. database primary keys, composite keys, byte offsets)
   without requiring the key to be part of the projected row type.

2. **Partial grouping.** Grouping is always fully delegated to the datasource
   in the initial design. Client-side grouping over datasource data would
   require buffering the entire filtered result set, defeating the purpose.
   This matches AG Grid's server-side model. A hybrid mode could be added
   later without changing the datasource interface — the grid would simply
   not set `GroupCols` in the request and group the returned flat rows
   locally.

3. **Streaming/tailing.** Out of scope for the initial design. `RowCount: -1`
   (unknown) already supports infinite scroll. A tailing datasource can be
   built on the current request/response pattern — the grid periodically
   re-requests the tail region, and the datasource returns the latest rows.
   True push-based streaming (unsolicited updates) would require a
   subscription mechanism and is a separate future design.

4. **Filter representation.** The request passes `filter.Filter` instances
   directly. The datasource type-asserts to extract state. This is an
   intentional coupling: the datasource inherently must understand filter
   semantics to translate them into queries, and for custom filter types the
   datasource author typically owns both sides. Additional interface methods
   for generic state extraction can be added in the future without breaking
   existing implementations.
