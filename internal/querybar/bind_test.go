package querybar

import (
	"strings"
	"testing"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/searchquery"
)

func cols2() []data.Column[map[string]any] {
	return []data.Column[map[string]any]{
		{ColumnID: "state", Filter: filter.NewSetFilter("open", "closed"), Filterable: true},
		{ColumnID: "title", Filter: filter.NewTextFilter(), Filterable: true},
		{ColumnID: "count", Filter: filter.NewNumberFilter(), Filterable: true},
	}
}

func TestRerender_NoFiltersNoBareTerms_EmptyText(t *testing.T) {
	c := cols2()
	text, lossy := Rerender(c, "")
	if text != "" {
		t.Errorf("text = %q, want empty", text)
	}
	if len(lossy) != 0 {
		t.Errorf("lossy = %v, want none", lossy)
	}
}

func TestRerender_OneClauseAndBareTerms(t *testing.T) {
	c := cols2()
	c[0].Filter.(*filter.SetFilter).Exclude("closed")
	c[1].Filter.(*filter.TextFilter).SetText("memory")
	text, lossy := Rerender(c, "extra terms")
	if !strings.Contains(text, "state:open") {
		t.Errorf("text %q missing state:open", text)
	}
	if !strings.Contains(text, "title:memory") {
		t.Errorf("text %q missing title:memory", text)
	}
	if !strings.HasSuffix(text, "extra terms") {
		t.Errorf("text %q does not end with bare terms", text)
	}
	if len(lossy) != 0 {
		t.Errorf("lossy = %v, want none", lossy)
	}
}

func TestRerender_LossyFilterAnnotated(t *testing.T) {
	c := cols2()
	c[1].Filter.(*filter.TextFilter).SetRegex(true)
	c[1].Filter.(*filter.TextFilter).SetText("foo.*bar")
	_, lossy := Rerender(c, "")
	if len(lossy) != 1 || lossy[0] != "title" {
		t.Errorf("lossy = %v, want [title]", lossy)
	}
}

func TestRerender_NumberFilterClauseFormat(t *testing.T) {
	c := cols2()
	c[2].Filter.(*filter.NumberFilter).SetText(">10")
	text, _ := Rerender(c, "")
	if !strings.Contains(text, "count:>10") {
		t.Errorf("text %q missing count:>10", text)
	}
}

func TestRerender_SetFilterCommaList(t *testing.T) {
	c := []data.Column[map[string]any]{
		{ColumnID: "state", Filter: filter.NewSetFilter("a", "b", "c", "d"), Filterable: true},
	}
	c[0].Filter.(*filter.SetFilter).Exclude("c")
	c[0].Filter.(*filter.SetFilter).Exclude("d")
	text, _ := Rerender(c, "")
	if !strings.Contains(text, "state:a,b") && !strings.Contains(text, "state:b,a") {
		t.Errorf("text %q missing state:a,b (or any order)", text)
	}
}

func TestRerender_SetFilterNegateForm(t *testing.T) {
	c := []data.Column[map[string]any]{
		{ColumnID: "state", Filter: filter.NewSetFilter("a", "b", "c"), Filterable: true},
	}
	c[0].Filter.(*filter.SetFilter).Exclude("a")
	text, _ := Rerender(c, "")
	if !strings.Contains(text, "-state:a") {
		t.Errorf("text %q missing -state:a (negation flip)", text)
	}
}

func TestRerender_MultiSetRepeatedClauses(t *testing.T) {
	c := []data.Column[map[string]any]{
		{ColumnID: "label", Filter: filter.NewMultiSetFilter(), Filterable: true},
	}
	mf := c[0].Filter.(*filter.MultiSetFilter)
	mf.AddConstraint("bug")
	mf.AddConstraint("urgent")
	text, _ := Rerender(c, "")
	if !strings.Contains(text, "label:bug") || !strings.Contains(text, "label:urgent") {
		t.Errorf("text %q missing label:bug or label:urgent", text)
	}
}

func TestRerender_NonRoundTrippableFilterIsLossy(t *testing.T) {
	c := []data.Column[map[string]any]{
		{ColumnID: "x", Filter: &nonRTFilter{}, Filterable: true},
	}
	_, lossy := Rerender(c, "")
	if len(lossy) != 0 {
		t.Errorf("inactive non-RT filter should not be lossy; got %v", lossy)
	}
}

func vocab2() *searchquery.Vocabulary {
	return BuildAutoVocab(cols2())
}

func TestApply_ScalarTextFilter(t *testing.T) {
	c := cols2()
	ast, _ := searchquery.Parse("title:memory", vocab2())
	res := Apply(c, ast)
	if res.BareTerms != "" {
		t.Errorf("BareTerms = %q, want empty", res.BareTerms)
	}
	if len(res.Errors) != 0 {
		t.Errorf("Errors = %v, want none", res.Errors)
	}
	tf := c[1].Filter.(*filter.TextFilter)
	if !tf.Active() {
		t.Errorf("TextFilter inactive after Apply")
	}
	values, _, _ := tf.Clause()
	if len(values) != 1 || values[0] != "memory" {
		t.Errorf("after Apply: values=%v, want [memory]", values)
	}
}

func TestApply_SetFilterCommaList(t *testing.T) {
	c := cols2()
	ast, _ := searchquery.Parse("state:open,closed", vocab2())
	Apply(c, ast)
	sf := c[0].Filter.(*filter.SetFilter)
	if !sf.Matches("open") || !sf.Matches("closed") {
		t.Errorf("SetFilter should include open and closed")
	}
}

func TestApply_BareTermsExtracted(t *testing.T) {
	c := cols2()
	ast, _ := searchquery.Parse("memory leak", vocab2())
	res := Apply(c, ast)
	if res.BareTerms != "memory leak" {
		t.Errorf("BareTerms = %q, want %q", res.BareTerms, "memory leak")
	}
}

func TestApply_ClearsFiltersNotMentioned(t *testing.T) {
	c := cols2()
	c[1].Filter.(*filter.TextFilter).SetText("old text")
	ast, _ := searchquery.Parse("state:open", vocab2())
	Apply(c, ast)
	if c[1].Filter.(*filter.TextFilter).Active() {
		t.Errorf("title TextFilter should be cleared (not in submitted query)")
	}
}

func TestApply_LossyFilterLeftAlone(t *testing.T) {
	c := cols2()
	c[1].Filter.(*filter.TextFilter).SetRegex(true)
	c[1].Filter.(*filter.TextFilter).SetText("foo.*")
	ast, _ := searchquery.Parse("state:open", vocab2())
	Apply(c, ast)
	tf := c[1].Filter.(*filter.TextFilter)
	if !tf.Active() {
		t.Errorf("lossy TextFilter (regex) should not be cleared by Apply")
	}
}

func TestApply_MultiSetClearsAndAccumulates(t *testing.T) {
	c := []data.Column[map[string]any]{
		{ColumnID: "label", Filter: filter.NewMultiSetFilter(), Filterable: true},
	}
	mf := c[0].Filter.(*filter.MultiSetFilter)
	mf.AddConstraint("old")
	v := BuildAutoVocab(c)
	ast, _ := searchquery.Parse("label:bug label:urgent", v)
	Apply(c, ast)
	cs := mf.Constraints()
	if len(cs) != 2 || cs[0] != "bug" || cs[1] != "urgent" {
		t.Errorf("after Apply: %v, want [bug urgent] (old cleared, new appended)", cs)
	}
}

func TestApply_PerClauseErrorContinues(t *testing.T) {
	c := cols2()
	ast, _ := searchquery.Parse("count:notanumber state:open", vocab2())
	res := Apply(c, ast)
	if len(res.Errors) != 1 {
		t.Errorf("Errors = %v, want exactly one", res.Errors)
	}
	if !c[0].Filter.(*filter.SetFilter).Matches("open") {
		t.Errorf("good clause did not apply despite bad sibling")
	}
	if c[2].Filter.(*filter.NumberFilter).Active() {
		t.Errorf("bad clause should not have set NumberFilter active")
	}
}

func TestApply_UnknownFieldIgnored(t *testing.T) {
	c := cols2()
	ast, _ := searchquery.Parse("nonexistent:foo state:open", vocab2())
	res := Apply(c, ast)
	if len(res.Errors) != 0 {
		t.Errorf("unknown field should not produce error; got %v", res.Errors)
	}
	if !c[0].Filter.(*filter.SetFilter).Matches("open") {
		t.Errorf("known clause did not apply alongside unknown")
	}
}
