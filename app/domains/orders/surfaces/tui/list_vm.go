package tui

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	menusqueries "github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListViewModel renders the orders list and detail panes.
type ListViewModel struct {
	app    *app.App
	styles tui.ListViewStyles
	keys   tui.ListViewKeys

	dialogStyles dialog.DialogStyles
	dialogKeys   dialog.DialogKeys

	ordersQueries *queries.Queries
	menuQueries   *menusqueries.Queries

	list    list.Model
	detail  *DetailViewModel
	dialog  *dialog.ConfirmDialog
	spinner components.Spinner
	loading bool
	err     error

	completeTarget *ordersmodels.Order
	cancelTarget   *ordersmodels.Order

	width       int
	height      int
	listWidth   int
	detailWidth int
}

func NewListViewModel(app *app.App) *ListViewModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = tuistyles.App.ListView.Selected
	delegate.Styles.SelectedDesc = tuistyles.App.ListView.Selected

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Orders"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)

	vm := &ListViewModel{
		app:           app,
		styles:        tuistyles.App.ListView,
		keys:          tuikeys.App.ListView,
		dialogStyles:  tuistyles.App.Dialog,
		dialogKeys:    tuikeys.App.Dialog,
		ordersQueries: queries.New(),
		menuQueries:   menusqueries.New(),
		list:          l,
		detail:        NewDetailViewModel(tuistyles.App.ListView, app),
		loading:       true,
	}
	vm.spinner = components.NewSpinner("Loading orders...", vm.styles.Subtitle)
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
		if m.dialog != nil {
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case OrderCompletedMsg:
		m.dialog = nil
		m.completeTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadOrders())
	case OrderCancelledMsg:
		m.dialog = nil
		m.cancelTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadOrders())
	case CompleteErrorMsg:
		m.dialog = nil
		m.completeTarget = nil
		m.err = msg.Err
		return m, nil
	case CancelErrorMsg:
		m.dialog = nil
		m.cancelTarget = nil
		m.err = msg.Err
		return m, nil
	case showCompleteDialogMsg:
		m.dialog = msg.dialog
		m.completeTarget = &msg.target
		m.cancelTarget = nil
		if m.dialog != nil {
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case showCancelDialogMsg:
		m.dialog = msg.dialog
		m.cancelTarget = &msg.target
		m.completeTarget = nil
		if m.dialog != nil {
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case dialog.ConfirmMsg:
		m.dialog = nil
		if m.completeTarget != nil {
			return m, m.performComplete()
		}
		if m.cancelTarget != nil {
			return m, m.performCancel()
		}
		return m, nil
	case dialog.CancelMsg:
		m.dialog = nil
		m.completeTarget = nil
		m.cancelTarget = nil
		return m, nil
	case tea.KeyMsg:
		if m.dialog != nil {
			break
		}
		switch {
		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Init(), m.loadOrders())
		case key.Matches(msg, m.keys.Complete):
			return m, m.startComplete()
		case key.Matches(msg, m.keys.CancelOrder):
			return m, m.startCancel()
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

	if m.dialog != nil {
		var cmd tea.Cmd
		m.dialog, cmd = m.dialog.Update(msg)
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
	detailView = m.styles.DetailPane.Width(m.detailWidth).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	if m.dialog != nil {
		return []key.Binding{m.dialogKeys.Confirm, m.keys.Back, m.dialogKeys.Switch}
	}
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Complete, m.keys.CancelOrder, m.keys.Refresh, m.keys.Back}
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	if m.dialog != nil {
		return [][]key.Binding{
			{m.dialogKeys.Confirm, m.keys.Back},
			{m.dialogKeys.Switch},
		}
	}
	return [][]key.Binding{
		{m.keys.Up, m.keys.Down, m.keys.Enter},
		{m.keys.Complete, m.keys.CancelOrder},
		{m.keys.Refresh, m.keys.Back},
	}
}

func (m *ListViewModel) loadOrders() tea.Cmd {
	return func() tea.Msg {
		ordersList, err := m.ordersQueries.List(m.context(), queries.ListFilter{})
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

type showCompleteDialogMsg struct {
	dialog *dialog.ConfirmDialog
	target ordersmodels.Order
}

type showCancelDialogMsg struct {
	dialog *dialog.ConfirmDialog
	target ordersmodels.Order
}

func (m *ListViewModel) startComplete() tea.Cmd {
	order := m.selectedOrder()
	if order == nil {
		return nil
	}
	return m.showCompleteConfirm(order)
}

func (m *ListViewModel) showCompleteConfirm(order *ordersmodels.Order) tea.Cmd {
	if order == nil {
		return nil
	}
	return func() tea.Msg {
		switch order.Status {
		case ordersmodels.OrderStatusPending, ordersmodels.OrderStatusPreparing:
		case ordersmodels.OrderStatusCompleted:
			return CompleteErrorMsg{Err: errors.Invalidf("order is already completed")}
		case ordersmodels.OrderStatusCancelled:
			return CompleteErrorMsg{Err: errors.Invalidf("cannot complete a cancelled order")}
		}
		message := fmt.Sprintf(
			"Complete order #%s?\n\n%d item(s) will be marked as served.\nInventory will be decremented accordingly.",
			truncateID(order.ID.String()),
			len(order.Items),
		)
		confirm := dialog.NewConfirmDialog(
			"Complete Order",
			message,
			dialog.WithConfirmText("Complete"),
			dialog.WithStyles(m.dialogStyles),
			dialog.WithKeys(m.dialogKeys),
		)
		return showCompleteDialogMsg{dialog: confirm, target: *order}
	}
}

func (m *ListViewModel) performComplete() tea.Cmd {
	if m.completeTarget == nil {
		return nil
	}
	target := m.completeTarget
	return func() tea.Msg {
		updated, err := m.app.Orders.Complete(m.context(), &ordersmodels.Order{ID: target.ID})
		if err != nil {
			return CompleteErrorMsg{Err: err}
		}
		return OrderCompletedMsg{Order: updated}
	}
}

func (m *ListViewModel) startCancel() tea.Cmd {
	order := m.selectedOrder()
	if order == nil {
		return nil
	}
	return m.showCancelConfirm(order)
}

func (m *ListViewModel) showCancelConfirm(order *ordersmodels.Order) tea.Cmd {
	if order == nil {
		return nil
	}
	return func() tea.Msg {
		switch order.Status {
		case ordersmodels.OrderStatusPending, ordersmodels.OrderStatusPreparing:
		case ordersmodels.OrderStatusCompleted:
			return CancelErrorMsg{Err: errors.Invalidf("cannot cancel a completed order")}
		case ordersmodels.OrderStatusCancelled:
			return CancelErrorMsg{Err: errors.Invalidf("order is already cancelled")}
		}
		message := fmt.Sprintf(
			"Cancel order #%s?\n\nThis order has %d item(s).\nNo inventory changes will be made.",
			truncateID(order.ID.String()),
			len(order.Items),
		)
		confirm := dialog.NewConfirmDialog(
			"Cancel Order",
			message,
			dialog.WithDangerous(),
			dialog.WithFocusCancel(),
			dialog.WithConfirmText("Cancel Order"),
			dialog.WithStyles(m.dialogStyles),
			dialog.WithKeys(m.dialogKeys),
		)
		return showCancelDialogMsg{dialog: confirm, target: *order}
	}
}

func (m *ListViewModel) performCancel() tea.Cmd {
	if m.cancelTarget == nil {
		return nil
	}
	target := m.cancelTarget
	return func() tea.Msg {
		updated, err := m.app.Orders.Cancel(m.context(), &ordersmodels.Order{ID: target.ID})
		if err != nil {
			return CancelErrorMsg{Err: err}
		}
		return OrderCancelledMsg{Order: updated}
	}
}

func (m *ListViewModel) selectedOrder() *ordersmodels.Order {
	item, ok := m.list.SelectedItem().(orderItem)
	if !ok {
		return nil
	}
	order := item.order
	return &order
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
	menu, err := m.menuQueries.Get(m.context(), menuID)
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

func (m *ListViewModel) context() *middleware.Context {
	return m.app.Context()
}
