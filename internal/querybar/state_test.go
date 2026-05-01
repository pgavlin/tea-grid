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
