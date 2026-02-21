package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grid"
	"github.com/pgavlin/tea-grid/selection"
)

type Row = []string

type model struct {
	grid grid.Model[Row]
}

func (m model) Init() tea.Cmd { return m.grid.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.grid.SetWidth(msg.Width)
		m.grid.SetHeight(msg.Height)
	}
	var cmd tea.Cmd
	m.grid, cmd = m.grid.Update(msg)
	return m, cmd
}

func (m model) View() string { return m.grid.View() }

func loadCSV(path string) ([]string, []Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(records) < 1 {
		return nil, nil, fmt.Errorf("csv file is empty")
	}

	headers := records[0]
	rows := make([]Row, len(records)-1)
	for i, rec := range records[1:] {
		rows[i] = rec
	}
	return headers, rows, nil
}

func inferType(col int, rows []Row) string {
	allBool := true
	allNumber := true

	for _, row := range rows {
		if col >= len(row) {
			continue
		}
		v := strings.TrimSpace(row[col])
		if v == "" {
			continue
		}

		lower := strings.ToLower(v)
		if lower != "true" && lower != "false" {
			allBool = false
		}
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			allNumber = false
		}

		if !allBool && !allNumber {
			return "string"
		}
	}

	if allBool {
		return "bool"
	}
	if allNumber {
		return "number"
	}
	return "string"
}

func buildColumns(headers []string, rows []Row) []data.Column[Row] {
	cols := make([]data.Column[Row], len(headers))

	rightAligned := lipgloss.NewStyle().Align(lipgloss.Right)

	for i, hdr := range headers {
		idx := i // capture
		typ := inferType(idx, rows)

		col := data.Column[Row]{
			ColumnID:      fmt.Sprintf("col%d", idx),
			HeaderName: hdr,
			Sortable:   true,
			Filterable: true,
		}

		switch typ {
		case "number":
			col.ValueGetter = func(r Row) any {
				if idx >= len(r) {
					return 0.0
				}
				f, _ := strconv.ParseFloat(strings.TrimSpace(r[idx]), 64)
				return f
			}
			col.Filter = filter.NewNumberFilter()
			col.ValueFormatter = func(v any, _ Row) string {
				return formatNumber(v.(float64))
			}
			col.CellStyle = func(_ any, _ Row) lipgloss.Style { return rightAligned }
			col.Width = 14

		case "bool":
			col.ValueGetter = func(r Row) any {
				if idx >= len(r) {
					return false
				}
				return strings.EqualFold(strings.TrimSpace(r[idx]), "true")
			}
			col.Filter = filter.NewBoolFilter()
			col.ValueFormatter = func(v any, _ Row) string {
				if v.(bool) {
					return "✓"
				}
				return "✗"
			}
			col.Width = 10

		default: // string
			col.ValueGetter = func(r Row) any {
				if idx >= len(r) {
					return ""
				}
				return r[idx]
			}

			// Use SetFilter for low-cardinality columns, TextFilter otherwise.
			distinct := collectDistinct(idx, rows)
			if len(distinct) > 0 && len(distinct) <= 20 {
				col.Filter = filter.NewSetFilter(distinct...)
			} else {
				col.Filter = filter.NewTextFilter()
			}
			col.MinWidth = 8
			col.Flex = 1
		}

		// Pin first column left.
		if i == 0 {
			col.Pinned = data.PinLeft
			col.MinWidth = 16
			col.Flex = 2
		}

		cols[i] = col
	}

	return cols
}

func collectDistinct(col int, rows []Row) []string {
	seen := make(map[string]struct{})
	var vals []string
	for _, row := range rows {
		if col >= len(row) {
			continue
		}
		v := strings.TrimSpace(row[col])
		if v == "" {
			continue
		}
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			vals = append(vals, v)
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
	// Insert commas from the right.
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
	path := "data.csv"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	headers, rows, err := loadCSV(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading CSV: %v\n", err)
		os.Exit(1)
	}

	cols := buildColumns(headers, rows)

	g := grid.New(
		grid.WithColumns(cols),
		grid.WithRows(rows),
		grid.WithRowID(func(r Row) string {
			if len(r) > 0 {
				return r[0]
			}
			return ""
		}),
		grid.WithSelection[Row](selection.SelectMulti),
		grid.WithQuickFilter[Row](true),
		grid.WithFocused[Row](true),
		grid.WithMultiSort[Row](true),
	)

	p := tea.NewProgram(model{grid: g}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
