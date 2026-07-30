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
	fyneui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
)

type View struct {
	presenter *Presenter
	root      *framework.Container

	expression                                                       *fyneui.SemanticEntry
	category, limit                                                  *widget.Select
	name                                                             *fyneui.SemanticEntry
	description                                                      *fyneui.SemanticEntry
	tags                                                             *fyneui.SemanticEntry
	formCategory                                                     *widget.Select
	formUnit                                                         *widget.Select
	save                                                             *fyneui.SemanticButton
	refresh, create, edit, delete, tagAction, cancel, previous, next *fyneui.SemanticButton
	rows                                                             map[string]*fyneui.SemanticButton
}

var _ fyneui.View = (*View)(nil)
var _ fyneui.Activated = (*View)(nil)

func NewView(presenter *Presenter) *View {
	v := &View{presenter: presenter, root: container.NewStack()}
	presenter.OnChange(v.render)
	v.render(presenter.Snapshot())
	return v
}

func (v *View) Title() string { return "Ingredients" }

func (v *View) Content() framework.CanvasObject { return v.root }

func (v *View) Activate() { v.presenter.Load() }

func (v *View) ExecuteCommand(command fyneui.Command) bool {
	state := v.presenter.Snapshot()
	switch command {
	case fyneui.CommandRefresh:
		return state.Mode == Browse && fyneui.Trigger(v.refresh)
	case fyneui.CommandNew:
		return state.Mode == Browse && fyneui.Trigger(v.create)
	case fyneui.CommandSave:
		return state.Mode != Browse && fyneui.Trigger(v.save)
	case fyneui.CommandCancel:
		return state.Mode != Browse && fyneui.Trigger(v.cancel)
	default:
		return false
	}
}

func (v *View) render(state State) {
	v.expression = fyneui.NewEntry("ingredients-filter")
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
	apply := fyneui.NewButton("ingredients-apply-filter", "Apply", func() {
		category := models.Category(v.category.Selected)
		if v.category.Selected == "all" {
			category = ""
		}
		limit, _ := strconv.Atoi(v.limit.Selected)
		v.presenter.Filter(category, v.expression.Text, limit)
	})
	filters := container.NewBorder(nil, nil, container.NewHBox(v.category, v.limit), apply, v.expression)

	v.refresh = fyneui.NewButton("ingredients-refresh", "Refresh", v.presenter.Load)
	v.create = fyneui.NewButton("ingredients-create", "Create", v.presenter.StartCreate)
	v.edit = fyneui.NewButton("ingredients-edit", "Edit", v.presenter.StartEdit)
	v.delete = fyneui.NewButton("ingredients-delete", "Delete", v.presenter.RequestDelete)
	v.tagAction = fyneui.NewButton("ingredients-tags", "Tags", v.presenter.StartTags)
	v.previous = fyneui.NewButton("ingredients-previous", "Previous", v.presenter.PreviousPage)
	v.next = fyneui.NewButton("ingredients-next", "Next", v.presenter.NextPage)
	busy := state.Mode != Browse || state.Submitting || state.Status == fyneui.Loading
	if busy {
		for _, button := range []*fyneui.SemanticButton{v.refresh, v.create, v.edit, v.delete, v.tagAction, apply, v.previous, v.next} {
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
	toolbar := container.NewHBox(
		v.refresh, v.create,
		v.edit, v.delete, v.tagAction, v.previous, v.next,
	)

	rows := container.NewVBox()
	v.rows = make(map[string]*fyneui.SemanticButton, len(state.Items))
	for i := range state.Items {
		ingredient := state.Items[i]
		id := ingredient.ID
		label := fmt.Sprintf("%s  ·  %s", ingredient.Name, ingredient.Category)
		button := fyneui.NewButton("ingredient-select-"+id.String(), label, func() { v.presenter.Select(id) })
		if busy {
			button.Disable()
		}
		v.rows[id.String()] = button
		rows.Add(button)
	}
	if len(state.Items) == 0 && state.Status == fyneui.Loaded {
		rows.Add(widget.NewLabel("No ingredients found"))
	}
	list := container.NewScroll(rows)

	detail := v.detail(state)
	content := fyneui.ListDetail(list, detail, .38)
	status := ""
	switch state.Status {
	case fyneui.Loading:
		status = "Loading ingredients…"
	case fyneui.Failed:
		status = "Unable to load ingredients"
	case fyneui.Idle, fyneui.Loaded:
	}
	if state.Err != nil {
		status = "Error: " + state.Err.Error()
	}
	objects := []framework.CanvasObject{container.NewBorder(container.NewVBox(toolbar, filters, widget.NewLabel(status)), nil, nil, nil, content)}
	v.root.Objects = objects
	v.root.Refresh()
}

func (v *View) detail(state State) framework.CanvasObject {
	if state.Mode != Browse {
		return v.form(state)
	}
	if state.Selected == nil {
		return container.NewPadded(widget.NewLabel("Select an ingredient to view details"))
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
		v.tags = fyneui.NewEntry("ingredient-form-tags")
		v.tags.SetPlaceHolder("featured, region=west")
		v.tags.SetText(state.Form.Tags)
		return v.formFrame("Edit Tags", container.NewVBox(widget.NewLabel("Complete tag set (CSV)"), v.tags), state, func() Form {
			return Form{Tags: v.tags.Text}
		})
	}
	v.name = fyneui.NewEntry("ingredient-form-name")
	v.name.SetText(state.Form.Name)
	v.description = fyneui.NewEntry("ingredient-form-description")
	v.description.MultiLine = true
	v.description.SetText(state.Form.Description)
	v.tags = fyneui.NewEntry("ingredient-form-mutation-tags")
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
	save := fyneui.NewButton("ingredient-form-save", "Save", func() { v.presenter.Submit(value()) })
	v.save = save
	if state.Submitting {
		save.Disable()
	}
	v.cancel = fyneui.NewButton("ingredient-form-cancel", "Cancel", v.presenter.Cancel)
	if state.Submitting {
		v.cancel.Disable()
	}
	return container.NewPadded(container.NewVBox(
		widget.NewLabelWithStyle(title, framework.TextAlignLeading, framework.TextStyle{Bold: true}),
		widget.NewLabel(errorText), fields,
		container.NewHBox(save, v.cancel),
	))
}
