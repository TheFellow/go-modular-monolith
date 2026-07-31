package gui

import (
	"fmt"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
	cedar "github.com/cedar-policy/cedar-go"
)

const (
	ControlInspect    = "tags.operation.inspect"
	ControlAdd        = "tags.operation.add"
	ControlRemove     = "tags.operation.remove"
	ControlShowExact  = "tags.operation.show-exact"
	ControlShowKey    = "tags.operation.show-key"
	ControlSummary    = "tags.operation.summary"
	ControlSearch     = "tags.entity.search"
	ControlValue      = "tags.value"
	ControlSubmit     = "tags.submit"
	ControlBack       = "tags.back"
	ControlBreadcrumb = "tags.detail.breadcrumb"
)

func typeControl(kind cedar.EntityType) string { return "tags.type." + fmt.Sprint(kind) }
func entityControl(index int) string           { return fmt.Sprintf("tags.entity.%d", index) }

type View struct {
	presenter                                                         *Presenter
	root, browse, workflow, operations, types, entities, form, detail *framework.Container
	entityRows, resultRows                                            *framework.Container
	list                                                              *widget.Table
	listStack                                                         *framework.Container
	empty                                                             *framework.Container
	search, entitySearch, value                                       *ui.SemanticEntry
	apply, submit, back                                               *ui.SemanticButton
	status, workflowTitle, detailTitle, crumbName                     *widget.Label
	state                                                             State
	tagNaturalWidth                                                   float32
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(p *Presenter) *View {
	v := &View{presenter: p, state: p.State()}
	bar := ui.NewSingleRowFilterBar(ControlSearch+".summary", ControlSearch+".summary.apply", "Filter tags by key or value", "", nil, nil, func(expression string) { p.Search(expression) })
	v.search, v.apply = bar.Expression, bar.Apply
	columns := []string{"Tag", "Total", "Drinks", "Ingredients", "Inventory", "Menus", "Orders", "Actions"}
	v.list = ui.NewRowTable(func() (int, int) { return len(v.state.VisibleSummaries), len(columns) }, func() framework.CanvasObject {
		return ui.NewActionCell()
	}, func(id widget.TableCellID, object framework.CanvasObject) {
		cell := object
		r := v.state.VisibleSummaries[id.Row]
		values := []string{r.Tag, fmt.Sprint(r.Total), fmt.Sprint(r.Drinks), fmt.Sprint(r.Ingredients), fmt.Sprint(r.Inventory), fmt.Sprint(r.Menus), fmt.Sprint(r.Orders)}
		if id.Col == len(columns)-1 {
			index := id.Row
			ui.ShowCellActions(cell, []ui.RowAction{{Label: "View", Run: func() { p.SelectSummary(index) }}})
			return
		}
		if id.Col == 0 {
			ui.ShowCellTags(cell, values[id.Col])
			return
		}
		ui.ShowCellText(cell, values[id.Col], false)
	})
	v.list.OnSelected = func(id widget.TableCellID) {
		if id.Row >= 0 && id.Col < len(columns)-1 {
			v.list.UnselectAll()
			p.SelectSummary(id.Row)
		}
	}
	ui.ConfigureRowTable(v.list, []ui.TableColumn{{Title: "Tag", Width: 260, Sortable: true}, {Title: "Total", Width: 70, Sortable: true}, {Title: "Drinks", Width: 80, Sortable: true}, {Title: "Ingredients", Width: 100, Sortable: true}, {Title: "Inventory", Width: 90, Sortable: true}, {Title: "Menus", Width: 70, Sortable: true}, {Title: "Orders", Width: 70, Sortable: true}, {Title: "Actions", Width: 120}}, p.SortSummaries)
	v.empty = ui.EmptyCollection(ui.IconEmpty, "No active tag usage", "Adjust the filter or tag an active entity to begin discovery.")
	v.listStack = container.NewStack(v.list, v.empty)
	add := ui.WithIcon(ui.NewButton(ControlAdd+".list", "Tag entity", func() { p.Start(Add) }), ui.IconTag)
	remove := ui.WithIcon(ui.NewButton(ControlRemove+".list", "Untag entity", func() { p.Start(Remove) }), ui.IconDelete)
	v.status = widget.NewLabel("")
	v.browse = ui.StandardListPage(ui.ListPage{Title: "Tags", Subtitle: "Discover tag usage across active application entities.", Filters: bar.Content, CollectionActions: []framework.CanvasObject{add}, OtherActions: []framework.CanvasObject{remove}, List: v.listStack, Status: v.status}).(*framework.Container)

	v.detailTitle, v.crumbName = widget.NewLabel("Tag"), widget.NewLabel("")
	v.resultRows = container.NewVBox()
	breadcrumb := container.NewHBox(ui.WithIcon(ui.NewButton(ControlBack, "Back", func() { p.Back() }), ui.IconBack), ui.NewButton(ControlBreadcrumb, "Tags", p.ResetList), widget.NewLabel(">"), v.crumbName)
	detailAdd := ui.WithIcon(ui.NewButton(ControlAdd+".detail", "Tag entity", func() { p.Start(Add) }), ui.IconTag)
	detailRemove := ui.WithIcon(ui.NewButton(ControlRemove+".detail", "Untag entity", func() { p.Start(Remove) }), ui.IconDelete)
	detailHeader := container.NewVBox(breadcrumb, v.detailTitle, container.NewHBox(layout.NewSpacer(), detailAdd, detailRemove))
	v.detail = container.NewBorder(container.NewPadded(detailHeader), nil, nil, nil, container.NewPadded(container.NewVScroll(v.resultRows)))

	v.types = container.NewVBox()
	v.operations = container.NewVBox(
		ui.NewButton(ControlInspect, "Inspect entity tags", func() { p.Start(Inspect) }),
		ui.NewButton(ControlAdd, "Tag entity", func() { p.Start(Add) }),
		ui.NewButton(ControlRemove, "Untag entity", func() { p.Start(Remove) }),
		ui.NewButton(ControlShowExact, "Find exact tag", func() { p.Start(ShowExact) }),
		ui.NewButton(ControlShowKey, "Find tag key", func() { p.Start(ShowKey) }),
		ui.NewButton(ControlSummary, "Tag usage summary", func() { p.Start(Summary) }),
	)
	for _, item := range []struct {
		kind  cedar.EntityType
		label string
	}{{entity.TypeDrink, "Drinks"}, {entity.TypeIngredient, "Ingredients"}, {entity.TypeInventory, "Inventory"}, {entity.TypeMenu, "Menus"}, {entity.TypeOrder, "Orders"}} {
		kind := item.kind
		v.types.Add(ui.NewButton(typeControl(kind), item.label, func() { p.SelectType(kind) }))
	}
	v.entityRows = container.NewVBox()
	v.entitySearch = ui.NewEntry(ControlSearch)
	v.entitySearch.SetPlaceHolder("Search active entities by name")
	entityApply := ui.NewButton(ControlSearch+".apply", "Search", func() { p.Search(v.entitySearch.Text) })
	v.entitySearch.OnSubmitted = func(string) { entityApply.OnTapped() }
	v.entities = container.NewBorder(container.NewBorder(nil, nil, nil, entityApply, v.entitySearch), nil, nil, nil, container.NewVScroll(v.entityRows))
	v.value = ui.NewEntry(ControlValue)
	v.submit = ui.Primary(ui.WithIcon(ui.NewButton(ControlSubmit, "Apply", func() { p.SetValue(v.value.Text); p.Submit() }), ui.IconSave))
	v.form = container.NewVBox(v.value, container.NewHBox(layout.NewSpacer(), v.submit))
	v.workflowTitle = widget.NewLabelWithStyle("Tags", framework.TextAlignLeading, framework.TextStyle{Bold: true})
	v.back = ui.WithIcon(ui.NewButton(ControlBack+".workflow", "Back", func() { p.Back() }), ui.IconBack)
	v.workflow = ui.StandardPage("Tags", "Choose an active entity and apply a canonical tag value.", nil, container.NewVBox(v.workflowTitle, v.operations, v.types, v.entities, v.form), container.NewVBox(v.status, container.NewHBox(v.back))).(*framework.Container)
	v.root = container.NewStack(v.browse, v.detail, v.workflow)
	p.Observe(v.render)
	return v
}

func (v *View) Title() string                   { return "Tags" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.presenter.ResetList() }
func (v *View) HasUnsavedChanges() bool         { return v.state.Mode == EnteringValue }
func (v *View) ExecuteCommand(c ui.Command) bool {
	if c == ui.CommandSave && v.state.Mode == EnteringValue {
		return ui.Trigger(v.submit)
	}
	if c == ui.CommandCancel && v.state.Mode != Results {
		return ui.Trigger(v.back)
	}
	if c == ui.CommandRefresh {
		v.presenter.ResetList()
		return true
	}
	return false
}

func (v *View) render(s State) {
	if len(s.VisibleSummaries) > 0 {
		values := make([]string, len(s.VisibleSummaries))
		for i, summary := range s.VisibleSummaries {
			values[i] = summary.Tag
		}
		if width := ui.TagPillColumnWidth(values, 260); width > v.tagNaturalWidth {
			v.list.SetColumnWidth(0, width)
			v.tagNaturalWidth = width
		}
	}
	v.state = s
	list := s.Mode == Results && s.Operation == Summary
	detail := s.Mode == Results && s.Operation != Summary
	v.browse.Hidden, v.detail.Hidden, v.workflow.Hidden = !list, !detail, list || detail
	if v.search.Text != s.Query && list {
		v.search.SetText(s.Query)
	}
	v.empty.Hidden = s.Err != nil || len(s.VisibleSummaries) != 0
	v.list.Hidden = !v.empty.Hidden
	v.list.Refresh()
	if list {
		v.status.SetText(fmt.Sprintf("%d active tags", len(s.VisibleSummaries)))
	}
	v.types.Hidden = s.Mode != PickingType
	v.operations.Hidden = s.Mode != Browsing
	v.entities.Hidden = s.Mode != PickingEntity
	v.form.Hidden = s.Mode != EnteringValue
	if s.Mode == PickingEntity {
		if v.entitySearch.Text != s.Query {
			v.entitySearch.SetText(s.Query)
		}
		v.entityRows.RemoveAll()
		for i, item := range s.Visible {
			i := i
			v.entityRows.Add(ui.NewButton(entityControl(i), item.Name+" — "+item.Detail, func() { p := v.presenter; p.SelectEntity(i) }))
		}
		v.entityRows.Refresh()
	}
	if s.Mode == EnteringValue {
		v.workflowTitle.SetText(operationLabel(s.Operation))
		if v.value.Text != s.Value {
			v.value.SetText(s.Value)
		}
		if s.Operation == Remove || s.Operation == ShowKey {
			v.value.SetPlaceHolder("key")
		} else {
			v.value.SetPlaceHolder("key or key=value")
		}
	}
	if detail {
		v.detailTitle.SetText(detailHeading(s))
		v.crumbName.SetText(detailHeading(s))
		v.renderResults(s)
	}
	message := ""
	if s.Mode == Loading {
		message = "Loading…"
	}
	if s.Submitting {
		message = "Saving…"
	}
	if s.Err != nil {
		message = "Error: " + s.Err.Error()
	}
	if !list {
		v.status.SetText(message)
	}
	v.root.Refresh()
}

func detailHeading(s State) string {
	if s.Operation == ShowExact || s.Operation == ShowKey {
		return s.Value
	}
	if s.Result.TargetName != "" {
		return s.Result.TargetName
	}
	return operationLabel(s.Operation)
}

func (v *View) renderResults(s State) {
	v.resultRows.RemoveAll()
	if s.Err != nil {
		v.resultRows.Add(widget.NewLabel("The operation did not complete."))
		return
	}
	switch s.Operation {
	case Inspect, Add, Remove:
		v.resultRows.Add(row("ENTITY", "TAGS", "RESULT"))
		result := "inspected"
		if s.Operation != Inspect {
			result = "unchanged"
			if s.Result.Changed {
				result = "changed"
			}
		}
		v.resultRows.Add(rowObjects(widget.NewLabel(s.Result.TargetName), ui.TagPills([]string(s.Result.Tags.Canonical())), widget.NewLabel(result)))
	case ShowExact, ShowKey:
		v.resultRows.Add(row("ENTITY TYPE", "ENTITY", "TAG"))
		for _, r := range s.Result.References {
			v.resultRows.Add(rowObjects(widget.NewLabel(r.EntityType), widget.NewLabel(r.EntityID), ui.TagPills([]string{r.Tag})))
		}
		if len(s.Result.References) == 0 {
			v.resultRows.Add(widget.NewLabel("No matching active tag usage."))
		}
	}
	v.resultRows.Refresh()
}

func row(values ...string) framework.CanvasObject {
	objects := make([]framework.CanvasObject, 0, len(values))
	for _, value := range values {
		objects = append(objects, widget.NewLabel(value))
	}
	return container.NewGridWithColumns(len(objects), objects...)
}
func rowObjects(objects ...framework.CanvasObject) framework.CanvasObject {
	return container.NewGridWithColumns(len(objects), objects...)
}
func operationLabel(o Operation) string {
	switch o {
	case Inspect:
		return "Inspect entity tags"
	case Add:
		return "Tag entity"
	case Remove:
		return "Untag entity"
	case ShowExact:
		return "Tag references"
	case ShowKey:
		return "Tag key references"
	case Summary:
		return "Tags"
	}
	return "Tags"
}
