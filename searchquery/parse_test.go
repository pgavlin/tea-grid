package searchquery

import "testing"

func equalTokens(a, b []token) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTokenize_BareWords(t *testing.T) {
	toks, err := tokenize("memory leak")
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	want := []token{
		{kind: tokWord, value: "memory"},
		{kind: tokWord, value: "leak"},
	}
	if !equalTokens(toks, want) {
		t.Errorf("got %v, want %v", toks, want)
	}
}

func TestTokenize_Clause(t *testing.T) {
	toks, err := tokenize("state:open")
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	want := []token{
		{kind: tokWord, value: "state"},
		{kind: tokColon},
		{kind: tokWord, value: "open"},
	}
	if !equalTokens(toks, want) {
		t.Errorf("got %v, want %v", toks, want)
	}
}

func TestTokenize_Quoted(t *testing.T) {
	toks, err := tokenize(`label:"good first issue"`)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	want := []token{
		{kind: tokWord, value: "label"},
		{kind: tokColon},
		{kind: tokQuoted, value: "good first issue"},
	}
	if !equalTokens(toks, want) {
		t.Errorf("got %v, want %v", toks, want)
	}
}

func TestTokenize_QuotedEscapes(t *testing.T) {
	toks, err := tokenize(`field:"a\"b\\c"`)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	if len(toks) != 3 || toks[2].value != `a"b\c` {
		t.Errorf("got %v, want quoted value `a\"b\\c`", toks)
	}
}

func TestTokenize_Comma(t *testing.T) {
	toks, err := tokenize("state:open,closed")
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	want := []token{
		{kind: tokWord, value: "state"},
		{kind: tokColon},
		{kind: tokWord, value: "open"},
		{kind: tokComma},
		{kind: tokWord, value: "closed"},
	}
	if !equalTokens(toks, want) {
		t.Errorf("got %v, want %v", toks, want)
	}
}

func TestTokenize_LeadingDash(t *testing.T) {
	toks, err := tokenize("-state:closed")
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	want := []token{
		{kind: tokDash},
		{kind: tokWord, value: "state"},
		{kind: tokColon},
		{kind: tokWord, value: "closed"},
	}
	if !equalTokens(toks, want) {
		t.Errorf("got %v, want %v", toks, want)
	}
}

func TestTokenize_EmbeddedDash(t *testing.T) {
	// "good-first-issue" is one bare word — only leading dashes negate.
	toks, err := tokenize("good-first-issue")
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	if len(toks) != 1 || toks[0].value != "good-first-issue" {
		t.Errorf("got %v, want one bare word", toks)
	}
}

func TestTokenize_UnclosedQuote(t *testing.T) {
	if _, err := tokenize(`field:"abc`); err == nil {
		t.Error("expected unclosed-quote error")
	}
}

func TestParse_BareTerms(t *testing.T) {
	v := NewVocabulary(nil)
	ast, err := Parse("memory leak", v)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ast.Terms != "memory leak" {
		t.Errorf("Terms = %q, want %q", ast.Terms, "memory leak")
	}
	if len(ast.Clauses) != 0 {
		t.Errorf("Clauses = %v, want none", ast.Clauses)
	}
}

func TestParse_OneClause(t *testing.T) {
	v := NewVocabulary([]Field{{Name: "state"}})
	ast, err := Parse("state:open", v)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(ast.Clauses) != 1 {
		t.Fatalf("Clauses = %v, want 1", ast.Clauses)
	}
	c := ast.Clauses[0]
	if c.Field != "state" || len(c.Values) != 1 || c.Values[0] != "open" || c.Negate {
		t.Errorf("got %+v", c)
	}
	if ast.Terms != "" {
		t.Errorf("Terms = %q, want empty", ast.Terms)
	}
}

func TestParse_ClauseAndTerms(t *testing.T) {
	v := NewVocabulary([]Field{{Name: "repo"}})
	ast, err := Parse("memory repo:foo leak", v)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(ast.Clauses) != 1 || ast.Clauses[0].Field != "repo" {
		t.Errorf("expected one repo clause, got %+v", ast.Clauses)
	}
	if ast.Terms != "memory leak" {
		t.Errorf("Terms = %q, want %q", ast.Terms, "memory leak")
	}
}

func TestParse_AliasResolved(t *testing.T) {
	v := NewVocabulary([]Field{{Name: "state", Aliases: []string{"status"}}})
	ast, err := Parse("status:open", v)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ast.Clauses[0].Field != "state" {
		t.Errorf("alias not resolved; got Field=%q", ast.Clauses[0].Field)
	}
}

func TestParse_UnknownFieldKept(t *testing.T) {
	// Unknown fields parse permissively — binders surface them as ignored.
	v := NewVocabulary(nil)
	ast, err := Parse("stat:open", v)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(ast.Clauses) != 1 || ast.Clauses[0].Field != "stat" {
		t.Errorf("unknown field should parse as-is; got %+v", ast.Clauses)
	}
}

func TestParse_CommaList(t *testing.T) {
	v := NewVocabulary([]Field{{Name: "state", AcceptsList: true}})
	ast, err := Parse("state:open,closed", v)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c := ast.Clauses[0]
	if len(c.Values) != 2 || c.Values[0] != "open" || c.Values[1] != "closed" {
		t.Errorf("got Values=%v", c.Values)
	}
}

func TestParse_QuotedValue(t *testing.T) {
	v := NewVocabulary([]Field{{Name: "label"}})
	ast, err := Parse(`label:"good first issue"`, v)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ast.Clauses[0].Values[0] != "good first issue" {
		t.Errorf("got Values=%v", ast.Clauses[0].Values)
	}
}

func TestParse_QuotedBareTerm(t *testing.T) {
	v := NewVocabulary(nil)
	ast, err := Parse(`"memory leak"`, v)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if ast.Terms != "memory leak" {
		t.Errorf("got Terms=%q", ast.Terms)
	}
}

func TestParse_Negate(t *testing.T) {
	v := NewVocabulary([]Field{{Name: "state"}})
	ast, err := Parse("-state:closed", v)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c := ast.Clauses[0]
	if !c.Negate || c.Field != "state" || c.Values[0] != "closed" {
		t.Errorf("got %+v", c)
	}
}

func TestParse_NegateWithList(t *testing.T) {
	v := NewVocabulary([]Field{{Name: "state", AcceptsList: true}})
	ast, err := Parse("-state:open,closed", v)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c := ast.Clauses[0]
	if !c.Negate || len(c.Values) != 2 {
		t.Errorf("got %+v", c)
	}
}

func TestParse_DanglingDash(t *testing.T) {
	v := NewVocabulary(nil)
	if _, err := Parse("-", v); err == nil {
		t.Error("expected dangling-dash error")
	}
}

func TestParse_NegateOnBareTerm(t *testing.T) {
	v := NewVocabulary(nil)
	if _, err := Parse("-foo", v); err == nil {
		t.Error("expected error: '-' must precede a clause")
	}
}

// rewriteVocab builds a small vocabulary that exercises the rewrite
// path without depending on any application-level field set. `is:` is
// rewritten to either `state:<value>` or to `has:linked` — the
// "unlinked" alias asks the parser to flip negation so the resulting
// clause is `-has:linked`.
func rewriteVocab() *Vocabulary {
	v := NewVocabulary([]Field{
		{Name: "state"},
		{Name: "has"},
		{Name: "is"},
	})
	v.AddRewrite("is", func(value string) (string, string, bool, bool) {
		switch value {
		case "open", "closed":
			return "state", value, false, true
		case "linked":
			return "has", "linked", false, true
		case "unlinked":
			return "has", "linked", true, true // flip negate
		}
		return "", "", false, false
	})
	return v
}

func TestParse_RewriteSimple(t *testing.T) {
	ast, err := Parse("is:open", rewriteVocab())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c := ast.Clauses[0]
	if c.Field != "state" || c.Values[0] != "open" || c.Negate {
		t.Errorf("got %+v", c)
	}
}

func TestParse_RewriteFlipsNegateOnUnlinked(t *testing.T) {
	// is:unlinked rewrites to has:linked AND inverts the user-supplied
	// negate. Tests the parser's flipNegate code path.
	ast, err := Parse("is:unlinked", rewriteVocab())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c := ast.Clauses[0]
	if c.Field != "has" || c.Values[0] != "linked" || !c.Negate {
		t.Errorf("got %+v — expected -has:linked", c)
	}
}

func TestParse_RewriteAllValuesInList(t *testing.T) {
	// is:open,closed — both values rewrite to the same canonical field
	// so the list collapses into a single clause.
	ast, err := Parse("is:open,closed", rewriteVocab())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(ast.Clauses) != 1 || ast.Clauses[0].Field != "state" {
		t.Fatalf("expected one merged clause; got %+v", ast.Clauses)
	}
	if len(ast.Clauses[0].Values) != 2 {
		t.Errorf("expected two values; got %v", ast.Clauses[0].Values)
	}
}

func TestVocabulary_ResolveCanonical(t *testing.T) {
	v := NewVocabulary([]Field{{Name: "state"}})
	got, ok := v.Resolve("state")
	if !ok || got != "state" {
		t.Errorf("Resolve(state) = (%q, %v), want (state, true)", got, ok)
	}
}

func TestVocabulary_ResolveAlias(t *testing.T) {
	v := NewVocabulary([]Field{{Name: "state", Aliases: []string{"status"}}})
	got, ok := v.Resolve("status")
	if !ok || got != "state" {
		t.Errorf("Resolve(status) = (%q, %v), want (state, true)", got, ok)
	}
}

func TestVocabulary_ResolveUnknown(t *testing.T) {
	v := NewVocabulary([]Field{{Name: "state"}})
	if _, ok := v.Resolve("bogus"); ok {
		t.Error("Resolve(bogus) should not succeed")
	}
}

func TestVocabulary_Rewrite(t *testing.T) {
	v := NewVocabulary([]Field{{Name: "is"}})
	v.AddRewrite("is", func(value string) (string, string, bool, bool) {
		if value == "open" {
			return "state", "open", false, true
		}
		return "", "", false, false
	})
	f, val, neg, ok := v.Rewrite("is", "open")
	if !ok || f != "state" || val != "open" || neg {
		t.Errorf("Rewrite(is, open) = (%q, %q, neg=%v, ok=%v), want (state, open, false, true)", f, val, neg, ok)
	}
}
