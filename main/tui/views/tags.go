package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/main/tui/keys"
	"github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
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

type tagsMode int

const (
	tagsModeBrowsing tagsMode = iota
	tagsModePickingType
	tagsModePickingEntity
	tagsModeEnteringValue
	tagsModeLoading
	tagsModeResults
)

type tagOperationItem struct {
	operation tagOperation
	title     string
	desc      string
}

func (i tagOperationItem) Title() string       { return i.title }
func (i tagOperationItem) Description() string { return i.desc }
func (i tagOperationItem) FilterValue() string { return i.title }

type tagTypeItem struct {
	typeID cedar.EntityType
	label  string
}

func (i tagTypeItem) Title() string       { return i.label }
func (i tagTypeItem) Description() string { return "Select active " + strings.ToLower(i.label) }
func (i tagTypeItem) FilterValue() string { return i.label }

type tagEntityItem struct {
	uid  cedar.EntityUID
	name string
	desc string
}

func (i tagEntityItem) Title() string       { return i.name }
func (i tagEntityItem) Description() string { return i.desc }
func (i tagEntityItem) FilterValue() string { return i.name + " " + string(i.uid.ID) }

type tagEntitiesLoadedMsg struct {
	items []list.Item
	err   error
}

type tagResultMsg struct {
	operation  tagOperation
	target     cedar.EntityUID
	tags       tag.Tags
	changed    bool
	references []tagging.Reference
	summaries  []tagging.Summary
	err        error
}

// Tags is the cross-domain workspace for entity tag operations and discovery.
type Tags struct {
	app    *app.Session
	styles styles.Styles
	keys   keys.KeyMap

	operations list.Model
	picker     list.Model
	results    table.Model
	form       *forms.Form
	value      *forms.TextField
	spinner    tui.Spinner

	mode      tagsMode
	operation tagOperation
	target    cedar.EntityUID
	result    *tagResultMsg
	err       error
	width     int
	height    int
}

func NewTags(application *app.Session) *Tags {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = tagSelectedStyle(styles.App)
	delegate.Styles.SelectedDesc = tagSelectedStyle(styles.App)
	operations := list.New(tagOperationItems(), delegate, 0, 0)
	operations.Title = "Tags"
	operations.SetShowHelp(false)
	operations.SetShowStatusBar(false)
	operations.SetFilteringEnabled(false)

	picker := list.New(nil, delegate, 0, 0)
	picker.SetShowHelp(false)
	picker.SetShowStatusBar(false)
	picker.SetFilteringEnabled(true)

	results := table.New(table.WithFocused(true))
	results.SetStyles(tagTableStyles(styles.App))

	value := forms.NewTextField("Tag / key", forms.WithRequired(), forms.WithPlaceholder("key or key=value"))
	vm := &Tags{
		app: application, styles: styles.App, keys: keys.App,
		operations: operations, picker: picker, results: results,
		value: value, form: forms.New(styles.App.Form, keys.App.Form, value),
	}
	vm.spinner = tui.NewSpinner("Working with tags...", vm.styles.Subtitle)
	return vm
}

func tagOperationItems() []list.Item {
	return []list.Item{
		tagOperationItem{tagOperationInspect, "Inspect entity tags", "Choose an entity and view its tags"},
		tagOperationItem{tagOperationAdd, "Add or replace a tag", "Choose an entity, then enter key or key=value"},
		tagOperationItem{tagOperationRemove, "Remove a tag", "Choose an entity, then enter the key"},
		tagOperationItem{tagOperationShow, "Show exact tag", "List active entities carrying key or key=value"},
		tagOperationItem{tagOperationShowKey, "Show all values for key", "List active entities carrying any value for a key"},
		tagOperationItem{tagOperationSummary, "Tag usage summary", "Count active tag usage by entity type"},
	}
}

func (m *Tags) Init() tea.Cmd { return nil }

func (m *Tags) Interaction() Interaction {
	return Interaction{
		HandlesBack: m.mode != tagsModeBrowsing,
		CapturesText: m.mode == tagsModeEnteringValue ||
			(m.mode == tagsModePickingType || m.mode == tagsModePickingEntity) && m.picker.SettingFilter(),
	}
}

func (m *Tags) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(typed.Width, typed.Height)
		return m, nil
	case tagEntitiesLoadedMsg:
		if typed.err != nil {
			m.mode, m.err = tagsModePickingType, typed.err
			return m, nil
		}
		m.picker.Title = "Select entity"
		m.picker.SetItems(typed.items)
		m.picker.ResetSelected()
		m.mode, m.err = tagsModePickingEntity, nil
		return m, nil
	case tagResultMsg:
		m.mode = tagsModeResults
		m.err = typed.err
		if typed.err == nil {
			result := typed
			m.result = &result
			m.setResultTable(result)
		}
		return m, nil
	case tea.KeyMsg:
		if (m.mode == tagsModePickingType || m.mode == tagsModePickingEntity) && m.picker.SettingFilter() {
			var cmd tea.Cmd
			m.picker, cmd = m.picker.Update(typed)
			return m, cmd
		}
		if key.Matches(typed, m.keys.Back) && m.mode != tagsModeBrowsing {
			m.back()
			return m, nil
		}
		if key.Matches(typed, m.keys.Enter) {
			switch m.mode {
			case tagsModeBrowsing:
				return m, m.selectOperation()
			case tagsModePickingType:
				return m, m.selectType()
			case tagsModePickingEntity:
				return m, m.selectEntity()
			case tagsModeEnteringValue, tagsModeLoading, tagsModeResults:
			}
		}
		if key.Matches(typed, m.keys.Submit) && m.mode == tagsModeEnteringValue {
			return m, m.submitValue()
		}
	}

	switch m.mode {
	case tagsModeBrowsing:
		var cmd tea.Cmd
		m.operations, cmd = m.operations.Update(msg)
		return m, cmd
	case tagsModePickingType, tagsModePickingEntity:
		var cmd tea.Cmd
		m.picker, cmd = m.picker.Update(msg)
		return m, cmd
	case tagsModeEnteringValue:
		var cmd tea.Cmd
		m.form, cmd = m.form.Update(msg)
		return m, cmd
	case tagsModeLoading:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tagsModeResults:
		var cmd tea.Cmd
		m.results, cmd = m.results.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Tags) View() string {
	var content string
	switch m.mode {
	case tagsModeBrowsing:
		content = m.operations.View()
	case tagsModePickingType, tagsModePickingEntity:
		content = m.styles.DialogModal.Render(m.picker.View())
	case tagsModeEnteringValue:
		content = m.styles.DialogModal.Render(lipgloss.JoinVertical(lipgloss.Left,
			m.styles.DialogTitle.Render(operationTitle(m.operation)), "", m.form.View(), "",
			m.styles.HelpDesc.Render("ctrl+s submit • esc cancel")))
	case tagsModeLoading:
		content = m.spinner.View()
	case tagsModeResults:
		title := m.styles.Title.Render(operationTitle(m.operation))
		if m.err != nil {
			content = lipgloss.JoinVertical(lipgloss.Left, title, "", m.styles.ErrorText.Render("Error: "+m.err.Error()), "", m.styles.HelpDesc.Render("esc back"))
		} else {
			content = lipgloss.JoinVertical(lipgloss.Left, title, "", m.results.View(), "", m.styles.HelpDesc.Render("↑/↓ navigate • esc back"))
		}
	}
	if m.width > 0 && m.height > 0 && m.mode != tagsModeBrowsing {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}

func (m *Tags) ShortHelp() []key.Binding {
	if m.mode == tagsModeEnteringValue {
		return []key.Binding{m.keys.Submit, m.keys.Back}
	}
	if m.mode == tagsModeBrowsing {
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back}
	}
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back}
}

func (m *Tags) FullHelp() [][]key.Binding { return [][]key.Binding{m.ShortHelp()} }

func (m *Tags) selectOperation() tea.Cmd {
	selected, ok := m.operations.SelectedItem().(tagOperationItem)
	if !ok {
		return nil
	}
	m.operation, m.err, m.result = selected.operation, nil, nil
	switch selected.operation {
	case tagOperationInspect, tagOperationAdd, tagOperationRemove:
		m.picker.Title = "Select entity type"
		m.picker.SetItems(entityTypeItems())
		m.picker.ResetSelected()
		m.mode = tagsModePickingType
		return nil
	case tagOperationShow, tagOperationShowKey:
		if err := m.value.SetValue(""); err != nil {
			m.err = err
			return nil
		}
		m.mode = tagsModeEnteringValue
		return m.form.Init()
	case tagOperationSummary:
		m.mode = tagsModeLoading
		return tea.Batch(m.spinner.Init(), m.runOperation(tag.Tag{}))
	}
	return nil
}

func entityTypeItems() []list.Item {
	return []list.Item{
		tagTypeItem{entity.TypeDrink, "Drinks"}, tagTypeItem{entity.TypeIngredient, "Ingredients"},
		tagTypeItem{entity.TypeInventory, "Inventory"}, tagTypeItem{entity.TypeMenu, "Menus"},
		tagTypeItem{entity.TypeOrder, "Orders"},
	}
}

func (m *Tags) selectType() tea.Cmd {
	selected, ok := m.picker.SelectedItem().(tagTypeItem)
	if !ok {
		return nil
	}
	m.mode = tagsModeLoading
	return tea.Batch(m.spinner.Init(), m.loadEntities(selected.typeID))
}

func (m *Tags) selectEntity() tea.Cmd {
	selected, ok := m.picker.SelectedItem().(tagEntityItem)
	if !ok {
		return nil
	}
	m.target = selected.uid
	if m.operation == tagOperationInspect {
		m.mode = tagsModeLoading
		return tea.Batch(m.spinner.Init(), m.runOperation(tag.Tag{}))
	}
	if err := m.value.SetValue(""); err != nil {
		m.err = err
		return nil
	}
	m.mode = tagsModeEnteringValue
	return m.form.Init()
}

func (m *Tags) submitValue() tea.Cmd {
	if err := m.form.Validate(); err != nil {
		m.err = errors.Invalidf("%v", err)
		return nil
	}
	raw, _ := m.value.Value().(string)
	var parsed tag.Tag
	var err error
	if m.operation == tagOperationRemove || m.operation == tagOperationShowKey {
		parsed, err = tag.New(strings.TrimSpace(raw), "")
	} else {
		parsed, err = tag.Parse(strings.TrimSpace(raw))
	}
	if err != nil {
		m.err = err
		return nil
	}
	m.mode, m.err = tagsModeLoading, nil
	return tea.Batch(m.spinner.Init(), m.runOperation(parsed))
}

func (m *Tags) runOperation(value tag.Tag) tea.Cmd {
	return func() tea.Msg {
		msg := tagResultMsg{operation: m.operation, target: m.target}
		switch m.operation {
		case tagOperationInspect:
			msg.tags, msg.err = m.app.Tags.List(m.context(), m.target)
		case tagOperationAdd:
			result, err := m.app.Tags.Upsert(m.context(), m.target, value)
			msg.tags, msg.changed, msg.err = result.Tags, result.Changed, err
		case tagOperationRemove:
			result, err := m.app.Tags.Remove(m.context(), m.target, value.Key)
			msg.tags, msg.changed, msg.err = result.Tags, result.Changed, err
		case tagOperationShow:
			msg.references, msg.err = m.app.Tags.Show(m.context(), value, true)
		case tagOperationShowKey:
			msg.references, msg.err = m.app.Tags.Show(m.context(), value, false)
		case tagOperationSummary:
			msg.summaries, msg.err = m.app.Tags.Summary(m.context())
		}
		return msg
	}
}

func (m *Tags) loadEntities(entityType cedar.EntityType) tea.Cmd {
	return func() tea.Msg {
		ctx := m.context()
		items := []list.Item{}
		switch entityType {
		case entity.TypeDrink:
			page, err := m.app.Drinks.List(ctx, drinks.ListRequest{Limit: 1000})
			if err != nil {
				return tagEntitiesLoadedMsg{err: err}
			}
			for _, v := range page.Items {
				items = append(items, tagEntityItem{v.EntityUID(), v.Name, fmt.Sprintf("%s • %s", v.Category, v.ID)})
			}
		case entity.TypeIngredient:
			page, err := m.app.Ingredients.List(ctx, ingredients.ListRequest{Limit: 1000})
			if err != nil {
				return tagEntitiesLoadedMsg{err: err}
			}
			for _, v := range page.Items {
				items = append(items, tagEntityItem{v.EntityUID(), v.Name, fmt.Sprintf("%s • %s", v.Category, v.ID)})
			}
		case entity.TypeInventory:
			page, err := m.app.Inventory.List(ctx, inventory.ListRequest{Limit: 1000})
			if err != nil {
				return tagEntitiesLoadedMsg{err: err}
			}
			for _, v := range page.Items {
				items = append(items, tagEntityItem{v.EntityUID(), v.ID.String(), "Ingredient " + v.IngredientID.String()})
			}
		case entity.TypeMenu:
			page, err := m.app.Menus.List(ctx, menus.ListRequest{Limit: 1000})
			if err != nil {
				return tagEntitiesLoadedMsg{err: err}
			}
			for _, v := range page.Items {
				items = append(items, tagEntityItem{v.EntityUID(), v.Name, fmt.Sprintf("%s • %s", v.Status, v.ID)})
			}
		case entity.TypeOrder:
			page, err := m.app.Orders.List(ctx, orders.ListRequest{Limit: 1000})
			if err != nil {
				return tagEntitiesLoadedMsg{err: err}
			}
			for _, v := range page.Items {
				items = append(items, tagEntityItem{v.EntityUID(), v.ID.String(), fmt.Sprintf("%s • menu %s", v.Status, v.MenuID)})
			}
		}
		return tagEntitiesLoadedMsg{items: items}
	}
}

func (m *Tags) setResultTable(result tagResultMsg) {
	switch result.operation {
	case tagOperationSummary:
		rows := make([]table.Row, 0, len(result.summaries))
		for _, v := range result.summaries {
			rows = append(rows, table.Row{v.Tag, fmt.Sprint(v.Total), fmt.Sprint(v.Drinks), fmt.Sprint(v.Ingredients), fmt.Sprint(v.Inventory), fmt.Sprint(v.Menus), fmt.Sprint(v.Orders)})
		}
		m.replaceResultTable(summaryColumns(m.width), rows)
	case tagOperationShow, tagOperationShowKey:
		rows := make([]table.Row, 0, len(result.references))
		for _, v := range result.references {
			rows = append(rows, table.Row{v.EntityType, v.EntityID, v.Tag})
		}
		m.replaceResultTable(referenceColumns(m.width), rows)
	case tagOperationInspect, tagOperationAdd, tagOperationRemove:
		columns := []table.Column{{Title: "ENTITY", Width: 34}, {Title: "TAGS", Width: max(m.width-40, 30)}, {Title: "RESULT", Width: 10}}
		state := "inspected"
		if result.operation != tagOperationInspect {
			state = "unchanged"
			if result.changed {
				state = "changed"
			}
		}
		values := result.tags.Canonical().String()
		if values == "" {
			values = "(none)"
		}
		m.replaceResultTable(columns, []table.Row{{string(result.target.ID), values, state}})
	}
	m.results.SetCursor(0)
}

// replaceResultTable safely changes a Bubble Tea table's schema. SetColumns
// eagerly renders the existing viewport, so rows from the previous schema
// must be removed before changing the column count.
func (m *Tags) replaceResultTable(columns []table.Column, rows []table.Row) {
	m.results.SetRows(nil)
	m.results.SetColumns(columns)
	m.results.SetRows(rows)
}

func referenceColumns(width int) []table.Column {
	available := max(width-8, 72)
	return []table.Column{{Title: "ENTITY TYPE", Width: 14}, {Title: "ENTITY ID", Width: 34}, {Title: "TAG", Width: max(available-52, 20)}}
}

func summaryColumns(width int) []table.Column {
	tagWidth := max(width-70, 22)
	return []table.Column{{Title: "TAG", Width: tagWidth}, {Title: "TOTAL", Width: 7}, {Title: "DRINKS", Width: 8}, {Title: "INGREDIENTS", Width: 13}, {Title: "INVENTORY", Width: 11}, {Title: "MENUS", Width: 7}, {Title: "ORDERS", Width: 8}}
}

func tagTableStyles(s styles.Styles) table.Styles {
	result := table.DefaultStyles()
	result.Header = s.ListView.Title.Padding(0, 1)
	result.Cell = s.ListView.Muted.Padding(0, 1)
	result.Selected = tagSelectedStyle(s).Padding(0, 1)
	return result
}

// tagSelectedStyle avoids a filled selection background. Some terminal color
// profiles resolve adaptive foreground and background colors independently,
// which can otherwise produce light text on a light background.
func tagSelectedStyle(s styles.Styles) lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Underline(true).Foreground(s.Primary)
}

func (m *Tags) setSize(width, height int) {
	m.width, m.height = width, height
	contentWidth, contentHeight := max(width-4, 40), max(height-4, 10)
	m.operations.SetSize(contentWidth, contentHeight)
	m.picker.SetSize(min(contentWidth, 72), min(contentHeight, 24))
	m.form.SetWidth(min(contentWidth, 72))
	m.results.SetWidth(contentWidth)
	m.results.SetHeight(max(contentHeight-5, 5))
	if m.result != nil {
		m.setResultTable(*m.result)
	}
}

func (m *Tags) back() {
	switch m.mode {
	case tagsModeBrowsing, tagsModePickingType, tagsModeLoading, tagsModeResults:
		m.mode = tagsModeBrowsing
	case tagsModePickingEntity:
		m.picker.Title = "Select entity type"
		m.picker.SetItems(entityTypeItems())
		m.mode = tagsModePickingType
	case tagsModeEnteringValue:
		if !m.target.IsZero() {
			m.mode = tagsModePickingEntity
		} else {
			m.mode = tagsModeBrowsing
		}
	}
	m.err, m.result = nil, nil
}

func operationTitle(value tagOperation) string {
	for _, item := range tagOperationItems() {
		operation := item.(tagOperationItem)
		if operation.operation == value {
			return operation.title
		}
	}
	return "Tags"
}

func (m *Tags) context() *middleware.Context { return m.app.Context() }
