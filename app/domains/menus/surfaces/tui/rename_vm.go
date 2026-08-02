package tui

import (
	"errors"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// RenameMenuVM renders an inline menu update form.
type RenameMenuVM struct {
	app         *app.Session
	form        *forms.Form
	name        *forms.TextField
	description *forms.TextField
	tags        *forms.TextField
	menu        *models.Menu
	styles      forms.FormStyles
	keys        forms.FormKeys
	err         error
	submitting  bool
}

// MenuRenamedMsg is sent when the menu has been renamed.
type MenuRenamedMsg struct {
	Menu *models.Menu
}

// RenameErrorMsg is sent when renaming fails.
type RenameErrorMsg struct {
	Err error
}

// NewRenameMenuVM builds a RenameMenuVM with input configured.
func NewRenameMenuVM(app *app.Session, menu *models.Menu) *RenameMenuVM {
	if menu == nil {
		menu = &models.Menu{}
	}
	name := forms.NewTextField("Name", forms.WithRequired(), forms.WithMaxLength(100), forms.WithInitialValue(menu.Name))
	description := forms.NewTextField("Description", forms.WithMaxLength(500), forms.WithInitialValue(menu.Description))
	tags := components.NewOptionalTagsField(menu.Tags.Canonical().String())
	formStyles := styles.Standard.Form
	formKeys := keys.Standard.Form

	return &RenameMenuVM{
		app: app, form: forms.New(formStyles, formKeys, name, description, tags), name: name, description: description, tags: tags,
		menu: menu, styles: formStyles, keys: formKeys,
	}
}

// Init initializes the input.
func (m *RenameMenuVM) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages for the rename form.
func (m *RenameMenuVM) Update(msg tea.Msg) (*RenameMenuVM, tea.Cmd) {
	switch typed := msg.(type) {
	case RenameErrorMsg:
		m.submitting = false
		m.err = typed.Err
		return m, nil
	case MenuRenamedMsg:
		m.submitting = false
		m.err = nil
		return m, nil
	case tea.KeyMsg:
		if key.Matches(typed, m.keys.Submit) || typed.String() == "enter" {
			return m, m.submit()
		}
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

// View renders the rename form.
func (m *RenameMenuVM) View() string {
	view := strings.Join([]string{"Edit Menu", "", m.form.View(), "", "Leave description blank to preserve it."}, "\n")
	if m.err != nil {
		errText := m.styles.Error.Render("Error: " + m.err.Error())
		return strings.Join([]string{errText, "", view}, "\n")
	}
	return view
}

// SetWidth sets the input width.
func (m *RenameMenuVM) SetWidth(w int) {
	if w <= 0 {
		return
	}
	m.form.SetWidth(w)
}

// IsDirty reports whether the input has been modified.
func (m *RenameMenuVM) IsDirty() bool {
	return m.form.IsDirty()
}

func (m *RenameMenuVM) submit() tea.Cmd {
	if m.submitting {
		return nil
	}
	name := strings.TrimSpace(toString(m.name.Value()))
	if name == "" {
		m.err = errors.New("name is required")
		return nil
	}
	if m.menu == nil {
		m.err = errors.New("menu not loaded")
		return nil
	}
	desired, err := components.DesiredTags(m.tags, tag.ParseCollection)
	if err != nil {
		m.err = err
		return nil
	}
	m.err = nil
	m.submitting = true

	updated := &models.Menu{
		ID:          m.menu.ID,
		Name:        name,
		Description: strings.TrimSpace(toString(m.description.Value())),
	}

	return func() tea.Msg {
		menu, err := app.RunTaggedMutation(m.app.App, m.context(), desired, func(ctx *middleware.Context) (*models.Menu, error) {
			return m.app.Menus.Update(ctx, updated)
		})
		if err != nil {
			return RenameErrorMsg{Err: err}
		}
		return MenuRenamedMsg{Menu: menu}
	}
}

func (m *RenameMenuVM) context() *middleware.Context {
	return m.app.Context()
}
