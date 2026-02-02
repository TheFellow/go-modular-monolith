package tui

import (
	"errors"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// RenameDeps defines dependencies for the rename menu form.
type RenameDeps struct {
	FormStyles forms.FormStyles
	FormKeys   forms.FormKeys
	Ctx        *middleware.Context
	UpdateFunc func(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error)
}

// RenameMenuVM renders an inline rename form.
type RenameMenuVM struct {
	input      textinput.Model
	menu       *models.Menu
	deps       RenameDeps
	err        error
	submitting bool
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
func NewRenameMenuVM(menu *models.Menu, deps RenameDeps) *RenameMenuVM {
	if menu == nil {
		menu = &models.Menu{}
	}
	input := textinput.New()
	input.Prompt = ""
	input.CharLimit = 100
	input.SetValue(menu.Name)
	input.Focus()

	return &RenameMenuVM{
		input: input,
		menu:  menu,
		deps:  deps,
	}
}

// Init initializes the input.
func (m *RenameMenuVM) Init() tea.Cmd {
	return nil
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
		if key.Matches(typed, m.deps.FormKeys.Submit) || typed.String() == "enter" {
			return m, m.submit()
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the rename form.
func (m *RenameMenuVM) View() string {
	label := m.deps.FormStyles.Label.Render("Name")
	inputView := m.input.View()
	if m.input.Focused() {
		inputView = m.deps.FormStyles.InputFocused.Render(inputView)
	} else {
		inputView = m.deps.FormStyles.Input.Render(inputView)
	}
	view := strings.Join([]string{"Rename Menu", "", label, inputView}, "\n")
	if m.err != nil {
		errText := m.deps.FormStyles.Error.Render("Error: " + m.err.Error())
		return strings.Join([]string{errText, "", view}, "\n")
	}
	return view
}

// SetWidth sets the input width.
func (m *RenameMenuVM) SetWidth(w int) {
	if w <= 0 {
		return
	}
	m.input.Width = w
}

// IsDirty reports whether the input has been modified.
func (m *RenameMenuVM) IsDirty() bool {
	return strings.TrimSpace(m.input.Value()) != strings.TrimSpace(m.menu.Name)
}

func (m *RenameMenuVM) submit() tea.Cmd {
	if m.submitting {
		return nil
	}
	name := strings.TrimSpace(m.input.Value())
	if name == "" {
		m.err = errors.New("name is required")
		return nil
	}
	if m.deps.UpdateFunc == nil {
		m.err = errors.New("update function not configured")
		return nil
	}
	if m.menu == nil {
		m.err = errors.New("menu not loaded")
		return nil
	}
	m.err = nil
	m.submitting = true

	updated := &models.Menu{
		ID:   m.menu.ID,
		Name: name,
	}

	return func() tea.Msg {
		menu, err := m.deps.UpdateFunc(m.deps.Ctx, updated)
		if err != nil {
			return RenameErrorMsg{Err: err}
		}
		return MenuRenamedMsg{Menu: menu}
	}
}
