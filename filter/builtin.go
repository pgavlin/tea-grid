package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// --- TextFilter ---

// TextFilter performs substring or regex matching on string values.
type TextFilter struct {
	text     string
	regex    bool
	compiled *regexp.Regexp

	// Editing state
	cursor  int
	width   int
	editing bool
}

// NewTextFilter creates a new TextFilter.
func NewTextFilter() *TextFilter {
	return &TextFilter{}
}

func (f *TextFilter) SetText(text string) {
	f.text = text
	f.compiled = nil
	if f.regex {
		f.compiled, _ = regexp.Compile(text)
	}
}

func (f *TextFilter) SetRegex(regex bool) {
	f.regex = regex
	if regex && f.text != "" {
		f.compiled, _ = regexp.Compile(f.text)
	} else {
		f.compiled = nil
	}
}

func (f *TextFilter) Matches(value any) bool {
	s := fmt.Sprintf("%v", value)
	if f.regex && f.compiled != nil {
		return f.compiled.MatchString(s)
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(f.text))
}

func (f *TextFilter) View() string {
	if f.editing {
		return renderEditorLine(f.text, f.cursor, f.width, "")
	}
	if f.text == "" {
		return ""
	}
	return f.text
}

func (f *TextFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	switch msg := msg.(type) {
	case FilterFocusMsg:
		f.editing = true
		f.width = msg.Width
		f.cursor = len(f.text)
		return f, nil
	case FilterBlurMsg:
		f.editing = false
		return f, nil
	case tea.KeyMsg:
		if !f.editing {
			return f, nil
		}
		switch msg.Type {
		case tea.KeyBackspace:
			if f.cursor > 0 {
				f.text = f.text[:f.cursor-1] + f.text[f.cursor:]
				f.cursor--
				f.updateCompiled()
			}
		case tea.KeyDelete:
			if f.cursor < len(f.text) {
				f.text = f.text[:f.cursor] + f.text[f.cursor+1:]
				f.updateCompiled()
			}
		case tea.KeyLeft:
			if f.cursor > 0 {
				f.cursor--
			}
		case tea.KeyRight:
			if f.cursor < len(f.text) {
				f.cursor++
			}
		case tea.KeyHome, tea.KeyCtrlA:
			f.cursor = 0
		case tea.KeyEnd, tea.KeyCtrlE:
			f.cursor = len(f.text)
		case tea.KeySpace:
			f.text = f.text[:f.cursor] + " " + f.text[f.cursor:]
			f.cursor++
			f.updateCompiled()
		case tea.KeyRunes:
			f.text = f.text[:f.cursor] + string(msg.Runes) + f.text[f.cursor:]
			f.cursor += len(msg.Runes)
			f.updateCompiled()
		}
	}
	return f, nil
}

func (f *TextFilter) updateCompiled() {
	f.compiled = nil
	if f.regex && f.text != "" {
		f.compiled, _ = regexp.Compile(f.text)
	}
}

func (f *TextFilter) Active() bool {
	return f.text != ""
}

// --- NumberFilter ---

// NumberFilter performs comparison operations on numeric values.
// Supports operators: =, !=, <, >, <=, >=, and ranges (e.g., "10..50").
type NumberFilter struct {
	text    string
	op      string
	val     float64
	val2    float64 // for range
	isRange bool

	// Editing state
	cursor  int
	width   int
	editing bool
}

// NewNumberFilter creates a new NumberFilter.
func NewNumberFilter() *NumberFilter {
	return &NumberFilter{}
}

func (f *NumberFilter) SetText(text string) {
	f.text = text
	f.parseText()
}

func (f *NumberFilter) parseText() {
	text := strings.TrimSpace(f.text)
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
		return renderEditorLine(f.text, f.cursor, f.width, "")
	}
	if f.text == "" {
		return ""
	}
	return f.text
}

func (f *NumberFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	switch msg := msg.(type) {
	case FilterFocusMsg:
		f.editing = true
		f.width = msg.Width
		f.cursor = len(f.text)
		return f, nil
	case FilterBlurMsg:
		f.editing = false
		return f, nil
	case tea.KeyMsg:
		if !f.editing {
			return f, nil
		}
		switch msg.Type {
		case tea.KeyBackspace:
			if f.cursor > 0 {
				f.text = f.text[:f.cursor-1] + f.text[f.cursor:]
				f.cursor--
				f.parseText()
			}
		case tea.KeyDelete:
			if f.cursor < len(f.text) {
				f.text = f.text[:f.cursor] + f.text[f.cursor+1:]
				f.parseText()
			}
		case tea.KeyLeft:
			if f.cursor > 0 {
				f.cursor--
			}
		case tea.KeyRight:
			if f.cursor < len(f.text) {
				f.cursor++
			}
		case tea.KeyHome, tea.KeyCtrlA:
			f.cursor = 0
		case tea.KeyEnd, tea.KeyCtrlE:
			f.cursor = len(f.text)
		case tea.KeySpace:
			f.text = f.text[:f.cursor] + " " + f.text[f.cursor:]
			f.cursor++
			f.parseText()
		case tea.KeyRunes:
			f.text = f.text[:f.cursor] + string(msg.Runes) + f.text[f.cursor:]
			f.cursor += len(msg.Runes)
			f.parseText()
		}
	}
	return f, nil
}

func (f *NumberFilter) Active() bool {
	return f.text != ""
}

// --- SetFilter ---

// SetFilter includes/excludes from a set of distinct values.
type SetFilter struct {
	values    map[string]bool // value -> included
	allValues []string

	// Editing state
	editing      bool
	searchText   string
	searchCursor int
	filtered     []string // allValues filtered by searchText
	selectedIdx  int      // cursor in filtered list
	scrollTop    int      // first visible item
	width        int
	maxLines     int
	inList       bool // false = typing in search, true = navigating list
}

// NewSetFilter creates a new SetFilter.
func NewSetFilter() *SetFilter {
	return &SetFilter{
		values: make(map[string]bool),
	}
}

func (f *SetFilter) SetValues(values []string) {
	f.allValues = values
	f.values = make(map[string]bool, len(values))
	for _, v := range values {
		f.values[v] = true
	}
}

func (f *SetFilter) Include(value string) {
	f.values[value] = true
}

func (f *SetFilter) Exclude(value string) {
	f.values[value] = false
}

// IncludeAll sets all values to included.
func (f *SetFilter) IncludeAll() {
	for k := range f.values {
		f.values[k] = true
	}
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
	searchLine := renderEditorLine(f.searchText, f.searchCursor, f.width, "")
	if f.inList {
		// Dim the search line when in list mode
		searchLine = "\x1b[2m" + searchLine + "\x1b[22m"
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
		entry = truncateOrPad(entry, f.width)

		if f.inList && i == f.selectedIdx {
			entry = "\x1b[7m" + entry + "\x1b[27m"
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
		f.searchText = ""
		f.searchCursor = 0
		f.selectedIdx = 0
		f.scrollTop = 0
		f.inList = false
		f.refilter()
		return f, nil
	case FilterBlurMsg:
		f.editing = false
		return f, nil
	case tea.KeyMsg:
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

func (f *SetFilter) updateSearchMode(msg tea.KeyMsg) (Filter, tea.Cmd) {
	switch msg.Type {
	case tea.KeyBackspace:
		if f.searchCursor > 0 {
			f.searchText = f.searchText[:f.searchCursor-1] + f.searchText[f.searchCursor:]
			f.searchCursor--
			f.refilter()
		}
	case tea.KeyDelete:
		if f.searchCursor < len(f.searchText) {
			f.searchText = f.searchText[:f.searchCursor] + f.searchText[f.searchCursor+1:]
			f.refilter()
		}
	case tea.KeyLeft:
		if f.searchCursor > 0 {
			f.searchCursor--
		}
	case tea.KeyRight:
		if f.searchCursor < len(f.searchText) {
			f.searchCursor++
		}
	case tea.KeyHome, tea.KeyCtrlA:
		f.searchCursor = 0
	case tea.KeyEnd, tea.KeyCtrlE:
		f.searchCursor = len(f.searchText)
	case tea.KeyDown, tea.KeyTab:
		if len(f.filtered) > 0 {
			f.inList = true
		}
	case tea.KeySpace:
		f.searchText = f.searchText[:f.searchCursor] + " " + f.searchText[f.searchCursor:]
		f.searchCursor++
		f.refilter()
	case tea.KeyRunes:
		f.searchText = f.searchText[:f.searchCursor] + string(msg.Runes) + f.searchText[f.searchCursor:]
		f.searchCursor += len(msg.Runes)
		f.refilter()
	}
	return f, nil
}

func (f *SetFilter) updateListMode(msg tea.KeyMsg) (Filter, tea.Cmd) {
	switch msg.Type {
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
			f.values[val] = !f.values[val]
		}
	case tea.KeyTab:
		f.inList = false
	case tea.KeyRunes:
		// Switch back to search mode and process the rune
		f.inList = false
		f.searchText = f.searchText[:f.searchCursor] + string(msg.Runes) + f.searchText[f.searchCursor:]
		f.searchCursor += len(msg.Runes)
		f.refilter()
	}
	return f, nil
}

func (f *SetFilter) refilter() {
	if f.searchText == "" {
		f.filtered = make([]string, len(f.allValues))
		copy(f.filtered, f.allValues)
	} else {
		lower := strings.ToLower(f.searchText)
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
	for _, included := range f.values {
		if !included {
			return true
		}
	}
	return false
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
		return truncateOrPad(line, f.width)
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
	case tea.KeyMsg:
		if !f.editing {
			return f, nil
		}
		switch msg.Type {
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

// --- TimeFilter ---

// TimeFilter filters time.Time values by a date/time range.
// Accepts "start..end" format where either bound can be omitted.
type TimeFilter struct {
	text  string
	start *time.Time
	end   *time.Time

	// Editing state
	cursor  int
	width   int
	editing bool
}

// NewTimeFilter creates a new TimeFilter.
func NewTimeFilter() *TimeFilter {
	return &TimeFilter{}
}

func (f *TimeFilter) SetText(text string) {
	f.text = text
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
		return renderEditorLine(f.text, f.cursor, f.width, "")
	}
	if f.text == "" {
		return ""
	}
	return f.text
}

func (f *TimeFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	switch msg := msg.(type) {
	case FilterFocusMsg:
		f.editing = true
		f.width = msg.Width
		f.cursor = len(f.text)
		return f, nil
	case FilterBlurMsg:
		f.editing = false
		return f, nil
	case tea.KeyMsg:
		if !f.editing {
			return f, nil
		}
		switch msg.Type {
		case tea.KeyBackspace:
			if f.cursor > 0 {
				f.text = f.text[:f.cursor-1] + f.text[f.cursor:]
				f.cursor--
				f.SetText(f.text)
			}
		case tea.KeyDelete:
			if f.cursor < len(f.text) {
				f.text = f.text[:f.cursor] + f.text[f.cursor+1:]
				f.SetText(f.text)
			}
		case tea.KeyLeft:
			if f.cursor > 0 {
				f.cursor--
			}
		case tea.KeyRight:
			if f.cursor < len(f.text) {
				f.cursor++
			}
		case tea.KeyHome, tea.KeyCtrlA:
			f.cursor = 0
		case tea.KeyEnd, tea.KeyCtrlE:
			f.cursor = len(f.text)
		case tea.KeySpace:
			f.text = f.text[:f.cursor] + " " + f.text[f.cursor:]
			f.cursor++
			f.SetText(f.text)
		case tea.KeyRunes:
			f.text = f.text[:f.cursor] + string(msg.Runes) + f.text[f.cursor:]
			f.cursor += len(msg.Runes)
			f.SetText(f.text)
		}
	}
	return f, nil
}

func (f *TimeFilter) Active() bool {
	return f.start != nil || f.end != nil
}

// --- Helpers ---

// ANSI escape sequences to toggle reverse video without resetting other attributes.
const (
	reverseOn  = "\x1b[7m"
	reverseOff = "\x1b[27m"
)

// renderEditorLine renders a single-line editor value as a viewport of the
// given width, ensuring the cursor is always visible.
func renderEditorLine(value string, cursor, width int, suffix string) string {
	if width <= 0 {
		return ""
	}

	suffixRunes := []rune(suffix)
	viewWidth := width - len(suffixRunes)
	if viewWidth < 1 {
		viewWidth = 1
	}

	runes := []rune(value)

	var cursorRune rune = ' '
	if cursor < len(runes) {
		cursorRune = runes[cursor]
	}

	start := 0
	if cursor >= viewWidth {
		start = cursor - viewWidth + 1
	}

	end := start + viewWidth
	if end > len(runes)+1 {
		end = len(runes) + 1
	}

	var before, after string
	if start < len(runes) {
		beforeEnd := cursor
		if beforeEnd > len(runes) {
			beforeEnd = len(runes)
		}
		if start < beforeEnd {
			before = string(runes[start:beforeEnd])
		}
	}

	afterStart := cursor + 1
	if afterStart < len(runes) {
		afterEnd := end
		if afterEnd > len(runes) {
			afterEnd = len(runes)
		}
		if afterStart < afterEnd {
			after = string(runes[afterStart:afterEnd])
		}
	}

	rendered := before + reverseOn + string(cursorRune) + reverseOff + after

	visibleLen := len([]rune(before)) + 1 + len([]rune(after))
	if visibleLen < viewWidth {
		rendered += strings.Repeat(" ", viewWidth-visibleLen)
	}

	return rendered + suffix
}

func truncateOrPad(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > width {
		if width > 1 {
			return string(runes[:width-1]) + "…"
		}
		return string(runes[:width])
	}
	if len(runes) < width {
		return s + strings.Repeat(" ", width-len(runes))
	}
	return s
}
