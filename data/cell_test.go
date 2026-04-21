package data

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// ANSI escape sequences used in tests to match rendered output.
const (
	testReverseOn  = "\x1b[7m"
	testReverseOff = "\x1b[27m"
)

// --- TruncateOrPad ---

func TestTruncateOrPadZeroWidth(t *testing.T) {
	if got := TruncateOrPad("hello", 0); got != "" {
		t.Errorf("width=0: expected empty, got %q", got)
	}
}

func TestTruncateOrPadShorter(t *testing.T) {
	got := TruncateOrPad("hi", 5)
	if got != "hi   " {
		t.Errorf("expected %q, got %q", "hi   ", got)
	}
}

func TestTruncateOrPadExact(t *testing.T) {
	got := TruncateOrPad("hello", 5)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

func TestTruncateOrPadLonger(t *testing.T) {
	got := TruncateOrPad("hello world", 5)
	if got != "hell…" {
		t.Errorf("expected %q, got %q", "hell…", got)
	}
}

func TestTruncateOrPadWidth1(t *testing.T) {
	got := TruncateOrPad("hello", 1)
	if got != "h" {
		t.Errorf("width=1: expected %q, got %q", "h", got)
	}
}

func TestTruncateOrPadUnicode(t *testing.T) {
	got := TruncateOrPad("héllo wörld", 6)
	if got != "héllo…" {
		t.Errorf("expected %q, got %q", "héllo…", got)
	}
}

// --- RenderEditorLine ---

func TestRenderEditorLineCursorAtStart(t *testing.T) {
	result := RenderEditorLine("abc", 0, 10, "")
	// The first character 'a' should be wrapped in reverse video
	if !strings.Contains(result, testReverseOn+"a"+testReverseOff) {
		t.Errorf("cursor at 0: expected 'a' in reverse, got %q", result)
	}
}

func TestRenderEditorLineCursorAtEnd(t *testing.T) {
	result := RenderEditorLine("abc", 3, 10, "")
	// Cursor at end shows space in reverse
	if !strings.Contains(result, testReverseOn+" "+testReverseOff) {
		t.Errorf("cursor at end: expected space in reverse, got %q", result)
	}
}

func TestRenderEditorLineCursorMiddle(t *testing.T) {
	result := RenderEditorLine("abc", 1, 10, "")
	if !strings.Contains(result, testReverseOn+"b"+testReverseOff) {
		t.Errorf("cursor at 1: expected 'b' in reverse, got %q", result)
	}
}

func TestRenderEditorLineZeroWidth(t *testing.T) {
	result := RenderEditorLine("abc", 0, 0, "")
	if result != "" {
		t.Errorf("width=0: expected empty, got %q", result)
	}
}

func TestRenderEditorLineEmptyString(t *testing.T) {
	result := RenderEditorLine("", 0, 5, "")
	// Should show space cursor plus padding
	if !strings.Contains(result, testReverseOn+" "+testReverseOff) {
		t.Errorf("empty string: expected space cursor, got %q", result)
	}
}

// --- addThousandsSep ---

func TestAddThousandsSep(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1234", "1,234"},
		{"1234567", "1,234,567"},
		{"123", "123"},
		{"-1234", "-1,234"},
		{"1234.56", "1,234.56"},
		{"0", "0"},
		{"12", "12"},
		{"-123", "-123"},
		{"1000000", "1,000,000"},
	}
	for _, tt := range tests {
		got := addThousandsSep(tt.input)
		if got != tt.want {
			t.Errorf("addThousandsSep(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- toFloat64 ---

func TestToFloat64(t *testing.T) {
	tests := []struct {
		input any
		want  float64
	}{
		{int(42), 42},
		{int8(8), 8},
		{int16(16), 16},
		{int32(32), 32},
		{int64(64), 64},
		{uint(10), 10},
		{uint8(8), 8},
		{uint16(16), 16},
		{uint32(32), 32},
		{uint64(64), 64},
		{float32(3.14), float64(float32(3.14))},
		{float64(2.718), 2.718},
		{"unknown", 0},
		{nil, 0},
	}
	for _, tt := range tests {
		got := toFloat64(tt.input)
		if got != tt.want {
			t.Errorf("toFloat64(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- relativeTime / formatDuration ---

func TestFormatDurationSeconds(t *testing.T) {
	d := 30 * time.Second
	got := formatDuration(d)
	if got != "30s" {
		t.Errorf("expected 30s, got %s", got)
	}
}

func TestFormatDurationMinutes(t *testing.T) {
	d := 5 * time.Minute
	got := formatDuration(d)
	if got != "5m" {
		t.Errorf("expected 5m, got %s", got)
	}
}

func TestFormatDurationHours(t *testing.T) {
	d := 3 * time.Hour
	got := formatDuration(d)
	if got != "3h" {
		t.Errorf("expected 3h, got %s", got)
	}
}

func TestFormatDurationDays(t *testing.T) {
	d := 5 * 24 * time.Hour
	got := formatDuration(d)
	if got != "5d" {
		t.Errorf("expected 5d, got %s", got)
	}
}

func TestFormatDurationMonths(t *testing.T) {
	d := 60 * 24 * time.Hour
	got := formatDuration(d)
	if got != "2mo" {
		t.Errorf("expected 2mo, got %s", got)
	}
}

func TestFormatDurationYears(t *testing.T) {
	d := 400 * 24 * time.Hour
	got := formatDuration(d)
	if got != "1y" {
		t.Errorf("expected 1y, got %s", got)
	}
}

func TestRelativeTimePast(t *testing.T) {
	past := time.Now().Add(-2 * time.Hour)
	got := relativeTime(past)
	if !strings.HasSuffix(got, " ago") {
		t.Errorf("past time should end with ' ago', got %q", got)
	}
}

func TestRelativeTimeFuture(t *testing.T) {
	future := time.Now().Add(2 * time.Hour)
	got := relativeTime(future)
	if !strings.HasPrefix(got, "in ") {
		t.Errorf("future time should start with 'in ', got %q", got)
	}
}

// --- Renderers ---

func TestTextRendererBasic(t *testing.T) {
	r := TextRenderer[string]{}
	ctx := CellContext[string]{FormattedValue: "hello", Width: 10}
	got := r.Render(ctx)
	if got != "hello     " {
		t.Errorf("expected padded output, got %q", got)
	}
}

func TestTextRendererZeroWidth(t *testing.T) {
	r := TextRenderer[string]{}
	ctx := CellContext[string]{FormattedValue: "hello", Width: 0}
	got := r.Render(ctx)
	if got != "" {
		t.Errorf("width=0: expected empty, got %q", got)
	}
}

func TestNumberRendererRightAligned(t *testing.T) {
	r := NumberRenderer[string]{}
	ctx := CellContext[string]{FormattedValue: "42", Width: 6}
	got := r.Render(ctx)
	if got != "    42" {
		t.Errorf("expected right-aligned, got %q", got)
	}
}

func TestNumberRendererThousandsSep(t *testing.T) {
	r := NumberRenderer[string]{ThousandsSep: true}
	ctx := CellContext[string]{FormattedValue: "1234567", Width: 12}
	got := r.Render(ctx)
	if !strings.Contains(got, "1,234,567") {
		t.Errorf("expected thousands sep, got %q", got)
	}
}

func TestBoolRendererDefaults(t *testing.T) {
	r := BoolRenderer[string]{}
	ctxTrue := CellContext[string]{Value: true, Width: 5}
	ctxFalse := CellContext[string]{Value: false, Width: 5}

	gotTrue := r.Render(ctxTrue)
	if !strings.Contains(gotTrue, "✓") {
		t.Errorf("true: expected checkmark, got %q", gotTrue)
	}
	gotFalse := r.Render(ctxFalse)
	if !strings.Contains(gotFalse, "✗") {
		t.Errorf("false: expected cross, got %q", gotFalse)
	}
}

func TestBoolRendererCustomGlyphs(t *testing.T) {
	r := BoolRenderer[string]{TrueGlyph: "Y", FalseGlyph: "N"}
	ctx := CellContext[string]{Value: true, Width: 5}
	got := r.Render(ctx)
	if !strings.Contains(got, "Y") {
		t.Errorf("expected Y, got %q", got)
	}
}

func TestBoolRendererNonBool(t *testing.T) {
	r := BoolRenderer[string]{}
	ctx := CellContext[string]{Value: "notbool", FormattedValue: "notbool", Width: 10}
	got := r.Render(ctx)
	if !strings.Contains(got, "notbool") {
		t.Errorf("non-bool should fallback to formatted, got %q", got)
	}
}

func TestTimeRendererFormat(t *testing.T) {
	r := TimeRenderer[string]{Format: "2006-01-02"}
	tm := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	ctx := CellContext[string]{Value: tm, Width: 15}
	got := r.Render(ctx)
	if !strings.Contains(got, "2024-06-15") {
		t.Errorf("expected formatted date, got %q", got)
	}
}

func TestTimeRendererRelative(t *testing.T) {
	r := TimeRenderer[string]{Relative: true}
	tm := time.Now().Add(-2 * time.Hour)
	ctx := CellContext[string]{Value: tm, Width: 15}
	got := r.Render(ctx)
	if !strings.Contains(got, "ago") {
		t.Errorf("expected relative time, got %q", got)
	}
}

func TestBarRendererProportional(t *testing.T) {
	r := BarRenderer[string]{MaxValue: 100}
	ctx := CellContext[string]{Value: float64(50), Width: 10}
	got := r.Render(ctx)
	// 50% of 10 = 5 bar characters
	barCount := strings.Count(got, "█")
	if barCount != 5 {
		t.Errorf("expected 5 bar chars, got %d in %q", barCount, got)
	}
}

func TestBarRendererZeroWidth(t *testing.T) {
	r := BarRenderer[string]{MaxValue: 100}
	ctx := CellContext[string]{Value: float64(50), Width: 0}
	got := r.Render(ctx)
	if got != "" {
		t.Errorf("width=0: expected empty, got %q", got)
	}
}

func TestSparklineRenderer(t *testing.T) {
	r := SparklineRenderer[string]{}
	ctx := CellContext[string]{Value: []float64{0, 0.5, 1.0}, Width: 10}
	got := r.Render(ctx)
	// Should contain block characters
	if len(got) == 0 {
		t.Error("expected non-empty sparkline")
	}
	// First value (min) should be lowest block
	runes := []rune(got)
	if runes[0] != '▁' {
		t.Errorf("min value should map to ▁, got %c", runes[0])
	}
}

func TestSparklineRendererNonSlice(t *testing.T) {
	r := SparklineRenderer[string]{}
	ctx := CellContext[string]{Value: "notslice", FormattedValue: "notslice", Width: 10}
	got := r.Render(ctx)
	if !strings.Contains(got, "notslice") {
		t.Errorf("non-slice should fallback, got %q", got)
	}
}

func TestProgressRenderer(t *testing.T) {
	r := ProgressRenderer[string]{MaxValue: 100}
	ctx := CellContext[string]{Value: float64(50), Width: 10}
	got := r.Render(ctx)
	filled := strings.Count(got, "━")
	empty := strings.Count(got, "─")
	if filled+empty != 10 {
		t.Errorf("expected 10 total chars, got %d filled + %d empty in %q", filled, empty, got)
	}
	if filled != 5 {
		t.Errorf("50%%: expected 5 filled, got %d", filled)
	}
}

// --- Editors ---

func TestTextEditorInit(t *testing.T) {
	e := NewTextEditor[string]()
	ctx := CellContext[string]{FormattedValue: "hello", Width: 20}
	e.Init(ctx)
	if e.Value() != "hello" {
		t.Errorf("Init: value should be 'hello', got %v", e.Value())
	}
}

func TestTextEditorRuneInput(t *testing.T) {
	e := NewTextEditor[string]()
	ctx := CellContext[string]{FormattedValue: "", Width: 20}
	e.Init(ctx)
	e.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	e.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	if e.Value() != "ab" {
		t.Errorf("after typing 'ab', got %v", e.Value())
	}
}

func TestTextEditorBackspace(t *testing.T) {
	e := NewTextEditor[string]()
	ctx := CellContext[string]{FormattedValue: "hello", Width: 20}
	e.Init(ctx)
	e.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if e.Value() != "hell" {
		t.Errorf("after backspace, got %v", e.Value())
	}
}

func TestTextEditorDelete(t *testing.T) {
	e := NewTextEditor[string]()
	ctx := CellContext[string]{FormattedValue: "hello", Width: 20}
	e.Init(ctx)
	// Move cursor to start
	e.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	e.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	if e.Value() != "ello" {
		t.Errorf("after delete at start, got %v", e.Value())
	}
}

func TestTextEditorCursorMovement(t *testing.T) {
	e := NewTextEditor[string]()
	ctx := CellContext[string]{FormattedValue: "abc", Width: 20}
	e.Init(ctx)
	// Cursor starts at end (3)
	e.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	e.Update(tea.KeyPressMsg{Code: 'X', Text: "X"})
	if e.Value() != "abXc" {
		t.Errorf("insert at cursor: got %v", e.Value())
	}
}

func TestTextEditorHomeEnd(t *testing.T) {
	e := NewTextEditor[string]()
	ctx := CellContext[string]{FormattedValue: "hello", Width: 20}
	e.Init(ctx)
	e.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	e.Update(tea.KeyPressMsg{Code: 'X', Text: "X"})
	if e.Value() != "Xhello" {
		t.Errorf("after Home+type: got %v", e.Value())
	}
}

func TestTextEditorValidate(t *testing.T) {
	e := NewTextEditor[string]()
	ctx := CellContext[string]{FormattedValue: "anything", Width: 20}
	e.Init(ctx)
	if msg := e.Validate(); msg != "" {
		t.Errorf("TextEditor Validate should always return empty, got %q", msg)
	}
}

// NumberEditor

func TestNumberEditorInit(t *testing.T) {
	e := NewNumberEditor[string]()
	ctx := CellContext[string]{Value: float64(42), Width: 20}
	e.Init(ctx)
	if e.Value() != float64(42) {
		t.Errorf("Init: value should be 42, got %v", e.Value())
	}
}

func TestNumberEditorOnlyDigits(t *testing.T) {
	e := NewNumberEditor[string]()
	ctx := CellContext[string]{Value: float64(0), Width: 20}
	e.Init(ctx)
	// Clear existing text
	for range len(e.text) {
		e.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	e.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	e.Update(tea.KeyPressMsg{Code: 'a', Text: "a"}) // should be ignored
	e.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	if e.text != "12" {
		t.Errorf("non-digit should be rejected, got text=%q", e.text)
	}
}

func TestNumberEditorUpDown(t *testing.T) {
	e := NewNumberEditor[string]()
	ctx := CellContext[string]{Value: float64(10), Width: 20}
	e.Init(ctx)
	e.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	v := e.Value().(float64)
	if v != 11 {
		t.Errorf("Up should increment by 1, got %v", v)
	}
	e.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	e.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	v = e.Value().(float64)
	if v != 9 {
		t.Errorf("Down twice should give 9, got %v", v)
	}
}

func TestNumberEditorValidateRejectsNonNumeric(t *testing.T) {
	e := NewNumberEditor[string]()
	e.text = "abc"
	if msg := e.Validate(); msg == "" {
		t.Error("should reject non-numeric text")
	}
}

func TestNumberEditorValidateMinMax(t *testing.T) {
	e := NewNumberEditor[string]()
	e.WithMin(0).WithMax(100)
	e.text = "-5"
	if msg := e.Validate(); msg == "" {
		t.Error("should reject below min")
	}
	e.text = "150"
	if msg := e.Validate(); msg == "" {
		t.Error("should reject above max")
	}
	e.text = "50"
	if msg := e.Validate(); msg != "" {
		t.Errorf("50 should be valid, got %q", msg)
	}
}

// SelectEditor

func TestSelectEditorInit(t *testing.T) {
	e := NewSelectEditor[string]([]string{"a", "b", "c"})
	ctx := CellContext[string]{Value: "b", Width: 20}
	e.Init(ctx)
	if e.Value() != "b" {
		t.Errorf("Init: should find matching option, got %v", e.Value())
	}
}

func TestSelectEditorCycleForward(t *testing.T) {
	e := NewSelectEditor[string]([]string{"a", "b", "c"})
	ctx := CellContext[string]{Value: "a", Width: 20}
	e.Init(ctx)
	e.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if e.Value() != "b" {
		t.Errorf("Down from a: expected b, got %v", e.Value())
	}
}

func TestSelectEditorCycleBackward(t *testing.T) {
	e := NewSelectEditor[string]([]string{"a", "b", "c"})
	ctx := CellContext[string]{Value: "a", Width: 20}
	e.Init(ctx)
	e.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if e.Value() != "c" {
		t.Errorf("Up from a (wrap): expected c, got %v", e.Value())
	}
}

func TestSelectEditorWrapForward(t *testing.T) {
	e := NewSelectEditor[string]([]string{"a", "b", "c"})
	ctx := CellContext[string]{Value: "c", Width: 20}
	e.Init(ctx)
	e.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if e.Value() != "a" {
		t.Errorf("Down from c (wrap): expected a, got %v", e.Value())
	}
}

// BoolEditor

func TestBoolEditorInit(t *testing.T) {
	e := NewBoolEditor[string]()
	ctx := CellContext[string]{Value: true, Width: 20}
	e.Init(ctx)
	if e.Value() != true {
		t.Errorf("Init: expected true, got %v", e.Value())
	}
}

func TestBoolEditorToggle(t *testing.T) {
	e := NewBoolEditor[string]()
	ctx := CellContext[string]{Value: false, Width: 20}
	e.Init(ctx)
	e.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if e.Value() != true {
		t.Errorf("after toggle: expected true, got %v", e.Value())
	}
	e.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if e.Value() != false {
		t.Errorf("after second toggle: expected false, got %v", e.Value())
	}
}

// TimeEditor

func TestTimeEditorInit(t *testing.T) {
	e := NewTimeEditor[string]()
	tm := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	ctx := CellContext[string]{Value: tm, Width: 30}
	e.Init(ctx)
	if e.editor.Text() != "2024-06-15 10:30" {
		t.Errorf("Init: expected formatted time, got %q", e.editor.Text())
	}
}

func TestTimeEditorValueParsesBack(t *testing.T) {
	e := NewTimeEditor[string]()
	tm := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	ctx := CellContext[string]{Value: tm, Width: 30}
	e.Init(ctx)
	val := e.Value().(time.Time)
	if val.Year() != 2024 || val.Month() != time.June || val.Day() != 15 {
		t.Errorf("Value parse: expected 2024-06-15, got %v", val)
	}
}

func TestTimeEditorValidateGood(t *testing.T) {
	e := NewTimeEditor[string]()
	e.editor.SetText("2024-06-15")
	if msg := e.Validate(); msg != "" {
		t.Errorf("valid date should pass, got %q", msg)
	}
}

func TestTimeEditorValidateBad(t *testing.T) {
	e := NewTimeEditor[string]()
	e.editor.SetText("not a date")
	if msg := e.Validate(); msg == "" {
		t.Error("invalid date should fail validation")
	}
}

func TestTimeEditorMultipleFormats(t *testing.T) {
	formats := []string{
		"2024-06-15",
		"2024-06-15 10:30",
		"Jun 15, 2024",
		"06/15/2024",
	}
	for _, f := range formats {
		e := NewTimeEditor[string]()
		e.editor.SetText(f)
		if msg := e.Validate(); msg != "" {
			t.Errorf("format %q should be valid, got %q", f, msg)
		}
	}
}

func TestRowNodeZeroValue(t *testing.T) {
	var rn RowNode[string]
	if rn.IsGroup {
		t.Error("zero value IsGroup should be false")
	}
	if rn.Expanded {
		t.Error("zero value Expanded should be false")
	}
	if rn.GroupLevel != 0 {
		t.Error("zero value GroupLevel should be 0")
	}
	if rn.Children != nil {
		t.Error("zero value Children should be nil")
	}
	if rn.Parent != nil {
		t.Error("zero value Parent should be nil")
	}
}

func TestRowNodeTreeStructure(t *testing.T) {
	parent := &RowNode[string]{
		ID:      "parent",
		IsGroup: true,
	}

	child1 := &RowNode[string]{ID: "c1", Data: "child1", Parent: parent}
	child2 := &RowNode[string]{ID: "c2", Data: "child2", Parent: parent}
	parent.Children = []*RowNode[string]{child1, child2}

	if len(parent.Children) != 2 {
		t.Fatalf("parent should have 2 children, got %d", len(parent.Children))
	}
	if child1.Parent != parent {
		t.Error("child1 Parent should point to parent")
	}
	if child2.Parent != parent {
		t.Error("child2 Parent should point to parent")
	}
	if parent.Children[0].Data != "child1" {
		t.Error("first child data mismatch")
	}
	if parent.Children[1].Data != "child2" {
		t.Error("second child data mismatch")
	}
}

// --- NaturalWidthRenderer ---

// naturalWidthTestRenderer is a test renderer that implements
// both CellRenderer and NaturalWidthRenderer.
type naturalWidthTestRenderer struct {
	natural int
}

func (r naturalWidthTestRenderer) Render(ctx CellContext[int]) string {
	return "x"
}

func (r naturalWidthTestRenderer) NaturalWidth(ctx CellContext[int]) int {
	return r.natural
}

func TestNaturalWidthRenderer_InterfaceSatisfied(t *testing.T) {
	var r CellRenderer[int] = naturalWidthTestRenderer{natural: 7}
	nw, ok := r.(NaturalWidthRenderer[int])
	if !ok {
		t.Fatal("expected renderer to satisfy NaturalWidthRenderer[int]")
	}
	if got := nw.NaturalWidth(CellContext[int]{}); got != 7 {
		t.Errorf("NaturalWidth = %d, want 7", got)
	}
}
