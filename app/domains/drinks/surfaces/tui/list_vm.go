package tui

import (
	"context"
	"errors"
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	drinks "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListViewStyles contains styles needed by the drinks list view.
type ListViewStyles struct {
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Muted       lipgloss.Style
	Selected    lipgloss.Style
	ListPane    lipgloss.Style
	DetailPane  lipgloss.Style
	ErrorText   lipgloss.Style
	WarningText lipgloss.Style
}

// ListViewKeys contains key bindings needed by the drinks list view.
type ListViewKeys struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Refresh key.Binding
	Back    key.Binding
}

// ListViewModel renders the drinks list and detail panes.
type ListViewModel struct {
	app    *app.App
	ctx    *middleware.Context
	styles ListViewStyles
	keys   ListViewKeys

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

func NewListViewModel(app *app.App, ctx *middleware.Context, styles ListViewStyles, keys ListViewKeys) *ListViewModel {
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
		app:     app,
		ctx:     ctx,
		styles:  styles,
		keys:    keys,
		list:    l,
		detail:  NewDetailViewModel(styles),
		loading: true,
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
		if m.app == nil {
			return DrinksLoadedMsg{Err: errors.New("drinks view requires app")}
		}

		ctx := m.ctx
		if ctx == nil {
			ctx = m.app.Context(context.Background(), authn.Anonymous())
		}

		drinksList, err := m.app.Drinks.List(ctx, drinks.ListRequest{})
		if err != nil {
			return DrinksLoadedMsg{Err: err}
		}

		items := make([]models.Drink, 0, len(drinksList))
		for _, drink := range drinksList {
			if drink == nil {
				continue
			}
			items = append(items, *drink)
		}

		return DrinksLoadedMsg{Drinks: items}
	}
}

func (m *ListViewModel) renderLoading() string {
	content := m.spinner.View()
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}

func (m *ListViewModel) setSize(width, height int) {
	m.width = width
	m.height = height

	if width <= 0 {
		return
	}

	listWidth := int(float64(width) * 0.6)
	if listWidth < 32 {
		listWidth = width / 2
	}
	detailWidth := width - listWidth
	if detailWidth < 24 {
		detailWidth = width - 24
		if detailWidth < 0 {
			detailWidth = 0
		}
		listWidth = width - detailWidth
	}

	m.list.SetSize(listWidth, height)
	m.detail.SetSize(detailWidth, height)
	m.listWidth = listWidth
	m.detailWidth = detailWidth
}

func (m *ListViewModel) syncDetail() {
	item, ok := m.list.SelectedItem().(drinkItem)
	if !ok {
		m.detail.SetDrink(nil)
		return
	}
	drink := item.drink
	m.detail.SetDrink(&drink)
}
