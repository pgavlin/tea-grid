package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/filter"
	"github.com/pgavlin/tea-grid/grid"
)

const (
	defaultRows = 20
	defaultCols = 10
)

type clearStatusMsg struct{ seq int }

var statusStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252")).
	Background(lipgloss.Color("235"))

type model struct {
	grid         grid.Model[*SpreadsheetRow]
	rows         []*SpreadsheetRow
	numCols      int
	colFmts      map[string]*CellFormat
	rowFmts      map[int]*CellFormat
	deps         DepGraph
	sidebar      SidebarModel
	sidebarOn    bool
	gridFocus    bool
	helpOn       bool
	fileForm     *huh.Form
	fileFormMode string // "save" or "load"
	status       string
	statusSeq    int
	width        int
	height       int
	filename     string
}

func newModel(filename string) model {
	if filename == "" {
		filename = "spreadsheet.txs"
	}

	m := model{
		numCols:   defaultCols,
		colFmts:   make(map[string]*CellFormat),
		rowFmts:   make(map[int]*CellFormat),
		deps:      NewDepGraph(),
		sidebar:   NewSidebar(),
		gridFocus: true,
		filename:  filename,
	}

	if rows, colFmts, rowFmts, numCols, err := loadNative(filename); err == nil {
		m.rows = rows
		m.colFmts = colFmts
		m.rowFmts = rowFmts
		m.numCols = numCols
		m.status = fmt.Sprintf("Loaded %s", filename)
	} else {
		m.status = fmt.Sprintf("New file: %s", filename)
		m.rows = makeEmptyRows(defaultRows, m.numCols)
	}

	// Build columns and grid
	cols := m.buildColumns()
	m.grid = grid.New(
		grid.WithColumns(cols),
		grid.WithRows(m.rows),
		grid.WithRowID(func(r *SpreadsheetRow) string {
			return fmt.Sprintf("row-%d", r.RowIndex)
		}),
		grid.WithEditable[*SpreadsheetRow](true),
		grid.WithQuickFilter[*SpreadsheetRow](true),
		grid.WithFocused[*SpreadsheetRow](true),
		grid.WithMultiSort[*SpreadsheetRow](true),
	)

	// Initial calculation
	recalcAll(m.rows, &m.deps)
	m.grid.SetRows(m.rows)

	return m
}

func makeEmptyRows(numRows, numCols int) []*SpreadsheetRow {
	rows := make([]*SpreadsheetRow, numRows)
	for i := range rows {
		cells := make(map[string]*Cell, numCols)
		for j := 0; j < numCols; j++ {
			cells[indexToColLetter(j)] = &Cell{}
		}
		rows[i] = &SpreadsheetRow{
			RowIndex: i,
			Cells:    cells,
		}
	}
	return rows
}

func (m *model) buildColumns() []data.Column[*SpreadsheetRow] {
	cols := make([]data.Column[*SpreadsheetRow], 0, m.numCols+1)

	// Row number column
	cols = append(cols, data.Column[*SpreadsheetRow]{
		ColumnID:   "_rownum",
		HeaderName: "",
		Pinned:     data.PinLeft,
		Width:      5,
		Sortable:   false,
		Filterable: false,
		Editable:   false,
		ValueGetter: func(r *SpreadsheetRow) any {
			return r.RowIndex + 1
		},
		ValueFormatter: func(v any, _ *SpreadsheetRow) string {
			return fmt.Sprintf("%v", v)
		},
		CellStyle: func(_ any, _ *SpreadsheetRow) lipgloss.Style {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Align(lipgloss.Right)
		},
	})

	// Data columns A-J (or however many)
	for i := 0; i < m.numCols; i++ {
		colLetter := indexToColLetter(i)
		colID := colLetter
		colFmts := m.colFmts // capture reference
		rowFmts := m.rowFmts // capture reference

		col := data.Column[*SpreadsheetRow]{
			ColumnID:   colID,
			HeaderName: colLetter,
			Flex:       1,
			MinWidth:   8,
			Sortable:   true,
			Filterable: true,
			Editable:   true,
			Filter:     filter.NewTextFilter(),
			CellEditor: NewFormulaEditor(colID),
			ValueGetter: func(r *SpreadsheetRow) any {
				cell, ok := r.Cells[colID]
				if !ok {
					return nil
				}
				return cell.Value
			},
			ValueFormatter: func(v any, r *SpreadsheetRow) string {
				cell := r.Cells[colID]
				resolved := resolveFormat(cell, rowFmts[r.RowIndex], colFmts[colID])
				return formatValue(v, resolved)
			},
			ValueSetter: func(r **SpreadsheetRow, value any) {
				cell := (*r).getCell(colID)
				cell.Raw = fmt.Sprintf("%v", value)
			},
			CellStyle: func(v any, r *SpreadsheetRow) lipgloss.Style {
				cell := r.Cells[colID]
				resolved := resolveFormat(cell, rowFmts[r.RowIndex], colFmts[colID])
				return cellStyle(v, resolved)
			},
		}

		cols = append(cols, col)
	}

	return cols
}

func (m model) Init() tea.Cmd {
	clearStatus := tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
	return tea.Batch(clearStatus, m.grid.Init())
}

func (m *model) setStatus(s string) tea.Cmd {
	m.status = s
	m.statusSeq++
	seq := m.statusSeq
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{seq: seq}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// File form active — route all messages to it
	if m.fileForm != nil {
		_, cmd := m.fileForm.Update(msg)
		switch m.fileForm.State {
		case huh.StateCompleted:
			statusCmd := m.completeFileForm()
			m.fileForm = nil
			return m, statusCmd
		case huh.StateAborted:
			statusCmd := m.setStatus("Cancelled")
			m.fileForm = nil
			return m, statusCmd
		default:
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case clearStatusMsg:
		if msg.seq == m.statusSeq {
			m.status = ""
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeComponents()
		var cmd tea.Cmd
		m.grid, cmd = m.grid.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Help screen absorbs all keys
		if m.helpOn {
			m.helpOn = false
			return m, nil
		}

		// App-level keys checked first
		if cmd := m.handleAppKey(msg); cmd != nil {
			return m, cmd
		}

		// Route to sidebar or grid
		if m.sidebarOn && !m.gridFocus {
			m.sidebar.Update(msg)
			m.applySidebarFormat()
			return m, nil
		}

		var cmd tea.Cmd
		m.grid, cmd = m.grid.Update(msg)
		return m, cmd

	case grid.CellValueChangedMsg[*SpreadsheetRow]:
		// Cell was edited - recalculate
		pos := msg.Position
		if pos.Col > 0 && pos.Col <= m.numCols {
			colLetter := indexToColLetter(pos.Col - 1) // -1 for row number column
			ref := cellRefString(colLetter, pos.Row+1) // 1-based row
			recalcFrom(ref, m.rows, &m.deps)
			m.grid.SetRows(m.rows)
		}
		return m, nil

	case grid.FocusChangedMsg:
		if m.sidebarOn {
			m.updateSidebarTarget()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.grid, cmd = m.grid.Update(msg)
	return m, cmd
}

func (m *model) handleAppKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "ctrl+c":
		return tea.Quit

	case "q":
		if !m.sidebarOn {
			return tea.Quit
		}
		return nil

	case "ctrl+s":
		return m.showFileForm("save")

	case "ctrl+o":
		return m.showFileForm("load")

	case "ctrl+r":
		return m.insertRowBelow()

	case "ctrl+t":
		return m.insertColRight()

	case "ctrl+d":
		return m.deleteCurrentRow()

	case "ctrl+w":
		return m.deleteCurrentCol()

	case "?", "f1":
		m.helpOn = true
		return nil

	case "f":
		if !m.sidebarOn {
			m.sidebarOn = true
			m.gridFocus = false
			m.updateSidebarTarget()
			m.resizeComponents()
			return nil
		}

	case "tab":
		if m.sidebarOn {
			m.gridFocus = !m.gridFocus
			return nil
		}

	case "esc":
		if m.sidebarOn && !m.gridFocus {
			m.sidebarOn = false
			m.gridFocus = true
			m.resizeComponents()
			return nil
		}
	}

	return nil
}

func (m *model) insertRowBelow() tea.Cmd {
	pos := m.grid.FocusedCell()
	insertIdx := pos.Row + 1
	if insertIdx < 0 {
		insertIdx = 0
	}
	if insertIdx > len(m.rows) {
		insertIdx = len(m.rows)
	}

	// Create new row with empty cells
	cells := make(map[string]*Cell, m.numCols)
	for j := 0; j < m.numCols; j++ {
		cells[indexToColLetter(j)] = &Cell{}
	}
	newRow := &SpreadsheetRow{Cells: cells}

	// Insert into slice
	m.rows = append(m.rows, nil)
	copy(m.rows[insertIdx+1:], m.rows[insertIdx:])
	m.rows[insertIdx] = newRow

	// Shift row formats: indices >= insertIdx move up by 1
	newRowFmts := make(map[int]*CellFormat)
	for idx, f := range m.rowFmts {
		if idx >= insertIdx {
			newRowFmts[idx+1] = f
		} else {
			newRowFmts[idx] = f
		}
	}
	m.rowFmts = newRowFmts

	// Reindex all rows
	for i, r := range m.rows {
		r.RowIndex = i
	}

	// Full recalc to pick up shifted references
	recalcAll(m.rows, &m.deps)
	m.grid.SetRows(m.rows)
	return m.setStatus(fmt.Sprintf("Inserted row %d", insertIdx+1))
}

func (m *model) insertColRight() tea.Cmd {
	pos := m.grid.FocusedCell()
	// pos.Col is grid column index (0 = row-number column), so data column = pos.Col - 1
	insertDataIdx := pos.Col // insert after current data column (right of it)
	if insertDataIdx < 0 {
		insertDataIdx = 0
	}
	if insertDataIdx > m.numCols {
		insertDataIdx = m.numCols
	}

	m.numCols++
	newColLetter := indexToColLetter(m.numCols - 1)

	// Add empty cell for the new column in every row
	for _, row := range m.rows {
		row.Cells[newColLetter] = &Cell{}
	}

	// Rebuild columns and refresh grid
	recalcAll(m.rows, &m.deps)
	cols := m.buildColumns()
	m.grid.SetColumns(cols)
	m.grid.SetRows(m.rows)
	return m.setStatus(fmt.Sprintf("Inserted column %s", newColLetter))
}

func (m *model) deleteCurrentRow() tea.Cmd {
	if len(m.rows) <= 1 {
		return m.setStatus("Cannot delete last row")
	}

	pos := m.grid.FocusedCell()
	idx := pos.Row
	if idx < 0 || idx >= len(m.rows) {
		return nil
	}

	deletedNum := idx + 1
	m.rows = append(m.rows[:idx], m.rows[idx+1:]...)

	// Shift row formats: delete at idx, shift indices > idx down by 1
	newRowFmts := make(map[int]*CellFormat)
	for i, f := range m.rowFmts {
		if i == idx {
			continue
		}
		if i > idx {
			newRowFmts[i-1] = f
		} else {
			newRowFmts[i] = f
		}
	}
	m.rowFmts = newRowFmts

	// Reindex
	for i, r := range m.rows {
		r.RowIndex = i
	}

	recalcAll(m.rows, &m.deps)
	m.grid.SetRows(m.rows)
	return m.setStatus(fmt.Sprintf("Deleted row %d", deletedNum))
}

func (m *model) deleteCurrentCol() tea.Cmd {
	if m.numCols <= 1 {
		return m.setStatus("Cannot delete last column")
	}

	pos := m.grid.FocusedCell()
	dataIdx := pos.Col - 1 // subtract row-number column
	if dataIdx < 0 || dataIdx >= m.numCols {
		return nil
	}

	deletedLetter := indexToColLetter(dataIdx)

	// Remove the last column's data from every row and shift cells left
	// Strategy: rebuild each row's cell map with the new column set
	newNumCols := m.numCols - 1
	for _, row := range m.rows {
		newCells := make(map[string]*Cell, newNumCols)
		dst := 0
		for src := 0; src < m.numCols; src++ {
			if src == dataIdx {
				continue
			}
			srcLetter := indexToColLetter(src)
			dstLetter := indexToColLetter(dst)
			if c, ok := row.Cells[srcLetter]; ok {
				newCells[dstLetter] = c
			} else {
				newCells[dstLetter] = &Cell{}
			}
			dst++
		}
		row.Cells = newCells
	}

	// Remove column format
	delete(m.colFmts, deletedLetter)
	// Shift column formats left
	newColFmts := make(map[string]*CellFormat)
	dst := 0
	for src := 0; src < m.numCols; src++ {
		if src == dataIdx {
			continue
		}
		srcLetter := indexToColLetter(src)
		dstLetter := indexToColLetter(dst)
		if f, ok := m.colFmts[srcLetter]; ok {
			newColFmts[dstLetter] = f
		}
		dst++
	}
	m.colFmts = newColFmts
	m.numCols = newNumCols

	recalcAll(m.rows, &m.deps)
	cols := m.buildColumns()
	m.grid.SetColumns(cols)
	m.grid.SetRows(m.rows)
	return m.setStatus(fmt.Sprintf("Deleted column %s", deletedLetter))
}

func (m *model) showFileForm(mode string) tea.Cmd {
	m.fileFormMode = mode
	fileFormPath := m.filename
	fileFormFmt := strings.TrimPrefix(filepath.Ext(m.filename), ".")
	if fileFormFmt != "csv" {
		fileFormFmt = "txs"
	}

	var title string
	if mode == "save" {
		title = "Save As"
	} else {
		title = "Open File"
	}

	var pathField huh.Field
	if mode == "load" {
		cwd, _ := os.Getwd()
		pathField = huh.NewFilePicker().
			Title("File").
			CurrentDirectory(cwd).
			FileAllowed(true).
			DirAllowed(false).
			ShowSize(true).
			AllowedTypes([]string{".txs", ".csv"}).
			Height(10).
			Value(&fileFormPath).
			Key("path")
	} else {
		pathField = huh.NewInput().
			Title("File").
			Placeholder("spreadsheet.txs").
			Value(&fileFormPath).
			Key("path")
	}

	formatField := huh.NewSelect[string]().
		Title("Format").
		Options(
			huh.NewOption("Sheet", "txs"),
			huh.NewOption("CSV", "csv"),
		).
		Value(&fileFormFmt).
		Key("format")

	m.fileForm = huh.NewForm(
		huh.NewGroup(pathField, formatField).Title(title),
	).WithWidth(50).WithShowHelp(true).WithShowErrors(true)

	return m.fileForm.Init()
}

func (m *model) completeFileForm() tea.Cmd {
	path := m.fileForm.GetString("path")
	format := m.fileForm.GetString("format")

	if path == "" {
		return m.setStatus("No file specified")
	}

	if m.fileFormMode == "save" {
		var err error
		if format == "csv" {
			err = exportCSV(path, m.rows, m.numCols)
		} else {
			err = saveNative(path, m.rows, m.colFmts, m.rowFmts, m.numCols)
		}
		if err != nil {
			return m.setStatus(fmt.Sprintf("Save error: %v", err))
		}
		m.filename = path
		return m.setStatus(fmt.Sprintf("Saved %s", path))
	}

	// Load mode
	if format == "csv" {
		rows, numCols, err := importCSV(path)
		if err != nil {
			return m.setStatus(fmt.Sprintf("Load error: %v", err))
		}
		m.rows = rows
		m.numCols = numCols
		m.colFmts = make(map[string]*CellFormat)
		m.rowFmts = make(map[int]*CellFormat)
	} else {
		rows, colFmts, rowFmts, numCols, err := loadNative(path)
		if err != nil {
			return m.setStatus(fmt.Sprintf("Load error: %v", err))
		}
		m.rows = rows
		m.colFmts = colFmts
		m.rowFmts = rowFmts
		m.numCols = numCols
	}
	m.filename = path
	m.deps = NewDepGraph()
	recalcAll(m.rows, &m.deps)
	cols := m.buildColumns()
	m.grid.SetColumns(cols)
	m.grid.SetRows(m.rows)
	return m.setStatus(fmt.Sprintf("Loaded %s", path))
}

func (m *model) resizeComponents() {
	gridWidth := m.width
	if m.sidebarOn {
		sidebarWidth := m.width / 5
		if sidebarWidth < 24 {
			sidebarWidth = 24
		}
		gridWidth = m.width - sidebarWidth
		m.sidebar.width = sidebarWidth
		m.sidebar.height = m.height - 1 // status bar
	}

	m.grid.SetWidth(gridWidth)
	m.grid.SetHeight(m.height - 1) // reserve 1 line for status bar
}

func (m *model) updateSidebarTarget() {
	pos := m.grid.FocusedCell()
	rowIdx := pos.Row

	if pos.Col == 0 {
		// Row number column — only row formatting is available.
		var rowFmt *CellFormat
		if rowIdx >= 0 && rowIdx < len(m.rows) {
			rowFmt = m.rowFmts[rowIdx]
		}
		m.sidebar.SetTarget("", rowIdx, nil, true, rowFmt, nil)
	} else if pos.Col > 0 && pos.Col <= m.numCols {
		// Data column — cell and column formatting available, not row.
		colLetter := indexToColLetter(pos.Col - 1)
		var cell *Cell
		if rowIdx >= 0 && rowIdx < len(m.rows) {
			cell = m.rows[rowIdx].Cells[colLetter]
		}
		m.sidebar.SetTarget(colLetter, rowIdx, cell, false, m.rowFmts[rowIdx], m.colFmts[colLetter])
	}
}

func (m *model) applySidebarFormat() {
	f := *m.sidebar.Format()
	colID := m.sidebar.colID

	switch m.sidebar.Target() {
	case TargetCell:
		pos := m.grid.FocusedCell()
		if pos.Col > 0 && pos.Col <= m.numCols {
			rowIdx := pos.Row
			if rowIdx >= 0 && rowIdx < len(m.rows) {
				cell := m.rows[rowIdx].getCell(colID)
				// Clear cell format if it matches the next in precedence (row > column > default).
				var parent CellFormat
				if rf := m.rowFmts[rowIdx]; rf != nil {
					parent = *rf
				} else if cf := m.colFmts[colID]; cf != nil {
					parent = *cf
				}
				if f == parent {
					cell.Format = nil
				} else {
					cell.Format = &f
				}
			}
		}
	case TargetRow:
		pos := m.grid.FocusedCell()
		rowIdx := pos.Row
		if rowIdx >= 0 && rowIdx < len(m.rows) {
			// Clear row format if it matches the next in precedence (column > default).
			var parent CellFormat
			if cf := m.colFmts[colID]; cf != nil {
				parent = *cf
			}
			if f == parent {
				delete(m.rowFmts, rowIdx)
			} else {
				m.rowFmts[rowIdx] = &f
			}
		}
	case TargetColumn:
		// Clear column format if it matches default (zero value).
		if f == (CellFormat{}) {
			delete(m.colFmts, colID)
		} else {
			m.colFmts[colID] = &f
		}
	}

	// Refresh display
	m.grid.SetRows(m.rows)
}

func (m model) View() string {
	if m.helpOn {
		return m.renderHelp()
	}

	if m.fileForm != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.fileForm.View())
	}

	gridView := m.grid.View()

	var view string
	if m.sidebarOn {
		sidebarView := m.sidebar.View()
		view = lipgloss.JoinHorizontal(lipgloss.Top, gridView, sidebarView)
	} else {
		view = gridView
	}

	// Status bar
	bar := m.renderStatusBar()

	return view + "\n" + bar
}

func (m model) renderStatusBar() string {
	left := m.statusLeft()
	right := "?:help  f:format  ctrl+s:save  ctrl+o:open  q:quit"

	leftRendered := statusStyle.PaddingLeft(1).Render(left)
	rightRendered := statusStyle.PaddingRight(1).Render(right)

	gap := m.width - lipgloss.Width(leftRendered) - lipgloss.Width(rightRendered)
	if gap < 0 {
		gap = 0
	}
	fill := statusStyle.Render(strings.Repeat(" ", gap))

	return leftRendered + fill + rightRendered
}

func (m model) statusLeft() string {
	parts := []string{}

	if m.status != "" {
		parts = append(parts, m.status)
	}

	pos := m.grid.FocusedCell()
	if pos.Col > 0 && pos.Col <= m.numCols {
		colLetter := indexToColLetter(pos.Col - 1)
		ref := fmt.Sprintf("%s%d", colLetter, pos.Row+1)
		parts = append(parts, ref)

		if pos.Row >= 0 && pos.Row < len(m.rows) {
			cell := m.rows[pos.Row].Cells[colLetter]
			if cell != nil && cell.Raw != "" {
				parts = append(parts, cell.Raw)
			}
		}
	}

	return strings.Join(parts, "  |  ")
}

func (m model) renderHelp() string {
	title := lipgloss.NewStyle().Bold(true).Underline(true).
		MarginBottom(1).Render("Spreadsheet Help")

	sections := []struct {
		heading string
		keys    [][2]string
	}{
		{"Navigation", [][2]string{
			{"h/j/k/l, arrows", "Move between cells"},
			{"g/G", "Jump to first/last row"},
			{"0/$", "Jump to first/last column"},
		}},
		{"Editing", [][2]string{
			{"e, enter", "Edit cell (type formula or value)"},
			{"enter", "Confirm edit"},
			{"esc", "Cancel edit"},
		}},
		{"Rows & Columns", [][2]string{
			{"ctrl+r", "Insert row below"},
			{"ctrl+d", "Delete current row"},
			{"ctrl+t", "Insert column right"},
			{"ctrl+w", "Delete current column"},
		}},
		{"Formatting", [][2]string{
			{"f", "Open format sidebar"},
			{"tab", "Switch focus (grid/sidebar)"},
			{"esc", "Close sidebar"},
		}},
		{"Sorting & Filtering", [][2]string{
			{"s", "Sort by column"},
			{"/", "Quick filter"},
		}},
		{"File", [][2]string{
			{"ctrl+s", "Save (JSON or CSV)"},
			{"ctrl+o", "Open (JSON or CSV)"},
		}},
		{"Formulas", [][2]string{
			{"=A1+B2", "Arithmetic (+, -, *, /, ^)"},
			{"=SUM(A1:A5)", "Range functions"},
			{"=AVG, MIN, MAX", "Aggregation functions"},
			{"=COUNT, ABS, ROUND", "Utility functions"},
		}},
		{"Other", [][2]string{
			{"?/f1", "This help screen"},
			{"q, ctrl+c", "Quit"},
		}},
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Width(20)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	headingStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).MarginTop(1)

	var body strings.Builder
	for _, sec := range sections {
		body.WriteString(headingStyle.Render(sec.heading) + "\n")
		for _, kv := range sec.keys {
			body.WriteString("  " + keyStyle.Render(kv[0]) + descStyle.Render(kv[1]) + "\n")
		}
	}

	content := title + "\n" + body.String() + "\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render("Press any key to close")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 3)

	rendered := box.Render(content)

	// Center on screen
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, rendered)
}

func main() {
	filename := ""
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	m := newModel(filename)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
