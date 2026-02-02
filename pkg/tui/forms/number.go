package forms

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NumberField is a numeric input field.
type NumberField struct {
	label              string
	input              textinput.Model
	validators         []Validator
	precision          int
	allowNeg           bool
	err                error
	styles             FieldStyles
	labelRequiredStyle lipgloss.Style
	required           bool
	focused            bool
	width              int
}

// NewNumberField creates a new NumberField.
func NewNumberField(label string, opts ...FieldOption) *NumberField {
	input := textinput.New()
	input.Prompt = ""
	field := &NumberField{
		label:     label,
		input:     input,
		precision: -1,
	}
	applyFieldOptions(field, opts)
	return field
}

// WithPrecision limits the number of decimal places.
func WithPrecision(p int) FieldOption {
	return func(field Field) {
		if p < 0 {
			return
		}
		typed, ok := field.(*NumberField)
		if !ok {
			return
		}
		typed.precision = p
	}
}

// WithMin adds a minimum numeric validator.
func WithMin(min float64) FieldOption {
	return func(field Field) {
		typed, ok := field.(*NumberField)
		if !ok {
			return
		}
		typed.validators = append(typed.validators, Min(min))
	}
}

// WithMax adds a maximum numeric validator.
func WithMax(max float64) FieldOption {
	return func(field Field) {
		typed, ok := field.(*NumberField)
		if !ok {
			return
		}
		typed.validators = append(typed.validators, Max(max))
	}
}

// WithAllowNegative allows negative numbers.
func WithAllowNegative() FieldOption {
	return func(field Field) {
		typed, ok := field.(*NumberField)
		if !ok {
			return
		}
		typed.allowNeg = true
	}
}

// Init initializes the field.
func (n *NumberField) Init() tea.Cmd {
	return nil
}

// Update updates the field with a message.
func (n *NumberField) Update(msg tea.Msg) (Field, tea.Cmd) {
	var cmd tea.Cmd
	n.input, cmd = n.input.Update(msg)
	return n, cmd
}

// View renders the field.
func (n *NumberField) View() string {
	label := n.label
	if n.required {
		label = label + " *"
		label = n.labelRequiredStyle.Render(label)
	} else {
		label = n.styles.Label.Render(label)
	}

	inputView := n.input.View()
	if n.focused {
		inputView = n.styles.InputFocused.Render(inputView)
	} else {
		inputView = n.styles.Input.Render(inputView)
	}

	lines := []string{label, inputView}
	if n.err != nil {
		lines = append(lines, n.styles.Error.Render(n.err.Error()))
	}
	return strings.Join(lines, "\n")
}

// Focus focuses the field.
func (n *NumberField) Focus() {
	n.focused = true
	n.input.Focus()
}

// Blur blurs the field.
func (n *NumberField) Blur() {
	n.focused = false
	n.input.Blur()
}

// IsFocused returns true if the field is focused.
func (n *NumberField) IsFocused() bool {
	return n.focused
}

// Value returns the numeric value when possible.
func (n *NumberField) Value() any {
	raw := strings.TrimSpace(n.input.Value())
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return value
}

// SetValue sets the field value.
func (n *NumberField) SetValue(v any) error {
	if v == nil {
		n.input.SetValue("")
		return nil
	}
	switch typed := v.(type) {
	case float64:
		n.input.SetValue(n.formatFloat(typed))
	case float32:
		n.input.SetValue(n.formatFloat(float64(typed)))
	case int:
		n.input.SetValue(strconv.Itoa(typed))
	case int64:
		n.input.SetValue(strconv.FormatInt(typed, 10))
	case int32:
		n.input.SetValue(strconv.FormatInt(int64(typed), 10))
	case int16:
		n.input.SetValue(strconv.FormatInt(int64(typed), 10))
	case int8:
		n.input.SetValue(strconv.FormatInt(int64(typed), 10))
	case uint:
		n.input.SetValue(strconv.FormatUint(uint64(typed), 10))
	case uint64:
		n.input.SetValue(strconv.FormatUint(typed, 10))
	case uint32:
		n.input.SetValue(strconv.FormatUint(uint64(typed), 10))
	case uint16:
		n.input.SetValue(strconv.FormatUint(uint64(typed), 10))
	case uint8:
		n.input.SetValue(strconv.FormatUint(uint64(typed), 10))
	case string:
		n.input.SetValue(typed)
	case []byte:
		n.input.SetValue(string(typed))
	default:
		n.input.SetValue(fmt.Sprint(v))
	}
	return nil
}

// Validate runs field validators.
func (n *NumberField) Validate() error {
	n.err = nil
	raw := strings.TrimSpace(n.input.Value())
	if raw == "" {
		return n.runValidators(nil)
	}
	if !n.allowNeg && strings.HasPrefix(raw, "-") {
		n.err = errors.New("must be positive")
		return n.err
	}
	if n.precision >= 0 {
		if !precisionValid(raw, n.precision) {
			n.err = fmt.Errorf("maximum precision is %d", n.precision)
			return n.err
		}
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		n.err = errors.New("invalid number")
		return n.err
	}
	return n.runValidators(value)
}

// Error returns the last validation error.
func (n *NumberField) Error() error {
	return n.err
}

// Label returns the field label.
func (n *NumberField) Label() string {
	return n.label
}

// SetWidth sets the field width.
func (n *NumberField) SetWidth(w int) {
	n.width = w
	n.input.Width = w
}

func (n *NumberField) setStyles(styles FieldStyles) {
	n.styles = styles
}

func (n *NumberField) setLabelRequiredStyle(style lipgloss.Style) {
	n.labelRequiredStyle = style
}

func (n *NumberField) runValidators(value any) error {
	for _, validator := range n.validators {
		if err := validator(value); err != nil {
			n.err = err
			return err
		}
	}
	n.err = nil
	return nil
}

func (n *NumberField) formatFloat(value float64) string {
	if n.precision >= 0 {
		return fmt.Sprintf("%.*f", n.precision, value)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func precisionValid(raw string, max int) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return true
	}
	if strings.ContainsAny(trimmed, "eE") {
		return false
	}
	if trimmed[0] == '-' {
		trimmed = trimmed[1:]
	}
	parts := strings.SplitN(trimmed, ".", 2)
	if len(parts) != 2 {
		return true
	}
	return len(parts[1]) <= max
}
