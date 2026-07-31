package gui

import (
	"fmt"
	"strconv"
	"strings"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const (
	ControlRefresh     = "audit.refresh"
	ControlApplyFilter = "audit.filter.apply"
	ControlScope       = "audit.filter.scope"
	ControlEntity      = "audit.filter.entity"
	ControlPrincipal   = "audit.filter.principal"
	ControlAction      = "audit.filter.action"
	ControlFrom        = "audit.filter.from"
	ControlTo          = "audit.filter.to"
	ControlExpression  = "audit.filter.expression"
	ControlLimit       = "audit.filter.limit"
	ControlPrevious    = "audit.page.previous"
	ControlNext        = "audit.page.next"
)

func rowControl(id string) string { return "audit.row." + id }

type View struct {
	presenter *Presenter
	root      *framework.Container
	rows      *framework.Container
	detail    *widget.Label
	status    *widget.Label

	scope                          *ui.FilterSelect
	expression, limit              *ui.SemanticEntry
	apply, refresh, previous, next *ui.SemanticButton
	rowButtons                     map[string]*ui.SemanticButton
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(presenter *Presenter) *View {
	v := &View{presenter: presenter, rowButtons: make(map[string]*ui.SemanticButton)}
	v.limit = ui.NewEntry(ControlLimit)
	v.limit.SetText(strconv.Itoa(presenter.State().Filter.Limit))
	v.refresh = ui.NewButton(ControlRefresh, "Refresh", presenter.Refresh)
	bar := ui.NewFilterBar(ControlExpression, ControlApplyFilter, `Filter activity (for example: !success && error.contains("permission"))`, presenter.State().Filter.Expression,
		[]ui.FilterPreset{{ID: ControlScope, Placeholder: "Outcome", Options: []ui.FilterOption{{Label: "Any outcome"}, {Label: "Succeeded", Expression: "success"}, {Label: "Failed", Expression: "!success"}}}},
		[]ui.FilterPreset{
			{ID: ControlAction, Placeholder: "Activity", Options: []ui.FilterOption{{Label: "Any activity"}, {Label: "Created", Expression: `action.contains("create")`}, {Label: "Updated", Expression: `action.contains("update")`}, {Label: "Deleted", Expression: `action.contains("delete")`}}},
			{ID: ControlPrincipal, Placeholder: "Actor", Options: []ui.FilterOption{{Label: "Any actor"}, {Label: "Owner", Expression: `principal.contains("owner")`}, {Label: "Manager", Expression: `principal.contains("manager")`}, {Label: "Bartender", Expression: `principal.contains("bartender")`}}},
		}, container.NewBorder(nil, nil, widget.NewLabel("Page size"), nil, v.limit), func(expression string) {
			limit, err := strconv.Atoi(strings.TrimSpace(v.limit.Text))
			if err != nil {
				limit = -1
			}
			presenter.ApplyFilter(Filter{Expression: expression, Limit: limit})
		})
	v.expression, v.apply = bar.Expression, bar.Apply
	v.scope = bar.Presets[0]

	v.rows = container.NewVBox()
	v.detail = widget.NewLabel("Select an audit entry")
	v.detail.Wrapping = framework.TextWrapWord
	v.status = widget.NewLabel("")
	v.previous = ui.NewButton(ControlPrevious, "Previous", presenter.PreviousPage)
	v.next = ui.NewButton(ControlNext, "Next", presenter.NextPage)
	paging := container.NewHBox(v.previous, v.next, layout.NewSpacer())
	v.root = ui.StandardListPage(ui.ListPage{
		Title: "Audit", Subtitle: "Review application activity and inspect a selected event.", Filters: bar.Content,
		CollectionActions: []framework.CanvasObject{v.refresh},
		List:              container.NewVScroll(v.rows), Detail: container.NewVScroll(v.detail), Status: v.status,
		Paging: paging, ListRatio: .42,
	}).(*framework.Container)
	presenter.Observe(v.render)
	return v
}

func (v *View) Title() string                   { return "Audit" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.presenter.Refresh() }
func (v *View) ExecuteCommand(command ui.Command) bool {
	if command != ui.CommandRefresh {
		return false
	}
	return ui.Trigger(v.refresh)
}

func (v *View) applyFilter() {
	limit, err := strconv.Atoi(strings.TrimSpace(v.limit.Text))
	if err != nil {
		limit = -1
	}
	v.presenter.ApplyFilter(Filter{Expression: v.expression.Text, Limit: limit})
}

func (v *View) render(state State) {
	v.rebuildRows(state)
	if state.Selected == nil {
		v.detail.SetText("Select an audit entry")
	} else {
		v.detail.SetText(detailText(*state.Selected))
	}
	switch {
	case state.Loading:
		v.status.SetText("Loading…")
	case state.Err != nil:
		v.status.SetText("Error: " + state.Err.Error())
	case len(state.Rows) == 0:
		v.status.SetText("No audit entries")
	default:
		v.status.SetText(fmt.Sprintf("Page %d · %d audit entries", len(state.History)+1, len(state.Rows)))
	}
	v.setFilterEnabled(!state.Loading)
	if state.Loading || len(state.History) == 0 {
		v.previous.Disable()
	} else {
		v.previous.Enable()
	}
	if state.Loading || state.Next == "" {
		v.next.Disable()
	} else {
		v.next.Enable()
	}
	v.root.Refresh()
}

func (v *View) rebuildRows(state State) {
	v.rows.RemoveAll()
	v.rowButtons = make(map[string]*ui.SemanticButton, len(state.Rows))
	for index, row := range state.Rows {
		label := summary(row)
		if !row.Entry.Success {
			label += "  [failed]"
		}
		button := ui.NewButton(rowControl(row.Entry.ID.String()), label, func() { v.presenter.Select(index) })
		if state.Loading {
			button.Disable()
		}
		v.rowButtons[row.Entry.ID.String()] = button
		v.rows.Add(button)
	}
}

func (v *View) setFilterEnabled(enabled bool) {
	entries := []interface {
		Enable()
		Disable()
	}{v.scope, v.expression, v.limit, v.apply, v.refresh}
	for _, entry := range entries {
		if enabled {
			entry.Enable()
		} else {
			entry.Disable()
		}
	}
}

func detailText(row Row) string {
	entry := row.Entry
	errorText := entry.Error
	if strings.TrimSpace(errorText) == "" {
		errorText = "(none)"
	}
	touches := "(none)"
	if len(row.Touches) > 0 {
		touches = "- " + strings.Join(row.Touches, "\n- ")
	}
	return strings.Join([]string{
		"Audit Entry",
		"ID: " + entry.ID.String(),
		"Action: " + entry.Action,
		"Principal: " + entry.Principal.String(),
		"Resource: " + entry.Resource.String(),
		"Started: " + formatTime(entry.StartedAt),
		"Completed: " + formatTime(entry.CompletedAt),
		"Duration: " + formatDuration(entry.StartedAt, entry.CompletedAt),
		fmt.Sprintf("Success: %t", entry.Success),
		"Error: " + errorText,
		"",
		"Touched Entities",
		touches,
	}, "\n")
}
