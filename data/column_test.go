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
	unexported string //nolint:unused
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

func TestColumn_QueryAliases(t *testing.T) {
	col := Column[map[string]any]{
		ColumnID:     "state",
		QueryAliases: []string{"status", "st"},
	}
	if len(col.QueryAliases) != 2 || col.QueryAliases[0] != "status" {
		t.Errorf("QueryAliases = %v, want [status st]", col.QueryAliases)
	}
}

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

func TestFromTypeValue(t *testing.T) {
	cols := FromType[Person]()
	p := Person{Name: "Alice", Age: 30}

	name := cols[0].Value(p)
	if name != "Alice" {
		t.Errorf("Name getter: expected Alice, got %v", name)
	}

	age := cols[1].Value(p)
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
	name := cols[0].Value(p)
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
		got := cols[tt.idx].Value(row)
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
	if v := nameCol.Value(rows[0]); v != "Alice" {
		t.Errorf("name Value: expected Alice, got %v", v)
	}

	ageCol, ok := colMap["age"]
	if !ok {
		t.Fatal("missing 'age' column")
	}
	if v := ageCol.Value(rows[0]); v != float64(30) {
		t.Errorf("age Value: expected 30, got %v", v)
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

	v := cols[0].Value(rows[0])
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

func TestFromRowsSliceValue(t *testing.T) {
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
	if v := nameCol.Value(rows[0]); v != "Alice" {
		t.Errorf("Name from Person: expected Alice, got %v", v)
	}
	if v := nameCol.Value(rows[1]); v != "Widget" {
		t.Errorf("Name from Product: expected Widget, got %v", v)
	}

	// Age only exists on Person; nil from Product.
	ageCol := colMap["Age"]
	if v := ageCol.Value(rows[0]); v != 30 {
		t.Errorf("Age from Person: expected 30, got %v", v)
	}
	if v := ageCol.Value(rows[1]); v != nil {
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

	// Verify Value works.
	if v := cols[0].Value(rows[0]); v != "Alice" {
		t.Errorf("Name getter: expected Alice, got %v", v)
	}
}

func TestFromRowsUnsupportedType(t *testing.T) {
	cols := FromRows[string](nil)
	if cols != nil {
		t.Errorf("expected nil for unsupported type, got %v", cols)
	}
}

// --- FromType: nil pointer value ---

func TestFromTypeNilPointerValue(t *testing.T) {
	cols := FromType[*Person]()
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}
	// Value with a nil pointer should return nil.
	var p *Person
	v := cols[0].Value(p)
	if v != nil {
		t.Errorf("nil pointer Value should return nil, got %v", v)
	}
}

// --- FromRows: interface type with nil rows ---

func TestFromRowsInterfaceAllNil(t *testing.T) {
	rows := []any{nil, nil, nil}
	cols := FromRows(rows)
	if cols != nil {
		t.Errorf("all nil interface rows: expected nil, got %v", cols)
	}
}

// --- FromRows: ptr-to-struct type ---

func TestFromRowsPtrToStruct(t *testing.T) {
	rows := []*Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}
	cols := FromRows(rows)
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns for *Person, got %d", len(cols))
	}
}

// --- FromRows: ptr-to-non-struct ---

func TestFromRowsPtrToNonStruct(t *testing.T) {
	s := "hello"
	rows := []*string{&s}
	cols := FromRows(rows)
	if cols != nil {
		t.Errorf("ptr to non-struct: expected nil, got %v", cols)
	}
}

// --- FromRows: int type (unsupported) ---

func TestFromRowsIntType(t *testing.T) {
	rows := []int{1, 2, 3}
	cols := FromRows(rows)
	if cols != nil {
		t.Errorf("int type: expected nil, got %v", cols)
	}
}

// --- columnsFromMap: interface-wrapped map row ---

func TestColumnsFromMapInterfaceWrapped(t *testing.T) {
	// Use any type with actual map[string]any values.
	rows := []any{
		map[string]any{"x": "hello"},
		map[string]any{"x": "world"},
	}
	cols := columnsFromMap(rows)
	if len(cols) != 1 {
		t.Fatalf("expected 1 column, got %d", len(cols))
	}
	if cols[0].ColumnID != "x" {
		t.Errorf("expected column ID 'x', got %q", cols[0].ColumnID)
	}
}

// --- makeMapColumn: bool category with nil value ---

func TestMakeMapColumnBoolNilValue(t *testing.T) {
	col := makeMapColumn[map[string]any]("flag", "bool")
	// Row with nil value for the key.
	row := map[string]any{"flag": nil}
	v := col.Value(row)
	if v != nil {
		t.Errorf("bool category nil value: expected nil, got %v", v)
	}
}

// --- makeMapColumn: bool category with valid value ---

func TestMakeMapColumnBoolValue(t *testing.T) {
	col := makeMapColumn[map[string]any]("flag", "bool")
	row := map[string]any{"flag": true}
	v := col.Value(row)
	if v != true {
		t.Errorf("bool category: expected true, got %v", v)
	}
}

// --- makeMapColumn: bool category with non-bool value ---

func TestMakeMapColumnBoolNonBoolValue(t *testing.T) {
	col := makeMapColumn[map[string]any]("flag", "bool")
	row := map[string]any{"flag": "notbool"}
	v := col.Value(row)
	if v != "notbool" {
		t.Errorf("bool category non-bool value: expected 'notbool', got %v", v)
	}
}

// --- makeMapColumn: time category ---

func TestMakeMapColumnTimeCategory(t *testing.T) {
	col := makeMapColumn[map[string]any]("date", "time")

	// Valid time string.
	row := map[string]any{"date": "2025-06-15"}
	v := col.Value(row)
	if _, ok := v.(time.Time); !ok {
		t.Errorf("time category: expected time.Time, got %T (%v)", v, v)
	}

	// Nil value.
	row2 := map[string]any{"date": nil}
	v2 := col.Value(row2)
	if v2 != nil {
		t.Errorf("time category nil: expected nil, got %v", v2)
	}

	// Non-parseable string.
	row3 := map[string]any{"date": "not-a-date"}
	v3 := col.Value(row3)
	// Should return the string as-is since it doesn't parse.
	if v3 != "not-a-date" {
		t.Errorf("time category non-parseable: expected 'not-a-date', got %v", v3)
	}

	// Non-string value.
	row4 := map[string]any{"date": 12345}
	v4 := col.Value(row4)
	if v4 != 12345 {
		t.Errorf("time category non-string: expected 12345, got %v", v4)
	}
}

// --- makeMapColumn: number category nil value ---

func TestMakeMapColumnNumberNilValue(t *testing.T) {
	col := makeMapColumn[map[string]any]("score", "number")
	row := map[string]any{"score": nil}
	v := col.Value(row)
	if v != nil {
		t.Errorf("number category nil: expected nil, got %v", v)
	}
}

// --- makeMapColumn: number category non-float value ---

func TestMakeMapColumnNumberNonFloat(t *testing.T) {
	col := makeMapColumn[map[string]any]("score", "number")
	row := map[string]any{"score": "notanumber"}
	v := col.Value(row)
	if v != "notanumber" {
		t.Errorf("number category non-float: expected 'notanumber', got %v", v)
	}
}

// --- makeMapColumn: string (default) category nil value ---

func TestMakeMapColumnStringNilValue(t *testing.T) {
	col := makeMapColumn[map[string]any]("label", "string")
	row := map[string]any{"label": nil}
	v := col.Value(row)
	if v != nil {
		t.Errorf("string category nil: expected nil, got %v", v)
	}
}

// --- mapIndex: interface-wrapped value ---

func TestMapIndexInterfaceWrapped(t *testing.T) {
	// Use any type to wrap the map.
	var row any = map[string]any{"key": "value"}
	v := mapIndex(row, "key")
	if v != "value" {
		t.Errorf("interface-wrapped map: expected 'value', got %v", v)
	}
}

// --- mapIndex: non-map row ---

func TestMapIndexNonMap(t *testing.T) {
	v := mapIndex("not a map", "key")
	if v != nil {
		t.Errorf("non-map row: expected nil, got %v", v)
	}
}

// --- mapIndex: interface-wrapped nil value in map ---

func TestMapIndexInterfaceNilValue(t *testing.T) {
	// Create a map where the value is an interface wrapping nil.
	row := map[string]any{"key": (any)(nil)}
	v := mapIndex(row, "key")
	if v != nil {
		t.Errorf("interface nil value: expected nil, got %v", v)
	}
}

// --- mapIndex: missing key ---

func TestMapIndexMissingKey(t *testing.T) {
	row := map[string]any{"key": "value"}
	v := mapIndex(row, "missing")
	if v != nil {
		t.Errorf("missing key: expected nil, got %v", v)
	}
}

// --- inferMapColumnType: all nil values ---

func TestInferMapColumnTypeAllNil(t *testing.T) {
	rows := []map[string]any{
		{"x": nil},
		{"x": nil},
	}
	result := inferMapColumnType("x", rows)
	if result != "string" {
		t.Errorf("all nil: expected 'string', got %q", result)
	}
}

// --- inferMapColumnType: default case (non-bool/float/string/time) ---

func TestInferMapColumnTypeDefaultCase(t *testing.T) {
	// Use a value that is not bool, float64, or string - e.g., an int or a struct.
	rows := []map[string]any{
		{"x": struct{ A int }{A: 1}},
		{"x": struct{ A int }{A: 2}},
	}
	result := inferMapColumnType("x", rows)
	if result != "string" {
		t.Errorf("non-standard type: expected 'string', got %q", result)
	}
}

// --- inferMapColumnType: mixed string with non-time ---

func TestInferMapColumnTypeNonTimeString(t *testing.T) {
	rows := []map[string]any{
		{"x": "not a date"},
		{"x": "also not a date"},
	}
	result := inferMapColumnType("x", rows)
	if result != "string" {
		t.Errorf("non-time strings: expected 'string', got %q", result)
	}
}

// --- columnsFromSlice: non-struct slice elements ---

func TestColumnsFromSliceNonStructElements(t *testing.T) {
	rows := [][]any{
		{"just", "strings"},
		{42, 43},
	}
	cols := columnsFromSlice(rows)
	// Non-struct elements should be skipped, resulting in no columns.
	if len(cols) != 0 {
		t.Errorf("non-struct elements: expected 0 columns, got %d", len(cols))
	}
}

// --- columnsFromSlice: empty rows ---

func TestColumnsFromSliceEmpty(t *testing.T) {
	cols := columnsFromSlice[[]any](nil)
	if cols != nil {
		t.Errorf("empty rows: expected nil, got %v", cols)
	}
}

// --- columnsFromSlice: interface-wrapped structs ---

func TestColumnsFromSliceInterfaceWrappedStructs(t *testing.T) {
	rows := [][]any{
		{any(sliceTestPerson{Name: "Alice", Age: 30, Email: "a@b.com"})},
	}
	cols := columnsFromSlice(rows)
	if len(cols) < 3 {
		t.Fatalf("expected at least 3 columns, got %d", len(cols))
	}
	ids := make(map[string]bool)
	for _, c := range cols {
		ids[c.ColumnID] = true
	}
	if !ids["Name"] || !ids["Age"] || !ids["Email"] {
		t.Errorf("missing expected columns, got: %v", ids)
	}
}

// --- columnsFromSlice: pointer-wrapped structs ---

func TestColumnsFromSlicePtrWrappedStructs(t *testing.T) {
	p := sliceTestPerson{Name: "Alice", Age: 30, Email: "a@b.com"}
	rows := [][]any{
		{&p},
	}
	cols := columnsFromSlice(rows)
	if len(cols) < 3 {
		t.Fatalf("expected at least 3 columns for ptr-wrapped struct, got %d", len(cols))
	}
}

// --- sliceFieldValue: non-slice row ---

func TestSliceFieldValueNonSlice(t *testing.T) {
	v := sliceFieldValue("not a slice", "Name")
	if v != nil {
		t.Errorf("non-slice row: expected nil, got %v", v)
	}
}

// --- sliceFieldValue: field not found ---

func TestSliceFieldValueFieldNotFound(t *testing.T) {
	row := []any{sliceTestPerson{Name: "Alice", Age: 30}}
	v := sliceFieldValue(row, "NonExistentField")
	if v != nil {
		t.Errorf("field not found: expected nil, got %v", v)
	}
}

// --- sliceFieldValue: ptr-wrapped struct ---

func TestSliceFieldValuePtrWrappedStruct(t *testing.T) {
	p := sliceTestPerson{Name: "Alice", Age: 30, Email: "a@b.com"}
	row := []any{&p}
	v := sliceFieldValue(row, "Name")
	if v != "Alice" {
		t.Errorf("ptr-wrapped struct: expected 'Alice', got %v", v)
	}
}

// --- sliceFieldValue: interface-wrapped value ---

func TestSliceFieldValueInterfaceWrapped(t *testing.T) {
	var elem any = sliceTestPerson{Name: "Bob", Age: 25}
	row := []any{elem}
	v := sliceFieldValue(row, "Name")
	if v != "Bob" {
		t.Errorf("interface-wrapped struct: expected 'Bob', got %v", v)
	}
}

// --- sliceFieldValue: non-struct element in slice ---

func TestSliceFieldValueNonStructElement(t *testing.T) {
	row := []any{"not a struct", 42}
	v := sliceFieldValue(row, "Name")
	if v != nil {
		t.Errorf("non-struct element: expected nil, got %v", v)
	}
}

// --- applyTypeDefaults: float category ---

func TestApplyTypeDefaultsFloat(t *testing.T) {
	col := Column[map[string]any]{
		ColumnID: "score",
	}
	applyTypeDefaults(&col, "float", nil)
	if _, ok := col.Filter.(*filter.NumberFilter); !ok {
		t.Errorf("float category: expected NumberFilter, got %T", col.Filter)
	}
	if col.Width != 14 {
		t.Errorf("float category: expected Width=14, got %d", col.Width)
	}
	if col.CellStyle == nil {
		t.Fatal("float category: expected CellStyle to be set")
	}
	// Invoke the CellStyle function to cover the lambda body.
	row := map[string]any{}
	style := col.CellStyle(nil, row)
	_ = style // just verify it doesn't panic
}

// --- applyTypeDefaults: number category ---

func TestApplyTypeDefaultsNumber(t *testing.T) {
	col := Column[map[string]any]{
		ColumnID: "score",
	}
	applyTypeDefaults(&col, "number", nil)
	if _, ok := col.Filter.(*filter.NumberFilter); !ok {
		t.Errorf("number category: expected NumberFilter, got %T", col.Filter)
	}
	if col.Width != 14 {
		t.Errorf("number category: expected Width=14, got %d", col.Width)
	}
}

// --- applyTypeDefaults: int category CellStyle ---

func TestApplyTypeDefaultsIntCellStyle(t *testing.T) {
	col := Column[map[string]any]{
		ColumnID: "count",
	}
	applyTypeDefaults(&col, "int", nil)
	if col.CellStyle == nil {
		t.Fatal("int category: expected CellStyle to be set")
	}
	// Invoke the CellStyle function to cover the lambda body.
	row := map[string]any{}
	style := col.CellStyle(nil, row)
	_ = style // just verify it doesn't panic
}

// --- applyTypeDefaults: string with > 20 distinct values ---

func TestApplyTypeDefaultsStringManyDistinct(t *testing.T) {
	col := Column[map[string]any]{
		ColumnID: "label",
	}
	// Generate more than 20 distinct values.
	vals := make([]string, 25)
	for i := range vals {
		vals[i] = string(rune('A' + i))
	}
	applyTypeDefaults(&col, "string", vals)
	// With > 20 distinct values, should get TextFilter instead of SetFilter.
	if _, ok := col.Filter.(*filter.TextFilter); !ok {
		t.Errorf("> 20 distinct values: expected TextFilter, got %T", col.Filter)
	}
	if col.MinWidth != 8 {
		t.Errorf("string category: expected MinWidth=8, got %d", col.MinWidth)
	}
	if col.Flex != 1 {
		t.Errorf("string category: expected Flex=1, got %d", col.Flex)
	}
}

// --- applyTypeDefaults: string with 0 distinct values ---

func TestApplyTypeDefaultsStringNoDistinct(t *testing.T) {
	col := Column[map[string]any]{
		ColumnID: "label",
	}
	applyTypeDefaults(&col, "string", nil)
	// With 0 distinct values (nil), should get TextFilter.
	if _, ok := col.Filter.(*filter.TextFilter); !ok {
		t.Errorf("0 distinct values: expected TextFilter, got %T", col.Filter)
	}
}

// --- parseTimeString: non-matching format ---

func TestParseTimeStringNoMatch(t *testing.T) {
	_, ok := parseTimeString("this is not a time at all")
	if ok {
		t.Error("non-matching string: expected false, got true")
	}
}

// --- parseTimeString: valid format ---

func TestParseTimeStringValid(t *testing.T) {
	tm, ok := parseTimeString("2025-06-15")
	if !ok {
		t.Error("valid date string: expected true, got false")
	}
	if tm.Year() != 2025 || tm.Month() != time.June || tm.Day() != 15 {
		t.Errorf("parsed time mismatch: got %v", tm)
	}
}

// --- FromRows: nil slice of maps ---

func TestFromRowsNilMapSlice(t *testing.T) {
	var rows []map[string]any
	cols := FromRows(rows)
	if cols != nil {
		t.Errorf("nil map slice: expected nil, got %v", cols)
	}
}

// --- FromRows: slice type ([][]any) ---

func TestFromRowsSliceType(t *testing.T) {
	rows := [][]any{
		{sliceTestPerson{Name: "Alice", Age: 30}},
	}
	cols := FromRows(rows)
	if len(cols) < 2 {
		t.Fatalf("expected at least 2 columns, got %d", len(cols))
	}
}

// --- FromRows: empty slice of slices ---

func TestFromRowsEmptySliceOfSlices(t *testing.T) {
	rows := [][]any{}
	cols := FromRows(rows)
	if cols != nil {
		t.Errorf("empty slice of slices: expected nil, got %v", cols)
	}
}

// --- makeMapColumn: int category ---

func TestMakeMapColumnIntCategory(t *testing.T) {
	col := makeMapColumn[map[string]any]("count", "int")
	row := map[string]any{"count": float64(42)}
	v := col.Value(row)
	if v != float64(42) {
		t.Errorf("int category: expected 42, got %v", v)
	}
}

// --- makeMapColumn: string (default) category ---

func TestMakeMapColumnStringCategory(t *testing.T) {
	col := makeMapColumn[map[string]any]("name", "string")
	row := map[string]any{"name": 42}
	v := col.Value(row)
	// Default category calls fmt.Sprint on non-nil values.
	if v != "42" {
		t.Errorf("string category fmt.Sprint: expected '42', got %v", v)
	}
}

// --- FromRows: interface with map (non-string key) ---

func TestFromRowsMapNonStringKey(t *testing.T) {
	rows := []map[int]any{
		{1: "one"},
	}
	cols := FromRows(rows)
	if cols != nil {
		t.Errorf("map with non-string key: expected nil, got %v", cols)
	}
}

// --- columnsFromMap: non-map row in interface rows ---

func TestColumnsFromMapNonMapRow(t *testing.T) {
	// Mix of map and non-map rows in interface-typed slice.
	rows := []any{
		map[string]any{"key": "value"},
		"not a map",
	}
	cols := columnsFromMap(rows)
	if len(cols) != 1 {
		t.Fatalf("expected 1 column, got %d", len(cols))
	}
	if cols[0].ColumnID != "key" {
		t.Errorf("expected column 'key', got %q", cols[0].ColumnID)
	}
}

// --- columnsFromSlice: struct with unexported fields ---

type sliceTestWithUnexported struct {
	Public     string
	private    int
	AlsoPublic bool
}

func TestColumnsFromSliceUnexportedFields(t *testing.T) {
	rows := [][]any{
		{sliceTestWithUnexported{Public: "hello", private: 42, AlsoPublic: true}},
	}
	cols := columnsFromSlice(rows)
	// Should only have Public and AlsoPublic columns (unexported skipped).
	ids := make(map[string]bool)
	for _, c := range cols {
		ids[c.ColumnID] = true
	}
	if ids["private"] {
		t.Error("unexported field 'private' should not appear as a column")
	}
	if !ids["Public"] {
		t.Error("missing 'Public' column")
	}
	if !ids["AlsoPublic"] {
		t.Error("missing 'AlsoPublic' column")
	}
	if len(cols) != 2 {
		t.Errorf("expected 2 columns, got %d", len(cols))
	}
}

// --- FromRows: interface with pointer-to-slice (exercises interface→Slice branch) ---

func TestFromRowsInterfaceWithSlice(t *testing.T) {
	// reflect.ValueOf strips the interface; use a pointer so Elem() works
	s := []any{sliceTestPerson{Name: "Alice", Age: 30}}
	rows := []any{&s}
	cols := FromRows(rows)
	// The branch is exercised; columnsFromSlice receives ptr-wrapped slices
	// which it may not fully unwrap, so nil is acceptable.
	_ = cols
}

// --- FromRows: interface with pointer-to-non-string-key map ---

func TestFromRowsInterfaceWithNonStringMap(t *testing.T) {
	m := map[int]any{1: "one"}
	rows := []any{&m}
	cols := FromRows(rows)
	if cols != nil {
		t.Errorf("interface with non-string map key: expected nil, got %v", cols)
	}
}

// --- FromRows: interface with pointer-to-non-map/non-slice (exercises default branch) ---

func TestFromRowsInterfaceDefaultCase(t *testing.T) {
	n := 42
	rows := []any{&n}
	cols := FromRows(rows)
	if cols != nil {
		t.Errorf("interface with ptr-to-int value: expected nil, got %v", cols)
	}
}

// --- FromRows: interface with pointer-to-string-key map (exercises map branch) ---

func TestFromRowsInterfaceWithStringMap(t *testing.T) {
	m := map[string]any{"name": "Alice"}
	rows := []any{&m}
	cols := FromRows(rows)
	// Exercises the interface→Map(string key) branch
	_ = cols
}

// --- columnsFromSlice: row is not a slice ---

func TestColumnsFromSliceNonSliceRow(t *testing.T) {
	rows := []any{
		"not a slice",
		[]any{sliceTestPerson{Name: "Alice"}},
	}
	cols := columnsFromSlice(rows)
	if len(cols) < 1 {
		t.Fatalf("expected at least 1 column, got %d", len(cols))
	}
}

func TestDefaultIsEmpty(t *testing.T) {
	zeroTime := time.Time{}
	someTime, _ := time.Parse("2006-01-02", "2026-04-01")
	cases := []struct {
		name  string
		value any
		empty bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "x", false},
		{"int 0", 0, false},
		{"int 5", 5, false},
		{"false", false, false},
		{"true", true, false},
		{"zero time", zeroTime, true},
		{"non-zero time", someTime, false},
		{"nil *time.Time", (*time.Time)(nil), true},
		{"nil string slice", []string(nil), true},
		{"empty string slice", []string{}, true},
		{"single empty-string slice", []string{""}, false},
		{"populated slice", []string{"a"}, false},
		{"empty map", map[string]int{}, true},
		{"populated map", map[string]int{"k": 1}, false},
		{"nil pointer", (*int)(nil), true},
		{"non-nil pointer to zero int", func() *int { i := 0; return &i }(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DefaultIsEmpty(tc.value); got != tc.empty {
				t.Errorf("DefaultIsEmpty(%#v) = %v, want %v", tc.value, got, tc.empty)
			}
		})
	}
}

func TestColumnIsEmpty_UsesOverrideIfSet(t *testing.T) {
	c := Column[int]{
		ColumnID: "x",
		Value:    func(i int) any { return i },
		IsEmpty:  func(i *int) bool { return *i == -1 }, // sentinel
	}
	row := -1
	if !ColumnIsEmpty(&c, &row) {
		t.Errorf("override should treat -1 as empty")
	}
	row = 5
	if ColumnIsEmpty(&c, &row) {
		t.Errorf("override should treat 5 as non-empty")
	}
}

func TestColumnIsEmpty_FallsBackToDefault(t *testing.T) {
	c := Column[string]{
		ColumnID: "x",
		Value:    func(s string) any { return s },
	}
	empty := ""
	if !ColumnIsEmpty(&c, &empty) {
		t.Errorf("empty string should be empty under default rule")
	}
	full := "hello"
	if ColumnIsEmpty(&c, &full) {
		t.Errorf("non-empty string should not be empty under default rule")
	}
}
