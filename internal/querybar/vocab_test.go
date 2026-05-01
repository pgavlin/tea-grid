package querybar

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/searchquery"
)

func TestBuildAutoVocab_Empty(t *testing.T) {
	v := BuildAutoVocab[map[string]any](nil)
	if v == nil {
		t.Fatal("BuildAutoVocab(nil) = nil")
	}
	if _, ok := v.Resolve("anything"); ok {
		t.Errorf("empty vocab resolved unexpected name")
	}
}

func TestBuildAutoVocab_TextColumn(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "title", Filter: filter.NewTextFilter(), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	canon, ok := v.Resolve("title")
	if !ok || canon != "title" {
		t.Errorf("Resolve(title) = (%q, %v), want (title, true)", canon, ok)
	}
	f, ok := v.Field("title")
	if !ok || f.Type != searchquery.FieldString {
		t.Errorf("Field(title).Type = %v, want FieldString", f.Type)
	}
}

func TestBuildAutoVocab_TimeColumnInfersTimeType(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "created", Filter: filter.NewTimeFilter(), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	f, ok := v.Field("created")
	if !ok || f.Type != searchquery.FieldTime {
		t.Errorf("Field(created).Type = %v, want FieldTime", f.Type)
	}
}

func TestBuildAutoVocab_BoolColumnInfersBoolType(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "active", Filter: filter.NewBoolFilter(), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	f, _ := v.Field("active")
	if f.Type != searchquery.FieldBool {
		t.Errorf("Field(active).Type = %v, want FieldBool", f.Type)
	}
}

func TestBuildAutoVocab_SetColumnAcceptsList(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "label", Filter: filter.NewSetFilter("a", "b"), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	f, _ := v.Field("label")
	if !f.AcceptsList {
		t.Errorf("Field(label).AcceptsList = false, want true (SetFilter)")
	}
}

func TestBuildAutoVocab_MultiSetColumnSingleValue(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "tag", Filter: filter.NewMultiSetFilter(), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	f, _ := v.Field("tag")
	if f.AcceptsList {
		t.Errorf("Field(tag).AcceptsList = true, want false (MultiSetFilter)")
	}
}

func TestBuildAutoVocab_AliasesFromColumn(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{
			ColumnID:     "state",
			Filter:       filter.NewSetFilter("open", "closed"),
			Filterable:   true,
			QueryAliases: []string{"status"},
		},
	}
	v := BuildAutoVocab(cols)
	canon, ok := v.Resolve("status")
	if !ok || canon != "state" {
		t.Errorf("Resolve(status) = (%q, %v), want (state, true)", canon, ok)
	}
}

func TestBuildAutoVocab_LowercaseAliasForCapitalizedID(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "State", Filter: filter.NewSetFilter("open", "closed"), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	canon, ok := v.Resolve("state")
	if !ok || canon != "State" {
		t.Errorf("Resolve(state) = (%q, %v), want (State, true) — auto lowercase alias", canon, ok)
	}
}

func TestBuildAutoVocab_SkipsNonRoundTrippableFilters(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "x", Filter: &nonRTFilter{}, Filterable: true},
	}
	v := BuildAutoVocab(cols)
	if _, ok := v.Field("x"); ok {
		t.Errorf("Field(x) = ok, want skipped (filter not RoundTrippable)")
	}
}

func TestBuildAutoVocab_SkipsHiddenAndUnfilterable(t *testing.T) {
	cols := []data.Column[map[string]any]{
		{ColumnID: "hidden", Filter: filter.NewTextFilter(), Filterable: true, Hide: true},
		{ColumnID: "unfilterable", Filter: filter.NewTextFilter(), Filterable: false},
		{ColumnID: "ok", Filter: filter.NewTextFilter(), Filterable: true},
	}
	v := BuildAutoVocab(cols)
	if _, ok := v.Field("hidden"); ok {
		t.Errorf("hidden should be skipped")
	}
	if _, ok := v.Field("unfilterable"); ok {
		t.Errorf("unfilterable should be skipped")
	}
	if _, ok := v.Field("ok"); !ok {
		t.Errorf("ok should be included")
	}
}

// nonRTFilter is a Filter that does not implement RoundTrippable.
type nonRTFilter struct{}

func (n *nonRTFilter) Matches(value any) bool                      { return true }
func (n *nonRTFilter) View() string                                { return "" }
func (n *nonRTFilter) Update(msg tea.Msg) (filter.Filter, tea.Cmd) { return n, nil }
func (n *nonRTFilter) Active() bool                                { return false }
func (n *nonRTFilter) Clear()                                      {}
