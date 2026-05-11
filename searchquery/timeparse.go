package searchquery

import (
	"strings"
	"time"
)

// TimeRange is a half-open or closed interval used by created/updated
// clauses. Either bound may be nil for an open end. The grammar follows
// GitHub's date filter syntax.
type TimeRange = Range[time.Time]

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
	return ParseRange(s, parseDate)
}

// parseDate accepts YYYY-MM-DD or a full ISO 8601 timestamp; the time
// portion is silently stripped after the T.
func parseDate(s string) (time.Time, error) {
	if i := strings.Index(s, "T"); i >= 0 {
		s = s[:i]
	}
	return time.Parse("2006-01-02", s)
}
