package searchquery

import (
	"testing"
	"time"
)

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad date %q: %v", s, err)
	}
	return d
}

func TestParseTimeRange_Exact(t *testing.T) {
	r, err := ParseTimeRange("2025-01-01")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := mustDate(t, "2025-01-01")
	if r.From == nil || !r.From.Equal(want) || r.To == nil || !r.To.Equal(want) {
		t.Errorf("got %+v, want From==To==2025-01-01", r)
	}
}

func TestParseTimeRange_Greater(t *testing.T) {
	r, err := ParseTimeRange(">2025-01-01")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := mustDate(t, "2025-01-01")
	if r.From == nil || !r.From.Equal(want) || !r.FromExclusive {
		t.Errorf("got %+v, want From=2025-01-01 exclusive", r)
	}
	if r.To != nil {
		t.Errorf("expected no upper bound; got %v", r.To)
	}
}

func TestParseTimeRange_Less(t *testing.T) {
	r, err := ParseTimeRange("<2025-01-01")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From != nil {
		t.Errorf("expected no lower bound; got %v", r.From)
	}
	want := mustDate(t, "2025-01-01")
	if r.To == nil || !r.To.Equal(want) || !r.ToExclusive {
		t.Errorf("got %+v, want To=2025-01-01 exclusive", r)
	}
}

func TestParseTimeRange_GreaterOrEqual(t *testing.T) {
	r, err := ParseTimeRange(">=2025-01-01")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.FromExclusive {
		t.Errorf("expected inclusive lower bound")
	}
}

func TestParseTimeRange_DotDot(t *testing.T) {
	r, err := ParseTimeRange("2025-01-01..2025-04-01")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	from := mustDate(t, "2025-01-01")
	to := mustDate(t, "2025-04-01")
	if r.From == nil || !r.From.Equal(from) || r.To == nil || !r.To.Equal(to) {
		t.Errorf("got %+v, want %v..%v", r, from, to)
	}
	if r.FromExclusive || r.ToExclusive {
		t.Errorf("range should be inclusive on both sides; got %+v", r)
	}
}

func TestParseTimeRange_OpenUpper(t *testing.T) {
	r, err := ParseTimeRange("2025-01-01..*")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From == nil || r.To != nil {
		t.Errorf("got %+v, want From=2025-01-01 To=nil", r)
	}
}

func TestParseTimeRange_OpenLower(t *testing.T) {
	r, err := ParseTimeRange("*..2025-04-01")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From != nil || r.To == nil {
		t.Errorf("got %+v, want From=nil To=2025-04-01", r)
	}
}

func TestParseTimeRange_BadDate(t *testing.T) {
	if _, err := ParseTimeRange("2025-13-01"); err == nil {
		t.Error("expected error for bad date")
	}
}

func TestParseTimeRange_DateTimeStripped(t *testing.T) {
	r, err := ParseTimeRange("2025-01-01T12:00:00Z")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From == nil || !r.From.Equal(mustDate(t, "2025-01-01")) {
		t.Errorf("got %+v, want From=2025-01-01", r)
	}
}

func TestParseTimeRange_Empty(t *testing.T) {
	if _, err := ParseTimeRange(""); err == nil {
		t.Errorf("empty input: err=nil, want error")
	}
}

func TestParseTimeRange_InvalidComparators(t *testing.T) {
	cases := []string{">notadate", ">=notadate", "<notadate", "<=notadate"}
	for _, in := range cases {
		if _, err := ParseTimeRange(in); err == nil {
			t.Errorf("%s: err=nil, want error", in)
		}
	}
}

func TestParseTimeRange_InvalidRange(t *testing.T) {
	cases := []string{"notadate..2026-01-01", "2026-01-01..notadate"}
	for _, in := range cases {
		if _, err := ParseTimeRange(in); err == nil {
			t.Errorf("%s: err=nil, want error", in)
		}
	}
}
