package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// stepsField keeps one preparation step per line without inventing delimiters
// that could alter accepted instructions. Enter inserts a line; Tab accepts.
type stepsField struct {
	label   string
	input   textarea.Model
	focused bool
}

func newStepsField(label, value string) *stepsField {
	input := textarea.New()
	input.CharLimit = 0
	input.SetHeight(4)
	input.SetValue(value)
	return &stepsField{label: label, input: input}
}
func (f *stepsField) Init() tea.Cmd { return nil }
func (f *stepsField) Update(msg tea.Msg) (forms.Field, tea.Cmd) {
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return f, cmd
}
func (f *stepsField) View() string {
	style := styles.Standard.Form.Input
	if f.focused {
		style = styles.Standard.Form.InputFocused
	}
	return styles.Standard.Form.Label.Render(f.label) + "\n" + style.Render(f.input.View())
}
func (f *stepsField) Focus()             { f.focused = true }
func (f *stepsField) Blur()              { f.focused = false; f.input.Blur() }
func (f *stepsField) IsFocused() bool    { return f.focused }
func (f *stepsField) BeginEdit() tea.Cmd { return f.input.Focus() }
func (f *stepsField) EndEdit()           { f.input.Blur() }
func (f *stepsField) OwnsAccept() bool   { return true }
func (f *stepsField) OwnsNavigation(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyUp || msg.Type == tea.KeyDown
}
func (f *stepsField) Value() any               { return f.input.Value() }
func (f *stepsField) SetValue(value any) error { f.input.SetValue(fmt.Sprint(value)); return nil }
func (f *stepsField) Validate() error          { return nil }
func (f *stepsField) Error() error             { return nil }
func (f *stepsField) Label() string            { return f.label }
func (f *stepsField) SetWidth(width int)       { f.input.SetWidth(max(10, width-4)) }
