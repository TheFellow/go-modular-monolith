package tui

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// CreateDrinkVM renders a create drink form.
type CreateDrinkVM struct {
	app         *app.Session
	form        *forms.Form
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
	viewport    formViewport
}

// DrinkCreatedMsg is sent when the drink has been created.
type DrinkCreatedMsg struct {
	Drink *models.Drink
}

// CreateErrorMsg is sent when creation fails.
type CreateErrorMsg struct {
	Err error
}

// NewCreateDrinkVM builds a CreateDrinkVM with fields configured.
func NewCreateDrinkVM(app *app.Session) *CreateDrinkVM {
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
	)
	categoryField := forms.NewSelectField(
		"Category",
		categoryOptions,
		forms.WithRequired(),
	)
	glassField := forms.NewSelectField(
		"Glass",
		glassOptions,
		forms.WithRequired(),
	)
	descriptionField := forms.NewTextField(
		"Description",
		forms.WithMaxLength(500),
	)
	tagsField := components.NewOptionalTagsField("")
	formStyles := styles.Standard.Form
	formKeys := keys.Standard.Form
	recipeField := NewRecipeEditor(app, formStyles, models.Recipe{})
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

	return &CreateDrinkVM{
		app:         app,
		form:        form,
		styles:      formStyles,
		keys:        formKeys,
		nameField:   nameField,
		category:    categoryField,
		glass:       glassField,
		description: descriptionField,
		tags:        tagsField,
		recipe:      recipeField,
		viewport:    newFormViewport(),
	}
}

// Init initializes the form.
func (m *CreateDrinkVM) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages for the form.
func (m *CreateDrinkVM) Update(msg tea.Msg) (*CreateDrinkVM, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.SetSize(size.Width, size.Height)
	}
	if loaded, ok := msg.(ingredientCatalogLoadedMsg); ok && m.recipe.AcceptCatalog(loaded) {
		return m, nil
	}
	switch typed := msg.(type) {
	case CreateErrorMsg:
		m.submitting = false
		m.err = typed.Err
		return m, nil
	case DrinkCreatedMsg:
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
func (m *CreateDrinkVM) View() string {
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
func (m *CreateDrinkVM) SetWidth(w int) {
	m.form.SetWidth(w)
	m.viewport.SetSize(w, m.viewport.height)
}

// SetSize sets the form viewport dimensions.
func (m *CreateDrinkVM) SetSize(width, height int) {
	m.form.SetWidth(width)
	m.viewport.SetSize(width, height)
}

// IsDirty reports whether the form has been modified.
func (m *CreateDrinkVM) IsDirty() bool {
	return m.form.IsDirty()
}

// Submitting reports whether a mutation is in flight.
func (m *CreateDrinkVM) Submitting() bool { return m.submitting }

func (m *CreateDrinkVM) submit() tea.Cmd {
	if m.submitting {
		return nil
	}
	if err := m.form.Validate(); err != nil {
		m.err = err
		return nil
	}
	recipe := m.recipe.Value().(models.Recipe)
	desired, err := components.DesiredTags(m.tags, tag.ParseCollection)
	if err != nil {
		m.err = err
		return nil
	}
	m.err = nil
	m.submitting = true

	drink := &models.Drink{
		Name:        strings.TrimSpace(toString(m.nameField.Value())),
		Category:    toDrinkCategory(m.category.Value()),
		Glass:       toGlassType(m.glass.Value()),
		Description: strings.TrimSpace(toString(m.description.Value())),
		Recipe:      recipe,
	}

	return func() tea.Msg {
		created, err := app.RunTaggedMutation(m.app.App, m.context(), desired, func(ctx *middleware.Context) (*models.Drink, error) {
			return m.app.Drinks.Create(ctx, drink)
		})
		if err != nil {
			return CreateErrorMsg{Err: err}
		}
		return DrinkCreatedMsg{Drink: created}
	}
}

func (m *CreateDrinkVM) context() *middleware.Context {
	return m.app.Context()
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}

func toDrinkCategory(value any) models.DrinkCategory {
	switch typed := value.(type) {
	case models.DrinkCategory:
		return typed
	case string:
		return models.DrinkCategory(typed)
	default:
		return ""
	}
}

func toGlassType(value any) models.GlassType {
	switch typed := value.(type) {
	case models.GlassType:
		return typed
	case string:
		return models.GlassType(typed)
	default:
		return ""
	}
}
