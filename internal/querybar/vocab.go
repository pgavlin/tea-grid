// Package querybar implements the query-bar widget used by tea-grid's
// grid.Model[T]. The package is internal because its surface is tightly
// coupled to grid internals; consumers configure the bar via the
// re-exported options in package grid.
package querybar

import (
	"strings"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/searchquery"
)

// MetaFieldHas is the canonical name of the has: meta-field used for
// presence checks. Each value names a column that must be non-empty.
const MetaFieldHas = "has"

// MetaFieldNo is the inverse of MetaFieldHas: each value names a
// column that must be empty.
const MetaFieldNo = "no"

// BuildAutoVocab derives a *searchquery.Vocabulary from the column set.
// Every visible, filterable column with a RoundTrippable filter becomes
// a queryable Field; field type is inferred from filter type; aliases
// come from Column.QueryAliases. Two more aliases are auto-registered
// so users can query by what they see in the grid:
//
//   - the lowercased ColumnID, when it has uppercase characters
//     (`state:open` matches a "State" column);
//   - a lowercased + slugified form of HeaderName, when the slug is
//     non-empty and differs from the ColumnID (`population:>5000000`
//     matches a column with HeaderName "Population" and a synthetic
//     ColumnID like "col2").
//
// Slugification lowercases and replaces any run of non-alphanumeric
// characters with a single underscore, trimming leading/trailing
// underscores. Columns with no filter, a non-RoundTrippable filter,
// Hide=true, or Filterable=false are skipped.
func BuildAutoVocab[T any](cols []data.Column[T]) *searchquery.Vocabulary {
	var fields []searchquery.Field
	for i := range cols {
		c := &cols[i]
		if c.Hide || !c.Filterable || c.Filter == nil {
			continue
		}
		if _, ok := c.Filter.(filter.RoundTrippable); !ok {
			continue
		}
		aliases := append([]string(nil), c.QueryAliases...)
		seen := make(map[string]bool, len(aliases)+2)
		seen[c.ColumnID] = true
		for _, a := range aliases {
			seen[a] = true
		}
		addAlias := func(s string) {
			if s == "" || seen[s] {
				return
			}
			aliases = append(aliases, s)
			seen[s] = true
		}
		addAlias(strings.ToLower(c.ColumnID))
		addAlias(slugifyHeader(c.HeaderName))
		fields = append(fields, searchquery.Field{
			Name:        c.ColumnID,
			Aliases:     aliases,
			Type:        inferFieldType(c.Filter),
			AcceptsList: filterAcceptsList(c.Filter),
		})
	}
	// Register the has:/no: meta-fields so the parser resolves them
	// (without these, `has:state` would parse with field="has" and the
	// binder would never see it as a meta-clause).
	fields = append(fields,
		searchquery.Field{Name: MetaFieldHas, Type: searchquery.FieldString, AcceptsList: true},
		searchquery.Field{Name: MetaFieldNo, Type: searchquery.FieldString, AcceptsList: true},
	)
	return searchquery.NewVocabulary(fields)
}

// slugifyHeader lowercases s and replaces any run of non-alphanumeric
// characters with a single underscore, trimming the result. Returns ""
// if the slug would be empty.
func slugifyHeader(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	prevUnderscore := true // suppresses leading underscores
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevUnderscore = false
		default:
			if !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	return strings.TrimRight(b.String(), "_")
}

// inferFieldType picks a searchquery.FieldType from the concrete filter
// type. NumberFilter/SetFilter/TextFilter/MultiSetFilter all report
// FieldString — the parser is content-agnostic and the binder handles
// numeric parsing inside NumberFilter itself.
func inferFieldType(f filter.Filter) searchquery.FieldType {
	switch f.(type) {
	case *filter.TimeFilter:
		return searchquery.FieldTime
	case *filter.BoolFilter:
		return searchquery.FieldBool
	default:
		return searchquery.FieldString
	}
}

// filterAcceptsList reports whether a filter accepts comma-separated
// value lists in a single clause. Only SetFilter does in v1;
// MultiSetFilter takes one value per repeated clause.
func filterAcceptsList(f filter.Filter) bool {
	_, ok := f.(*filter.SetFilter)
	return ok
}
