package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/grid"
	"github.com/pgavlin/tea-grid/selection"
)

type Employee struct {
	Name       string
	Department string
	Title      string
	Salary     float64
}

var statusStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252")).
	Background(lipgloss.Color("235")).
	Padding(0, 1)

type model struct {
	grid   grid.Model[Employee]
	width  int
	height int
}

func (m model) Init() tea.Cmd { return m.grid.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.grid.SetWidth(msg.Width)
		m.grid.SetHeight(msg.Height - 1) // reserve one line for status bar
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.grid, cmd = m.grid.Update(msg)
	return m, cmd
}

func (m model) View() string {
	gridView := m.grid.View()

	selected := m.grid.SelectedRows()
	var status string
	if len(selected) == 0 {
		status = "No rows selected  |  Space: toggle  Ctrl+A: select all  Esc: clear"
	} else {
		names := make([]string, len(selected))
		for i, emp := range selected {
			names[i] = emp.Name
		}
		status = fmt.Sprintf("%d selected: %s", len(selected), strings.Join(names, ", "))
	}

	bar := statusStyle.Width(m.width).Render(status)
	return gridView + "\n" + bar
}

func main() {
	cols := data.FromType[Employee]()

	rows := []Employee{
		{"Alice Johnson", "Engineering", "Senior Engineer", 145000},
		{"Bob Smith", "Engineering", "Staff Engineer", 160000},
		{"Carol Davis", "Marketing", "Marketing Manager", 110000},
		{"Dave Wilson", "Marketing", "Content Lead", 95000},
		{"Eve Brown", "Sales", "Account Executive", 88000},
		{"Frank Lee", "Sales", "Sales Director", 135000},
		{"Grace Kim", "Engineering", "Junior Engineer", 95000},
		{"Hank Patel", "Product", "Product Manager", 125000},
		{"Ivy Chen", "Product", "UX Designer", 105000},
		{"Jack Turner", "Engineering", "DevOps Engineer", 140000},
	}

	g := grid.New(
		grid.WithColumns(cols),
		grid.WithRows(rows),
		grid.WithRowID(func(e Employee) string { return e.Name }),
		grid.WithSelection[Employee](selection.SelectMulti),
		grid.WithSelectionColumn[Employee](true),
		grid.WithQuickFilter[Employee](true),
		grid.WithFocused[Employee](true),
	)

	p := tea.NewProgram(model{grid: g}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
