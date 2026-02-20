package main

import (
	"fmt"
	"os"
	"reflect"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/grid"
	"github.com/pgavlin/tea-grid/selection"
)

// Row is a slice of any, where each element is a struct value.
// Different elements may be different struct types.
type Row = []any

// Sample struct types that share the Name field but differ otherwise.

type Person struct {
	Name  string
	Age   int
	Email string
}

type Product struct {
	Name    string
	Price   float64
	InStock bool
}

type Event struct {
	Name      string
	Date      time.Time
	Attendees int
}

func nameGetter(r Row) string {
	for _, elem := range r {
		v := reflect.ValueOf(elem)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			continue
		}
		fv := v.FieldByName("Name")
		if fv.IsValid() {
			return fmt.Sprint(fv.Interface())
		}
	}
	return ""
}

func main() {
	rows := []Row{
		{Person{Name: "Alice", Age: 30, Email: "alice@example.com"}},
		{Person{Name: "Bob", Age: 25, Email: "bob@example.com"}},
		{Product{Name: "Widget", Price: 9.99, InStock: true}},
		{Product{Name: "Gadget", Price: 24.50, InStock: false}},
		{Event{Name: "GoConf", Date: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC), Attendees: 500}},
		{Event{Name: "HackDay", Date: time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), Attendees: 120}},
		{Person{Name: "Carol", Age: 42, Email: "carol@example.com"}},
		{Product{Name: "Doohickey", Price: 4.75, InStock: true}},
		{Event{Name: "MeetUp", Date: time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC), Attendees: 45}},
		{Person{Name: "Dave", Age: 37, Email: "dave@example.com"}},
	}

	cols := column.FromRows[Row](rows)

	// Pin first column left.
	if len(cols) > 0 {
		cols[0].Pinned = column.PinLeft
		cols[0].MinWidth = 16
		cols[0].Flex = 2
	}

	g := grid.New(
		grid.WithColumns(cols),
		grid.WithRows(rows),
		grid.WithRowID(nameGetter),
		grid.WithSelection[Row](selection.SelectMulti),
		grid.WithQuickFilter[Row](true),
		grid.WithFocused[Row](true),
		grid.WithMultiSort[Row](true),
	)

	p := tea.NewProgram(g, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
