package forms

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fieldStyleSetter interface {
	setStyles(styles FieldStyles)
}

type labelRequiredStyleSetter interface {
	setLabelRequiredStyle(style lipgloss.Style)
}

// Form manages a set of fields and their interactions.
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

// New creates a new form with the provided styles, keys, and fields.
func New(styles FormStyles, keys FormKeys, fields ...Field) *Form {
	form := &Form{
		fields:  fields,
		focused: 0,
		styles:  styles,
		keys:    keys,
	}
	form.applyStyles()
	return form
}

// Init initializes the form and focuses the first field.
func (f *Form) Init() tea.Cmd {
	f.applyStyles()
	if len(f.fields) == 0 {
		return nil
	}
	if f.focused < 0 || f.focused >= len(f.fields) {
		f.focused = 0
	}
	f.fields[f.focused].Focus()
	cmds := make([]tea.Cmd, 0, len(f.fields))
	for i := range f.fields {
		if cmd := f.fields[i].Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// Update processes messages and updates the focused field.
func (f *Form) Update(msg tea.Msg) (*Form, tea.Cmd) {
	if len(f.fields) == 0 {
		return f, nil
	}
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		f.SetWidth(typed.Width)
		return f, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(typed, f.keys.NextField):
			f.FocusNext()
			return f, nil
		case key.Matches(typed, f.keys.PrevField):
			f.FocusPrev()
			return f, nil
		case key.Matches(typed, f.keys.Submit):
			f.submitted = true
			f.err = f.Validate()
			return f, nil
		case key.Matches(typed, f.keys.Cancel):
			return f, nil
		}
	}
	if f.focused < 0 || f.focused >= len(f.fields) {
		return f, nil
	}
	updated, cmd := f.fields[f.focused].Update(msg)
	if updated != nil {
		f.fields[f.focused] = updated
	}
	if _, ok := msg.(tea.KeyMsg); ok {
		f.dirty = true
	}
	return f, cmd
}

// View renders the form.
func (f *Form) View() string {
	if len(f.fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(f.fields))
	for _, field := range f.fields {
		parts = append(parts, field.View())
	}
	view := strings.Join(parts, "\n\n")
	return f.styles.Form.Render(view)
}

// Validate validates all fields and returns the first error.
func (f *Form) Validate() error {
	var firstErr error
	for _, field := range f.fields {
		if err := field.Validate(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	f.err = firstErr
	return firstErr
}

// IsDirty returns true if any field has been updated.
func (f *Form) IsDirty() bool {
	return f.dirty
}

// SetWidth sets the width on the form and all fields.
func (f *Form) SetWidth(w int) {
	f.width = w
	for _, field := range f.fields {
		field.SetWidth(w)
	}
}

// FocusNext moves focus to the next field.
func (f *Form) FocusNext() {
	if len(f.fields) == 0 {
		return
	}
	if f.focused < 0 || f.focused >= len(f.fields) {
		f.focused = 0
		f.fields[f.focused].Focus()
		return
	}
	f.fields[f.focused].Blur()
	f.focused = (f.focused + 1) % len(f.fields)
	f.fields[f.focused].Focus()
}

// FocusPrev moves focus to the previous field.
func (f *Form) FocusPrev() {
	if len(f.fields) == 0 {
		return
	}
	if f.focused < 0 || f.focused >= len(f.fields) {
		f.focused = 0
		f.fields[f.focused].Focus()
		return
	}
	f.fields[f.focused].Blur()
	f.focused--
	if f.focused < 0 {
		f.focused = len(f.fields) - 1
	}
	f.fields[f.focused].Focus()
}

// FocusedField returns the currently focused field.
func (f *Form) FocusedField() Field {
	if f.focused < 0 || f.focused >= len(f.fields) {
		return nil
	}
	return f.fields[f.focused]
}

func (f *Form) applyStyles() {
	fieldStyles := FieldStyles{
		Label:        f.styles.Label,
		Input:        f.styles.Input,
		InputFocused: f.styles.InputFocused,
		Error:        f.styles.Error,
	}
	for _, field := range f.fields {
		if styler, ok := field.(fieldStyleSetter); ok {
			styler.setStyles(fieldStyles)
		}
		if required, ok := field.(labelRequiredStyleSetter); ok {
			required.setLabelRequiredStyle(f.styles.LabelRequired)
		}
	}
}
