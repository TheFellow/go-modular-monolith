package gui

import (
	"slices"
	"sync/atomic"
	"time"

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

const RowActionsWidth float32 = 120

// TableTimestamp keeps date columns readable at desktop table widths. Full
// precision remains available in each entity's detail view.
func TableTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04")
}

// TableSort retains a table's active in-memory ordering while more cursor
// pages are appended.
type TableSort struct {
	Column    int
	Direction SortDirection
}

func (s TableSort) Active() bool {
	return s.Column >= 0 && (s.Direction == SortAscending || s.Direction == SortDescending)
}

func ApplyTableSort[T any](rows []T, state TableSort, compare func(column int, left, right T) int) {
	if !state.Active() || compare == nil {
		return
	}
	slices.SortStableFunc(rows, func(left, right T) int {
		result := compare(state.Column, left, right)
		if state.Direction == SortDescending {
			return -result
		}
		return result
	})
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
		button.Alignment = widget.ButtonAlignLeading
		if id.Col == len(columns)-1 && column.Title == "Actions" {
			button.Alignment = widget.ButtonAlignTrailing
		}
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
			// A disabled Fyne button renders low-contrast text. Header cells that
			// are informational rather than sortable remain enabled with no tap
			// callback so their labels use the normal foreground color.
			button.Enable()
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
	actions.Alignment = framework.TextAlignTrailing
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

// NewAutoPagingRowTable creates a row table that asks for more data when its
// final row becomes visible. The callback runs outside Fyne's render stack so
// presenters may safely publish a new table snapshot from it.
func NewAutoPagingRowTable(length func() (rows int, cols int), create func() framework.CanvasObject, update func(widget.TableCellID, framework.CanvasObject), loadMore func()) *widget.Table {
	var requestedRows atomic.Int64
	requestedRows.Store(-1)
	return NewRowTable(length, create, func(id widget.TableCellID, object framework.CanvasObject) {
		update(id, object)
		rows, cols := length()
		if loadMore == nil || rows == 0 || id.Row != rows-1 || id.Col != cols-1 {
			return
		}
		count := int64(rows)
		if requestedRows.Swap(count) == count {
			return
		}
		go loadMore()
	})
}

// NewActionCell returns a native container because widget.Table requires one
// reusable template type for both text and action columns.
func NewActionCell() *framework.Container {
	label := widget.NewLabel("")
	label.Truncation = framework.TextTruncateEllipsis
	actions := NewActionSelect(nil, nil)
	actions.Hide()
	tags := newTableTagCell()
	tags.Hide()
	return container.New(&rowCellLayout{}, label, actions, tags, widget.NewSeparator())
}

func actionCellParts(object framework.CanvasObject) (*widget.Label, *ActionSelect, *tableTagCell) {
	cell := object.(*framework.Container)
	return cell.Objects[0].(*widget.Label), cell.Objects[1].(*ActionSelect), cell.Objects[2].(*tableTagCell)
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
	label, actions, tags := actionCellParts(object)
	actions.Hide()
	tags.Hide()
	label.Show()
	label.TextStyle = framework.TextStyle{Bold: header}
	label.SetText(text)
}

// ShowCellTags renders a bounded pill summary. The dedicated widget keeps its
// fixed renderer objects inside the recycled table cell at every layout pass.
func ShowCellTags(object framework.CanvasObject, value string) {
	label, actions, tags := actionCellParts(object)
	label.Hide()
	actions.Hide()
	tags.SetCSV(value)
	tags.Show()
}

func ShowCellActions(object framework.CanvasObject, rowActions []RowAction) {
	label, actions, tags := actionCellParts(object)
	label.Hide()
	tags.Hide()
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
