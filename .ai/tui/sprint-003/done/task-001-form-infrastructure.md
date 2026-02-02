# Task 001: Form Infrastructure

## Goal

Create reusable form infrastructure in `pkg/tui/forms/` that all domain forms will use.

## Files to Create

```
pkg/tui/forms/
├── form.go       # Base Form model with field navigation, validation, dirty state
├── field.go      # Field interface and base field types
├── text.go       # TextField wrapping bubbles/textinput
├── select.go     # SelectField for enum/option selection
├── number.go     # NumberField for numeric input
├── validation.go # Common validators (Required, MinLength, etc.)
├── styles.go     # FormStyles type
└── keys.go       # FormKeys type
```

## Design Principles

1. **Reusable from the start** - All types in `pkg/tui/forms/` not domain-specific
2. **Wrap bubbles** - Use `bubbles/textinput` internally, expose simpler API
3. **Follow pkg/tui pattern** - Styles/Keys passed in, not hardcoded

## Implementation

### Form Model

```go
// pkg/tui/forms/form.go
package forms

type Form struct {
    fields    []Field
    focused   int
    submitted bool
    dirty     bool
    err       error
    width     int
    styles    FormStyles
    keys      FormKeys
}

func New(styles FormStyles, keys FormKeys, fields ...Field) *Form

func (f *Form) Init() tea.Cmd
func (f *Form) Update(msg tea.Msg) (*Form, tea.Cmd)
func (f *Form) View() string
func (f *Form) Validate() error
func (f *Form) IsDirty() bool
func (f *Form) SetWidth(w int)

// Navigation
func (f *Form) FocusNext()
func (f *Form) FocusPrev()
func (f *Form) FocusedField() Field
```

### Field Interface

```go
// pkg/tui/forms/field.go
package forms

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
```

### TextField

```go
// pkg/tui/forms/text.go
package forms

type TextField struct {
    label      string
    input      textinput.Model
    validators []Validator
    err        error
    width      int
    styles     FieldStyles
}

func NewTextField(label string, opts ...FieldOption) *TextField

// FieldOption pattern for configuration
type FieldOption func(Field)

func WithPlaceholder(p string) FieldOption
func WithValidator(v Validator) FieldOption
func WithRequired() FieldOption
func WithMaxLength(n int) FieldOption
```

### SelectField

```go
// pkg/tui/forms/select.go
package forms

type SelectField struct {
    label    string
    options  []SelectOption
    selected int
    open     bool
    focused  bool
    styles   FieldStyles
}

type SelectOption struct {
    Label string
    Value any
}

func NewSelectField(label string, options []SelectOption, opts ...FieldOption) *SelectField

// Navigation: up/down to change selection when focused
// Enter to confirm selection
```

### NumberField

```go
// pkg/tui/forms/number.go
package forms

type NumberField struct {
    label      string
    input      textinput.Model
    validators []Validator
    precision  int  // decimal places
    allowNeg   bool
    err        error
    styles     FieldStyles
}

func NewNumberField(label string, opts ...FieldOption) *NumberField

func WithPrecision(p int) FieldOption
func WithMin(min float64) FieldOption
func WithMax(max float64) FieldOption
func WithAllowNegative() FieldOption
```

### Validators

```go
// pkg/tui/forms/validation.go
package forms

type Validator func(value any) error

func Required() Validator
func MinLength(n int) Validator
func MaxLength(n int) Validator
func Pattern(regex string) Validator
func Min(n float64) Validator
func Max(n float64) Validator
```

### Styles and Keys

```go
// pkg/tui/forms/styles.go
package forms

type FormStyles struct {
    Form          lipgloss.Style
    Label         lipgloss.Style
    LabelRequired lipgloss.Style  // Label with * indicator
    Input         lipgloss.Style
    InputFocused  lipgloss.Style
    Error         lipgloss.Style
    Help          lipgloss.Style
}

type FieldStyles struct {
    Label        lipgloss.Style
    Input        lipgloss.Style
    InputFocused lipgloss.Style
    Error        lipgloss.Style
}

// pkg/tui/forms/keys.go
package forms

type FormKeys struct {
    NextField key.Binding  // tab
    PrevField key.Binding  // shift+tab
    Submit    key.Binding  // ctrl+s / enter on last field
    Cancel    key.Binding  // esc
}
```

## Notes

- TextField wraps `bubbles/textinput` - don't reinvent
- SelectField is custom (bubbles doesn't have a simple select)
- NumberField uses TextField internally with numeric validation
- All fields support `SetWidth()` for responsive layout
- Validators return `error` for validation messages

## Checklist

- [x] Create `pkg/tui/forms/` directory
- [x] Implement `form.go` with Form model
- [x] Implement `field.go` with Field interface
- [x] Implement `text.go` with TextField
- [x] Implement `select.go` with SelectField
- [x] Implement `number.go` with NumberField
- [x] Implement `validation.go` with validators
- [x] Implement `styles.go` and `keys.go`
- [x] Add basic tests for form navigation and validation
- [x] `go build ./pkg/tui/forms/...` passes
- [x] `go test ./pkg/tui/forms/...` passes
