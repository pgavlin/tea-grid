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

// BuildAutoVocab derives a *searchquery.Vocabulary from the column set.
// Every visible, filterable column with a RoundTrippable filter becomes
// a queryable Field; field type is inferred from filter type; aliases
// come from Column.QueryAliases. A lowercase alias is also registered
// for any column whose ColumnID has uppercase characters, so users can
// type `state:open` against a `State` column without having to match
// case. Columns with no filter, a non-RoundTrippable filter, Hide=true,
// or Filterable=false are skipped.
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
		if lower := strings.ToLower(c.ColumnID); lower != c.ColumnID {
			aliases = append(aliases, lower)
		}
		fields = append(fields, searchquery.Field{
			Name:        c.ColumnID,
			Aliases:     aliases,
			Type:        inferFieldType(c.Filter),
			AcceptsList: filterAcceptsList(c.Filter),
		})
	}
	return searchquery.NewVocabulary(fields)
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
