package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grid"
	"github.com/pgavlin/tea-grid/selection"
)

type Employee struct {
	Name       string
	Department string
	Salary     float64
	Active     bool
}

type model struct {
	grid grid.Model[Employee]
}

func (m model) Init() tea.Cmd { return m.grid.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.grid.SetWidth(msg.Width)
		m.grid.SetHeight(msg.Height)
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.grid, cmd = m.grid.Update(msg)
	return m, cmd
}

func (m model) View() string { return m.grid.View() }

func main() {
	cols := []data.Column[Employee]{
		{
			ColumnID:       "name",
			HeaderName:  "Employee Name",
			ValueGetter: func(e Employee) any { return e.Name },
			Pinned:      data.PinLeft,
			MinWidth:    20,
			Flex:        2,
			Filterable:  true,
			Filter:      filter.NewTextFilter(),
		},
		{
			ColumnID:       "department",
			HeaderName:  "Dept",
			ValueGetter: func(e Employee) any { return e.Department },
			Width:       15,
			Sortable:    true,
			Filterable:  true,
			Filter:      filter.NewSetFilter("Engineering", "Marketing", "Sales"),
		},
		{
			ColumnID:       "salary",
			HeaderName:  "Salary",
			ValueGetter: func(e Employee) any { return e.Salary },
			Width:       12,
			Sortable:    true,
			Filterable:  true,
			Filter:      filter.NewNumberFilter(),
			ValueFormatter: func(v any, _ Employee) string {
				return fmt.Sprintf("$%.0f", v.(float64))
			},
			AggFunc: "sum",
		},
		{
			ColumnID:       "active",
			HeaderName:  "Active",
			ValueGetter: func(e Employee) any { return e.Active },
			Width:       8,
			Filterable:  true,
			Filter:      filter.NewBoolFilter(),
			ValueFormatter: func(v any, _ Employee) string {
				if v.(bool) {
					return "✓"
				}
				return "✗"
			},
		},
	}

	rows := []Employee{
		{"Alice Johnson", "Engineering", 145000, true},
		{"Bob Smith", "Engineering", 130000, true},
		{"Carol Davis", "Marketing", 95000, false},
		{"Dave Wilson", "Marketing", 105000, true},
		{"Eve Brown", "Sales", 88000, true},
	}

	g := grid.New(
		grid.WithColumns(cols),
		grid.WithRows(rows),
		grid.WithRowID(func(e Employee) string { return e.Name }),
		grid.WithSelection[Employee](selection.SelectMulti),
		grid.WithQuickFilter[Employee](true),
		grid.WithGrouping[Employee]("department"),
		grid.WithGroupDefaultExpanded[Employee](-1),
		grid.WithFocused[Employee](true),
	)

	p := tea.NewProgram(model{grid: g}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
