package tui

import (
	"errors"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// CreateDeps defines dependencies for the create drink form.
type CreateDeps struct {
	FormStyles      forms.FormStyles
	FormKeys        forms.FormKeys
	Ctx             *middleware.Context
	CreateFunc      func(ctx *middleware.Context, drink *models.Drink) (*models.Drink, error)
	ListIngredients func(ctx *middleware.Context) ([]*ingredientsmodels.Ingredient, error)
}

// CreateDrinkVM renders a create drink form.
type CreateDrinkVM struct {
	form        *forms.Form
	deps        CreateDeps
	err         error
	submitting  bool
	nameField   *forms.TextField
	category    *forms.SelectField
	glass       *forms.SelectField
	description *forms.TextField
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
func NewCreateDrinkVM(deps CreateDeps) *CreateDrinkVM {
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

	form := forms.New(
		deps.FormStyles,
		deps.FormKeys,
		nameField,
		categoryField,
		glassField,
		descriptionField,
	)

	return &CreateDrinkVM{
		form:        form,
		deps:        deps,
		nameField:   nameField,
		category:    categoryField,
		glass:       glassField,
		description: descriptionField,
	}
}

// Init initializes the form.
func (m *CreateDrinkVM) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages for the form.
func (m *CreateDrinkVM) Update(msg tea.Msg) (*CreateDrinkVM, tea.Cmd) {
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
		if key.Matches(typed, m.deps.FormKeys.Submit) {
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
	if m.err != nil {
		errText := m.deps.FormStyles.Error.Render("Error: " + m.err.Error())
		return strings.Join([]string{errText, "", view}, "\n")
	}
	return view
}

// SetWidth sets the width of the form.
func (m *CreateDrinkVM) SetWidth(w int) {
	m.form.SetWidth(w)
}

// IsDirty reports whether the form has been modified.
func (m *CreateDrinkVM) IsDirty() bool {
	return m.form.IsDirty()
}

func (m *CreateDrinkVM) submit() tea.Cmd {
	if m.submitting {
		return nil
	}
	if err := m.form.Validate(); err != nil {
		m.err = err
		return nil
	}
	if m.deps.CreateFunc == nil {
		m.err = errors.New("create function not configured")
		return nil
	}
	recipe, err := m.defaultRecipe()
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
		created, err := m.deps.CreateFunc(m.deps.Ctx, drink)
		if err != nil {
			return CreateErrorMsg{Err: err}
		}
		return DrinkCreatedMsg{Drink: created}
	}
}

func (m *CreateDrinkVM) defaultRecipe() (models.Recipe, error) {
	if m.deps.ListIngredients == nil {
		return models.Recipe{}, errors.New("ingredient list function not configured")
	}
	ingredients, err := m.deps.ListIngredients(m.deps.Ctx)
	if err != nil {
		return models.Recipe{}, err
	}
	if len(ingredients) == 0 {
		return models.Recipe{}, errors.New("at least one ingredient is required to create a drink")
	}
	first := ingredients[0]
	amount, err := measurement.NewAmount(1, first.Unit)
	if err != nil {
		return models.Recipe{}, err
	}
	return models.Recipe{
		Ingredients: []models.RecipeIngredient{
			{IngredientID: first.ID, Amount: amount},
		},
		Steps: []string{"Add ingredients and serve."},
	}, nil
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
