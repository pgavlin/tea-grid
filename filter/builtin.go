package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pgavlin/tea-grid/internal/lineedit"
)

// --- TextFilter ---

// TextFilter performs substring or regex matching on string values.
type TextFilter struct {
	editor   lineedit.Model
	regex    bool
	compiled *regexp.Regexp

	// Editing state
	width   int
	editing bool
}

// NewTextFilter creates a new TextFilter.
func NewTextFilter() *TextFilter {
	return &TextFilter{}
}

// SetText sets the filter text and recompiles the regex if enabled.
func (f *TextFilter) SetText(text string) {
	f.editor.SetText(text)
	f.compiled = nil
	if f.regex {
		f.compiled, _ = regexp.Compile(text)
	}
}

// SetRegex enables or disables regex matching mode.
func (f *TextFilter) SetRegex(regex bool) {
	f.regex = regex
	if regex && f.editor.Text() != "" {
		f.compiled, _ = regexp.Compile(f.editor.Text())
	} else {
		f.compiled = nil
	}
}

func (f *TextFilter) Matches(value any) bool {
	s := fmt.Sprintf("%v", value)
	if f.regex && f.compiled != nil {
		return f.compiled.MatchString(s)
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(f.editor.Text()))
}

func (f *TextFilter) View() string {
	if f.editing {
		return f.editor.RenderLine(f.width, "")
	}
	if f.editor.Text() == "" {
		return ""
	}
	return f.editor.Text()
}

func (f *TextFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	switch msg := msg.(type) {
	case FilterFocusMsg:
		f.editing = true
		f.width = msg.Width
		f.editor.CursorToEnd()
		return f, nil
	case FilterBlurMsg:
		f.editing = false
		return f, nil
	case tea.KeyPressMsg:
		if !f.editing {
			return f, nil
		}
		if f.editor.HandleKeyMsg(msg) {
			f.updateCompiled()
		}
	}
	return f, nil
}

func (f *TextFilter) updateCompiled() {
	f.compiled = nil
	if f.regex && f.editor.Text() != "" {
		f.compiled, _ = regexp.Compile(f.editor.Text())
	}
}

func (f *TextFilter) Active() bool {
	return f.editor.Text() != ""
}

func (f *TextFilter) Clear() {
	f.SetText("")
}

// Clause implements RoundTrippable. Returns the substring as a single
// value. Regex mode is lossy: returns ok=false so the query bar can
// annotate it instead of trying to render it textually.
func (f *TextFilter) Clause() (values []string, negate bool, ok bool) {
	if !f.Active() {
		return nil, false, false
	}
	if f.regex {
		return nil, false, false
	}
	return []string{f.editor.Text()}, false, true
}

// SetClause implements RoundTrippable. Resets regex mode and applies
// the value as substring text. Rejects negation and multi-value lists.
func (f *TextFilter) SetClause(values []string, negate bool) error {
	if negate {
		return fmt.Errorf("TextFilter does not support negation")
	}
	if len(values) != 1 {
		return fmt.Errorf("TextFilter expects exactly one value, got %d", len(values))
	}
	f.regex = false
	f.SetText(values[0])
	return nil
}

// --- NumberFilter ---

// NumberFilter performs comparison operations on numeric values.
// Supports operators: =, !=, <, >, <=, >=, and ranges (e.g., "10..50").
type NumberFilter struct {
	editor  lineedit.Model
	op      string
	val     float64
	val2    float64 // for range
	isRange bool

	// Editing state
	width   int
	editing bool
}

// NewNumberFilter creates a new NumberFilter.
func NewNumberFilter() *NumberFilter {
	return &NumberFilter{}
}

// SetText sets the filter expression (e.g. ">10", "5..20", "=42").
func (f *NumberFilter) SetText(text string) {
	f.editor.SetText(text)
	f.parseText()
}

func (f *NumberFilter) parseText() {
	text := strings.TrimSpace(f.editor.Text())
	f.isRange = false
	f.op = ""

	// Check for range: "10..50"
	if parts := strings.SplitN(text, "..", 2); len(parts) == 2 {
		v1, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		v2, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err1 == nil && err2 == nil {
			f.val = v1
			f.val2 = v2
			f.isRange = true
			return
		}
	}

	// Check for operators
	for _, op := range []string{"!=", "<=", ">=", "<", ">", "="} {
		if strings.HasPrefix(text, op) {
			rest := strings.TrimSpace(text[len(op):])
			if v, err := strconv.ParseFloat(rest, 64); err == nil {
				f.op = op
				f.val = v
				return
			}
		}
	}

	// Plain number (equals)
	if v, err := strconv.ParseFloat(text, 64); err == nil {
		f.op = "="
		f.val = v
	}
}

func (f *NumberFilter) Matches(value any) bool {
	var num float64
	switch v := value.(type) {
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case float64:
		num = v
	case float32:
		num = float64(v)
	default:
		return false
	}

	if f.isRange {
		return num >= f.val && num <= f.val2
	}

	switch f.op {
	case "=":
		return num == f.val
	case "!=":
		return num != f.val
	case "<":
		return num < f.val
	case ">":
		return num > f.val
	case "<=":
		return num <= f.val
	case ">=":
		return num >= f.val
	default:
		return true
	}
}

func (f *NumberFilter) View() string {
	if f.editing {
		return f.editor.RenderLine(f.width, "")
	}
	if f.editor.Text() == "" {
		return ""
	}
	return f.editor.Text()
}

func (f *NumberFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	switch msg := msg.(type) {
	case FilterFocusMsg:
		f.editing = true
		f.width = msg.Width
		f.editor.CursorToEnd()
		return f, nil
	case FilterBlurMsg:
		f.editing = false
		return f, nil
	case tea.KeyPressMsg:
		if !f.editing {
			return f, nil
		}
		if f.editor.HandleKeyMsg(msg) {
			f.parseText()
		}
	}
	return f, nil
}

func (f *NumberFilter) Active() bool {
	return f.editor.Text() != ""
}

func (f *NumberFilter) Clear() {
	f.SetText("")
}

// Clause implements RoundTrippable. Returns the editor text verbatim;
// it is already in the canonical form NumberFilter accepts.
func (f *NumberFilter) Clause() (values []string, negate bool, ok bool) {
	if !f.Active() {
		return nil, false, false
	}
	return []string{f.editor.Text()}, false, true
}

// SetClause implements RoundTrippable. The single value is parsed via
// the existing SetText path; an unparseable value returns an error and
// leaves prior state intact.
func (f *NumberFilter) SetClause(values []string, negate bool) error {
	if negate {
		return fmt.Errorf("NumberFilter does not support negation")
	}
	if len(values) != 1 {
		return fmt.Errorf("NumberFilter expects exactly one value, got %d", len(values))
	}
	prev := f.editor.Text()
	f.SetText(values[0])
	// parseText leaves op="" and isRange=false when it could not
	// extract any predicate. Active() only checks editor text, so we
	// inspect the parsed state directly.
	if values[0] != "" && f.op == "" && !f.isRange {
		f.SetText(prev)
		return fmt.Errorf("NumberFilter: could not parse %q", values[0])
	}
	return nil
}

// --- SetFilter ---

// SetFilter includes/excludes from a set of distinct values.
type SetFilter struct {
	values        map[string]bool // value -> included
	excludedCount int
	allValues     []string

	// Editing state
	editing     bool
	search      lineedit.Model
	filtered    []string // allValues filtered by searchText
	selectedIdx int      // cursor in filtered list
	scrollTop   int      // first visible item
	width       int
	maxLines    int
	inList      bool // false = typing in search, true = navigating list
}

// NewSetFilter creates a new SetFilter.
func NewSetFilter(allValues ...string) *SetFilter {
	values := make(map[string]bool)
	for _, v := range allValues {
		values[v] = true
	}
	return &SetFilter{
		values:    values,
		allValues: allValues,
	}
}

// SetValues replaces the set of known values, all initially included.
func (f *SetFilter) SetValues(values []string) {
	f.allValues = values
	f.values = make(map[string]bool, len(values))
	for _, v := range values {
		f.values[v] = true
	}
	f.excludedCount = 0
}

// Include marks a value as included in the filter.
func (f *SetFilter) Include(value string) {
	if !f.values[value] {
		f.excludedCount--
	}
	f.values[value] = true
}

// Exclude marks a value as excluded from the filter.
func (f *SetFilter) Exclude(value string) {
	if f.values[value] {
		f.excludedCount++
	}
	f.values[value] = false
}

// IncludeAll sets all values to included.
func (f *SetFilter) IncludeAll() {
	for k := range f.values {
		f.values[k] = true
	}
	f.excludedCount = 0
}

func (f *SetFilter) Matches(value any) bool {
	s := fmt.Sprintf("%v", value)
	included, exists := f.values[s]
	if !exists {
		return true // values not in the set are not filtered
	}
	return included
}

func (f *SetFilter) View() string {
	if !f.editing {
		return ""
	}

	var lines []string

	// Line 1: search input
	searchLine := f.search.RenderLine(f.width, "")
	if f.inList {
		// Dim the search line when in list mode
		dimStyle := lipgloss.NewStyle().Faint(true)
		searchLine = dimStyle.Render(searchLine)
	}
	lines = append(lines, searchLine)

	// Remaining lines: scrollable checkbox list
	listLines := f.maxLines - 1 // reserve 1 for search
	if listLines < 1 {
		listLines = 1
	}

	end := f.scrollTop + listLines
	if end > len(f.filtered) {
		end = len(f.filtered)
	}

	for i := f.scrollTop; i < end; i++ {
		val := f.filtered[i]
		checkbox := "[ ]"
		if f.values[val] {
			checkbox = "[x]"
		}

		entry := checkbox + " " + val
		// Truncate or pad to width
		entry = lineedit.TruncateOrPad(entry, f.width)

		if f.inList && i == f.selectedIdx {
			reverseStyle := lipgloss.NewStyle().Reverse(true)
			entry = reverseStyle.Render(entry)
		}

		lines = append(lines, entry)
	}

	return strings.Join(lines, "\n")
}

func (f *SetFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	switch msg := msg.(type) {
	case FilterFocusMsg:
		f.editing = true
		f.width = msg.Width
		f.maxLines = msg.MaxLines
		f.search.SetText("")
		f.search.SetCursor(0)
		f.selectedIdx = 0
		f.scrollTop = 0
		f.inList = false
		f.refilter()
		return f, nil
	case FilterBlurMsg:
		f.editing = false
		return f, nil
	case tea.KeyPressMsg:
		if !f.editing {
			return f, nil
		}
		if f.inList {
			return f.updateListMode(msg)
		}
		return f.updateSearchMode(msg)
	}
	return f, nil
}

func (f *SetFilter) updateSearchMode(msg tea.KeyPressMsg) (Filter, tea.Cmd) {
	switch msg.Code {
	case tea.KeyDown, tea.KeyTab:
		if len(f.filtered) > 0 {
			f.inList = true
		}
	default:
		if f.search.HandleKeyMsg(msg) {
			f.refilter()
		}
	}
	return f, nil
}

func (f *SetFilter) updateListMode(msg tea.KeyPressMsg) (Filter, tea.Cmd) {
	switch msg.Code {
	case tea.KeyUp:
		if f.selectedIdx > 0 {
			f.selectedIdx--
			f.ensureVisible()
		}
	case tea.KeyDown:
		if f.selectedIdx < len(f.filtered)-1 {
			f.selectedIdx++
			f.ensureVisible()
		}
	case tea.KeySpace:
		if f.selectedIdx >= 0 && f.selectedIdx < len(f.filtered) {
			val := f.filtered[f.selectedIdx]
			if msg.Mod.Contains(tea.ModCtrl) {
				// Ctrl+Space: select only the highlighted item (exclude all others).
				for k := range f.values {
					f.values[k] = false
				}
				f.values[val] = true
				f.excludedCount = len(f.values) - 1
			} else {
				if f.values[val] {
					f.excludedCount++
				} else {
					f.excludedCount--
				}
				f.values[val] = !f.values[val]
			}
		}
	case tea.KeyTab:
		f.inList = false
	default:
		if len(msg.Text) > 0 {
			// Switch back to search mode and process the text
			f.inList = false
			f.search.Insert(msg.Text)
			f.refilter()
		}
	}
	return f, nil
}

func (f *SetFilter) refilter() {
	if f.search.Text() == "" {
		f.filtered = make([]string, len(f.allValues))
		copy(f.filtered, f.allValues)
	} else {
		lower := strings.ToLower(f.search.Text())
		f.filtered = nil
		for _, v := range f.allValues {
			if strings.Contains(strings.ToLower(v), lower) {
				f.filtered = append(f.filtered, v)
			}
		}
	}
	f.selectedIdx = 0
	f.scrollTop = 0
}

func (f *SetFilter) ensureVisible() {
	listLines := f.maxLines - 1
	if listLines < 1 {
		listLines = 1
	}
	if f.selectedIdx < f.scrollTop {
		f.scrollTop = f.selectedIdx
	}
	if f.selectedIdx >= f.scrollTop+listLines {
		f.scrollTop = f.selectedIdx - listLines + 1
	}
}

func (f *SetFilter) Active() bool {
	return f.excludedCount > 0
}

func (f *SetFilter) Clear() {
	f.IncludeAll()
}

// Clause implements RoundTrippable. Returns the included subset by
// default; once more than half of allValues is included, returns the
// excluded subset with negate=true to keep the bar text small.
//
// New values that appear in row data after a clause was applied are
// treated as "not included" — consistent with current Include/SetValues
// semantics. Documented in the package README.
func (f *SetFilter) Clause() (values []string, negate bool, ok bool) {
	if !f.Active() {
		return nil, false, false
	}
	includedCount := len(f.values) - f.excludedCount
	if includedCount*2 > len(f.values) {
		var excluded []string
		for _, v := range f.allValues {
			if !f.values[v] {
				excluded = append(excluded, v)
			}
		}
		return excluded, true, true
	}
	var included []string
	for _, v := range f.allValues {
		if f.values[v] {
			included = append(included, v)
		}
	}
	return included, false, true
}

// SetClause implements RoundTrippable. With negate=false, includes
// exactly the listed values and excludes the rest. With negate=true,
// includes everything except the listed values. Values not in
// allValues are ignored (treated as "not included" — see Clause's
// godoc).
func (f *SetFilter) SetClause(values []string, negate bool) error {
	want := make(map[string]bool, len(values))
	for _, v := range values {
		want[v] = true
	}
	f.excludedCount = 0
	for _, v := range f.allValues {
		var included bool
		if negate {
			included = !want[v]
		} else {
			included = want[v]
		}
		f.values[v] = included
		if !included {
			f.excludedCount++
		}
	}
	return nil
}

// --- BoolFilter ---

// BoolFilter filters true/false/any values.
type BoolFilter struct {
	state int // 0 = any, 1 = true only, 2 = false only

	// Editing state
	width   int
	editing bool
}

// NewBoolFilter creates a new BoolFilter.
func NewBoolFilter() *BoolFilter {
	return &BoolFilter{state: 0}
}

// Toggle cycles the filter state: any → true only → false only → any.
func (f *BoolFilter) Toggle() {
	f.state = (f.state + 1) % 3
}

func (f *BoolFilter) Matches(value any) bool {
	if f.state == 0 {
		return true
	}
	b, ok := value.(bool)
	if !ok {
		return false
	}
	if f.state == 1 {
		return b
	}
	return !b
}

func (f *BoolFilter) View() string {
	if f.editing {
		anyR, trueR, falseR := "( )", "( )", "( )"
		switch f.state {
		case 0:
			anyR = "(\u25cf)"
		case 1:
			trueR = "(\u25cf)"
		case 2:
			falseR = "(\u25cf)"
		}
		line := anyR + " Any  " + trueR + " True  " + falseR + " False"
		return lineedit.TruncateOrPad(line, f.width)
	}
	switch f.state {
	case 1:
		return "true"
	case 2:
		return "false"
	default:
		return ""
	}
}

func (f *BoolFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	switch msg := msg.(type) {
	case FilterFocusMsg:
		f.editing = true
		f.width = msg.Width
		return f, nil
	case FilterBlurMsg:
		f.editing = false
		return f, nil
	case tea.KeyPressMsg:
		if !f.editing {
			return f, nil
		}
		switch msg.Code {
		case tea.KeySpace, tea.KeyRight:
			f.state = (f.state + 1) % 3
		case tea.KeyLeft:
			f.state = (f.state + 2) % 3
		}
	}
	return f, nil
}

func (f *BoolFilter) Active() bool {
	return f.state != 0
}

func (f *BoolFilter) Clear() {
	f.state = 0
}

// Clause implements RoundTrippable.
func (f *BoolFilter) Clause() (values []string, negate bool, ok bool) {
	switch f.state {
	case 1:
		return []string{"true"}, false, true
	case 2:
		return []string{"false"}, false, true
	default:
		return nil, false, false
	}
}

// SetClause implements RoundTrippable. Accepts "true", "false", "1",
// or "0" (case-insensitive).
func (f *BoolFilter) SetClause(values []string, negate bool) error {
	if negate {
		return fmt.Errorf("BoolFilter does not support negation")
	}
	if len(values) != 1 {
		return fmt.Errorf("BoolFilter expects exactly one value, got %d", len(values))
	}
	switch strings.ToLower(values[0]) {
	case "true", "1":
		f.state = 1
	case "false", "0":
		f.state = 2
	default:
		return fmt.Errorf("BoolFilter: unrecognized value %q", values[0])
	}
	return nil
}

// --- TimeFilter ---

// TimeFilter filters time.Time values by a date/time range.
// Accepts "start..end" format where either bound can be omitted.
type TimeFilter struct {
	editor lineedit.Model
	start  *time.Time
	end    *time.Time

	// Editing state
	width   int
	editing bool
}

// NewTimeFilter creates a new TimeFilter.
func NewTimeFilter() *TimeFilter {
	return &TimeFilter{}
}

// SetText sets the filter expression (e.g. "2024-01-01", "2024-01-01..2024-12-31").
func (f *TimeFilter) SetText(text string) {
	f.editor.SetText(text)
	f.start = nil
	f.end = nil

	parts := strings.SplitN(text, "..", 2)
	if len(parts) == 2 {
		if s := strings.TrimSpace(parts[0]); s != "" {
			if t, err := parseTime(s); err == nil {
				f.start = &t
			}
		}
		if s := strings.TrimSpace(parts[1]); s != "" {
			if t, err := parseTime(s); err == nil {
				f.end = &t
			}
		}
	} else {
		// Single date — match that exact day
		if t, err := parseTime(strings.TrimSpace(text)); err == nil {
			f.start = &t
			end := t.Add(24 * time.Hour)
			f.end = &end
		}
	}
}

var timeFormats = []string{
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"Jan 2 2006 3:04pm",
	"Jan 2 2006 3:04PM",
	"Jan 2 2006",
	"Jan 2, 2006",
	"January 2, 2006",
	"01/02/2006",
	time.RFC1123Z,
	time.RFC1123,
	time.RFC3339,
}

func parseTime(s string) (time.Time, error) {
	for _, format := range timeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

func (f *TimeFilter) Matches(value any) bool {
	var t time.Time
	switch v := value.(type) {
	case time.Time:
		t = v
	case *time.Time:
		if v == nil {
			return false
		}
		t = *v
	default:
		return false
	}

	if f.start != nil && t.Before(*f.start) {
		return false
	}
	if f.end != nil && t.After(*f.end) {
		return false
	}
	return true
}

func (f *TimeFilter) View() string {
	if f.editing {
		return f.editor.RenderLine(f.width, "")
	}
	if f.editor.Text() == "" {
		return ""
	}
	return f.editor.Text()
}

func (f *TimeFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	switch msg := msg.(type) {
	case FilterFocusMsg:
		f.editing = true
		f.width = msg.Width
		f.editor.CursorToEnd()
		return f, nil
	case FilterBlurMsg:
		f.editing = false
		return f, nil
	case tea.KeyPressMsg:
		if !f.editing {
			return f, nil
		}
		if f.editor.HandleKeyMsg(msg) {
			f.SetText(f.editor.Text())
		}
	}
	return f, nil
}

func (f *TimeFilter) Active() bool {
	return f.start != nil || f.end != nil
}

func (f *TimeFilter) Clear() {
	f.SetText("")
}

// Clause implements RoundTrippable. Returns the editor text verbatim;
// it is already in the canonical form TimeFilter accepts.
func (f *TimeFilter) Clause() (values []string, negate bool, ok bool) {
	if !f.Active() {
		return nil, false, false
	}
	return []string{f.editor.Text()}, false, true
}

// SetClause implements RoundTrippable. Parses via the existing SetText
// path; if parsing yields no bounds, restores prior text and returns
// an error.
func (f *TimeFilter) SetClause(values []string, negate bool) error {
	if negate {
		return fmt.Errorf("TimeFilter does not support negation")
	}
	if len(values) != 1 {
		return fmt.Errorf("TimeFilter expects exactly one value, got %d", len(values))
	}
	prev := f.editor.Text()
	f.SetText(values[0])
	if !f.Active() && values[0] != "" {
		f.SetText(prev)
		return fmt.Errorf("TimeFilter: could not parse %q", values[0])
	}
	return nil
}
