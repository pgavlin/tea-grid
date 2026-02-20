package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pgavlin/tea-grid/column"
	"github.com/pgavlin/tea-grid/grid"
)

type Stock struct {
	Ticker   string
	Company  string
	Sector   string
	Price    float64
	Open     float64
	High     float64
	Low      float64
	Volume   int
	MarketCap string
	PE       float64
	Dividend float64
	YTD      float64
}

type model struct {
	grid grid.Model[Stock]
}

func (m model) Init() tea.Cmd { return m.grid.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		// Cap width at 80 so the 12 columns always exceed the viewport.
		w := msg.Width
		if w > 80 {
			w = 80
		}
		m.grid.SetWidth(w)
		m.grid.SetHeight(msg.Height)
	}
	var cmd tea.Cmd
	m.grid, cmd = m.grid.Update(msg)
	return m, cmd
}

func (m model) View() string { return m.grid.View() }

func main() {
	cols := []column.ColDef[Stock]{
		{ColID: "ticker", HeaderName: "Ticker", ValueGetter: func(s Stock) any { return s.Ticker }, Width: 8, Pinned: column.PinLeft, Sortable: true},
		{ColID: "company", HeaderName: "Company", ValueGetter: func(s Stock) any { return s.Company }, Width: 22, Sortable: true},
		{ColID: "sector", HeaderName: "Sector", ValueGetter: func(s Stock) any { return s.Sector }, Width: 16, Sortable: true},
		{ColID: "price", HeaderName: "Price", ValueGetter: func(s Stock) any { return s.Price }, Width: 10, Sortable: true, ValueFormatter: func(v any, _ Stock) string { return fmt.Sprintf("$%.2f", v.(float64)) }},
		{ColID: "open", HeaderName: "Open", ValueGetter: func(s Stock) any { return s.Open }, Width: 10, Sortable: true, ValueFormatter: func(v any, _ Stock) string { return fmt.Sprintf("$%.2f", v.(float64)) }},
		{ColID: "high", HeaderName: "High", ValueGetter: func(s Stock) any { return s.High }, Width: 10, Sortable: true, ValueFormatter: func(v any, _ Stock) string { return fmt.Sprintf("$%.2f", v.(float64)) }},
		{ColID: "low", HeaderName: "Low", ValueGetter: func(s Stock) any { return s.Low }, Width: 10, Sortable: true, ValueFormatter: func(v any, _ Stock) string { return fmt.Sprintf("$%.2f", v.(float64)) }},
		{ColID: "volume", HeaderName: "Volume", ValueGetter: func(s Stock) any { return s.Volume }, Width: 12, Sortable: true, ValueFormatter: func(v any, _ Stock) string { return fmt.Sprintf("%d", v.(int)) }},
		{ColID: "mktcap", HeaderName: "Mkt Cap", ValueGetter: func(s Stock) any { return s.MarketCap }, Width: 10, Sortable: true},
		{ColID: "pe", HeaderName: "P/E", ValueGetter: func(s Stock) any { return s.PE }, Width: 8, Sortable: true, ValueFormatter: func(v any, _ Stock) string { return fmt.Sprintf("%.1f", v.(float64)) }},
		{ColID: "dividend", HeaderName: "Div %", ValueGetter: func(s Stock) any { return s.Dividend }, Width: 8, Sortable: true, ValueFormatter: func(v any, _ Stock) string { return fmt.Sprintf("%.2f%%", v.(float64)) }},
		{ColID: "ytd", HeaderName: "YTD %", ValueGetter: func(s Stock) any { return s.YTD }, Width: 8, Sortable: true, ValueFormatter: func(v any, _ Stock) string { return fmt.Sprintf("%+.1f%%", v.(float64)) }},
	}

	rows := []Stock{
		{"AAPL", "Apple Inc.", "Technology", 189.84, 188.12, 191.07, 187.45, 54320000, "2.95T", 31.2, 0.51, 48.2},
		{"MSFT", "Microsoft Corp.", "Technology", 378.91, 375.00, 380.50, 374.20, 22100000, "2.81T", 36.8, 0.72, 42.1},
		{"GOOGL", "Alphabet Inc.", "Technology", 141.80, 140.50, 143.20, 139.90, 25600000, "1.77T", 25.4, 0.00, 55.3},
		{"AMZN", "Amazon.com Inc.", "Consumer", 178.25, 176.80, 179.50, 175.30, 47800000, "1.85T", 60.2, 0.00, 52.8},
		{"NVDA", "NVIDIA Corp.", "Technology", 495.22, 490.00, 498.75, 488.10, 41200000, "1.22T", 65.3, 0.04, 239.0},
		{"META", "Meta Platforms", "Technology", 357.42, 354.00, 359.80, 352.50, 18900000, "917B", 28.9, 0.00, 178.5},
		{"TSLA", "Tesla Inc.", "Consumer", 248.50, 245.00, 252.30, 243.80, 98500000, "789B", 78.1, 0.00, 128.7},
		{"BRK.B", "Berkshire Hathaway", "Financial", 362.11, 360.50, 364.00, 359.20, 3200000, "792B", 9.1, 0.00, 15.2},
		{"JNJ", "Johnson & Johnson", "Healthcare", 156.74, 155.80, 157.90, 155.10, 7100000, "378B", 11.2, 3.02, -8.3},
		{"V", "Visa Inc.", "Financial", 261.38, 259.50, 263.00, 258.90, 6500000, "536B", 31.5, 0.76, 25.4},
		{"JPM", "JPMorgan Chase", "Financial", 171.20, 169.80, 172.50, 168.90, 9800000, "495B", 11.8, 2.45, 27.8},
		{"PG", "Procter & Gamble", "Consumer", 152.89, 151.50, 154.20, 150.80, 6200000, "360B", 25.6, 2.42, 0.5},
		{"UNH", "UnitedHealth Group", "Healthcare", 527.30, 523.00, 530.50, 521.80, 3400000, "487B", 23.1, 1.32, -0.7},
		{"XOM", "Exxon Mobil Corp.", "Energy", 104.58, 103.20, 105.90, 102.80, 15600000, "418B", 10.8, 3.48, 3.8},
		{"PFE", "Pfizer Inc.", "Healthcare", 28.91, 28.50, 29.30, 28.20, 32100000, "163B", 42.5, 5.72, -43.5},
	}

	g := grid.New(
		grid.WithColumns(cols),
		grid.WithRows(rows),
		grid.WithRowID(func(s Stock) string { return s.Ticker }),
		grid.WithFocused[Stock](true),
		grid.WithMultiSort[Stock](true),
	)

	p := tea.NewProgram(model{grid: g}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
