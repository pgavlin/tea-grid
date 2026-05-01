package grid

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
)

var updateGolden = flag.Bool("update-golden", false, "update golden files")

// goldenTest compares output against a golden file, updating it when -update-golden is set.
func goldenTest(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *updateGolden {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file %s not found; run with -update-golden to create it: %v", path, err)
	}
	if got != string(want) {
		t.Errorf("output does not match golden file %s\n\ngot:\n%s\n\nwant:\n%s", path, got, string(want))
	}
}

// plainStyles returns styles with no ANSI colors for deterministic golden output.
func plainStyles() Styles {
	return Styles{
		Table:        lipgloss.NewStyle(),
		Header:       lipgloss.NewStyle(),
		HeaderCell:   lipgloss.NewStyle().Padding(0, 1),
		SortAsc:      "^",
		SortDesc:     "v",
		Cell:         lipgloss.NewStyle().Padding(0, 1),
		CellFocused:  lipgloss.NewStyle().Padding(0, 1),
		CellSelected: lipgloss.NewStyle().Padding(0, 1),
		CellEvenRow:  lipgloss.NewStyle().Padding(0, 1),
		CellOddRow:   lipgloss.NewStyle().Padding(0, 1),
		CellPinned:   lipgloss.NewStyle().Padding(0, 1),

		PinnedLeft:   lipgloss.NewStyle(),
		PinnedRight:  lipgloss.NewStyle(),
		PinSeparator: "|",
		ScrollLeft:   "<",
		ScrollRight:  ">",

		GroupRow:       lipgloss.NewStyle().Padding(0, 1),
		GroupExpanded:  "v",
		GroupCollapsed: ">",
		GroupIndent:    2,

		Border:       lipgloss.NormalBorder(),
		BorderHeader: true,
		BorderRow:    false,
		BorderColumn: true,

		FilterInput:  lipgloss.NewStyle().Padding(0, 1),
		FilterMatch:  lipgloss.NewStyle(),
		FilterActive: "*",

		EditorInput: lipgloss.NewStyle(),
		EditorError: lipgloss.NewStyle(),
	}
}

// bgStyles returns styles with distinct background colors so golden files
// can demonstrate that background fills the full cell height.
func bgStyles() Styles {
	s := plainStyles()
	s.CellEvenRow = lipgloss.NewStyle().Padding(0, 1).Background(lipgloss.Color("22"))
	s.CellOddRow = lipgloss.NewStyle().Padding(0, 1).Background(lipgloss.Color("52"))
	s.CellFocused = lipgloss.NewStyle().Padding(0, 1).Background(lipgloss.Color("57"))
	return s
}

func TestGolden_MultiLineRow_SingleCol_Height2(t *testing.T) {
	cols := []data.Column[TestRow]{
		{
			ColumnID:   "Name",
			HeaderName: "Name",
			Value:      func(r TestRow) any { return r.Name },
			Width:      12,
			CellRenderer: data.CellRendererFunc[TestRow](func(ctx data.CellContext[TestRow]) string {
				return fmt.Sprintf("%s\n(details)", ctx.Value)
			}),
		},
	}
	m := New[TestRow](
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{"Alice", "Eng", 95000, true}, {"Bob", "Sales", 75000, false}}),
		WithWidth[TestRow](12),
		WithHeight[TestRow](10),
		WithRowHeight[TestRow](2),
		WithStyles[TestRow](bgStyles()),
		WithFocused[TestRow](true),
	)
	goldenTest(t, "multiline_single_col_height2", m.View())
}

func TestGolden_MultiLineRow_TwoCols_OnlyFirstMultiLine(t *testing.T) {
	cols := []data.Column[TestRow]{
		{
			ColumnID:   "Name",
			HeaderName: "Name",
			Value:      func(r TestRow) any { return r.Name },
			Width:      10,
			CellRenderer: data.CellRendererFunc[TestRow](func(ctx data.CellContext[TestRow]) string {
				return fmt.Sprintf("%s\nline2", ctx.Value)
			}),
		},
		{
			ColumnID:   "Dept",
			HeaderName: "Dept",
			Value:      func(r TestRow) any { return r.Department },
			Width:      10,
		},
	}
	m := New[TestRow](
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{"Alice", "Eng", 95000, true}, {"Bob", "Sales", 75000, false}}),
		WithWidth[TestRow](21),
		WithHeight[TestRow](10),
		WithRowHeight[TestRow](2),
		WithStyles[TestRow](bgStyles()),
		WithFocused[TestRow](true),
	)
	goldenTest(t, "multiline_two_cols_first_multiline", m.View())
}

func TestGolden_MultiLineRow_TwoCols_BothMultiLine(t *testing.T) {
	cols := []data.Column[TestRow]{
		{
			ColumnID:   "Name",
			HeaderName: "Name",
			Value:      func(r TestRow) any { return r.Name },
			Width:      10,
			CellRenderer: data.CellRendererFunc[TestRow](func(ctx data.CellContext[TestRow]) string {
				return fmt.Sprintf("%s\n(%s)", ctx.Value, "info")
			}),
		},
		{
			ColumnID:   "Dept",
			HeaderName: "Dept",
			Value:      func(r TestRow) any { return r.Department },
			Width:      10,
			CellRenderer: data.CellRendererFunc[TestRow](func(ctx data.CellContext[TestRow]) string {
				return fmt.Sprintf("%s\n---", ctx.Value)
			}),
		},
	}
	m := New[TestRow](
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{"Alice", "Eng", 95000, true}, {"Bob", "Sales", 75000, false}}),
		WithWidth[TestRow](21),
		WithHeight[TestRow](10),
		WithRowHeight[TestRow](2),
		WithStyles[TestRow](bgStyles()),
		WithFocused[TestRow](true),
	)
	goldenTest(t, "multiline_two_cols_both_multiline", m.View())
}

func TestGolden_MultiLineRow_Height3_ShortContent(t *testing.T) {
	cols := []data.Column[TestRow]{
		{
			ColumnID:   "Name",
			HeaderName: "Name",
			Value:      func(r TestRow) any { return r.Name },
			Width:      10,
			CellRenderer: data.CellRendererFunc[TestRow](func(ctx data.CellContext[TestRow]) string {
				// Only 1 line of content in a 3-line cell
				return fmt.Sprintf("%v", ctx.Value)
			}),
		},
		{
			ColumnID:   "Dept",
			HeaderName: "Dept",
			Value:      func(r TestRow) any { return r.Department },
			Width:      10,
			CellRenderer: data.CellRendererFunc[TestRow](func(ctx data.CellContext[TestRow]) string {
				return fmt.Sprintf("%s\nline2\nline3", ctx.Value)
			}),
		},
	}
	m := New[TestRow](
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{{"Alice", "Eng", 95000, true}}),
		WithWidth[TestRow](21),
		WithHeight[TestRow](10),
		WithRowHeight[TestRow](3),
		WithStyles[TestRow](bgStyles()),
		WithFocused[TestRow](true),
	)
	goldenTest(t, "multiline_height3_short_content", m.View())
}

func TestGolden_MultiLineRow_EvenOddBg(t *testing.T) {
	cols := []data.Column[TestRow]{
		{
			ColumnID:   "Name",
			HeaderName: "Name",
			Value:      func(r TestRow) any { return r.Name },
			Width:      10,
		},
		{
			ColumnID:   "Dept",
			HeaderName: "Dept",
			Value:      func(r TestRow) any { return r.Department },
			Width:      10,
		},
	}
	m := New[TestRow](
		WithColumns[TestRow](cols),
		WithRows[TestRow]([]TestRow{
			{"Alice", "Eng", 95000, true},
			{"Bob", "Sales", 75000, false},
			{"Carol", "Eng", 110000, true},
		}),
		WithWidth[TestRow](21),
		WithHeight[TestRow](15),
		WithRowHeight[TestRow](2),
		WithStyles[TestRow](bgStyles()),
		WithFocused[TestRow](true),
	)
	goldenTest(t, "multiline_even_odd_bg", m.View())
}

// queryBarCols returns a small column set with a SetFilter (state) and a
// TextFilter (title) suitable for query-bar golden tests.
func queryBarCols() []data.Column[TestRow] {
	return []data.Column[TestRow]{
		{
			ColumnID:   "Name",
			HeaderName: "Name",
			Value:      func(r TestRow) any { return r.Name },
			Width:      12,
			Filter:     filter.NewTextFilter(),
			Filterable: true,
		},
		{
			ColumnID:   "Department",
			HeaderName: "Department",
			Value:      func(r TestRow) any { return r.Department },
			Width:      14,
			Filter:     filter.NewSetFilter("Eng", "Sales", "Marketing"),
			Filterable: true,
		},
	}
}

func newQueryBarGolden(t *testing.T, opts ...Option[TestRow]) Model[TestRow] {
	t.Helper()
	defaults := []Option[TestRow]{
		WithColumns[TestRow](queryBarCols()),
		WithRows[TestRow]([]TestRow{
			{"Alice", "Eng", 95000, true},
			{"Bob", "Sales", 75000, false},
		}),
		WithRowID[TestRow](func(r TestRow) string { return r.Name }),
		WithWidth[TestRow](40),
		WithHeight[TestRow](8),
		WithStyles[TestRow](plainStyles()),
		WithFocused[TestRow](true),
		WithQueryBar[TestRow](),
	}
	return New(append(defaults, opts...)...)
}

func TestGolden_QueryBar_Collapsed(t *testing.T) {
	// Bar enabled but no filters and not editing → collapsed (no row).
	m := newQueryBarGolden(t)
	goldenTest(t, "querybar_collapsed", m.View())
}

func TestGolden_QueryBar_Active(t *testing.T) {
	// User pressed '/' to open the bar; empty editor with cursor.
	m := newQueryBarGolden(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	goldenTest(t, "querybar_active_empty", m.View())
}

func TestGolden_QueryBar_ActiveWithText(t *testing.T) {
	m := newQueryBarGolden(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "Department:Eng" {
		m, _ = m.Update(tea.KeyPressMsg{Code: rune(r), Text: string(r)})
	}
	goldenTest(t, "querybar_active_with_text", m.View())
}

func TestGolden_QueryBar_FilterActiveBarInactive(t *testing.T) {
	// Programmatic filter set; bar shows canonical text without cursor.
	m := newQueryBarGolden(t)
	sf := m.cols[1].Filter.(*filter.SetFilter)
	sf.Exclude("Sales")
	sf.Exclude("Marketing")
	m.invalidateQueryBar()
	goldenTest(t, "querybar_filter_active_bar_inactive", m.View())
}

func TestGolden_QueryBar_LossyAnnotation(t *testing.T) {
	m := newQueryBarGolden(t)
	tf := m.cols[0].Filter.(*filter.TextFilter)
	tf.SetRegex(true)
	tf.SetText("Al.*")
	m.invalidateQueryBar()
	goldenTest(t, "querybar_lossy_annotation", m.View())
}

func TestGolden_QueryBar_TabCompletion(t *testing.T) {
	m := newQueryBarGolden(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "Department:E" {
		m, _ = m.Update(tea.KeyPressMsg{Code: rune(r), Text: string(r)})
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	goldenTest(t, "querybar_tab_completion", m.View())
}
