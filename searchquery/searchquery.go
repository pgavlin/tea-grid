// Package searchquery implements a small GitHub-style query language for
// filtering tabular data. The grammar is a thin layer:
//
//	query    := token (WS+ token)*
//	token    := "-"? clause | bareword | quoted
//	clause   := ident ":" value ("," value)*
//	value    := bareword | quoted
//
// Bare terms accumulate as a free-text query (suitable for tea-grid's
// quick-filter mechanism); field:value tokens become typed clauses
// against named columns. Comma-separated values list as OR within one
// clause; repeated clauses on the same field default to AND across the
// AST (with the exact semantics chosen by each binder).
//
// The package is split into three layers consumers compose as needed:
//
//  1. Field metadata + Vocabulary — names, aliases, parse-time rewrites.
//  2. Parser — produces a content-agnostic AST of clauses + bare terms.
//  3. Binders (in callers' packages) — turn the AST into engine-specific
//     predicates: SQL, in-memory row matchers, tea-grid column filters,
//     etc. Layers 1 and 2 know nothing about any backend.
package searchquery

// FieldType discriminates the value-shape a field accepts. Used by the
// parser to validate values and by binders to decide how to coerce them.
type FieldType int

const (
	// FieldString is the default — values are arbitrary strings.
	FieldString FieldType = iota
	// FieldTime values use the GitHub-style time grammar (see timeparse.go).
	FieldTime
	// FieldBool values are "true" / "false" / "1" / "0".
	FieldBool
)

// Field is the parser-visible shape of one queryable token. The same
// Field definition is reused by every binder; what differs is how each
// binder translates the resulting Clause into engine-specific predicates.
type Field struct {
	Name        string
	Aliases     []string
	Description string
	Type        FieldType
	AcceptsList bool
}

// Rewrite handles parse-time sugar (e.g. is:open → state:open). Returns
// (canonicalField, canonicalValue, negate, true) when it matches;
// (_, _, _, false) when the input doesn't apply to this rewrite.
//
// The negate return inverts the parser's user-supplied negation flag,
// allowing single-token sugar that means "the negation of some other
// canonical clause" — e.g. `is:unlinked` rewrites to `has:linked` with
// negate=true so the resulting clause is `-has:linked`.
type Rewrite func(value string) (canonField, canonValue string, negate, ok bool)

// Clause is one parsed token, post-alias-rewrite. Field is canonical.
// Values is non-empty; len(Values)==1 unless the source field had
// AcceptsList=true and the user wrote a comma-separated list.
type Clause struct {
	Field  string
	Values []string
	Negate bool
}

// AST is the parsed query: a sequence of clauses plus the residual bare
// terms (joined with single spaces) ready to be passed to FTS or to a
// quick-filter style substring matcher.
type AST struct {
	Clauses []Clause
	Terms   string
}

// Vocabulary is the registry the parser consults for alias resolution
// and rewrites. Build it once for a static field set; rebuild at parse
// time if the queryable surface depends on dynamic input (e.g. the set
// of currently-bound columns).
type Vocabulary struct {
	fields   map[string]*Field
	aliases  map[string]string
	rewrites map[string]Rewrite
}

// NewVocabulary builds a Vocabulary from a slice of Fields. The slice
// is copied; mutating it after the call has no effect on the Vocabulary.
func NewVocabulary(fields []Field) *Vocabulary {
	v := &Vocabulary{
		fields:   make(map[string]*Field),
		aliases:  make(map[string]string),
		rewrites: make(map[string]Rewrite),
	}
	for i := range fields {
		f := fields[i]
		v.fields[f.Name] = &f
		for _, a := range f.Aliases {
			v.aliases[a] = f.Name
		}
	}
	return v
}

// AddRewrite registers a parse-time rewrite for a field. Used by sugar
// layers like `is:` that map several user-facing values onto canonical
// (field, value) pairs.
func (v *Vocabulary) AddRewrite(name string, r Rewrite) {
	v.rewrites[name] = r
}

// Resolve maps an alias (or canonical name) to its canonical name.
// Returns ("", false) when the name isn't in the vocabulary.
func (v *Vocabulary) Resolve(name string) (string, bool) {
	if _, ok := v.fields[name]; ok {
		return name, true
	}
	if canon, ok := v.aliases[name]; ok {
		return canon, true
	}
	return "", false
}

// Field returns the Field struct for a canonical name.
func (v *Vocabulary) Field(name string) (*Field, bool) {
	f, ok := v.fields[name]
	return f, ok
}

// Rewrite applies the field's rewrite to a value, returning the
// canonicalized clause if the rewrite matches. The negate return
// indicates whether the rewrite implies inverting the parser's
// user-supplied negation flag.
func (v *Vocabulary) Rewrite(name, value string) (canonField, canonValue string, negate, ok bool) {
	r, found := v.rewrites[name]
	if !found {
		return "", "", false, false
	}
	return r(value)
}
