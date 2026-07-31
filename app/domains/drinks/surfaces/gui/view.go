package gui

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

type ingredientField string

const (
	ingredientFieldIngredient ingredientField = "ingredient"
	ingredientFieldAmount     ingredientField = "amount"
	ingredientFieldUnit       ingredientField = "unit"
	ingredientFieldOptional   ingredientField = "optional"
	ingredientFieldRemove     ingredientField = "remove"
	ingredientFieldSubstitute ingredientField = "substitute"
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
	ingredient    *semanticSelectEntry
	amount        *ui.SemanticEntry
	unit          *semanticSelect
	optional      *semanticCheck
	substitutes   map[entity.IngredientID]*semanticCheck
	substituteBox *framework.Container
	remove        *ui.SemanticButton
}
type View struct {
	presenter                                             *Presenter
	root                                                  *framework.Container
	list                                                  *widget.Table
	status, formStatus, tagStatus                         *widget.Label
	detailTitle, crumbName                                *widget.Label
	browse, formPanel, tagsPanel                          *framework.Container
	filterExpression                                      *ui.SemanticEntry
	filterBar                                             *ui.FilterBar
	filterLimit                                           *semanticSelect
	name, description, steps, garnish, tags, mutationTags *ui.SemanticEntry
	category, glass                                       *semanticSelect
	recipeBox                                             *framework.Container
	recipe                                                []recipeWidgets
	save, cancel, tagSave, tagCancel                      *ui.SemanticButton
	addIngredient                                         *ui.SemanticButton
	refresh, create, previous, next                       *ui.SemanticButton
	detailActions                                         []*ui.SemanticButton
	renderedMode                                          Mode
	renderedFormInstance                                  uint64
	renderedForm                                          Form
	formRendered                                          bool
	rendering                                             bool
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
	columns := []string{"Name", "Category", "Glass", "Ingredients", "Tags"}
	v.list = widget.NewTable(func() (int, int) { return len(p.State().Items) + 1, len(columns) }, func() framework.CanvasObject {
		label := widget.NewLabel("")
		label.Truncation = framework.TextTruncateEllipsis
		return label
	}, func(id widget.TableCellID, o framework.CanvasObject) {
		label := o.(*widget.Label)
		if id.Row == 0 {
			label.TextStyle = framework.TextStyle{Bold: true}
			label.SetText(columns[id.Col])
			return
		}
		label.TextStyle = framework.TextStyle{}
		item := p.State().Items[id.Row-1]
		values := []string{item.Name, string(item.Category), string(item.Glass), strconv.Itoa(len(item.Recipe.Ingredients)), item.Tags.Canonical().String()}
		label.SetText(values[id.Col])
	})
	v.list.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 {
			v.list.UnselectAll()
			p.Select(id.Row - 1)
		}
	}
	v.list.SetColumnWidth(0, 190)
	v.list.SetColumnWidth(1, 110)
	v.list.SetColumnWidth(2, 110)
	v.list.SetColumnWidth(3, 125)
	v.list.SetColumnWidth(4, 190)
	v.refresh = ui.NewButton(ControlRefresh, "Refresh", p.Refresh)
	v.create = ui.Primary(ui.NewButton(ControlCreate, "New drink", p.StartCreate))
	v.previous = ui.NewButton(ControlPrevious, "Previous", p.PreviousPage)
	v.next = ui.NewButton(ControlNext, "Next", p.NextPage)
	edit := ui.NewButton(ControlEdit, "Edit", p.StartEdit)
	tagsAction := ui.NewButton(ControlTags, "Tags", p.StartTags)
	deleteAction := ui.Destructive(ui.NewButton(ControlDelete, "Delete", p.Delete))
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
	v.mutationTags = ui.NewEntry(ControlTagValues + ".mutation")
	v.formStatus = widget.NewLabel("")
	v.recipeBox = container.NewVBox()
	v.save = ui.NewButton(ControlSave, "Save", func() { v.readForm(); p.Save() })
	v.cancel = ui.NewButton(ControlCancel, "Cancel", p.Cancel)
	v.addIngredient = ui.NewButton(ControlAddIngredient, "Add ingredient", func() {
		v.readForm()
		f := p.State().Form
		f.Recipe = append(f.Recipe, RecipeRow{Unit: measurement.UnitOz})
		p.SetForm(f)
	})
	fields := container.NewVBox(field("Name", v.name), field("Category", v.category), field("Glass", v.glass), field("Description", v.description), widget.NewLabelWithStyle("Recipe", framework.TextAlignLeading, framework.TextStyle{Bold: true}), v.recipeBox, v.addIngredient, field("Steps (one per line)", v.steps), field("Garnish", v.garnish), field("Tags (complete set)", v.mutationTags))
	v.detailTitle = widget.NewLabel("Drink")
	v.crumbName = widget.NewLabel("")
	back := ui.NewButton(ControlBack, "Back", p.Back)
	crumb := ui.NewButton(ControlBreadcrumb, "Drinks", p.ResetList)
	// Retain semantic command targets for global shortcuts and compatibility;
	// row selection already opens an editable detail when permitted.
	edit.Hide()
	tagsAction.Hide()
	deleteAction.Hide()
	breadcrumb := container.NewHBox(back, crumb, widget.NewLabel(">"), v.crumbName, edit, tagsAction, deleteAction)
	v.formPanel = ui.StandardFormPage(ui.FormPage{TitleLabel: v.detailTitle, Breadcrumb: breadcrumb, Fields: fields, Status: v.formStatus, Save: v.save, Cancel: v.cancel}).(*framework.Container)
	v.tags = ui.NewEntry(ControlTagValues)
	v.tagStatus = widget.NewLabel("")
	v.tagSave = ui.NewButton(ControlSave+".tags", "Save", func() { v.readForm(); p.Save() })
	v.tagCancel = ui.NewButton(ControlCancel+".tags", "Cancel", p.Cancel)
	v.tagsPanel = ui.StandardFormPage(ui.FormPage{Title: "Edit tags", Subtitle: "Replace the complete tag set.", Fields: container.NewVBox(widget.NewLabel("Comma-separated key or key=value tags"), v.tags), Status: v.tagStatus, Save: v.tagSave, Cancel: v.tagCancel}).(*framework.Container)
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
		v.mutationTags.SetText(state.Form.Tags)
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
		v.presenter.SetForm(Form{Tags: v.tags.Text})
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
	v.presenter.SetForm(Form{Name: v.name.Text, Category: v.category.Selected, Glass: v.glass.Selected, Description: v.description.Text, Recipe: rows, Steps: v.steps.Text, Garnish: v.garnish.Text, Tags: v.mutationTags.Text, ReplaceTags: true})
}
func (v *View) render(state State) {
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
		v.mutationTags.SetText(state.Form.Tags)
		v.rebuildRecipe(state)
		v.renderedForm = cloneForm(state.Form)
		v.renderedMode = state.Mode
		v.renderedFormInstance = state.FormInstance
		v.formRendered = true
	} else if state.Mode == Creating || state.Mode == Editing || state.Mode == Viewing {
		v.updateRecipeOptions(state)
	} else if state.Mode == Tagging {
		v.tags.SetText(state.Form.Tags)
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
		v.save.Hide()
		v.cancel.Hide()
		v.addIngredient.Hide()
	} else {
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
			check := newCheck(ingredientSubstituteControl(i, option.ID), option.Name)
			check.SetChecked(selected[option.ID])
			substitutes[option.ID] = check
			substituteBox.Add(check)
		}
		index := i
		remove := ui.NewButton(ingredientControl(i, ingredientFieldRemove), "Remove", func() {
			v.readForm()
			f := v.presenter.State().Form
			f.Recipe = append(f.Recipe[:index], f.Recipe[index+1:]...)
			v.presenter.SetForm(f)
		})
		ingredient.OnChanged = func(string) { v.formChanged() }
		amount.OnChanged = func(string) { v.formChanged() }
		unit.OnChanged = func(string) { v.formChanged() }
		optional.OnChanged = func(bool) { v.formChanged() }
		for _, check := range substitutes {
			check.OnChanged = func(bool) { v.formChanged() }
		}
		v.recipe = append(v.recipe, recipeWidgets{ingredient: ingredient, amount: amount, unit: unit, optional: optional, substitutes: substitutes, substituteBox: substituteBox, remove: remove})
		v.recipeBox.Add(widget.NewCard(fmt.Sprintf("Ingredient %d", i+1), "", container.NewVBox(field("Ingredient", ingredient), field("Amount", amount), field("Unit", unit), optional, widget.NewLabel("Substitutes"), substituteBox, remove)))
	}
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
		selected := make(map[entity.IngredientID]bool)
		if i < len(state.Form.Recipe) {
			for _, id := range state.Form.Recipe[i].Substitutes {
				selected[id] = true
			}
		}
		for id, check := range w.substitutes {
			selected[id] = check.Checked
		}
		w.substituteBox.RemoveAll()
		w.substitutes = make(map[entity.IngredientID]*semanticCheck)
		for _, option := range state.Ingredients {
			check := newCheck(ingredientSubstituteControl(i, option.ID), option.Name)
			check.SetChecked(selected[option.ID])
			w.substitutes[option.ID] = check
			w.substituteBox.Add(check)
		}
		v.recipe[i] = w
	}
}
func (v *View) setMutableEnabled(enabled bool) {
	objects := []interface {
		Enable()
		Disable()
	}{v.name, v.category, v.glass, v.description, v.steps, v.garnish, v.tags, v.addIngredient}
	for _, object := range objects {
		if enabled {
			object.Enable()
		} else {
			object.Disable()
		}
	}
	for _, row := range v.recipe {
		objects = []interface {
			Enable()
			Disable()
		}{row.ingredient, row.amount, row.unit, row.optional, row.remove}
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
	return container.NewBorder(nil, nil, widget.NewLabel(label), nil, o)
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
