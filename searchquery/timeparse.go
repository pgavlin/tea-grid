package searchquery

import (
	"fmt"
	"strings"
	"time"
)

// TimeRange is a half-open or closed interval used by created/updated
// clauses. Either bound may be nil for an open end. The grammar follows
// GitHub's date filter syntax.
type TimeRange struct {
	From          *time.Time
	To            *time.Time
	FromExclusive bool
	ToExclusive   bool
}

// ParseTimeRange parses GitHub's date grammar:
//
//	YYYY-MM-DD                exact date (From == To, both inclusive)
//	>YYYY-MM-DD               after that date (exclusive)
//	>=YYYY-MM-DD              on or after
//	<YYYY-MM-DD               before
//	<=YYYY-MM-DD              on or before
//	YYYY-MM-DD..YYYY-MM-DD    inclusive range
//	YYYY-MM-DD..*             open-ended upper
//	*..YYYY-MM-DD             open-ended lower
//
// Full ISO timestamps (with a T...Z suffix) are accepted; the time
// portion is silently stripped. Anything else returns an error.
func ParseTimeRange(s string) (TimeRange, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return TimeRange{}, fmt.Errorf("empty time value")
	}

	// Range form first (must check before comparator forms — `..` could
	// otherwise be mistaken for two separate dates).
	if i := strings.Index(s, ".."); i >= 0 {
		left, right := s[:i], s[i+2:]
		var r TimeRange
		if left != "*" {
			d, err := parseDate(left)
			if err != nil {
				return TimeRange{}, err
			}
			r.From = &d
		}
		if right != "*" {
			d, err := parseDate(right)
			if err != nil {
				return TimeRange{}, err
			}
			r.To = &d
		}
		return r, nil
	}

	// Comparator form. `>=` and `<=` must come before `>` and `<` so the
	// two-byte prefixes win.
	switch {
	case strings.HasPrefix(s, ">="):
		d, err := parseDate(s[2:])
		if err != nil {
			return TimeRange{}, err
		}
		return TimeRange{From: &d, FromExclusive: false}, nil
	case strings.HasPrefix(s, "<="):
		d, err := parseDate(s[2:])
		if err != nil {
			return TimeRange{}, err
		}
		return TimeRange{To: &d, ToExclusive: false}, nil
	case strings.HasPrefix(s, ">"):
		d, err := parseDate(s[1:])
		if err != nil {
			return TimeRange{}, err
		}
		return TimeRange{From: &d, FromExclusive: true}, nil
	case strings.HasPrefix(s, "<"):
		d, err := parseDate(s[1:])
		if err != nil {
			return TimeRange{}, err
		}
		return TimeRange{To: &d, ToExclusive: true}, nil
	}

	// Bare date — exact match (inclusive on both sides).
	d, err := parseDate(s)
	if err != nil {
		return TimeRange{}, err
	}
	return TimeRange{From: &d, To: &d}, nil
}

// parseDate accepts YYYY-MM-DD or a full ISO 8601 timestamp; the time
// portion is silently stripped after the T.
func parseDate(s string) (time.Time, error) {
	if i := strings.Index(s, "T"); i >= 0 {
		s = s[:i]
	}
	return time.Parse("2006-01-02", s)
}
