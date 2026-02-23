package grid

import "testing"

func TestSelectionRegion_RowRange_NoSwap(t *testing.T) {
	r := SelectionRegion{
		Anchor: CellPosition{Row: 2, Col: 0},
		Cursor: CellPosition{Row: 5, Col: 0},
	}
	lo, hi := r.RowRange()
	if lo != 2 || hi != 5 {
		t.Errorf("expected (2,5), got (%d,%d)", lo, hi)
	}
}

func TestSelectionRegion_RowRange_Swap(t *testing.T) {
	r := SelectionRegion{
		Anchor: CellPosition{Row: 8, Col: 0},
		Cursor: CellPosition{Row: 3, Col: 0},
	}
	lo, hi := r.RowRange()
	if lo != 3 || hi != 8 {
		t.Errorf("expected (3,8), got (%d,%d)", lo, hi)
	}
}

func TestSelectionRegion_ColRange_NoSwap(t *testing.T) {
	r := SelectionRegion{
		Anchor: CellPosition{Row: 0, Col: 1},
		Cursor: CellPosition{Row: 0, Col: 4},
	}
	lo, hi := r.ColRange()
	if lo != 1 || hi != 4 {
		t.Errorf("expected (1,4), got (%d,%d)", lo, hi)
	}
}

func TestSelectionRegion_ColRange_Swap(t *testing.T) {
	r := SelectionRegion{
		Anchor: CellPosition{Row: 0, Col: 7},
		Cursor: CellPosition{Row: 0, Col: 2},
	}
	lo, hi := r.ColRange()
	if lo != 2 || hi != 7 {
		t.Errorf("expected (2,7), got (%d,%d)", lo, hi)
	}
}
