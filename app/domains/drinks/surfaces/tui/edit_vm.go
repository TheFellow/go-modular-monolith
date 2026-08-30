package tui

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	toolkittui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// EditDrinkVM renders an edit drink form.
type EditDrinkVM struct {
	app         *app.Session
	form        *forms.Form
	drink       *models.Drink
	styles      forms.FormStyles
	keys        forms.FormKeys
	err         error
	submitting  bool
	nameField   *forms.TextField
	category    *forms.SelectField
	glass       *forms.SelectField
	description *forms.TextField
	tags        *forms.TextField
	recipe      *RecipeEditor
	viewport    toolkittui.FormViewport
}

// DrinkUpdatedMsg is sent when the drink has been updated.
type DrinkUpdatedMsg struct {
	Drink *models.Drink
}

// UpdateErrorMsg is sent when updating fails.
type UpdateErrorMsg struct {
	Err error
}

// NewEditDrinkVM builds an EditDrinkVM with fields configured.
func NewEditDrinkVM(app *app.Session, drink *models.Drink) *EditDrinkVM {
	if drink == nil {
		drink = &models.Drink{}
	}
	categoryOptions := make([]forms.SelectOption, len(models.AllDrinkCategories()))
	for i, c := range models.AllDrinkCategories() {
		categoryOptions[i] = forms.SelectOption{Label: string(c), Value: c}
	}

	glassOptions := make([]forms.SelectOption, len(models.AllGlassTypes()))
	for i, g := range models.AllGlassTypes() {
		glassOptions[i] = forms.SelectOption{Label: string(g), Value: g}
	}

	nameField := forms.NewTextField(
		"Name",
		forms.WithRequired(),
		forms.WithMaxLength(100),
		forms.WithInitialValue(drink.Name),
	)
	categoryField := forms.NewSelectField(
		"Category",
		categoryOptions,
		forms.WithRequired(),
		forms.WithInitialValue(drink.Category),
	)
	glassField := forms.NewSelectField(
		"Glass",
		glassOptions,
		forms.WithRequired(),
		forms.WithInitialValue(drink.Glass),
	)
	descriptionField := forms.NewTextField(
		"Description",
		forms.WithMaxLength(500),
		forms.WithInitialValue(drink.Description),
	)
	tagsField := components.NewOptionalTagsField(drink.Tags.Canonical().String())

	formStyles := styles.Standard.Form
	formKeys := keys.Standard.Form
	recipeField := NewRecipeEditor(app, formStyles, drink.Recipe)
	form := forms.New(
		formStyles,
		formKeys,
		nameField,
		categoryField,
		glassField,
		descriptionField,
		recipeField,
		tagsField,
	)

	return &EditDrinkVM{
		app:         app,
		form:        form,
		drink:       drink,
		styles:      formStyles,
		keys:        formKeys,
		nameField:   nameField,
		category:    categoryField,
		glass:       glassField,
		description: descriptionField,
		tags:        tagsField,
		recipe:      recipeField,
		viewport:    toolkittui.NewFormViewport(),
	}
}

// Init initializes the form.
func (m *EditDrinkVM) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages for the form.
func (m *EditDrinkVM) Update(msg tea.Msg) (*EditDrinkVM, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.SetSize(size.Width, size.Height)
	}
	if loaded, ok := msg.(ingredientCatalogLoadedMsg); ok && m.recipe.AcceptCatalog(loaded) {
		return m, nil
	}
	switch typed := msg.(type) {
	case UpdateErrorMsg:
		m.submitting = false
		m.err = typed.Err
		return m, nil
	case DrinkUpdatedMsg:
		m.submitting = false
		m.err = nil
		return m, nil
	case tea.KeyMsg:
		if key.Matches(typed, m.keys.Submit) {
			return m, m.submit()
		}
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

// View renders the form.
func (m *EditDrinkVM) View() string {
	view := m.form.View()
	errorView := ""
	if m.err != nil {
		errorView = m.styles.Error.Render("Error: " + m.err.Error())
		view = strings.Join([]string{errorView, "", view}, "\n")
	}
	footer := ""
	if m.recipe.IsFocused() {
		footer = m.styles.Help.Render(recipeNavigationHelp)
	}
	offset := recipeFieldOffset(errorView, m.nameField, m.category, m.glass, m.description)
	return m.viewport.View(view, recipeFocusLine(offset, m.recipe), footer)
}

// SetWidth sets the width of the form.
func (m *EditDrinkVM) SetWidth(w int) {
	m.form.SetWidth(w)
	m.viewport.SetWidth(w)
}

// SetSize sets the form viewport dimensions.
func (m *EditDrinkVM) SetSize(width, height int) {
	m.form.SetWidth(width)
	m.viewport.SetSize(width, height)
}

// IsDirty reports whether the form has been modified.
func (m *EditDrinkVM) IsDirty() bool {
	return m.form.IsDirty()
}

// Submitting reports whether a mutation is in flight.
func (m *EditDrinkVM) Submitting() bool { return m.submitting }

func (m *EditDrinkVM) submit() tea.Cmd {
	if m.submitting {
		return nil
	}
	if err := m.form.Validate(); err != nil {
		m.err = err
		return nil
	}
	if m.drink == nil {
		m.err = errors.New("drink not loaded")
		return nil
	}
	desired, err := components.DesiredTags(m.tags, tag.ParseCollection)
	if err != nil {
		m.err = err
		return nil
	}
	m.err = nil
	m.submitting = true

	updated := *m.drink
	updated.Name = strings.TrimSpace(toString(m.nameField.Value()))
	updated.Category = toDrinkCategory(m.category.Value())
	updated.Glass = toGlassType(m.glass.Value())
	updated.Description = strings.TrimSpace(toString(m.description.Value()))
	updated.Recipe = m.recipe.Value().(models.Recipe)

	return func() tea.Msg {
		drink, err := app.RunTaggedMutation(m.app.App, m.context(), desired, func(ctx *middleware.Context) (*models.Drink, error) {
			return m.app.Drinks.Update(ctx, &updated)
		})
		if err != nil {
			return UpdateErrorMsg{Err: err}
		}
		return DrinkUpdatedMsg{Drink: drink}
	}
}

func (m *EditDrinkVM) context() *middleware.Context {
	return m.app.Context()
}
