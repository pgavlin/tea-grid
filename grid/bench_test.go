package grid

import (
	"fmt"
	"strconv"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/selection"
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

// BenchmarkRecomputeDisplayRows_SortChangeOnly measures the cost of recompute
// when only sort state changes (filter results should be cached).
func BenchmarkRecomputeDisplayRows_SortChangeOnly(b *testing.B) {
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
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				// Simulate sort-only change: dirty=true but filterDirty=false.
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

// ---------------------------------------------------------------------------
// View() rendering
//
// Benchmarks the rendering hot path. The viewport is fixed (120×40,
// ≈38 visible rows); total row count affects only recompute, not render.
// ---------------------------------------------------------------------------

func BenchmarkView_Plain(b *testing.B) {
	rows := makeBenchRows(10_000)
	m := newBenchGrid(rows)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = m.View()
	}
}

func BenchmarkView_WithSort(b *testing.B) {
	rows := makeBenchRows(10_000)
	m := newBenchGrid(rows,
		WithDefaultSort[benchRow]([]gridsort.SortCriterion{
			{ColumnID: "Salary", Direction: data.SortAsc},
		}),
	)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = m.View()
	}
}

func BenchmarkView_WithGrouping(b *testing.B) {
	rows := makeBenchRows(10_000)
	m := newBenchGrid(rows,
		WithGrouping[benchRow]("Department"),
	)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = m.View()
	}
}

func BenchmarkView_WithSelection(b *testing.B) {
	rows := makeBenchRows(10_000)
	m := newBenchGrid(rows,
		WithSelection[benchRow](selection.SelectMulti),
	)
	m.SelectAllRows()
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = m.View()
	}
}

// ---------------------------------------------------------------------------
// Filter cold path — re-evaluation without cache
//
// Complements the existing filter benchmarks (which hit the cache after
// iteration 1) by forcing filterDirty=true each iteration to measure the
// actual cost of running filters across all rows.
// ---------------------------------------------------------------------------

func BenchmarkRecomputeDisplayRows_ColumnFilter_Cold(b *testing.B) {
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
				m.filterDirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkRecomputeDisplayRows_QuickFilter_Cold(b *testing.B) {
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
				m.filterDirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func benchColsWithQuickFilterMatch() []data.Column[benchRow] {
	cols := benchCols()
	cols[0].QuickFilterMatch = func(r *benchRow, word string) bool { return containsFold(r.Name, word) }
	cols[1].QuickFilterMatch = func(r *benchRow, word string) bool { return containsFold(r.Department, word) }
	cols[2].QuickFilterMatch = func(r *benchRow, word string) bool { return containsFold(r.City, word) }
	cols[3].QuickFilterMatch = func(r *benchRow, word string) bool { return containsFold(r.Country, word) }
	cols[4].QuickFilterMatch = func(r *benchRow, word string) bool { return containsFold(fmt.Sprintf("%g", r.Salary), word) }
	cols[5].QuickFilterMatch = func(r *benchRow, word string) bool { return containsFold(strconv.Itoa(r.Age), word) }
	cols[6].QuickFilterMatch = func(r *benchRow, word string) bool { return containsFold(strconv.FormatBool(r.Active), word) }
	cols[7].QuickFilterMatch = func(r *benchRow, word string) bool { return containsFold(fmt.Sprintf("%g", r.Score), word) }
	return cols
}

func BenchmarkRecomputeDisplayRows_QuickFilter_Cold_WithMatch(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows,
				WithColumns[benchRow](benchColsWithQuickFilterMatch()),
				WithQuickFilterText[benchRow]("engineering"),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.filterDirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkRecomputeDisplayRows_Full_Cold(b *testing.B) {
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
				m.filterDirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SelectedRows / SelectedRowNodes — selection materialization
//
// Benchmarks the cost of iterating displayRows and building the result slice,
// with a full (100%) selection to exercise the worst-case allocation path.
// ---------------------------------------------------------------------------

func BenchmarkSelectedRows(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows,
				WithSelection[benchRow](selection.SelectMulti),
			)
			m.SelectAllRows()
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				_ = m.SelectedRows()
			}
		})
	}
}

func BenchmarkSelectedRowNodes(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows,
				WithSelection[benchRow](selection.SelectMulti),
			)
			m.SelectAllRows()
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				_ = m.SelectedRowNodes()
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Grid construction and data loading
// ---------------------------------------------------------------------------

func BenchmarkNew(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			cols := benchCols()
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				_ = New(
					WithColumns[benchRow](cols),
					WithRows[benchRow](rows),
					WithWidth[benchRow](120),
					WithHeight[benchRow](40),
				)
			}
		})
	}
}

func BenchmarkSetRows(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.SetRows(rows)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Sort — isolate comparison and findCol overhead
// ---------------------------------------------------------------------------

func BenchmarkRecomputeDisplayRows_Sort_Cold(b *testing.B) {
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
				m.filterDirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkRecomputeDisplayRows_Sort_MultiKey(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows,
				WithDefaultSort[benchRow]([]gridsort.SortCriterion{
					{ColumnID: "Department", Direction: data.SortAsc},
					{ColumnID: "Salary", Direction: data.SortDesc},
					{ColumnID: "Name", Direction: data.SortAsc},
				}),
			)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m.dirty = true
				m.filterDirty = true
				m.recomputeDisplayRows()
			}
		})
	}
}

func BenchmarkDefaultCompare(b *testing.B) {
	b.Run("string", func(b *testing.B) {
		a, bv := any("hello"), any("world")
		b.ReportAllocs()
		for range b.N {
			defaultCompare(a, bv)
		}
	})
	b.Run("int", func(b *testing.B) {
		a, bv := any(42), any(99)
		b.ReportAllocs()
		for range b.N {
			defaultCompare(a, bv)
		}
	})
	b.Run("float64", func(b *testing.B) {
		a, bv := any(3.14), any(2.72)
		b.ReportAllocs()
		for range b.N {
			defaultCompare(a, bv)
		}
	})
	b.Run("fallback", func(b *testing.B) {
		a, bv := any(42), any("hello")
		b.ReportAllocs()
		for range b.N {
			defaultCompare(a, bv)
		}
	})
}

// ---------------------------------------------------------------------------
// Update full cycles — keystroke to recompute
// ---------------------------------------------------------------------------

func BenchmarkUpdate_SortToggle(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows)
			m.focusedCell = CellPosition{Row: -1, Col: 0}
			sortKey := tea.KeyPressMsg{Code: 'S', Text: "S"}
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m, _ = m.Update(sortKey)
			}
		})
	}
}

func BenchmarkUpdate_QuickFilterKeystroke(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			m := newBenchGrid(rows,
				WithQuickFilter[benchRow](true),
				WithQuickFilterDebounce[benchRow](0), // disable debounce to avoid goroutine leak
			)
			// Activate quick filter mode
			m.quickFilterActive = true
			key := tea.KeyPressMsg{Code: 'e', Text: "e"}
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				m, _ = m.Update(key)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Render with varying column counts
// ---------------------------------------------------------------------------

func BenchmarkView_ColumnCount(b *testing.B) {
	for _, ncols := range []int{4, 8, 16, 32} {
		b.Run(fmt.Sprintf("cols=%d", ncols), func(b *testing.B) {
			rows := makeBenchRows(10_000)
			cols := make([]data.Column[benchRow], ncols)
			base := benchCols()
			for i := range cols {
				cols[i] = base[i%len(base)]
				cols[i].ColumnID = fmt.Sprintf("col_%d", i)
			}
			m := newBenchGrid(rows, WithColumns[benchRow](cols))
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				_ = m.View()
			}
		})
	}
}
