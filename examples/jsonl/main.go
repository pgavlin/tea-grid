package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/grid"
	"github.com/pgavlin/tea-grid/selection"
)

// Row is a JSON object decoded as a map.
type Row = map[string]any

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

	// Discover columns before adding internal keys.
	cols := column.FromRows[Row](rows)

	// Assign row indices for use as row IDs.
	for i := range rows {
		rows[i]["__index__"] = i
	}

	// Pin first column left.
	if len(cols) > 0 {
		cols[0].Pinned = column.PinLeft
		cols[0].MinWidth = 16
		cols[0].Flex = 2
	}

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

	p := tea.NewProgram(model{grid: g}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
