package column

import (
	"testing"
	"time"
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

func TestPinDirectionConstants(t *testing.T) {
	if PinNone == PinLeft || PinNone == PinRight || PinLeft == PinRight {
		t.Fatal("PinDirection constants must be distinct")
	}
}

func TestSortDirectionConstants(t *testing.T) {
	if SortNone == SortAsc || SortNone == SortDesc || SortAsc == SortDesc {
		t.Fatal("SortDirection constants must be distinct")
	}
}

// --- Columns[T]() ---

func TestColumnsExportedFields(t *testing.T) {
	cols := Columns[Person]()
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}

	if cols[0].ColID != "Name" || cols[0].HeaderName != "Name" {
		t.Errorf("first col: got ColID=%q HeaderName=%q", cols[0].ColID, cols[0].HeaderName)
	}
	if cols[1].ColID != "Age" || cols[1].HeaderName != "Age" {
		t.Errorf("second col: got ColID=%q HeaderName=%q", cols[1].ColID, cols[1].HeaderName)
	}

	for _, c := range cols {
		if !c.Sortable {
			t.Errorf("col %q: Sortable should be true", c.ColID)
		}
		if !c.Filterable {
			t.Errorf("col %q: Filterable should be true", c.ColID)
		}
		if c.MinWidth != 4 {
			t.Errorf("col %q: MinWidth should be 4, got %d", c.ColID, c.MinWidth)
		}
	}
}

func TestColumnsValueGetter(t *testing.T) {
	cols := Columns[Person]()
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

func TestColumnsPointerType(t *testing.T) {
	cols := Columns[*Person]()
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns for *Person, got %d", len(cols))
	}

	p := &Person{Name: "Bob", Age: 25}
	name := cols[0].ValueGetter(p)
	if name != "Bob" {
		t.Errorf("pointer Name getter: expected Bob, got %v", name)
	}
}

func TestColumnsNonStructPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for non-struct type")
		}
	}()
	Columns[string]()
}

func TestColumnsEmptyStruct(t *testing.T) {
	cols := Columns[struct{}]()
	if len(cols) != 0 {
		t.Fatalf("expected 0 columns for empty struct, got %d", len(cols))
	}
}

func TestColumnsUnexportedSkipped(t *testing.T) {
	cols := Columns[mixedFields]()
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns (exported only), got %d", len(cols))
	}
	if cols[0].ColID != "Exported" {
		t.Errorf("first col should be Exported, got %q", cols[0].ColID)
	}
	if cols[1].ColID != "Another" {
		t.Errorf("second col should be Another, got %q", cols[1].ColID)
	}
}

func TestColumnsMultipleFieldTypes(t *testing.T) {
	cols := Columns[multiType]()
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
