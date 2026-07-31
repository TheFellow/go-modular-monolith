package gui

import (
	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type RowAction struct {
	Label string
	Run   func()
}

// NewRowTable creates a table styled as aligned rows rather than a boxed grid.
// Fyne's native separator setting is all-or-nothing, so the table dividers are
// hidden and NewActionCell supplies a subtle horizontal rule for each row.
func NewRowTable(length func() (rows int, cols int), create func() framework.CanvasObject, update func(widget.TableCellID, framework.CanvasObject)) *widget.Table {
	table := widget.NewTable(length, create, update)
	table.HideSeparators = true
	return table
}

// NewActionCell returns a native container because widget.Table requires one
// reusable template type for both text and action columns.
func NewActionCell() *framework.Container {
	label := widget.NewLabel("")
	label.Truncation = framework.TextTruncateEllipsis
	actions := widget.NewSelect(nil, nil)
	actions.PlaceHolder = "Actions"
	actions.Hide()
	return container.New(&rowCellLayout{}, label, actions, widget.NewSeparator())
}

func actionCellParts(object framework.CanvasObject) (*widget.Label, *widget.Select) {
	cell := object.(*framework.Container)
	return cell.Objects[0].(*widget.Label), cell.Objects[1].(*widget.Select)
}

type rowCellLayout struct{}

func (*rowCellLayout) Layout(objects []framework.CanvasObject, size framework.Size) {
	for _, object := range objects[:2] {
		object.Move(framework.NewPos(0, 0))
		object.Resize(size)
	}
	separator := objects[2]
	height := separator.MinSize().Height
	separator.Move(framework.NewPos(0, size.Height-height))
	separator.Resize(framework.NewSize(size.Width, height))
}

func (*rowCellLayout) MinSize(objects []framework.CanvasObject) framework.Size {
	return objects[0].MinSize().Max(objects[1].MinSize())
}

func ShowCellText(object framework.CanvasObject, text string, header bool) {
	label, actions := actionCellParts(object)
	actions.Hide()
	label.Show()
	label.TextStyle = framework.TextStyle{Bold: header}
	label.SetText(text)
}

func ShowCellActions(object framework.CanvasObject, rowActions []RowAction) {
	label, actions := actionCellParts(object)
	label.Hide()
	actions.Show()
	actions.Options = actions.Options[:0]
	byLabel := make(map[string]func(), len(rowActions))
	for _, action := range rowActions {
		actions.Options = append(actions.Options, action.Label)
		byLabel[action.Label] = action.Run
	}
	// Clear before rebinding so an old OnChanged callback cannot fire for the
	// row whose cell was just recycled.
	actions.OnChanged = nil
	actions.ClearSelected()
	clearing := false
	actions.OnChanged = func(selected string) {
		if clearing || selected == "" {
			return
		}
		run := byLabel[selected]
		clearing = true
		actions.ClearSelected()
		clearing = false
		if run != nil {
			run()
		}
	}
	actions.Refresh()
}

// ActionSelector exposes the selector for focused interaction tests.
func ActionSelector(object framework.CanvasObject) *widget.Select {
	_, actions := actionCellParts(object)
	return actions
}
