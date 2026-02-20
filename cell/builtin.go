package cell

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Built-in Renderers ---

// TextRenderer is the default renderer. Truncates/pads text to fit width.
type TextRenderer[T any] struct{}

func (r TextRenderer[T]) Render(ctx CellContext[T]) string {
	return truncateOrPad(ctx.FormattedValue, ctx.Width)
}

// NumberRenderer is a right-aligned renderer with optional thousands separator.
type NumberRenderer[T any] struct {
	ThousandsSep bool
}

func (r NumberRenderer[T]) Render(ctx CellContext[T]) string {
	s := ctx.FormattedValue
	if r.ThousandsSep {
		s = addThousandsSep(s)
	}
	// Right-align
	if len(s) < ctx.Width {
		s = strings.Repeat(" ", ctx.Width-len(s)) + s
	} else if len(s) > ctx.Width {
		s = s[:ctx.Width]
	}
	return s
}

// TimeRenderer renders time.Time values with a configurable format.
type TimeRenderer[T any] struct {
	Format   string // Go time format string. Default: "2006-01-02 15:04".
	Relative bool   // If true, renders relative time (e.g., "2h ago").
}

func (r TimeRenderer[T]) Render(ctx CellContext[T]) string {
	t, ok := ctx.Value.(time.Time)
	if !ok {
		return truncateOrPad(ctx.FormattedValue, ctx.Width)
	}

	var s string
	if r.Relative {
		s = relativeTime(t)
	} else {
		format := r.Format
		if format == "" {
			format = "2006-01-02 15:04"
		}
		s = t.Format(format)
	}
	return truncateOrPad(s, ctx.Width)
}

// BarRenderer renders a horizontal bar proportional to value.
type BarRenderer[T any] struct {
	MaxValue float64
	BarChar  string
	Style    lipgloss.Style
}

func (r BarRenderer[T]) Render(ctx CellContext[T]) string {
	val := toFloat64(ctx.Value)
	maxVal := r.MaxValue
	if maxVal == 0 {
		maxVal = 100
	}
	barChar := r.BarChar
	if barChar == "" {
		barChar = "█"
	}

	ratio := val / maxVal
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	barWidth := int(float64(ctx.Width) * ratio)
	bar := strings.Repeat(barChar, barWidth)
	if r.Style.Value() != "" {
		bar = r.Style.Render(bar)
	}
	return truncateOrPad(bar, ctx.Width)
}

// SparklineRenderer renders an inline sparkline for numeric series.
type SparklineRenderer[T any] struct{}

func (r SparklineRenderer[T]) Render(ctx CellContext[T]) string {
	values, ok := ctx.Value.([]float64)
	if !ok {
		return truncateOrPad(ctx.FormattedValue, ctx.Width)
	}

	blocks := []rune("▁▂▃▄▅▆▇█")
	if len(values) == 0 {
		return truncateOrPad("", ctx.Width)
	}

	minV, maxV := values[0], values[0]
	for _, v := range values {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}

	rangeV := maxV - minV
	if rangeV == 0 {
		rangeV = 1
	}

	var sb strings.Builder
	for _, v := range values {
		idx := int((v - minV) / rangeV * float64(len(blocks)-1))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		sb.WriteRune(blocks[idx])
	}
	return truncateOrPad(sb.String(), ctx.Width)
}

// BoolRenderer renders checkmark/cross glyphs for boolean values.
type BoolRenderer[T any] struct {
	TrueGlyph  string
	FalseGlyph string
}

func (r BoolRenderer[T]) Render(ctx CellContext[T]) string {
	b, ok := ctx.Value.(bool)
	if !ok {
		return truncateOrPad(ctx.FormattedValue, ctx.Width)
	}
	trueG := r.TrueGlyph
	if trueG == "" {
		trueG = "✓"
	}
	falseG := r.FalseGlyph
	if falseG == "" {
		falseG = "✗"
	}
	if b {
		return truncateOrPad(trueG, ctx.Width)
	}
	return truncateOrPad(falseG, ctx.Width)
}

// ProgressRenderer renders a mini progress bar within the cell.
type ProgressRenderer[T any] struct {
	MaxValue float64
	FilledChar string
	EmptyChar  string
}

func (r ProgressRenderer[T]) Render(ctx CellContext[T]) string {
	val := toFloat64(ctx.Value)
	maxVal := r.MaxValue
	if maxVal == 0 {
		maxVal = 100
	}
	filled := r.FilledChar
	if filled == "" {
		filled = "━"
	}
	empty := r.EmptyChar
	if empty == "" {
		empty = "─"
	}

	ratio := val / maxVal
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	filledWidth := int(float64(ctx.Width) * ratio)
	emptyWidth := ctx.Width - filledWidth
	return strings.Repeat(filled, filledWidth) + strings.Repeat(empty, emptyWidth)
}

// --- Built-in Editors ---

// TextEditorModel is a single-line text input editor.
type TextEditorModel[T any] struct {
	value string
	cursor int
}

func NewTextEditor[T any]() *TextEditorModel[T] {
	return &TextEditorModel[T]{}
}

func (e *TextEditorModel[T]) Init(ctx CellContext[T]) tea.Cmd {
	e.value = ctx.FormattedValue
	e.cursor = len(e.value)
	return nil
}

func (e *TextEditorModel[T]) Update(msg tea.Msg) (CellEditor[T], tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyBackspace:
			if e.cursor > 0 {
				e.value = e.value[:e.cursor-1] + e.value[e.cursor:]
				e.cursor--
			}
		case tea.KeyDelete:
			if e.cursor < len(e.value) {
				e.value = e.value[:e.cursor] + e.value[e.cursor+1:]
			}
		case tea.KeyLeft:
			if e.cursor > 0 {
				e.cursor--
			}
		case tea.KeyRight:
			if e.cursor < len(e.value) {
				e.cursor++
			}
		case tea.KeyHome, tea.KeyCtrlA:
			e.cursor = 0
		case tea.KeyEnd, tea.KeyCtrlE:
			e.cursor = len(e.value)
		case tea.KeyRunes:
			e.value = e.value[:e.cursor] + string(msg.Runes) + e.value[e.cursor:]
			e.cursor += len(msg.Runes)
		}
	}
	return e, nil
}

func (e *TextEditorModel[T]) View() string {
	return e.value
}

func (e *TextEditorModel[T]) Value() any {
	return e.value
}

func (e *TextEditorModel[T]) Validate() string {
	return ""
}

// NumberEditorModel is a numeric input editor.
type NumberEditorModel[T any] struct {
	text string
	min  *float64
	max  *float64
	step float64
}

func NewNumberEditor[T any]() *NumberEditorModel[T] {
	return &NumberEditorModel[T]{step: 1}
}

func (e *NumberEditorModel[T]) WithMin(min float64) *NumberEditorModel[T] {
	e.min = &min
	return e
}

func (e *NumberEditorModel[T]) WithMax(max float64) *NumberEditorModel[T] {
	e.max = &max
	return e
}

func (e *NumberEditorModel[T]) WithStep(step float64) *NumberEditorModel[T] {
	e.step = step
	return e
}

func (e *NumberEditorModel[T]) Init(ctx CellContext[T]) tea.Cmd {
	e.text = fmt.Sprintf("%v", ctx.Value)
	return nil
}

func (e *NumberEditorModel[T]) Update(msg tea.Msg) (CellEditor[T], tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyBackspace:
			if len(e.text) > 0 {
				e.text = e.text[:len(e.text)-1]
			}
		case tea.KeyRunes:
			for _, r := range msg.Runes {
				if (r >= '0' && r <= '9') || r == '.' || r == '-' {
					e.text += string(r)
				}
			}
		case tea.KeyUp:
			if v, err := strconv.ParseFloat(e.text, 64); err == nil {
				e.text = strconv.FormatFloat(v+e.step, 'f', -1, 64)
			}
		case tea.KeyDown:
			if v, err := strconv.ParseFloat(e.text, 64); err == nil {
				e.text = strconv.FormatFloat(v-e.step, 'f', -1, 64)
			}
		}
	}
	return e, nil
}

func (e *NumberEditorModel[T]) View() string {
	return e.text
}

func (e *NumberEditorModel[T]) Value() any {
	v, _ := strconv.ParseFloat(e.text, 64)
	return v
}

func (e *NumberEditorModel[T]) Validate() string {
	v, err := strconv.ParseFloat(e.text, 64)
	if err != nil {
		return "invalid number"
	}
	if e.min != nil && v < *e.min {
		return fmt.Sprintf("minimum value is %v", *e.min)
	}
	if e.max != nil && v > *e.max {
		return fmt.Sprintf("maximum value is %v", *e.max)
	}
	return ""
}

// SelectEditorModel cycles through a list of options.
type SelectEditorModel[T any] struct {
	options []string
	index   int
}

func NewSelectEditor[T any](options []string) *SelectEditorModel[T] {
	return &SelectEditorModel[T]{options: options}
}

func (e *SelectEditorModel[T]) Init(ctx CellContext[T]) tea.Cmd {
	s := fmt.Sprintf("%v", ctx.Value)
	for i, opt := range e.options {
		if opt == s {
			e.index = i
			break
		}
	}
	return nil
}

func (e *SelectEditorModel[T]) Update(msg tea.Msg) (CellEditor[T], tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyUp, tea.KeyLeft:
			if e.index > 0 {
				e.index--
			} else {
				e.index = len(e.options) - 1
			}
		case tea.KeyDown, tea.KeyRight:
			e.index = (e.index + 1) % len(e.options)
		}
	}
	return e, nil
}

func (e *SelectEditorModel[T]) View() string {
	if len(e.options) == 0 {
		return ""
	}
	return e.options[e.index]
}

func (e *SelectEditorModel[T]) Value() any {
	if len(e.options) == 0 {
		return ""
	}
	return e.options[e.index]
}

func (e *SelectEditorModel[T]) Validate() string {
	return ""
}

// BoolEditorModel toggles true/false.
type BoolEditorModel[T any] struct {
	value bool
}

func NewBoolEditor[T any]() *BoolEditorModel[T] {
	return &BoolEditorModel[T]{}
}

func (e *BoolEditorModel[T]) Init(ctx CellContext[T]) tea.Cmd {
	if b, ok := ctx.Value.(bool); ok {
		e.value = b
	}
	return nil
}

func (e *BoolEditorModel[T]) Update(msg tea.Msg) (CellEditor[T], tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeySpace, tea.KeyUp, tea.KeyDown:
			e.value = !e.value
		}
	}
	return e, nil
}

func (e *BoolEditorModel[T]) View() string {
	if e.value {
		return "true"
	}
	return "false"
}

func (e *BoolEditorModel[T]) Value() any {
	return e.value
}

func (e *BoolEditorModel[T]) Validate() string {
	return ""
}

// TimeEditorModel edits time.Time values via text input.
type TimeEditorModel[T any] struct {
	text     string
	parseErr string
}

func NewTimeEditor[T any]() *TimeEditorModel[T] {
	return &TimeEditorModel[T]{}
}

var timeEditFormats = []string{
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

func (e *TimeEditorModel[T]) Init(ctx CellContext[T]) tea.Cmd {
	if t, ok := ctx.Value.(time.Time); ok {
		e.text = t.Format("2006-01-02 15:04")
	} else {
		e.text = ctx.FormattedValue
	}
	return nil
}

func (e *TimeEditorModel[T]) Update(msg tea.Msg) (CellEditor[T], tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyBackspace:
			if len(e.text) > 0 {
				e.text = e.text[:len(e.text)-1]
			}
		case tea.KeyRunes:
			e.text += string(msg.Runes)
		}
		e.parseErr = ""
	}
	return e, nil
}

func (e *TimeEditorModel[T]) View() string {
	if e.parseErr != "" {
		return e.text + " (" + e.parseErr + ")"
	}
	return e.text
}

func (e *TimeEditorModel[T]) Value() any {
	for _, format := range timeEditFormats {
		if t, err := time.Parse(format, e.text); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (e *TimeEditorModel[T]) Validate() string {
	for _, format := range timeEditFormats {
		if _, err := time.Parse(format, e.text); err == nil {
			return ""
		}
	}
	e.parseErr = "invalid time format"
	return "invalid time format"
}

// --- Helpers ---

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

func addThousandsSep(s string) string {
	// Find the decimal point
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]

	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}

	if len(intPart) <= 3 {
		if negative {
			intPart = "-" + intPart
		}
		if len(parts) == 2 {
			return intPart + "." + parts[1]
		}
		return intPart
	}

	var result strings.Builder
	remainder := len(intPart) % 3
	if remainder > 0 {
		result.WriteString(intPart[:remainder])
	}
	for i := remainder; i < len(intPart); i += 3 {
		if result.Len() > 0 {
			result.WriteByte(',')
		}
		result.WriteString(intPart[i : i+3])
	}

	s = result.String()
	if negative {
		s = "-" + s
	}
	if len(parts) == 2 {
		s += "." + parts[1]
	}
	return s
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	default:
		return 0
	}
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = -d
		return "in " + formatDuration(d)
	}
	return formatDuration(d) + " ago"
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy", int(d.Hours()/(24*365)))
	}
}
