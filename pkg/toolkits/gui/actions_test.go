package gui

import "testing"

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
