package tui

import (
	"fmt"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"

	"github.com/TheFellow/go-modular-monolith/app"
	drinks "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menus "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type listMode int

const (
	listModeBrowsing listMode = iota
	listModeCreating
	listModeEditing
	listModeTagging
	listModeConfirmingDelete
	listModeFiltering
)

// ListViewModel renders the drinks list and detail panes.
type ListViewModel struct {
	app    *app.Session
	styles tui.ListViewStyles
	keys   listViewKeys

	formStyles   forms.FormStyles
	formKeys     forms.FormKeys
	dialogStyles dialog.DialogStyles
	dialogKeys   dialog.DialogKeys

	list       list.Model
	detail     *DetailViewModel
	detailPane tui.DetailViewport
	mode       listMode
	create     *CreateDrinkVM
	edit       *EditDrinkVM
	tags       *components.TagEditor[cedar.EntityUID, tag.Tags]
	dialog     *dialog.ConfirmDialog
	spinner    tui.Spinner
	loading    bool
	err        error
	actionErr  error
	projector  drinks.ActionProjector
	actions    map[actions.ID]actions.State
	filter     *filterVM
	request    drinks.ListRequest
	next       paging.Cursor
	history    []paging.Cursor
	loadToken  uint64

	deleteTarget *models.Drink

	width        int
	height       int
	listWidth    int
	detailWidth  int
	detailHeight int
}

func NewListViewModel(app *app.Session) *ListViewModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = styles.Standard.ListView.Selected
	delegate.Styles.SelectedDesc = styles.Standard.ListView.Selected

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Drinks"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.Paginator.Type = paginator.Arabic
	l.SetFilteringEnabled(false)

	vm := &ListViewModel{
		app:          app,
		styles:       styles.Standard.ListView,
		keys:         newListViewKeys(),
		formStyles:   styles.Standard.Form,
		formKeys:     keys.Standard.Form,
		dialogStyles: styles.Standard.Dialog,
		dialogKeys:   keys.Standard.Dialog,
		list:         l,
		detail:       NewDetailViewModel(styles.Standard.ListView, app),
		detailPane:   tui.NewDetailViewport(),
		loading:      true,
		projector:    drinks.NewActionProjector(),
	}
	vm.syncActions()
	vm.spinner = tui.NewSpinner("Loading drinks...", vm.styles.Subtitle)
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	if !m.actionEnabled(drinks.ControlList) {
		m.loading = false
		return nil
	}
	m.loading = true
	return tea.Batch(m.spinner.Init(), m.loadDrinks(""))
}

func (m *ListViewModel) Interaction() tui.Interaction {
	return tui.Interaction{
		HandlesBack:  m.mode != listModeBrowsing,
		CapturesText: m.mode == listModeFiltering || m.mode == listModeCreating || m.mode == listModeEditing || m.mode == listModeTagging,
	}
}

func (m *ListViewModel) Update(msg tea.Msg) (tui.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.DataInvalidatedMsg:
		if m.mode != listModeBrowsing || !m.actionEnabled(drinks.ControlList) {
			return m, nil
		}
		m.loading, m.err = true, nil
		return m, tea.Batch(m.spinner.Init(), m.loadDrinks(m.request.Cursor))
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		switch m.mode {
		case listModeBrowsing:
		case listModeCreating:
			m.create.SetSize(m.detailWidth, m.detailHeight)
		case listModeEditing:
			m.edit.SetSize(m.detailWidth, m.detailHeight)
		case listModeTagging:
			m.tags.SetWidth(m.width)
		case listModeConfirmingDelete:
			m.dialog.SetWidth(m.width)
		case listModeFiltering:
			m.filter.form.SetWidth(m.detailWidth)
		}
		return m, nil
	case DrinkCreatedMsg:
		m.mode = listModeBrowsing
		m.create = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadDrinks(m.request.Cursor))
	case DrinkUpdatedMsg:
		m.mode = listModeBrowsing
		m.edit = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadDrinks(m.request.Cursor))
	case components.TagsSavedMsg[cedar.EntityUID, tag.Tags]:
		if m.mode != listModeTagging || m.tags == nil || !m.tags.Owns(msg.Target) {
			return m, nil
		}
		m.mode, m.tags, m.loading, m.err = listModeBrowsing, nil, true, nil
		return m, tea.Batch(m.spinner.Init(), m.loadDrinks(m.request.Cursor))
	case DrinkDeletedMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.deleteTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadDrinks(m.request.Cursor))
	case DeleteErrorMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.deleteTarget = nil
		m.err = msg.Err
		return m, nil
	case showDeleteDialogMsg:
		m.mode = listModeConfirmingDelete
		m.dialog = msg.dialog
		m.deleteTarget = &msg.target
		m.dialog.SetWidth(m.width)
		return m, nil
	case dialog.ConfirmMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		return m, m.performDelete()
	case dialog.CancelMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.deleteTarget = nil
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case listModeBrowsing:
		case listModeConfirmingDelete:
		case listModeCreating:
			if key.Matches(msg, m.keys.Back) && !m.create.form.IsEditing() {
				if m.create.Submitting() {
					return m, nil
				}
				m.mode = listModeBrowsing
				m.create = nil
				return m, nil
			}
		case listModeEditing:
			if key.Matches(msg, m.keys.Back) && !m.edit.form.IsEditing() {
				if m.edit.Submitting() {
					return m, nil
				}
				m.mode = listModeBrowsing
				m.edit = nil
				return m, nil
			}
		case listModeTagging:
			if key.Matches(msg, m.keys.Back) && !m.tags.FormEditing() {
				if m.tags.Saving() {
					return m, nil
				}
				m.mode, m.tags = listModeBrowsing, nil
				return m, nil
			}
		case listModeFiltering:
			if key.Matches(msg, m.keys.Back) && !m.filter.form.IsEditing() {
				m.mode, m.filter = listModeBrowsing, nil
				return m, nil
			}
			if filterSubmit(msg) {
				if !m.actionEnabled(drinks.ControlList) {
					return m, nil
				}
				req, err := m.filter.Request()
				if err != nil {
					return m, nil
				}
				m.request, m.history, m.next = req, nil, ""
				m.mode, m.filter, m.loading, m.err = listModeBrowsing, nil, true, nil
				return m, tea.Batch(m.spinner.Init(), m.loadDrinks(""))
			}
		}
		if m.mode != listModeBrowsing {
			break
		}
		switch {
		case key.Matches(msg, m.keys.Refresh):
			if !m.actionEnabled(drinks.ControlList) {
				return m, nil
			}
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Init(), m.loadDrinks(m.request.Cursor))
		case msg.String() == "f":
			if !m.actionEnabled(drinks.ControlList) {
				return m, nil
			}
			m.mode, m.filter = listModeFiltering, newFilterVM(m.request)
			m.filter.form.SetWidth(m.detailWidth)
			return m, m.filter.Init()
		case msg.String() == "]" && m.next != "" && m.actionEnabled(drinks.ControlList):
			m.history = append(m.history, m.request.Cursor)
			m.request.Cursor = m.next
			m.loading = true
			return m, tea.Batch(m.spinner.Init(), m.loadDrinks(m.request.Cursor))
		case msg.String() == "[" && len(m.history) > 0 && m.actionEnabled(drinks.ControlList):
			i := len(m.history) - 1
			m.request.Cursor = m.history[i]
			m.history = m.history[:i]
			m.loading = true
			return m, tea.Batch(m.spinner.Init(), m.loadDrinks(m.request.Cursor))
		case key.Matches(msg, m.keys.Create):
			if !m.actionEnabled(drinks.ControlCreate) {
				return m, nil
			}
			return m, m.startCreate()
		case key.Matches(msg, m.keys.Edit), key.Matches(msg, m.keys.Enter):
			if !m.actionEnabled(drinks.ControlEdit) {
				return m, nil
			}
			return m, m.startEdit()
		case key.Matches(msg, m.keys.Delete):
			if !m.actionEnabled(drinks.ControlDelete) {
				return m, nil
			}
			return m, m.startDelete()
		case key.Matches(msg, m.keys.Tags):
			if !m.actionEnabled(drinks.ControlTags) {
				return m, nil
			}
			return m, m.startTags()
		}
	case DrinksLoadedMsg:
		if msg.Token != m.loadToken {
			return m, nil
		}
		m.loading = false
		m.err = msg.Err
		if msg.Err != nil {
			m.syncActions()
			return m, nil
		}
		m.next = msg.Next
		selected := selectedDrinkID(m.selectedDrink())
		items := make([]list.Item, 0, len(msg.Drinks))
		for _, drink := range msg.Drinks {
			items = append(items, newDrinkItem(drink))
		}
		m.list.SetItems(items)
		selectDrinkID(&m.list, selected)
		m.updateTitle()
		m.syncDetail()
		m.syncActions()
		return m, nil
	}

	switch m.mode {
	case listModeBrowsing:
	case listModeConfirmingDelete:
		var cmd tea.Cmd
		m.dialog, cmd = m.dialog.Update(msg)
		return m, cmd
	case listModeEditing:
		var cmd tea.Cmd
		m.edit, cmd = m.edit.Update(msg)
		return m, cmd
	case listModeTagging:
		var cmd tea.Cmd
		m.tags, cmd = m.tags.Update(msg)
		return m, cmd
	case listModeCreating:
		var cmd tea.Cmd
		m.create, cmd = m.create.Update(msg)
		return m, cmd
	case listModeFiltering:
		return m, m.filter.Update(msg)
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	if m.detailPane.Update(msg) {
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.syncDetail()
	m.syncActions()
	return m, cmd
}

func (m *ListViewModel) View() string {
	if m.loading {
		return m.renderLoading()
	}

	if m.mode == listModeConfirmingDelete {
		dialogView := m.dialog.View()
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialogView)
		}
		return dialogView
	}
	if m.mode == listModeTagging {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.tags.View())
	}
	if m.mode == listModeFiltering {
		return m.filter.View()
	}

	listView := m.list.View()
	if m.err != nil {
		listView = m.styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.err))
	}
	listView = m.styles.ListPane.Width(tui.PaneStyleWidth(m.styles.ListPane, m.listWidth)).Render(listView)

	detailView := m.detailPane.View(m.detail.View())
	switch m.mode {
	case listModeBrowsing, listModeConfirmingDelete:
	case listModeTagging:
	case listModeCreating:
		detailView = m.create.View()
	case listModeEditing:
		detailView = m.edit.View()
	case listModeFiltering:
	}
	detailView = m.styles.DetailPane.Width(tui.PaneStyleWidth(m.styles.DetailPane, m.detailWidth)).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	switch m.mode {
	case listModeConfirmingDelete:
		return []key.Binding{m.dialogKeys.Confirm, m.keys.Back, m.dialogKeys.Switch}
	case listModeTagging:
		return []key.Binding{m.formKeys.Submit, m.keys.Back}
	case listModeCreating, listModeEditing:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Edit, m.keys.Enter, m.formKeys.Submit, m.keys.Back}
	case listModeBrowsing:
		bindings := []key.Binding{}
		if m.actionEnabled(drinks.ControlList) {
			bindings = append(bindings, m.keys.Up, m.keys.Down, m.list.KeyMap.PrevPage, m.list.KeyMap.NextPage)
		}
		bindings = append(bindings, m.visibleBindings()...)
		if m.actionEnabled(drinks.ControlList) {
			bindings = append(bindings, m.keys.Refresh)
		}
		return append(bindings, tui.DetailScrollHelp, m.keys.Back)
	case listModeFiltering:
	}
	return nil
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	switch m.mode {
	case listModeConfirmingDelete:
		return [][]key.Binding{
			{m.dialogKeys.Confirm, m.keys.Back},
			{m.dialogKeys.Switch},
		}
	case listModeTagging:
		return [][]key.Binding{{m.formKeys.Submit, m.keys.Back}}
	case listModeCreating, listModeEditing:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down, m.keys.Edit, m.keys.Enter, m.formKeys.Submit},
			{m.keys.Back},
		}
	case listModeBrowsing:
		navigation := []key.Binding{}
		pagingHelp := []key.Binding{}
		if m.actionEnabled(drinks.ControlList) {
			navigation = append(navigation, m.keys.Up, m.keys.Down)
			pagingHelp = append(pagingHelp, m.list.KeyMap.PrevPage, m.list.KeyMap.NextPage)
		}
		pagingHelp = append(pagingHelp, tui.DetailScrollHelp)
		if m.actionEnabled(drinks.ControlList) && m.actionEnabled(drinks.ControlEdit) {
			navigation = append(navigation, m.keys.Enter)
		}
		last := []key.Binding{m.keys.Back}
		if m.actionEnabled(drinks.ControlList) {
			last = append([]key.Binding{m.keys.Refresh}, last...)
		}
		return [][]key.Binding{
			navigation,
			pagingHelp,
			m.visibleBindings(),
			last,
		}
	case listModeFiltering:
	}
	return nil
}

func (m *ListViewModel) loadDrinks(cursor paging.Cursor) tea.Cmd {
	m.loadToken++
	token := m.loadToken
	req := m.request
	req.Cursor = cursor
	return func() tea.Msg {
		drinksList, err := m.app.Drinks.List(m.context(), req)
		if err != nil {
			return DrinksLoadedMsg{Err: err, Token: token}
		}

		var items []models.Drink
		for i, drink := range drinksList.Items {
			if drink == nil {
				return DrinksLoadedMsg{Err: fmt.Errorf("drink %d missing", i), Token: token}
			}
			items = append(items, *drink)
		}

		return DrinksLoadedMsg{Drinks: items, Next: drinksList.Next, Token: token}
	}
}

type showDeleteDialogMsg struct {
	dialog *dialog.ConfirmDialog
	target models.Drink
}

func (m *ListViewModel) startCreate() tea.Cmd {
	m.mode = listModeCreating
	m.create = NewCreateDrinkVM(m.app)
	m.create.SetSize(m.detailWidth, m.detailHeight)
	return m.create.Init()
}

func (m *ListViewModel) syncActions() {
	states, err := m.projector.Project(m.context(), m.context().Principal(), m.selectedDrink())
	if err != nil {
		m.actions = nil
		m.actionErr = err
		m.err = err
		return
	}
	if m.actionErr != nil && errors.Is(m.err, m.actionErr) {
		m.err = nil
	}
	m.actionErr = nil
	m.actions = make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		m.actions[state.ID] = state
	}
	if !m.actionEnabled(drinks.ControlList) {
		m.list.SetItems(nil)
		m.syncDetail()
	}
}

func (m *ListViewModel) actionEnabled(id actions.ID) bool {
	state, ok := m.actions[id]
	return ok && state.Visible && state.Enabled
}

func (m *ListViewModel) visibleBindings() []key.Binding {
	pairs := []struct {
		id      actions.ID
		binding key.Binding
	}{
		{drinks.ControlCreate, m.keys.Create},
		{drinks.ControlEdit, m.keys.Edit},
		{drinks.ControlDelete, m.keys.Delete},
		{drinks.ControlTags, m.keys.Tags},
	}
	bindings := make([]key.Binding, 0, len(pairs))
	for _, pair := range pairs {
		if m.actionEnabled(pair.id) {
			bindings = append(bindings, pair.binding)
		}
	}
	return bindings
}

func (m *ListViewModel) startEdit() tea.Cmd {
	drink := m.selectedDrink()
	if drink == nil {
		return nil
	}
	m.mode = listModeEditing
	m.edit = NewEditDrinkVM(m.app, drink)
	m.edit.SetSize(m.detailWidth, m.detailHeight)
	return m.edit.Init()
}

func (m *ListViewModel) startDelete() tea.Cmd {
	drink := m.selectedDrink()
	if drink == nil {
		return nil
	}
	return m.showDeleteConfirm(drink)
}

func (m *ListViewModel) showDeleteConfirm(drink *models.Drink) tea.Cmd {
	if drink == nil {
		return nil
	}
	return func() tea.Msg {
		menusList, err := paging.Collect(func(cursor paging.Cursor) (paging.Page[*menusmodels.Menu], error) {
			return m.app.Menus.List(m.context(), menus.ListRequest{Cursor: cursor})
		})
		if err != nil {
			return DeleteErrorMsg{Err: err}
		}
		menuCount := countMenusWithDrink(menusList, drink.ID)
		message := fmt.Sprintf("Delete %q?", drink.Name)
		if menuCount > 0 {
			message = fmt.Sprintf(
				"Delete %q?\n\nThis drink appears on %d menu(s) and will be removed from them.",
				drink.Name,
				menuCount,
			)
		}
		confirm := dialog.NewConfirmDialog(
			"Delete Drink",
			message,
			dialog.WithDangerous(),
			dialog.WithFocusCancel(),
			dialog.WithConfirmText("Delete"),
			dialog.WithStyles(m.dialogStyles),
			dialog.WithKeys(m.dialogKeys),
		)
		return showDeleteDialogMsg{dialog: confirm, target: *drink}
	}
}

func (m *ListViewModel) performDelete() tea.Cmd {
	target := *m.deleteTarget
	return func() tea.Msg {
		deleted, err := m.app.Drinks.Delete(m.context(), target.ID)
		if err != nil {
			return DeleteErrorMsg{Err: err}
		}
		return DrinkDeletedMsg{Drink: deleted}
	}
}

func (m *ListViewModel) context() *middleware.Context {
	return m.app.Context()
}

func (m *ListViewModel) selectedDrink() *models.Drink {
	item, ok := m.list.SelectedItem().(drinkItem)
	if !ok {
		return nil
	}
	drink := item.Value
	return &drink
}

func selectedDrinkID(value *models.Drink) entity.DrinkID {
	if value == nil {
		return entity.DrinkID{}
	}
	return value.ID
}

func selectDrinkID(values *list.Model, id entity.DrinkID) {
	if id.IsZero() {
		return
	}
	for i, item := range values.Items() {
		if drink, ok := item.(drinkItem); ok && drink.Value.ID == id {
			values.Select(i)
			return
		}
	}
}

func (m *ListViewModel) updateTitle() {
	parts := []string{"Drinks"}
	if m.request.Name != "" {
		parts = append(parts, "name="+m.request.Name)
	}
	if m.request.Category != "" {
		parts = append(parts, "category="+string(m.request.Category))
	}
	if m.request.Glass != "" {
		parts = append(parts, "glass="+string(m.request.Glass))
	}
	if m.request.Filter != "" {
		parts = append(parts, "filter="+m.request.Filter)
	}
	parts = append(parts, fmt.Sprintf("page size=%d", effectiveDrinkLimit(m.request.Limit)))
	if len(m.history) > 0 {
		parts = append(parts, fmt.Sprintf("page=%d", len(m.history)+1))
	}
	m.list.Title = strings.Join(parts, " • ")
}

func effectiveDrinkLimit(limit int) int {
	if limit <= 0 {
		return paging.DefaultLimit
	}
	return limit
}

func (m *ListViewModel) startTags() tea.Cmd {
	drink := m.selectedDrink()
	if drink == nil {
		return nil
	}
	m.mode = listModeTagging
	m.tags = components.NewTagEditor(m.app.ReplaceTags, tag.ParseCollection, drink.EntityUID(), drink.Name, drink.Tags.Canonical().String())
	m.tags.SetWidth(m.width)
	return m.tags.Init()
}

func countMenusWithDrink(menusList []*menusmodels.Menu, drinkID entity.DrinkID) int {
	count := 0
	for _, menu := range menusList {
		if menu == nil {
			continue
		}
		if menuHasDrink(menu, drinkID) {
			count++
		}
	}
	return count
}

func menuHasDrink(menu *menusmodels.Menu, drinkID entity.DrinkID) bool {
	for _, item := range menu.Items {
		if item.DrinkID == drinkID {
			return true
		}
	}
	return false
}

func (m *ListViewModel) renderLoading() string {
	content := m.spinner.View()
	if m.width <= 0 || m.height <= 0 {
		return content
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *ListViewModel) setSize(width, height int) {
	m.width = width
	m.height = height

	if width <= 0 {
		return
	}

	listWidth, detailWidth := tui.SplitListDetailWidths(width)
	listWidth = tui.PaneContentWidth(m.styles.ListPane, listWidth)
	detailWidth = tui.PaneContentWidth(m.styles.DetailPane, detailWidth)

	m.list.SetSize(listWidth, height)
	m.detail.SetSize(detailWidth, height)
	_, frameHeight := m.styles.DetailPane.GetFrameSize()
	m.detailPane.SetSize(detailWidth, max(height-frameHeight, 1))
	m.listWidth = listWidth
	m.detailWidth = detailWidth
	m.detailHeight = max(height-frameHeight, 1)
}

func (m *ListViewModel) syncDetail() {
	item, ok := m.list.SelectedItem().(drinkItem)
	if !ok {
		m.detail.SetDrink(optional.None[models.Drink]())
		return
	}
	m.detail.SetDrink(optional.Some(item.Value))
}
