package grouping

import (
	"fmt"
	"testing"

	"github.com/pgavlin/tea-grid/data"
)

type benchRow struct {
	Name       string
	Department string
	City       string
	Salary     float64
}

var benchDepartments = []string{"Engineering", "Sales", "Marketing", "Support", "Finance", "Legal", "HR", "Product", "Design", "Operations"}
var benchCities = []string{"New York", "San Francisco", "London", "Berlin", "Tokyo", "Sydney", "Toronto", "Paris", "Singapore", "Mumbai"}

func makeBenchRows(n int) []data.RowNode[benchRow] {
	nodes := make([]data.RowNode[benchRow], n)
	for i := range nodes {
		nodes[i] = data.RowNode[benchRow]{
			Data: benchRow{
				Name:       fmt.Sprintf("Person_%d", i),
				Department: benchDepartments[i%len(benchDepartments)],
				City:       benchCities[i%len(benchCities)],
				Salary:     50000 + float64(i%100)*1000,
			},
			ID: fmt.Sprintf("row_%d", i),
		}
	}
	return nodes
}

func benchCols() []data.Column[benchRow] {
	return []data.Column[benchRow]{
		{ColumnID: "Name", HeaderName: "Name", ValueGetter: func(r benchRow) any { return r.Name }},
		{ColumnID: "Department", HeaderName: "Department", ValueGetter: func(r benchRow) any { return r.Department }},
		{ColumnID: "City", HeaderName: "City", ValueGetter: func(r benchRow) any { return r.City }},
		{ColumnID: "Salary", HeaderName: "Salary", ValueGetter: func(r benchRow) any { return r.Salary }},
	}
}

// ---------------------------------------------------------------------------
// BuildGroups benchmarks
// ---------------------------------------------------------------------------

func BenchmarkBuildGroups_SingleLevel(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			cols := benchCols()
			expanded := map[string]bool{}
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				BuildGroups(rows, cols, []string{"Department"}, expanded, -1)
			}
		})
	}
}

func BenchmarkBuildGroups_TwoLevels(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			cols := benchCols()
			expanded := map[string]bool{}
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				BuildGroups(rows, cols, []string{"Department", "City"}, expanded, -1)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FlattenGroups benchmarks (Issue #9: copies RowNode by value)
// ---------------------------------------------------------------------------

func BenchmarkFlattenGroups_SingleLevel(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			cols := benchCols()
			expanded := map[string]bool{}
			groups := BuildGroups(rows, cols, []string{"Department"}, expanded, -1)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				FlattenGroups(groups)
			}
		})
	}
}

func BenchmarkFlattenGroups_TwoLevels(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := makeBenchRows(n)
			cols := benchCols()
			expanded := map[string]bool{}
			groups := BuildGroups(rows, cols, []string{"Department", "City"}, expanded, -1)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				FlattenGroups(groups)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Aggregate benchmarks
// ---------------------------------------------------------------------------

func BenchmarkAggregate_Sum(b *testing.B) {
	values := make([]any, 10_000)
	for i := range values {
		values[i] = float64(i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		Aggregate(values, "sum")
	}
}

func BenchmarkAggregate_Avg(b *testing.B) {
	values := make([]any, 10_000)
	for i := range values {
		values[i] = float64(i)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		Aggregate(values, "avg")
	}
}
