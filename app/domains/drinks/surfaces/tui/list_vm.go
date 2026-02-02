package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	drinksqueries "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListViewModel renders the drinks list and detail panes.
type ListViewModel struct {
	app    *app.App
	ctx    *middleware.Context
	styles tui.ListViewStyles
	keys   tui.ListViewKeys

	drinksQueries *drinksqueries.Queries

	list    list.Model
	detail  *DetailViewModel
	spinner components.Spinner
	loading bool
	err     error

	width       int
	height      int
	listWidth   int
	detailWidth int
}

func NewListViewModel(app *app.App, ctx *middleware.Context, styles tui.ListViewStyles, keys tui.ListViewKeys) *ListViewModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = styles.Selected
	delegate.Styles.SelectedDesc = styles.Selected

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Drinks"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)

	vm := &ListViewModel{
		app:           app,
		ctx:           ctx,
		styles:        styles,
		keys:          keys,
		drinksQueries: drinksqueries.New(),
		list:          l,
		detail:        NewDetailViewModel(styles, ctx),
		loading:       true,
	}
	vm.spinner = components.NewSpinner("Loading drinks...", styles.Subtitle)
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
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Init(), m.loadDrinks())
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

	listView := m.list.View()
	if m.err != nil {
		listView = m.styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.err))
	}
	listView = m.styles.ListPane.Width(m.listWidth).Render(listView)

	detailView := m.detail.View()
	detailView = m.styles.DetailPane.Width(m.detailWidth).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Refresh, m.keys.Back}
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.Up, m.keys.Down, m.keys.Enter},
		{m.keys.Refresh, m.keys.Back},
	}
}

func (m *ListViewModel) loadDrinks() tea.Cmd {
	return func() tea.Msg {
		drinksList, err := m.drinksQueries.List(m.ctx, drinksdao.ListFilter{})
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
