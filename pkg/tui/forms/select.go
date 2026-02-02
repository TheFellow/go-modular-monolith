package forms

import (
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectField is a field for selecting from options.
type SelectField struct {
	label              string
	options            []SelectOption
	selected           int
	open               bool
	focused            bool
	validators         []Validator
	err                error
	styles             FieldStyles
	labelRequiredStyle lipgloss.Style
	required           bool
	width              int
}

// SelectOption represents an option in a SelectField.
type SelectOption struct {
	Label string
	Value any
}

// NewSelectField creates a new SelectField.
func NewSelectField(label string, options []SelectOption, opts ...FieldOption) *SelectField {
	copied := make([]SelectOption, len(options))
	copy(copied, options)
	field := &SelectField{
		label:   label,
		options: copied,
	}
	applyFieldOptions(field, opts)
	return field
}

// Init initializes the field.
func (s *SelectField) Init() tea.Cmd {
	return nil
}

// Update updates the field with a message.
func (s *SelectField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if !s.focused {
		return s, nil
	}
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch keyMsg.String() {
	case "up":
		s.move(-1)
	case "down":
		s.move(1)
	case "enter":
		s.open = !s.open
	}
	return s, nil
}

// View renders the field.
func (s *SelectField) View() string {
	label := s.label
	if s.required {
		label = label + " *"
		label = s.labelRequiredStyle.Render(label)
	} else {
		label = s.styles.Label.Render(label)
	}

	lines := []string{label}
	if s.open {
		for i, option := range s.options {
			prefix := "  "
			style := s.styles.Input
			if i == s.selected {
				prefix = "> "
				style = s.styles.InputFocused
			}
			lines = append(lines, style.Render(prefix+option.Label))
		}
	} else {
		value := s.selectedLabel()
		inputView := value
		if s.focused {
			inputView = s.styles.InputFocused.Render(inputView)
		} else {
			inputView = s.styles.Input.Render(inputView)
		}
		lines = append(lines, inputView)
	}
	if s.err != nil {
		lines = append(lines, s.styles.Error.Render(s.err.Error()))
	}
	return strings.Join(lines, "\n")
}

// Focus focuses the field.
func (s *SelectField) Focus() {
	s.focused = true
	s.open = true
}

// Blur blurs the field.
func (s *SelectField) Blur() {
	s.focused = false
	s.open = false
}

// IsFocused returns true if the field is focused.
func (s *SelectField) IsFocused() bool {
	return s.focused
}

// Value returns the selected option value.
func (s *SelectField) Value() any {
	if s.selected < 0 || s.selected >= len(s.options) {
		return nil
	}
	return s.options[s.selected].Value
}

// SetValue sets the selected value.
func (s *SelectField) SetValue(v any) error {
	s.selected = s.findOptionIndex(v)
	return nil
}

// Validate runs field validators.
func (s *SelectField) Validate() error {
	s.err = nil
	value := s.Value()
	for _, validator := range s.validators {
		if err := validator(value); err != nil {
			s.err = err
			return err
		}
	}
	return nil
}

// Error returns the last validation error.
func (s *SelectField) Error() error {
	return s.err
}

// Label returns the field label.
func (s *SelectField) Label() string {
	return s.label
}

// SetWidth sets the field width.
func (s *SelectField) SetWidth(w int) {
	s.width = w
}

func (s *SelectField) setStyles(styles FieldStyles) {
	s.styles = styles
}

func (s *SelectField) setLabelRequiredStyle(style lipgloss.Style) {
	s.labelRequiredStyle = style
}

func (s *SelectField) move(delta int) {
	if len(s.options) == 0 {
		return
	}
	s.selected += delta
	if s.selected < 0 {
		s.selected = len(s.options) - 1
	}
	if s.selected >= len(s.options) {
		s.selected = 0
	}
}

func (s *SelectField) selectedLabel() string {
	if s.selected < 0 || s.selected >= len(s.options) {
		return ""
	}
	return s.options[s.selected].Label
}

func (s *SelectField) findOptionIndex(v any) int {
	if len(s.options) == 0 {
		return 0
	}
	switch typed := v.(type) {
	case int:
		if typed >= 0 && typed < len(s.options) {
			return typed
		}
	case SelectOption:
		for i, option := range s.options {
			if reflect.DeepEqual(option.Value, typed.Value) || option.Label == typed.Label {
				return i
			}
		}
	case string:
		for i, option := range s.options {
			if option.Label == typed {
				return i
			}
			if value, ok := option.Value.(string); ok && value == typed {
				return i
			}
		}
	default:
		for i, option := range s.options {
			if reflect.DeepEqual(option.Value, v) {
				return i
			}
		}
	}
	return 0
}
