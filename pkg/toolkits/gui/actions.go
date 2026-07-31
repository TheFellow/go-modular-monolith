package gui

import (
	"slices"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type SortDirection uint8

const (
	SortAscending SortDirection = iota + 1
	SortDescending
)

type TableColumn struct {
	Title    string
	Width    float32
	Sortable bool
}

// ConfigureRowTable adds Fyne's native sticky header row. Native headers carry
// the column-divider drag affordances, so widths remain attached to the table
// for its entire lifetime (including refresh, filtering and detail/back).
func ConfigureRowTable(table *widget.Table, columns []TableColumn, onSort func(int, SortDirection)) {
	table.ShowHeaderRow = true
	table.CreateHeader = func() framework.CanvasObject { return widget.NewButton("", nil) }
	var sortedColumn = -1
	direction := SortAscending
	table.UpdateHeader = func(id widget.TableCellID, object framework.CanvasObject) {
		button := object.(*widget.Button)
		if id.Row != -1 || id.Col < 0 || id.Col >= len(columns) {
			button.SetText("")
			button.Disable()
			return
		}
		column := columns[id.Col]
		text := column.Title
		if id.Col == sortedColumn {
			if direction == SortAscending {
				text += "  ↑"
			} else {
				text += "  ↓"
			}
		}
		button.SetText(text)
		button.OnTapped = nil
		if !column.Sortable || onSort == nil {
			button.Disable()
			return
		}
		button.Enable()
		col := id.Col
		button.OnTapped = func() {
			if sortedColumn == col {
				if direction == SortAscending {
					direction = SortDescending
				} else {
					direction = SortAscending
				}
			} else {
				sortedColumn, direction = col, SortAscending
			}
			onSort(col, direction)
			table.Refresh()
		}
	}
	for col, column := range columns {
		table.SetColumnWidth(col, column.Width)
	}
}

type RowAction struct {
	Label string
	Run   func()
}

// ActionSelect is a momentary action menu rather than a value-bearing Select.
// It owns Fyne's reset callback so callers cannot accidentally introduce the
// recursive ClearSelected/OnChanged loop that a raw Select permits.
type ActionSelect struct {
	widget.Select
	resetting   bool
	dispatching bool
	onAction    func(string)
}

// NewActionSelect creates a compact, rebindable action selector. Use SetActions
// when a recycled widget begins representing a different row.
func NewActionSelect(options []string, onAction func(string)) *ActionSelect {
	actions := &ActionSelect{}
	actions.PlaceHolder = "Actions"
	actions.OnChanged = actions.changed
	actions.ExtendBaseWidget(actions)
	actions.SetActions(options, onAction)
	return actions
}

func (actions *ActionSelect) changed(selected string) {
	if actions.resetting || actions.dispatching || selected == "" {
		return
	}
	actions.resetting = true
	actions.ClearSelected()
	actions.resetting = false
	actions.dispatching = true
	defer func() { actions.dispatching = false }()
	if actions.onAction != nil {
		actions.onAction(selected)
	}
}

// SetActions atomically replaces both choices and their callback. Resetting is
// deliberately callback-silent, which makes table-cell recycling deterministic.
func (actions *ActionSelect) SetActions(options []string, onAction func(string)) {
	actions.resetting = true
	actions.ClearSelected()
	actions.Options = append(actions.Options[:0], options...)
	actions.onAction = onAction
	actions.resetting = false
	actions.Refresh()
}

// SetSelected accepts only a currently bound action. This shadows the raw
// Select method, which otherwise accepts stale values after a recycled rebind.
func (actions *ActionSelect) SetSelected(selected string) {
	if actions.dispatching || actions.Disabled() || actions.Hidden {
		return
	}
	if selected == "" {
		actions.Select.SetSelected(selected)
		return
	}
	if slices.Contains(actions.Options, selected) {
		actions.Select.SetSelected(selected)
	}
}

// SelectAction is the event-level API for tests and keyboard integrations. It
// uses the same Fyne selection path as pointer interaction.
func (actions *ActionSelect) SelectAction(selected string) {
	actions.SetSelected(selected)
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
	actions := NewActionSelect(nil, nil)
	actions.Hide()
	pills := container.New(&compactPillRowLayout{})
	pills.Hide()
	return container.New(&rowCellLayout{}, label, actions, pills, widget.NewSeparator())
}

func actionCellParts(object framework.CanvasObject) (*widget.Label, *ActionSelect, *framework.Container) {
	cell := object.(*framework.Container)
	return cell.Objects[0].(*widget.Label), cell.Objects[1].(*ActionSelect), cell.Objects[2].(*framework.Container)
}

type rowCellLayout struct{}

func (*rowCellLayout) Layout(objects []framework.CanvasObject, size framework.Size) {
	for _, object := range objects[:3] {
		object.Move(framework.NewPos(0, 0))
		object.Resize(size)
	}
	separator := objects[3]
	height := separator.MinSize().Height
	separator.Move(framework.NewPos(0, size.Height-height))
	separator.Resize(framework.NewSize(size.Width, height))
}

func (*rowCellLayout) MinSize(objects []framework.CanvasObject) framework.Size {
	return objects[0].MinSize().Max(objects[1].MinSize()).Max(objects[2].MinSize())
}

func ShowCellText(object framework.CanvasObject, text string, header bool) {
	label, actions, pills := actionCellParts(object)
	actions.Hide()
	pills.Hide()
	label.Show()
	label.TextStyle = framework.TextStyle{Bold: header}
	label.SetText(text)
}

// ShowCellTags renders a canonical CSV tag collection as compact pills. The
// container is reused because widget.Table recycles cells while scrolling.
func ShowCellTags(object framework.CanvasObject, value string) {
	label, actions, pills := actionCellParts(object)
	label.Hide()
	actions.Hide()
	pills.RemoveAll()
	for _, tag := range parseTagCSV(value) {
		pills.Add(compactTagPill(tag))
	}
	if len(pills.Objects) == 0 {
		pills.Add(widget.NewLabel(""))
	}
	pills.Show()
	pills.Refresh()
}

func ShowCellActions(object framework.CanvasObject, rowActions []RowAction) {
	label, actions, pills := actionCellParts(object)
	label.Hide()
	pills.Hide()
	actions.Show()
	byLabel := make(map[string]func(), len(rowActions))
	options := make([]string, 0, len(rowActions))
	for _, action := range rowActions {
		options = append(options, action.Label)
		byLabel[action.Label] = action.Run
	}
	actions.SetActions(options, func(selected string) {
		run := byLabel[selected]
		if run != nil {
			run()
		}
	})
}

// ActionSelector exposes the selector for focused interaction tests.
func ActionSelector(object framework.CanvasObject) *ActionSelect {
	_, actions, _ := actionCellParts(object)
	return actions
}

// CellTagPills exposes the recycled pill container for focused tests.
func CellTagPills(object framework.CanvasObject) *framework.Container {
	_, _, pills := actionCellParts(object)
	return pills
}
