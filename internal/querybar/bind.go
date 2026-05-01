package querybar

import (
	"strings"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
)

// Rerender renders the current filter state as bar text plus a list of
// column IDs in lossy state. Inputs:
//
//   - cols: the grid's current column set. Each column's Filter is
//     inspected for RoundTrippable + Active.
//   - bareTerms: the residual quick-filter text (appended verbatim).
//
// Returns the joined text and the lossy column IDs (in column order).
func Rerender[T any](cols []data.Column[T], bareTerms string) (text string, lossy []string) {
	var clauses []string
	for i := range cols {
		c := &cols[i]
		if c.Filter == nil || !c.Filter.Active() {
			continue
		}
		rt, ok := c.Filter.(filter.RoundTrippable)
		if !ok {
			lossy = append(lossy, c.ColumnID)
			continue
		}
		values, negate, ok := rt.Clause()
		if !ok {
			lossy = append(lossy, c.ColumnID)
			continue
		}
		clauses = append(clauses, formatClause(c.ColumnID, values, negate, isMultiClause(c.Filter)))
	}
	parts := clauses
	if bareTerms != "" {
		parts = append(parts, bareTerms)
	}
	return strings.Join(parts, " "), lossy
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
