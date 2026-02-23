package selection

import (
	"testing"
)

// --- New ---

func TestNew(t *testing.T) {
	m := New(SelectMulti)
	if m.Mode != SelectMulti {
		t.Errorf("expected SelectMulti, got %d", m.Mode)
	}
	if m.Active() {
		t.Error("new model should not be active")
	}
}

// --- Active / Clear ---

func TestActive_EmptyIsInactive(t *testing.T) {
	m := New(SelectMulti)
	if m.Active() {
		t.Error("expected inactive when no rects")
	}
}

func TestClear(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindFullRow, Anchor: Position{0, 0}, Cursor: Position{0, 3}})
	m.Clear()
	if m.Active() {
		t.Error("expected inactive after Clear")
	}
}

// --- Replace ---

func TestReplace_SelectNoneIsNoop(t *testing.T) {
	m := New(SelectNone)
	m.Replace(Rect{Kind: KindFullRow, Anchor: Position{0, 0}, Cursor: Position{0, 3}})
	if m.Active() {
		t.Error("Replace should be no-op in SelectNone mode")
	}
}

func TestReplace_SetsExactlyOneRect(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindFullRow, Anchor: Position{0, 0}, Cursor: Position{0, 3}})
	if len(m.Rects) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(m.Rects))
	}
	if m.Rects[0].Kind != KindFullRow {
		t.Errorf("expected KindFullRow, got %d", m.Rects[0].Kind)
	}
}

func TestReplace_OverwritesPrevious(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindFullRow, Anchor: Position{0, 0}, Cursor: Position{0, 3}})
	m.Replace(Rect{Kind: KindFullCol, Anchor: Position{0, 1}, Cursor: Position{9, 1}})
	if len(m.Rects) != 1 {
		t.Fatalf("expected 1 rect after second Replace, got %d", len(m.Rects))
	}
	if m.Rects[0].Kind != KindFullCol {
		t.Error("expected KindFullCol after Replace")
	}
}

// --- ToggleFullRow ---

func TestToggleFullRow_SelectNoneIsNoop(t *testing.T) {
	m := New(SelectNone)
	m.ToggleFullRow(0)
	if m.Active() {
		t.Error("ToggleFullRow should be no-op in SelectNone mode")
	}
}

func TestToggleFullRow_AddsRow(t *testing.T) {
	m := New(SelectMulti)
	m.ToggleFullRow(2)
	if len(m.Rects) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(m.Rects))
	}
	if m.Rects[0].Kind != KindFullRow {
		t.Error("expected KindFullRow")
	}
	if m.Rects[0].Anchor.Row != 2 || m.Rects[0].Cursor.Row != 2 {
		t.Errorf("expected row 2, got anchor=%d cursor=%d", m.Rects[0].Anchor.Row, m.Rects[0].Cursor.Row)
	}
}

func TestToggleFullRow_RemovesExistingRow(t *testing.T) {
	m := New(SelectMulti)
	m.ToggleFullRow(2)
	m.ToggleFullRow(2) // toggle off
	if m.Active() {
		t.Error("expected no rects after toggling same row twice")
	}
}

func TestToggleFullRow_AccumulatesMultipleRows(t *testing.T) {
	m := New(SelectMulti)
	m.ToggleFullRow(1)
	m.ToggleFullRow(5)
	m.ToggleFullRow(9)
	if len(m.Rects) != 3 {
		t.Fatalf("expected 3 rects, got %d", len(m.Rects))
	}
}

func TestToggleFullRow_RemovesMiddleRow(t *testing.T) {
	m := New(SelectMulti)
	m.ToggleFullRow(1)
	m.ToggleFullRow(5)
	m.ToggleFullRow(9)
	m.ToggleFullRow(5) // remove row 5
	if len(m.Rects) != 2 {
		t.Fatalf("expected 2 rects after removing middle, got %d", len(m.Rects))
	}
}

func TestToggleFullRow_ClearsNonFullRowRects(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindRect, Anchor: Position{0, 0}, Cursor: Position{3, 3}})
	m.ToggleFullRow(2)
	if len(m.Rects) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(m.Rects))
	}
	if m.Rects[0].Kind != KindFullRow {
		t.Error("expected KindFullRow")
	}
}

func TestToggleFullRow_SingleModeReplacesExisting(t *testing.T) {
	m := New(SelectSingle)
	m.ToggleFullRow(1)
	m.ToggleFullRow(5)
	if len(m.Rects) != 1 {
		t.Fatalf("expected 1 rect in single mode, got %d", len(m.Rects))
	}
	if m.Rects[0].Anchor.Row != 5 {
		t.Errorf("expected row 5, got %d", m.Rects[0].Anchor.Row)
	}
}

// --- ContainsCell ---

func TestContainsCell_NoSelect(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindFullRow, Anchor: Position{0, 0}, Cursor: Position{0, 3}})
	if m.ContainsCell(0, 1, true) {
		t.Error("ContainsCell should return false when noSelect is true")
	}
}

func TestContainsCell_Empty(t *testing.T) {
	m := New(SelectMulti)
	if m.ContainsCell(0, 0, false) {
		t.Error("ContainsCell should return false with no rects")
	}
}

func TestContainsCell_FullRow(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindFullRow, Anchor: Position{2, 0}, Cursor: Position{4, 3}})

	if !m.ContainsCell(2, 0, false) {
		t.Error("expected (2,0) in full-row selection")
	}
	if !m.ContainsCell(3, 99, false) {
		t.Error("expected (3,99) in full-row selection (any column)")
	}
	if m.ContainsCell(1, 0, false) {
		t.Error("expected (1,0) NOT in full-row selection")
	}
	if m.ContainsCell(5, 0, false) {
		t.Error("expected (5,0) NOT in full-row selection")
	}
}

func TestContainsCell_FullCol(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindFullCol, Anchor: Position{0, 2}, Cursor: Position{9, 4}})

	if !m.ContainsCell(0, 2, false) {
		t.Error("expected (0,2) in full-col selection")
	}
	if !m.ContainsCell(99, 3, false) {
		t.Error("expected (99,3) in full-col selection (any row)")
	}
	if m.ContainsCell(0, 1, false) {
		t.Error("expected (0,1) NOT in full-col selection")
	}
	if m.ContainsCell(0, 5, false) {
		t.Error("expected (0,5) NOT in full-col selection")
	}
}

func TestContainsCell_Rect(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindRect, Anchor: Position{1, 1}, Cursor: Position{3, 3}})

	if !m.ContainsCell(2, 2, false) {
		t.Error("expected (2,2) in rect selection")
	}
	if !m.ContainsCell(1, 1, false) {
		t.Error("expected (1,1) in rect selection (anchor corner)")
	}
	if !m.ContainsCell(3, 3, false) {
		t.Error("expected (3,3) in rect selection (cursor corner)")
	}
	if m.ContainsCell(0, 2, false) {
		t.Error("expected (0,2) NOT in rect selection")
	}
	if m.ContainsCell(2, 0, false) {
		t.Error("expected (2,0) NOT in rect selection")
	}
}

func TestContainsCell_RectReversedAnchors(t *testing.T) {
	// Anchor > Cursor (selection made upward/leftward)
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindRect, Anchor: Position{5, 5}, Cursor: Position{2, 2}})

	if !m.ContainsCell(3, 3, false) {
		t.Error("expected (3,3) in reversed-anchor rect")
	}
	if m.ContainsCell(1, 3, false) {
		t.Error("expected (1,3) NOT in reversed-anchor rect")
	}
}

// --- FullRowRanges ---

func TestFullRowRanges_Empty(t *testing.T) {
	m := New(SelectMulti)
	ranges := m.FullRowRanges()
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges, got %d", len(ranges))
	}
}

func TestFullRowRanges_NonRowRectsExcluded(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindRect, Anchor: Position{0, 0}, Cursor: Position{5, 5}})
	ranges := m.FullRowRanges()
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges for KindRect, got %d", len(ranges))
	}
}

func TestFullRowRanges_MultipleRows(t *testing.T) {
	m := New(SelectMulti)
	m.ToggleFullRow(1)
	m.ToggleFullRow(5)
	m.ToggleFullRow(9)
	ranges := m.FullRowRanges()
	if len(ranges) != 3 {
		t.Fatalf("expected 3 ranges, got %d", len(ranges))
	}
}

func TestFullRowRanges_ReversedAnchor(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindFullRow, Anchor: Position{5, 0}, Cursor: Position{2, 3}})
	ranges := m.FullRowRanges()
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	if ranges[0][0] != 2 || ranges[0][1] != 5 {
		t.Errorf("expected (2,5), got (%d,%d)", ranges[0][0], ranges[0][1])
	}
}

// --- BoundingRect ---

func TestBoundingRect_Empty(t *testing.T) {
	m := New(SelectMulti)
	rLo, rHi, cLo, cHi := m.BoundingRect()
	if rLo != -1 || rHi != -1 || cLo != -1 || cHi != -1 {
		t.Errorf("expected (-1,-1,-1,-1), got (%d,%d,%d,%d)", rLo, rHi, cLo, cHi)
	}
}

func TestBoundingRect_SingleRect(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindRect, Anchor: Position{1, 2}, Cursor: Position{4, 5}})
	rLo, rHi, cLo, cHi := m.BoundingRect()
	if rLo != 1 || rHi != 4 || cLo != 2 || cHi != 5 {
		t.Errorf("expected (1,4,2,5), got (%d,%d,%d,%d)", rLo, rHi, cLo, cHi)
	}
}

func TestBoundingRect_MultipleRects(t *testing.T) {
	m := New(SelectMulti)
	m.Rects = []Rect{
		{Kind: KindFullRow, Anchor: Position{2, 0}, Cursor: Position{2, 3}},
		{Kind: KindFullRow, Anchor: Position{8, 0}, Cursor: Position{8, 3}},
	}
	rLo, rHi, cLo, cHi := m.BoundingRect()
	if rLo != 2 || rHi != 8 || cLo != 0 || cHi != 3 {
		t.Errorf("expected (2,8,0,3), got (%d,%d,%d,%d)", rLo, rHi, cLo, cHi)
	}
}

func TestBoundingRect_AllBranches(t *testing.T) {
	// Exercise all four comparison branches:
	// Second rect has: rLo < rowLo, rHi > rowHi, cLo < colLo, cHi > colHi
	m := New(SelectMulti)
	m.Rects = []Rect{
		{Kind: KindRect, Anchor: Position{3, 3}, Cursor: Position{5, 5}},
		{Kind: KindRect, Anchor: Position{1, 1}, Cursor: Position{8, 8}},
	}
	rLo, rHi, cLo, cHi := m.BoundingRect()
	if rLo != 1 || rHi != 8 || cLo != 1 || cHi != 8 {
		t.Errorf("expected (1,8,1,8), got (%d,%d,%d,%d)", rLo, rHi, cLo, cHi)
	}
}

func TestBoundingRect_PartialBranches(t *testing.T) {
	// First rect already has the lowest row but highest col
	m := New(SelectMulti)
	m.Rects = []Rect{
		{Kind: KindRect, Anchor: Position{1, 5}, Cursor: Position{6, 8}},
		{Kind: KindRect, Anchor: Position{0, 2}, Cursor: Position{3, 4}},
	}
	rLo, rHi, cLo, cHi := m.BoundingRect()
	if rLo != 0 || rHi != 6 || cLo != 2 || cHi != 8 {
		t.Errorf("expected (0,6,2,8), got (%d,%d,%d,%d)", rLo, rHi, cLo, cHi)
	}
}

func TestBoundingRect_ReversedAnchors(t *testing.T) {
	m := New(SelectMulti)
	m.Replace(Rect{Kind: KindRect, Anchor: Position{5, 5}, Cursor: Position{1, 1}})
	rLo, rHi, cLo, cHi := m.BoundingRect()
	if rLo != 1 || rHi != 5 || cLo != 1 || cHi != 5 {
		t.Errorf("expected (1,5,1,5), got (%d,%d,%d,%d)", rLo, rHi, cLo, cHi)
	}
}

// --- Mode enforcement ---

func TestSelectSingle_ReplaceEnforcesOneRect(t *testing.T) {
	m := New(SelectSingle)
	m.Replace(Rect{Kind: KindFullRow, Anchor: Position{0, 0}, Cursor: Position{0, 3}})
	if len(m.Rects) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(m.Rects))
	}
	m.Replace(Rect{Kind: KindFullRow, Anchor: Position{5, 0}, Cursor: Position{5, 3}})
	if len(m.Rects) != 1 {
		t.Fatalf("expected 1 rect after second Replace, got %d", len(m.Rects))
	}
	if m.Rects[0].Anchor.Row != 5 {
		t.Errorf("expected row 5, got %d", m.Rects[0].Anchor.Row)
	}
}
