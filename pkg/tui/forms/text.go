package forms

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TextField is a text input field.
type TextField struct {
	label              string
	input              textinput.Model
	validators         []Validator
	err                error
	width              int
	styles             FieldStyles
	labelRequiredStyle lipgloss.Style
	required           bool
	focused            bool
}

// NewTextField creates a new TextField.
func NewTextField(label string, opts ...FieldOption) *TextField {
	input := textinput.New()
	input.Prompt = ""
	field := &TextField{
		label: label,
		input: input,
	}
	applyFieldOptions(field, opts)
	return field
}

// Init initializes the field.
func (t *TextField) Init() tea.Cmd {
	return nil
}

// Update updates the field with a message.
func (t *TextField) Update(msg tea.Msg) (Field, tea.Cmd) {
	var cmd tea.Cmd
	t.input, cmd = t.input.Update(msg)
	return t, cmd
}

// View renders the field.
func (t *TextField) View() string {
	label := t.label
	if t.required {
		label = label + " *"
		label = t.labelRequiredStyle.Render(label)
	} else {
		label = t.styles.Label.Render(label)
	}

	inputView := t.input.View()
	if t.focused {
		inputView = t.styles.InputFocused.Render(inputView)
	} else {
		inputView = t.styles.Input.Render(inputView)
	}

	lines := []string{label, inputView}
	if t.err != nil {
		lines = append(lines, t.styles.Error.Render(t.err.Error()))
	}
	return strings.Join(lines, "\n")
}

// Focus focuses the field.
func (t *TextField) Focus() {
	t.focused = true
	t.input.Focus()
}

// Blur blurs the field.
func (t *TextField) Blur() {
	t.focused = false
	t.input.Blur()
}

// IsFocused returns true if the field is focused.
func (t *TextField) IsFocused() bool {
	return t.focused
}

// Value returns the field value.
func (t *TextField) Value() any {
	return t.input.Value()
}

// SetValue sets the field value.
func (t *TextField) SetValue(v any) error {
	if v == nil {
		t.input.SetValue("")
		return nil
	}
	switch typed := v.(type) {
	case string:
		t.input.SetValue(typed)
	case []byte:
		t.input.SetValue(string(typed))
	default:
		t.input.SetValue(fmt.Sprint(v))
	}
	return nil
}

// Validate runs field validators.
func (t *TextField) Validate() error {
	t.err = nil
	value := t.input.Value()
	for _, validator := range t.validators {
		if err := validator(value); err != nil {
			t.err = err
			return err
		}
	}
	return nil
}

// Error returns the last validation error.
func (t *TextField) Error() error {
	return t.err
}

// Label returns the field label.
func (t *TextField) Label() string {
	return t.label
}

// SetWidth sets the field width.
func (t *TextField) SetWidth(w int) {
	t.width = w
	t.input.Width = w
}

func (t *TextField) setStyles(styles FieldStyles) {
	t.styles = styles
}

func (t *TextField) setLabelRequiredStyle(style lipgloss.Style) {
	t.labelRequiredStyle = style
}
