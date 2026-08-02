package components

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// TaggedConfirm adds a compact, optional complete-tag-set step before a
// destructive or lifecycle confirmation. ctrl+s advances to the confirmation.
type TaggedConfirm[T any] struct {
	field      *forms.TextField
	form       *forms.Form
	dialog     *dialog.ConfirmDialog
	confirming bool
	width      int
	err        error
	parse      func(string) (T, error)
}

func NewTaggedConfirm[T any](current string, parse func(string) (T, error), confirm *dialog.ConfirmDialog) *TaggedConfirm[T] {
	field := NewOptionalTagsField(current)
	return &TaggedConfirm[T]{field: field, form: forms.New(styles.Standard.Form, keys.Standard.Form, field), dialog: confirm, parse: parse}
}

func (m *TaggedConfirm[T]) Init() tea.Cmd { return m.form.Init() }
func (m *TaggedConfirm[T]) FormEditing() bool {
	return !m.confirming && m.form.IsEditing()
}
func (m *TaggedConfirm[T]) Update(msg tea.Msg) (*TaggedConfirm[T], tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.SetWidth(size.Width)
	}
	if m.confirming {
		var cmd tea.Cmd
		m.dialog, cmd = m.dialog.Update(msg)
		return m, cmd
	}
	if typed, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(typed, keys.Standard.Form.Cancel) {
			return m, func() tea.Msg { return dialog.CancelMsg{} }
		}
		if key.Matches(typed, keys.Standard.Form.Submit) {
			if _, err := DesiredTags(m.field, m.parse); err != nil {
				m.err = err
				return m, nil
			}
			m.err = nil
			m.confirming = true
			m.field.Blur()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}
func (m *TaggedConfirm[T]) View() string {
	if m.confirming {
		return m.dialog.View()
	}
	parts := []string{m.form.View()}
	if m.err != nil {
		parts = append(parts, "", "Error: "+m.err.Error())
	}
	return strings.Join(append(parts, "", "ctrl+s continue • esc cancel"), "\n")
}
func (m *TaggedConfirm[T]) SetWidth(width int) {
	m.width = width
	m.form.SetWidth(max(width-8, 20))
	m.dialog.SetWidth(width)
}
func (m *TaggedConfirm[T]) DesiredTags() (*T, error) {
	if m == nil {
		return nil, nil
	}
	return DesiredTags(m.field, m.parse)
}
