package tui

import (
	"errors"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// CreateDeps defines dependencies for the create ingredient form.
type CreateDeps struct {
	FormStyles forms.FormStyles
	FormKeys   forms.FormKeys
	Ctx        *middleware.Context
	CreateFunc func(ctx *middleware.Context, ing *models.Ingredient) (*models.Ingredient, error)
}

// CreateIngredientVM renders a create ingredient form.
type CreateIngredientVM struct {
	form        *forms.Form
	deps        CreateDeps
	err         error
	submitting  bool
	nameField   *forms.TextField
	category    *forms.SelectField
	unit        *forms.SelectField
	description *forms.TextField
}

// IngredientCreatedMsg is sent when the ingredient has been created.
type IngredientCreatedMsg struct {
	Ingredient *models.Ingredient
}

// CreateErrorMsg is sent when creation fails.
type CreateErrorMsg struct {
	Err error
}

// NewCreateIngredientVM builds a CreateIngredientVM with fields configured.
func NewCreateIngredientVM(deps CreateDeps) *CreateIngredientVM {
	categoryOptions := make([]forms.SelectOption, len(models.AllCategories()))
	for i, c := range models.AllCategories() {
		categoryOptions[i] = forms.SelectOption{Label: string(c), Value: c}
	}

	unitOptions := make([]forms.SelectOption, len(measurement.AllUnits()))
	for i, u := range measurement.AllUnits() {
		unitOptions[i] = forms.SelectOption{Label: string(u), Value: u}
	}

	nameField := forms.NewTextField(
		"Name",
		forms.WithRequired(),
		forms.WithMaxLength(100),
		forms.WithPlaceholder("e.g., Vodka"),
	)
	categoryField := forms.NewSelectField(
		"Category",
		categoryOptions,
		forms.WithRequired(),
	)
	unitField := forms.NewSelectField(
		"Unit",
		unitOptions,
		forms.WithRequired(),
	)
	descriptionField := forms.NewTextField(
		"Description",
		forms.WithMaxLength(500),
		forms.WithPlaceholder("Optional description"),
	)

	form := forms.New(
		deps.FormStyles,
		deps.FormKeys,
		nameField,
		categoryField,
		unitField,
		descriptionField,
	)

	return &CreateIngredientVM{
		form:        form,
		deps:        deps,
		nameField:   nameField,
		category:    categoryField,
		unit:        unitField,
		description: descriptionField,
	}
}

// Init initializes the form.
func (m *CreateIngredientVM) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages for the form.
func (m *CreateIngredientVM) Update(msg tea.Msg) (*CreateIngredientVM, tea.Cmd) {
	switch typed := msg.(type) {
	case CreateErrorMsg:
		m.submitting = false
		m.err = typed.Err
		return m, nil
	case IngredientCreatedMsg:
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
func (m *CreateIngredientVM) View() string {
	view := m.form.View()
	if m.err != nil {
		errText := m.deps.FormStyles.Error.Render("Error: " + m.err.Error())
		return strings.Join([]string{errText, "", view}, "\n")
	}
	return view
}

// SetWidth sets the width of the form.
func (m *CreateIngredientVM) SetWidth(w int) {
	m.form.SetWidth(w)
}

// IsDirty reports whether the form has been modified.
func (m *CreateIngredientVM) IsDirty() bool {
	return m.form.IsDirty()
}

func (m *CreateIngredientVM) submit() tea.Cmd {
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
	m.err = nil
	m.submitting = true

	ingredient := &models.Ingredient{
		Name:        strings.TrimSpace(toString(m.nameField.Value())),
		Category:    toCategory(m.category.Value()),
		Unit:        toUnit(m.unit.Value()),
		Description: strings.TrimSpace(toString(m.description.Value())),
	}

	return func() tea.Msg {
		created, err := m.deps.CreateFunc(m.deps.Ctx, ingredient)
		if err != nil {
			return CreateErrorMsg{Err: err}
		}
		return IngredientCreatedMsg{Ingredient: created}
	}
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

func toCategory(value any) models.Category {
	switch typed := value.(type) {
	case models.Category:
		return typed
	case string:
		return models.Category(typed)
	default:
		return ""
	}
}

func toUnit(value any) measurement.Unit {
	switch typed := value.(type) {
	case measurement.Unit:
		return typed
	case string:
		return measurement.Unit(typed)
	default:
		return ""
	}
}
