package tui

import (
	"context"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// CreateMenuVM renders a create menu form.
type CreateMenuVM struct {
	app         *app.App
	principal   cedar.EntityUID
	form        *forms.Form
	styles      forms.FormStyles
	keys        forms.FormKeys
	err         error
	submitting  bool
	nameField   *forms.TextField
	description *forms.TextField
}

// MenuCreatedMsg is sent when the menu has been created.
type MenuCreatedMsg struct {
	Menu *models.Menu
}

// CreateErrorMsg is sent when creation fails.
type CreateErrorMsg struct {
	Err error
}

// NewCreateMenuVM builds a CreateMenuVM with fields configured.
func NewCreateMenuVM(app *app.App, principal cedar.EntityUID) *CreateMenuVM {
	nameField := forms.NewTextField(
		"Name",
		forms.WithRequired(),
		forms.WithMaxLength(100),
		forms.WithPlaceholder("e.g., Summer Cocktails"),
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
		descriptionField,
	)

	return &CreateMenuVM{
		app:         app,
		principal:   principal,
		form:        form,
		styles:      formStyles,
		keys:        formKeys,
		nameField:   nameField,
		description: descriptionField,
	}
}

// Init initializes the form.
func (m *CreateMenuVM) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages for the form.
func (m *CreateMenuVM) Update(msg tea.Msg) (*CreateMenuVM, tea.Cmd) {
	switch typed := msg.(type) {
	case CreateErrorMsg:
		m.submitting = false
		m.err = typed.Err
		return m, nil
	case MenuCreatedMsg:
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
func (m *CreateMenuVM) View() string {
	view := m.form.View()
	if m.err != nil {
		errText := m.styles.Error.Render("Error: " + m.err.Error())
		return strings.Join([]string{errText, "", view}, "\n")
	}
	return view
}

// SetWidth sets the width of the form.
func (m *CreateMenuVM) SetWidth(w int) {
	m.form.SetWidth(w)
}

// IsDirty reports whether the form has been modified.
func (m *CreateMenuVM) IsDirty() bool {
	return m.form.IsDirty()
}

func (m *CreateMenuVM) submit() tea.Cmd {
	if m.submitting {
		return nil
	}
	if err := m.form.Validate(); err != nil {
		m.err = err
		return nil
	}
	m.err = nil
	m.submitting = true

	menu := &models.Menu{
		Name:        strings.TrimSpace(toString(m.nameField.Value())),
		Description: strings.TrimSpace(toString(m.description.Value())),
	}

	return func() tea.Msg {
		created, err := m.app.Menu.Create(m.context(), menu)
		if err != nil {
			return CreateErrorMsg{Err: err}
		}
		return MenuCreatedMsg{Menu: created}
	}
}

func (m *CreateMenuVM) context() *middleware.Context {
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
