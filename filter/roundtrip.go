package filter

// RoundTrippable is implemented by filters that can serialize their
// state to and from a query-bar clause. Filters that do not implement
// it are treated as opaque by the query bar — annotated, not edited.
//
// SetClause replaces the filter's state with what the clause expresses.
// Returning an error leaves prior state unchanged and surfaces the
// error in the bar's status line; one error per offending clause.
//
// Clause returns the filter's state as a clause. ok=false when the
// filter is inactive (no clause to surface) or in a state that can not
// be expressed in the query syntax — the bar treats both ok=false
// states the same way: skip in the bar text, mention in the lossy
// annotation if Active() is true.
type RoundTrippable interface {
	SetClause(values []string, negate bool) error
	Clause() (values []string, negate bool, ok bool)
}

// Compile-time interface assertions for built-in filters.
var (
	_ RoundTrippable = (*TextFilter)(nil)
	_ RoundTrippable = (*NumberFilter)(nil)
	_ RoundTrippable = (*SetFilter)(nil)
)
