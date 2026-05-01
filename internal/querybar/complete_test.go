package querybar

import (
	"reflect"
	"testing"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
)

func completeCols() []data.Column[map[string]any] {
	return []data.Column[map[string]any]{
		{ColumnID: "state", Filter: filter.NewSetFilter("Open", "Closed", "Draft"), Filterable: true},
		{ColumnID: "title", Filter: filter.NewTextFilter(), Filterable: true},
		{ColumnID: "active", Filter: filter.NewBoolFilter(), Filterable: true},
	}
}

func TestSuggest_FieldNamePartial(t *testing.T) {
	cols := completeCols()
	cands, start, end := Suggest("st", 2, cols)
	if !reflect.DeepEqual(cands, []string{"state"}) {
		t.Errorf("candidates = %v, want [state]", cands)
	}
	if start != 0 || end != 2 {
		t.Errorf("range = (%d, %d), want (0, 2)", start, end)
	}
}

func TestSuggest_FieldNameWithLeadingDash(t *testing.T) {
	cols := completeCols()
	cands, start, end := Suggest("-st", 3, cols)
	if !reflect.DeepEqual(cands, []string{"state"}) {
		t.Errorf("candidates = %v, want [state]", cands)
	}
	if start != 1 || end != 3 {
		t.Errorf("range = (%d, %d), want (1, 3) — dash skipped", start, end)
	}
}

func TestSuggest_ValueAfterColon(t *testing.T) {
	cols := completeCols()
	cands, start, end := Suggest("state:o", 7, cols)
	if !reflect.DeepEqual(cands, []string{"Open"}) {
		t.Errorf("candidates = %v, want [Open]", cands)
	}
	if start != 6 || end != 7 {
		t.Errorf("range = (%d, %d), want (6, 7)", start, end)
	}
}

func TestSuggest_ValueAfterColonEmpty(t *testing.T) {
	cols := completeCols()
	cands, _, _ := Suggest("state:", 6, cols)
	if !reflect.DeepEqual(cands, []string{"Open", "Closed", "Draft"}) {
		t.Errorf("candidates = %v, want all values", cands)
	}
}

func TestSuggest_ValueAfterComma(t *testing.T) {
	cols := completeCols()
	cands, start, end := Suggest("state:Open,d", 12, cols)
	if !reflect.DeepEqual(cands, []string{"Draft"}) {
		t.Errorf("candidates = %v, want [Draft]", cands)
	}
	if start != 11 || end != 12 {
		t.Errorf("range = (%d, %d), want (11, 12)", start, end)
	}
}

func TestSuggest_BoolFilter(t *testing.T) {
	cols := completeCols()
	cands, _, _ := Suggest("active:t", 8, cols)
	if !reflect.DeepEqual(cands, []string{"true"}) {
		t.Errorf("candidates = %v, want [true]", cands)
	}
}

func TestSuggest_TextFilterNoCandidates(t *testing.T) {
	cols := completeCols()
	cands, _, _ := Suggest("title:foo", 9, cols)
	if cands != nil {
		t.Errorf("candidates = %v, want nil (TextFilter has no value completions)", cands)
	}
}

func TestSuggest_UnknownFieldNoCandidates(t *testing.T) {
	cols := completeCols()
	cands, _, _ := Suggest("nonexistent:foo", 15, cols)
	if cands != nil {
		t.Errorf("candidates = %v, want nil (unknown field)", cands)
	}
}

func TestSuggest_FieldNameMidQuery(t *testing.T) {
	cols := completeCols()
	// After whitespace, partial "ti" should match "title".
	text := "state:Open ti"
	cands, start, end := Suggest(text, len(text), cols)
	if !reflect.DeepEqual(cands, []string{"title"}) {
		t.Errorf("candidates = %v, want [title]", cands)
	}
	if start != 11 || end != 13 {
		t.Errorf("range = (%d, %d), want (11, 13)", start, end)
	}
}

func TestSuggest_HeaderSlugAlias(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "col2", HeaderName: "Population", Filter: filter.NewNumberFilter(), Filterable: true},
	}
	cands, _, _ := Suggest("pop", 3, cols)
	if !reflect.DeepEqual(cands, []string{"population"}) {
		t.Errorf("candidates = %v, want [population]", cands)
	}
}

func TestSuggest_CaseInsensitivePrefix(t *testing.T) {
	cols := completeCols()
	cands, _, _ := Suggest("state:OP", 8, cols)
	// "OP" should match "Open" case-insensitively.
	if !reflect.DeepEqual(cands, []string{"Open"}) {
		t.Errorf("candidates = %v, want [Open]", cands)
	}
}

func TestSuggest_OutOfBoundsCursor(t *testing.T) {
	cols := completeCols()
	if cands, _, _ := Suggest("abc", -1, cols); cands != nil {
		t.Errorf("negative cursor: cands=%v, want nil", cands)
	}
	if cands, _, _ := Suggest("abc", 100, cols); cands != nil {
		t.Errorf("cursor past end: cands=%v, want nil", cands)
	}
}

func TestFieldCandidates_SkipsHiddenAndNonRoundTrippable(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "visible", Filter: filter.NewTextFilter(), Filterable: true},
		{ColumnID: "hidden", Filter: filter.NewTextFilter(), Filterable: true, Hide: true},
		{ColumnID: "unfilterable", Filter: filter.NewTextFilter(), Filterable: false},
		{ColumnID: "noFilter", Filterable: true},
	}
	cands := fieldCandidates(cols)
	if len(cands) != 1 || cands[0] != "visible" {
		t.Errorf("fieldCandidates = %v, want [visible]", cands)
	}
}

func TestLookupCol_AliasResolution(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{
			ColumnID:     "State",
			HeaderName:   "State",
			Filter:       filter.NewSetFilter("a"),
			Filterable:   true,
			QueryAliases: []string{"st"},
		},
	}
	cases := []struct {
		name string
		key  string
		want bool
	}{
		{"exact ColumnID", "State", true},
		{"lowercased ColumnID", "state", true},
		{"explicit alias", "st", true},
		{"explicit alias case-insensitive", "ST", true},
		{"unknown", "nope", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lookupCol(cols, tc.key)
			if (got != nil) != tc.want {
				t.Errorf("lookupCol(%q) found=%v, want %v", tc.key, got != nil, tc.want)
			}
		})
	}
}
