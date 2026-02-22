package data

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- CellRendererFunc ---

func TestCellRendererFuncRender(t *testing.T) {
	fn := CellRendererFunc[string](func(ctx CellContext[string]) string {
		return "custom:" + ctx.FormattedValue
	})
	ctx := CellContext[string]{FormattedValue: "hello", Width: 20}
	got := fn.Render(ctx)
	if got != "custom:hello" {
		t.Errorf("CellRendererFunc.Render: expected %q, got %q", "custom:hello", got)
	}
}

// --- NumberRenderer ---

func TestNumberRendererThousandsSepEnabled(t *testing.T) {
	r := NumberRenderer[string]{ThousandsSep: true}
	ctx := CellContext[string]{FormattedValue: "1234567", Width: 20}
	got := r.Render(ctx)
	if !strings.Contains(got, "1,234,567") {
		t.Errorf("ThousandsSep=true: expected thousands separator, got %q", got)
	}
}

func TestNumberRendererTruncation(t *testing.T) {
	r := NumberRenderer[string]{}
	// FormattedValue longer than width should be truncated.
	ctx := CellContext[string]{FormattedValue: "123456789", Width: 5}
	got := r.Render(ctx)
	if got != "12345" {
		t.Errorf("truncation: expected %q, got %q", "12345", got)
	}
}

func TestNumberRendererExactWidth(t *testing.T) {
	r := NumberRenderer[string]{}
	ctx := CellContext[string]{FormattedValue: "12345", Width: 5}
	got := r.Render(ctx)
	if got != "12345" {
		t.Errorf("exact width: expected %q, got %q", "12345", got)
	}
}

// --- TimeRenderer ---

func TestTimeRendererRelativeMode(t *testing.T) {
	r := TimeRenderer[string]{Relative: true}
	tm := time.Now().Add(-2 * time.Hour)
	ctx := CellContext[string]{Value: tm, Width: 20}
	got := r.Render(ctx)
	if !strings.Contains(got, "ago") {
		t.Errorf("Relative=true: expected 'ago', got %q", got)
	}
}

func TestTimeRendererNonTimeValue(t *testing.T) {
	r := TimeRenderer[string]{}
	ctx := CellContext[string]{Value: "not a time", FormattedValue: "fallback", Width: 20}
	got := r.Render(ctx)
	if !strings.Contains(got, "fallback") {
		t.Errorf("non-time value: expected fallback, got %q", got)
	}
}

func TestTimeRendererEmptyFormatDefault(t *testing.T) {
	r := TimeRenderer[string]{} // Format is empty, should default to "2006-01-02 15:04"
	tm := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	ctx := CellContext[string]{Value: tm, Width: 30}
	got := r.Render(ctx)
	if !strings.Contains(got, "2024-06-15 10:30") {
		t.Errorf("empty format default: expected '2024-06-15 10:30', got %q", got)
	}
}

// --- BarRenderer ---

func TestBarRendererNegativeRatio(t *testing.T) {
	r := BarRenderer[string]{MaxValue: 100}
	ctx := CellContext[string]{Value: float64(-50), Width: 10}
	got := r.Render(ctx)
	// ratio < 0 clamped to 0, so no bar chars
	barCount := strings.Count(got, "\u2588") // "█"
	if barCount != 0 {
		t.Errorf("negative ratio: expected 0 bar chars, got %d in %q", barCount, got)
	}
}

func TestBarRendererExceedMaxRatio(t *testing.T) {
	r := BarRenderer[string]{MaxValue: 100}
	ctx := CellContext[string]{Value: float64(200), Width: 10}
	got := r.Render(ctx)
	// ratio > 1 clamped to 1, so full bar
	barCount := strings.Count(got, "\u2588")
	if barCount != 10 {
		t.Errorf("ratio > 1: expected 10 bar chars, got %d in %q", barCount, got)
	}
}

func TestBarRendererDefaultMaxValue(t *testing.T) {
	r := BarRenderer[string]{} // MaxValue = 0, defaults to 100
	ctx := CellContext[string]{Value: float64(50), Width: 10}
	got := r.Render(ctx)
	barCount := strings.Count(got, "\u2588")
	if barCount != 5 {
		t.Errorf("default MaxValue: expected 5 bar chars, got %d in %q", barCount, got)
	}
}

func TestBarRendererDefaultBarChar(t *testing.T) {
	r := BarRenderer[string]{MaxValue: 100} // BarChar is empty, defaults to "█"
	ctx := CellContext[string]{Value: float64(30), Width: 10}
	got := r.Render(ctx)
	if !strings.Contains(got, "\u2588") {
		t.Errorf("default BarChar: expected '█' in output, got %q", got)
	}
}

func TestBarRendererCustomBarChar(t *testing.T) {
	r := BarRenderer[string]{MaxValue: 100, BarChar: "#"}
	ctx := CellContext[string]{Value: float64(50), Width: 10}
	got := r.Render(ctx)
	hashCount := strings.Count(got, "#")
	if hashCount != 5 {
		t.Errorf("custom BarChar: expected 5 '#', got %d in %q", hashCount, got)
	}
}

func TestBarRendererStyleNonEmpty(t *testing.T) {
	// SetString makes Style.Value() return a non-empty string, triggering the Style.Render branch.
	style := lipgloss.NewStyle().SetString("bar")
	r := BarRenderer[string]{MaxValue: 100, Style: style}
	ctx := CellContext[string]{Value: float64(50), Width: 10}
	got := r.Render(ctx)
	// Just ensure it doesn't panic and produces some output.
	if len(got) == 0 {
		t.Error("Style non-empty: expected non-empty output")
	}
}

// --- SparklineRenderer ---

func TestSparklineRendererNonSliceValue(t *testing.T) {
	r := SparklineRenderer[string]{}
	ctx := CellContext[string]{Value: "not a slice", FormattedValue: "fallback", Width: 10}
	got := r.Render(ctx)
	if !strings.Contains(got, "fallback") {
		t.Errorf("non-slice value: expected fallback, got %q", got)
	}
}

func TestSparklineRendererEmptyValues(t *testing.T) {
	r := SparklineRenderer[string]{}
	ctx := CellContext[string]{Value: []float64{}, Width: 10}
	got := r.Render(ctx)
	// Empty values should produce empty padded string
	if len(strings.TrimSpace(got)) != 0 {
		t.Errorf("empty values: expected padded empty, got %q", got)
	}
}

func TestSparklineRendererIdxClamp(t *testing.T) {
	// All same values => rangeV=0 => rangeV set to 1 => idx computation could be edge-case
	r := SparklineRenderer[string]{}
	ctx := CellContext[string]{Value: []float64{5, 5, 5}, Width: 10}
	got := r.Render(ctx)
	// All values are the same, so (v - minV) / rangeV * 7 = 0 for all, should get lowest block
	runes := []rune(got)
	for i := 0; i < 3 && i < len(runes); i++ {
		if runes[i] != '\u2581' { // "▁"
			t.Errorf("sparkline idx clamp: expected lowest block at position %d, got %c", i, runes[i])
		}
	}
}

func TestSparklineRendererDecreasingValues(t *testing.T) {
	// This exercises the v < minV branch in the min/max discovery loop.
	r := SparklineRenderer[string]{}
	ctx := CellContext[string]{Value: []float64{10, 5, 1}, Width: 10}
	got := r.Render(ctx)
	runes := []rune(got)
	if len(runes) < 3 {
		t.Fatalf("expected at least 3 runes, got %d", len(runes))
	}
	// First value (10) is max => highest block; last value (1) is min => lowest block
	if runes[0] != '\u2588' { // "█"
		t.Errorf("max value should map to highest block, got %c", runes[0])
	}
	if runes[2] != '\u2581' { // "▁"
		t.Errorf("min value should map to lowest block, got %c", runes[2])
	}
}

// --- ProgressRenderer ---

func TestProgressRendererNegativeRatio(t *testing.T) {
	r := ProgressRenderer[string]{MaxValue: 100}
	ctx := CellContext[string]{Value: float64(-50), Width: 10}
	got := r.Render(ctx)
	// ratio < 0 clamped to 0, all empty chars
	filledCount := strings.Count(got, "\u2501") // "━"
	emptyCount := strings.Count(got, "\u2500")  // "─"
	if filledCount != 0 {
		t.Errorf("negative ratio: expected 0 filled, got %d", filledCount)
	}
	if emptyCount != 10 {
		t.Errorf("negative ratio: expected 10 empty, got %d", emptyCount)
	}
}

func TestProgressRendererExceedMaxRatio(t *testing.T) {
	r := ProgressRenderer[string]{MaxValue: 100}
	ctx := CellContext[string]{Value: float64(200), Width: 10}
	got := r.Render(ctx)
	// ratio > 1 clamped to 1, all filled chars
	filledCount := strings.Count(got, "\u2501")
	emptyCount := strings.Count(got, "\u2500")
	if filledCount != 10 {
		t.Errorf("ratio > 1: expected 10 filled, got %d", filledCount)
	}
	if emptyCount != 0 {
		t.Errorf("ratio > 1: expected 0 empty, got %d", emptyCount)
	}
}

func TestProgressRendererDefaultMaxValue(t *testing.T) {
	r := ProgressRenderer[string]{} // MaxValue = 0, defaults to 100
	ctx := CellContext[string]{Value: float64(50), Width: 10}
	got := r.Render(ctx)
	filledCount := strings.Count(got, "\u2501") // "━"
	emptyCount := strings.Count(got, "\u2500")  // "─"
	if filledCount != 5 {
		t.Errorf("default MaxValue: expected 5 filled, got %d in %q", filledCount, got)
	}
	if emptyCount != 5 {
		t.Errorf("default MaxValue: expected 5 empty, got %d in %q", emptyCount, got)
	}
}

func TestProgressRendererCustomChars(t *testing.T) {
	r := ProgressRenderer[string]{MaxValue: 100, FilledChar: "=", EmptyChar: "."}
	ctx := CellContext[string]{Value: float64(50), Width: 10}
	got := r.Render(ctx)
	eqCount := strings.Count(got, "=")
	dotCount := strings.Count(got, ".")
	if eqCount != 5 {
		t.Errorf("custom chars: expected 5 '=', got %d in %q", eqCount, got)
	}
	if dotCount != 5 {
		t.Errorf("custom chars: expected 5 '.', got %d in %q", dotCount, got)
	}
}

// --- TextEditorModel.View ---

func TestTextEditorView(t *testing.T) {
	e := NewTextEditor[string]()
	ctx := CellContext[string]{FormattedValue: "hello", Width: 20}
	e.Init(ctx)
	got := e.View()
	if len(got) == 0 {
		t.Error("TextEditorModel.View: expected non-empty output")
	}
	// Cursor should produce reverse video markers somewhere
	if !strings.Contains(got, testReverseOn) {
		t.Error("TextEditorModel.View: expected reverse video in output")
	}
}

// --- NumberEditorModel ---

func TestNumberEditorWithStep(t *testing.T) {
	e := NewNumberEditor[string]().WithStep(5)
	ctx := CellContext[string]{Value: float64(10), Width: 20}
	e.Init(ctx)
	e.Update(tea.KeyMsg{Type: tea.KeyUp})
	v := e.Value().(float64)
	if v != 15 {
		t.Errorf("WithStep(5) + Up: expected 15, got %v", v)
	}
	e.Update(tea.KeyMsg{Type: tea.KeyDown})
	v = e.Value().(float64)
	if v != 10 {
		t.Errorf("WithStep(5) + Down: expected 10, got %v", v)
	}
}

func TestNumberEditorView(t *testing.T) {
	e := NewNumberEditor[string]()
	ctx := CellContext[string]{Value: float64(42), Width: 20}
	e.Init(ctx)
	got := e.View()
	if got != "42" {
		t.Errorf("NumberEditorModel.View: expected %q, got %q", "42", got)
	}
}

// --- SelectEditorModel ---

func TestSelectEditorKeyLeftWrapPastZero(t *testing.T) {
	e := NewSelectEditor[string]([]string{"a", "b", "c"})
	ctx := CellContext[string]{Value: "a", Width: 20}
	e.Init(ctx)
	// index starts at 0 for "a"
	// KeyLeft at 0 should wrap to len(options)-1
	e.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if e.Value() != "c" {
		t.Errorf("KeyLeft from 0: expected wrap to 'c', got %v", e.Value())
	}
}

func TestSelectEditorKeyLeftDecrement(t *testing.T) {
	// Start at index > 0 and KeyLeft to decrement (not wrap).
	e := NewSelectEditor[string]([]string{"a", "b", "c"})
	ctx := CellContext[string]{Value: "b", Width: 20}
	e.Init(ctx)
	// index starts at 1 for "b"
	e.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if e.Value() != "a" {
		t.Errorf("KeyLeft from b: expected 'a', got %v", e.Value())
	}
}

func TestSelectEditorViewEmpty(t *testing.T) {
	e := NewSelectEditor[string]([]string{})
	ctx := CellContext[string]{Value: "", Width: 20}
	e.Init(ctx)
	got := e.View()
	if got != "" {
		t.Errorf("empty options View: expected empty, got %q", got)
	}
}

func TestSelectEditorViewNonEmpty(t *testing.T) {
	e := NewSelectEditor[string]([]string{"x", "y", "z"})
	ctx := CellContext[string]{Value: "y", Width: 20}
	e.Init(ctx)
	got := e.View()
	if got != "y" {
		t.Errorf("View: expected 'y', got %q", got)
	}
}

func TestSelectEditorValueEmpty(t *testing.T) {
	e := NewSelectEditor[string]([]string{})
	ctx := CellContext[string]{Value: "", Width: 20}
	e.Init(ctx)
	got := e.Value()
	if got != "" {
		t.Errorf("empty options Value: expected empty string, got %v", got)
	}
}

func TestSelectEditorValidate(t *testing.T) {
	e := NewSelectEditor[string]([]string{"a", "b"})
	ctx := CellContext[string]{Value: "a", Width: 20}
	e.Init(ctx)
	if msg := e.Validate(); msg != "" {
		t.Errorf("SelectEditor.Validate should return empty, got %q", msg)
	}
}

// --- BoolEditorModel ---

func TestBoolEditorViewTrue(t *testing.T) {
	e := NewBoolEditor[string]()
	ctx := CellContext[string]{Value: true, Width: 20}
	e.Init(ctx)
	got := e.View()
	if got != "true" {
		t.Errorf("BoolEditor.View(true): expected 'true', got %q", got)
	}
}

func TestBoolEditorViewFalse(t *testing.T) {
	e := NewBoolEditor[string]()
	ctx := CellContext[string]{Value: false, Width: 20}
	e.Init(ctx)
	got := e.View()
	if got != "false" {
		t.Errorf("BoolEditor.View(false): expected 'false', got %q", got)
	}
}

func TestBoolEditorValidate(t *testing.T) {
	e := NewBoolEditor[string]()
	ctx := CellContext[string]{Value: true, Width: 20}
	e.Init(ctx)
	if msg := e.Validate(); msg != "" {
		t.Errorf("BoolEditor.Validate should return empty, got %q", msg)
	}
}

// --- TimeEditorModel ---

func TestTimeEditorInitNonTimeValue(t *testing.T) {
	e := NewTimeEditor[string]()
	ctx := CellContext[string]{Value: "not a time", FormattedValue: "formatted-str", Width: 30}
	e.Init(ctx)
	if e.editor.Text() != "formatted-str" {
		t.Errorf("Init non-time: expected FormattedValue, got %q", e.editor.Text())
	}
}

func TestTimeEditorUpdate(t *testing.T) {
	e := NewTimeEditor[string]()
	tm := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	ctx := CellContext[string]{Value: tm, Width: 30}
	e.Init(ctx)

	// Type a character
	e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	text := e.editor.Text()
	if !strings.Contains(text, "X") {
		t.Errorf("Update should pass key to editor, got %q", text)
	}

	// Set parseErr manually and verify Update clears it
	e.parseErr = "some error"
	e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	if e.parseErr != "" {
		t.Errorf("Update should clear parseErr, got %q", e.parseErr)
	}
}

func TestTimeEditorView(t *testing.T) {
	e := NewTimeEditor[string]()
	tm := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	ctx := CellContext[string]{Value: tm, Width: 30}
	e.Init(ctx)
	got := e.View()
	if len(got) == 0 {
		t.Error("TimeEditor.View: expected non-empty output")
	}
}

func TestTimeEditorViewWithParseErr(t *testing.T) {
	e := NewTimeEditor[string]()
	tm := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	ctx := CellContext[string]{Value: tm, Width: 50}
	e.Init(ctx)
	e.parseErr = "invalid time format"
	got := e.View()
	if !strings.Contains(got, "invalid time format") {
		t.Errorf("View with parseErr: expected error suffix, got %q", got)
	}
}

func TestTimeEditorValueNoValidFormat(t *testing.T) {
	e := NewTimeEditor[string]()
	e.editor.SetText("gibberish not a date")
	v := e.Value()
	tv, ok := v.(time.Time)
	if !ok {
		t.Fatalf("Value should return time.Time, got %T", v)
	}
	if !tv.IsZero() {
		t.Errorf("Value with invalid text should return zero time, got %v", tv)
	}
}

// --- addThousandsSep edge case ---

func TestAddThousandsSepNegativeWithDecimal(t *testing.T) {
	got := addThousandsSep("-1234567.89")
	if got != "-1,234,567.89" {
		t.Errorf("negative with decimal: expected '-1,234,567.89', got %q", got)
	}
}

func TestAddThousandsSepNegativeSmallWithDecimal(t *testing.T) {
	// Negative number with <= 3 digits and decimal
	got := addThousandsSep("-123.45")
	if got != "-123.45" {
		t.Errorf("negative small with decimal: expected '-123.45', got %q", got)
	}
}

// --- Non-KeyMsg to Update ---

func TestTimeEditorUpdateNonKeyMsg(t *testing.T) {
	e := NewTimeEditor[string]()
	tm := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	ctx := CellContext[string]{Value: tm, Width: 30}
	e.Init(ctx)
	textBefore := e.editor.Text()
	// Send a non-KeyMsg (e.g. a tea.WindowSizeMsg)
	e.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	textAfter := e.editor.Text()
	if textBefore != textAfter {
		t.Errorf("non-KeyMsg should not change text: before=%q, after=%q", textBefore, textAfter)
	}
}

func TestSelectEditorUpdateNonKeyMsg(t *testing.T) {
	e := NewSelectEditor[string]([]string{"a", "b"})
	ctx := CellContext[string]{Value: "a", Width: 20}
	e.Init(ctx)
	e.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if e.Value() != "a" {
		t.Errorf("non-KeyMsg should not change selection, got %v", e.Value())
	}
}

func TestBoolEditorUpdateNonKeyMsg(t *testing.T) {
	e := NewBoolEditor[string]()
	ctx := CellContext[string]{Value: true, Width: 20}
	e.Init(ctx)
	e.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if e.Value() != true {
		t.Errorf("non-KeyMsg should not change value, got %v", e.Value())
	}
}

func TestNumberEditorUpdateNonKeyMsg(t *testing.T) {
	e := NewNumberEditor[string]()
	ctx := CellContext[string]{Value: float64(42), Width: 20}
	e.Init(ctx)
	e.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if e.Value().(float64) != 42 {
		t.Errorf("non-KeyMsg should not change value, got %v", e.Value())
	}
}

func TestTextEditorUpdateNonKeyMsg(t *testing.T) {
	e := NewTextEditor[string]()
	ctx := CellContext[string]{FormattedValue: "hello", Width: 20}
	e.Init(ctx)
	e.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if e.Value() != "hello" {
		t.Errorf("non-KeyMsg should not change value, got %v", e.Value())
	}
}

// --- SelectEditor KeyRight cycling ---

func TestSelectEditorKeyRightCycle(t *testing.T) {
	e := NewSelectEditor[string]([]string{"a", "b", "c"})
	ctx := CellContext[string]{Value: "a", Width: 20}
	e.Init(ctx)
	e.Update(tea.KeyMsg{Type: tea.KeyRight})
	if e.Value() != "b" {
		t.Errorf("KeyRight from a: expected b, got %v", e.Value())
	}
}

// --- BoolEditor toggle with Up/Down ---

func TestBoolEditorToggleUpDown(t *testing.T) {
	e := NewBoolEditor[string]()
	ctx := CellContext[string]{Value: false, Width: 20}
	e.Init(ctx)
	e.Update(tea.KeyMsg{Type: tea.KeyUp})
	if e.Value() != true {
		t.Errorf("KeyUp toggle: expected true, got %v", e.Value())
	}
	e.Update(tea.KeyMsg{Type: tea.KeyDown})
	if e.Value() != false {
		t.Errorf("KeyDown toggle: expected false, got %v", e.Value())
	}
}
