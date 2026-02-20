package filter

import (
	"testing"
	"time"
)

// --- TextFilter ---

func TestTextFilterSubstringMatch(t *testing.T) {
	f := NewTextFilter()
	f.SetText("abc")
	if !f.Matches("xabcx") {
		t.Error("should match substring")
	}
}

func TestTextFilterCaseInsensitive(t *testing.T) {
	f := NewTextFilter()
	f.SetText("ABC")
	if !f.Matches("abc") {
		t.Error("should match case-insensitively")
	}
}

func TestTextFilterNoMatch(t *testing.T) {
	f := NewTextFilter()
	f.SetText("xyz")
	if f.Matches("abc") {
		t.Error("should not match")
	}
}

func TestTextFilterEmptyInactive(t *testing.T) {
	f := NewTextFilter()
	if f.Active() {
		t.Error("empty filter should be inactive")
	}
}

func TestTextFilterClearResets(t *testing.T) {
	f := NewTextFilter()
	f.SetText("hello")
	if !f.Active() {
		t.Error("should be active after SetText")
	}
	f.Clear()
	if f.Active() {
		t.Error("should be inactive after Clear")
	}
}

func TestTextFilterRegexMode(t *testing.T) {
	f := NewTextFilter()
	f.SetRegex(true)
	f.SetText(`^\d+$`)
	if !f.Matches("123") {
		t.Error("regex should match digits")
	}
	if f.Matches("abc") {
		t.Error("regex should not match letters")
	}
}

func TestTextFilterInvalidRegex(t *testing.T) {
	f := NewTextFilter()
	f.SetRegex(true)
	f.SetText(`[invalid`)
	// Should not crash and should not match
	if f.Matches("anything") {
		t.Error("invalid regex should not match")
	}
}

func TestTextFilterNonStringValue(t *testing.T) {
	f := NewTextFilter()
	f.SetText("42")
	if !f.Matches(42) {
		t.Error("should convert non-string via Sprint")
	}
}

func TestTextFilterNilValue(t *testing.T) {
	f := NewTextFilter()
	f.SetText("nil")
	// fmt.Sprintf("%v", nil) = "<nil>"
	// "nil" is contained in "<nil>"
	if !f.Matches(nil) {
		t.Error("nil value contains 'nil'")
	}
}

// --- NumberFilter ---

func TestNumberFilterEquality(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("5")
	if !f.Matches(float64(5)) {
		t.Error("5 should match 5.0")
	}
	if f.Matches(float64(6)) {
		t.Error("5 should not match 6.0")
	}
}

func TestNumberFilterNotEqual(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("!=5")
	if !f.Matches(float64(6)) {
		t.Error("!=5 should match 6.0")
	}
	if f.Matches(float64(5)) {
		t.Error("!=5 should not match 5.0")
	}
}

func TestNumberFilterLessThan(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("<10")
	if !f.Matches(float64(5)) {
		t.Error("<10 should match 5")
	}
	if f.Matches(float64(10)) {
		t.Error("<10 should not match 10")
	}
	if f.Matches(float64(15)) {
		t.Error("<10 should not match 15")
	}
}

func TestNumberFilterGreaterThan(t *testing.T) {
	f := NewNumberFilter()
	f.SetText(">10")
	if !f.Matches(float64(15)) {
		t.Error(">10 should match 15")
	}
	if f.Matches(float64(10)) {
		t.Error(">10 should not match 10")
	}
}

func TestNumberFilterLessOrEqual(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("<=10")
	if !f.Matches(float64(10)) {
		t.Error("<=10 should match 10")
	}
	if !f.Matches(float64(5)) {
		t.Error("<=10 should match 5")
	}
	if f.Matches(float64(11)) {
		t.Error("<=10 should not match 11")
	}
}

func TestNumberFilterGreaterOrEqual(t *testing.T) {
	f := NewNumberFilter()
	f.SetText(">=10")
	if !f.Matches(float64(10)) {
		t.Error(">=10 should match 10")
	}
	if !f.Matches(float64(15)) {
		t.Error(">=10 should match 15")
	}
	if f.Matches(float64(9)) {
		t.Error(">=10 should not match 9")
	}
}

func TestNumberFilterRange(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("10..50")
	if !f.Matches(float64(25)) {
		t.Error("10..50 should match 25")
	}
	if f.Matches(float64(5)) {
		t.Error("10..50 should not match 5")
	}
	if f.Matches(float64(55)) {
		t.Error("10..50 should not match 55")
	}
}

func TestNumberFilterRangeBoundaries(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("10..50")
	if !f.Matches(float64(10)) {
		t.Error("range should include lower bound")
	}
	if !f.Matches(float64(50)) {
		t.Error("range should include upper bound")
	}
}

func TestNumberFilterIntegerValues(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("5")
	if !f.Matches(int(5)) {
		t.Error("should match int(5)")
	}
	if !f.Matches(int64(5)) {
		t.Error("should match int64(5)")
	}
}

func TestNumberFilterEmptyInactive(t *testing.T) {
	f := NewNumberFilter()
	if f.Active() {
		t.Error("empty should be inactive")
	}
}

func TestNumberFilterClear(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("42")
	f.Clear()
	if f.Active() {
		t.Error("should be inactive after Clear")
	}
}

func TestNumberFilterInvalidInput(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("notanumber")
	// op should remain empty, so Matches returns true for default case
	if !f.Matches(float64(42)) {
		t.Error("invalid input: op is empty, should match by default")
	}
}

// --- SetFilter ---

func TestSetFilterAllIncludedByDefault(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	if !f.Matches("a") || !f.Matches("b") || !f.Matches("c") {
		t.Error("all values should match by default")
	}
	if f.Active() {
		t.Error("all included = not active")
	}
}

func TestSetFilterExcludeOne(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("b")
	if !f.Matches("a") {
		t.Error("a should still match")
	}
	if f.Matches("b") {
		t.Error("b should be excluded")
	}
	if !f.Matches("c") {
		t.Error("c should still match")
	}
}

func TestSetFilterActiveWhenExcluded(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("b")
	if !f.Active() {
		t.Error("should be active when any excluded")
	}
}

func TestSetFilterIncludeAll(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("b")
	f.IncludeAll()
	if !f.Matches("b") {
		t.Error("IncludeAll should re-include b")
	}
	if f.Active() {
		t.Error("should be inactive after IncludeAll")
	}
}

func TestSetFilterUnknownValues(t *testing.T) {
	f := NewSetFilter("a", "b")
	f.Exclude("a")
	if !f.Matches("unknown") {
		t.Error("unknown values should pass through")
	}
}

func TestSetFilterClear(t *testing.T) {
	f := NewSetFilter("a", "b")
	f.Exclude("a")
	f.Clear()
	if f.Active() {
		t.Error("Clear should include all")
	}
}

func TestSetFilterSetValues(t *testing.T) {
	f := NewSetFilter("a", "b")
	f.SetValues([]string{"x", "y", "z"})
	if !f.Matches("x") || !f.Matches("y") || !f.Matches("z") {
		t.Error("new values should all match")
	}
}

func TestSetFilterNonStringValues(t *testing.T) {
	f := NewSetFilter("42", "true")
	if !f.Matches(42) {
		t.Error("int 42 should match via Sprint")
	}
	if !f.Matches(true) {
		t.Error("bool true should match via Sprint")
	}
}

// --- BoolFilter ---

func TestBoolFilterDefaultAny(t *testing.T) {
	f := NewBoolFilter()
	if f.Active() {
		t.Error("default any should be inactive")
	}
	if !f.Matches(true) || !f.Matches(false) {
		t.Error("any should match both")
	}
}

func TestBoolFilterToggleToTrue(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle() // any -> true
	if !f.Matches(true) {
		t.Error("true filter should match true")
	}
	if f.Matches(false) {
		t.Error("true filter should reject false")
	}
	if !f.Active() {
		t.Error("should be active")
	}
}

func TestBoolFilterToggleToFalse(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle() // any -> true
	f.Toggle() // true -> false
	if f.Matches(true) {
		t.Error("false filter should reject true")
	}
	if !f.Matches(false) {
		t.Error("false filter should match false")
	}
}

func TestBoolFilterToggleCycle(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle() // any -> true
	f.Toggle() // true -> false
	f.Toggle() // false -> any
	if f.Active() {
		t.Error("cycled back to any should be inactive")
	}
}

func TestBoolFilterClear(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle()
	f.Clear()
	if f.Active() {
		t.Error("Clear should reset to any")
	}
}

func TestBoolFilterNonBoolValue(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle() // true mode
	if f.Matches("notbool") {
		t.Error("non-bool value should not match in true mode")
	}
}

// --- TimeFilter ---

func TestTimeFilterSingleDate(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-06-15")
	// Should match times on that date
	tm := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	if !f.Matches(tm) {
		t.Error("should match time on the same day")
	}
	// Should not match next day
	tmNext := time.Date(2024, 6, 16, 12, 0, 0, 0, time.UTC)
	if f.Matches(tmNext) {
		t.Error("should not match next day")
	}
}

func TestTimeFilterDateRange(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-01-01..2024-06-30")
	mid := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !f.Matches(mid) {
		t.Error("should match date in range")
	}
	before := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	if f.Matches(before) {
		t.Error("should not match before range")
	}
}

func TestTimeFilterOpenStart(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("..2024-06-30")
	tm := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if !f.Matches(tm) {
		t.Error("open start: should match early date")
	}
	late := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if f.Matches(late) {
		t.Error("open start: should not match after end")
	}
}

func TestTimeFilterOpenEnd(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-01-01..")
	tm := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	if !f.Matches(tm) {
		t.Error("open end: should match future date")
	}
	early := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
	if f.Matches(early) {
		t.Error("open end: should not match before start")
	}
}

func TestTimeFilterMultipleFormats(t *testing.T) {
	formats := []string{
		"2024-06-15",
		"Jun 15 2024",
		"Jun 15, 2024",
		"June 15, 2024",
		"06/15/2024",
	}
	for _, fmt := range formats {
		f := NewTimeFilter()
		f.SetText(fmt)
		if !f.Active() {
			t.Errorf("format %q should parse and be active", fmt)
		}
	}
}

func TestTimeFilterTimeValue(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-06-15")
	tm := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	if !f.Matches(tm) {
		t.Error("should match time.Time value")
	}
}

func TestTimeFilterPointerToTime(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-06-15")
	tm := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	if !f.Matches(&tm) {
		t.Error("should match *time.Time value")
	}
}

func TestTimeFilterNilPointer(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-06-15")
	var tp *time.Time
	if f.Matches(tp) {
		t.Error("nil *time.Time should not match")
	}
}

func TestTimeFilterClear(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-06-15")
	f.Clear()
	if f.Active() {
		t.Error("should be inactive after Clear")
	}
}

func TestTimeFilterInvalidDate(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("notadate")
	if f.Active() {
		t.Error("unparseable date should be inactive")
	}
}
