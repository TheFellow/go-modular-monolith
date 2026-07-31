package gui

import (
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

// ConfigureActionSelect binds a compact action selector so each action can be
// chosen repeatedly. Clearing a Select emits OnChanged("") in Fyne, therefore
// reset must be guarded to avoid recursively clearing until the stack overflows.
func ConfigureActionSelect(actions *widget.Select, options []string, onAction func(string)) {
	actions.OnChanged = nil
	actions.Options = append(actions.Options[:0], options...)
	actions.ClearSelected()
	clearing := false
	actions.OnChanged = func(selected string) {
		if clearing || selected == "" {
			return
		}
		clearing = true
		actions.ClearSelected()
		clearing = false
		if onAction != nil {
			onAction(selected)
		}
	}
	actions.Refresh()
}

// NewActionSelect creates a compact action selector with re-entrancy-safe
// selection reset behavior.
func NewActionSelect(options []string, onAction func(string)) *widget.Select {
	actions := widget.NewSelect(nil, nil)
	actions.PlaceHolder = "Actions"
	ConfigureActionSelect(actions, options, onAction)
	return actions
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
	pills := container.New(&compactPillRowLayout{})
	pills.Hide()
	return container.New(&rowCellLayout{}, label, actions, pills, widget.NewSeparator())
}

func actionCellParts(object framework.CanvasObject) (*widget.Label, *widget.Select, *framework.Container) {
	cell := object.(*framework.Container)
	return cell.Objects[0].(*widget.Label), cell.Objects[1].(*widget.Select), cell.Objects[2].(*framework.Container)
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
	ConfigureActionSelect(actions, options, func(selected string) {
		run := byLabel[selected]
		if run != nil {
			run()
		}
	})
}

// ActionSelector exposes the selector for focused interaction tests.
func ActionSelector(object framework.CanvasObject) *widget.Select {
	_, actions, _ := actionCellParts(object)
	return actions
}

// CellTagPills exposes the recycled pill container for focused tests.
func CellTagPills(object framework.CanvasObject) *framework.Container {
	_, _, pills := actionCellParts(object)
	return pills
}
