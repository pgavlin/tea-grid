package data

import (
	"testing"
	"time"

	"github.com/pgavlin/tea-grid/filter"
)

// --- Test types ---

type Person struct {
	Name string
	Age  int
}

type mixedFields struct {
	Exported   string
	unexported string
	Another    int
}

type multiType struct {
	S string
	I int
	F float64
	B bool
	T time.Time
}

// --- Constants ---

func TestPinConstants(t *testing.T) {
	if PinNone == PinLeft || PinNone == PinRight || PinLeft == PinRight {
		t.Fatal("Pin constants must be distinct")
	}
	if PinNone == PinTop || PinNone == PinBottom || PinTop == PinBottom {
		t.Fatal("Pin constants must be distinct")
	}
}

func TestSortDirectionConstants(t *testing.T) {
	if SortNone == SortAsc || SortNone == SortDesc || SortAsc == SortDesc {
		t.Fatal("SortDirection constants must be distinct")
	}
}

// --- FromType[T]() ---

func TestFromTypeExportedFields(t *testing.T) {
	cols := FromType[Person]()
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}

	if cols[0].ColumnID != "Name" || cols[0].HeaderName != "Name" {
		t.Errorf("first col: got ColumnID=%q HeaderName=%q", cols[0].ColumnID, cols[0].HeaderName)
	}
	if cols[1].ColumnID != "Age" || cols[1].HeaderName != "Age" {
		t.Errorf("second col: got ColumnID=%q HeaderName=%q", cols[1].ColumnID, cols[1].HeaderName)
	}

	for _, c := range cols {
		if !c.Sortable {
			t.Errorf("col %q: Sortable should be true", c.ColumnID)
		}
		if !c.Filterable {
			t.Errorf("col %q: Filterable should be true", c.ColumnID)
		}
		if c.MinWidth != 4 {
			t.Errorf("col %q: MinWidth should be 4, got %d", c.ColumnID, c.MinWidth)
		}
	}
}

func TestFromTypeValueGetter(t *testing.T) {
	cols := FromType[Person]()
	p := Person{Name: "Alice", Age: 30}

	name := cols[0].ValueGetter(p)
	if name != "Alice" {
		t.Errorf("Name getter: expected Alice, got %v", name)
	}

	age := cols[1].ValueGetter(p)
	if age != 30 {
		t.Errorf("Age getter: expected 30, got %v", age)
	}
}

func TestFromTypePointerType(t *testing.T) {
	cols := FromType[*Person]()
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns for *Person, got %d", len(cols))
	}

	p := &Person{Name: "Bob", Age: 25}
	name := cols[0].ValueGetter(p)
	if name != "Bob" {
		t.Errorf("pointer Name getter: expected Bob, got %v", name)
	}
}

func TestFromTypeNonStructPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for non-struct type")
		}
	}()
	FromType[string]()
}

func TestFromTypeEmptyStruct(t *testing.T) {
	cols := FromType[struct{}]()
	if len(cols) != 0 {
		t.Fatalf("expected 0 columns for empty struct, got %d", len(cols))
	}
}

func TestFromTypeUnexportedSkipped(t *testing.T) {
	cols := FromType[mixedFields]()
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns (exported only), got %d", len(cols))
	}
	if cols[0].ColumnID != "Exported" {
		t.Errorf("first col should be Exported, got %q", cols[0].ColumnID)
	}
	if cols[1].ColumnID != "Another" {
		t.Errorf("second col should be Another, got %q", cols[1].ColumnID)
	}
}

func TestFromTypeMultipleFieldTypes(t *testing.T) {
	cols := FromType[multiType]()
	if len(cols) != 5 {
		t.Fatalf("expected 5 columns, got %d", len(cols))
	}

	now := time.Now()
	row := multiType{S: "hello", I: 42, F: 3.14, B: true, T: now}

	tests := []struct {
		idx  int
		want any
	}{
		{0, "hello"},
		{1, 42},
		{2, 3.14},
		{3, true},
		{4, now},
	}

	for _, tt := range tests {
		got := cols[tt.idx].ValueGetter(row)
		if got != tt.want {
			t.Errorf("col %d: got %v, want %v", tt.idx, got, tt.want)
		}
	}
}

func TestFromTypeFilterString(t *testing.T) {
	cols := FromType[Person]()
	if _, ok := cols[0].Filter.(*filter.TextFilter); !ok {
		t.Errorf("Name (string): expected TextFilter, got %T", cols[0].Filter)
	}
}

func TestFromTypeFilterInt(t *testing.T) {
	cols := FromType[Person]()
	if _, ok := cols[1].Filter.(*filter.NumberFilter); !ok {
		t.Errorf("Age (int): expected NumberFilter, got %T", cols[1].Filter)
	}
}

type filterTestUint struct {
	Count uint64
}

func TestFromTypeFilterUint(t *testing.T) {
	cols := FromType[filterTestUint]()
	if _, ok := cols[0].Filter.(*filter.NumberFilter); !ok {
		t.Errorf("Count (uint64): expected NumberFilter, got %T", cols[0].Filter)
	}
}

func TestFromTypeFilterAllTypes(t *testing.T) {
	cols := FromType[multiType]()

	tests := []struct {
		idx      int
		name     string
		wantType string
	}{
		{0, "S (string)", "*filter.TextFilter"},
		{1, "I (int)", "*filter.NumberFilter"},
		{2, "F (float64)", "*filter.NumberFilter"},
		{3, "B (bool)", "*filter.BoolFilter"},
		{4, "T (time.Time)", "*filter.TimeFilter"},
	}

	for _, tt := range tests {
		var ok bool
		switch tt.idx {
		case 0:
			_, ok = cols[tt.idx].Filter.(*filter.TextFilter)
		case 1, 2:
			_, ok = cols[tt.idx].Filter.(*filter.NumberFilter)
		case 3:
			_, ok = cols[tt.idx].Filter.(*filter.BoolFilter)
		case 4:
			_, ok = cols[tt.idx].Filter.(*filter.TimeFilter)
		}
		if !ok {
			t.Errorf("%s: expected %s, got %T", tt.name, tt.wantType, cols[tt.idx].Filter)
		}
	}
}

// --- FromRows tests ---

func TestFromRowsMapStringAny(t *testing.T) {
	rows := []map[string]any{
		{"name": "Alice", "age": float64(30)},
		{"name": "Bob", "age": float64(25)},
	}
	cols := FromRows(rows)
	if len(cols) < 2 {
		t.Fatalf("expected at least 2 columns, got %d", len(cols))
	}

	// Find columns by ID.
	colMap := make(map[string]Column[map[string]any])
	for _, c := range cols {
		colMap[c.ColumnID] = c
	}

	nameCol, ok := colMap["name"]
	if !ok {
		t.Fatal("missing 'name' column")
	}
	if v := nameCol.ValueGetter(rows[0]); v != "Alice" {
		t.Errorf("name ValueGetter: expected Alice, got %v", v)
	}

	ageCol, ok := colMap["age"]
	if !ok {
		t.Fatal("missing 'age' column")
	}
	if v := ageCol.ValueGetter(rows[0]); v != float64(30) {
		t.Errorf("age ValueGetter: expected 30, got %v", v)
	}
}

func TestFromRowsMapInfersTypes(t *testing.T) {
	rows := []map[string]any{
		{"b": true, "i": float64(10), "f": float64(3.14), "t": "2025-06-15", "s": "hello"},
		{"b": false, "i": float64(20), "f": float64(2.71), "t": "2025-09-01", "s": "world"},
	}
	cols := FromRows(rows)

	colMap := make(map[string]Column[map[string]any])
	for _, c := range cols {
		colMap[c.ColumnID] = c
	}

	// Bool column gets BoolFilter.
	if _, ok := colMap["b"].Filter.(*filter.BoolFilter); !ok {
		t.Errorf("bool column: expected BoolFilter, got %T", colMap["b"].Filter)
	}

	// Int column (integer-valued float64) gets NumberFilter.
	if _, ok := colMap["i"].Filter.(*filter.NumberFilter); !ok {
		t.Errorf("int column: expected NumberFilter, got %T", colMap["i"].Filter)
	}

	// Float column gets NumberFilter.
	if _, ok := colMap["f"].Filter.(*filter.NumberFilter); !ok {
		t.Errorf("float column: expected NumberFilter, got %T", colMap["f"].Filter)
	}

	// Time column gets TimeFilter.
	if _, ok := colMap["t"].Filter.(*filter.TimeFilter); !ok {
		t.Errorf("time column: expected TimeFilter, got %T", colMap["t"].Filter)
	}

	// String column gets SetFilter (2 distinct values <= 20).
	if _, ok := colMap["s"].Filter.(*filter.SetFilter); !ok {
		t.Errorf("string column: expected SetFilter, got %T", colMap["s"].Filter)
	}
}

func TestFromRowsMapTimeConversion(t *testing.T) {
	rows := []map[string]any{
		{"date": "2025-06-15"},
		{"date": "2025-09-01"},
	}
	cols := FromRows(rows)
	if len(cols) != 1 {
		t.Fatalf("expected 1 column, got %d", len(cols))
	}

	v := cols[0].ValueGetter(rows[0])
	if _, ok := v.(time.Time); !ok {
		t.Errorf("expected time.Time, got %T (%v)", v, v)
	}
}

func TestFromRowsMapPreservesKeyOrder(t *testing.T) {
	// Use a single row so map iteration order defines first-appearance order.
	// Since Go map iteration is random, we use multiple rows with one key each.
	rows := []map[string]any{
		{"alpha": "a"},
		{"beta": "b"},
		{"gamma": "c"},
	}
	cols := FromRows(rows)

	// alpha must come first since it appears in the first row.
	if len(cols) < 3 {
		t.Fatalf("expected 3 columns, got %d", len(cols))
	}
	if cols[0].ColumnID != "alpha" {
		t.Errorf("first column: expected alpha, got %q", cols[0].ColumnID)
	}
	if cols[1].ColumnID != "beta" {
		t.Errorf("second column: expected beta, got %q", cols[1].ColumnID)
	}
	if cols[2].ColumnID != "gamma" {
		t.Errorf("third column: expected gamma, got %q", cols[2].ColumnID)
	}
}

func TestFromRowsMapEmptyRows(t *testing.T) {
	cols := FromRows[map[string]any](nil)
	if cols != nil {
		t.Errorf("expected nil for empty rows, got %v", cols)
	}

	cols = FromRows([]map[string]any{})
	if cols != nil {
		t.Errorf("expected nil for zero-length rows, got %v", cols)
	}
}

type sliceTestPerson struct {
	Name  string
	Age   int
	Email string
}

type sliceTestProduct struct {
	Name    string
	Price   float64
	InStock bool
}

func TestFromRowsSliceOfStructs(t *testing.T) {
	rows := [][]any{
		{sliceTestPerson{Name: "Alice", Age: 30, Email: "alice@test.com"}},
		{sliceTestProduct{Name: "Widget", Price: 9.99, InStock: true}},
	}
	cols := FromRows(rows)

	ids := make(map[string]bool)
	for _, c := range cols {
		ids[c.ColumnID] = true
	}

	expected := []string{"Name", "Age", "Email", "Price", "InStock"}
	for _, e := range expected {
		if !ids[e] {
			t.Errorf("missing column %q", e)
		}
	}
}

func TestFromRowsSliceValueGetter(t *testing.T) {
	rows := [][]any{
		{sliceTestPerson{Name: "Alice", Age: 30, Email: "alice@test.com"}},
		{sliceTestProduct{Name: "Widget", Price: 9.99, InStock: true}},
	}
	cols := FromRows(rows)

	colMap := make(map[string]Column[[]any])
	for _, c := range cols {
		colMap[c.ColumnID] = c
	}

	// Name exists on both types.
	nameCol := colMap["Name"]
	if v := nameCol.ValueGetter(rows[0]); v != "Alice" {
		t.Errorf("Name from Person: expected Alice, got %v", v)
	}
	if v := nameCol.ValueGetter(rows[1]); v != "Widget" {
		t.Errorf("Name from Product: expected Widget, got %v", v)
	}

	// Age only exists on Person; nil from Product.
	ageCol := colMap["Age"]
	if v := ageCol.ValueGetter(rows[0]); v != 30 {
		t.Errorf("Age from Person: expected 30, got %v", v)
	}
	if v := ageCol.ValueGetter(rows[1]); v != nil {
		t.Errorf("Age from Product: expected nil, got %v", v)
	}
}

func TestFromRowsSliceFieldTypes(t *testing.T) {
	type TimeStruct struct {
		When time.Time
	}
	rows := [][]any{
		{sliceTestPerson{Age: 30}},
		{sliceTestProduct{Price: 9.99, InStock: true}},
		{TimeStruct{When: time.Now()}},
	}
	cols := FromRows(rows)

	colMap := make(map[string]Column[[]any])
	for _, c := range cols {
		colMap[c.ColumnID] = c
	}

	// int field gets NumberFilter.
	if _, ok := colMap["Age"].Filter.(*filter.NumberFilter); !ok {
		t.Errorf("Age: expected NumberFilter, got %T", colMap["Age"].Filter)
	}
	// float field gets NumberFilter.
	if _, ok := colMap["Price"].Filter.(*filter.NumberFilter); !ok {
		t.Errorf("Price: expected NumberFilter, got %T", colMap["Price"].Filter)
	}
	// bool field gets BoolFilter.
	if _, ok := colMap["InStock"].Filter.(*filter.BoolFilter); !ok {
		t.Errorf("InStock: expected BoolFilter, got %T", colMap["InStock"].Filter)
	}
	// time field gets TimeFilter.
	if _, ok := colMap["When"].Filter.(*filter.TimeFilter); !ok {
		t.Errorf("When: expected TimeFilter, got %T", colMap["When"].Filter)
	}
}

func TestFromRowsStructFallback(t *testing.T) {
	rows := []Person{
		{Name: "Alice", Age: 30},
	}
	cols := FromRows(rows)
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}
	if cols[0].ColumnID != "Name" {
		t.Errorf("first col: expected Name, got %q", cols[0].ColumnID)
	}
	if cols[1].ColumnID != "Age" {
		t.Errorf("second col: expected Age, got %q", cols[1].ColumnID)
	}

	// Verify ValueGetter works.
	if v := cols[0].ValueGetter(rows[0]); v != "Alice" {
		t.Errorf("Name getter: expected Alice, got %v", v)
	}
}

func TestFromRowsUnsupportedType(t *testing.T) {
	cols := FromRows[string](nil)
	if cols != nil {
		t.Errorf("expected nil for unsupported type, got %v", cols)
	}
}
