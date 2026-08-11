package tui

import (
	"fmt"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	orders "github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keyname"
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
	listModeConfirmingComplete
	listModeConfirmingCancel
	listModeTagging
	listModeFiltering
	listModePlacing
)

func (m listMode) isConfirming() bool {
	switch m {
	case listModeConfirmingComplete, listModeConfirmingCancel:
		return true
	case listModeBrowsing, listModeTagging, listModeFiltering, listModePlacing:
		return false
	}
	return false
}

// ListViewModel renders the orders list and detail panes.
type ListViewModel struct {
	app    *app.Session
	styles tui.ListViewStyles
	keys   listViewKeys

	dialogStyles dialog.DialogStyles
	dialogKeys   dialog.DialogKeys

	list      list.Model
	detail    *DetailViewModel
	mode      listMode
	dialog    *components.TaggedConfirm[tag.Tags]
	tags      *components.TagEditor[cedar.EntityUID, tag.Tags]
	filter    *filterVM
	place     *placeVM
	workflow  uint64
	loadToken uint64
	spinner   tui.Spinner
	loading   bool
	mutating  bool
	err       error
	actionErr error

	completeTarget *ordersmodels.Order
	cancelTarget   *ordersmodels.Order
	request        orders.ListRequest
	next           paging.Cursor
	history        []paging.Cursor
	restoreID      entity.OrderID
	projector      orders.ActionProjector
	actions        map[actions.ID]actions.State

	width       int
	height      int
	listWidth   int
	detailWidth int
}

func NewListViewModel(app *app.Session) *ListViewModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = styles.Standard.ListView.Selected
	delegate.Styles.SelectedDesc = styles.Standard.ListView.Selected

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Orders"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.Paginator.Type = paginator.Arabic
	l.SetFilteringEnabled(true)

	vm := &ListViewModel{
		app:          app,
		styles:       styles.Standard.ListView,
		keys:         newListViewKeys(),
		dialogStyles: styles.Standard.Dialog,
		dialogKeys:   keys.Standard.Dialog,
		list:         l,
		detail:       NewDetailViewModel(styles.Standard.ListView, app),
		projector:    orders.NewActionProjector(),
		loading:      true,
		request:      orders.ListRequest{Limit: paging.DefaultLimit},
	}
	vm.spinner = tui.NewSpinner("Loading orders...", vm.styles.Subtitle)
	if app != nil {
		vm.syncActions()
	}
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	if !m.actionEnabled(orders.ControlList) {
		m.loading = false
		return nil
	}
	m.loading = true
	return tea.Batch(m.spinner.Init(), m.loadOrders())
}

func (m *ListViewModel) Interaction() tui.Interaction {
	return tui.Interaction{
		HandlesBack:  m.mutating || m.mode != listModeBrowsing || m.list.SettingFilter(),
		CapturesText: m.mutating || m.list.SettingFilter() || m.mode == listModeTagging || m.mode == listModeFiltering || m.mode == listModePlacing,
	}
}

func (m *ListViewModel) Update(msg tea.Msg) (tui.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.DataInvalidatedMsg:
		if m.mode != listModeBrowsing || !m.actionEnabled(orders.ControlList) {
			return m, nil
		}
		m.loading, m.err = true, nil
		return m, tea.Batch(m.spinner.Init(), m.loadOrders())
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		if m.mode.isConfirming() {
			m.dialog.SetWidth(m.width)
		} else if m.mode == listModeTagging {
			m.tags.SetWidth(m.width)
		}
		return m, nil
	case OrderCompletedMsg:
		m.mutating = false
		m.mode = listModeBrowsing
		m.dialog = nil
		m.completeTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadOrders())
	case OrderCancelledMsg:
		m.mutating = false
		m.mode = listModeBrowsing
		m.dialog = nil
		m.cancelTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadOrders())
	case components.TagsSavedMsg[cedar.EntityUID, tag.Tags]:
		if m.mode != listModeTagging || m.tags == nil || !m.tags.Owns(msg.Target) {
			return m, nil
		}
		m.mode, m.tags, m.loading, m.err = listModeBrowsing, nil, true, nil
		return m, tea.Batch(m.spinner.Init(), m.loadOrders())
	case placeCatalogLoadedMsg:
		if m.mode == listModePlacing && m.place != nil && msg.workflow == m.place.workflow {
			m.place.setCatalog(msg)
		}
		return m, nil
	case orderPlacedMsg:
		if m.mode != listModePlacing || m.place == nil || msg.workflow != m.place.workflow {
			return m, nil
		}
		m.place.saving, m.place.err = false, msg.err
		if msg.err != nil {
			return m, nil
		}
		m.mode, m.place, m.loading, m.err = listModeBrowsing, nil, true, nil
		m.request.Cursor, m.next, m.history = "", "", nil
		return m, tea.Batch(m.spinner.Init(), m.loadOrders())
	case CompleteErrorMsg:
		m.mutating = false
		m.mode = listModeBrowsing
		m.dialog = nil
		m.completeTarget = nil
		m.err = msg.Err
		return m, nil
	case CancelErrorMsg:
		m.mutating = false
		m.mode = listModeBrowsing
		m.dialog = nil
		m.cancelTarget = nil
		m.err = msg.Err
		return m, nil
	case showCompleteDialogMsg:
		m.mode = listModeConfirmingComplete
		m.dialog = msg.dialog
		m.completeTarget = &msg.target
		m.cancelTarget = nil
		m.dialog.SetWidth(m.width)
		return m, m.dialog.Init()
	case showCancelDialogMsg:
		m.mode = listModeConfirmingCancel
		m.dialog = msg.dialog
		m.cancelTarget = &msg.target
		m.completeTarget = nil
		m.dialog.SetWidth(m.width)
		return m, m.dialog.Init()
	case dialog.ConfirmMsg:
		mode := m.mode
		m.mode = listModeBrowsing
		m.loading = false
		switch mode {
		case listModeConfirmingComplete:
			m.mutating = true
			return m, m.performComplete()
		case listModeConfirmingCancel:
			m.mutating = true
			return m, m.performCancel()
		case listModeBrowsing, listModeTagging, listModeFiltering, listModePlacing:
			panic(fmt.Sprintf("confirm message received in %v mode", m.mode))
		}
		return m, nil
	case dialog.CancelMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.completeTarget = nil
		m.cancelTarget = nil
		return m, nil
	case tea.KeyMsg:
		if m.mutating {
			return m, nil
		}
		if m.mode == listModeBrowsing && m.list.SettingFilter() {
			break
		}
		if m.mode == listModeTagging {
			if key.Matches(msg, m.keys.Back) && !m.tags.FormEditing() {
				if m.tags.Saving() {
					return m, nil
				}
				m.mode, m.tags = listModeBrowsing, nil
				return m, nil
			}
			break
		}
		if m.mode == listModeFiltering {
			if key.Matches(msg, m.keys.Back) && !m.filter.form.IsEditing() {
				m.mode, m.filter = listModeBrowsing, nil
				return m, nil
			}
			if msg.String() == keyname.Submit {
				if !m.actionEnabled(orders.ControlList) {
					return m, nil
				}
				req, err := m.filter.Request()
				if err != nil {
					return m, nil
				}
				m.request, m.next, m.history, m.restoreID = req, "", nil, entity.OrderID{}
				m.mode, m.filter, m.loading, m.err = listModeBrowsing, nil, true, nil
				return m, tea.Batch(m.spinner.Init(), m.loadOrders())
			}
			break
		}
		if m.mode == listModePlacing {
			if key.Matches(msg, m.keys.Back) && !m.place.editing {
				if m.place.saving || !m.place.mayClose() {
					return m, nil
				}
				m.workflow++
				m.mode, m.place = listModeBrowsing, nil
				return m, nil
			}
			if msg.String() == keyname.Submit {
				m.place.finishEdit(false)
				return m, m.place.submit()
			}
			break
		}
		if m.mode.isConfirming() {
			break
		}
		switch {
		case key.Matches(msg, m.keys.Refresh):
			if !m.actionEnabled(orders.ControlList) {
				return m, nil
			}
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Init(), m.loadOrders())
		case key.Matches(msg, m.keys.Complete):
			if !m.actionEnabled(orders.ControlComplete) {
				return m, nil
			}
			return m, m.startComplete()
		case key.Matches(msg, m.keys.Cancel):
			if !m.actionEnabled(orders.ControlCancel) {
				return m, nil
			}
			return m, m.startCancel()
		case key.Matches(msg, m.keys.Tags):
			if !m.actionEnabled(orders.ControlTags) {
				return m, nil
			}
			return m, m.startTags()
		case key.Matches(msg, m.keys.Create):
			if !m.actionEnabled(orders.ControlPlace) {
				return m, nil
			}
			m.workflow++
			m.mode = listModePlacing
			m.place = newPlaceVM(m.app, m.workflow)
			return m, m.place.Init()
		case msg.String() == "f":
			if !m.actionEnabled(orders.ControlList) {
				return m, nil
			}
			m.mode, m.filter = listModeFiltering, newFilterVM(m.request)
			return m, m.filter.Init()
		case msg.String() == "]" && m.next != "" && m.actionEnabled(orders.ControlList):
			m.history = append(m.history, m.request.Cursor)
			m.restoreID = selectedOrderID(m.selectedOrder())
			m.request.Cursor = m.next
			m.loading, m.err = true, nil
			return m, tea.Batch(m.spinner.Init(), m.loadOrders())
		case msg.String() == "[" && len(m.history) > 0 && m.actionEnabled(orders.ControlList):
			i := len(m.history) - 1
			m.request.Cursor = m.history[i]
			m.history = m.history[:i]
			m.restoreID = selectedOrderID(m.selectedOrder())
			m.loading, m.err = true, nil
			return m, tea.Batch(m.spinner.Init(), m.loadOrders())
		}
	case OrdersLoadedMsg:
		if msg.Token != m.loadToken {
			return m, nil
		}
		m.loading = false
		m.err = msg.Err
		if msg.Err != nil {
			return m, nil
		}
		m.next = msg.Next
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
		m.restoreSelection()
		m.syncDetail()
		m.syncActions()
		return m, nil
	}

	if m.mode.isConfirming() {
		var cmd tea.Cmd
		m.dialog, cmd = m.dialog.Update(msg)
		return m, cmd
	}
	if m.mode == listModeTagging {
		var cmd tea.Cmd
		m.tags, cmd = m.tags.Update(msg)
		return m, cmd
	}
	if m.mode == listModeFiltering {
		return m, m.filter.Update(msg)
	}
	if m.mode == listModePlacing {
		return m, m.place.Update(msg)
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	if !m.actionEnabled(orders.ControlList) {
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

	if m.mode.isConfirming() {
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
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.filter.View())
	}
	if m.mode == listModePlacing {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.place.View())
	}

	listView := m.list.View()
	if m.mutating {
		listView = m.styles.Subtitle.Render("Updating order…") + "\n\n" + listView
	}
	if m.err != nil {
		listView = m.styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.err))
	} else if m.actionErr != nil {
		listView = m.styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.actionErr))
	}
	listView = m.styles.ListPane.Width(tui.PaneStyleWidth(m.styles.ListPane, m.listWidth)).Render(listView)

	detailView := m.detail.View()
	detailView = m.styles.DetailPane.Width(tui.PaneStyleWidth(m.styles.DetailPane, m.detailWidth)).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	if m.mode.isConfirming() {
		return []key.Binding{m.dialogKeys.Confirm, m.keys.Back, m.dialogKeys.Switch}
	}
	if m.mode == listModeTagging {
		return []key.Binding{keys.Standard.Submit, m.keys.Back}
	}
	if m.mode == listModeFiltering {
		return []key.Binding{keys.Standard.Submit, m.keys.Back}
	}
	if m.mode == listModePlacing {
		return []key.Binding{keys.Standard.Submit, m.keys.Back}
	}
	bindings := []key.Binding{}
	if m.actionEnabled(orders.ControlList) {
		bindings = append(bindings, m.keys.Up, m.keys.Down, m.list.KeyMap.PrevPage, m.list.KeyMap.NextPage)
	}
	bindings = append(bindings, m.visibleBindings([]key.Binding{m.keys.Create, m.keys.Complete, m.keys.Cancel, m.keys.Tags})...)
	if m.actionEnabled(orders.ControlList) {
		bindings = append(bindings, m.keys.Refresh)
	}
	return append(bindings, m.keys.Back)
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	if m.mode.isConfirming() {
		return [][]key.Binding{
			{m.dialogKeys.Confirm, m.keys.Back},
			{m.dialogKeys.Switch},
		}
	}
	if m.mode == listModeTagging {
		return [][]key.Binding{{keys.Standard.Submit, m.keys.Back}}
	}
	if m.mode == listModeFiltering {
		return [][]key.Binding{{keys.Standard.Submit, m.keys.Back}}
	}
	if m.mode == listModePlacing {
		return [][]key.Binding{{keys.Standard.Submit, m.keys.Back}}
	}
	navigation, paging, last := []key.Binding{}, []key.Binding{}, []key.Binding{m.keys.Back}
	if m.actionEnabled(orders.ControlList) {
		navigation = append(navigation, m.keys.Up, m.keys.Down, m.keys.Enter)
		paging = append(paging, m.list.KeyMap.PrevPage, m.list.KeyMap.NextPage)
		last = append([]key.Binding{m.keys.Refresh}, last...)
	}
	return m.visibleBindingGroups([][]key.Binding{navigation, paging, []key.Binding{m.keys.Create, m.keys.Complete, m.keys.Cancel, m.keys.Tags}, last})
}

func (m *ListViewModel) syncActions() {
	states, err := m.projector.Project(m.app.Context(), m.app.Context().Principal(), m.selectedOrder())
	if err != nil {
		m.actions = nil
		m.actionErr = err
		m.loading = false
		m.detail.SetActions(nil)
		return
	}
	m.actionErr = nil
	m.actions = make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		m.actions[state.ID] = state
	}
	m.detail.SetActions(m.actions)
	if !m.actionEnabled(orders.ControlList) {
		m.loading = false
		m.list.SetItems(nil)
		m.syncDetail()
	}
}

func (m *ListViewModel) actionEnabled(id actions.ID) bool {
	state, ok := m.actions[id]
	return ok && state.Visible && state.Enabled
}

func (m *ListViewModel) actionVisibleForBinding(binding key.Binding) bool {
	switch binding.Help().Key {
	case m.keys.Create.Help().Key:
		return m.actions[orders.ControlPlace].Visible
	case m.keys.Complete.Help().Key:
		return m.actions[orders.ControlComplete].Visible
	case m.keys.Cancel.Help().Key:
		return m.actions[orders.ControlCancel].Visible
	case m.keys.Tags.Help().Key:
		return m.actions[orders.ControlTags].Visible
	default:
		return true
	}
}

func (m *ListViewModel) visibleBindings(in []key.Binding) []key.Binding {
	out := make([]key.Binding, 0, len(in))
	for _, binding := range in {
		if m.actionVisibleForBinding(binding) {
			out = append(out, binding)
		}
	}
	return out
}

func (m *ListViewModel) visibleBindingGroups(in [][]key.Binding) [][]key.Binding {
	out := make([][]key.Binding, 0, len(in))
	for _, group := range in {
		if visible := m.visibleBindings(group); len(visible) > 0 {
			out = append(out, visible)
		}
	}
	return out
}

func (m *ListViewModel) loadOrders() tea.Cmd {
	m.loadToken++
	token := m.loadToken
	return func() tea.Msg {
		ordersList, err := m.app.Orders.List(m.context(), m.request)
		if err != nil {
			return OrdersLoadedMsg{Err: err, Token: token}
		}

		orders := make([]ordersmodels.Order, 0, len(ordersList.Items))
		for i, order := range ordersList.Items {
			if order == nil {
				return OrdersLoadedMsg{Err: errors.Internalf("order %d missing", i), Token: token}
			}
			orders = append(orders, *order)
		}

		return OrdersLoadedMsg{Orders: orders, Next: ordersList.Next, Token: token}
	}
}

func selectedOrderID(order *ordersmodels.Order) entity.OrderID {
	if order == nil {
		return entity.OrderID{}
	}
	return order.ID
}
func (m *ListViewModel) restoreSelection() {
	if m.restoreID.IsZero() {
		return
	}
	for i, raw := range m.list.Items() {
		if item, ok := raw.(orderItem); ok && item.Value.ID == m.restoreID {
			m.list.Select(i)
			break
		}
	}
	m.restoreID = entity.OrderID{}
}

type showCompleteDialogMsg struct {
	dialog *components.TaggedConfirm[tag.Tags]
	target ordersmodels.Order
}

type showCancelDialogMsg struct {
	dialog *components.TaggedConfirm[tag.Tags]
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
		case ordersmodels.OrderStatusPending:
		case ordersmodels.OrderStatusBlocked:
			return CompleteErrorMsg{Err: errors.FailedPreconditionf("reserved stock is short")}
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
		return showCompleteDialogMsg{dialog: components.NewTaggedConfirm(order.Tags.Canonical().String(), tag.ParseCollection, confirm), target: *order}
	}
}

func (m *ListViewModel) performComplete() tea.Cmd {
	target := *m.completeTarget
	desired, err := m.dialog.DesiredTags()
	if err != nil {
		return func() tea.Msg { return CompleteErrorMsg{Err: err} }
	}
	return func() tea.Msg {
		updated, err := app.RunTaggedMutation(m.app.App, m.context(), desired, func(ctx *middleware.Context) (*ordersmodels.Order, error) {
			return m.app.Orders.Complete(ctx, &ordersmodels.Order{ID: target.ID})
		})
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
		case ordersmodels.OrderStatusPending, ordersmodels.OrderStatusBlocked:
		case ordersmodels.OrderStatusCompleted:
			return CancelErrorMsg{Err: errors.Invalidf("cannot cancel a completed order")}
		case ordersmodels.OrderStatusCancelled:
			return CancelErrorMsg{Err: errors.Invalidf("order is already cancelled")}
		}
		message := fmt.Sprintf(
			"Cancel order #%s?\n\nThis order has %d item(s).\nReserved inventory will be released.",
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
		return showCancelDialogMsg{dialog: components.NewTaggedConfirm(order.Tags.Canonical().String(), tag.ParseCollection, confirm), target: *order}
	}
}

func (m *ListViewModel) performCancel() tea.Cmd {
	target := *m.cancelTarget
	desired, err := m.dialog.DesiredTags()
	if err != nil {
		return func() tea.Msg { return CancelErrorMsg{Err: err} }
	}
	return func() tea.Msg {
		updated, err := app.RunTaggedMutation(m.app.App, m.context(), desired, func(ctx *middleware.Context) (*ordersmodels.Order, error) {
			return m.app.Orders.Cancel(ctx, &ordersmodels.Order{ID: target.ID})
		})
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
	order := item.Value
	return &order
}

func (m *ListViewModel) startTags() tea.Cmd {
	order := m.selectedOrder()
	if order == nil {
		return nil
	}
	m.mode = listModeTagging
	m.tags = components.NewTagEditor(m.app.ReplaceTags, tag.ParseCollection, order.EntityUID(), order.ID.String(), order.Tags.Canonical().String())
	m.tags.SetWidth(m.width)
	return m.tags.Init()
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
	m.listWidth = listWidth
	m.detailWidth = detailWidth
}

func (m *ListViewModel) syncDetail() {
	item, ok := m.list.SelectedItem().(orderItem)
	if !ok {
		m.detail.SetOrder(optional.None[ordersmodels.Order]())
		return
	}
	m.detail.SetOrder(optional.Some(item.Value))
}

func (m *ListViewModel) menuName(menuID entity.MenuID) (string, error) {
	if menuID.IsZero() {
		return "", errors.Internalf("order missing menu id")
	}
	menu, err := m.app.Menus.Get(m.context(), menuID)
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
