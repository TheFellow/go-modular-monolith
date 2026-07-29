package components

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/main/tui/keys"
	"github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// TaggedConfirm adds a compact, optional complete-tag-set step before a
// destructive or lifecycle confirmation. ctrl+s advances to the confirmation.
type TaggedConfirm struct {
	field      *forms.TextField
	form       *forms.Form
	dialog     *dialog.ConfirmDialog
	confirming bool
	width      int
	err        error
}

func NewTaggedConfirm(current tag.Tags, confirm *dialog.ConfirmDialog) *TaggedConfirm {
	field := NewOptionalTagsField(current)
	return &TaggedConfirm{field: field, form: forms.New(styles.App.Form, keys.App.Form, field), dialog: confirm}
}

func (m *TaggedConfirm) Init() tea.Cmd { return m.form.Init() }
func (m *TaggedConfirm) Update(msg tea.Msg) (*TaggedConfirm, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.SetWidth(size.Width)
	}
	if m.confirming {
		var cmd tea.Cmd
		m.dialog, cmd = m.dialog.Update(msg)
		return m, cmd
	}
	if typed, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(typed, keys.App.Form.Cancel) {
			return m, func() tea.Msg { return dialog.CancelMsg{} }
		}
		if key.Matches(typed, keys.App.Form.Submit) {
			if _, err := DesiredTags(m.field); err != nil {
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
func (m *TaggedConfirm) View() string {
	if m.confirming {
		return m.dialog.View()
	}
	parts := []string{m.form.View()}
	if m.err != nil {
		parts = append(parts, "", "Error: "+m.err.Error())
	}
	return strings.Join(append(parts, "", "ctrl+s continue • esc cancel"), "\n")
}
func (m *TaggedConfirm) SetWidth(width int) {
	m.width = width
	m.form.SetWidth(max(width-8, 20))
	m.dialog.SetWidth(width)
}
func (m *TaggedConfirm) DesiredTags() (*tag.Tags, error) {
	if m == nil {
		return nil, nil
	}
	return DesiredTags(m.field)
}
