package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pgavlin/tea-grid/data"
	"github.com/pgavlin/tea-grid/internal/lineedit"
)

// DataType indicates how a cell value should be formatted.
type DataType int

const (
	DataTypeText DataType = iota
	DataTypeNumber
	DataTypeDate
)

// Alignment represents horizontal text alignment.
type Alignment int

const (
	AlignAuto Alignment = iota // left for text, right for numbers
	AlignLeft
	AlignCenter
	AlignRight
)

// CellFormat controls per-cell or per-column formatting.
type CellFormat struct {
	DataType    DataType
	NumDecimals int
	DateFormat  string
	Prefix      string
	Suffix      string
	Align       Alignment
	FgColor     string
	BgColor     string
	Bold        bool
	Italic      bool
	Underline   bool
}

// Cell represents a single spreadsheet cell.
type Cell struct {
	Raw    string      // user input: literal or "=formula"
	Value  any         // computed result (float64, string, or error string)
	Format *CellFormat // nil = inherit column format
}

// SpreadsheetRow represents a single row in the spreadsheet.
type SpreadsheetRow struct {
	RowIndex int              // 0-based
	Cells    map[string]*Cell // keyed by column letter "A", "B", ...
}

// getCell returns the cell for a column, creating it if necessary.
func (r *SpreadsheetRow) getCell(col string) *Cell {
	if r.Cells == nil {
		r.Cells = make(map[string]*Cell)
	}
	c, ok := r.Cells[col]
	if !ok {
		c = &Cell{}
		r.Cells[col] = c
	}
	return c
}

// FormulaEditor implements data.CellEditor[*SpreadsheetRow].
// It edits the raw formula/text of a cell rather than the computed value.
type FormulaEditor struct {
	editor lineedit.Model
	width  int
	colID  string
}

func NewFormulaEditor(colID string) *FormulaEditor {
	return &FormulaEditor{colID: colID}
}

func (e *FormulaEditor) Init(ctx data.CellContext[*SpreadsheetRow]) tea.Cmd {
	// Show the raw formula text, not the computed value
	cell := ctx.Data.getCell(e.colID)
	e.editor.SetText(cell.Raw)
	e.editor.CursorToEnd()
	e.width = ctx.Width
	return nil
}

func (e *FormulaEditor) Update(msg tea.Msg) (data.CellEditor[*SpreadsheetRow], tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		e.editor.HandleKeyMsg(msg)
	}
	return e, nil
}

func (e *FormulaEditor) View() string {
	return e.editor.RenderLine(e.width, "")
}

func (e *FormulaEditor) Value() any {
	return e.editor.Text()
}

func (e *FormulaEditor) Validate() string {
	return ""
}

// formatValue formats a computed value according to the given format.
func formatValue(v any, fmt *CellFormat) string {
	if v == nil {
		return ""
	}

	if fmt == nil {
		return defaultFormat(v)
	}

	switch fmt.DataType {
	case DataTypeNumber:
		f, ok := toFloat(v)
		if !ok {
			return defaultFormat(v)
		}
		decimals := fmt.NumDecimals
		if decimals < 0 {
			decimals = 0
		}
		s := formatFloat(f, decimals)
		return fmt.Prefix + s + fmt.Suffix

	case DataTypeDate:
		layout := fmt.DateFormat
		if layout == "" {
			layout = "2006-01-02"
		}
		switch tv := v.(type) {
		case time.Time:
			return fmt.Prefix + tv.Format(layout) + fmt.Suffix
		case string:
			return fmt.Prefix + tv + fmt.Suffix
		default:
			return defaultFormat(v)
		}

	default: // DataTypeText
		s := defaultFormat(v)
		return fmt.Prefix + s + fmt.Suffix
	}
}

// resolveFormat returns the effective format with precedence: cell > row > column.
func resolveFormat(cell *Cell, rowFmt *CellFormat, colFmt *CellFormat) *CellFormat {
	if cell != nil && cell.Format != nil {
		return cell.Format
	}
	if rowFmt != nil {
		return rowFmt
	}
	return colFmt
}

// cellStyle builds a lipgloss.Style from a CellFormat and cell value.
func cellStyle(v any, fmt *CellFormat) lipgloss.Style {
	s := lipgloss.NewStyle()
	if fmt == nil {
		// Right-align numbers by default
		if _, ok := toFloat(v); ok {
			s = s.Align(lipgloss.Right)
		}
		return s
	}

	if fmt.FgColor != "" {
		s = s.Foreground(lipgloss.Color(fmt.FgColor))
	}
	if fmt.BgColor != "" {
		s = s.Background(lipgloss.Color(fmt.BgColor))
	}
	if fmt.Bold {
		s = s.Bold(true)
	}
	if fmt.Italic {
		s = s.Italic(true)
	}
	if fmt.Underline {
		s = s.Underline(true)
	}

	switch fmt.Align {
	case AlignLeft:
		s = s.Align(lipgloss.Left)
	case AlignCenter:
		s = s.Align(lipgloss.Center)
	case AlignRight:
		s = s.Align(lipgloss.Right)
	default: // AlignAuto
		if _, ok := toFloat(v); ok {
			s = s.Align(lipgloss.Right)
		}
	}

	return s
}

// defaultFormat produces a simple string representation.
func defaultFormat(v any) string {
	switch val := v.(type) {
	case float64:
		return formatFloat(val, 2)
	case string:
		return val
	case error:
		return val.Error()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatFloat formats a float with the given decimal places,
// stripping trailing zeros for cleanliness when the value is integral.
func formatFloat(f float64, decimals int) string {
	s := fmt.Sprintf("%.*f", decimals, f)
	// If it has a decimal point, strip trailing zeros (but keep at least one decimal if decimals > 0)
	if strings.Contains(s, ".") && decimals > 0 {
		return s // keep exact decimal places
	}
	// For zero decimals, no decimal point
	return s
}

// toFloat attempts to convert a value to float64.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
