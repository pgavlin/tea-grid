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
	text    string
	regex   bool
	compiled *regexp.Regexp
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
	if f.text == "" {
		return ""
	}
	return f.text
}

func (f *TextFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	return f, nil
}

func (f *TextFilter) Active() bool {
	return f.text != ""
}

// --- NumberFilter ---

// NumberFilter performs comparison operations on numeric values.
// Supports operators: =, !=, <, >, <=, >=, and ranges (e.g., "10..50").
type NumberFilter struct {
	text string
	op   string
	val  float64
	val2 float64 // for range
	isRange bool
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
	if f.text == "" {
		return ""
	}
	return f.text
}

func (f *NumberFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	return f, nil
}

func (f *NumberFilter) Active() bool {
	return f.text != ""
}

// --- SetFilter ---

// SetFilter includes/excludes from a set of distinct values.
type SetFilter struct {
	values   map[string]bool // value -> included
	allValues []string
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

func (f *SetFilter) Matches(value any) bool {
	s := fmt.Sprintf("%v", value)
	included, exists := f.values[s]
	if !exists {
		return true // values not in the set are not filtered
	}
	return included
}

func (f *SetFilter) View() string {
	return ""
}

func (f *SetFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	return f, nil
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
	if f.text == "" {
		return ""
	}
	return f.text
}

func (f *TimeFilter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	return f, nil
}

func (f *TimeFilter) Active() bool {
	return f.start != nil || f.end != nil
}
