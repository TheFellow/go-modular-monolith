package gui

import (
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func TestRowTableHidesNativeCellSeparators(t *testing.T) {
	table := NewRowTable(func() (int, int) { return 1, 1 }, func() framework.CanvasObject {
		return NewActionCell()
	}, func(widget.TableCellID, framework.CanvasObject) {})
	if !table.HideSeparators {
		t.Fatal("row table should hide native horizontal and vertical cell separators")
	}
	cell := NewActionCell()
	if len(cell.Objects) != 3 {
		t.Fatalf("row cell objects = %d, want label, actions, and horizontal separator", len(cell.Objects))
	}
	if _, ok := cell.Objects[2].(*widget.Separator); !ok {
		t.Fatalf("row cell trailing object = %T, want *widget.Separator", cell.Objects[2])
	}
}

func TestActionCellRunsSelectedActionAndClearsSelection(t *testing.T) {
	startTestApp(t)
	cell := NewActionCell()
	calls := 0
	ShowCellActions(cell, []RowAction{{Label: "View", Run: func() { calls++ }}})

	selector := ActionSelector(cell)
	selector.SetSelected("View")
	selector.SetSelected("View")
	if calls != 2 || selector.Selected != "" {
		t.Fatalf("calls=%d selected=%q", calls, selector.Selected)
	}
}

func TestActionCellRebindDoesNotRetainRecycledRowCallback(t *testing.T) {
	startTestApp(t)
	cell := NewActionCell()
	first, second := 0, 0
	ShowCellActions(cell, []RowAction{{Label: "View", Run: func() { first++ }}})
	ShowCellActions(cell, []RowAction{{Label: "View", Run: func() { second++ }}})

	ActionSelector(cell).SetSelected("View")
	if first != 0 || second != 1 {
		t.Fatalf("recycled callback targeted wrong row: first=%d second=%d", first, second)
	}
}
