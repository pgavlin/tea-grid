package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/grid"
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
	cols := data.FromType[Employee]()

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
		grid.WithQuickFilter[Employee](true),
		grid.WithFocused[Employee](true),
	)

	p := tea.NewProgram(model{grid: g}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
