package row

import "testing"

func TestPinPositionConstants(t *testing.T) {
	if PinNone == PinTop || PinNone == PinBottom || PinTop == PinBottom {
		t.Fatal("PinPosition constants must be distinct")
	}
}

func TestRowNodeZeroValue(t *testing.T) {
	var rn RowNode[string]
	if rn.IsGroup {
		t.Error("zero value IsGroup should be false")
	}
	if rn.Expanded {
		t.Error("zero value Expanded should be false")
	}
	if rn.GroupLevel != 0 {
		t.Error("zero value GroupLevel should be 0")
	}
	if rn.Children != nil {
		t.Error("zero value Children should be nil")
	}
	if rn.Parent != nil {
		t.Error("zero value Parent should be nil")
	}
}

func TestRowNodeTreeStructure(t *testing.T) {
	parent := &RowNode[string]{
		ID:      "parent",
		IsGroup: true,
	}

	child1 := &RowNode[string]{ID: "c1", Data: "child1", Parent: parent}
	child2 := &RowNode[string]{ID: "c2", Data: "child2", Parent: parent}
	parent.Children = []*RowNode[string]{child1, child2}

	if len(parent.Children) != 2 {
		t.Fatalf("parent should have 2 children, got %d", len(parent.Children))
	}
	if child1.Parent != parent {
		t.Error("child1 Parent should point to parent")
	}
	if child2.Parent != parent {
		t.Error("child2 Parent should point to parent")
	}
	if parent.Children[0].Data != "child1" {
		t.Error("first child data mismatch")
	}
	if parent.Children[1].Data != "child2" {
		t.Error("second child data mismatch")
	}
}
