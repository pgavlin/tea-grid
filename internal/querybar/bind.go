package querybar

import (
	"fmt"
	"strings"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/searchquery"
)

// RerenderWithPresence is Rerender with an additional PresenceProvider
// for emitting has:/no: clauses. See Rerender for details.
func RerenderWithPresence[T any](cols []data.Column[T], bareTerms string, presence PresenceProvider) (text string, lossy []string) {
	text, lossy = Rerender(cols, bareTerms)
	if presence == nil {
		return text, lossy
	}
	colByID := make(map[string]*data.Column[T], len(cols))
	for i := range cols {
		colByID[cols[i].ColumnID] = &cols[i]
	}
	var hasVals, noVals []string
	presence.EachPresence(func(colID string, present bool) {
		// Use the column's display name when we know about it.
		name := colID
		if c, ok := colByID[colID]; ok {
			name = displayFieldName(c)
		}
		if present {
			hasVals = append(hasVals, name)
		} else {
			noVals = append(noVals, name)
		}
	})
	parts := []string{}
	if text != "" {
		parts = append(parts, text)
	}
	if len(hasVals) > 0 {
		parts = append(parts, formatClause(MetaFieldHas, hasVals, false, false))
	}
	if len(noVals) > 0 {
		parts = append(parts, formatClause(MetaFieldNo, noVals, false, false))
	}
	return strings.Join(parts, " "), lossy
}

// Rerender renders the current filter state as bar text plus a list of
// column IDs in lossy state. Inputs:
//
//   - cols: the grid's current column set. Each column's Filter is
//     inspected for RoundTrippable + Active.
//   - bareTerms: the residual quick-filter text (appended verbatim).
//
// Returns the joined text and the lossy column IDs (in column order).
// Clauses are emitted using the column's display name (slugified
// HeaderName when meaningful, else lowercased ColumnID, else the raw
// ColumnID) so the bar round-trips back to the alias users actually
// type rather than to a synthetic ID like "col2".
//
// Use RerenderWithPresence to also emit has:/no: meta-clauses; the
// plain Rerender ignores those constraints.
// PresenceProvider is the small accessor the bar uses to read the
// grid's has:/no: state during Rerender. Implemented by grid.Model[T]
// (passed through Rerender to keep this package generic-but-light).
type PresenceProvider interface {
	// EachPresence calls fn(colID, present) for every active has:/no:
	// constraint. present=true → has:; present=false → no:.
	EachPresence(fn func(colID string, present bool))
}

// PresenceMutator is the inverse of PresenceProvider: it lets the
// binder write back has:/no: state during Apply. Implemented by
// grid.Model[T].
type PresenceMutator interface {
	SetColumnPresence(colID string, present bool)
	ClearColumnPresence(colID string)
}

func Rerender[T any](cols []data.Column[T], bareTerms string) (text string, lossy []string) {
	var clauses []string
	for i := range cols {
		c := &cols[i]
		if c.Filter == nil || !c.Filter.Active() {
			continue
		}
		display := displayFieldName(c)
		// Columns that BuildAutoVocab would skip (hidden or unfilterable)
		// are not addressable from the bar — surface them as lossy so the
		// user knows filtering is happening, but do not emit a clause they
		// could not edit.
		if c.Hide || !c.Filterable {
			lossy = append(lossy, display)
			continue
		}
		rt, ok := c.Filter.(filter.RoundTrippable)
		if !ok {
			lossy = append(lossy, display)
			continue
		}
		values, negate, ok := rt.Clause()
		if !ok {
			lossy = append(lossy, display)
			continue
		}
		clauses = append(clauses, formatClause(display, values, negate, isMultiClause(c.Filter)))
	}
	parts := clauses
	if bareTerms != "" {
		parts = append(parts, bareTerms)
	}
	return strings.Join(parts, " "), lossy
}

// displayFieldName picks the most user-friendly name for a column when
// emitting a clause. Prefers a non-empty slugified HeaderName over the
// raw ColumnID; falls back to a lowercased ColumnID when the ID itself
// is mixed-case; finally falls back to the raw ColumnID.
func displayFieldName[T any](c *data.Column[T]) string {
	if slug := slugifyHeader(c.HeaderName); slug != "" && slug != c.ColumnID {
		return slug
	}
	if lower := strings.ToLower(c.ColumnID); lower != c.ColumnID {
		return lower
	}
	return c.ColumnID
}

// formatClause builds the textual form of a clause. For multi-clause
// filters (MultiSetFilter), one clause per value is emitted. For OR
// list filters (SetFilter), values are comma-joined. For scalar
// filters, exactly one value is expected.
func formatClause(field string, values []string, negate, multiClause bool) string {
	prefix := ""
	if negate {
		prefix = "-"
	}
	if multiClause {
		out := make([]string, len(values))
		for i, v := range values {
			out[i] = prefix + field + ":" + quoteIfNeeded(v)
		}
		return strings.Join(out, " ")
	}
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = quoteIfNeeded(v)
	}
	return prefix + field + ":" + strings.Join(quoted, ",")
}

// isMultiClause reports whether the filter type emits one clause per
// value (AND semantics). Only MultiSetFilter does.
func isMultiClause(f filter.Filter) bool {
	_, ok := f.(*filter.MultiSetFilter)
	return ok
}

// quoteIfNeeded wraps a value in double quotes when it contains any
// character that would break the bare-word grammar (whitespace or one
// of the structural characters `:`, `,`, `"`).
func quoteIfNeeded(v string) string {
	if v == "" {
		return `""`
	}
	for _, r := range v {
		if r == ' ' || r == '\t' || r == '\n' || r == ':' || r == ',' || r == '"' {
			return `"` + strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `"`, `\"`) + `"`
		}
	}
	return v
}

// ApplyResult reports the outcome of an Apply call.
type ApplyResult struct {
	// BareTerms is the residual bare-term string from the AST,
	// suitable for the grid's existing quick-filter mechanism.
	BareTerms string

	// Errors lists per-clause errors. A non-empty Errors does not
	// imply Apply rolled back; clauses that succeeded have already
	// been applied.
	Errors []string
}

// ApplyWithPresence is Apply with an additional PresenceMutator that
// handles has:/no: meta-clauses. Resolves each value through the
// column's display-name aliases before writing to the presence map.
// All has:/no: constraints not present in this AST are cleared.
func ApplyWithPresence[T any](cols []data.Column[T], ast searchquery.AST, presence PresenceMutator) ApplyResult {
	if presence == nil {
		return Apply(cols, ast)
	}
	// Split the AST into meta-clauses (has:, no:) and the rest.
	var meta, rest []searchquery.Clause
	for _, c := range ast.Clauses {
		switch c.Field {
		case MetaFieldHas, MetaFieldNo:
			meta = append(meta, c)
		default:
			rest = append(rest, c)
		}
	}
	res := Apply(cols, searchquery.AST{Clauses: rest, Terms: ast.Terms})

	// Re-apply has:/no:. First collect every column referenced by an
	// active filter or by a meta-clause so we know which presence
	// entries to clear (anything not mentioned).
	wanted := make(map[string]bool, 8)
	for _, c := range meta {
		// has:foo with negate inverts to no:; no:foo with negate inverts to has:.
		present := (c.Field == MetaFieldHas) != c.Negate
		for _, v := range c.Values {
			col := lookupCol(cols, v)
			if col == nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: unknown column %q", c.Field, v))
				continue
			}
			wanted[col.ColumnID] = present
		}
	}
	// Clear any presence not in `wanted`.
	cleared := make(map[string]struct{}, 4)
	if pp, ok := presence.(PresenceProvider); ok {
		pp.EachPresence(func(colID string, _ bool) {
			if _, keep := wanted[colID]; !keep {
				cleared[colID] = struct{}{}
			}
		})
	}
	for colID := range cleared {
		presence.ClearColumnPresence(colID)
	}
	for colID, p := range wanted {
		presence.SetColumnPresence(colID, p)
	}
	return res
}

// Apply pushes an AST into the column filters. For each clause:
//
//   - SetFilter: merge values across same-field clauses; one SetClause
//     call with the merged values.
//   - MultiSetFilter: clear prior constraints, then one SetClause per
//     clause (each appends one constraint).
//   - Scalar filters (Text/Number/Bool/Time): use the last clause for
//     the field; record a warning if there were multiple.
//
// Columns whose filter is RoundTrippable and Active but NOT mentioned
// in the AST are Cleared. Lossy filters (RoundTrippable.Clause returns
// ok=false) are left alone — the bar can not represent them and we
// take that as "the user is not editing them right now."
//
// Per-clause errors do not abort: good clauses apply, bad ones surface
// in ApplyResult.Errors.
func Apply[T any](cols []data.Column[T], ast searchquery.AST) ApplyResult {
	res := ApplyResult{BareTerms: ast.Terms}

	// Group clauses by canonical field.
	grouped := make(map[string][]searchquery.Clause)
	for _, cl := range ast.Clauses {
		grouped[cl.Field] = append(grouped[cl.Field], cl)
	}

	// Index columns by ID for lookup, and remember which we touched.
	colIdx := make(map[string]int, len(cols))
	for i := range cols {
		colIdx[cols[i].ColumnID] = i
	}
	mentioned := make(map[string]bool, len(grouped))

	// Apply clauses.
	for field, clauses := range grouped {
		idx, ok := colIdx[field]
		if !ok {
			// Unknown field: parser keeps it; binder ignores.
			continue
		}
		mentioned[field] = true
		f := cols[idx].Filter
		rt, ok := f.(filter.RoundTrippable)
		if !ok {
			continue
		}

		switch typed := f.(type) {
		case *filter.MultiSetFilter:
			typed.Clear()
			for _, c := range clauses {
				if err := rt.SetClause(c.Values, c.Negate); err != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", field, err))
				}
			}
		case *filter.SetFilter:
			merged := mergeValues(clauses)
			negate := false
			for _, c := range clauses {
				if c.Negate {
					negate = true
					break
				}
			}
			if err := rt.SetClause(merged, negate); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", field, err))
			}
		default:
			last := clauses[len(clauses)-1]
			if len(clauses) > 1 {
				res.Errors = append(res.Errors,
					fmt.Sprintf("%s: multiple clauses on scalar field; using last", field))
			}
			if err := rt.SetClause(last.Values, last.Negate); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", field, err))
			}
		}
	}

	// Clear active RoundTrippable filters not mentioned in the AST.
	// Lossy filters (Clause ok=false) are left alone.
	for i := range cols {
		c := &cols[i]
		if c.Filter == nil || !c.Filter.Active() {
			continue
		}
		if mentioned[c.ColumnID] {
			continue
		}
		rt, ok := c.Filter.(filter.RoundTrippable)
		if !ok {
			continue
		}
		_, _, clauseOk := rt.Clause()
		if !clauseOk {
			continue
		}
		c.Filter.Clear()
	}

	return res
}

// mergeValues collects all values across a set of clauses on the same
// field. Duplicates are preserved; SetFilter's SetClause de-duplicates
// implicitly via the include set.
func mergeValues(clauses []searchquery.Clause) []string {
	var out []string
	for _, c := range clauses {
		out = append(out, c.Values...)
	}
	return out
}
