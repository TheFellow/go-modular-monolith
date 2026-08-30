package gui

import (
	"fmt"
	"strconv"
	"strings"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const (
	ControlRefresh     = "audit.refresh"
	ControlApplyFilter = "audit.filter.apply"
	ControlScope       = "audit.filter.scope"
	ControlAction      = "audit.filter.action"
	ControlPrincipal   = "audit.filter.principal"
	ControlExpression  = "audit.filter.expression"
	ControlBack        = "audit.detail.back"
	ControlBreadcrumb  = "audit.detail.breadcrumb"
)

type View struct {
	presenter                                    *Presenter
	root, browse, detailPanel                    *framework.Container
	list                                         *widget.Table
	listStack                                    *framework.Container
	empty                                        *framework.Container
	expression                                   *ui.SemanticEntry
	scope                                        *ui.FilterSelect
	apply, refresh                               *ui.SemanticButton
	status, detailTitle, crumbName, detailStatus *widget.Label
	detailFields                                 []*ui.SemanticEntry
	state                                        State
	rendering                                    bool
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(p *Presenter) *View {
	v := &View{presenter: p, state: p.State()}
	bar := ui.NewSingleRowFilterBar(ControlExpression, ControlApplyFilter, `Filter activity (for example: !success && error.contains("permission"))`, v.state.Filter.Expression,
		[]ui.FilterPreset{
			{ID: ControlScope, Placeholder: "Outcome", Options: []ui.FilterOption{{Label: "Any outcome"}, {Label: "Succeeded", Expression: "success"}, {Label: "Failed", Expression: "!success"}}},
			{ID: ControlAction, Placeholder: "Activity", Options: []ui.FilterOption{{Label: "Any activity"}, {Label: "Created", Expression: `action.contains("create")`}, {Label: "Updated", Expression: `action.contains("update")`}, {Label: "Deleted", Expression: `action.contains("delete")`}}},
			{ID: ControlPrincipal, Placeholder: "Actor", Options: []ui.FilterOption{{Label: "Any actor"}, {Label: "Owner", Expression: `principal.contains("owner")`}, {Label: "Manager", Expression: `principal.contains("manager")`}, {Label: "Bartender", Expression: `principal.contains("bartender")`}}},
		}, nil, func(expression string) { v.applyExpression(expression) })
	v.expression, v.apply, v.scope = bar.Expression, bar.Apply, bar.Presets[0]

	columns := []string{"Started", "Duration", "Action", "Resource", "Actor", "Outcome", "Error", "Actions"}
	v.list = ui.NewAutoPagingRowTable(func() (int, int) { return len(v.state.Rows), len(columns) }, func() framework.CanvasObject {
		return ui.NewActionCell()
	}, func(id widget.TableCellID, object framework.CanvasObject) {
		cell := object
		r := v.state.Rows[id.Row]
		outcome := "Failed"
		if r.Entry.Success {
			outcome = "Succeeded"
		}
		values := []string{ui.TableTimestamp(r.Entry.StartedAt), formatDuration(r.Entry.StartedAt, r.Entry.CompletedAt), actionLabel(r.Entry.Action), entityTypeLabel(string(r.Entry.Resource.Type)), string(r.Entry.Principal.ID), outcome, r.Entry.Error}
		if id.Col == len(columns)-1 {
			index := id.Row
			projected := r.Actions[audit.ControlView]
			rowActions := []ui.RowAction{}
			if projected.Visible {
				rowActions = append(rowActions, ui.RowAction{Label: "View", Run: func() { p.Select(index) }})
			}
			ui.ShowCellActions(cell, rowActions)
			return
		}
		ui.ShowCellText(cell, values[id.Col], false)
	}, func() { framework.Do(p.NextPage) })
	v.list.OnSelected = func(id widget.TableCellID) {
		if id.Row >= 0 && id.Col < len(columns)-1 {
			v.list.UnselectAll()
			p.Select(id.Row)
		}
	}
	ui.ConfigureRowTable(v.list, []ui.TableColumn{{Title: "Started", Width: 140, Sortable: true}, {Title: "Duration", Width: 75, Sortable: true}, {Title: "Action", Width: 110, Sortable: true}, {Title: "Resource", Width: 100, Sortable: true}, {Title: "Actor", Width: 80, Sortable: true}, {Title: "Outcome", Width: 100, Sortable: true}, {Title: "Error", Width: 105, Sortable: true}, {Title: "Actions", Width: ui.RowActionsWidth}}, func(column int, direction ui.SortDirection) {
		sortColumns := []int{0, 2, 3, 4, 5, 6, 8}
		p.SortRows(sortColumns[column], direction)
	})
	v.empty = ui.EmptyCollection(ui.IconEmpty, "No audit activity found", "Adjust the filter or return later after application activity occurs.")
	v.listStack = container.NewStack(v.list, v.empty)
	v.refresh = ui.WithIcon(ui.NewButton(ControlRefresh, "Refresh", p.Refresh), ui.IconRefresh)
	v.status = widget.NewLabel("")
	v.browse = ui.StandardListPage(ui.ListPage{Title: "Audit", Filters: bar.Content, CollectionActions: []framework.CanvasObject{v.refresh}, List: v.listStack, Status: v.status}).(*framework.Container)

	v.detailTitle, v.crumbName, v.detailStatus = widget.NewLabel("Audit activity"), widget.NewLabel(""), widget.NewLabel("")
	labels := []string{"ID", "Action", "Entity", "Actor", "Started", "Completed", "Duration", "Success", "Error", "Touched entities"}
	items := make([]framework.CanvasObject, 0, len(labels))
	for i, label := range labels {
		entry := ui.NewEntry(fmt.Sprintf("audit.detail.field.%d", i))
		entry.MultiLine = label == "Error" || label == "Touched entities"
		entry.OnChanged = func(string) { v.restoreDetail() }
		v.detailFields = append(v.detailFields, entry)
		items = append(items, ui.DetailField(label, entry))
	}
	breadcrumb := container.NewHBox(ui.WithIcon(ui.NewButton(ControlBack, "Back", p.Back), ui.IconBack), ui.NewButton(ControlBreadcrumb, "Audit", p.ResetList), widget.NewLabel("›"), v.crumbName)
	v.detailPanel = ui.StandardFormPage(ui.FormPage{TitleLabel: v.detailTitle, Breadcrumb: breadcrumb, Fields: ui.DetailForm(items...), Status: v.detailStatus}).(*framework.Container)
	v.root = container.NewStack(v.browse, v.detailPanel)
	p.Observe(v.render)
	return v
}

func (v *View) Title() string                   { return "Audit" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.presenter.ResetList() }
func (v *View) ExecuteCommand(c ui.Command) bool {
	return c == ui.CommandRefresh && v.state.Mode == Browsing && ui.Trigger(v.refresh)
}

func entityTypeLabel(entityType string) string {
	if index := strings.LastIndex(entityType, "::"); index >= 0 {
		return entityType[index+2:]
	}
	return entityType
}

func actionLabel(action string) string {
	if index := strings.LastIndex(action, "::"); index >= 0 {
		action = action[index+2:]
	}
	return strings.Trim(action, `"`)
}

func (v *View) applyExpression(expression string) {
	v.presenter.ApplyFilter(Filter{Expression: expression, Limit: ui.PageLimit})
}
func (v *View) applyFilter() { v.applyExpression(v.expression.Text) }

func (v *View) restoreDetail() {
	if v.rendering || v.state.Selected == nil {
		return
	}
	v.populateDetail(*v.state.Selected)
}
func (v *View) populateDetail(row Row) {
	v.rendering = true
	defer func() { v.rendering = false }()
	errorText := row.Entry.Error
	if strings.TrimSpace(errorText) == "" {
		errorText = "(none)"
	}
	touches := "(none)"
	if len(row.Touches) > 0 {
		touches = strings.Join(row.Touches, "\n")
	}
	values := []string{row.Entry.ID.String(), row.Entry.Action, row.Entry.Resource.String(), row.Entry.Principal.String(), formatTime(row.Entry.StartedAt), formatTime(row.Entry.CompletedAt), formatDuration(row.Entry.StartedAt, row.Entry.CompletedAt), strconv.FormatBool(row.Entry.Success), errorText, touches}
	for i, value := range values {
		if v.detailFields[i].Text != value {
			v.detailFields[i].SetText(value)
		}
	}
}
func (v *View) render(s State) {
	v.state = s
	v.browse.Hidden = s.Mode != Browsing
	v.detailPanel.Hidden = s.Mode != Viewing
	if s.Selected != nil {
		title := s.Selected.Entry.Action
		if title == "" {
			title = "Audit activity"
		}
		v.detailTitle.SetText(title)
		v.crumbName.SetText(title)
		v.populateDetail(*s.Selected)
	}
	v.empty.Hidden = s.Loading || s.Err != nil || len(s.Rows) != 0
	v.list.Hidden = !v.empty.Hidden
	if s.Loading {
		v.status.SetText("Loading audit activity…")
	} else if s.Err != nil {
		v.status.SetText("Error: " + s.Err.Error())
	} else {
		v.status.SetText(fmt.Sprintf("%d audit entries", len(s.Rows)))
	}
	for _, control := range []interface {
		Enable()
		Disable()
	}{v.scope, v.expression, v.apply, v.refresh} {
		list := s.Actions[audit.ControlList]
		if s.Loading || !list.Visible || !list.Enabled {
			control.Disable()
		} else {
			control.Enable()
		}
	}
	v.list.Refresh()
	v.root.Refresh()
}
