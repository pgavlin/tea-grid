package grid

import "testing"

func TestVisibleRowRange_NegativeTopRow(t *testing.T) {
	vp := viewport{topRow: -1, visibleLines: 5}
	start, end := vp.visibleRowRange(10, func(int) int { return 1 })
	if start != 0 {
		t.Errorf("expected start clamped to 0, got %d", start)
	}
	if end != 5 {
		t.Errorf("expected end 5, got %d", end)
	}
}

func TestVisibleRowRange_TallRowForceVisible(t *testing.T) {
	// Single row taller than visibleLines should still show at least one row
	vp := viewport{topRow: 0, visibleLines: 3}
	start, end := vp.visibleRowRange(1, func(int) int { return 10 })
	if start != 0 {
		t.Errorf("expected start 0, got %d", start)
	}
	if end != 1 {
		t.Errorf("expected end 1 (at-least-one-row), got %d", end)
	}
}

func TestVisibleRowRange_Normal(t *testing.T) {
	vp := viewport{topRow: 0, visibleLines: 3}
	start, end := vp.visibleRowRange(5, func(int) int { return 1 })
	if start != 0 || end != 3 {
		t.Errorf("expected (0,3), got (%d,%d)", start, end)
	}
}

func TestScrollToBottom_ZeroRows(t *testing.T) {
	vp := viewport{visibleLines: 5}
	vp.scrollToBottom(0, func(int) int { return 1 })
	if vp.topRow != 0 {
		t.Errorf("expected topRow clamped to 0, got %d", vp.topRow)
	}
}

func TestScrollToBottom_TallRows(t *testing.T) {
	// Rows with varying heights
	heights := []int{1, 3, 2, 1, 4}
	vp := viewport{visibleLines: 5}
	vp.scrollToBottom(5, func(i int) int { return heights[i] })
	// Walking backward from end: row4=4 > 5, so stop. topRow should be 4.
	// Actually: row4=4 (usedLines=4, <=5), row3=1 (usedLines=5, <=5), row2=2 (usedLines=7, >5)
	// So topRow should be 3
	if vp.topRow != 3 {
		t.Errorf("expected topRow 3, got %d", vp.topRow)
	}
}

func TestEnsureRowVisible_RowAbove(t *testing.T) {
	vp := viewport{topRow: 5, visibleLines: 3}
	vp.ensureRowVisible(2, 10, func(int) int { return 1 })
	if vp.topRow != 2 {
		t.Errorf("expected topRow 2, got %d", vp.topRow)
	}
}

func TestEnsureRowVisible_RowAlreadyVisible(t *testing.T) {
	vp := viewport{topRow: 3, visibleLines: 5}
	vp.ensureRowVisible(5, 10, func(int) int { return 1 })
	if vp.topRow != 3 {
		t.Errorf("expected topRow to stay 3, got %d", vp.topRow)
	}
}

func TestEnsureRowVisible_RowBelow(t *testing.T) {
	vp := viewport{topRow: 0, visibleLines: 3}
	vp.ensureRowVisible(5, 10, func(int) int { return 1 })
	// Should scroll down so row 5 is visible: topRow = 3
	if vp.topRow != 3 {
		t.Errorf("expected topRow 3, got %d", vp.topRow)
	}
}

func TestEnsureColVisible_ColBefore(t *testing.T) {
	vp := viewport{leftCol: 5, visibleCols: 3}
	vp.ensureColVisible(2)
	if vp.leftCol != 2 {
		t.Errorf("expected leftCol 2, got %d", vp.leftCol)
	}
}

func TestEnsureColVisible_ColAfter(t *testing.T) {
	vp := viewport{leftCol: 0, visibleCols: 3}
	vp.ensureColVisible(5)
	if vp.leftCol != 3 {
		t.Errorf("expected leftCol 3, got %d", vp.leftCol)
	}
}

func TestEnsureColVisible_NegativeClamp(t *testing.T) {
	vp := viewport{leftCol: 0, visibleCols: 3}
	vp.ensureColVisible(-1)
	if vp.leftCol != 0 {
		t.Errorf("expected leftCol clamped to 0, got %d", vp.leftCol)
	}
}

func TestScrollToTop(t *testing.T) {
	vp := viewport{topRow: 10}
	vp.scrollToTop()
	if vp.topRow != 0 {
		t.Errorf("expected topRow 0, got %d", vp.topRow)
	}
}
