package tui

import (
	"errors"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// EditDrinkVM renders an edit drink form.
type EditDrinkVM struct {
	app         *app.App
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
func NewEditDrinkVM(app *app.App, drink *models.Drink) *EditDrinkVM {
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

	formStyles := tuistyles.App.Form
	formKeys := tuikeys.App.Form
	form := forms.New(
		formStyles,
		formKeys,
		nameField,
		categoryField,
		glassField,
		descriptionField,
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
	}
}

// Init initializes the form.
func (m *EditDrinkVM) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages for the form.
func (m *EditDrinkVM) Update(msg tea.Msg) (*EditDrinkVM, tea.Cmd) {
	switch typed := msg.(type) {
	case UpdateErrorMsg:
		m.submitting = false
		m.err = typed.Err
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
	if m.err != nil {
		errText := m.styles.Error.Render("Error: " + m.err.Error())
		return strings.Join([]string{errText, "", view}, "\n")
	}
	return view
}

// SetWidth sets the width of the form.
func (m *EditDrinkVM) SetWidth(w int) {
	m.form.SetWidth(w)
}

// IsDirty reports whether the form has been modified.
func (m *EditDrinkVM) IsDirty() bool {
	return m.form.IsDirty()
}

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
	m.err = nil
	m.submitting = true

	updated := *m.drink
	updated.Name = strings.TrimSpace(toString(m.nameField.Value()))
	updated.Category = toDrinkCategory(m.category.Value())
	updated.Glass = toGlassType(m.glass.Value())
	updated.Description = strings.TrimSpace(toString(m.description.Value()))

	return func() tea.Msg {
		drink, err := m.app.Drinks.Update(m.context(), &updated)
		if err != nil {
			return UpdateErrorMsg{Err: err}
		}
		return DrinkUpdatedMsg{Drink: drink}
	}
}

func (m *EditDrinkVM) context() *middleware.Context {
	return m.app.Context()
}
