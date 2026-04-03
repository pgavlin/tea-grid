package grid

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	gridsort "github.com/pgavlin/tea-grid/sort"
)

// benchRow is a representative row type sized similarly to real-world usage
// (e.g. a log record struct at ~80 bytes of data fields).
type benchRow struct {
	Name       string
	Department string
	City       string
	Country    string
	Salary     float64
	Age        int
	Active     bool
	Score      float64
}

var departments = []string{"Engineering", "Sales", "Marketing", "Support", "Finance", "Legal", "HR", "Product", "Design", "Operations"}
var cities = []string{"New York", "San Francisco", "London", "Berlin", "Tokyo", "Sydney", "Toronto", "Paris", "Singapore", "Mumbai"}
var countries = []string{"US", "UK", "DE", "JP", "AU", "CA", "FR", "SG", "IN", "BR"}

func makeBenchRows(n int) []benchRow {
	rows := make([]benchRow, n)
	for i := range rows {
		rows[i] = benchRow{
			Name:       fmt.Sprintf("Person_%d", i),
			Department: departments[i%len(departments)],
			City:       cities[i%len(cities)],
			Country:    countries[i%len(countries)],
			Salary:     50000 + float64(i%100)*1000,
			Age:        20 + i%50,
			Active:     i%3 != 0,
			Score:      float64(i%100) / 10.0,
		}
	}
	return rows
}

func benchCols() []data.Column[benchRow] {
	return []data.Column[benchRow]{
		{ColumnID: "Name", HeaderName: "Name", ValueGetter: func(r benchRow) any { return r.Name }, Sortable: true, Filterable: true, MinWidth: 4, Flex: 1},
		{ColumnID: "Department", HeaderName: "Department", ValueGetter: func(r benchRow) any { return r.Department }, Sortable: true, Filterable: true, MinWidth: 4, Flex: 1},
		{ColumnID: "City", HeaderName: "City", ValueGetter: func(r benchRow) any { return r.City }, Sortable: true, Filterable: true, MinWidth: 4, Flex: 1},
		{ColumnID: "Country", HeaderName: "Country", ValueGetter: func(r benchRow) any { return r.Country }, Sortable: true, Filterable: true, MinWidth: 4, Flex: 1},
		{ColumnID: "Salary", HeaderName: "Salary", ValueGetter: func(r benchRow) any { return r.Salary }, Sortable: true, Filterable: true, MinWidth: 4},
		{ColumnID: "Age", HeaderName: "Age", ValueGetter: func(r benchRow) any { return r.Age }, Sortable: true, Filterable: true, MinWidth: 4},
		{ColumnID: "Active", HeaderName: "Active", ValueGetter: func(r benchRow) any { return r.Active }, Sortable: true, Filterable: true, MinWidth: 4},
		{ColumnID: "Score", HeaderName: "Score", ValueGetter: func(r benchRow) any { return r.Score }, Sortable: true, Filterable: true, MinWidth: 4},
	}
}

func newBenchGrid(rows []benchRow, opts ...Option[benchRow]) Model[benchRow] {
	defaults := []Option[benchRow]{
		WithColumns[benchRow](benchCols()),
		WithRows[benchRow](rows),
		WithFocused[benchRow](true),
		WithWidth[benchRow](120),
		WithHeight[benchRow](40),
	}
	return New(append(defaults, opts...)...)
}

// ---------------------------------------------------------------------------
// Issue #6: Redundant double recomputeDisplayRows() in Update()
//
// Benchmarks the cost of Update() for pure navigation keystrokes (no data
// change). Each Update() currently calls recomputeDisplayRows() twice.
// ---------------------------------------------------------------------------

func BenchmarkUpdate_Navigation(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows)
			down := tea.KeyPressMsg{Code: tea.KeyDown}
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m, _ = m.Update(down)
			}
		})
	}
}

func BenchmarkUpdate_NavigationWithSort(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows,
				WithDefaultSort[benchRow]([]gridsort.SortCriterion{
					{ColumnID: "Salary", Direction: data.SortAsc},
				}),
			)
			down := tea.KeyPressMsg{Code: tea.KeyDown}
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m, _ = m.Update(down)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Issue #7: Cache filtered results across recompute calls
//
// Benchmarks recomputeDisplayRows when filters haven't changed (the common
// case during navigation). The dirty flag is forced to simulate re-entry.
// ---------------------------------------------------------------------------

func BenchmarkRecomputeDisplayRows_NoFilter(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkRecomputeDisplayRows_WithColumnFilter(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			tf := filter.NewTextFilter()
			tf.SetText("eng")
			m := newBenchGrid(rows,
				WithColumnFilter[benchRow]("Department", tf),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkRecomputeDisplayRows_WithQuickFilter(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows,
				WithQuickFilterText[benchRow]("engineering"),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkRecomputeDisplayRows_WithSort(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows,
				WithDefaultSort[benchRow]([]gridsort.SortCriterion{
					{ColumnID: "Salary", Direction: data.SortAsc},
				}),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkRecomputeDisplayRows_WithGrouping(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows,
				WithGrouping[benchRow]("Department"),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkRecomputeDisplayRows_Full(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			tf := filter.NewTextFilter()
			tf.SetText("eng")
			m := newBenchGrid(rows,
				WithColumnFilter[benchRow]("Department", tf),
				WithQuickFilterText[benchRow]("person"),
				WithDefaultSort[benchRow]([]gridsort.SortCriterion{
					{ColumnID: "Salary", Direction: data.SortAsc},
				}),
				WithGrouping[benchRow]("City"),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Issue #8: passesQuickFilter() allocates strings.Builder per row
//
// Benchmarks the quick filter hot path in isolation via recompute, varying
// the number of filter words and the row count.
// ---------------------------------------------------------------------------

func BenchmarkRecomputeDisplayRows_QuickFilter_MultiWord(b *testing.B) {
	for _, words := range []string{"person", "person engineering", "person engineering new york"} {
		b.Run(fmt.Sprintf("words=%q", words), func(b *testing.B) {
			rows := makeBenchRows(100_000)
			m := newBenchGrid(rows,
				WithQuickFilterText[benchRow](words),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Issue #9: FlattenGroups() copies RowNode by value
//
// Benchmarks grouping + flattening with varying group cardinality.
// ---------------------------------------------------------------------------

func BenchmarkRecomputeDisplayRows_GroupFlatten_SingleLevel(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows,
				WithGrouping[benchRow]("Department"),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkRecomputeDisplayRows_GroupFlatten_TwoLevels(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows,
				WithGrouping[benchRow]("Department", "City"),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Issue #10: passesColumnFilters() checks all columns unconditionally
//
// Benchmarks column filter checking with 0, 1, and N active filters.
// ---------------------------------------------------------------------------

func BenchmarkRecomputeDisplayRows_ColumnFilters_NoneActive(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			// Attach inactive filters to every column (filter text is empty).
			cols := benchCols()
			for i := range cols {
				cols[i].Filter = filter.NewTextFilter()
			}
			m := newBenchGrid(rows, WithColumns[benchRow](cols))
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkRecomputeDisplayRows_ColumnFilters_OneActive(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			cols := benchCols()
			for i := range cols {
				cols[i].Filter = filter.NewTextFilter()
			}
			tf := filter.NewTextFilter()
			tf.SetText("eng")
			m := newBenchGrid(rows,
				WithColumns[benchRow](cols),
				WithColumnFilter[benchRow]("Department", tf),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkRecomputeDisplayRows_ColumnFilters_AllActive(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			cols := benchCols()
			for i := range cols {
				cols[i].Filter = filter.NewTextFilter()
			}
			// Set text filters on all string columns.
			tfName := filter.NewTextFilter()
			tfName.SetText("person")
			tfDept := filter.NewTextFilter()
			tfDept.SetText("eng")
			tfCity := filter.NewTextFilter()
			tfCity.SetText("new")
			tfCountry := filter.NewTextFilter()
			tfCountry.SetText("us")
			m := newBenchGrid(rows,
				WithColumns[benchRow](cols),
				WithColumnFilter[benchRow]("Name", tfName),
				WithColumnFilter[benchRow]("Department", tfDept),
				WithColumnFilter[benchRow]("City", tfCity),
				WithColumnFilter[benchRow]("Country", tfCountry),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}
