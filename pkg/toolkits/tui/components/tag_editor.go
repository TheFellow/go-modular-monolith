package components

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TagsSavedMsg reports a successful complete tag-set replacement.
type TagsSavedMsg[Target comparable, Tags any] struct {
	Target Target
	Tags   Tags
}

type tagsSaveFailedMsg[Target comparable] struct {
	target Target
	err    error
}

// TagEditor edits the complete tag set for one entity.
type TagEditor[Target comparable, Tags any] struct {
	replace func(Target, Tags) (Tags, error)
	parse   func(string) (Tags, error)
	target  Target
	label   string
	field   *forms.TextField
	form    *forms.Form
	err     error
	width   int
	saving  bool
}

// NewTagEditor creates an editor prefilled with the entity's canonical tags.
func NewTagEditor[Target comparable, Tags any](replace func(Target, Tags) (Tags, error), parse func(string) (Tags, error), target Target, label, current string) *TagEditor[Target, Tags] {
	field := forms.NewTextField("Tags", forms.WithPlaceholder("featured,region=west"))
	_ = field.SetValue(current)
	return &TagEditor[Target, Tags]{
		replace: replace, parse: parse, target: target, label: label, field: field,
		form: forms.New(styles.Standard.Form, keys.Standard.Form, field),
	}
}

func (m *TagEditor[Target, Tags]) Init() tea.Cmd { return m.form.Init() }

// Owns reports whether a result belongs to this editor instance.
func (m *TagEditor[Target, Tags]) Owns(target Target) bool { return m.target == target }

// Saving reports whether a replacement command is in flight.
func (m *TagEditor[Target, Tags]) Saving() bool { return m.saving }

// FormEditing reports whether Escape belongs to the active tag field rather
// than the enclosing workflow.
func (m *TagEditor[Target, Tags]) FormEditing() bool { return m.form.IsEditing() }

func (m *TagEditor[Target, Tags]) Update(msg tea.Msg) (*TagEditor[Target, Tags], tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetWidth(typed.Width)
		return m, nil
	case tagsSaveFailedMsg[Target]:
		if typed.target == m.target {
			m.saving = false
			m.err = typed.err
		}
		return m, nil
	case tea.KeyMsg:
		if m.saving {
			return m, nil
		}
		if key.Matches(typed, keys.Standard.Submit) {
			return m, m.save()
		}
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m *TagEditor[Target, Tags]) View() string {
	parts := []string{
		styles.Standard.DialogTitle.Render("Manage tags"),
		styles.Standard.DialogMessage.Render(m.label),
		"",
		m.form.View(),
		styles.Standard.FormHelp.Render("Comma-separated key or key=value tags. Empty clears all tags."),
	}
	if m.err != nil {
		parts = append(parts, "", styles.Standard.ErrorText.Width(m.contentWidth()).Render(m.err.Error()))
	}
	parts = append(parts, "", styles.Standard.HelpDesc.Render("ctrl+s save • esc cancel"))
	if m.saving {
		parts[len(parts)-1] = styles.Standard.HelpDesc.Render("saving tags…")
	}
	return styles.Standard.DialogModal.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m *TagEditor[Target, Tags]) SetWidth(width int) {
	m.width = width
	m.form.SetWidth(m.contentWidth())
}

func (m *TagEditor[Target, Tags]) contentWidth() int { return max(min(m.width-8, 72), 20) }

func (m *TagEditor[Target, Tags]) save() tea.Cmd {
	if m.saving {
		return nil
	}
	raw, _ := m.field.Value().(string)
	desired, err := m.parse(strings.TrimSpace(raw))
	if err != nil {
		m.err = errors.Invalidf("invalid tags: %v", err)
		return nil
	}
	m.err = nil
	m.saving = true
	return func() tea.Msg {
		result, err := m.replace(m.target, desired)
		if err != nil {
			return tagsSaveFailedMsg[Target]{target: m.target, err: err}
		}
		return TagsSavedMsg[Target, Tags]{Target: m.target, Tags: result}
	}
}
