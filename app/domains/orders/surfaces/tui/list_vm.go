package tui

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	menusqueries "github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	ordersdao "github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	ordersqueries "github.com/TheFellow/go-modular-monolith/app/domains/orders/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListViewModel renders the orders list and detail panes.
type ListViewModel struct {
	app    *app.App
	ctx    *middleware.Context
	styles tui.ListViewStyles
	keys   tui.ListViewKeys

	ordersQueries *ordersqueries.Queries
	menuQueries   *menusqueries.Queries

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
	l.Title = "Orders"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)

	vm := &ListViewModel{
		app:           app,
		ctx:           ctx,
		styles:        styles,
		keys:          keys,
		ordersQueries: ordersqueries.New(),
		menuQueries:   menusqueries.New(),
		list:          l,
		detail:        NewDetailViewModel(styles, ctx),
		loading:       true,
	}
	vm.spinner = components.NewSpinner("Loading orders...", styles.Subtitle)
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Init(), m.loadOrders())
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
			return m, tea.Batch(m.spinner.Init(), m.loadOrders())
		}
	case OrdersLoadedMsg:
		m.loading = false
		m.err = msg.Err
		items := make([]list.Item, 0, len(msg.Orders))
		for _, order := range msg.Orders {
			menuName, err := m.menuName(order.MenuID)
			if err != nil {
				m.err = err
				break
			}
			items = append(items, newOrderItem(order, menuName, m.styles))
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

func (m *ListViewModel) loadOrders() tea.Cmd {
	return func() tea.Msg {
		ordersList, err := m.ordersQueries.List(m.ctx, ordersdao.ListFilter{})
		if err != nil {
			return OrdersLoadedMsg{Err: err}
		}

		orders := make([]ordersmodels.Order, 0, len(ordersList))
		for i, order := range ordersList {
			if order == nil {
				return OrdersLoadedMsg{Err: errors.Internalf("order %d missing", i)}
			}
			orders = append(orders, *order)
		}

		return OrdersLoadedMsg{Orders: orders}
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
	item, ok := m.list.SelectedItem().(orderItem)
	if !ok {
		m.detail.SetOrder(optional.None[ordersmodels.Order]())
		return
	}
	m.detail.SetOrder(optional.Some(item.order))
}

func (m *ListViewModel) menuName(menuID entity.MenuID) (string, error) {
	if menuID.IsZero() {
		return "", errors.Internalf("order missing menu id")
	}
	menu, err := m.menuQueries.Get(m.ctx, menuID)
	if err != nil {
		return "", errors.Internalf("load menu %s: %w", menuID.String(), err)
	}
	if menu == nil {
		return "", errors.Internalf("menu %s missing", menuID.String())
	}
	name := strings.TrimSpace(menu.Name)
	if name == "" {
		return "", errors.Internalf("menu %s missing name", menuID.String())
	}
	return name, nil
}
