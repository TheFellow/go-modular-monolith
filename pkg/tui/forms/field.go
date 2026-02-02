package forms

import tea "github.com/charmbracelet/bubbletea"

// Field defines the interface for form fields.
type Field interface {
	// Lifecycle
	Init() tea.Cmd
	Update(msg tea.Msg) (Field, tea.Cmd)
	View() string

	// State
	Focus()
	Blur()
	IsFocused() bool

	// Value
	Value() any
	SetValue(v any) error

	// Validation
	Validate() error
	Error() error

	// Display
	Label() string
	SetWidth(w int)
}

// FieldOption configures a field.
type FieldOption func(Field)

func applyFieldOptions(field Field, opts []FieldOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(field)
		}
	}
}

// WithPlaceholder sets the placeholder text when supported.
func WithPlaceholder(p string) FieldOption {
	return func(field Field) {
		switch typed := field.(type) {
		case *TextField:
			typed.input.Placeholder = p
		case *NumberField:
			typed.input.Placeholder = p
		}
	}
}

// WithValidator appends a validator when supported.
func WithValidator(v Validator) FieldOption {
	return func(field Field) {
		if v == nil {
			return
		}
		switch typed := field.(type) {
		case *TextField:
			typed.validators = append(typed.validators, v)
		case *NumberField:
			typed.validators = append(typed.validators, v)
		case *SelectField:
			typed.validators = append(typed.validators, v)
		}
	}
}

// WithRequired marks a field as required and adds a Required validator.
func WithRequired() FieldOption {
	return func(field Field) {
		switch typed := field.(type) {
		case *TextField:
			typed.required = true
			typed.validators = append(typed.validators, Required())
		case *NumberField:
			typed.required = true
			typed.validators = append(typed.validators, Required())
		case *SelectField:
			typed.required = true
			typed.validators = append(typed.validators, Required())
		}
	}
}

// WithMaxLength sets a max length validator for text fields.
func WithMaxLength(n int) FieldOption {
	return func(field Field) {
		if n <= 0 {
			return
		}
		switch typed := field.(type) {
		case *TextField:
			typed.validators = append(typed.validators, MaxLength(n))
			typed.input.CharLimit = n
		}
	}
}
