package searchquery

import (
	"strconv"
	"strings"
	"testing"
)

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

func TestParseRange_Bare(t *testing.T) {
	r, err := ParseRange("5", parseFloat)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From == nil || *r.From != 5 || r.To == nil || *r.To != 5 {
		t.Errorf("got %+v, want From==To==5", r)
	}
	if r.FromExclusive || r.ToExclusive {
		t.Errorf("bare value should be inclusive on both sides")
	}
}

func TestParseRange_Greater(t *testing.T) {
	r, err := ParseRange(">5", parseFloat)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From == nil || *r.From != 5 || !r.FromExclusive {
		t.Errorf("got %+v, want From=5 exclusive", r)
	}
	if r.To != nil {
		t.Errorf("expected no upper bound; got %v", r.To)
	}
}

func TestParseRange_GreaterOrEqual(t *testing.T) {
	r, err := ParseRange(">=5", parseFloat)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From == nil || *r.From != 5 || r.FromExclusive {
		t.Errorf("got %+v, want From=5 inclusive", r)
	}
}

func TestParseRange_Less(t *testing.T) {
	r, err := ParseRange("<5", parseFloat)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.To == nil || *r.To != 5 || !r.ToExclusive {
		t.Errorf("got %+v, want To=5 exclusive", r)
	}
	if r.From != nil {
		t.Errorf("expected no lower bound; got %v", r.From)
	}
}

func TestParseRange_LessOrEqual(t *testing.T) {
	r, err := ParseRange("<=5", parseFloat)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.To == nil || *r.To != 5 || r.ToExclusive {
		t.Errorf("got %+v, want To=5 inclusive", r)
	}
}

func TestParseRange_DotDot(t *testing.T) {
	r, err := ParseRange("5..20", parseFloat)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From == nil || *r.From != 5 || r.To == nil || *r.To != 20 {
		t.Errorf("got %+v, want 5..20", r)
	}
	if r.FromExclusive || r.ToExclusive {
		t.Errorf("range should be inclusive on both sides")
	}
}

func TestParseRange_OpenUpper(t *testing.T) {
	r, err := ParseRange("5..*", parseFloat)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From == nil || *r.From != 5 || r.To != nil {
		t.Errorf("got %+v, want From=5 To=nil", r)
	}
}

func TestParseRange_OpenLower(t *testing.T) {
	r, err := ParseRange("*..20", parseFloat)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From != nil || r.To == nil || *r.To != 20 {
		t.Errorf("got %+v, want From=nil To=20", r)
	}
}

func TestParseRange_Unbounded(t *testing.T) {
	r, err := ParseRange("*..*", parseFloat)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From != nil || r.To != nil {
		t.Errorf("got %+v, want both bounds nil", r)
	}
}

func TestParseRange_Empty(t *testing.T) {
	if _, err := ParseRange("", parseFloat); err == nil {
		t.Errorf("empty input: err=nil, want error")
	}
}

func TestParseRange_Whitespace(t *testing.T) {
	r, err := ParseRange("  >5  ", parseFloat)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From == nil || *r.From != 5 || !r.FromExclusive {
		t.Errorf("leading/trailing whitespace not trimmed: %+v", r)
	}
}

func TestParseRange_BadValue(t *testing.T) {
	cases := []string{"notanumber", ">notanumber", ">=nope", "<x", "<=x", "1..nope", "nope..1"}
	for _, in := range cases {
		if _, err := ParseRange(in, parseFloat); err == nil {
			t.Errorf("%s: err=nil, want error", in)
		}
	}
}

// String-typed Range -- proves the generic plumbing works for non-numeric
// value types (e.g. SQL date binders that store dates as YYYY-MM-DD strings).
func TestParseRange_StringType(t *testing.T) {
	identity := func(s string) (string, error) { return s, nil }
	r, err := ParseRange(">=2025-01-01", identity)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.From == nil || *r.From != "2025-01-01" || r.FromExclusive {
		t.Errorf("got %+v, want From=2025-01-01 inclusive", r)
	}
}

// Caller errors propagate verbatim.
func TestParseRange_CallerErrorPropagates(t *testing.T) {
	sentinel := strconv.ErrSyntax
	fail := func(string) (int, error) { return 0, sentinel }
	if _, err := ParseRange(">5", fail); err == nil {
		t.Errorf("err=nil, want propagated parse error")
	}
}
