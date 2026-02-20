package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/grid"
)

type Employee struct {
	Name       string
	Department string
	Salary     float64
	Active     bool
}

func main() {
	cols := column.FromType[Employee]()

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

	p := tea.NewProgram(g, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
