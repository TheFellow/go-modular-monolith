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
	if len(cell.Objects) != 4 {
		t.Fatalf("row cell objects = %d, want label, actions, pills, and horizontal separator", len(cell.Objects))
	}
	if _, ok := cell.Objects[3].(*widget.Separator); !ok {
		t.Fatalf("row cell trailing object = %T, want *widget.Separator", cell.Objects[3])
	}
}

func TestActionCellRecyclesSafelyAcrossTextTagsAndActions(t *testing.T) {
	startTestApp(t)
	cell := NewActionCell()
	ShowCellTags(cell, `featured,"region=west, coast"`)
	if CellTagPills(cell).Hidden || len(CellTagPills(cell).Objects) != 2 {
		t.Fatalf("tag mode hidden=%v pills=%d", CellTagPills(cell).Hidden, len(CellTagPills(cell).Objects))
	}
	ShowCellText(cell, "Name", false)
	if !CellTagPills(cell).Hidden || !cell.Objects[0].Visible() {
		t.Fatal("text mode retained recycled pill content")
	}
	ShowCellActions(cell, []RowAction{{Label: "View"}})
	if !CellTagPills(cell).Hidden || cell.Objects[0].Visible() || ActionSelector(cell).Hidden {
		t.Fatal("action mode retained recycled text or pill content")
	}
	ShowCellTags(cell, "new")
	if !ActionSelector(cell).Hidden || len(CellTagPills(cell).Objects) != 1 {
		t.Fatal("pill mode retained recycled action content")
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

func TestConfiguredRowTableUsesResizableNativeHeadersAndTogglesSort(t *testing.T) {
	startTestApp(t)
	table := NewRowTable(func() (int, int) { return 1, 2 }, func() framework.CanvasObject { return NewActionCell() }, func(widget.TableCellID, framework.CanvasObject) {})
	var directions []SortDirection
	ConfigureRowTable(table, []TableColumn{{Title: "Name", Width: 180, Sortable: true}, {Title: "Actions", Width: 120}}, func(_ int, direction SortDirection) {
		directions = append(directions, direction)
	})
	if !table.ShowHeaderRow || table.CreateHeader == nil || table.UpdateHeader == nil {
		t.Fatal("table does not use the native resizable header row")
	}
	header := table.CreateHeader()
	table.UpdateHeader(widget.TableCellID{Row: -1, Col: 0}, header)
	button := header.(*widget.Button)
	button.OnTapped()
	button.OnTapped()
	if len(directions) != 2 || directions[0] != SortAscending || directions[1] != SortDescending {
		t.Fatalf("sort directions = %v", directions)
	}
	table.UpdateHeader(widget.TableCellID{Row: -1, Col: 0}, header)
	if button.Text != "Name  ↓" {
		t.Fatalf("sorted header = %q", button.Text)
	}
	table.UpdateHeader(widget.TableCellID{Row: -1, Col: 1}, header)
	if !button.Disabled() {
		t.Fatal("unsupported column should remain visibly non-sortable")
	}
}
