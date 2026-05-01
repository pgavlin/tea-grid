package querybar

import (
	"strings"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
)

// Suggest returns completion candidates for the token at the cursor
// position. start..end is the byte range in text that should be
// replaced with a chosen candidate. Returns (nil, cursor, cursor) when
// no completions apply.
//
// Two contexts are recognized:
//
//   - Field-name position: the current token (between whitespace and
//     the cursor) does not contain ':'. Candidates are the queryable
//     field names derived from cols (display names + QueryAliases).
//     A leading '-' is treated as negation and skipped over.
//
//   - Value position: the current token contains ':' before the
//     cursor. Candidates depend on the column's filter type —
//     SetFilter contributes its allValues; BoolFilter contributes
//     "true"/"false". Other filter types have no completions.
//     Comma-lists are recognized; the partial after the last comma is
//     what gets matched.
//
// All matching is case-insensitive; candidates are returned in the
// order they appear in the underlying source.
func Suggest[T any](text string, cursor int, cols []data.Column[T]) (candidates []string, start, end int) {
	if cursor < 0 || cursor > len(text) {
		return nil, cursor, cursor
	}

	// Find the start of the current token: scan back to whitespace.
	tokenStart := cursor
	for tokenStart > 0 && !isCompleteSep(text[tokenStart-1]) {
		tokenStart--
	}

	// Locate ':' and the most recent ',' between tokenStart and cursor.
	colonPos := -1
	commaPos := -1
	for i := tokenStart; i < cursor; i++ {
		switch text[i] {
		case ':':
			if colonPos < 0 {
				colonPos = i
			}
		case ',':
			if colonPos >= 0 {
				commaPos = i
			}
		}
	}

	if colonPos >= 0 {
		// Value context.
		fieldName := strings.TrimPrefix(text[tokenStart:colonPos], "-")
		valueStart := colonPos + 1
		if commaPos >= 0 {
			valueStart = commaPos + 1
		}
		valuePartial := text[valueStart:cursor]
		col := lookupCol(cols, fieldName)
		if col == nil {
			return nil, cursor, cursor
		}
		return matchPrefix(filterValueCandidates(col.Filter), valuePartial), valueStart, cursor
	}

	// Field-name context.
	fieldStart := tokenStart
	if fieldStart < cursor && text[fieldStart] == '-' {
		fieldStart++
	}
	fieldPartial := text[fieldStart:cursor]
	return matchPrefix(fieldCandidates(cols), fieldPartial), fieldStart, cursor
}

// isCompleteSep reports whether b ends a token for completion
// purposes. Whitespace only — colons and commas are part of the token.
func isCompleteSep(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// matchPrefix returns the subset of cands whose lowercased form starts
// with the lowercased partial, in their original order.
func matchPrefix(cands []string, partial string) []string {
	if len(cands) == 0 {
		return nil
	}
	lp := strings.ToLower(partial)
	var out []string
	for _, c := range cands {
		if strings.HasPrefix(strings.ToLower(c), lp) {
			out = append(out, c)
		}
	}
	return out
}

// fieldCandidates returns the user-typeable names for every queryable
// column: the display name plus any explicit QueryAliases. Hidden,
// non-Filterable, and non-RoundTrippable columns are skipped.
func fieldCandidates[T any](cols []data.Column[T]) []string {
	var out []string
	seen := make(map[string]bool)
	add := func(s string) {
		if s == "" || seen[s] {
			return
		}
		out = append(out, s)
		seen[s] = true
	}
	for i := range cols {
		c := &cols[i]
		if c.Hide || !c.Filterable || c.Filter == nil {
			continue
		}
		if _, ok := c.Filter.(filter.RoundTrippable); !ok {
			continue
		}
		add(displayFieldName(c))
		for _, a := range c.QueryAliases {
			add(a)
		}
	}
	return out
}

// filterValueCandidates returns the value-completion candidates a
// filter exposes. SetFilter contributes its allValues; BoolFilter
// contributes "true"/"false". Other filter types have no candidates.
func filterValueCandidates(f filter.Filter) []string {
	switch typed := f.(type) {
	case *filter.SetFilter:
		return typed.AllValues()
	case *filter.BoolFilter:
		return []string{"true", "false"}
	default:
		return nil
	}
}

// lookupCol resolves a field name (canonical, alias, slug, or
// lowercased ColumnID) to its column. Returns nil if not found.
func lookupCol[T any](cols []data.Column[T], fieldName string) *data.Column[T] {
	for i := range cols {
		c := &cols[i]
		if c.ColumnID == fieldName {
			return c
		}
	}
	// Try alias resolution: any column whose displayFieldName, lowercased
	// ColumnID, or QueryAliases match.
	lower := strings.ToLower(fieldName)
	for i := range cols {
		c := &cols[i]
		if displayFieldName(c) == fieldName || displayFieldName(c) == lower {
			return c
		}
		if strings.ToLower(c.ColumnID) == lower {
			return c
		}
		for _, a := range c.QueryAliases {
			if a == fieldName || a == lower {
				return c
			}
		}
	}
	return nil
}
