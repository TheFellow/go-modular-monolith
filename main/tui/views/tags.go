package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/main/tui/keys"
	"github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
)

type tagOperation string

const (
	tagOperationInspect tagOperation = "inspect"
	tagOperationAdd     tagOperation = "add"
	tagOperationRemove  tagOperation = "remove"
)

// TagsLoadedMsg carries the result of an authorized tag operation.
type TagsLoadedMsg struct {
	EntityID  string
	Tags      tag.Tags
	Operation tagOperation
	Changed   bool
	Err       error
}

// Tags is the cross-domain workspace for inspecting and mutating entity tags.
type Tags struct {
	app       *app.Session
	styles    styles.Styles
	keys      keys.KeyMap
	form      *forms.Form
	entityID  *forms.TextField
	operation *forms.SelectField
	value     *forms.TextField
	spinner   components.Spinner
	loading   bool
	result    *TagsLoadedMsg
	err       error
	width     int
	height    int
}

func NewTags(application *app.Session) *Tags {
	entityID := forms.NewTextField("Entity ID", forms.WithRequired(), forms.WithPlaceholder("e.g., drk-..."))
	operation := forms.NewSelectField("Operation", []forms.SelectOption{
		{Label: "Inspect", Value: tagOperationInspect},
		{Label: "Add or replace", Value: tagOperationAdd},
		{Label: "Remove", Value: tagOperationRemove},
	}, forms.WithRequired())
	value := forms.NewTextField("Tag / key", forms.WithPlaceholder("Add: key or key=value • Remove: key"))
	vm := &Tags{
		app: application, styles: styles.App, keys: keys.App,
		entityID: entityID, operation: operation, value: value,
		form: forms.New(styles.App.Form, keys.App.Form, entityID, operation, value),
	}
	vm.spinner = components.NewSpinner("Updating tags...", vm.styles.Subtitle)
	return vm
}

func (m *Tags) Init() tea.Cmd { return m.form.Init() }

func (m *Tags) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = typed.Width, typed.Height
		m.form.SetWidth(max(typed.Width-8, 20))
		return m, nil
	case TagsLoadedMsg:
		m.loading = false
		m.err = typed.Err
		if typed.Err == nil {
			result := typed
			m.result = &result
			return m, nil
		}
		return m, func() tea.Msg { return ErrorMsg{Err: typed.Err} }
	case tea.KeyMsg:
		if key.Matches(typed, m.keys.Submit) {
			return m, m.submit()
		}
	}
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m *Tags) View() string {
	header := m.styles.Title.Render("Entity Tags")
	subtitle := m.styles.Subtitle.Render("Inspect, add, replace, or remove user-authored tags on any operational entity")
	parts := []string{header, subtitle, "", m.form.View(), "", m.styles.InfoText.Render("Submit with ctrl+s • Tags use key or key=value • Removal uses key")}
	if m.loading {
		parts = append(parts, "", m.spinner.View())
	} else if m.err != nil {
		parts = append(parts, "", m.styles.ErrorText.Render("Error: "+m.err.Error()))
	} else if m.result != nil {
		parts = append(parts, "", m.renderResult())
	}
	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
	}
	return body
}

func (m *Tags) ShortHelp() []key.Binding {
	return []key.Binding{m.keys.NextField, m.keys.PrevField, m.keys.Submit, m.keys.Back}
}

func (m *Tags) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.NextField, m.keys.PrevField, m.keys.Submit},
		{m.keys.Back, m.keys.Help, m.keys.Quit},
	}
}

func (m *Tags) submit() tea.Cmd {
	if m.loading {
		return nil
	}
	if err := m.form.Validate(); err != nil {
		m.err = errors.Invalidf("%v", err)
		return nil
	}
	rawID := strings.TrimSpace(stringValue(m.entityID.Value()))
	target, err := entity.ParseID(rawID)
	if err != nil {
		m.err = err
		return nil
	}
	operation, ok := m.operation.Value().(tagOperation)
	if !ok {
		m.err = errors.Invalidf("operation is required")
		return nil
	}
	rawValue := strings.TrimSpace(stringValue(m.value.Value()))
	var parsed tag.Tag
	switch operation {
	case tagOperationInspect:
	case tagOperationAdd:
		parsed, err = tag.Parse(rawValue)
	case tagOperationRemove:
		parsed, err = tag.New(rawValue, "")
	default:
		err = errors.Invalidf("unsupported tag operation: %s", operation)
	}
	if err != nil {
		m.err = err
		return nil
	}
	m.loading = true
	m.err = nil
	m.result = nil
	return tea.Batch(m.spinner.Init(), func() tea.Msg {
		msg := TagsLoadedMsg{EntityID: rawID, Operation: operation}
		switch operation {
		case tagOperationInspect:
			msg.Tags, msg.Err = m.app.Tags.List(m.context(), target)
		case tagOperationAdd:
			result, runErr := m.app.Tags.Upsert(m.context(), target, parsed)
			msg.Tags, msg.Changed, msg.Err = result.Tags, result.Changed, runErr
		case tagOperationRemove:
			result, runErr := m.app.Tags.Remove(m.context(), target, parsed.Key)
			msg.Tags, msg.Changed, msg.Err = result.Tags, result.Changed, runErr
		}
		return msg
	})
}

func (m *Tags) renderResult() string {
	values := m.result.Tags.Canonical().String()
	if values == "" {
		values = "(none)"
	}
	state := "inspected"
	if m.result.Operation != tagOperationInspect {
		state = "unchanged"
		if m.result.Changed {
			state = "changed"
		}
	}
	return m.styles.Card.Render(fmt.Sprintf("%s\nTags: %s\nResult: %s", m.result.EntityID, values, state))
}

func (m *Tags) context() *middleware.Context { return m.app.Context() }

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
