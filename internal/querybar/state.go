package querybar

import (
	"github.com/pgavlin/tea-grid/internal/lineedit"
	"github.com/pgavlin/tea-grid/searchquery"
)

// State holds the query bar's UI state and its current view of the
// canonical filter state (text + lossy column IDs). It is consumed by
// grid.Model[T]; consumers configure the bar via WithQueryBar* options.
//
// State is non-generic: its content (text, lossy IDs, editing flags)
// does not depend on T. The generic Apply / Rerender functions in
// bind.go take cols []data.Column[T] separately.
type State struct {
	enabled bool
	editing bool

	editor lineedit.Model

	// text is the canonical projection of the current filter state.
	// It mirrors the filter side; user edits go through the editor and
	// are committed to text on submit.
	text string

	// lossy is the set of column IDs whose filter state could not be
	// represented in the bar (regex on TextFilter, non-RoundTrippable
	// filters). The bar renders these in an annotation, not text.
	lossy []string

	// parseErr holds the last parse error from a submit, surfaced in
	// the bar's status line.
	parseErr string

	// auto is the column-derived vocabulary; rebuilt on column changes.
	// custom is an explicit override set via WithQueryBarVocabulary; if
	// non-nil, it shadows auto.
	auto   *searchquery.Vocabulary
	custom *searchquery.Vocabulary
}

// New returns a new State with the given auto-vocabulary. The bar is
// disabled until Enable is called. The auto vocabulary may be nil; it
// is rebuilt by SetAutoVocabulary when columns are known.
func New(auto *searchquery.Vocabulary) *State {
	return &State{auto: auto}
}

// Enable marks the bar as active. Until enabled, grid.Model[T] does
// not render the bar or route keys to it.
func (s *State) Enable() { s.enabled = true }

// Enabled reports whether the bar is enabled.
func (s *State) Enabled() bool { return s.enabled }

// Editing reports whether the bar's textinput currently has focus.
func (s *State) Editing() bool { return s.editing }

// BeginEdit puts focus on the textinput and copies the canonical text
// into the editor for editing.
func (s *State) BeginEdit() {
	s.editing = true
	s.editor.SetText(s.text)
	s.editor.CursorToEnd()
}

// EndEdit drops focus on the textinput and discards uncommitted edits.
func (s *State) EndEdit() {
	s.editing = false
	s.editor.SetText(s.text)
}

// Text returns the canonical bar text (the projection of filter state).
func (s *State) Text() string { return s.text }

// SetText replaces the canonical bar text. Called by Rerender.
func (s *State) SetText(t string) {
	s.text = t
	if !s.editing {
		s.editor.SetText(t)
	}
}

// EditorText returns the textinput's current value (may diverge from
// Text while Editing()).
func (s *State) EditorText() string { return s.editor.Text() }

// Editor exposes the underlying editor for grid.Model to forward key
// messages into.
func (s *State) Editor() *lineedit.Model { return &s.editor }

// Lossy returns the set of column IDs in lossy state.
func (s *State) Lossy() []string {
	out := make([]string, len(s.lossy))
	copy(out, s.lossy)
	return out
}

// SetLossy replaces the lossy-column-ID set. Called by Rerender.
func (s *State) SetLossy(ids []string) {
	s.lossy = append(s.lossy[:0], ids...)
}

// ParseErr returns the last parse error from a submit, or "".
func (s *State) ParseErr() string { return s.parseErr }

// SetParseErr stores a parse-error string for display in the status
// line.
func (s *State) SetParseErr(e string) { s.parseErr = e }

// SetAutoVocabulary updates the column-derived vocabulary. Called when
// columns change (SetColumns).
func (s *State) SetAutoVocabulary(v *searchquery.Vocabulary) { s.auto = v }

// SetVocabulary sets an explicit override that shadows the column-
// derived vocabulary.
func (s *State) SetVocabulary(v *searchquery.Vocabulary) { s.custom = v }

// Vocab returns the active vocabulary: the override if set, otherwise
// the column-derived one.
func (s *State) Vocab() *searchquery.Vocabulary {
	if s.custom != nil {
		return s.custom
	}
	return s.auto
}
