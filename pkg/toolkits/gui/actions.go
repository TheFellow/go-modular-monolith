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

// NewActionCell returns a native container because widget.Table requires one
// reusable template type for both text and action columns.
func NewActionCell() *framework.Container {
	label := widget.NewLabel("")
	label.Truncation = framework.TextTruncateEllipsis
	actions := widget.NewSelect(nil, nil)
	actions.PlaceHolder = "Actions"
	actions.Hide()
	return container.NewStack(label, actions)
}

func actionCellParts(object framework.CanvasObject) (*widget.Label, *widget.Select) {
	cell := object.(*framework.Container)
	return cell.Objects[0].(*widget.Label), cell.Objects[1].(*widget.Select)
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
