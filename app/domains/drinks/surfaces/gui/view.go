package gui

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

type ingredientField string

const (
	ingredientFieldIngredient       ingredientField = "ingredient"
	ingredientFieldAmount           ingredientField = "amount"
	ingredientFieldUnit             ingredientField = "unit"
	ingredientFieldOptional         ingredientField = "optional"
	ingredientFieldRemove           ingredientField = "remove"
	ingredientFieldSubstitute       ingredientField = "substitute"
	ingredientFieldAddSubstitute    ingredientField = "substitute.add"
	ingredientFieldCancelSubstitute ingredientField = "substitute.cancel"
	ingredientFieldChooseSubstitute ingredientField = "substitute.choose"
	ingredientFieldSubstitutePicker ingredientField = "substitute.picker"
)

const (
	ControlRefresh          = "drinks.refresh"
	ControlCreate           = "drinks.create"
	ControlEdit             = "drinks.edit"
	ControlDelete           = "drinks.delete"
	ControlTags             = "drinks.tags"
	ControlApplyFilter      = "drinks.filter.apply"
	ControlFilterName       = "drinks.filter.name"
	ControlFilterCategory   = "drinks.filter.category"
	ControlFilterGlass      = "drinks.filter.glass"
	ControlFilterExpression = "drinks.filter.expression"
	ControlPrevious         = "drinks.previous"
	ControlNext             = "drinks.next"
	ControlName             = "drinks.form.name"
	ControlCategory         = "drinks.form.category"
	ControlGlass            = "drinks.form.glass"
	ControlDescription      = "drinks.form.description"
	ControlSteps            = "drinks.form.steps"
	ControlGarnish          = "drinks.form.garnish"
	ControlTagValues        = "drinks.form.tags"
	ControlAddIngredient    = "drinks.form.ingredient.add"
	ControlSave             = "drinks.form.save"
	ControlCancel           = "drinks.form.cancel"
	ControlBack             = "drinks.detail.back"
	ControlBreadcrumb       = "drinks.detail.breadcrumb"
)

func ingredientControl(i int, field ingredientField) string {
	return fmt.Sprintf("drinks.form.ingredient.%d.%s", i, field)
}

func ingredientSubstituteControl(i int, id entity.IngredientID) string {
	return ingredientControl(i, ingredientFieldSubstitute) + "." + id.String()
}

type semanticSelect struct {
	widget.Select
	id string
}

func newSelect(id string, options []string) *semanticSelect {
	s := &semanticSelect{id: id}
	s.Options = options
	s.ExtendBaseWidget(s)
	return s
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

type semanticCheck struct {
	widget.Check
	id string
}

func newCheck(id, label string) *semanticCheck {
	c := &semanticCheck{id: id}
	c.Text = label
	c.ExtendBaseWidget(c)
	return c
}
func (c *semanticCheck) SemanticID() string { return c.id }

type recipeWidgets struct {
	ingredient         *semanticSelectEntry
	amount             *ui.SemanticEntry
	unit               *semanticSelect
	optional           *semanticCheck
	substitutes        map[entity.IngredientID]*semanticCheck
	substituteBox      *framework.Container
	substitutePicker   *semanticSelectEntry
	addSubstitute      *ui.SemanticButton
	confirmSubstitute  *ui.SemanticButton
	cancelSubstitute   *ui.SemanticButton
	choosingSubstitute bool
	remove             *ui.SemanticButton
	actions            *widget.Select
}
type View struct {
	presenter                         *Presenter
	root                              *framework.Container
	list                              *widget.Table
	status, formStatus, tagStatus     *widget.Label
	detailTitle, crumbName            *widget.Label
	browse, formPanel, tagsPanel      *framework.Container
	filterExpression                  *ui.SemanticEntry
	filterBar                         *ui.FilterBar
	filterLimit                       *semanticSelect
	name, description, steps, garnish *ui.SemanticEntry
	tags, mutationTags                *ui.TagTokenEditor
	category, glass                   *semanticSelect
	recipeBox                         *framework.Container
	recipe                            []recipeWidgets
	save, cancel, tagSave, tagCancel  *ui.SemanticButton
	addIngredient                     *ui.SemanticButton
	refresh, create, previous, next   *ui.SemanticButton
	detailActions                     []*ui.SemanticButton
	renderedMode                      Mode
	renderedFormInstance              uint64
	renderedForm                      Form
	formRendered                      bool
	rendering                         bool
	tagNaturalWidth                   float32
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(p *Presenter) *View {
	v := &View{presenter: p}
	v.filterLimit = newSelect("drinks.filter.limit", []string{"25", "50", "100"})
	v.filterLimit.SetSelected(strconv.Itoa(p.State().Filter.Limit))
	categoryPresets := []ui.FilterOption{{Label: "Any category"}}
	for _, category := range categoryOptions() {
		categoryPresets = append(categoryPresets, ui.FilterOption{Label: category, Expression: fmt.Sprintf(`category == %q`, category)})
	}
	glassPresets := []ui.FilterOption{{Label: "Any glass"}}
	for _, glass := range glassOptions() {
		glassPresets = append(glassPresets, ui.FilterOption{Label: glass, Expression: fmt.Sprintf(`glass == %q`, glass)})
	}
	bar := ui.NewSingleRowFilterBar(ControlFilterExpression, ControlApplyFilter, `Filter drinks (for example: name.contains("martini"))`, p.State().Filter.Expression,
		[]ui.FilterPreset{{ID: ControlFilterCategory, Placeholder: "Category", Options: categoryPresets}, {ID: ControlFilterGlass, Placeholder: "Glass", Options: glassPresets}},
		container.NewBorder(nil, nil, widget.NewLabel("Page size"), nil, v.filterLimit), func(expression string) {
			limit, _ := strconv.Atoi(v.filterLimit.Selected)
			if p.SetFilter(Filter{Expression: expression, Limit: limit}) {
				p.Refresh()
			}
		})
	v.filterExpression = bar.Expression
	v.filterBar = bar
	filters := bar.Content
	columns := []string{"Name", "Category", "Glass", "Ingredients", "Tags", "Actions"}
	v.list = ui.NewRowTable(func() (int, int) { return len(p.State().Items), len(columns) }, func() framework.CanvasObject {
		return ui.NewActionCell()
	}, func(id widget.TableCellID, o framework.CanvasObject) {
		cell := o
		item := p.State().Items[id.Row]
		values := []string{item.Name, string(item.Category), string(item.Glass), strconv.Itoa(len(item.Recipe.Ingredients)), item.Tags.Canonical().String()}
		if id.Col == len(columns)-1 {
			index := id.Row
			ui.ShowCellActions(cell, []ui.RowAction{{Label: "View", Run: func() { p.Select(index) }}})
			return
		}
		if id.Col == 4 {
			ui.ShowCellTags(cell, values[id.Col])
			return
		}
		ui.ShowCellText(cell, values[id.Col], false)
	})
	v.list.OnSelected = func(id widget.TableCellID) {
		if id.Row >= 0 && id.Col < len(columns)-1 {
			v.list.UnselectAll()
			p.Select(id.Row)
		}
	}
	ui.ConfigureRowTable(v.list, []ui.TableColumn{{Title: "Name", Width: 190}, {Title: "Category", Width: 110}, {Title: "Glass", Width: 110}, {Title: "Ingredients", Width: 125}, {Title: "Tags", Width: 190}, {Title: "Actions", Width: 120}}, nil)
	v.refresh = ui.WithIcon(ui.NewButton(ControlRefresh, "Refresh", p.Refresh), ui.IconRefresh)
	v.create = ui.Primary(ui.WithIcon(ui.NewButton(ControlCreate, "New drink", p.StartCreate), ui.IconAdd))
	v.previous = ui.WithIcon(ui.NewButton(ControlPrevious, "Previous", p.PreviousPage), ui.IconPrevious)
	v.next = ui.WithIcon(ui.NewButton(ControlNext, "Next", p.NextPage), ui.IconNext)
	edit := ui.NewButton(ControlEdit, "Edit", p.StartEdit)
	tagsAction := ui.NewButton(ControlTags, "Tags", p.StartTags)
	deleteAction := ui.Destructive(ui.WithIcon(ui.NewButton(ControlDelete, "Delete", p.Delete), ui.IconDelete))
	v.detailActions = []*ui.SemanticButton{tagsAction, deleteAction}
	v.status = widget.NewLabel("")
	v.browse = ui.StandardListPage(ui.ListPage{
		Title: "Drinks", Filters: filters,
		CollectionActions: []framework.CanvasObject{v.create, v.refresh},
		List:              v.list, Status: v.status,
		Paging: container.NewHBox(v.previous, v.next), ListRatio: .35,
	}).(*framework.Container)
	v.name = ui.NewEntry(ControlName)
	v.category = newSelect(ControlCategory, categoryOptions())
	v.glass = newSelect(ControlGlass, glassOptions())
	v.description = ui.NewEntry(ControlDescription)
	v.description.MultiLine = true
	v.steps = ui.NewEntry(ControlSteps)
	v.steps.MultiLine = true
	v.garnish = ui.NewEntry(ControlGarnish)
	v.mutationTags = ui.NewTagTokenEditor(ControlTagValues+".mutation", "")
	v.mutationTags.Normalize = tag.UpsertCollection
	v.formStatus = widget.NewLabel("")
	v.recipeBox = container.NewVBox()
	v.save = ui.WithIcon(ui.NewButton(ControlSave, "Save", func() { v.readForm(); p.Save() }), ui.IconSave)
	v.cancel = ui.WithIcon(ui.NewButton(ControlCancel, "Cancel", p.Cancel), ui.IconCancel)
	v.addIngredient = ui.NewButton(ControlAddIngredient, "Add ingredient", func() {
		v.readForm()
		f := p.State().Form
		f.Recipe = append(f.Recipe, RecipeRow{Unit: measurement.UnitOz})
		p.SetForm(f)
	})
	fields := container.NewVBox(field("Name", v.name), field("Category", v.category), field("Glass", v.glass), field("Description", v.description), widget.NewLabelWithStyle("Ingredients", framework.TextAlignLeading, framework.TextStyle{Bold: true}), v.recipeBox, container.NewHBox(layout.NewSpacer(), v.addIngredient), field("Steps (one per line)", v.steps), field("Garnish", v.garnish), field("Tags", v.mutationTags.Content))
	v.detailTitle = widget.NewLabel("Drink")
	v.crumbName = widget.NewLabel("")
	back := ui.WithIcon(ui.NewButton(ControlBack, "Back", p.Back), ui.IconBack)
	crumb := ui.NewButton(ControlBreadcrumb, "Drinks", p.ResetList)
	// Retain semantic command targets for global shortcuts and compatibility;
	// row selection already opens an editable detail when permitted.
	edit.Hide()
	tagsAction.Hide()
	deleteAction.Hide()
	breadcrumb := container.NewHBox(back, crumb, widget.NewLabel(">"), v.crumbName, edit, tagsAction, deleteAction)
	v.formPanel = ui.StandardFormPage(ui.FormPage{TitleLabel: v.detailTitle, Breadcrumb: breadcrumb, Fields: fields, Status: v.formStatus, Save: v.save, Cancel: v.cancel}).(*framework.Container)
	v.tags = ui.NewTagTokenEditor(ControlTagValues, "")
	v.tags.Normalize = tag.UpsertCollection
	v.tagStatus = widget.NewLabel("")
	v.tagSave = ui.NewButton(ControlSave+".tags", "Save", func() { v.readForm(); p.Save() })
	v.tagCancel = ui.NewButton(ControlCancel+".tags", "Cancel", p.Cancel)
	v.tagsPanel = ui.StandardFormPage(ui.FormPage{Title: "Edit tags", Subtitle: "Type a key or key=value and press Enter.", Fields: v.tags.Content, Status: v.tagStatus, Save: v.tagSave, Cancel: v.tagCancel}).(*framework.Container)
	v.root = container.NewStack(v.browse, v.formPanel, v.tagsPanel)
	v.name.OnChanged = func(string) { v.formChanged() }
	v.category.OnChanged = func(string) { v.formChanged() }
	v.glass.OnChanged = func(string) { v.formChanged() }
	v.description.OnChanged = func(string) { v.formChanged() }
	v.steps.OnChanged = func(string) { v.formChanged() }
	v.garnish.OnChanged = func(string) { v.formChanged() }
	v.mutationTags.OnChanged = func(string) { v.formChanged() }
	p.Observe(v.render)
	return v
}
func (v *View) formChanged() {
	if v.rendering {
		return
	}
	if state := v.presenter.State(); state.Mode == Viewing {
		// Fyne has no enabled-looking read-only Entry. Keep the controls enabled
		// so their text remains selectable/copyable, while rejecting mutations.
		v.rendering = true
		v.name.SetText(state.Form.Name)
		v.category.SetSelected(state.Form.Category)
		v.glass.SetSelected(state.Form.Glass)
		v.description.SetText(state.Form.Description)
		v.steps.SetText(state.Form.Steps)
		v.garnish.SetText(state.Form.Garnish)
		v.mutationTags.SetCSV(state.Form.Tags)
		v.rebuildRecipe(state)
		v.rendering = false
		return
	} else if state.Mode != Editing {
		return
	}
	v.readForm()
}
func (v *View) Title() string                   { return "Drinks" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.presenter.ResetList() }
func (v *View) HasUnsavedChanges() bool {
	return v.presenter.State().Dirty || v.presenter.State().Mode == Creating || v.presenter.State().Mode == Tagging
}
func (v *View) ExecuteCommand(command ui.Command) bool {
	switch command {
	case ui.CommandRefresh:
		return v.presenter.State().Mode == Browsing && ui.Trigger(v.refresh)
	case ui.CommandNew:
		return v.presenter.State().Mode == Browsing && ui.Trigger(v.create)
	case ui.CommandSave:
		if v.presenter.State().Mode == Tagging {
			return ui.Trigger(v.tagSave)
		}
		return ui.Trigger(v.save)
	case ui.CommandCancel:
		if v.presenter.State().Mode == Tagging {
			return ui.Trigger(v.tagCancel)
		}
		return v.presenter.State().Mode != Browsing && ui.Trigger(v.cancel)
	}
	return false
}
func (v *View) readForm() {
	if v.presenter.State().Mode == Tagging {
		v.presenter.SetForm(Form{Tags: v.tags.CSV()})
		return
	}
	rows := make([]RecipeRow, len(v.recipe))
	for i, w := range v.recipe {
		var substitutes []entity.IngredientID
		for _, option := range v.presenter.State().Ingredients {
			if check := w.substitutes[option.ID]; check != nil && check.Checked {
				substitutes = append(substitutes, option.ID)
			}
		}
		rows[i] = RecipeRow{Ingredient: v.optionID(w.ingredient.Text), Amount: w.amount.Text, Unit: measurement.Unit(w.unit.Selected), Optional: w.optional.Checked, Substitutes: substitutes}
	}
	v.presenter.SetForm(Form{Name: v.name.Text, Category: v.category.Selected, Glass: v.glass.Selected, Description: v.description.Text, Recipe: rows, Steps: v.steps.Text, Garnish: v.garnish.Text, Tags: v.mutationTags.CSV(), ReplaceTags: true})
}
func (v *View) render(state State) {
	if len(state.Items) > 0 {
		values := make([]string, len(state.Items))
		for i, item := range state.Items {
			values[i] = item.Tags.Canonical().String()
		}
		if width := ui.TagPillColumnWidth(values, 190); width > v.tagNaturalWidth {
			v.list.SetColumnWidth(4, width)
			v.tagNaturalWidth = width
		}
	}
	v.rendering = true
	defer func() { v.rendering = false }()
	if len(state.History) == 0 || state.Loading {
		v.previous.Disable()
	} else {
		v.previous.Enable()
	}
	if state.Next == "" || state.Loading {
		v.next.Disable()
	} else {
		v.next.Enable()
	}
	v.browse.Hidden = state.Mode != Browsing
	v.formPanel.Hidden = state.Mode != Creating && state.Mode != Editing && state.Mode != Viewing
	v.tagsPanel.Hidden = state.Mode != Tagging
	formChanged := !v.formRendered || v.renderedMode != state.Mode || v.renderedFormInstance != state.FormInstance || !reflect.DeepEqual(v.renderedForm, state.Form)
	if (state.Mode == Creating || state.Mode == Editing || state.Mode == Viewing) && formChanged {
		v.name.SetText(state.Form.Name)
		if state.Form.Category == "" {
			v.category.ClearSelected()
		} else {
			v.category.SetSelected(state.Form.Category)
		}
		if state.Form.Glass == "" {
			v.glass.ClearSelected()
		} else {
			v.glass.SetSelected(state.Form.Glass)
		}
		v.description.SetText(state.Form.Description)
		v.steps.SetText(state.Form.Steps)
		v.garnish.SetText(state.Form.Garnish)
		v.mutationTags.SetCSV(state.Form.Tags)
		v.rebuildRecipe(state)
		v.renderedForm = cloneForm(state.Form)
		v.renderedMode = state.Mode
		v.renderedFormInstance = state.FormInstance
		v.formRendered = true
	} else if state.Mode == Creating || state.Mode == Editing || state.Mode == Viewing {
		v.updateRecipeOptions(state)
	} else if state.Mode == Tagging {
		v.tags.SetCSV(state.Form.Tags)
	}
	if state.Submitting || state.Loading || (state.Mode == Editing && !state.Dirty) {
		v.save.Disable()
		v.tagSave.Disable()
	} else {
		v.save.Enable()
		v.tagSave.Enable()
	}
	if state.Submitting || (state.Mode == Editing && !state.Dirty) {
		v.cancel.Disable()
		v.tagCancel.Disable()
	} else {
		v.cancel.Enable()
		v.tagCancel.Enable()
	}
	switch {
	case state.Loading:
		v.formStatus.SetText("Loading ingredients…")
	case state.Submitting:
		v.formStatus.SetText("Saving…")
	case state.Err != nil:
		v.formStatus.SetText("Error: " + state.Err.Error())
	default:
		v.formStatus.SetText("")
	}
	v.tagStatus.SetText(v.formStatus.Text)
	v.setMutableEnabled(!state.Submitting)
	if state.Mode == Viewing {
		v.mutationTags.SetEnabled(false)
		v.save.Hide()
		v.cancel.Hide()
		v.addIngredient.Hide()
	} else {
		v.mutationTags.SetEnabled(!state.Submitting)
		v.save.Show()
		v.cancel.Show()
		v.addIngredient.Show()
	}
	if state.Mode == Creating {
		v.detailTitle.SetText("New drink")
		v.crumbName.SetText("New")
	} else if state.Selected != nil {
		v.detailTitle.SetText(state.Selected.Name)
		v.crumbName.SetText(state.Selected.Name)
	}
	v.list.Refresh()
	for _, action := range v.detailActions {
		allowed := state.Selected != nil
		switch action.SemanticID() {
		case ControlTags:
			allowed = allowed && state.CanTag
		case ControlDelete:
			allowed = allowed && state.CanDelete
		}
		action.Hidden = !allowed
		action.Refresh()
	}
	switch {
	case state.Loading:
		v.status.SetText("Loading…")
	case state.Err != nil:
		v.status.SetText("Error: " + state.Err.Error())
	default:
		v.status.SetText(fmt.Sprintf("%d drinks", len(state.Items)))
	}
	v.root.Refresh()
}
func (v *View) rebuildRecipe(state State) {
	labels := optionLabels(state.Ingredients)
	v.recipe = nil
	v.recipeBox.RemoveAll()
	for i, row := range state.Form.Recipe {
		ingredient := newSelectEntry(ingredientControl(i, ingredientFieldIngredient), labels)
		ingredient.SetText(v.optionLabel(row.Ingredient))
		amount := ui.NewEntry(ingredientControl(i, ingredientFieldAmount))
		amount.SetText(row.Amount)
		unit := newSelect(ingredientControl(i, ingredientFieldUnit), unitOptions())
		unit.SetSelected(string(row.Unit))
		optional := newCheck(ingredientControl(i, ingredientFieldOptional), "Optional")
		optional.SetChecked(row.Optional)
		substituteBox := container.NewVBox()
		substitutes := make(map[entity.IngredientID]*semanticCheck)
		selected := make(map[entity.IngredientID]bool)
		for _, id := range row.Substitutes {
			selected[id] = true
		}
		for _, option := range state.Ingredients {
			if !selected[option.ID] {
				continue
			}
			check := newCheck(ingredientSubstituteControl(i, option.ID), option.Name)
			check.SetChecked(true)
			substitutes[option.ID] = check
		}
		substitutePicker := newSelectEntry(ingredientControl(i, ingredientFieldSubstitutePicker), nil)
		substitutePicker.SetPlaceHolder("Choose a substitute")
		addSubstitute := ui.NewButton(ingredientControl(i, ingredientFieldChooseSubstitute), "Add substitute", nil)
		confirmSubstitute := ui.NewButton(ingredientControl(i, ingredientFieldAddSubstitute), "Add", nil)
		cancelSubstitute := ui.NewButton(ingredientControl(i, ingredientFieldCancelSubstitute), "Cancel", nil)
		index := i
		remove := ui.NewButton(ingredientControl(i, ingredientFieldRemove), "Remove", func() {
			v.readForm()
			f := v.presenter.State().Form
			f.Recipe = append(f.Recipe[:index], f.Recipe[index+1:]...)
			v.presenter.SetForm(f)
		})
		if state.Mode == Viewing {
			remove.Hide()
		}
		ingredient.OnChanged = func(string) { v.formChanged() }
		amount.OnChanged = func(string) { v.formChanged() }
		unit.OnChanged = func(string) { v.formChanged() }
		optional.OnChanged = func(bool) { v.formChanged() }
		for _, check := range substitutes {
			check.OnChanged = func(bool) { v.formChanged() }
		}
		v.recipe = append(v.recipe, recipeWidgets{ingredient: ingredient, amount: amount, unit: unit, optional: optional, substitutes: substitutes, substituteBox: substituteBox, substitutePicker: substitutePicker, addSubstitute: addSubstitute, confirmSubstitute: confirmSubstitute, cancelSubstitute: cancelSubstitute, remove: remove})
		addSubstitute.OnTapped = func() {
			v.recipe[index].choosingSubstitute = true
			v.rebuildSubstituteControls(index, v.presenter.State())
		}
		confirmSubstitute.OnTapped = func() { v.addRecipeSubstitute(index) }
		cancelSubstitute.OnTapped = func() {
			v.recipe[index].choosingSubstitute = false
			v.recipe[index].substitutePicker.SetText("")
			v.rebuildSubstituteControls(index, v.presenter.State())
		}
		v.rebuildSubstituteControls(index, state)
		// A saved recipe component is prescribed as a whole. Present it as a
		// compact line item instead of making immutable identity and measurements
		// look like a large editable form. A newly-added blank component remains
		// an explicit editor until the drink is saved.
		if row.Ingredient == (entity.IngredientID{}) {
			rowFields := container.NewVBox(container.NewGridWithColumns(3, ingredient, amount, unit), optional)
			v.recipeBox.Add(container.NewVBox(container.NewBorder(nil, nil, nil, remove, rowFields), substituteBox, widget.NewSeparator()))
			continue
		}
		name := widget.NewLabelWithStyle(v.ingredientName(row.Ingredient), framework.TextAlignLeading, framework.TextStyle{Bold: true})
		summary := fmt.Sprintf("%s %s", row.Amount, row.Unit)
		if row.Optional {
			summary += "  ·  Optional"
		}
		options := []string(nil)
		if state.Mode != Viewing {
			options = []string{"Add substitute", "Remove"}
		}
		actions := ui.NewActionSelect(options, func(choice string) {
			switch choice {
			case "Add substitute":
				v.recipe[index].choosingSubstitute = true
				v.rebuildSubstituteControls(index, v.presenter.State())
			case "Remove":
				remove.OnTapped()
			}
		})
		if state.Mode == Viewing {
			actions.Hide()
		}
		copyID := widget.NewButtonWithIcon("", ui.IconResource(ui.IconCopy), func() {
			if app := framework.CurrentApp(); app != nil {
				app.Clipboard().SetContent(row.Ingredient.String())
			}
		})
		copyID.Importance = widget.LowImportance
		if state.Submitting {
			actions.Disable()
			copyID.Disable()
		}
		v.recipe[index].actions = actions
		ingredient.Hide()
		amount.Hide()
		unit.Hide()
		optional.Hide()
		remove.Hide()
		line := container.NewBorder(nil, nil, nil, widget.NewLabel(summary), name)
		trailing := container.NewCenter(container.NewHBox(copyID, actions))
		v.recipeBox.Add(container.NewVBox(container.NewBorder(nil, nil, nil, trailing, line), substituteBox, widget.NewSeparator()))
	}
}

func (v *View) addRecipeSubstitute(index int) {
	if index < 0 || index >= len(v.recipe) {
		return
	}
	w := &v.recipe[index]
	id := v.optionID(w.substitutePicker.Text)
	if id == (entity.IngredientID{}) || w.substitutes[id] != nil || id == v.optionID(w.ingredient.Text) {
		return
	}
	check := newCheck(ingredientSubstituteControl(index, id), v.ingredientName(id))
	check.SetChecked(true)
	check.OnChanged = func(bool) { v.formChanged() }
	w.substitutes[id] = check
	w.substitutePicker.SetText("")
	w.choosingSubstitute = false
	v.rebuildSubstituteControls(index, v.presenter.State())
	v.formChanged()
}

func (v *View) removeRecipeSubstitute(index int, id entity.IngredientID) {
	if index < 0 || index >= len(v.recipe) {
		return
	}
	delete(v.recipe[index].substitutes, id)
	v.rebuildSubstituteControls(index, v.presenter.State())
	v.formChanged()
}

func (v *View) rebuildSubstituteControls(index int, state State) {
	if index < 0 || index >= len(v.recipe) {
		return
	}
	w := &v.recipe[index]
	w.substituteBox.RemoveAll()
	selected := make([]entity.IngredientID, 0, len(w.substitutes))
	for _, option := range state.Ingredients {
		if check := w.substitutes[option.ID]; check != nil && check.Checked {
			selected = append(selected, option.ID)
		}
	}
	if len(selected) > 0 {
		pills := make([]framework.CanvasObject, 0, len(selected))
		for _, id := range selected {
			id := id
			label := ingredientName(state.Ingredients, id)
			if state.Mode == Viewing {
				pills = append(pills, ui.TagPills([]string{label}))
			} else {
				remove := ui.NewButton(ingredientSubstituteControl(index, id), label+"  ×", func() { v.removeRecipeSubstitute(index, id) })
				remove.Importance = widget.LowImportance
				pills = append(pills, remove)
			}
		}
		w.substituteBox.Add(container.NewHBox(widget.NewLabel("Substitutes"), container.NewHBox(pills...)))
	}
	if state.Mode != Viewing {
		primary := v.optionID(w.ingredient.Text)
		options := make([]string, 0, len(state.Ingredients))
		for _, option := range state.Ingredients {
			if option.ID != primary && w.substitutes[option.ID] == nil {
				options = append(options, optionLabel(option))
			}
		}
		w.substitutePicker.SetOptions(options)
		if w.choosingSubstitute {
			w.substituteBox.Add(container.NewBorder(nil, nil, nil, container.NewHBox(w.cancelSubstitute, w.confirmSubstitute), w.substitutePicker))
		} else if primary == (entity.IngredientID{}) {
			w.substituteBox.Add(container.NewHBox(w.addSubstitute))
		}
	}
	w.substituteBox.Refresh()
}
func (v *View) optionID(label string) entity.IngredientID {
	for _, o := range v.presenter.State().Ingredients {
		if optionLabel(o) == strings.TrimSpace(label) {
			return o.ID
		}
	}
	return entity.IngredientID{}
}
func (v *View) updateRecipeOptions(state State) {
	labels := optionLabels(state.Ingredients)
	for i, w := range v.recipe {
		w.ingredient.SetOptions(labels)
		if i < len(state.Form.Recipe) && w.ingredient.Text == state.Form.Recipe[i].Ingredient.String() {
			w.ingredient.SetText(v.optionLabel(state.Form.Recipe[i].Ingredient))
		}
		if i < len(state.Form.Recipe) {
			for _, id := range state.Form.Recipe[i].Substitutes {
				if check := w.substitutes[id]; check != nil {
					check.Text = ingredientName(state.Ingredients, id)
					check.Refresh()
					continue
				}
				check := newCheck(ingredientSubstituteControl(i, id), ingredientName(state.Ingredients, id))
				check.SetChecked(true)
				check.OnChanged = func(bool) { v.formChanged() }
				w.substitutes[id] = check
			}
		}
		v.recipe[i] = w
		v.rebuildSubstituteControls(i, state)
	}
}
func (v *View) setMutableEnabled(enabled bool) {
	objects := []interface {
		Enable()
		Disable()
	}{v.name, v.category, v.glass, v.description, v.steps, v.garnish, v.addIngredient}
	for _, object := range objects {
		if enabled {
			object.Enable()
		} else {
			object.Disable()
		}
	}
	v.tags.SetEnabled(enabled)
	for _, row := range v.recipe {
		objects = []interface {
			Enable()
			Disable()
		}{row.ingredient, row.amount, row.unit, row.optional, row.remove, row.substitutePicker, row.addSubstitute, row.confirmSubstitute, row.cancelSubstitute}
		if row.actions != nil {
			objects = append(objects, row.actions)
		}
		for _, object := range objects {
			if enabled {
				object.Enable()
			} else {
				object.Disable()
			}
		}
		for _, check := range row.substitutes {
			if enabled {
				check.Enable()
			} else {
				check.Disable()
			}
		}
	}
}
func (v *View) optionLabel(id entity.IngredientID) string {
	for _, o := range v.presenter.State().Ingredients {
		if o.ID == id {
			return optionLabel(o)
		}
	}
	return id.String()
}
func (v *View) ingredientName(id entity.IngredientID) string {
	return ingredientName(v.presenter.State().Ingredients, id)
}
func ingredientName(options []IngredientOption, id entity.IngredientID) string {
	for _, o := range options {
		if o.ID == id {
			return o.Name
		}
	}
	return id.String()
}
func optionLabel(o IngredientOption) string { return fmt.Sprintf("%s (%s)", o.Name, o.ID.String()) }
func optionLabels(options []IngredientOption) []string {
	out := make([]string, len(options))
	for i, o := range options {
		out[i] = optionLabel(o)
	}
	return out
}
func categoryOptions() []string {
	values := models.AllDrinkCategories()
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}
func glassOptions() []string {
	values := models.AllGlassTypes()
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}
func unitOptions() []string {
	values := measurement.AllUnits()
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}
func field(label string, o framework.CanvasObject) framework.CanvasObject {
	return ui.DetailField(label, o)
}
func detailText(d *models.Drink, options []IngredientOption) string {
	name := func(id entity.IngredientID) string {
		for _, o := range options {
			if o.ID == id {
				return o.Name
			}
		}
		return id.String()
	}
	var ingredients []string
	for _, ingredient := range d.Recipe.Ingredients {
		line := fmt.Sprintf("• %s — %s", name(ingredient.IngredientID), ingredient.Amount.String())
		if ingredient.Optional {
			line += " (optional)"
		}
		if len(ingredient.Substitutes) > 0 {
			var values []string
			for _, id := range ingredient.Substitutes {
				values = append(values, name(id))
			}
			line += " [substitutes: " + strings.Join(values, ", ") + "]"
		}
		ingredients = append(ingredients, line)
	}
	var steps []string
	for i, step := range d.Recipe.Steps {
		steps = append(steps, fmt.Sprintf("%d. %s", i+1, step))
	}
	return fmt.Sprintf("%s\n\nCategory: %s\nGlass: %s\nTags: %s\n\n%s\n\nRecipe\n%s\n\nSteps\n%s\n\nGarnish: %s", d.Name, d.Category, d.Glass, d.Tags.Canonical(), d.Description, strings.Join(ingredients, "\n"), strings.Join(steps, "\n"), d.Recipe.Garnish)
}
