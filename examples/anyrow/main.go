package main

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grid"
	"github.com/pgavlin/tea-grid/selection"
)

// Row is a slice of any, where each element is a struct value.
// Different elements may be different struct types.
type Row = []any

// Sample struct types that share the Name field but differ otherwise.

type Person struct {
	Name  string
	Age   int
	Email string
}

type Product struct {
	Name    string
	Price   float64
	InStock bool
}

type Event struct {
	Name      string
	Date      time.Time
	Attendees int
}

// fieldInfo tracks a discovered field's name, type, and first-seen order.
type fieldInfo struct {
	name    string
	goType  reflect.Type
	order   int
}

// discoverColumns iterates all row elements, reflects on each struct value,
// and returns field metadata in first-appearance order.
func discoverColumns(rows []Row) []fieldInfo {
	seen := make(map[string]*fieldInfo)
	var fields []fieldInfo
	order := 0

	for _, row := range rows {
		for _, elem := range row {
			v := reflect.ValueOf(elem)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			if v.Kind() != reflect.Struct {
				continue
			}
			t := v.Type()
			for i := range t.NumField() {
				sf := t.Field(i)
				if !sf.IsExported() {
					continue
				}
				if _, ok := seen[sf.Name]; !ok {
					fi := fieldInfo{name: sf.Name, goType: sf.Type, order: order}
					seen[sf.Name] = &fi
					fields = append(fields, fi)
					order++
				}
			}
		}
	}
	return fields
}

// makeValueGetter returns a function that extracts the named field from a Row.
// It iterates the row's elements and reflects to find the field.
// Returns nil if no element has the field.
func makeValueGetter(fieldName string) func(Row) any {
	return func(row Row) any {
		for _, elem := range row {
			v := reflect.ValueOf(elem)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			if v.Kind() != reflect.Struct {
				continue
			}
			fv := v.FieldByName(fieldName)
			if fv.IsValid() {
				return fv.Interface()
			}
		}
		return nil
	}
}

// typeCategory classifies a reflect.Type into a category string.
func typeCategory(t reflect.Type) string {
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

// buildColumns creates column definitions from discovered field metadata.
func buildColumns(fields []fieldInfo, rows []Row) []column.ColDef[Row] {
	cols := make([]column.ColDef[Row], len(fields))
	rightAligned := lipgloss.NewStyle().Align(lipgloss.Right)

	for i, fi := range fields {
		getter := makeValueGetter(fi.name)
		cat := typeCategory(fi.goType)

		col := column.ColDef[Row]{
			ColID:       fi.name,
			HeaderName:  fi.name,
			ValueGetter: getter,
			Sortable:    true,
			Filterable:  true,
		}

		switch cat {
		case "int":
			col.Filter = filter.NewNumberFilter()
			col.ValueFormatter = func(v any, _ Row) string {
				if v == nil {
					return ""
				}
				return formatInt(toInt64(v))
			}
			col.CellStyle = func(_ any, _ Row) lipgloss.Style { return rightAligned }
			col.Width = 12

		case "float":
			col.Filter = filter.NewNumberFilter()
			col.ValueFormatter = func(v any, _ Row) string {
				if v == nil {
					return ""
				}
				return formatNumber(toFloat64(v))
			}
			col.CellStyle = func(_ any, _ Row) lipgloss.Style { return rightAligned }
			col.Width = 14

		case "bool":
			col.Filter = filter.NewBoolFilter()
			col.ValueFormatter = func(v any, _ Row) string {
				if v == nil {
					return ""
				}
				if v.(bool) {
					return "✓"
				}
				return "✗"
			}
			col.Width = 10

		case "time":
			col.Filter = filter.NewTimeFilter()
			col.ValueFormatter = func(v any, _ Row) string {
				if v == nil {
					return ""
				}
				return v.(time.Time).Format("2006-01-02")
			}
			col.Width = 14

		default: // string
			distinct := collectDistinctForField(fi.name, rows)
			if len(distinct) > 0 && len(distinct) <= 20 {
				col.Filter = filter.NewSetFilter(distinct...)
			} else {
				col.Filter = filter.NewTextFilter()
			}
			col.ValueFormatter = func(v any, _ Row) string {
				if v == nil {
					return ""
				}
				return fmt.Sprint(v)
			}
			col.MinWidth = 8
			col.Flex = 1
		}

		// Pin first column left.
		if i == 0 {
			col.Pinned = column.PinLeft
			col.MinWidth = 16
			col.Flex = 2
		}

		cols[i] = col
	}

	return cols
}

// collectDistinctForField collects distinct string values for a field across all rows.
func collectDistinctForField(fieldName string, rows []Row) []string {
	getter := makeValueGetter(fieldName)
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

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	default:
		return 0
	}
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case int32:
		return int64(n)
	case int16:
		return int64(n)
	case int8:
		return int64(n)
	case uint:
		return int64(n)
	case uint64:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func formatNumber(f float64) string {
	if f == float64(int64(f)) {
		return formatInt(int64(f))
	}
	return strconv.FormatFloat(f, 'f', 2, 64)
}

func formatInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	start := len(s) % 3
	if start == 0 {
		start = 3
	}
	b.WriteString(s[:start])
	for i := start; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func nameGetter(r Row) string {
	for _, elem := range r {
		v := reflect.ValueOf(elem)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			continue
		}
		fv := v.FieldByName("Name")
		if fv.IsValid() {
			return fmt.Sprint(fv.Interface())
		}
	}
	return ""
}

func main() {
	rows := []Row{
		{Person{Name: "Alice", Age: 30, Email: "alice@example.com"}},
		{Person{Name: "Bob", Age: 25, Email: "bob@example.com"}},
		{Product{Name: "Widget", Price: 9.99, InStock: true}},
		{Product{Name: "Gadget", Price: 24.50, InStock: false}},
		{Event{Name: "GoConf", Date: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC), Attendees: 500}},
		{Event{Name: "HackDay", Date: time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), Attendees: 120}},
		{Person{Name: "Carol", Age: 42, Email: "carol@example.com"}},
		{Product{Name: "Doohickey", Price: 4.75, InStock: true}},
		{Event{Name: "MeetUp", Date: time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC), Attendees: 45}},
		{Person{Name: "Dave", Age: 37, Email: "dave@example.com"}},
	}

	fields := discoverColumns(rows)
	cols := buildColumns(fields, rows)

	g := grid.New(
		grid.WithColumns(cols),
		grid.WithRows(rows),
		grid.WithRowID(nameGetter),
		grid.WithSelection[Row](selection.SelectMulti),
		grid.WithQuickFilter[Row](true),
		grid.WithFocused[Row](true),
		grid.WithMultiSort[Row](true),
	)

	p := tea.NewProgram(g, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
