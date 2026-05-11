package filter

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
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

// --- Helper for creating key messages ---

func runeKeyMsg(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func keyMsg(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// ==========================================================================
// View() and Update() tests for all filter types
// ==========================================================================

// --- TextFilter View/Update ---

func TestTextFilterViewEmptyReturnsEmpty(t *testing.T) {
	f := NewTextFilter()
	if f.View() != "" {
		t.Error("empty text filter should return empty View")
	}
}

func TestTextFilterViewWithTextReturnsText(t *testing.T) {
	f := NewTextFilter()
	f.SetText("hello")
	v := f.View()
	if v != "hello" {
		t.Errorf("expected 'hello', got %q", v)
	}
}

func TestTextFilterViewWhileEditingShowsEditor(t *testing.T) {
	f := NewTextFilter()
	f.SetText("abc")
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})
	v := f.View()
	if v == "" {
		t.Error("View while editing should not be empty")
	}
	// In editing mode, View uses RenderLine which includes ANSI escape sequences
	if !strings.Contains(v, "\x1b[7m") {
		t.Error("editing view should contain reverse-video ANSI escape for cursor")
	}
}

func TestTextFilterUpdateFocusAndBlur(t *testing.T) {
	f := NewTextFilter()

	// Focus
	result, cmd := f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})
	tf := result.(*TextFilter)
	if !tf.editing {
		t.Error("should be editing after FilterFocusMsg")
	}
	if tf.width != 40 {
		t.Errorf("width should be 40, got %d", tf.width)
	}
	if cmd != nil {
		t.Error("cmd should be nil")
	}

	// Blur
	result, cmd = tf.Update(FilterBlurMsg{})
	tf = result.(*TextFilter)
	if tf.editing {
		t.Error("should not be editing after FilterBlurMsg")
	}
	if cmd != nil {
		t.Error("cmd should be nil")
	}
}

func TestTextFilterUpdateTypingRunes(t *testing.T) {
	f := NewTextFilter()
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})

	f.Update(runeKeyMsg('h'))
	f.Update(runeKeyMsg('i'))

	if f.editor.Text() != "hi" {
		t.Errorf("expected 'hi', got %q", f.editor.Text())
	}
	if !f.Active() {
		t.Error("should be active after typing")
	}
}

func TestTextFilterUpdateBackspace(t *testing.T) {
	f := NewTextFilter()
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})
	f.Update(runeKeyMsg('a'))
	f.Update(runeKeyMsg('b'))
	f.Update(keyMsg(tea.KeyBackspace))

	if f.editor.Text() != "a" {
		t.Errorf("expected 'a' after backspace, got %q", f.editor.Text())
	}
}

func TestTextFilterUpdateKeyIgnoredWhenNotEditing(t *testing.T) {
	f := NewTextFilter()
	// Not focused, key should be ignored
	f.Update(runeKeyMsg('x'))
	if f.editor.Text() != "" {
		t.Error("key should be ignored when not editing")
	}
}

func TestTextFilterUpdateRegexRecompiled(t *testing.T) {
	f := NewTextFilter()
	f.SetRegex(true)
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})
	// Type a regex pattern character by character
	for _, r := range `\d+` {
		f.Update(runeKeyMsg(r))
	}
	if !f.Matches("123") {
		t.Error("regex should match after typing pattern")
	}
	if f.Matches("abc") {
		t.Error("regex should not match non-digits")
	}
}

// --- NumberFilter View/Update ---

func TestNumberFilterViewEmptyReturnsEmpty(t *testing.T) {
	f := NewNumberFilter()
	if f.View() != "" {
		t.Error("empty number filter should return empty View")
	}
}

func TestNumberFilterViewWithTextReturnsText(t *testing.T) {
	f := NewNumberFilter()
	f.SetText(">10")
	v := f.View()
	if v != ">10" {
		t.Errorf("expected '>10', got %q", v)
	}
}

func TestNumberFilterViewWhileEditingShowsEditor(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("42")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})
	v := f.View()
	if v == "" {
		t.Error("View while editing should not be empty")
	}
	if !strings.Contains(v, "\x1b[7m") {
		t.Error("editing view should contain reverse-video ANSI escape for cursor")
	}
}

func TestNumberFilterUpdateFocusAndBlur(t *testing.T) {
	f := NewNumberFilter()

	result, _ := f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})
	nf := result.(*NumberFilter)
	if !nf.editing {
		t.Error("should be editing after FilterFocusMsg")
	}
	if nf.width != 30 {
		t.Errorf("width should be 30, got %d", nf.width)
	}

	result, _ = nf.Update(FilterBlurMsg{})
	nf = result.(*NumberFilter)
	if nf.editing {
		t.Error("should not be editing after FilterBlurMsg")
	}
}

func TestNumberFilterUpdateTypingNumber(t *testing.T) {
	f := NewNumberFilter()
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	f.Update(runeKeyMsg('>'))
	f.Update(runeKeyMsg('5'))

	if f.editor.Text() != ">5" {
		t.Errorf("expected '>5', got %q", f.editor.Text())
	}
	if !f.Matches(float64(10)) {
		t.Error(">5 should match 10")
	}
	if f.Matches(float64(3)) {
		t.Error(">5 should not match 3")
	}
}

func TestNumberFilterUpdateBackspace(t *testing.T) {
	f := NewNumberFilter()
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})
	f.Update(runeKeyMsg('4'))
	f.Update(runeKeyMsg('2'))
	f.Update(keyMsg(tea.KeyBackspace))

	if f.editor.Text() != "4" {
		t.Errorf("expected '4' after backspace, got %q", f.editor.Text())
	}
}

func TestNumberFilterUpdateKeyIgnoredWhenNotEditing(t *testing.T) {
	f := NewNumberFilter()
	f.Update(runeKeyMsg('5'))
	if f.editor.Text() != "" {
		t.Error("key should be ignored when not editing")
	}
}

func TestNumberFilterUpdateParsesDuringTyping(t *testing.T) {
	f := NewNumberFilter()
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Type "10..50" character by character
	for _, r := range "10..50" {
		f.Update(runeKeyMsg(r))
	}

	if !f.rngOK || f.rng.From == nil || *f.rng.From != 10 || f.rng.To == nil || *f.rng.To != 50 {
		t.Errorf("should parse as range while typing; got rngOK=%v rng=%+v", f.rngOK, f.rng)
	}
	if !f.Matches(float64(25)) {
		t.Error("range 10..50 should match 25")
	}
}

// --- SetFilter View/Update ---

func TestSetFilterViewWhenNotEditingReturnsEmpty(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	if f.View() != "" {
		t.Error("View when not editing should return empty")
	}
}

func TestSetFilterViewWhenEditingShowsSearchAndList(t *testing.T) {
	f := NewSetFilter("alpha", "beta", "gamma")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})
	v := f.View()
	if v == "" {
		t.Error("View while editing should not be empty")
	}
	// Should contain checkbox list items
	if !strings.Contains(v, "[x]") {
		t.Error("View should contain checked checkboxes for included values")
	}
	if !strings.Contains(v, "alpha") {
		t.Error("View should contain 'alpha'")
	}
}

func TestSetFilterUpdateFocusAndBlur(t *testing.T) {
	f := NewSetFilter("a", "b", "c")

	result, _ := f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})
	sf := result.(*SetFilter)
	if !sf.editing {
		t.Error("should be editing after FilterFocusMsg")
	}
	if sf.width != 30 {
		t.Errorf("width should be 30, got %d", sf.width)
	}
	if sf.maxLines != 5 {
		t.Errorf("maxLines should be 5, got %d", sf.maxLines)
	}
	// All values should be in filtered list
	if len(sf.filtered) != 3 {
		t.Errorf("filtered should have 3 items, got %d", len(sf.filtered))
	}

	result, _ = sf.Update(FilterBlurMsg{})
	sf = result.(*SetFilter)
	if sf.editing {
		t.Error("should not be editing after FilterBlurMsg")
	}
}

func TestSetFilterUpdateSearchFilters(t *testing.T) {
	f := NewSetFilter("alpha", "beta", "gamma")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Type "al" to filter
	f.Update(runeKeyMsg('a'))
	f.Update(runeKeyMsg('l'))

	if len(f.filtered) != 1 {
		t.Errorf("expected 1 filtered result for 'al', got %d", len(f.filtered))
	}
	if f.filtered[0] != "alpha" {
		t.Errorf("expected 'alpha', got %q", f.filtered[0])
	}
}

func TestSetFilterUpdateTabSwitchesToListMode(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Tab should switch to list mode
	f.Update(keyMsg(tea.KeyTab))
	if !f.inList {
		t.Error("Tab should switch to list mode")
	}
}

func TestSetFilterUpdateDownSwitchesToListMode(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	f.Update(keyMsg(tea.KeyDown))
	if !f.inList {
		t.Error("Down should switch to list mode")
	}
}

func TestSetFilterUpdateListNavigation(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode
	f.Update(keyMsg(tea.KeyDown))

	// Navigate down
	f.Update(keyMsg(tea.KeyDown))
	if f.selectedIdx != 1 {
		t.Errorf("expected selectedIdx 1, got %d", f.selectedIdx)
	}

	// Navigate up
	f.Update(keyMsg(tea.KeyUp))
	if f.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", f.selectedIdx)
	}

	// Don't go above 0
	f.Update(keyMsg(tea.KeyUp))
	if f.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0 at top, got %d", f.selectedIdx)
	}
}

func TestSetFilterUpdateListToggle(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode
	f.Update(keyMsg(tea.KeyDown))

	// Toggle first item with space
	f.Update(keyMsg(tea.KeySpace))
	if f.values["a"] {
		t.Error("'a' should be excluded after space toggle")
	}
	if !f.Active() {
		t.Error("filter should be active after excluding a value")
	}

	// Toggle again to re-include
	f.Update(keyMsg(tea.KeySpace))
	if !f.values["a"] {
		t.Error("'a' should be included after second space toggle")
	}
}

func TestSetFilterUpdateListCtrlSpaceSelectsOnly(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode and move to "b"
	f.Update(keyMsg(tea.KeyDown))
	f.Update(keyMsg(tea.KeyDown))

	// Press Ctrl+Space to select only "b"
	f.Update(tea.KeyPressMsg{Code: tea.KeySpace, Mod: tea.ModCtrl})

	if f.values["a"] {
		t.Error("'a' should be excluded after Ctrl+Space select-only")
	}
	if !f.values["b"] {
		t.Error("'b' should be included after Ctrl+Space select-only")
	}
	if f.values["c"] {
		t.Error("'c' should be excluded after Ctrl+Space select-only")
	}
	if !f.Active() {
		t.Error("filter should be active after select-only")
	}
}

func TestSetFilterUpdateListTabReturnsToSearch(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode
	f.Update(keyMsg(tea.KeyTab))
	if !f.inList {
		t.Error("should be in list mode")
	}

	// Tab back to search mode
	f.Update(keyMsg(tea.KeyTab))
	if f.inList {
		t.Error("Tab in list mode should return to search mode")
	}
}

func TestSetFilterUpdateListTypingSwitchesToSearch(t *testing.T) {
	f := NewSetFilter("alpha", "beta", "gamma")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode
	f.Update(keyMsg(tea.KeyDown))

	// Typing a rune while in list mode should switch back to search
	f.Update(runeKeyMsg('b'))
	if f.inList {
		t.Error("typing rune in list mode should switch to search mode")
	}
	if f.search.Text() != "b" {
		t.Errorf("expected search text 'b', got %q", f.search.Text())
	}
	// Should filter the list
	if len(f.filtered) != 1 || f.filtered[0] != "beta" {
		t.Errorf("expected filtered to contain only 'beta', got %v", f.filtered)
	}
}

func TestSetFilterUpdateKeyIgnoredWhenNotEditing(t *testing.T) {
	f := NewSetFilter("a", "b")
	f.Update(runeKeyMsg('x'))
	// search text should remain empty
	if f.search.Text() != "" {
		t.Error("key should be ignored when not editing")
	}
}

func TestSetFilterViewHighlightsSelectedInList(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode
	f.Update(keyMsg(tea.KeyDown))

	v := f.View()
	// Selected item should have reverse-video ANSI escape
	if !strings.Contains(v, "\x1b[7m") {
		t.Error("selected item in list should have reverse-video highlight")
	}
}

// --- BoolFilter View/Update ---

func TestBoolFilterViewDefaultAny(t *testing.T) {
	f := NewBoolFilter()
	v := f.View()
	if v != "" {
		t.Errorf("default any state should return empty View, got %q", v)
	}
}

func TestBoolFilterViewTrueState(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle() // any -> true
	v := f.View()
	if v != "true" {
		t.Errorf("expected 'true', got %q", v)
	}
}

func TestBoolFilterViewFalseState(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle() // any -> true
	f.Toggle() // true -> false
	v := f.View()
	if v != "false" {
		t.Errorf("expected 'false', got %q", v)
	}
}

func TestBoolFilterViewWhileEditingShowsRadioButtons(t *testing.T) {
	f := NewBoolFilter()
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})
	v := f.View()
	if v == "" {
		t.Error("editing view should not be empty")
	}
	if !strings.Contains(v, "Any") || !strings.Contains(v, "True") || !strings.Contains(v, "False") {
		t.Errorf("editing view should contain radio options, got %q", v)
	}
	// The "Any" radio should be selected (filled dot)
	if !strings.Contains(v, "(\u25cf)") {
		t.Error("editing view should contain a filled radio button")
	}
}

func TestBoolFilterUpdateFocusAndBlur(t *testing.T) {
	f := NewBoolFilter()

	result, _ := f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})
	bf := result.(*BoolFilter)
	if !bf.editing {
		t.Error("should be editing after FilterFocusMsg")
	}
	if bf.width != 40 {
		t.Errorf("width should be 40, got %d", bf.width)
	}

	result, _ = bf.Update(FilterBlurMsg{})
	bf = result.(*BoolFilter)
	if bf.editing {
		t.Error("should not be editing after FilterBlurMsg")
	}
}

func TestBoolFilterUpdateSpaceToggles(t *testing.T) {
	f := NewBoolFilter()
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})

	// Space: any -> true
	f.Update(keyMsg(tea.KeySpace))
	if f.state != 1 {
		t.Errorf("expected state 1 (true), got %d", f.state)
	}

	// Space: true -> false
	f.Update(keyMsg(tea.KeySpace))
	if f.state != 2 {
		t.Errorf("expected state 2 (false), got %d", f.state)
	}

	// Space: false -> any
	f.Update(keyMsg(tea.KeySpace))
	if f.state != 0 {
		t.Errorf("expected state 0 (any), got %d", f.state)
	}
}

func TestBoolFilterUpdateRightTogglesForward(t *testing.T) {
	f := NewBoolFilter()
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})

	// Right: any -> true
	f.Update(keyMsg(tea.KeyRight))
	if f.state != 1 {
		t.Errorf("expected state 1 after Right, got %d", f.state)
	}
}

func TestBoolFilterUpdateLeftTogglesBackward(t *testing.T) {
	f := NewBoolFilter()
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})

	// Left from any (0) should go to false (2), since (0+2)%3 = 2
	f.Update(keyMsg(tea.KeyLeft))
	if f.state != 2 {
		t.Errorf("expected state 2 after Left from any, got %d", f.state)
	}
}

func TestBoolFilterUpdateKeyIgnoredWhenNotEditing(t *testing.T) {
	f := NewBoolFilter()
	f.Update(keyMsg(tea.KeySpace))
	if f.state != 0 {
		t.Error("key should be ignored when not editing")
	}
}

func TestBoolFilterViewEditingReflectsState(t *testing.T) {
	f := NewBoolFilter()
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})

	// Toggle to true
	f.Update(keyMsg(tea.KeySpace))
	v := f.View()
	// The "True" radio should now be the one with a filled dot.
	// Check that the view changed by ensuring "True" section has the bullet.
	// We know the format: anyR + " Any  " + trueR + " True  " + falseR + " False"
	// When state=1, trueR = "(\u25cf)"
	if !strings.Contains(v, "(\u25cf) True") {
		t.Errorf("True radio should be selected, got %q", v)
	}
}

// --- TimeFilter View/Update ---

func TestTimeFilterViewEmptyReturnsEmpty(t *testing.T) {
	f := NewTimeFilter()
	if f.View() != "" {
		t.Error("empty time filter should return empty View")
	}
}

func TestTimeFilterViewWithTextReturnsText(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-06-15")
	v := f.View()
	if v != "2024-06-15" {
		t.Errorf("expected '2024-06-15', got %q", v)
	}
}

func TestTimeFilterViewWhileEditingShowsEditor(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-01-01")
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})
	v := f.View()
	if v == "" {
		t.Error("View while editing should not be empty")
	}
	if !strings.Contains(v, "\x1b[7m") {
		t.Error("editing view should contain reverse-video ANSI escape for cursor")
	}
}

func TestTimeFilterUpdateFocusAndBlur(t *testing.T) {
	f := NewTimeFilter()

	result, _ := f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})
	tf := result.(*TimeFilter)
	if !tf.editing {
		t.Error("should be editing after FilterFocusMsg")
	}
	if tf.width != 40 {
		t.Errorf("width should be 40, got %d", tf.width)
	}

	result, _ = tf.Update(FilterBlurMsg{})
	tf = result.(*TimeFilter)
	if tf.editing {
		t.Error("should not be editing after FilterBlurMsg")
	}
}

func TestTimeFilterUpdateTypingDate(t *testing.T) {
	f := NewTimeFilter()
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})

	for _, r := range "2024-06-15" {
		f.Update(runeKeyMsg(r))
	}

	if f.editor.Text() != "2024-06-15" {
		t.Errorf("expected '2024-06-15', got %q", f.editor.Text())
	}
	if !f.Active() {
		t.Error("should be active after typing a valid date")
	}
	tm := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	if !f.Matches(tm) {
		t.Error("should match a time on the typed date")
	}
}

func TestTimeFilterUpdateBackspace(t *testing.T) {
	f := NewTimeFilter()
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})
	for _, r := range "2024" {
		f.Update(runeKeyMsg(r))
	}
	f.Update(keyMsg(tea.KeyBackspace))

	if f.editor.Text() != "202" {
		t.Errorf("expected '202' after backspace, got %q", f.editor.Text())
	}
}

func TestTimeFilterUpdateKeyIgnoredWhenNotEditing(t *testing.T) {
	f := NewTimeFilter()
	f.Update(runeKeyMsg('2'))
	if f.editor.Text() != "" {
		t.Error("key should be ignored when not editing")
	}
}

func TestTimeFilterUpdateTypingRange(t *testing.T) {
	f := NewTimeFilter()
	f.Update(FilterFocusMsg{Width: 50, MaxLines: 5})

	for _, r := range "2024-01-01..2024-12-31" {
		f.Update(runeKeyMsg(r))
	}

	if !f.Active() {
		t.Error("should be active after typing a valid date range")
	}
	mid := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	if !f.Matches(mid) {
		t.Error("should match a time within the typed range")
	}
	before := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	if f.Matches(before) {
		t.Error("should not match a time before the range")
	}
}

// ==========================================================================
// Additional coverage gap tests
// ==========================================================================

// 1. SetFilter.Include - call Include() and verify the value is included
func TestSetFilterIncludeReIncludesExcludedValue(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("b")
	if f.Matches("b") {
		t.Error("b should be excluded after Exclude")
	}
	f.Include("b")
	if !f.Matches("b") {
		t.Error("b should be included after Include")
	}
	if f.Active() {
		t.Error("filter should be inactive when all values are included")
	}
}

// 2. SetFilter.View not editing - create a SetFilter with excluded values, don't focus, call View()
func TestSetFilterViewNotEditingReturnsEmptyEvenWhenActive(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("a") // Make the filter active
	if !f.Active() {
		t.Error("filter should be active after excluding")
	}
	v := f.View()
	if v != "" {
		t.Errorf("View when not editing should return empty string even when active, got %q", v)
	}
}

// 3. SetFilter.Update non-editing KeyMsg - send a KeyMsg without focusing first
func TestSetFilterUpdateNonEditingKeyMsgIgnored(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	// Send a KeyMsg (Tab) without focusing first
	result, cmd := f.Update(keyMsg(tea.KeyTab))
	sf := result.(*SetFilter)
	if sf.editing {
		t.Error("should not be editing")
	}
	if sf.inList {
		t.Error("should not enter list mode when not editing")
	}
	if cmd != nil {
		t.Error("cmd should be nil")
	}
}

// 4. SetFilter.ensureVisible scroll up - navigate upward past the visible area
func TestSetFilterEnsureVisibleScrollUp(t *testing.T) {
	// Create a set with many values and a small maxLines to force scrolling
	values := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	f := NewSetFilter(values...)
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 3}) // 3 lines total, 2 for list (1 for search)

	// Enter list mode and navigate to the bottom
	f.Update(keyMsg(tea.KeyDown))
	// Navigate down several times to push scrollTop forward
	for i := 0; i < 8; i++ {
		f.Update(keyMsg(tea.KeyDown))
	}

	// Now selectedIdx should be at 8, and scrollTop should have moved
	if f.selectedIdx != 8 {
		t.Errorf("expected selectedIdx 8, got %d", f.selectedIdx)
	}
	if f.scrollTop == 0 {
		t.Error("scrollTop should have moved from 0")
	}

	savedScrollTop := f.scrollTop

	// Now navigate UP past the visible area to trigger selectedIdx < scrollTop path
	// We need to go up enough so that selectedIdx < scrollTop
	f.Update(keyMsg(tea.KeyUp))
	f.Update(keyMsg(tea.KeyUp))

	// selectedIdx should now be less than the saved scrollTop,
	// which should have triggered the ensureVisible scroll-up path
	if f.selectedIdx != 6 {
		t.Errorf("expected selectedIdx 6, got %d", f.selectedIdx)
	}
	// scrollTop should now be <= selectedIdx
	if f.scrollTop > f.selectedIdx {
		t.Errorf("scrollTop (%d) should be <= selectedIdx (%d)", f.scrollTop, f.selectedIdx)
	}
	// Verify it actually scrolled up from where it was
	if f.scrollTop >= savedScrollTop {
		t.Errorf("scrollTop (%d) should have decreased from %d", f.scrollTop, savedScrollTop)
	}
}

// 5. NumberFilter.Matches with float32
func TestNumberFilterMatchesFloat32(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("5")
	if !f.Matches(float32(5.0)) {
		t.Error("should match float32(5.0)")
	}
	if f.Matches(float32(6.0)) {
		t.Error("should not match float32(6.0)")
	}
}

// 6. NumberFilter.Matches with unsupported type (string)
func TestNumberFilterMatchesUnsupportedType(t *testing.T) {
	f := NewNumberFilter()
	f.SetText("5")
	if f.Matches("five") {
		t.Error("string value should not match (unsupported type returns false)")
	}
	if f.Matches("5") {
		t.Error("string '5' should not match (unsupported type returns false)")
	}
}

// 7. TextFilter.SetRegex empty text - call SetRegex(true) when text is ""
func TestTextFilterSetRegexEmptyText(t *testing.T) {
	f := NewTextFilter()
	// Text is empty by default
	f.SetRegex(true)
	if f.compiled != nil {
		t.Error("compiled should be nil when text is empty and regex is enabled")
	}
	if f.Active() {
		t.Error("should not be active with empty text")
	}
	// Should still match anything since filter is inactive (empty text)
	if !f.Matches("anything") {
		t.Error("empty regex filter should match via substring (empty substring matches all)")
	}
}

// 8. BoolFilter.View state=2 non-editing - toggle to false state, check View outside editing
func TestBoolFilterViewFalseStateNotEditing(t *testing.T) {
	f := NewBoolFilter()
	f.Toggle() // any -> true (state=1)
	f.Toggle() // true -> false (state=2)
	// Not editing - should return "false"
	v := f.View()
	if v != "false" {
		t.Errorf("expected 'false' for state=2 non-editing, got %q", v)
	}
}

// 9. TimeFilter.Matches with *time.Time (non-nil pointer)
func TestTimeFilterMatchesPointerToTime(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-06-15")
	tm := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	if !f.Matches(&tm) {
		t.Error("should match *time.Time pointing to a matching date")
	}
	tmOther := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	if f.Matches(&tmOther) {
		t.Error("should not match *time.Time pointing to a non-matching date")
	}
}

// 10. TimeFilter.Matches with nil *time.Time
func TestTimeFilterMatchesNilPointerToTime(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-06-15")
	var tp *time.Time
	if f.Matches(tp) {
		t.Error("nil *time.Time should not match")
	}
}

// Additional: TimeFilter.Matches with non-time type returns false
func TestTimeFilterMatchesNonTimeType(t *testing.T) {
	f := NewTimeFilter()
	f.SetText("2024-06-15")
	if f.Matches("2024-06-15") {
		t.Error("string value should not match (not a time.Time)")
	}
	if f.Matches(42) {
		t.Error("int value should not match (not a time.Time)")
	}
}

// ==========================================================================
// Final coverage gap tests
// ==========================================================================

//  1. TextFilter.SetRegex with text already set - covers line 44 in builtin.go
//     (the `f.compiled, _ = regexp.Compile(f.editor.Text())` branch inside SetRegex)
func TestTextFilterSetRegexWithTextAlreadySet(t *testing.T) {
	f := NewTextFilter()
	// Set text FIRST, then enable regex so SetRegex sees non-empty text
	f.SetText(`^\d+$`)
	f.SetRegex(true)
	if f.compiled == nil {
		t.Error("compiled should be non-nil after SetRegex(true) with non-empty text")
	}
	if !f.Matches("123") {
		t.Error("regex should match digits after SetRegex(true) on pre-set text")
	}
	if f.Matches("abc") {
		t.Error("regex should not match letters")
	}
}

//  2. SetFilter.View with maxLines=1 - covers the `listLines < 1` branch in View()
//     When maxLines=1, listLines = 1-1 = 0, which triggers listLines = 1.
func TestSetFilterViewWithMaxLinesOne(t *testing.T) {
	f := NewSetFilter("alpha", "beta", "gamma")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 1})
	v := f.View()
	if v == "" {
		t.Error("View should not be empty even with maxLines=1")
	}
	// Should contain the search line and at least one list item
	if !strings.Contains(v, "alpha") {
		t.Error("View should contain at least one list item when maxLines=1")
	}
}

//  3. SetFilter.Update with an unknown message type - covers the final `return f, nil`
//     after the type switch (reached only for message types other than Focus/Blur/KeyMsg).
type unknownMsg struct{}

func TestSetFilterUpdateUnknownMsgType(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	result, cmd := f.Update(unknownMsg{})
	sf := result.(*SetFilter)
	if sf.editing {
		t.Error("should not be editing")
	}
	if cmd != nil {
		t.Error("cmd should be nil for unknown message type")
	}
}

//  4. SetFilter.ensureVisible with maxLines=1 - covers the `listLines < 1` branch
//     in ensureVisible(). When maxLines=1, listLines=0, which triggers listLines=1.
func TestSetFilterEnsureVisibleWithMaxLinesOne(t *testing.T) {
	values := []string{"a", "b", "c", "d", "e"}
	f := NewSetFilter(values...)
	// maxLines=1 means listLines = 1-1 = 0 < 1, so listLines is set to 1
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 1})

	// Enter list mode and navigate down to trigger ensureVisible
	f.Update(keyMsg(tea.KeyDown))
	f.Update(keyMsg(tea.KeyDown))

	if f.selectedIdx != 1 {
		t.Errorf("expected selectedIdx 1, got %d", f.selectedIdx)
	}
	// scrollTop should have adjusted so selectedIdx is visible
	if f.scrollTop > f.selectedIdx {
		t.Errorf("scrollTop (%d) should be <= selectedIdx (%d)", f.scrollTop, f.selectedIdx)
	}
}

//  5. BoolFilter.View state=2 while editing - covers `falseR = "(\u25cf)"` in View()
//     Toggle twice to reach state=2 (false) while editing.
func TestBoolFilterViewEditingFalseState(t *testing.T) {
	f := NewBoolFilter()
	f.Update(FilterFocusMsg{Width: 40, MaxLines: 5})

	// Toggle to true (state=1)
	f.Update(keyMsg(tea.KeySpace))
	// Toggle to false (state=2)
	f.Update(keyMsg(tea.KeySpace))
	if f.state != 2 {
		t.Fatalf("expected state 2, got %d", f.state)
	}

	v := f.View()
	// The "False" radio should now be the one with a filled dot.
	if !strings.Contains(v, "(\u25cf) False") {
		t.Errorf("False radio should be selected in editing view, got %q", v)
	}
	// The "Any" and "True" radios should be empty
	if !strings.Contains(v, "( ) Any") {
		t.Errorf("Any radio should be empty in editing view, got %q", v)
	}
	if !strings.Contains(v, "( ) True") {
		t.Errorf("True radio should be empty in editing view, got %q", v)
	}
}

// --- SetFilter excludedCount tracking ---

func TestSetFilterExcludedCount_NewFilterStartsAtZero(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	if f.excludedCount != 0 {
		t.Errorf("new filter excludedCount = %d, want 0", f.excludedCount)
	}
	if f.Active() {
		t.Error("new filter should not be active")
	}
}

func TestSetFilterExcludedCount_EmptyFilter(t *testing.T) {
	f := NewSetFilter()
	if f.excludedCount != 0 {
		t.Errorf("empty filter excludedCount = %d, want 0", f.excludedCount)
	}
	if f.Active() {
		t.Error("empty filter should not be active")
	}
}

func TestSetFilterExcludedCount_SingleValue(t *testing.T) {
	f := NewSetFilter("only")
	f.Exclude("only")
	if f.excludedCount != 1 {
		t.Errorf("excludedCount = %d, want 1", f.excludedCount)
	}
	if !f.Active() {
		t.Error("should be active with one excluded")
	}
	f.Include("only")
	if f.excludedCount != 0 {
		t.Errorf("excludedCount = %d, want 0", f.excludedCount)
	}
	if f.Active() {
		t.Error("should not be active after re-including")
	}
}

func TestSetFilterExcludedCount_ExcludeAll(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("a")
	f.Exclude("b")
	f.Exclude("c")
	if f.excludedCount != 3 {
		t.Errorf("excludedCount = %d, want 3", f.excludedCount)
	}
	if !f.Active() {
		t.Error("should be active when all excluded")
	}
}

func TestSetFilterExcludedCount_ExcludeIdempotent(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("a")
	f.Exclude("a") // already excluded
	f.Exclude("a") // still excluded
	if f.excludedCount != 1 {
		t.Errorf("repeated Exclude should not double-count: excludedCount = %d, want 1", f.excludedCount)
	}
}

func TestSetFilterExcludedCount_IncludeIdempotent(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Include("a") // already included
	f.Include("a") // still included
	if f.excludedCount != 0 {
		t.Errorf("repeated Include on included value should keep count at 0: excludedCount = %d, want 0", f.excludedCount)
	}
}

func TestSetFilterExcludedCount_IncludeAlreadyIncluded(t *testing.T) {
	f := NewSetFilter("a", "b")
	f.Exclude("a")
	// Include "b" which is already included
	f.Include("b")
	if f.excludedCount != 1 {
		t.Errorf("excludedCount = %d, want 1 (only 'a' excluded)", f.excludedCount)
	}
	if !f.Active() {
		t.Error("should still be active")
	}
}

func TestSetFilterExcludedCount_ExcludeAlreadyExcluded(t *testing.T) {
	f := NewSetFilter("a", "b")
	f.Exclude("a")
	f.Exclude("a") // already excluded
	if f.excludedCount != 1 {
		t.Errorf("excludedCount = %d, want 1", f.excludedCount)
	}
}

func TestSetFilterExcludedCount_IncludeAll(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("a")
	f.Exclude("b")
	f.Exclude("c")
	if f.excludedCount != 3 {
		t.Errorf("before IncludeAll: excludedCount = %d, want 3", f.excludedCount)
	}
	f.IncludeAll()
	if f.excludedCount != 0 {
		t.Errorf("after IncludeAll: excludedCount = %d, want 0", f.excludedCount)
	}
	if f.Active() {
		t.Error("should not be active after IncludeAll")
	}
}

func TestSetFilterExcludedCount_IncludeAllWhenAlreadyAllIncluded(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.IncludeAll()
	if f.excludedCount != 0 {
		t.Errorf("excludedCount = %d, want 0", f.excludedCount)
	}
}

func TestSetFilterExcludedCount_Clear(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("a")
	f.Exclude("b")
	f.Clear()
	if f.excludedCount != 0 {
		t.Errorf("after Clear: excludedCount = %d, want 0", f.excludedCount)
	}
	if f.Active() {
		t.Error("should not be active after Clear")
	}
}

func TestSetFilterExcludedCount_SetValuesResetsCount(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("a")
	f.Exclude("b")
	if f.excludedCount != 2 {
		t.Errorf("before SetValues: excludedCount = %d, want 2", f.excludedCount)
	}
	f.SetValues([]string{"x", "y", "z"})
	if f.excludedCount != 0 {
		t.Errorf("after SetValues: excludedCount = %d, want 0", f.excludedCount)
	}
	if f.Active() {
		t.Error("should not be active after SetValues")
	}
}

func TestSetFilterExcludedCount_SetValuesThenExclude(t *testing.T) {
	f := NewSetFilter("a", "b")
	f.SetValues([]string{"x", "y", "z"})
	f.Exclude("x")
	if f.excludedCount != 1 {
		t.Errorf("excludedCount = %d, want 1", f.excludedCount)
	}
	if !f.Active() {
		t.Error("should be active")
	}
}

func TestSetFilterExcludedCount_ExcludeThenIncludeSequence(t *testing.T) {
	f := NewSetFilter("a", "b", "c", "d")

	f.Exclude("a")
	if f.excludedCount != 1 {
		t.Fatalf("step 1: excludedCount = %d, want 1", f.excludedCount)
	}
	f.Exclude("b")
	if f.excludedCount != 2 {
		t.Fatalf("step 2: excludedCount = %d, want 2", f.excludedCount)
	}
	f.Include("a")
	if f.excludedCount != 1 {
		t.Fatalf("step 3: excludedCount = %d, want 1", f.excludedCount)
	}
	f.Exclude("c")
	if f.excludedCount != 2 {
		t.Fatalf("step 4: excludedCount = %d, want 2", f.excludedCount)
	}
	f.Include("b")
	if f.excludedCount != 1 {
		t.Fatalf("step 5: excludedCount = %d, want 1", f.excludedCount)
	}
	f.Include("c")
	if f.excludedCount != 0 {
		t.Fatalf("step 6: excludedCount = %d, want 0", f.excludedCount)
	}
	if f.Active() {
		t.Error("should not be active when all included")
	}
}

func TestSetFilterExcludedCount_ActiveConsistentWithMatches(t *testing.T) {
	// Verify Active() and Matches() agree across many operations.
	f := NewSetFilter("a", "b", "c")
	assertConsistent := func(label string) {
		t.Helper()
		anyExcluded := !f.Matches("a") || !f.Matches("b") || !f.Matches("c")
		if f.Active() != anyExcluded {
			t.Errorf("%s: Active()=%v but anyExcluded=%v (excludedCount=%d)",
				label, f.Active(), anyExcluded, f.excludedCount)
		}
	}

	assertConsistent("initial")
	f.Exclude("a")
	assertConsistent("after exclude a")
	f.Exclude("b")
	assertConsistent("after exclude b")
	f.Include("a")
	assertConsistent("after include a")
	f.IncludeAll()
	assertConsistent("after IncludeAll")
	f.Exclude("c")
	assertConsistent("after exclude c")
	f.Clear()
	assertConsistent("after Clear")
}

func TestSetFilterExcludedCount_SpaceToggleInUpdate(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode
	f.Update(keyMsg(tea.KeyDown))

	// Toggle "a" off
	f.Update(keyMsg(tea.KeySpace))
	if f.excludedCount != 1 {
		t.Errorf("after toggling off: excludedCount = %d, want 1", f.excludedCount)
	}
	if !f.Active() {
		t.Error("should be active after toggling off")
	}

	// Toggle "a" back on
	f.Update(keyMsg(tea.KeySpace))
	if f.excludedCount != 0 {
		t.Errorf("after toggling back on: excludedCount = %d, want 0", f.excludedCount)
	}
	if f.Active() {
		t.Error("should not be active after toggling back on")
	}
}

func TestSetFilterExcludedCount_SpaceToggleMultipleItems(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode, toggle "a" off
	f.Update(keyMsg(tea.KeyDown))
	f.Update(keyMsg(tea.KeySpace))

	// Move to "b", toggle off
	f.Update(keyMsg(tea.KeyDown))
	f.Update(keyMsg(tea.KeySpace))

	if f.excludedCount != 2 {
		t.Errorf("after toggling two off: excludedCount = %d, want 2", f.excludedCount)
	}

	// Toggle "b" back on
	f.Update(keyMsg(tea.KeySpace))
	if f.excludedCount != 1 {
		t.Errorf("after toggling one back on: excludedCount = %d, want 1", f.excludedCount)
	}
}

func TestSetFilterExcludedCount_CtrlSpaceInUpdate(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode, move to "b"
	f.Update(keyMsg(tea.KeyDown))
	f.Update(keyMsg(tea.KeyDown))

	// Ctrl+Space selects only "b"
	f.Update(tea.KeyPressMsg{Code: tea.KeySpace, Mod: tea.ModCtrl})

	if f.excludedCount != 2 {
		t.Errorf("after Ctrl+Space: excludedCount = %d, want 2", f.excludedCount)
	}
	if !f.Active() {
		t.Error("should be active after Ctrl+Space")
	}
}

func TestSetFilterExcludedCount_CtrlSpaceThenToggle(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode, Ctrl+Space on "a" (first item)
	f.Update(keyMsg(tea.KeyDown))
	f.Update(tea.KeyPressMsg{Code: tea.KeySpace, Mod: tea.ModCtrl})
	if f.excludedCount != 2 {
		t.Fatalf("after Ctrl+Space: excludedCount = %d, want 2", f.excludedCount)
	}

	// Move to "b", toggle it on
	f.Update(keyMsg(tea.KeyDown))
	f.Update(keyMsg(tea.KeySpace))
	if f.excludedCount != 1 {
		t.Errorf("after toggling 'b' on: excludedCount = %d, want 1", f.excludedCount)
	}

	// Move to "c", toggle it on
	f.Update(keyMsg(tea.KeyDown))
	f.Update(keyMsg(tea.KeySpace))
	if f.excludedCount != 0 {
		t.Errorf("after toggling 'c' on: excludedCount = %d, want 0", f.excludedCount)
	}
	if f.Active() {
		t.Error("should not be active when all included")
	}
}

func TestSetFilterExcludedCount_CtrlSpaceOnSingleValueFilter(t *testing.T) {
	f := NewSetFilter("only")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode, Ctrl+Space on "only"
	f.Update(keyMsg(tea.KeyDown))
	f.Update(tea.KeyPressMsg{Code: tea.KeySpace, Mod: tea.ModCtrl})

	// With only one value, selecting only it means excludedCount = 0
	if f.excludedCount != 0 {
		t.Errorf("Ctrl+Space on single-value filter: excludedCount = %d, want 0", f.excludedCount)
	}
	if f.Active() {
		t.Error("should not be active when only value is selected")
	}
}

func TestSetFilterExcludedCount_CtrlSpaceRepeated(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Update(FilterFocusMsg{Width: 30, MaxLines: 5})

	// Enter list mode
	f.Update(keyMsg(tea.KeyDown))

	// Ctrl+Space on "a"
	f.Update(tea.KeyPressMsg{Code: tea.KeySpace, Mod: tea.ModCtrl})
	if f.excludedCount != 2 {
		t.Fatalf("first Ctrl+Space: excludedCount = %d, want 2", f.excludedCount)
	}

	// Move to "b", Ctrl+Space again
	f.Update(keyMsg(tea.KeyDown))
	f.Update(tea.KeyPressMsg{Code: tea.KeySpace, Mod: tea.ModCtrl})
	if f.excludedCount != 2 {
		t.Errorf("second Ctrl+Space: excludedCount = %d, want 2", f.excludedCount)
	}

	// Move to "c", Ctrl+Space again
	f.Update(keyMsg(tea.KeyDown))
	f.Update(tea.KeyPressMsg{Code: tea.KeySpace, Mod: tea.ModCtrl})
	if f.excludedCount != 2 {
		t.Errorf("third Ctrl+Space: excludedCount = %d, want 2", f.excludedCount)
	}
}

func TestSetFilterExcludedCount_ExcludeThenSetValues(t *testing.T) {
	f := NewSetFilter("a", "b", "c")
	f.Exclude("a")
	f.Exclude("b")
	f.Exclude("c")
	if f.excludedCount != 3 {
		t.Fatalf("before SetValues: excludedCount = %d, want 3", f.excludedCount)
	}

	// SetValues with different count of values
	f.SetValues([]string{"x", "y"})
	if f.excludedCount != 0 {
		t.Errorf("after SetValues: excludedCount = %d, want 0", f.excludedCount)
	}

	// Now exclude from new values
	f.Exclude("x")
	if f.excludedCount != 1 {
		t.Errorf("after excluding from new values: excludedCount = %d, want 1", f.excludedCount)
	}
}

func TestSetFilterExcludedCount_LargeValueSet(t *testing.T) {
	values := make([]string, 100)
	for i := range values {
		values[i] = fmt.Sprintf("val%d", i)
	}
	f := NewSetFilter(values...)

	// Exclude all
	for _, v := range values {
		f.Exclude(v)
	}
	if f.excludedCount != 100 {
		t.Errorf("after excluding all 100: excludedCount = %d, want 100", f.excludedCount)
	}

	// Include half
	for _, v := range values[:50] {
		f.Include(v)
	}
	if f.excludedCount != 50 {
		t.Errorf("after including 50: excludedCount = %d, want 50", f.excludedCount)
	}

	// IncludeAll
	f.IncludeAll()
	if f.excludedCount != 0 {
		t.Errorf("after IncludeAll: excludedCount = %d, want 0", f.excludedCount)
	}
}
