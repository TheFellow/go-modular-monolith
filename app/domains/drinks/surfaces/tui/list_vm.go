package tui

import (
	"context"
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
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
	"github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListViewModel renders the drinks list and detail panes.
type ListViewModel struct {
	app       *app.App
	principal cedar.EntityUID
	styles    tui.ListViewStyles
	keys      tui.ListViewKeys

	formStyles   forms.FormStyles
	formKeys     forms.FormKeys
	dialogStyles dialog.DialogStyles
	dialogKeys   dialog.DialogKeys

	drinksQueries *queries.Queries

	list    list.Model
	detail  *DetailViewModel
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

func NewListViewModel(app *app.App, principal cedar.EntityUID) *ListViewModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = tuistyles.ListView.Selected
	delegate.Styles.SelectedDesc = tuistyles.ListView.Selected

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Drinks"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)

	vm := &ListViewModel{
		app:           app,
		principal:     principal,
		styles:        tuistyles.ListView,
		keys:          tuikeys.ListView,
		formStyles:    tuistyles.Form,
		formKeys:      tuikeys.Form,
		dialogStyles:  tuistyles.Dialog,
		dialogKeys:    tuikeys.Dialog,
		drinksQueries: queries.New(),
		list:          l,
		detail:        NewDetailViewModel(tuistyles.ListView, app, principal),
		loading:       true,
	}
	vm.spinner = components.NewSpinner("Loading drinks...", vm.styles.Subtitle)
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Init(), m.loadDrinks())
}

func (m *ListViewModel) Update(msg tea.Msg) (views.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		if m.create != nil {
			m.create.SetWidth(m.detailWidth)
		}
		if m.edit != nil {
			m.edit.SetWidth(m.detailWidth)
		}
		if m.dialog != nil {
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case DrinkCreatedMsg:
		m.create = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadDrinks())
	case DrinkUpdatedMsg:
		m.edit = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadDrinks())
	case DrinkDeletedMsg:
		m.dialog = nil
		m.deleteTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadDrinks())
	case DeleteErrorMsg:
		m.dialog = nil
		m.deleteTarget = nil
		m.err = msg.Err
		return m, nil
	case showDeleteDialogMsg:
		m.dialog = msg.dialog
		m.deleteTarget = &msg.target
		if m.dialog != nil {
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case dialog.ConfirmMsg:
		m.dialog = nil
		return m, m.performDelete()
	case dialog.CancelMsg:
		m.dialog = nil
		m.deleteTarget = nil
		return m, nil
	case tea.KeyMsg:
		if m.dialog != nil {
			break
		}
		if m.create != nil {
			if key.Matches(msg, m.keys.Back) {
				m.create = nil
				return m, nil
			}
			break
		}
		if m.edit != nil {
			if key.Matches(msg, m.keys.Back) {
				m.edit = nil
				return m, nil
			}
			break
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

	if m.dialog != nil {
		var cmd tea.Cmd
		m.dialog, cmd = m.dialog.Update(msg)
		return m, cmd
	}

	if m.edit != nil {
		var cmd tea.Cmd
		m.edit, cmd = m.edit.Update(msg)
		return m, cmd
	}

	if m.create != nil {
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

	if m.dialog != nil {
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
	if m.create != nil {
		detailView = m.create.View()
	} else if m.edit != nil {
		detailView = m.edit.View()
	}
	detailView = m.styles.DetailPane.Width(m.detailWidth).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	if m.dialog != nil {
		return []key.Binding{m.dialogKeys.Confirm, m.keys.Back, m.dialogKeys.Switch}
	}
	if m.create != nil || m.edit != nil {
		return []key.Binding{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit, m.keys.Back}
	}
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Create, m.keys.Edit, m.keys.Delete, m.keys.Refresh, m.keys.Back}
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	if m.dialog != nil {
		return [][]key.Binding{
			{m.dialogKeys.Confirm, m.keys.Back},
			{m.dialogKeys.Switch},
		}
	}
	if m.create != nil || m.edit != nil {
		return [][]key.Binding{
			{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit},
			{m.keys.Back},
		}
	}
	return [][]key.Binding{
		{m.keys.Up, m.keys.Down, m.keys.Enter},
		{m.keys.Create, m.keys.Edit, m.keys.Delete},
		{m.keys.Refresh, m.keys.Back},
	}
}

func (m *ListViewModel) loadDrinks() tea.Cmd {
	return func() tea.Msg {
		drinksList, err := m.drinksQueries.List(m.context(), queries.ListFilter{})
		if err != nil {
			return DrinksLoadedMsg{Err: err}
		}

		var items []models.Drink
		for _, drink := range drinksList {
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
	m.create = NewCreateDrinkVM(m.app, m.principal)
	m.create.SetWidth(m.detailWidth)
	return m.create.Init()
}

func (m *ListViewModel) startEdit() tea.Cmd {
	drink := m.selectedDrink()
	if drink == nil {
		return nil
	}
	m.edit = NewEditDrinkVM(m.app, m.principal, drink)
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
		menusList, err := m.app.Menu.List(m.context(), menus.ListRequest{})
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
	if m.deleteTarget == nil {
		return nil
	}
	target := m.deleteTarget
	return func() tea.Msg {
		deleted, err := m.app.Drinks.Delete(m.context(), target.ID)
		if err != nil {
			return DeleteErrorMsg{Err: err}
		}
		return DrinkDeletedMsg{Drink: deleted}
	}
}

func (m *ListViewModel) context() *middleware.Context {
	return m.app.Context(context.Background(), m.principal)
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
