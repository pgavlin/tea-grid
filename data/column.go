// Package data defines column, row, and cell types for the tea-grid component.
package data

import (
	"fmt"
	"math"
	"reflect"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/pgavlin/tea-grid/filter"
)

// Pin indicates the pin position for columns (left/right) or rows (top/bottom).
type Pin int

const (
	PinNone   Pin = iota
	PinLeft       // Column: pinned to left edge.
	PinRight      // Column: pinned to right edge.
	PinTop        // Row: pinned to top.
	PinBottom     // Row: pinned to bottom.
)

// SortDirection indicates the sort direction for a column.
type SortDirection int

const (
	SortNone SortDirection = iota
	SortAsc
	SortDesc
)

// Column defines a single column in the grid.
type Column[T any] struct {
	// Identity
	ColumnID   string // Unique identifier. Required.
	HeaderName string // Display name in the header row.

	// Data access
	ValueGetter    func(T) any                    // Extracts the cell value from the row data. Required.
	ValueFormatter func(value any, data T) string // Format the value for display.

	// Sizing
	Width    int // Fixed width in terminal columns. 0 = auto.
	MinWidth int // Minimum width (default: 4).
	MaxWidth int // Maximum width. 0 = unconstrained.
	Flex     int // Flex weight for distributing remaining space. 0 = no flex.

	// Sorting
	Sortable   bool               // Default: true.
	Comparator func(a, b any) int // Custom sort.

	// Filtering
	Filterable bool            // Default: true.
	Filter     filter.Filter   // Column filter.
	FilterText func(T) string  // Returns text for quick filter matching. If nil, ValueGetter + fmt.Sprint is used.

	// Pinning
	Pinned     Pin  // Left, Right, or None.
	LockPinned bool // Prevent user from changing pin state.

	// Cell rendering
	CellRenderer         CellRenderer[T]                        // Custom renderer.
	CellRendererSelector func(T) CellRenderer[T]                // Dynamic renderer per row.
	CellStyle            func(value any, data T) lipgloss.Style // Per-cell styling.

	// Cell editing
	Editable    bool                     // Default: false.
	CellEditor  CellEditor[T]            // Custom editor.
	ValueSetter func(data *T, value any) // Write the edited value back.

	// Column spanning
	ColumnSpan func(data T) int // Number of columns this cell spans. Default: 1.

	// Aggregation
	AggFunc       string                 // Aggregation function name: "sum", "avg", "count", "min", "max".
	AggFuncCustom func(values []any) any // Custom aggregation.

	// Visibility
	Hide     bool // If true, column is not rendered.
	NoSelect bool // If true, column is excluded from row and column selection highlighting.
}

// ColumnGroup produces a single level of grouped headers.
type ColumnGroup[T any] struct {
	HeaderName string
	Columns    []Column[T] // Leaf columns in this group.
}

// FromType returns a []Column[T] derived from T's exported struct fields.
// For each exported field, it produces a Column with:
//   - ColumnID and HeaderName set to the field name
//   - ValueGetter set to a function that retrieves the field's value from T
//
// Panics if T is not a struct type.
func FromType[T any]() []Column[T] {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic("column.Columns: T must be a struct type")
	}

	var cols []Column[T]
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		idx := i // capture for closure
		col := Column[T]{
			ColumnID:   field.Name,
			HeaderName: field.Name,
			ValueGetter: func(data T) any {
				v := reflect.ValueOf(data)
				if v.Kind() == reflect.Ptr {
					if v.IsNil() {
						return nil
					}
					v = v.Elem()
				}
				return v.Field(idx).Interface()
			},
			Sortable:   true,
			Filterable: true,
			MinWidth:   4,
			Filter:     filterForType(field.Type),
		}
		cols = append(cols, col)
	}
	return cols
}

// timeFormats lists time layouts tried when inferring time-typed columns from strings.
var timeFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// FromRows returns a []Column[T] inferred from the provided rows.
// It supports:
//   - map[string]any: discovers keys and infers value types
//   - []any (slice): discovers struct fields from heterogeneous elements
//   - struct or *struct: delegates to Columns[T]() (rows are ignored)
//
// For interface types, the first non-nil row is inspected to determine the
// actual kind. Returns nil for unsupported types or empty rows.
func FromRows[T any](rows []T) []Column[T] {
	var zero T
	t := reflect.TypeOf(&zero).Elem()

	// For concrete types, dispatch directly.
	switch t.Kind() {
	case reflect.Map:
		if t.Key().Kind() == reflect.String {
			return columnsFromMap(rows)
		}
		return nil
	case reflect.Slice:
		return columnsFromSlice(rows)
	case reflect.Struct:
		return FromType[T]()
	case reflect.Ptr:
		if t.Elem().Kind() == reflect.Struct {
			return FromType[T]()
		}
		return nil
	case reflect.Interface:
		// Inspect first non-nil row to determine actual kind.
		for _, row := range rows {
			v := reflect.ValueOf(row)
			if !v.IsValid() || v.IsNil() {
				continue
			}
			elem := v.Elem()
			switch elem.Kind() {
			case reflect.Map:
				if elem.Type().Key().Kind() == reflect.String {
					return columnsFromMap(rows)
				}
				return nil
			case reflect.Slice:
				return columnsFromSlice(rows)
			default:
				return nil
			}
		}
		return nil
	default:
		return nil
	}
}

// columnsFromMap discovers columns from map-typed rows (map[string]any).
// Keys are collected in first-appearance order across all rows.
func columnsFromMap[T any](rows []T) []Column[T] {
	if len(rows) == 0 {
		return nil
	}

	// Discover keys in first-appearance order.
	seen := make(map[string]struct{})
	var keys []string
	for _, row := range rows {
		v := reflect.ValueOf(row)
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		if v.Kind() != reflect.Map {
			continue
		}
		for _, k := range v.MapKeys() {
			name := k.String()
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				keys = append(keys, name)
			}
		}
	}

	cols := make([]Column[T], 0, len(keys))
	for _, key := range keys {
		category := inferMapColumnType(key, rows)
		col := makeMapColumn[T](key, category)
		applyTypeDefaults(&col, category, collectDistinctValues(col.ValueGetter, rows))
		cols = append(cols, col)
	}
	return cols
}

// makeMapColumn builds a Column for a map column with the given key and type category.
func makeMapColumn[T any](key, category string) Column[T] {
	col := Column[T]{
		ColumnID:   key,
		HeaderName: key,
		Sortable:   true,
		Filterable: true,
	}

	switch category {
	case "int", "number":
		col.ValueGetter = func(row T) any {
			v := mapIndex(row, key)
			if v == nil {
				return nil
			}
			if f, ok := v.(float64); ok {
				return f
			}
			return v
		}
	case "bool":
		col.ValueGetter = func(row T) any {
			v := mapIndex(row, key)
			if v == nil {
				return nil
			}
			if b, ok := v.(bool); ok {
				return b
			}
			return v
		}
	case "time":
		col.ValueGetter = func(row T) any {
			v := mapIndex(row, key)
			if v == nil {
				return nil
			}
			if s, ok := v.(string); ok {
				if t, ok := parseTimeString(s); ok {
					return t
				}
			}
			return v
		}
	default: // string
		col.ValueGetter = func(row T) any {
			v := mapIndex(row, key)
			if v == nil {
				return nil
			}
			return fmt.Sprint(v)
		}
	}

	return col
}

// mapIndex extracts a value from a map-typed row by string key.
func mapIndex(row any, key string) any {
	v := reflect.ValueOf(row)
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() != reflect.Map {
		return nil
	}
	result := v.MapIndex(reflect.ValueOf(key))
	if !result.IsValid() {
		return nil
	}
	iface := result.Interface()
	// Unwrap interface values.
	rv := reflect.ValueOf(iface)
	if rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil
		}
		return rv.Elem().Interface()
	}
	return iface
}

// inferMapColumnType inspects non-nil values for a key across map rows
// and returns a type category: "bool", "int", "number", "time", or "string".
func inferMapColumnType[T any](key string, rows []T) string {
	allBool := true
	allFloat := true
	allInt := true
	allTime := true
	count := 0

	for _, row := range rows {
		v := mapIndex(row, key)
		if v == nil {
			continue
		}
		count++

		switch val := v.(type) {
		case bool:
			allFloat = false
			allInt = false
			allTime = false
		case float64:
			allBool = false
			allTime = false
			if val != math.Trunc(val) {
				allInt = false
			}
		case string:
			allBool = false
			allFloat = false
			allInt = false
			parsed := false
			for _, layout := range timeFormats {
				if _, err := time.Parse(layout, val); err == nil {
					parsed = true
					break
				}
			}
			if !parsed {
				allTime = false
			}
		default:
			allBool = false
			allFloat = false
			allInt = false
			allTime = false
		}
	}

	if count == 0 {
		return "string"
	}
	if allBool {
		return "bool"
	}
	if allFloat && allInt {
		return "int"
	}
	if allFloat {
		return "number"
	}
	if allTime {
		return "time"
	}
	return "string"
}

// columnsFromSlice discovers columns from slice-typed rows ([]any).
// Each element is expected to be a struct; fields are collected in first-appearance order.
func columnsFromSlice[T any](rows []T) []Column[T] {
	if len(rows) == 0 {
		return nil
	}

	type fieldMeta struct {
		name   string
		goType reflect.Type
	}

	seen := make(map[string]struct{})
	var fields []fieldMeta

	for _, row := range rows {
		v := reflect.ValueOf(row)
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		if v.Kind() != reflect.Slice {
			continue
		}
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Interface {
				elem = elem.Elem()
			}
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			if elem.Kind() != reflect.Struct {
				continue
			}
			et := elem.Type()
			for j := 0; j < et.NumField(); j++ {
				sf := et.Field(j)
				if !sf.IsExported() {
					continue
				}
				if _, ok := seen[sf.Name]; !ok {
					seen[sf.Name] = struct{}{}
					fields = append(fields, fieldMeta{name: sf.Name, goType: sf.Type})
				}
			}
		}
	}

	cols := make([]Column[T], 0, len(fields))
	for _, fi := range fields {
		category := classifyReflectType(fi.goType)
		fieldName := fi.name // capture

		getter := func(row T) any {
			return sliceFieldValue(row, fieldName)
		}

		col := Column[T]{
			ColumnID:    fieldName,
			HeaderName:  fieldName,
			ValueGetter: getter,
			Sortable:    true,
			Filterable:  true,
		}
		applyTypeDefaults(&col, category, collectDistinctValues(getter, rows))
		cols = append(cols, col)
	}
	return cols
}

// sliceFieldValue extracts a named field from a slice-of-structs row.
func sliceFieldValue(row any, fieldName string) any {
	v := reflect.ValueOf(row)
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		return nil
	}
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Interface {
			elem = elem.Elem()
		}
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		if elem.Kind() != reflect.Struct {
			continue
		}
		fv := elem.FieldByName(fieldName)
		if fv.IsValid() {
			return fv.Interface()
		}
	}
	return nil
}

// filterForType returns an appropriate Filter for the given reflect.Type.
func filterForType(t reflect.Type) filter.Filter {
	switch classifyReflectType(t) {
	case "int", "float":
		return filter.NewNumberFilter()
	case "bool":
		return filter.NewBoolFilter()
	case "time":
		return filter.NewTimeFilter()
	default:
		return filter.NewTextFilter()
	}
}

// classifyReflectType maps a reflect.Type to a category string.
func classifyReflectType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "int"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Bool:
		return "bool"
	default:
		if t == reflect.TypeOf(time.Time{}) {
			return "time"
		}
		return "string"
	}
}

// applyTypeDefaults sets Filter and sizing on a Column based on the type category.
func applyTypeDefaults[T any](col *Column[T], category string, distinctValues []string) {
	rightAligned := lipgloss.NewStyle().Align(lipgloss.Right)

	switch category {
	case "int":
		col.Filter = filter.NewNumberFilter()
		col.CellStyle = func(_ any, _ T) lipgloss.Style { return rightAligned }
		col.Width = 12
	case "float", "number":
		col.Filter = filter.NewNumberFilter()
		col.CellStyle = func(_ any, _ T) lipgloss.Style { return rightAligned }
		col.Width = 14
	case "bool":
		col.Filter = filter.NewBoolFilter()
		col.Width = 10
	case "time":
		col.Filter = filter.NewTimeFilter()
		col.Width = 14
	default: // string
		if len(distinctValues) > 0 && len(distinctValues) <= 20 {
			col.Filter = filter.NewSetFilter(distinctValues...)
		} else {
			col.Filter = filter.NewTextFilter()
		}
		col.MinWidth = 8
		col.Flex = 1
	}
}

// collectDistinctValues scans rows and returns distinct non-empty string representations.
func collectDistinctValues[T any](getter func(T) any, rows []T) []string {
	seen := make(map[string]struct{})
	var vals []string
	for _, row := range rows {
		v := getter(row)
		if v == nil {
			continue
		}
		s := fmt.Sprint(v)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			vals = append(vals, s)
		}
	}
	return vals
}

// parseTimeString attempts to parse a string with known time layouts.
func parseTimeString(s string) (time.Time, bool) {
	for _, layout := range timeFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
