package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/main/tui/keys"
	"github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	cedar "github.com/cedar-policy/cedar-go"
)

type tagOperation string

const (
	tagOperationInspect tagOperation = "inspect"
	tagOperationAdd     tagOperation = "add"
	tagOperationRemove  tagOperation = "remove"
	tagOperationShow    tagOperation = "show"
	tagOperationShowKey tagOperation = "show-key"
	tagOperationSummary tagOperation = "summary"
)

// TagsLoadedMsg carries the result of an authorized tag operation.
type TagsLoadedMsg struct {
	EntityID   string
	Tags       tag.Tags
	Operation  tagOperation
	Changed    bool
	References []tagging.Reference
	Summaries  []tagging.Summary
	Err        error
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
	entityID := forms.NewTextField("Entity ID", forms.WithPlaceholder("Required for inspect/add/remove; e.g., drk-..."))
	operation := forms.NewSelectField("Operation", []forms.SelectOption{
		{Label: "Inspect", Value: tagOperationInspect},
		{Label: "Add or replace", Value: tagOperationAdd},
		{Label: "Remove", Value: tagOperationRemove},
		{Label: "Show exact tag", Value: tagOperationShow},
		{Label: "Show all values for key", Value: tagOperationShowKey},
		{Label: "Usage summary", Value: tagOperationSummary},
	}, forms.WithRequired())
	value := forms.NewTextField("Tag / key", forms.WithPlaceholder("Add: key or key=value • Remove: key"))
	vm := &Tags{
		app: application, styles: styles.App, keys: keys.App,
		entityID: entityID, operation: operation, value: value,
		form: forms.New(styles.App.Form, keys.App.Form, entityID, operation, value),
	}
	vm.spinner = components.NewSpinner("Working with tags...", vm.styles.Subtitle)
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
	subtitle := m.styles.Subtitle.Render("Manage entity tags or discover active tag usage across domains")
	parts := []string{header, subtitle, "", m.form.View(), "", m.styles.InfoText.Render("Submit with ctrl+s • Show exact uses key or key=value • Show key uses a key")}
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
	operation, ok := m.operation.Value().(tagOperation)
	if !ok {
		m.err = errors.Invalidf("operation is required")
		return nil
	}
	rawValue := strings.TrimSpace(stringValue(m.value.Value()))
	rawID := strings.TrimSpace(stringValue(m.entityID.Value()))
	var target cedar.EntityUID
	var err error
	var parsed tag.Tag
	switch operation {
	case tagOperationInspect, tagOperationAdd, tagOperationRemove:
		if rawID == "" {
			err = errors.Invalidf("entity ID is required")
		} else {
			target, err = entity.ParseID(rawID)
		}
	}
	if err == nil {
		switch operation {
		case tagOperationAdd:
			parsed, err = tag.Parse(rawValue)
		case tagOperationRemove:
			parsed, err = tag.New(rawValue, "")
		case tagOperationShow:
			parsed, err = tag.Parse(rawValue)
		case tagOperationShowKey:
			parsed, err = tag.New(rawValue, "")
		case tagOperationInspect, tagOperationSummary:
		default:
			err = errors.Invalidf("unsupported tag operation: %s", operation)
		}
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
		case tagOperationShow:
			msg.References, msg.Err = m.app.Tags.Show(m.context(), parsed, true)
		case tagOperationShowKey:
			msg.References, msg.Err = m.app.Tags.Show(m.context(), parsed, false)
		case tagOperationSummary:
			msg.Summaries, msg.Err = m.app.Tags.Summary(m.context())
		}
		return msg
	})
}

func (m *Tags) renderResult() string {
	if m.result.Operation == tagOperationShow || m.result.Operation == tagOperationShowKey {
		lines := []string{"ENTITY TYPE  ENTITY ID  TAG"}
		for _, row := range m.result.References {
			lines = append(lines, fmt.Sprintf("%-11s  %s  %s", row.EntityType, row.EntityID, row.Tag))
		}
		if len(m.result.References) == 0 {
			lines = append(lines, "(none)")
		}
		return m.styles.Card.Render(strings.Join(lines, "\n"))
	}
	if m.result.Operation == tagOperationSummary {
		lines := []string{"TAG  TOTAL  DRINKS  INGREDIENTS  INVENTORY  MENUS  ORDERS"}
		for _, row := range m.result.Summaries {
			lines = append(lines, fmt.Sprintf("%s  %d  %d  %d  %d  %d  %d", row.Tag, row.Total, row.Drinks, row.Ingredients, row.Inventory, row.Menus, row.Orders))
		}
		if len(m.result.Summaries) == 0 {
			lines = append(lines, "(none)")
		}
		return m.styles.Card.Render(strings.Join(lines, "\n"))
	}
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
