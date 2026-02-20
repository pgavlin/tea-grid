package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
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

// Row is a JSON object decoded as a map.
type Row = map[string]any

// loadJSONL reads a JSONL file and returns a slice of Row.
// Non-object lines and blank lines are skipped.
func loadJSONL(path string) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rows []Row
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var v any
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			continue
		}
		if obj, ok := v.(map[string]any); ok {
			rows = append(rows, obj)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

// discoverColumns returns column names in first-appearance order,
// computed as the union of all keys across all rows.
func discoverColumns(rows []Row) []string {
	seen := make(map[string]struct{})
	var cols []string
	for _, row := range rows {
		for key := range row {
			if key == "__index__" {
				continue
			}
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				cols = append(cols, key)
			}
		}
	}
	return cols
}

var timeFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// inferType inspects non-nil values for a column across all rows
// and returns a type category string.
func inferType(col string, rows []Row) string {
	allBool := true
	allFloat := true
	allInt := true
	allTime := true
	count := 0

	for _, row := range rows {
		v, ok := row[col]
		if !ok || v == nil {
			continue
		}
		count++

		switch v.(type) {
		case bool:
			allFloat = false
			allInt = false
			allTime = false
		case float64:
			allBool = false
			allTime = false
			f := v.(float64)
			if f != math.Trunc(f) {
				allInt = false
			}
		case string:
			allBool = false
			allFloat = false
			allInt = false
			s := v.(string)
			parsed := false
			for _, layout := range timeFormats {
				if _, err := time.Parse(layout, s); err == nil {
					parsed = true
					break
				}
			}
			if !parsed {
				allTime = false
			}
		default:
			// nested objects, arrays, etc.
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

// parseTime attempts to parse a string with known time layouts.
func parseTime(s string) (time.Time, bool) {
	for _, layout := range timeFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func buildColumns(colNames []string, rows []Row) []column.ColDef[Row] {
	cols := make([]column.ColDef[Row], len(colNames))
	rightAligned := lipgloss.NewStyle().Align(lipgloss.Right)

	for i, name := range colNames {
		key := name // capture
		typ := inferType(key, rows)

		col := column.ColDef[Row]{
			ColID:      key,
			HeaderName: key,
			Sortable:   true,
			Filterable: true,
		}

		switch typ {
		case "int":
			col.ValueGetter = func(r Row) any {
				v := r[key]
				if v == nil {
					return nil
				}
				if f, ok := v.(float64); ok {
					return f
				}
				return v
			}
			col.Filter = filter.NewNumberFilter()
			col.ValueFormatter = func(v any, _ Row) string {
				if v == nil {
					return ""
				}
				if f, ok := v.(float64); ok {
					return formatInt(int64(f))
				}
				return fmt.Sprint(v)
			}
			col.CellStyle = func(_ any, _ Row) lipgloss.Style { return rightAligned }
			col.Width = 12

		case "number":
			col.ValueGetter = func(r Row) any {
				v := r[key]
				if v == nil {
					return nil
				}
				if f, ok := v.(float64); ok {
					return f
				}
				return v
			}
			col.Filter = filter.NewNumberFilter()
			col.ValueFormatter = func(v any, _ Row) string {
				if v == nil {
					return ""
				}
				if f, ok := v.(float64); ok {
					return formatNumber(f)
				}
				return fmt.Sprint(v)
			}
			col.CellStyle = func(_ any, _ Row) lipgloss.Style { return rightAligned }
			col.Width = 14

		case "bool":
			col.ValueGetter = func(r Row) any {
				v := r[key]
				if v == nil {
					return nil
				}
				if b, ok := v.(bool); ok {
					return b
				}
				return v
			}
			col.Filter = filter.NewBoolFilter()
			col.ValueFormatter = func(v any, _ Row) string {
				if v == nil {
					return ""
				}
				if b, ok := v.(bool); ok {
					if b {
						return "✓"
					}
					return "✗"
				}
				return fmt.Sprint(v)
			}
			col.Width = 10

		case "time":
			col.ValueGetter = func(r Row) any {
				v := r[key]
				if v == nil {
					return nil
				}
				if s, ok := v.(string); ok {
					if t, ok := parseTime(s); ok {
						return t
					}
				}
				return v
			}
			col.Filter = filter.NewTimeFilter()
			col.ValueFormatter = func(v any, _ Row) string {
				if v == nil {
					return ""
				}
				if t, ok := v.(time.Time); ok {
					return t.Format("2006-01-02")
				}
				return fmt.Sprint(v)
			}
			col.Width = 14

		default: // string
			col.ValueGetter = func(r Row) any {
				v := r[key]
				if v == nil {
					return nil
				}
				return fmt.Sprint(v)
			}

			distinct := collectDistinct(key, rows)
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

func collectDistinct(key string, rows []Row) []string {
	seen := make(map[string]struct{})
	var vals []string
	for _, row := range rows {
		v := row[key]
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

func main() {
	path := "data.jsonl"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	rows, err := loadJSONL(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading JSONL: %v\n", err)
		os.Exit(1)
	}
	if len(rows) == 0 {
		fmt.Fprintf(os.Stderr, "No rows loaded from %s\n", path)
		os.Exit(1)
	}

	// Assign row indices for use as row IDs.
	for i := range rows {
		rows[i]["__index__"] = i
	}

	colNames := discoverColumns(rows)
	cols := buildColumns(colNames, rows)

	g := grid.New(
		grid.WithColumns(cols),
		grid.WithRows(rows),
		grid.WithRowID(func(r Row) string {
			return strconv.Itoa(r["__index__"].(int))
		}),
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
