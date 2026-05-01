package searchquery

import (
	"fmt"
	"strings"
	"unicode"
)

type tokenKind int

const (
	tokWord tokenKind = iota
	tokQuoted
	tokColon
	tokComma
	tokDash
)

type token struct {
	kind  tokenKind
	value string
}

// tokenize splits an input string into tokens. Whitespace separates
// tokens outside quotes; bare-word tokens stop at any structural rune
// (`:`, `,`, whitespace) and a leading `-` is its own token.
//
// Embedded dashes are part of the bare word ("good-first-issue" → one
// token). Only a `-` at the start of a fresh token (just after
// whitespace, or at the input's beginning) is treated as negation.
func tokenize(s string) ([]token, error) {
	var toks []token
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case unicode.IsSpace(rune(c)):
			i++
		case c == '-' && (i == 0 || unicode.IsSpace(rune(s[i-1]))):
			toks = append(toks, token{kind: tokDash})
			i++
		case c == ':':
			toks = append(toks, token{kind: tokColon})
			i++
		case c == ',':
			toks = append(toks, token{kind: tokComma})
			i++
		case c == '"':
			val, n, err := scanQuoted(s[i:])
			if err != nil {
				return nil, err
			}
			toks = append(toks, token{kind: tokQuoted, value: val})
			i += n
		default:
			start := i
			for i < len(s) && !isStructural(s[i]) {
				i++
			}
			toks = append(toks, token{kind: tokWord, value: s[start:i]})
		}
	}
	return toks, nil
}

// isStructural reports whether the byte is a token boundary in bare-word
// scanning. Whitespace, `:`, `,`, and `"` end a bare word. Dashes don't —
// they're part of multi-segment identifiers like "good-first-issue".
func isStructural(b byte) bool {
	return b == ':' || b == ',' || b == '"' || unicode.IsSpace(rune(b))
}

// Parse turns an input query string into an AST. Field names not in
// the vocabulary are kept verbatim — binders surface them as "ignored"
// hints rather than failing the whole query. Tokenization errors do
// propagate.
func Parse(input string, vocab *Vocabulary) (AST, error) {
	toks, err := tokenize(input)
	if err != nil {
		return AST{}, err
	}

	var ast AST
	var bareTerms []string
	i := 0
	for i < len(toks) {
		negate := false
		if toks[i].kind == tokDash {
			negate = true
			i++
			if i >= len(toks) {
				return AST{}, fmt.Errorf("dangling '-'")
			}
		}

		// A clause is `word ":" valueList`. Anything else is a bare term.
		if i+1 < len(toks) && toks[i].kind == tokWord && toks[i+1].kind == tokColon {
			fieldName := toks[i].value
			i += 2 // consume name + colon
			values, consumed, err := parseValueList(toks[i:])
			if err != nil {
				return AST{}, err
			}
			i += consumed

			canonical := fieldName
			if c, ok := vocab.Resolve(fieldName); ok {
				canonical = c
			}

			// Apply field-level rewrites (e.g. is:open → state:open). A
			// rewrite may also flip the negation flag — `is:unlinked`
			// rewrites to `has:linked` with negate inverted.
			if rewritten, rewriteField, allRewrote, flipNegate, err := applyRewrites(
				vocab, canonical, values,
			); err != nil {
				return AST{}, err
			} else if allRewrote {
				canonical = rewriteField
				values = rewritten
				if flipNegate {
					negate = !negate
				}
			}

			ast.Clauses = append(ast.Clauses, Clause{
				Field:  canonical,
				Values: values,
				Negate: negate,
			})
			continue
		}

		if negate {
			return AST{}, fmt.Errorf("'-' must precede a field:value clause")
		}

		// Plain word (or quoted) — bare term for the FTS query.
		switch toks[i].kind {
		case tokWord, tokQuoted:
			bareTerms = append(bareTerms, toks[i].value)
			i++
		default:
			return AST{}, fmt.Errorf("unexpected token at position %d", i)
		}
	}

	ast.Terms = strings.Join(bareTerms, " ")
	return ast, nil
}

// applyRewrites runs each value through the field's rewrite (if any)
// and merges the results. Returns:
//
//	rewritten: canonical values
//	rewriteField: canonical field (must be the same for all values)
//	allRewrote: whether ALL values matched the rewrite
//	flipNegate: whether at least one value's rewrite asked to invert
//	            negation (e.g. `is:unlinked` → `has:linked` with negate
//	            flipped). Driven by the rewrite's negate return; the
//	            parser stays agnostic to specific value semantics.
//
// If any value didn't match the rewrite, allRewrote is false and the
// caller falls back to the original field+values. We intentionally do
// not support rewrites that change the field name across some-but-not-
// all values in the same clause — `is:open,foo` (where foo doesn't
// rewrite) keeps the original `is:open,foo` clause and lets the binder
// deal with it.
func applyRewrites(vocab *Vocabulary, field string, values []string) (
	rewritten []string, rewriteField string, allRewrote, flipNegate bool, err error,
) {
	for i, v := range values {
		rf, rv, neg, ok := vocab.Rewrite(field, v)
		if !ok {
			return nil, "", false, false, nil
		}
		if i == 0 {
			rewriteField = rf
		} else if rf != rewriteField {
			return nil, "", false, false, fmt.Errorf("mixed rewrite for %s:%v", field, values)
		}
		if neg {
			flipNegate = true
		}
		rewritten = append(rewritten, rv)
	}
	return rewritten, rewriteField, true, flipNegate, nil
}

// parseValueList consumes one or more comma-separated values starting
// at toks[0]. Returns the values and the number of tokens consumed.
func parseValueList(toks []token) ([]string, int, error) {
	if len(toks) == 0 {
		return nil, 0, fmt.Errorf("expected value")
	}
	var values []string
	i := 0
	for {
		if i >= len(toks) || (toks[i].kind != tokWord && toks[i].kind != tokQuoted) {
			return nil, 0, fmt.Errorf("expected value")
		}
		values = append(values, toks[i].value)
		i++
		if i >= len(toks) || toks[i].kind != tokComma {
			break
		}
		i++ // consume comma; next iteration parses next value
	}
	return values, i, nil
}

// scanQuoted reads a "..." token starting at s[0]. Supports \" and \\
// escapes — any other escape sequence keeps the literal character that
// follows the backslash. Returns the unescaped value and the number of
// input bytes consumed (including both quote characters).
func scanQuoted(s string) (string, int, error) {
	if len(s) == 0 || s[0] != '"' {
		return "", 0, fmt.Errorf("expected '\"'")
	}
	var b strings.Builder
	i := 1
	for i < len(s) {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			b.WriteByte(s[i+1])
			i += 2
			continue
		}
		if c == '"' {
			return b.String(), i + 1, nil
		}
		b.WriteByte(c)
		i++
	}
	return "", 0, fmt.Errorf("unterminated quoted value")
}
