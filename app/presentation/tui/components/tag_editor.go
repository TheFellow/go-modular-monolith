package components

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/app/presentation/tui/keys"
	"github.com/TheFellow/go-modular-monolith/app/presentation/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TagsSavedMsg reports a successful complete tag-set replacement.
type TagsSavedMsg struct {
	Target cedar.EntityUID
	Tags   tag.Tags
}

type tagsSaveFailedMsg struct {
	target cedar.EntityUID
	err    error
}

// TagReplacer supplies the session-scoped tag mutation used by TagEditor.
type TagReplacer interface {
	ReplaceTags(target cedar.EntityUID, desired tag.Tags) (tag.Tags, error)
}

// TagEditor edits the complete tag set for one entity.
type TagEditor struct {
	tags   TagReplacer
	target cedar.EntityUID
	label  string
	field  *forms.TextField
	form   *forms.Form
	err    error
	width  int
	saving bool
}

// NewTagEditor creates an editor prefilled with the entity's canonical tags.
func NewTagEditor(tags TagReplacer, target cedar.EntityUID, label string, current tag.Tags) *TagEditor {
	field := forms.NewTextField("Tags", forms.WithPlaceholder("featured,region=west"))
	_ = field.SetValue(current.Canonical().String())
	return &TagEditor{
		tags: tags, target: target, label: label, field: field,
		form: forms.New(styles.App.Form, keys.App.Form, field),
	}
}

func (m *TagEditor) Init() tea.Cmd { return m.form.Init() }

// Owns reports whether a result belongs to this editor instance.
func (m *TagEditor) Owns(target cedar.EntityUID) bool { return m.target == target }

// Saving reports whether a replacement command is in flight.
func (m *TagEditor) Saving() bool { return m.saving }

// FormEditing reports whether Escape belongs to the active tag field rather
// than the enclosing workflow.
func (m *TagEditor) FormEditing() bool { return m.form.IsEditing() }

func (m *TagEditor) Update(msg tea.Msg) (*TagEditor, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetWidth(typed.Width)
		return m, nil
	case tagsSaveFailedMsg:
		if typed.target == m.target {
			m.saving = false
			m.err = typed.err
		}
		return m, nil
	case tea.KeyMsg:
		if m.saving {
			return m, nil
		}
		if key.Matches(typed, keys.App.Submit) {
			return m, m.save()
		}
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m *TagEditor) View() string {
	parts := []string{
		styles.App.DialogTitle.Render("Manage tags"),
		styles.App.DialogMessage.Render(m.label),
		"",
		m.form.View(),
		styles.App.FormHelp.Render("Comma-separated key or key=value tags. Empty clears all tags."),
	}
	if m.err != nil {
		parts = append(parts, "", styles.App.ErrorText.Width(m.contentWidth()).Render(m.err.Error()))
	}
	parts = append(parts, "", styles.App.HelpDesc.Render("ctrl+s save • esc cancel"))
	if m.saving {
		parts[len(parts)-1] = styles.App.HelpDesc.Render("saving tags…")
	}
	return styles.App.DialogModal.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m *TagEditor) SetWidth(width int) {
	m.width = width
	m.form.SetWidth(m.contentWidth())
}

func (m *TagEditor) contentWidth() int { return max(min(m.width-8, 72), 20) }

func (m *TagEditor) save() tea.Cmd {
	if m.saving {
		return nil
	}
	raw, _ := m.field.Value().(string)
	desired, err := tag.ParseCollection(strings.TrimSpace(raw))
	if err != nil {
		m.err = errors.Invalidf("invalid tags: %v", err)
		return nil
	}
	m.err = nil
	m.saving = true
	return func() tea.Msg {
		result, err := m.tags.ReplaceTags(m.target, desired)
		if err != nil {
			return tagsSaveFailedMsg{target: m.target, err: err}
		}
		return TagsSavedMsg{Target: m.target, Tags: result}
	}
}
