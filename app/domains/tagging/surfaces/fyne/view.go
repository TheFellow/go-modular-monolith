package fyne

import (
	"fmt"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	ui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
	cedar "github.com/cedar-policy/cedar-go"
)

const (
	ControlInspect   = "tags.operation.inspect"
	ControlAdd       = "tags.operation.add"
	ControlRemove    = "tags.operation.remove"
	ControlShowExact = "tags.operation.show-exact"
	ControlShowKey   = "tags.operation.show-key"
	ControlSummary   = "tags.operation.summary"
	ControlSearch    = "tags.entity.search"
	ControlValue     = "tags.value"
	ControlSubmit    = "tags.submit"
	ControlBack      = "tags.back"
)

func typeControl(kind cedar.EntityType) string { return "tags.type." + fmt.Sprint(kind) }
func entityControl(index int) string           { return fmt.Sprintf("tags.entity.%d", index) }

type View struct {
	presenter                                        *Presenter
	root, operations, types, entities, form, results *framework.Container
	entityRows, resultRows                           *framework.Container
	search, value                                    *ui.SemanticEntry
	searchButton, submit, back                       *ui.SemanticButton
	status, formTitle                                *widget.Label
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(p *Presenter) *View {
	v := &View{presenter: p}
	v.operations = container.NewVBox(
		operationButton(ControlInspect, "Inspect entity tags", Inspect, p),
		operationButton(ControlAdd, "Add or replace a tag", Add, p),
		operationButton(ControlRemove, "Remove a tag", Remove, p),
		operationButton(ControlShowExact, "Show exact tag", ShowExact, p),
		operationButton(ControlShowKey, "Show all values for key", ShowKey, p),
		operationButton(ControlSummary, "Tag usage summary", Summary, p),
	)
	v.types = container.NewVBox()
	for _, item := range []struct {
		kind  cedar.EntityType
		label string
	}{{entity.TypeDrink, "Drinks"}, {entity.TypeIngredient, "Ingredients"}, {entity.TypeInventory, "Inventory"}, {entity.TypeMenu, "Menus"}, {entity.TypeOrder, "Orders"}} {
		kind := item.kind
		v.types.Add(ui.NewButton(typeControl(kind), item.label, func() { p.SelectType(kind) }))
	}
	v.search = ui.NewEntry(ControlSearch)
	v.search.SetPlaceHolder("Search active entities by name")
	v.searchButton = ui.NewButton(ControlSearch+".apply", "Search", func() { p.Search(v.search.Text) })
	v.entityRows = container.NewVBox()
	v.entities = container.NewBorder(container.NewBorder(nil, nil, nil, v.searchButton, v.search), nil, nil, nil, container.NewVScroll(v.entityRows))
	v.value = ui.NewEntry(ControlValue)
	v.value.SetPlaceHolder("key or key=value")
	v.submit = ui.NewButton(ControlSubmit, "Submit", func() { p.SetValue(v.value.Text); p.Submit() })
	v.formTitle = widget.NewLabelWithStyle("Tag", framework.TextAlignLeading, framework.TextStyle{Bold: true})
	v.form = container.NewVBox(v.formTitle, v.value, container.NewHBox(layout.NewSpacer(), v.submit))
	v.resultRows = container.NewVBox()
	v.results = container.NewBorder(nil, nil, nil, nil, container.NewVScroll(v.resultRows))
	v.status = widget.NewLabel("")
	v.status.Wrapping = framework.TextWrapWord
	v.back = ui.NewButton(ControlBack, "Back", func() { p.Back() })
	content := container.NewStack(v.operations, v.types, v.entities, v.form, v.results)
	v.root = container.NewBorder(widget.NewLabelWithStyle("Tags", framework.TextAlignLeading, framework.TextStyle{Bold: true}), container.NewVBox(v.status, container.NewHBox(v.back)), nil, nil, content)
	p.Observe(v.render)
	return v
}

func operationButton(id, label string, operation Operation, p *Presenter) *ui.SemanticButton {
	return ui.NewButton(id, label, func() { p.Start(operation) })
}
func (v *View) Title() string                   { return "Tags" }
func (v *View) Content() framework.CanvasObject { return v.root }

// Activate intentionally preserves the current workflow. Unlike list-backed
// workspaces, the tags landing page has no data to refresh on re-entry.
func (v *View) Activate() {}

func (v *View) ExecuteCommand(command ui.Command) bool {
	state := v.presenter.State()
	switch command {
	case ui.CommandSave:
		return state.Mode == EnteringValue && ui.Trigger(v.submit)
	case ui.CommandCancel:
		return state.Mode != Browsing && ui.Trigger(v.back)
	case ui.CommandRefresh, ui.CommandNew:
		return false
	}
	return false
}

func (v *View) render(state State) {
	v.operations.Hidden = state.Mode != Browsing
	v.types.Hidden = state.Mode != PickingType
	v.entities.Hidden = state.Mode != PickingEntity
	v.form.Hidden = state.Mode != EnteringValue
	v.results.Hidden = state.Mode != Results
	v.operations.Refresh()
	v.types.Refresh()
	v.entities.Refresh()
	v.form.Refresh()
	v.results.Refresh()
	v.back.Hidden = state.Mode == Browsing
	if state.Mode == PickingEntity {
		if v.search.Text != state.Query {
			v.search.SetText(state.Query)
		}
		v.entityRows.RemoveAll()
		for i, item := range state.Visible {
			v.entityRows.Add(ui.NewButton(entityControl(i), item.Name+" — "+item.Detail, func() { v.presenter.SelectEntity(i) }))
		}
		v.entityRows.Refresh()
	}
	if state.Mode == EnteringValue {
		if state.Operation == Remove || state.Operation == ShowKey {
			v.value.SetPlaceHolder("key")
		} else {
			v.value.SetPlaceHolder("key or key=value")
		}
		if v.value.Text != state.Value {
			v.value.SetText(state.Value)
		}
		v.formTitle.SetText(operationLabel(state.Operation))
	}
	if state.Mode == Results {
		v.renderResults(state)
	}
	message := ""
	if state.Mode == Loading {
		message = "Loading…"
	}
	if state.Submitting {
		message = "Saving…"
	}
	if state.Err != nil {
		message = "Error: " + state.Err.Error()
	}
	v.status.SetText(message)
	if state.Submitting {
		v.submit.Disable()
		v.back.Disable()
	} else {
		v.submit.Enable()
		v.back.Enable()
	}
}

func (v *View) renderResults(state State) {
	v.resultRows.RemoveAll()
	if state.Err != nil {
		v.resultRows.Add(widget.NewLabel("The operation did not complete."))
		return
	}
	switch state.Operation {
	case Inspect, Add, Remove:
		result := "inspected"
		if state.Operation != Inspect {
			result = "unchanged"
			if state.Result.Changed {
				result = "changed"
			}
		}
		tags := state.Result.Tags.Canonical().String()
		if tags == "" {
			tags = "(none)"
		}
		v.resultRows.Add(row("ENTITY", "TAGS", "RESULT"))
		target := state.Result.TargetName
		if !state.Result.Target.IsZero() {
			target += " (" + string(state.Result.Target.ID) + ")"
		}
		v.resultRows.Add(row(target, tags, result))
	case ShowExact, ShowKey:
		v.resultRows.Add(row("ENTITY TYPE", "ENTITY ID", "TAG"))
		for _, r := range state.Result.References {
			v.resultRows.Add(row(r.EntityType, r.EntityID, r.Tag))
		}
	case Summary:
		v.resultRows.Add(row("TAG", "TOTAL", "DRINKS", "INGREDIENTS", "INVENTORY", "MENUS", "ORDERS"))
		for _, r := range state.Result.Summaries {
			v.resultRows.Add(row(r.Tag, fmt.Sprint(r.Total), fmt.Sprint(r.Drinks), fmt.Sprint(r.Ingredients), fmt.Sprint(r.Inventory), fmt.Sprint(r.Menus), fmt.Sprint(r.Orders)))
		}
	}
	if len(v.resultRows.Objects) == 1 && (state.Operation == ShowExact || state.Operation == ShowKey || state.Operation == Summary) {
		v.resultRows.Add(widget.NewLabel("No matching active tag usage."))
	}
	v.resultRows.Refresh()
}
func row(values ...string) framework.CanvasObject {
	objects := make([]framework.CanvasObject, 0, len(values))
	for _, value := range values {
		label := widget.NewLabel(value)
		label.Wrapping = framework.TextWrapWord
		objects = append(objects, label)
	}
	return container.NewGridWithColumns(len(objects), objects...)
}
func operationLabel(o Operation) string {
	switch o {
	case Inspect:
		return "Inspect entity tags"
	case Add:
		return "Add or replace a tag"
	case Remove:
		return "Remove a tag"
	case ShowExact:
		return "Show exact tag"
	case ShowKey:
		return "Show all values for key"
	case Summary:
		return "Tag usage summary"
	}
	return "Tags"
}
