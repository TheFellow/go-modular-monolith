package tui

import (
	"context"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// CreateIngredientVM renders a create ingredient form.
type CreateIngredientVM struct {
	app         *app.App
	principal   cedar.EntityUID
	form        *forms.Form
	styles      forms.FormStyles
	keys        forms.FormKeys
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
func NewCreateIngredientVM(app *app.App, principal cedar.EntityUID) *CreateIngredientVM {
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

	formStyles := tuistyles.Form
	formKeys := tuikeys.Form
	form := forms.New(
		formStyles,
		formKeys,
		nameField,
		categoryField,
		unitField,
		descriptionField,
	)

	return &CreateIngredientVM{
		app:         app,
		principal:   principal,
		form:        form,
		styles:      formStyles,
		keys:        formKeys,
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
		if key.Matches(typed, m.keys.Submit) {
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
		errText := m.styles.Error.Render("Error: " + m.err.Error())
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
	m.err = nil
	m.submitting = true

	ingredient := &models.Ingredient{
		Name:        strings.TrimSpace(toString(m.nameField.Value())),
		Category:    toCategory(m.category.Value()),
		Unit:        toUnit(m.unit.Value()),
		Description: strings.TrimSpace(toString(m.description.Value())),
	}

	return func() tea.Msg {
		created, err := m.app.Ingredients.Create(m.context(), ingredient)
		if err != nil {
			return CreateErrorMsg{Err: err}
		}
		return IngredientCreatedMsg{Ingredient: created}
	}
}

func (m *CreateIngredientVM) context() *middleware.Context {
	return m.app.Context(context.Background(), m.principal)
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
