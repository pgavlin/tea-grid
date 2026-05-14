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
			Sortable:   true,
			Flex:       2,
		},
		{
			ColumnID:   "State",
			HeaderName: "State",
			Value:      func(i Issue) any { return i.State },
			Filter:     filter.NewSetFilter("open", "closed", "draft"),
			Filterable: true,
			Sortable:   true,
			Width:      10,
		},
		{
			ColumnID:   "Priority",
			HeaderName: "Priority",
			Value:      func(i Issue) any { return i.Priority },
			Filter:     filter.NewNumberFilter(),
			Filterable: true,
			Sortable:   true,
			Width:      12,
		},
		{
			ColumnID:   "Active",
			HeaderName: "Active",
			Value:      func(i Issue) any { return i.Active },
			Filter:     filter.NewBoolFilter(),
			Filterable: true,
			Sortable:   true,
			Width:      8,
		},
		{
			ColumnID:   "Created",
			HeaderName: "Created",
			Value:      func(i Issue) any { return i.Created },
			Filter:     filter.NewTimeFilter(),
			Filterable: true,
			Sortable:   true,
			Width:      14,
		},
		{
			ColumnID:   "Labels",
			HeaderName: "Labels",
			Value:      func(i Issue) any { return i.Labels },
			Filter:     filter.NewMultiSetFilter(),
			Filterable: true,
			Sortable:   true,
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
		{Title: "publish v2 release notes", State: "draft", Priority: 2, Active: true, Created: mustParse("2026-04-02"), Labels: []string{"docs", "release"}},
		{Title: "kafka consumer stalls under load", State: "open", Priority: 5, Active: true, Created: mustParse("2025-11-28"), Labels: []string{"bug", "urgent", "ops"}},
		{Title: "migrate auth to OIDC", State: "open", Priority: 4, Active: true, Created: mustParse("2025-12-10"), Labels: []string{"security", "refactor"}},
		{Title: "deprecate v1 schema endpoints", State: "draft", Priority: 3, Active: false, Created: mustParse("2026-02-18"), Labels: []string{"api", "breaking"}},
		{Title: "redesign onboarding tour", State: "open", Priority: 2, Active: true, Created: mustParse("2026-04-22"), Labels: []string{"feature", "ux"}},
		{Title: "fix CSV import unicode handling", State: "closed", Priority: 3, Active: false, Created: mustParse("2025-10-04"), Labels: []string{"bug"}},
		{Title: "add export to Parquet", State: "open", Priority: 1, Active: true, Created: mustParse("2026-05-01"), Labels: []string{"feature"}},
		{Title: "remove legacy mobile API", State: "draft", Priority: 4, Active: true, Created: mustParse("2026-03-22"), Labels: []string{"refactor", "breaking"}},
		{Title: "intermittent 502s from edge", State: "open", Priority: 5, Active: true, Created: mustParse("2026-04-30"), Labels: []string{"bug", "urgent", "ops"}},
		{Title: "support SSO for enterprise tier", State: "open", Priority: 3, Active: true, Created: mustParse("2026-02-26"), Labels: []string{"feature", "security"}},
		{Title: "audit prometheus cardinality", State: "closed", Priority: 2, Active: false, Created: mustParse("2025-09-18"), Labels: []string{"ops"}},
		{Title: "rate-limit unauthenticated POSTs", State: "open", Priority: 4, Active: true, Created: mustParse("2026-03-08"), Labels: []string{"security", "ops"}},
		{Title: "i18n for billing emails", State: "draft", Priority: 1, Active: true, Created: mustParse("2026-05-06"), Labels: []string{"feature", "docs"}},
		{Title: "incident retro: 2025-11-22 outage", State: "closed", Priority: 3, Active: false, Created: mustParse("2025-11-25"), Labels: []string{"docs", "ops"}},
		{Title: "drop Python 3.9 from CI matrix", State: "closed", Priority: 1, Active: false, Created: mustParse("2026-01-08"), Labels: []string{"ops", "test"}},
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
