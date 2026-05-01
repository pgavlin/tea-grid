// querybar demonstrates the GitHub-style query bar with all built-in
// filter types: text (substring), number (>5, 5..20), set (comma-OR),
// bool (true/false), time (date and range), and multiset (AND-of-
// includes for slice-valued columns).
//
// Sample queries to try (after pressing '/'):
//
//	State:open
//	State:open,closed
//	Priority:>3
//	Active:true
//	Created:2026-01-01..2026-12-31
//	Labels:bug Labels:urgent
//	memory leak                       (bare terms — substring match across columns)
//	State:open critical               (mix of clauses and bare terms)
//
// Press '/' to focus the bar, Enter to submit, Esc to cancel.
package main

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grid"
)

type Issue struct {
	Title    string
	State    string
	Priority int
	Active   bool
	Created  time.Time
	Labels   []string
}

func columns() []data.Column[Issue] {
	return []data.Column[Issue]{
		{
			ColumnID:   "Title",
			HeaderName: "Title",
			Value:      func(i Issue) any { return i.Title },
			Filter:     filter.NewTextFilter(),
			Filterable: true,
			Flex:       2,
		},
		{
			ColumnID:   "State",
			HeaderName: "State",
			Value:      func(i Issue) any { return i.State },
			Filter:     filter.NewSetFilter("open", "closed", "draft"),
			Filterable: true,
			Width:      10,
		},
		{
			ColumnID:   "Priority",
			HeaderName: "Priority",
			Value:      func(i Issue) any { return i.Priority },
			Filter:     filter.NewNumberFilter(),
			Filterable: true,
			Width:      10,
		},
		{
			ColumnID:   "Active",
			HeaderName: "Active",
			Value:      func(i Issue) any { return i.Active },
			Filter:     filter.NewBoolFilter(),
			Filterable: true,
			Width:      8,
		},
		{
			ColumnID:   "Created",
			HeaderName: "Created",
			Value:      func(i Issue) any { return i.Created },
			Filter:     filter.NewTimeFilter(),
			Filterable: true,
			Width:      14,
		},
		{
			ColumnID:   "Labels",
			HeaderName: "Labels",
			Value:      func(i Issue) any { return i.Labels },
			Filter:     filter.NewMultiSetFilter(),
			Filterable: true,
			Flex:       1,
		},
	}
}

func sampleData() []Issue {
	mustParse := func(s string) time.Time {
		t, _ := time.Parse("2006-01-02", s)
		return t
	}
	return []Issue{
		{Title: "memory leak in worker pool", State: "open", Priority: 5, Active: true, Created: mustParse("2026-01-15"), Labels: []string{"bug", "urgent"}},
		{Title: "add dark mode", State: "open", Priority: 2, Active: true, Created: mustParse("2026-02-03"), Labels: []string{"feature", "ux"}},
		{Title: "flaky test in checkout", State: "closed", Priority: 4, Active: false, Created: mustParse("2026-01-20"), Labels: []string{"bug", "test"}},
		{Title: "rewrite billing module", State: "draft", Priority: 1, Active: true, Created: mustParse("2026-03-01"), Labels: []string{"refactor"}},
		{Title: "investigate pager noise", State: "open", Priority: 3, Active: true, Created: mustParse("2026-03-12"), Labels: []string{"ops", "urgent"}},
	}
}

type model struct {
	grid grid.Model[Issue]
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.grid.SetWidth(msg.Width)
		m.grid.SetHeight(msg.Height)
	case tea.KeyPressMsg:
		if !m.grid.Filtering() && (msg.String() == "q" || msg.String() == "ctrl+c") {
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.grid, cmd = m.grid.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	v := tea.NewView(m.grid.View())
	v.AltScreen = true
	return v
}

func main() {
	g := grid.New(
		grid.WithColumns(columns()),
		grid.WithRows(sampleData()),
		grid.WithRowID(func(i Issue) string { return i.Title }),
		grid.WithQueryBar[Issue](),
		grid.WithFocused[Issue](true),
	)
	p := tea.NewProgram(model{grid: g})
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
