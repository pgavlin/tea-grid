package main

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pgavlin/tea-grid/internal/lineedit"
)

// SidebarField identifies a field in the sidebar.
type SidebarField int

const (
	fieldTarget SidebarField = iota
	fieldDataType
	fieldDecimals
	fieldPrefix
	fieldSuffix
	fieldDateFormat
	fieldFgColor
	fieldBgColor
	fieldAlign
	fieldBold
	fieldItalic
	fieldUnderline
	fieldCount
)

// SidebarModel is the formatting sidebar component.
type SidebarModel struct {
	// Target cell/column
	targetCell bool // true = cell, false = column
	cellRef    string
	colID      string

	// Current format being edited
	format CellFormat

	// Navigation
	focusedField SidebarField
	editing      bool // true when editing a text field

	// Text input editors for string fields
	editors [fieldCount]lineedit.Model

	// Dimensions
	width  int
	height int
}

func NewSidebar() SidebarModel {
	return SidebarModel{
		targetCell: true,
		format: CellFormat{
			NumDecimals: 2,
			DateFormat:  "2006-01-02",
		},
	}
}

// SetTarget configures the sidebar for the given cell.
func (s *SidebarModel) SetTarget(colID string, rowIndex int, cell *Cell, colFmt *CellFormat) {
	s.colID = colID
	s.cellRef = fmt.Sprintf("%s%d", colID, rowIndex+1)
	s.editing = false

	if s.targetCell {
		if cell != nil && cell.Format != nil {
			s.format = *cell.Format
		} else if colFmt != nil {
			s.format = *colFmt
		} else {
			s.format = CellFormat{NumDecimals: 2, DateFormat: "2006-01-02"}
		}
	} else {
		if colFmt != nil {
			s.format = *colFmt
		} else {
			s.format = CellFormat{NumDecimals: 2, DateFormat: "2006-01-02"}
		}
	}

	s.syncEditors()
}

func (s *SidebarModel) syncEditors() {
	s.editors[fieldPrefix].SetText(s.format.Prefix)
	s.editors[fieldSuffix].SetText(s.format.Suffix)
	s.editors[fieldDateFormat].SetText(s.format.DateFormat)
	s.editors[fieldFgColor].SetText(s.format.FgColor)
	s.editors[fieldBgColor].SetText(s.format.BgColor)
}

// Format returns the current format.
func (s *SidebarModel) Format() *CellFormat {
	f := s.format
	return &f
}

// TargetCell returns whether targeting cell (true) or column (false).
func (s *SidebarModel) TargetCell() bool {
	return s.targetCell
}

// visibleFields returns the fields visible for the current data type.
func (s *SidebarModel) visibleFields() []SidebarField {
	fields := []SidebarField{
		fieldTarget,
		fieldDataType,
	}
	switch s.format.DataType {
	case DataTypeNumber:
		fields = append(fields, fieldDecimals, fieldPrefix, fieldSuffix)
	case DataTypeDate:
		fields = append(fields, fieldDateFormat, fieldPrefix, fieldSuffix)
	default:
		fields = append(fields, fieldPrefix, fieldSuffix)
	}
	fields = append(fields, fieldAlign, fieldFgColor, fieldBgColor, fieldBold, fieldItalic, fieldUnderline)
	return fields
}

func (s *SidebarModel) currentFieldIndex() int {
	visible := s.visibleFields()
	for i, f := range visible {
		if f == s.focusedField {
			return i
		}
	}
	return 0
}

func (s *SidebarModel) isTextEditField(f SidebarField) bool {
	return f == fieldPrefix || f == fieldSuffix || f == fieldDateFormat || f == fieldFgColor || f == fieldBgColor
}

// Update handles messages for the sidebar.
func (s *SidebarModel) Update(msg tea.Msg) tea.Cmd {
	if s.editing {
		return s.updateEditing(msg)
	}
	return s.updateNavigation(msg)
}

func (s *SidebarModel) updateNavigation(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	visible := s.visibleFields()
	idx := s.currentFieldIndex()

	switch keyMsg.String() {
	case "up", "k":
		if idx > 0 {
			s.focusedField = visible[idx-1]
		}
	case "down", "j":
		if idx < len(visible)-1 {
			s.focusedField = visible[idx+1]
		}
	case "left", "h":
		s.handleLeftRight(-1)
	case "right", "l":
		s.handleLeftRight(1)
	case "enter":
		if s.isTextEditField(s.focusedField) {
			s.editing = true
			s.editors[s.focusedField].CursorToEnd()
		}
	case " ":
		s.handleToggle()
	}
	return nil
}

func (s *SidebarModel) updateEditing(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	switch keyMsg.String() {
	case "enter", "esc":
		s.commitEdit()
		s.editing = false
		return nil
	default:
		s.editors[s.focusedField].HandleKeyMsg(keyMsg)
		s.commitEdit()
	}
	return nil
}

func (s *SidebarModel) commitEdit() {
	switch s.focusedField {
	case fieldPrefix:
		s.format.Prefix = s.editors[fieldPrefix].Text()
	case fieldSuffix:
		s.format.Suffix = s.editors[fieldSuffix].Text()
	case fieldDateFormat:
		s.format.DateFormat = s.editors[fieldDateFormat].Text()
	case fieldFgColor:
		s.format.FgColor = s.editors[fieldFgColor].Text()
	case fieldBgColor:
		s.format.BgColor = s.editors[fieldBgColor].Text()
	}
}

func (s *SidebarModel) handleLeftRight(dir int) {
	switch s.focusedField {
	case fieldTarget:
		s.targetCell = !s.targetCell
	case fieldDataType:
		dt := int(s.format.DataType) + dir
		if dt < 0 {
			dt = 2
		}
		if dt > 2 {
			dt = 0
		}
		s.format.DataType = DataType(dt)
	case fieldDecimals:
		s.format.NumDecimals += dir
		if s.format.NumDecimals < 0 {
			s.format.NumDecimals = 0
		}
		if s.format.NumDecimals > 10 {
			s.format.NumDecimals = 10
		}
	case fieldAlign:
		a := int(s.format.Align) + dir
		if a < 0 {
			a = int(AlignRight)
		}
		if a > int(AlignRight) {
			a = 0
		}
		s.format.Align = Alignment(a)
	}
}

func (s *SidebarModel) handleToggle() {
	switch s.focusedField {
	case fieldBold:
		s.format.Bold = !s.format.Bold
	case fieldItalic:
		s.format.Italic = !s.format.Italic
	case fieldUnderline:
		s.format.Underline = !s.format.Underline
	}
}

// View renders the sidebar.
func (s *SidebarModel) View() string {
	if s.width < 10 {
		return ""
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(s.width - 2).
		Height(s.height - 2).
		Padding(0, 1)

	title := lipgloss.NewStyle().Bold(true).Render("Format")
	content := title + "\n\n"

	visible := s.visibleFields()
	for _, f := range visible {
		focused := f == s.focusedField
		content += s.renderField(f, focused) + "\n"
	}

	return border.Render(content)
}

func (s *SidebarModel) renderField(f SidebarField, focused bool) string {
	label := s.fieldLabel(f)
	value := s.fieldValue(f)

	focusStyle := lipgloss.NewStyle()
	if focused {
		focusStyle = focusStyle.Foreground(lipgloss.Color("212"))
	}

	labelStyle := lipgloss.NewStyle().Width(10).Foreground(lipgloss.Color("245"))
	if focused {
		labelStyle = labelStyle.Foreground(lipgloss.Color("212"))
	}

	indicator := "  "
	if focused {
		indicator = "> "
	}

	if s.editing && focused && s.isTextEditField(f) {
		editWidth := s.width - 16
		if editWidth < 4 {
			editWidth = 4
		}
		return focusStyle.Render(indicator+labelStyle.Render(label)) + "\n" +
			"  " + s.editors[f].RenderLine(editWidth, "")
	}

	return focusStyle.Render(indicator + labelStyle.Render(label) + value)
}

func (s *SidebarModel) fieldLabel(f SidebarField) string {
	switch f {
	case fieldTarget:
		return "Target:"
	case fieldDataType:
		return "Type:"
	case fieldDecimals:
		return "Decimals:"
	case fieldPrefix:
		return "Prefix:"
	case fieldSuffix:
		return "Suffix:"
	case fieldDateFormat:
		return "Date Fmt:"
	case fieldFgColor:
		return "FG Color:"
	case fieldBgColor:
		return "BG Color:"
	case fieldAlign:
		return "Align:"
	case fieldBold:
		return "Bold:"
	case fieldItalic:
		return "Italic:"
	case fieldUnderline:
		return "Underline:"
	default:
		return ""
	}
}

func (s *SidebarModel) fieldValue(f SidebarField) string {
	switch f {
	case fieldTarget:
		if s.targetCell {
			return "< Cell " + s.cellRef + " >"
		}
		return "< Column " + s.colID + " >"
	case fieldDataType:
		switch s.format.DataType {
		case DataTypeText:
			return "< Text >"
		case DataTypeNumber:
			return "< Number >"
		case DataTypeDate:
			return "< Date >"
		}
	case fieldDecimals:
		return "< " + strconv.Itoa(s.format.NumDecimals) + " >"
	case fieldPrefix:
		if s.format.Prefix == "" {
			return "(none)"
		}
		return `"` + s.format.Prefix + `"`
	case fieldSuffix:
		if s.format.Suffix == "" {
			return "(none)"
		}
		return `"` + s.format.Suffix + `"`
	case fieldDateFormat:
		if s.format.DateFormat == "" {
			return "(none)"
		}
		return s.format.DateFormat
	case fieldFgColor:
		if s.format.FgColor == "" {
			return "(default)"
		}
		return s.format.FgColor
	case fieldBgColor:
		if s.format.BgColor == "" {
			return "(default)"
		}
		return s.format.BgColor
	case fieldAlign:
		switch s.format.Align {
		case AlignAuto:
			return "< Auto >"
		case AlignLeft:
			return "< Left >"
		case AlignCenter:
			return "< Center >"
		case AlignRight:
			return "< Right >"
		}
	case fieldBold:
		return boolCheck(s.format.Bold)
	case fieldItalic:
		return boolCheck(s.format.Italic)
	case fieldUnderline:
		return boolCheck(s.format.Underline)
	}
	return ""
}

func boolCheck(v bool) string {
	if v {
		return "[x]"
	}
	return "[ ]"
}
