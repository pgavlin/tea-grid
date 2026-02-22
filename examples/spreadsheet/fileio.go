package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// --- JSON Format ---

type jsonFile struct {
	Version    int                         `json:"version"`
	NumCols    int                         `json:"numCols"`
	NumRows    int                         `json:"numRows"`
	ColFormats map[string]*jsonCellFormat  `json:"colFormats,omitempty"`
	RowFormats map[int]*jsonCellFormat     `json:"rowFormats,omitempty"`
	Cells      map[string]*jsonCell        `json:"cells"`
}

type jsonCellFormat struct {
	DataType    DataType  `json:"dataType,omitempty"`
	NumDecimals int       `json:"numDecimals,omitempty"`
	DateFormat  string    `json:"dateFormat,omitempty"`
	Prefix      string    `json:"prefix,omitempty"`
	Suffix      string    `json:"suffix,omitempty"`
	Align       Alignment `json:"align,omitempty"`
	FgColor     string    `json:"fgColor,omitempty"`
	BgColor     string    `json:"bgColor,omitempty"`
	Bold        bool      `json:"bold,omitempty"`
	Italic      bool      `json:"italic,omitempty"`
	Underline   bool      `json:"underline,omitempty"`
}

type jsonCell struct {
	Raw    string          `json:"raw"`
	Format *jsonCellFormat `json:"format,omitempty"`
}

func cellFormatToJSON(f *CellFormat) *jsonCellFormat {
	if f == nil {
		return nil
	}
	return &jsonCellFormat{
		DataType:    f.DataType,
		NumDecimals: f.NumDecimals,
		DateFormat:  f.DateFormat,
		Prefix:      f.Prefix,
		Suffix:      f.Suffix,
		Align:       f.Align,
		FgColor:     f.FgColor,
		BgColor:     f.BgColor,
		Bold:        f.Bold,
		Italic:      f.Italic,
		Underline:   f.Underline,
	}
}

func jsonToCellFormat(j *jsonCellFormat) *CellFormat {
	if j == nil {
		return nil
	}
	return &CellFormat{
		DataType:    j.DataType,
		NumDecimals: j.NumDecimals,
		DateFormat:  j.DateFormat,
		Prefix:      j.Prefix,
		Suffix:      j.Suffix,
		Align:       j.Align,
		FgColor:     j.FgColor,
		BgColor:     j.BgColor,
		Bold:        j.Bold,
		Italic:      j.Italic,
		Underline:   j.Underline,
	}
}

// saveJSON writes the spreadsheet to a JSON file.
func saveJSON(filename string, rows []*SpreadsheetRow, colFmts map[string]*CellFormat, rowFmts map[int]*CellFormat, numCols int) error {
	file := jsonFile{
		Version:    1,
		NumCols:    numCols,
		NumRows:    len(rows),
		ColFormats: make(map[string]*jsonCellFormat),
		Cells:      make(map[string]*jsonCell),
	}

	for col, fmt := range colFmts {
		if fmt != nil {
			file.ColFormats[col] = cellFormatToJSON(fmt)
		}
	}

	if len(rowFmts) > 0 {
		file.RowFormats = make(map[int]*jsonCellFormat)
		for idx, fmt := range rowFmts {
			if fmt != nil {
				file.RowFormats[idx] = cellFormatToJSON(fmt)
			}
		}
	}

	for _, row := range rows {
		for col, cell := range row.Cells {
			if cell.Raw == "" {
				continue
			}
			ref := fmt.Sprintf("%s%d", col, row.RowIndex+1)
			jc := &jsonCell{Raw: cell.Raw}
			if cell.Format != nil {
				jc.Format = cellFormatToJSON(cell.Format)
			}
			file.Cells[ref] = jc
		}
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// loadJSON reads a spreadsheet from a JSON file.
// Returns rows, column formats, row formats, number of columns.
func loadJSON(filename string) ([]*SpreadsheetRow, map[string]*CellFormat, map[int]*CellFormat, int, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	var file jsonFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, nil, nil, 0, err
	}

	numCols := file.NumCols
	if numCols < 1 {
		numCols = 10
	}
	numRows := file.NumRows
	if numRows < 1 {
		numRows = 20
	}

	// Create rows
	rows := make([]*SpreadsheetRow, numRows)
	for i := range rows {
		rows[i] = &SpreadsheetRow{
			RowIndex: i,
			Cells:    make(map[string]*Cell),
		}
	}

	// Populate cells
	for ref, jc := range file.Cells {
		col, rowNum := parseCellRef(ref)
		if col == "" || rowNum < 1 || rowNum > numRows {
			continue
		}
		cell := &Cell{Raw: jc.Raw}
		if jc.Format != nil {
			cell.Format = jsonToCellFormat(jc.Format)
		}
		rows[rowNum-1].Cells[col] = cell
	}

	// Ensure all cells exist
	for _, row := range rows {
		for i := 0; i < numCols; i++ {
			col := indexToColLetter(i)
			if _, ok := row.Cells[col]; !ok {
				row.Cells[col] = &Cell{}
			}
		}
	}

	// Convert column formats
	colFmts := make(map[string]*CellFormat)
	for col, jf := range file.ColFormats {
		colFmts[col] = jsonToCellFormat(jf)
	}

	// Convert row formats
	rowFmts := make(map[int]*CellFormat)
	for idx, jf := range file.RowFormats {
		rowFmts[idx] = jsonToCellFormat(jf)
	}

	return rows, colFmts, rowFmts, numCols, nil
}

// --- CSV Import/Export ---

// exportCSV writes computed values to a CSV file.
func exportCSV(filename string, rows []*SpreadsheetRow, numCols int) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header row: column letters
	header := make([]string, numCols)
	for i := range header {
		header[i] = indexToColLetter(i)
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// Data rows: computed values
	for _, row := range rows {
		record := make([]string, numCols)
		for i := 0; i < numCols; i++ {
			col := indexToColLetter(i)
			cell, ok := row.Cells[col]
			if !ok || cell.Value == nil {
				record[i] = ""
				continue
			}
			record[i] = fmt.Sprintf("%v", cell.Value)
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// importCSV reads a CSV file and creates literal cells.
func importCSV(filename string) ([]*SpreadsheetRow, int, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, 0, err
	}
	if len(records) < 1 {
		return nil, 0, fmt.Errorf("CSV file is empty")
	}

	// First row is headers (column letters) - skip it if it matches pattern
	dataStart := 0
	headers := records[0]
	numCols := len(headers)

	// Check if first row is header row
	if len(headers) > 0 && isColHeader(headers) {
		dataStart = 1
	}

	if numCols < 1 {
		numCols = 10
	}

	rows := make([]*SpreadsheetRow, len(records)-dataStart)
	for i := dataStart; i < len(records); i++ {
		row := &SpreadsheetRow{
			RowIndex: i - dataStart,
			Cells:    make(map[string]*Cell),
		}
		for j := 0; j < numCols; j++ {
			col := indexToColLetter(j)
			raw := ""
			if j < len(records[i]) {
				raw = records[i][j]
			}
			row.Cells[col] = &Cell{Raw: raw}
		}
		rows[i-dataStart] = row
	}

	return rows, numCols, nil
}

// isColHeader checks if a row looks like column letter headers.
func isColHeader(headers []string) bool {
	if len(headers) == 0 {
		return false
	}
	for i, h := range headers {
		expected := indexToColLetter(i)
		if strings.ToUpper(strings.TrimSpace(h)) != expected {
			return false
		}
	}
	return true
}

// csvFilename derives a CSV filename from the current filename.
func csvFilename(jsonName string) string {
	if strings.HasSuffix(jsonName, ".json") {
		return strings.TrimSuffix(jsonName, ".json") + ".csv"
	}
	return jsonName + ".csv"
}

// sortedColLetters returns column letters in sorted order.
func sortedColLetters(numCols int) []string {
	cols := make([]string, numCols)
	for i := range cols {
		cols[i] = indexToColLetter(i)
	}
	sort.Strings(cols)
	return cols
}
