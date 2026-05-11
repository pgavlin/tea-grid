package searchquery

import (
	"fmt"
	"strings"
)

// Range is the parsed form of the comparator/range grammar shared by
// time and numeric filters. From and To are nil when unbounded
// (">=5" has To == nil; "*..10" has From == nil). FromExclusive and
// ToExclusive carry the operator semantics:
//
//	x       -> From == To == &x, neither exclusive
//	>x      -> From = &x, FromExclusive = true
//	>=x     -> From = &x
//	<x      -> To = &x, ToExclusive = true
//	<=x     -> To = &x
//	a..b    -> From = &a, To = &b, neither exclusive
//	a..*    -> From = &a
//	*..b    -> To = &b
type Range[T any] struct {
	From, To                   *T
	FromExclusive, ToExclusive bool
}

// ParseRange parses the comparator/range grammar into a typed Range.
// The caller supplies parse() so the function is value-agnostic; the
// callback is responsible for any value-specific cleanup (e.g.
// stripping a trailing T... suffix from an ISO timestamp).
//
// An empty input, or an input whose value portion fails to parse,
// returns an error.
func ParseRange[T any](s string, parse func(string) (T, error)) (Range[T], error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Range[T]{}, fmt.Errorf("empty range value")
	}

	// Range form first (must check before comparator forms -- `..`
	// could otherwise be mistaken for part of a value).
	if i := strings.Index(s, ".."); i >= 0 {
		left, right := s[:i], s[i+2:]
		var r Range[T]
		if left != "*" {
			v, err := parse(left)
			if err != nil {
				return Range[T]{}, err
			}
			r.From = &v
		}
		if right != "*" {
			v, err := parse(right)
			if err != nil {
				return Range[T]{}, err
			}
			r.To = &v
		}
		return r, nil
	}

	// Comparator form. `>=` and `<=` must come before `>` and `<` so
	// the two-byte prefixes win.
	switch {
	case strings.HasPrefix(s, ">="):
		v, err := parse(s[2:])
		if err != nil {
			return Range[T]{}, err
		}
		return Range[T]{From: &v}, nil
	case strings.HasPrefix(s, "<="):
		v, err := parse(s[2:])
		if err != nil {
			return Range[T]{}, err
		}
		return Range[T]{To: &v}, nil
	case strings.HasPrefix(s, ">"):
		v, err := parse(s[1:])
		if err != nil {
			return Range[T]{}, err
		}
		return Range[T]{From: &v, FromExclusive: true}, nil
	case strings.HasPrefix(s, "<"):
		v, err := parse(s[1:])
		if err != nil {
			return Range[T]{}, err
		}
		return Range[T]{To: &v, ToExclusive: true}, nil
	}

	// Bare value -- exact match (inclusive on both sides).
	v, err := parse(s)
	if err != nil {
		return Range[T]{}, err
	}
	return Range[T]{From: &v, To: &v}, nil
}
