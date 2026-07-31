package gui

import (
	"fmt"
	"strconv"
	"strings"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const (
	ControlFilter       = "ingredients-filter"
	ControlApplyFilter  = "ingredients-apply-filter"
	ControlRefresh      = "ingredients-refresh"
	ControlCreate       = "ingredients-create"
	ControlEdit         = "ingredients-edit"
	ControlDelete       = "ingredients-delete"
	ControlTags         = "ingredients-tags"
	ControlPrevious     = "ingredients-previous"
	ControlNext         = "ingredients-next"
	ControlSelectPrefix = "ingredient-select-"
	ControlFormTags     = "ingredient-form-tags"
	ControlName         = "ingredient-form-name"
	ControlDescription  = "ingredient-form-description"
	ControlMutationTags = "ingredient-form-mutation-tags"
	ControlSave         = "ingredient-form-save"
	ControlCancel       = "ingredient-form-cancel"
)

type View struct {
	presenter *Presenter
	root      *framework.Container

	expression                                                       *toolkit.SemanticEntry
	category, limit                                                  *widget.Select
	name                                                             *toolkit.SemanticEntry
	description                                                      *toolkit.SemanticEntry
	tags                                                             *toolkit.SemanticEntry
	formCategory                                                     *widget.Select
	formUnit                                                         *widget.Select
	save                                                             *toolkit.SemanticButton
	refresh, create, edit, delete, tagAction, cancel, previous, next *toolkit.SemanticButton
	rows                                                             map[string]*toolkit.SemanticButton
}

var _ toolkit.View = (*View)(nil)
var _ toolkit.Activated = (*View)(nil)

func NewView(presenter *Presenter) *View {
	v := &View{presenter: presenter, root: container.NewStack()}
	presenter.OnChange(v.render)
	v.render(presenter.Snapshot())
	return v
}

func (v *View) Title() string { return "Ingredients" }

func (v *View) Content() framework.CanvasObject { return v.root }

func (v *View) Activate() { v.presenter.Load() }

func (v *View) ExecuteCommand(command toolkit.Command) bool {
	state := v.presenter.Snapshot()
	switch command {
	case toolkit.CommandRefresh:
		return state.Mode == Browse && toolkit.Trigger(v.refresh)
	case toolkit.CommandNew:
		return state.Mode == Browse && toolkit.Trigger(v.create)
	case toolkit.CommandSave:
		return state.Mode != Browse && toolkit.Trigger(v.save)
	case toolkit.CommandCancel:
		return state.Mode != Browse && toolkit.Trigger(v.cancel)
	}
	return false
}

func (v *View) render(state State) {
	v.expression = toolkit.NewEntry(ControlFilter)
	v.expression.SetPlaceHolder(`category == "spirit"`)
	v.expression.SetText(state.Expression)
	categoryOptions := []string{"all"}
	for _, category := range models.AllCategories() {
		categoryOptions = append(categoryOptions, string(category))
	}
	v.category = widget.NewSelect(categoryOptions, nil)
	selectedCategory := "all"
	if state.Category != "" {
		selectedCategory = string(state.Category)
	}
	v.category.SetSelected(selectedCategory)
	v.limit = widget.NewSelect([]string{"25", "50", "100"}, nil)
	v.limit.SetSelected(strconv.Itoa(state.Limit))
	apply := toolkit.NewButton(ControlApplyFilter, "Apply", func() {
		category := models.Category(v.category.Selected)
		if v.category.Selected == "all" {
			category = ""
		}
		limit, _ := strconv.Atoi(v.limit.Selected)
		v.presenter.Filter(category, v.expression.Text, limit)
	})
	filters := container.NewBorder(nil, nil, container.NewHBox(v.category, v.limit), apply, v.expression)

	v.refresh = toolkit.NewButton(ControlRefresh, "Refresh", v.presenter.Load)
	v.create = toolkit.Primary(toolkit.NewButton(ControlCreate, "New ingredient", v.presenter.StartCreate))
	v.edit = toolkit.NewButton(ControlEdit, "Edit", v.presenter.StartEdit)
	v.delete = toolkit.Destructive(toolkit.NewButton(ControlDelete, "Delete", v.presenter.RequestDelete))
	v.tagAction = toolkit.NewButton(ControlTags, "Tags", v.presenter.StartTags)
	v.previous = toolkit.NewButton(ControlPrevious, "Previous", v.presenter.PreviousPage)
	v.next = toolkit.NewButton(ControlNext, "Next", v.presenter.NextPage)
	busy := state.Mode != Browse || state.Submitting || state.Status == toolkit.Loading
	if busy {
		for _, button := range []*toolkit.SemanticButton{v.refresh, v.create, v.edit, v.delete, v.tagAction, apply, v.previous, v.next} {
			button.Disable()
		}
		v.expression.Disable()
		v.category.Disable()
		v.limit.Disable()
	}
	if len(state.History) == 0 {
		v.previous.Disable()
	}
	if state.Next == "" {
		v.next.Disable()
	}
	if state.Selected == nil {
		v.edit.Disable()
		v.delete.Disable()
		v.tagAction.Disable()
	}
	rows := container.NewVBox()
	v.rows = make(map[string]*toolkit.SemanticButton, len(state.Items))
	for i := range state.Items {
		ingredient := state.Items[i]
		id := ingredient.ID
		label := fmt.Sprintf("%s  ·  %s", ingredient.Name, ingredient.Category)
		button := toolkit.NewButton(ControlSelectPrefix+id.String(), label, func() { v.presenter.Select(id) })
		if busy {
			button.Disable()
		}
		v.rows[id.String()] = button
		rows.Add(button)
	}
	if len(state.Items) == 0 && state.Status == toolkit.Loaded {
		rows.Add(widget.NewLabel("No ingredients found"))
	}
	list := container.NewScroll(rows)

	detail := v.detail(state)
	status := ""
	switch state.Status {
	case toolkit.Loading:
		status = "Loading ingredients…"
	case toolkit.Failed:
		status = "Unable to load ingredients"
	case toolkit.Idle, toolkit.Loaded:
	}
	if state.Err != nil {
		status = "Error: " + state.Err.Error()
	}
	statusLabel := widget.NewLabel(status)
	objects := []framework.CanvasObject{toolkit.StandardListPage(toolkit.ListPage{
		Title: "Ingredients", Subtitle: "Browse the ingredient catalog and select an item to inspect or edit it.", Filters: filters,
		PrimaryActions: []framework.CanvasObject{v.create, v.refresh},
		OtherActions:   []framework.CanvasObject{v.edit, v.tagAction, v.delete},
		List:           list, Detail: detail, Status: statusLabel,
		Paging: container.NewHBox(v.previous, v.next), ListRatio: .38,
	})}
	v.root.Objects = objects
	v.root.Refresh()
}

func (v *View) detail(state State) framework.CanvasObject {
	if state.Mode != Browse {
		return v.form(state)
	}
	if state.Selected == nil {
		return toolkit.EmptyDetail("an ingredient")
	}
	ingredient := state.Selected
	tags := ingredient.Tags.Canonical().String()
	if tags == "" {
		tags = "None"
	}
	values := container.NewVBox(
		widget.NewLabelWithStyle(ingredient.Name, framework.TextAlignLeading, framework.TextStyle{Bold: true}),
		widget.NewLabel("ID: "+ingredient.ID.String()),
		widget.NewLabel("Category: "+string(ingredient.Category)),
		widget.NewLabel("Unit: "+string(ingredient.Unit)),
		widget.NewLabel("Tags: "+tags),
	)
	if strings.TrimSpace(ingredient.Description) != "" {
		values.Add(widget.NewSeparator())
		values.Add(widget.NewLabelWithStyle("Description", framework.TextAlignLeading, framework.TextStyle{Bold: true}))
		description := widget.NewLabel(ingredient.Description)
		description.Wrapping = framework.TextWrapWord
		values.Add(description)
	}
	return container.NewPadded(values)
}

func (v *View) form(state State) framework.CanvasObject {
	if state.Mode == Tags {
		v.tags = toolkit.NewEntry(ControlFormTags)
		v.tags.SetPlaceHolder("featured, region=west")
		v.tags.SetText(state.Form.Tags)
		return v.formFrame("Edit Tags", container.NewVBox(widget.NewLabel("Complete tag set (CSV)"), v.tags), state, func() Form {
			return Form{Tags: v.tags.Text}
		})
	}
	v.name = toolkit.NewEntry(ControlName)
	v.name.SetText(state.Form.Name)
	v.description = toolkit.NewEntry(ControlDescription)
	v.description.MultiLine = true
	v.description.SetText(state.Form.Description)
	v.tags = toolkit.NewEntry(ControlMutationTags)
	v.tags.SetPlaceHolder("featured, region=west")
	v.tags.SetText(state.Form.Tags)
	categories := make([]string, 0, len(models.AllCategories()))
	for _, value := range models.AllCategories() {
		categories = append(categories, string(value))
	}
	v.formCategory = widget.NewSelect(categories, nil)
	v.formCategory.SetSelected(string(state.Form.Category))
	units := make([]string, 0, len(measurement.AllUnits()))
	for _, value := range measurement.AllUnits() {
		units = append(units, string(value))
	}
	v.formUnit = widget.NewSelect(units, nil)
	v.formUnit.SetSelected(string(state.Form.Unit))
	title := "Create Ingredient"
	if state.Mode == Edit {
		title = "Edit Ingredient"
	}
	fields := widget.NewForm(
		widget.NewFormItem("Name", v.name),
		widget.NewFormItem("Category", v.formCategory),
		widget.NewFormItem("Unit", v.formUnit),
		widget.NewFormItem("Description", v.description),
		widget.NewFormItem("Tags", v.tags),
	)
	return v.formFrame(title, fields, state, func() Form {
		return Form{Name: v.name.Text, Category: models.Category(v.formCategory.Selected), Unit: measurement.Unit(v.formUnit.Selected), Description: v.description.Text, Tags: v.tags.Text, ReplaceTags: true}
	})
}

func (v *View) formFrame(title string, fields framework.CanvasObject, state State, value func() Form) framework.CanvasObject {
	errorText := ""
	if state.Err != nil {
		errorText = "Error: " + state.Err.Error()
	}
	save := toolkit.NewButton(ControlSave, "Save", func() { v.presenter.Submit(value()) })
	v.save = save
	if state.Submitting {
		save.Disable()
	}
	v.cancel = toolkit.NewButton(ControlCancel, "Cancel", v.presenter.Cancel)
	if state.Submitting {
		v.cancel.Disable()
	}
	return toolkit.StandardFormPage(toolkit.FormPage{Title: title, Fields: fields, Status: widget.NewLabel(errorText), Save: save, Cancel: v.cancel})
}
