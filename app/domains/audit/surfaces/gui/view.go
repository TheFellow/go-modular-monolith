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

var scopeLabels = []string{"All activity", "Entity history", "Actor activity"}

type semanticSelect struct {
	widget.Select
	id string
}

func newSelect(id string, options []string) *semanticSelect {
	selectWidget := &semanticSelect{id: id}
	selectWidget.Options = options
	selectWidget.ExtendBaseWidget(selectWidget)
	return selectWidget
}
func (s *semanticSelect) SemanticID() string { return s.id }

type semanticSelectEntry struct {
	*widget.SelectEntry
	id string
}

func newSelectEntry(id string, options []string) *semanticSelectEntry {
	return &semanticSelectEntry{SelectEntry: widget.NewSelectEntry(options), id: id}
}
func (s *semanticSelectEntry) SemanticID() string { return s.id }

type View struct {
	presenter *Presenter
	root      *framework.Container
	rows      *framework.Container
	detail    *widget.Label
	status    *widget.Label

	scope                          *semanticSelect
	entity, principal, action      *semanticSelectEntry
	from, to, expression, limit    *ui.SemanticEntry
	apply, refresh, previous, next *ui.SemanticButton
	rowButtons                     map[string]*ui.SemanticButton
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(presenter *Presenter) *View {
	v := &View{presenter: presenter, rowButtons: make(map[string]*ui.SemanticButton)}
	v.scope = newSelect(ControlScope, scopeLabels)
	v.scope.SetSelected(scopeLabels[0])
	v.entity = newSelectEntry(ControlEntity, nil)
	v.entity.SetPlaceHolder(`Entity UID (Type::id)`)
	v.principal = newSelectEntry(ControlPrincipal, []string{"owner", "manager", "sommelier", "bartender", "anonymous"})
	v.principal.SetPlaceHolder("Actor or principal UID")
	v.action = newSelectEntry(ControlAction, nil)
	v.action.SetPlaceHolder(`Action UID (Type::Action::id)`)
	v.from = ui.NewEntry(ControlFrom)
	v.from.SetPlaceHolder("From (RFC3339 or YYYY-MM-DD)")
	v.to = ui.NewEntry(ControlTo)
	v.to.SetPlaceHolder("To (RFC3339 or YYYY-MM-DD)")
	v.expression = ui.NewEntry(ControlExpression)
	v.expression.SetPlaceHolder("Expression filter")
	v.limit = ui.NewEntry(ControlLimit)
	v.limit.SetText(strconv.Itoa(presenter.State().Filter.Limit))
	v.apply = ui.NewButton(ControlApplyFilter, "Apply", v.applyFilter)
	v.refresh = ui.NewButton(ControlRefresh, "Refresh", presenter.Refresh)

	filters := widget.NewForm(
		widget.NewFormItem("Scope", v.scope),
		widget.NewFormItem("Entity", v.entity),
		widget.NewFormItem("Principal", v.principal),
		widget.NewFormItem("Action", v.action),
		widget.NewFormItem("From", v.from),
		widget.NewFormItem("To", v.to),
		widget.NewFormItem("Expression", v.expression),
		widget.NewFormItem("Page size", v.limit),
	)
	filterPanel := widget.NewCard("Filters", "", container.NewVBox(filters, container.NewHBox(layout.NewSpacer(), v.refresh, v.apply)))

	v.rows = container.NewVBox()
	v.detail = widget.NewLabel("Select an audit entry")
	v.detail.Wrapping = framework.TextWrapWord
	v.status = widget.NewLabel("")
	v.previous = ui.NewButton(ControlPrevious, "Previous", presenter.PreviousPage)
	v.next = ui.NewButton(ControlNext, "Next", presenter.NextPage)
	paging := container.NewHBox(v.previous, v.next, layout.NewSpacer())
	browse := ui.ListDetail(container.NewVScroll(v.rows), container.NewVScroll(v.detail), .42)
	v.root = container.NewBorder(filterPanel, container.NewVBox(v.status, paging), nil, nil, browse)
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
	v.presenter.ApplyFilter(Filter{
		Scope: scopeFromLabel(v.scope.Selected), Entity: v.entity.Text,
		Principal: v.principal.Text, Action: v.action.Text, From: v.from.Text,
		To: v.to.Text, Expression: v.expression.Text, Limit: limit,
	})
}

func scopeFromLabel(label string) Scope {
	switch label {
	case scopeLabels[1]:
		return EntityHistory
	case scopeLabels[2]:
		return ActorActivity
	default:
		return AllActivity
	}
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
	}{v.scope, v.entity, v.principal, v.action, v.from, v.to, v.expression, v.limit, v.apply, v.refresh}
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
