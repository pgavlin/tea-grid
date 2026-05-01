package querybar

import (
	"testing"

	"github.com/pgavlin/tea-grid/searchquery"
)

func TestState_NewDisabled(t *testing.T) {
	s := New(nil)
	if s.Enabled() {
		t.Errorf("New(nil).Enabled() = true, want false (no vocab built yet)")
	}
}

func TestState_EnableWithVocab(t *testing.T) {
	v := searchquery.NewVocabulary(nil)
	s := New(v)
	s.Enable()
	if !s.Enabled() {
		t.Errorf("Enable() did not enable")
	}
}

func TestState_TextRoundTrip(t *testing.T) {
	s := New(searchquery.NewVocabulary(nil))
	s.Enable()
	s.SetText("hello world")
	if s.Text() != "hello world" {
		t.Errorf("Text() = %q, want %q", s.Text(), "hello world")
	}
}

func TestState_LossyAccessors(t *testing.T) {
	s := New(searchquery.NewVocabulary(nil))
	s.Enable()
	s.SetLossy([]string{"title", "labels"})
	got := s.Lossy()
	if len(got) != 2 || got[0] != "title" || got[1] != "labels" {
		t.Errorf("Lossy() = %v, want [title labels]", got)
	}
}

func TestState_Editing(t *testing.T) {
	s := New(searchquery.NewVocabulary(nil))
	s.Enable()
	if s.Editing() {
		t.Errorf("Editing() = true initially")
	}
	s.BeginEdit()
	if !s.Editing() {
		t.Errorf("BeginEdit did not enter editing mode")
	}
	s.EndEdit()
	if s.Editing() {
		t.Errorf("EndEdit did not exit editing mode")
	}
}

func TestState_VocabOverride(t *testing.T) {
	custom := searchquery.NewVocabulary([]searchquery.Field{{Name: "custom"}})
	s := New(nil)
	s.SetVocabulary(custom)
	if s.Vocab() != custom {
		t.Errorf("Vocab() did not return the override")
	}
}

func TestState_CompleteTab_FreshAndCycle(t *testing.T) {
	s := New(searchquery.NewVocabulary(nil))
	s.Enable()
	s.BeginEdit()
	s.Editor().SetText("st")
	s.Editor().CursorToEnd()

	// Suggest returns three candidates for the same partial.
	calls := 0
	suggest := func(text string, cursor int) ([]string, int, int) {
		calls++
		return []string{"state", "status", "step"}, 0, 2
	}

	if !s.CompleteTab(suggest) {
		t.Fatal("first CompleteTab returned false")
	}
	if got := s.Editor().Text(); got != "state" {
		t.Errorf("after first Tab: text=%q, want state", got)
	}

	// Cycling: next Tab without an intervening edit advances index.
	if !s.CompleteTab(suggest) {
		t.Fatal("second CompleteTab returned false")
	}
	if got := s.Editor().Text(); got != "status" {
		t.Errorf("after second Tab: text=%q, want status", got)
	}
	if calls != 1 {
		t.Errorf("suggest called %d times, want 1 (cycle reuses candidates)", calls)
	}

	if !s.CompleteTab(suggest) {
		t.Fatal("third CompleteTab returned false")
	}
	if got := s.Editor().Text(); got != "step" {
		t.Errorf("after third Tab: text=%q, want step", got)
	}

	// Wrap-around back to first.
	if !s.CompleteTab(suggest) {
		t.Fatal("fourth CompleteTab returned false")
	}
	if got := s.Editor().Text(); got != "state" {
		t.Errorf("after fourth Tab (wrap): text=%q, want state", got)
	}
}

func TestState_CompleteTab_ResetOnEdit(t *testing.T) {
	s := New(searchquery.NewVocabulary(nil))
	s.Enable()
	s.BeginEdit()
	s.Editor().SetText("st")
	s.Editor().CursorToEnd()

	suggest := func(text string, cursor int) ([]string, int, int) {
		return []string{"state", "status"}, 0, 2
	}
	s.CompleteTab(suggest)
	if got := s.Editor().Text(); got != "state" {
		t.Fatalf("text=%q, want state", got)
	}

	// Simulate user editing: this is what handleQueryBarKeyMsg does on
	// any non-Tab key.
	s.ResetCompletion()

	// Next Tab from a different partial should not cycle the old set.
	calls := 0
	suggest2 := func(text string, cursor int) ([]string, int, int) {
		calls++
		return []string{"step"}, 0, 5
	}
	s.CompleteTab(suggest2)
	if calls != 1 {
		t.Errorf("after Reset, suggest called %d times, want 1 (fresh cycle)", calls)
	}
	if got := s.Editor().Text(); got != "step" {
		t.Errorf("text=%q, want step", got)
	}
}

func TestState_CompleteTab_NoCandidates(t *testing.T) {
	s := New(searchquery.NewVocabulary(nil))
	s.Enable()
	s.BeginEdit()
	s.Editor().SetText("xyz")
	s.Editor().CursorToEnd()
	suggest := func(text string, cursor int) ([]string, int, int) {
		return nil, 0, 3
	}
	if s.CompleteTab(suggest) {
		t.Errorf("CompleteTab returned true for empty candidates")
	}
	if got := s.Editor().Text(); got != "xyz" {
		t.Errorf("text=%q, want xyz (unchanged)", got)
	}
}

func TestState_CompletionRange(t *testing.T) {
	s := New(searchquery.NewVocabulary(nil))
	s.Enable()
	s.BeginEdit()
	s.Editor().SetText("st")
	s.Editor().CursorToEnd()

	if _, _, ok := s.CompletionRange(); ok {
		t.Errorf("CompletionRange ok=true before any Tab")
	}

	suggest := func(text string, cursor int) ([]string, int, int) {
		return []string{"state", "status"}, 0, 2
	}
	s.CompleteTab(suggest)
	start, end, ok := s.CompletionRange()
	if !ok {
		t.Fatal("CompletionRange ok=false after Tab")
	}
	if start != 0 || end != 5 { // "state" is 5 bytes
		t.Errorf("range = (%d, %d), want (0, 5)", start, end)
	}

	// Edit invalidates the range.
	s.Editor().Insert("x")
	if _, _, ok := s.CompletionRange(); ok {
		t.Errorf("CompletionRange ok=true after edit")
	}
}
