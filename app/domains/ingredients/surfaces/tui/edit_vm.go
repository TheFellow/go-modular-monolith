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

// EditDeps defines dependencies for the edit ingredient form.
type EditDeps struct {
	FormStyles forms.FormStyles
	FormKeys   forms.FormKeys
	Ctx        *middleware.Context
	UpdateFunc func(ctx *middleware.Context, ing *models.Ingredient) (*models.Ingredient, error)
}

// EditIngredientVM renders an edit ingredient form.
type EditIngredientVM struct {
	form        *forms.Form
	ingredient  *models.Ingredient
	deps        EditDeps
	err         error
	submitting  bool
	nameField   *forms.TextField
	category    *forms.SelectField
	unit        *forms.SelectField
	description *forms.TextField
}

// IngredientUpdatedMsg is sent when the ingredient has been updated.
type IngredientUpdatedMsg struct {
	Ingredient *models.Ingredient
}

// UpdateErrorMsg is sent when updating fails.
type UpdateErrorMsg struct {
	Err error
}

// NewEditIngredientVM builds an EditIngredientVM with fields configured.
func NewEditIngredientVM(ingredient *models.Ingredient, deps EditDeps) *EditIngredientVM {
	if ingredient == nil {
		ingredient = &models.Ingredient{}
	}
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
		forms.WithInitialValue(ingredient.Name),
	)
	categoryField := forms.NewSelectField(
		"Category",
		categoryOptions,
		forms.WithRequired(),
		forms.WithInitialValue(ingredient.Category),
	)
	unitField := forms.NewSelectField(
		"Unit",
		unitOptions,
		forms.WithRequired(),
		forms.WithInitialValue(ingredient.Unit),
	)
	descriptionField := forms.NewTextField(
		"Description",
		forms.WithMaxLength(500),
		forms.WithInitialValue(ingredient.Description),
	)

	form := forms.New(
		deps.FormStyles,
		deps.FormKeys,
		nameField,
		categoryField,
		unitField,
		descriptionField,
	)

	return &EditIngredientVM{
		form:        form,
		deps:        deps,
		ingredient:  ingredient,
		nameField:   nameField,
		category:    categoryField,
		unit:        unitField,
		description: descriptionField,
	}
}

// Init initializes the form.
func (m *EditIngredientVM) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages for the form.
func (m *EditIngredientVM) Update(msg tea.Msg) (*EditIngredientVM, tea.Cmd) {
	switch typed := msg.(type) {
	case UpdateErrorMsg:
		m.submitting = false
		m.err = typed.Err
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
func (m *EditIngredientVM) View() string {
	view := m.form.View()
	if m.err != nil {
		errText := m.deps.FormStyles.Error.Render("Error: " + m.err.Error())
		return strings.Join([]string{errText, "", view}, "\n")
	}
	return view
}

// SetWidth sets the width of the form.
func (m *EditIngredientVM) SetWidth(w int) {
	m.form.SetWidth(w)
}

// IsDirty reports whether the form has been modified.
func (m *EditIngredientVM) IsDirty() bool {
	return m.form.IsDirty()
}

func (m *EditIngredientVM) submit() tea.Cmd {
	if m.submitting {
		return nil
	}
	if err := m.form.Validate(); err != nil {
		m.err = err
		return nil
	}
	if m.deps.UpdateFunc == nil {
		m.err = errors.New("update function not configured")
		return nil
	}
	if m.ingredient == nil {
		m.err = errors.New("ingredient not loaded")
		return nil
	}
	m.err = nil
	m.submitting = true

	updated := &models.Ingredient{
		ID:          m.ingredient.ID,
		Name:        strings.TrimSpace(toString(m.nameField.Value())),
		Category:    toCategory(m.category.Value()),
		Unit:        toUnit(m.unit.Value()),
		Description: strings.TrimSpace(toString(m.description.Value())),
	}

	return func() tea.Msg {
		ingredient, err := m.deps.UpdateFunc(m.deps.Ctx, updated)
		if err != nil {
			return UpdateErrorMsg{Err: err}
		}
		return IngredientUpdatedMsg{Ingredient: ingredient}
	}
}
