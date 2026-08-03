package gui

import (
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestRowTableHidesNativeCellSeparators(t *testing.T) { //nolint:paralleltest // Fyne widget state is process-global.
	table := NewRowTable(func() (int, int) { return 1, 1 }, func() framework.CanvasObject {
		return NewActionCell()
	}, func(widget.TableCellID, framework.CanvasObject) {})
	testutil.ErrorIf(t, !table.HideSeparators, "%v", "row table should hide native horizontal and vertical cell separators")
	cell := NewActionCell()
	testutil.ErrorIf(t, len(cell.Objects) != 4, "row cell objects = %d, want label, actions, pills, and horizontal separator", len(cell.Objects))
	{
		_, ok := cell.Objects[3].(*widget.Separator)
		testutil.ErrorIf(t, !ok, "row cell trailing object = %T, want *widget.Separator", cell.Objects[3])
	}
}

func TestActionCellRecyclesSafelyAcrossTextTagsAndActions(t *testing.T) { //nolint:paralleltest // Fyne widget state is process-global.
	startTestApp(t)
	cell := NewActionCell()
	ShowCellTags(cell, `featured,"region=west, coast"`)
	testutil.ErrorIf(t, CellTagPills(cell).Hidden || len(CellTagPills(cell).Objects) != 2, "tag mode hidden=%v pills=%d", CellTagPills(cell).Hidden, len(CellTagPills(cell).Objects))
	ShowCellText(cell, "Name", false)
	testutil.ErrorIf(t, !CellTagPills(cell).Hidden || !cell.Objects[0].Visible(), "%v", "text mode retained recycled pill content")
	ShowCellActions(cell, []RowAction{{Label: "View"}})
	testutil.ErrorIf(t, !CellTagPills(cell).Hidden || cell.Objects[0].Visible() || ActionSelector(cell).Hidden, "%v", "action mode retained recycled text or pill content")
	ShowCellTags(cell, "new")
	testutil.ErrorIf(t, !ActionSelector(cell).Hidden || len(CellTagPills(cell).Objects) != 1, "%v", "pill mode retained recycled action content")
}

func TestActionCellRunsSelectedActionAndClearsSelection(t *testing.T) { //nolint:paralleltest // Fyne widget state is process-global.
	startTestApp(t)
	cell := NewActionCell()
	calls := 0
	ShowCellActions(cell, []RowAction{{Label: "View", Run: func() { calls++ }}})

	selector := ActionSelector(cell)
	selector.SetSelected("View")
	selector.SetSelected("View")
	testutil.ErrorIf(t, calls != 2 || selector.Selected != "", "calls=%d selected=%q", calls, selector.Selected)
}

func TestActionSelectIgnoresResetEventWithoutRecursing(t *testing.T) { //nolint:paralleltest // Fyne widget state is process-global.
	startTestApp(t)
	calls := 0
	selector := NewActionSelect([]string{"Remove"}, func(selected string) {
		testutil.ErrorIf(t, selected != "Remove", "action = %q", selected)
		calls++
	})

	selector.SetSelected("Remove")
	selector.SetSelected("Remove")
	testutil.ErrorIf(t, calls != 2 || selector.Selected != "", "calls=%d selected=%q", calls, selector.Selected)
}

func TestActionSelectIgnoresEmptyUnknownAndReentrantSelections(t *testing.T) { //nolint:paralleltest // Fyne widget state is process-global.
	startTestApp(t)
	calls := 0
	var selector *ActionSelect
	selector = NewActionSelect([]string{"Remove"}, func(string) {
		calls++
		// An action may synchronously cause a render or binding update. Even if
		// that update selects again, one user gesture remains one invocation.
		selector.SelectAction("Remove")
	})

	selector.SetSelected("")
	selector.SelectAction("Unknown")
	selector.SelectAction("Remove")
	testutil.ErrorIf(t, calls != 1 || selector.Selected != "", "calls=%d selected=%q", calls, selector.Selected)
}

func TestActionSelectRebindIsSilentAndUsesOnlyLatestCallback(t *testing.T) { //nolint:paralleltest // Fyne widget state is process-global.
	startTestApp(t)
	first, second := 0, 0
	selector := NewActionSelect([]string{"View"}, func(string) { first++ })
	selector.SetSelected("View")
	selector.SetActions([]string{"Remove"}, func(string) { second++ })
	selector.SetSelected("View") // stale recycled-row choice is no longer valid.
	selector.SelectAction("Remove")
	testutil.ErrorIf(t, first != 1 || second != 1, "first=%d second=%d", first, second)
}

func TestActionSelectCannotDispatchWhileDisabledOrHidden(t *testing.T) { //nolint:paralleltest // Fyne widget state is process-global.
	startTestApp(t)
	calls := 0
	selector := NewActionSelect([]string{"Remove"}, func(string) { calls++ })
	selector.Disable()
	selector.SetSelected("Remove")
	selector.Enable()
	selector.Hide()
	selector.SetSelected("Remove")
	selector.Show()
	selector.SetSelected("Remove")
	testutil.ErrorIf(t, calls != 1, "action dispatched %d times, want only enabled visible selection", calls)
}

func TestActionSelectRecoversDispatchStateAfterCallbackPanic(t *testing.T) { //nolint:paralleltest // Fyne widget state is process-global.
	startTestApp(t)
	selector := NewActionSelect([]string{"Run"}, func(string) { panic("boom") })
	func() {
		defer func() {
			{
				recovered := recover()
				testutil.ErrorIf(t, recovered != "boom", "recovered %v", recovered)
			}
		}()
		selector.SetSelected("Run")
	}()
	calls := 0
	selector.SetActions([]string{"Run"}, func(string) { calls++ })
	selector.SetSelected("Run")
	testutil.ErrorIf(t, calls != 1, "selector remained dispatch-locked after panic: calls=%d", calls)
}

func TestActionCellRebindDoesNotRetainRecycledRowCallback(t *testing.T) { //nolint:paralleltest // Fyne widget state is process-global.
	startTestApp(t)
	cell := NewActionCell()
	first, second := 0, 0
	ShowCellActions(cell, []RowAction{{Label: "View", Run: func() { first++ }}})
	ShowCellActions(cell, []RowAction{{Label: "View", Run: func() { second++ }}})

	ActionSelector(cell).SetSelected("View")
	testutil.ErrorIf(t, first != 0 || second != 1, "recycled callback targeted wrong row: first=%d second=%d", first, second)
}

func TestConfiguredRowTableUsesResizableNativeHeadersAndTogglesSort(t *testing.T) { //nolint:paralleltest // Fyne widget state is process-global.
	startTestApp(t)
	table := NewRowTable(func() (int, int) { return 1, 2 }, func() framework.CanvasObject { return NewActionCell() }, func(widget.TableCellID, framework.CanvasObject) {})
	var directions []SortDirection
	ConfigureRowTable(table, []TableColumn{{Title: "Name", Width: 180, Sortable: true}, {Title: "Actions", Width: 120}}, func(_ int, direction SortDirection) {
		directions = append(directions, direction)
	})
	testutil.ErrorIf(t, !table.ShowHeaderRow || table.CreateHeader == nil || table.UpdateHeader == nil, "%v", "table does not use the native resizable header row")
	header := table.CreateHeader()
	table.UpdateHeader(widget.TableCellID{Row: -1, Col: 0}, header)
	button := header.(*widget.Button)
	button.OnTapped()
	button.OnTapped()
	testutil.ErrorIf(t, len(directions) != 2 || directions[0] != SortAscending || directions[1] != SortDescending, "sort directions = %v", directions)
	table.UpdateHeader(widget.TableCellID{Row: -1, Col: 0}, header)
	testutil.ErrorIf(t, button.Text != "Name  ↓", "sorted header = %q", button.Text)
	table.UpdateHeader(widget.TableCellID{Row: -1, Col: 1}, header)
	testutil.ErrorIf(t, !button.Disabled(), "%v", "unsupported column should remain visibly non-sortable")
}
