package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"
)

// --- Native Binary Format (CBOR + zstd) ---

// Magic number: "SHEETXS\x00" (8 bytes)
var nativeMagic = [8]byte{'S', 'H', 'E', 'E', 'T', 'X', 'S', 0x00}

type fileData struct {
	Version    int                        `cbor:"version"`
	NumCols    int                        `cbor:"numCols"`
	NumRows    int                        `cbor:"numRows"`
	ColFormats map[string]*fileCellFormat `cbor:"colFormats,omitempty"`
	RowFormats map[int]*fileCellFormat    `cbor:"rowFormats,omitempty"`
	Cells      map[string]*fileCell       `cbor:"cells"`
}

type fileCellFormat struct {
	DataType    DataType  `cbor:"dataType,omitempty"`
	NumDecimals int       `cbor:"numDecimals,omitempty"`
	DateFormat  string    `cbor:"dateFormat,omitempty"`
	Prefix      string    `cbor:"prefix,omitempty"`
	Suffix      string    `cbor:"suffix,omitempty"`
	Align       Alignment `cbor:"align,omitempty"`
	FgColor     string    `cbor:"fgColor,omitempty"`
	BgColor     string    `cbor:"bgColor,omitempty"`
	Bold        bool      `cbor:"bold,omitempty"`
	Italic      bool      `cbor:"italic,omitempty"`
	Underline   bool      `cbor:"underline,omitempty"`
}

type fileCell struct {
	Raw    string          `cbor:"raw"`
	Format *fileCellFormat `cbor:"format,omitempty"`
}

func cellFormatToFile(f *CellFormat) *fileCellFormat {
	if f == nil {
		return nil
	}
	return &fileCellFormat{
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

func fileToCellFormat(j *fileCellFormat) *CellFormat {
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

// saveNative writes the spreadsheet to a binary file (CBOR + zstd).
func saveNative(filename string, rows []*SpreadsheetRow, colFmts map[string]*CellFormat, rowFmts map[int]*CellFormat, numCols int) error {
	file := fileData{
		Version:    1,
		NumCols:    numCols,
		NumRows:    len(rows),
		ColFormats: make(map[string]*fileCellFormat),
		Cells:      make(map[string]*fileCell),
	}

	for col, fmt := range colFmts {
		if fmt != nil {
			file.ColFormats[col] = cellFormatToFile(fmt)
		}
	}

	if len(rowFmts) > 0 {
		file.RowFormats = make(map[int]*fileCellFormat)
		for idx, fmt := range rowFmts {
			if fmt != nil {
				file.RowFormats[idx] = cellFormatToFile(fmt)
			}
		}
	}

	for _, row := range rows {
		for col, cell := range row.Cells {
			if cell.Raw == "" {
				continue
			}
			ref := fmt.Sprintf("%s%d", col, row.RowIndex+1)
			fc := &fileCell{Raw: cell.Raw}
			if cell.Format != nil {
				fc.Format = cellFormatToFile(cell.Format)
			}
			file.Cells[ref] = fc
		}
	}

	payload, err := cbor.Marshal(file)
	if err != nil {
		return err
	}

	enc, err := zstd.NewWriter(nil)
	if err != nil {
		return err
	}
	defer enc.Close()
	compressed := enc.EncodeAll(payload, nil)

	out := make([]byte, 0, len(nativeMagic)+len(compressed))
	out = append(out, nativeMagic[:]...)
	out = append(out, compressed...)
	return os.WriteFile(filename, out, 0o644)
}

// loadNative reads a spreadsheet from a binary file (CBOR + zstd).
// Returns rows, column formats, row formats, number of columns.
func loadNative(filename string) ([]*SpreadsheetRow, map[string]*CellFormat, map[int]*CellFormat, int, error) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	if len(raw) < len(nativeMagic) || [8]byte(raw[:8]) != nativeMagic {
		return nil, nil, nil, 0, fmt.Errorf("not a valid .txs file (bad magic number)")
	}

	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	defer dec.Close()

	payload, err := dec.DecodeAll(raw[len(nativeMagic):], nil)
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("decompression failed: %w", err)
	}

	var file fileData
	if err := cbor.Unmarshal(payload, &file); err != nil {
		return nil, nil, nil, 0, fmt.Errorf("decode failed: %w", err)
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
	for ref, fc := range file.Cells {
		col, rowNum := parseCellRef(ref)
		if col == "" || rowNum < 1 || rowNum > numRows {
			continue
		}
		cell := &Cell{Raw: fc.Raw}
		if fc.Format != nil {
			cell.Format = fileToCellFormat(fc.Format)
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
	for col, ff := range file.ColFormats {
		colFmts[col] = fileToCellFormat(ff)
	}

	// Convert row formats
	rowFmts := make(map[int]*CellFormat)
	for idx, ff := range file.RowFormats {
		rowFmts[idx] = fileToCellFormat(ff)
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
func csvFilename(name string) string {
	if strings.HasSuffix(name, ".txs") {
		return strings.TrimSuffix(name, ".txs") + ".csv"
	}
	return name + ".csv"
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
