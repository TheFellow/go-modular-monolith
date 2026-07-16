package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	drinks "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menus "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
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
	listModeConfirmingDelete
)

// ListViewModel renders the drinks list and detail panes.
type ListViewModel struct {
	app    *app.Session
	styles tui.ListViewStyles
	keys   tui.ListViewKeys

	formStyles   forms.FormStyles
	formKeys     forms.FormKeys
	dialogStyles dialog.DialogStyles
	dialogKeys   dialog.DialogKeys

	list    list.Model
	detail  *DetailViewModel
	mode    listMode
	create  *CreateDrinkVM
	edit    *EditDrinkVM
	dialog  *dialog.ConfirmDialog
	spinner components.Spinner
	loading bool
	err     error

	deleteTarget *models.Drink

	width       int
	height      int
	listWidth   int
	detailWidth int
}

func NewListViewModel(app *app.Session) *ListViewModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = tuistyles.App.ListView.Selected
	delegate.Styles.SelectedDesc = tuistyles.App.ListView.Selected

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Drinks"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.Paginator.Type = paginator.Arabic
	l.SetFilteringEnabled(true)

	vm := &ListViewModel{
		app:          app,
		styles:       tuistyles.App.ListView,
		keys:         tuikeys.App.ListView,
		formStyles:   tuistyles.App.Form,
		formKeys:     tuikeys.App.Form,
		dialogStyles: tuistyles.App.Dialog,
		dialogKeys:   tuikeys.App.Dialog,
		list:         l,
		detail:       NewDetailViewModel(tuistyles.App.ListView, app),
		loading:      true,
	}
	vm.spinner = components.NewSpinner("Loading drinks...", vm.styles.Subtitle)
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Init(), m.loadDrinks())
}

func (m *ListViewModel) HandleBackKey() bool {
	return m.mode != listModeBrowsing
}

func (m *ListViewModel) Update(msg tea.Msg) (views.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		switch m.mode {
		case listModeBrowsing:
		case listModeCreating:
			m.create.SetWidth(m.detailWidth)
		case listModeEditing:
			m.edit.SetWidth(m.detailWidth)
		case listModeConfirmingDelete:
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case DrinkCreatedMsg:
		m.mode = listModeBrowsing
		m.create = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadDrinks())
	case DrinkUpdatedMsg:
		m.mode = listModeBrowsing
		m.edit = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadDrinks())
	case DrinkDeletedMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.deleteTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadDrinks())
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
			if key.Matches(msg, m.keys.Back) {
				m.mode = listModeBrowsing
				m.create = nil
				return m, nil
			}
		case listModeEditing:
			if key.Matches(msg, m.keys.Back) {
				m.mode = listModeBrowsing
				m.edit = nil
				return m, nil
			}
		}
		switch {
		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Init(), m.loadDrinks())
		case key.Matches(msg, m.keys.Create):
			return m, m.startCreate()
		case key.Matches(msg, m.keys.Edit), key.Matches(msg, m.keys.Enter):
			return m, m.startEdit()
		case key.Matches(msg, m.keys.Delete):
			return m, m.startDelete()
		}
	case DrinksLoadedMsg:
		m.loading = false
		m.err = msg.Err
		items := make([]list.Item, 0, len(msg.Drinks))
		for _, drink := range msg.Drinks {
			items = append(items, drinkItem{drink: drink})
		}
		m.list.SetItems(items)
		m.syncDetail()
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
	case listModeCreating:
		var cmd tea.Cmd
		m.create, cmd = m.create.Update(msg)
		return m, cmd
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.syncDetail()
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

	listView := m.list.View()
	if m.err != nil {
		listView = m.styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.err))
	}
	listView = m.styles.ListPane.Width(m.listWidth).Render(listView)

	detailView := m.detail.View()
	switch m.mode {
	case listModeBrowsing, listModeConfirmingDelete:
	case listModeCreating:
		detailView = m.create.View()
	case listModeEditing:
		detailView = m.edit.View()
	}
	detailView = m.styles.DetailPane.Width(m.detailWidth).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	switch m.mode {
	case listModeConfirmingDelete:
		return []key.Binding{m.dialogKeys.Confirm, m.keys.Back, m.dialogKeys.Switch}
	case listModeCreating, listModeEditing:
		return []key.Binding{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit, m.keys.Back}
	case listModeBrowsing:
		return []key.Binding{
			m.keys.Up, m.keys.Down,
			m.list.KeyMap.PrevPage, m.list.KeyMap.NextPage,
			m.keys.Create, m.keys.Edit, m.keys.Delete,
			m.keys.Refresh, m.keys.Back,
		}
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
	case listModeCreating, listModeEditing:
		return [][]key.Binding{
			{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit},
			{m.keys.Back},
		}
	case listModeBrowsing:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down, m.keys.Enter},
			{m.list.KeyMap.PrevPage, m.list.KeyMap.NextPage},
			{m.keys.Create, m.keys.Edit, m.keys.Delete},
			{m.keys.Refresh, m.keys.Back},
		}
	}
	return nil
}

func (m *ListViewModel) loadDrinks() tea.Cmd {
	return func() tea.Msg {
		drinksList, err := m.app.Drinks.List(m.context(), drinks.ListRequest{})
		if err != nil {
			return DrinksLoadedMsg{Err: err}
		}

		var items []models.Drink
		for i, drink := range drinksList {
			if drink == nil {
				return DrinksLoadedMsg{Err: fmt.Errorf("drink %d missing", i)}
			}
			items = append(items, *drink)
		}

		return DrinksLoadedMsg{Drinks: items}
	}
}

type showDeleteDialogMsg struct {
	dialog *dialog.ConfirmDialog
	target models.Drink
}

func (m *ListViewModel) startCreate() tea.Cmd {
	m.mode = listModeCreating
	m.create = NewCreateDrinkVM(m.app)
	m.create.SetWidth(m.detailWidth)
	return m.create.Init()
}

func (m *ListViewModel) startEdit() tea.Cmd {
	drink := m.selectedDrink()
	if drink == nil {
		return nil
	}
	m.mode = listModeEditing
	m.edit = NewEditDrinkVM(m.app, drink)
	m.edit.SetWidth(m.detailWidth)
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
		menusList, err := m.app.Menus.List(m.context(), menus.ListRequest{})
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
	drink := item.drink
	return &drink
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

	listWidth, detailWidth := views.SplitListDetailWidths(width)

	m.list.SetSize(listWidth, height)
	m.detail.SetSize(detailWidth, height)
	m.listWidth = listWidth
	m.detailWidth = detailWidth
}

func (m *ListViewModel) syncDetail() {
	item, ok := m.list.SelectedItem().(drinkItem)
	if !ok {
		m.detail.SetDrink(optional.None[models.Drink]())
		return
	}
	m.detail.SetDrink(optional.Some(item.drink))
}
