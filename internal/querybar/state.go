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

	// Tab-completion cycle state. Cleared on any non-Tab edit.
	// completionCandidates is the list of matches for the partial that
	// was active when the cycle began; completionIndex is the position
	// of the most-recently-inserted candidate; completionStart is the
	// byte offset where the partial began; completionLastText is the
	// editor text after the last candidate was inserted (used to detect
	// whether the user has edited since).
	completionCandidates []string
	completionIndex      int
	completionStart      int
	completionLastText   string
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

// CompleteTab advances the completion cycle by one. If a cycle is in
// progress (the editor text matches what was left after the previous
// Tab) the next candidate replaces the current one. Otherwise a fresh
// cycle starts: candidates are computed via the supplied suggest fn
// and the first match is inserted. Returns true if any candidate was
// inserted.
//
// suggest is a closure that takes (text, cursor) and returns
// (candidates, start, end). It is called only when starting a fresh
// cycle. Supplying it as a closure keeps this package free of a
// generic dependency on data.Column[T].
func (s *State) CompleteTab(suggest func(text string, cursor int) (cands []string, start, end int)) bool {
	if s.completionCandidates != nil && s.editor.Text() == s.completionLastText {
		s.completionIndex = (s.completionIndex + 1) % len(s.completionCandidates)
		s.applyCompletion()
		return true
	}
	cands, start, end := suggest(s.editor.Text(), s.editor.Cursor())
	if len(cands) == 0 {
		s.ResetCompletion()
		return false
	}
	s.completionCandidates = cands
	s.completionIndex = 0
	s.completionStart = start
	t := s.editor.Text()
	s.editor.SetText(t[:start] + cands[0] + t[end:])
	s.editor.SetCursor(start + len(cands[0]))
	s.completionLastText = s.editor.Text()
	return true
}

// applyCompletion replaces the previously-inserted candidate (which
// runs from completionStart to the cursor) with the candidate at
// completionIndex.
func (s *State) applyCompletion() {
	prev := s.completionCandidates[(s.completionIndex-1+len(s.completionCandidates))%len(s.completionCandidates)]
	next := s.completionCandidates[s.completionIndex]
	t := s.editor.Text()
	end := s.completionStart + len(prev)
	if end > len(t) {
		end = len(t)
	}
	s.editor.SetText(t[:s.completionStart] + next + t[end:])
	s.editor.SetCursor(s.completionStart + len(next))
	s.completionLastText = s.editor.Text()
}

// ResetCompletion clears the Tab-completion cycle state. Called by
// the bar's key handler whenever a non-Tab key edits the buffer.
func (s *State) ResetCompletion() {
	s.completionCandidates = nil
	s.completionIndex = 0
	s.completionStart = 0
	s.completionLastText = ""
}

// CompletionRange returns the byte range of the most-recently-inserted
// completion candidate in the editor text, suitable for dim styling
// in the bar render. ok=false when no cycle is in progress or the
// editor has been edited since the last completion was applied.
func (s *State) CompletionRange() (start, end int, ok bool) {
	if s.completionCandidates == nil || s.editor.Text() != s.completionLastText {
		return 0, 0, false
	}
	cand := s.completionCandidates[s.completionIndex]
	return s.completionStart, s.completionStart + len(cand), true
}
