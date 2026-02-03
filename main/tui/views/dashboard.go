package views

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/main/tui/keys"
	"github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/cedar-policy/cedar-go"
)

// Dashboard is the main navigation hub of the TUI.
type Dashboard struct {
	app       *app.App
	principal cedar.EntityUID
	styles    styles.DashboardStyles
	keys      keys.DashboardKeys
	width     int
	height    int

	loading bool
	spinner components.Spinner
	data    *DashboardData
	err     error
}

const (
	dashboardEdgeMargin = 2
	dashboardRecentMax  = 10
	dashboardUnknown    = -1
)

type DashboardData struct {
	DrinkCount       int
	IngredientCount  int
	InventoryCount   int
	MenuCount        int
	DraftMenus       int
	PublishedMenus   int
	LowStockCount    int
	OrderCount       int
	PendingOrders    int
	AuditCount       int
	AuditCountCapped bool
	RecentActivity   []AuditSummary
}

type AuditSummary struct {
	Timestamp string
	Actor     string
	Action    string
}

type DashboardLoadedMsg struct {
	Data *DashboardData
	Err  error
}

// NewDashboard creates a new Dashboard view.
func NewDashboard(app *app.App, principal cedar.EntityUID) *Dashboard {
	d := &Dashboard{
		app:       app,
		principal: principal,
		styles:    styles.App.Dashboard,
		keys:      keys.App.Dashboard,
		loading:   true,
	}
	d.spinner = components.NewSpinner("Loading dashboard...", d.styles.Subtitle)
	return d
}

// Init implements ViewModel.
func (d *Dashboard) Init() tea.Cmd {
	d.loading = true
	return tea.Batch(d.spinner.Init(), d.loadData())
}

// Update implements ViewModel.
func (d *Dashboard) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.Nav1):
			return d, navigateTo(ViewDrinks)
		case key.Matches(msg, d.keys.Nav2):
			return d, navigateTo(ViewIngredients)
		case key.Matches(msg, d.keys.Nav3):
			return d, navigateTo(ViewInventory)
		case key.Matches(msg, d.keys.Nav4):
			return d, navigateTo(ViewMenus)
		case key.Matches(msg, d.keys.Nav5):
			return d, navigateTo(ViewOrders)
		case key.Matches(msg, d.keys.Nav6):
			return d, navigateTo(ViewAudit)
		}
	case DashboardLoadedMsg:
		d.loading = false
		d.data = msg.Data
		d.err = msg.Err
		return d, nil
	}

	if d.loading {
		var cmd tea.Cmd
		d.spinner, cmd = d.spinner.Update(msg)
		return d, cmd
	}

	return d, nil
}

// View implements ViewModel.
func (d *Dashboard) View() string {
	if d.loading {
		return d.renderLoading()
	}

	header := d.styles.Title.Render("Dashboard")
	subtitle := d.styles.Subtitle.Render("Select a workspace to continue")
	cards := d.renderCountCards()

	cardWidth, columnCount := d.layoutConfig()
	content := d.renderCards(cards, cardWidth, columnCount)

	activity := d.renderRecentActivity()
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		subtitle,
		"",
		content,
		"",
		d.styles.Subtitle.Render("Recent Activity"),
		activity,
	)
	if d.width > 0 && d.height > 0 {
		return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, body)
	}

	return body
}

// ShortHelp implements ViewModel.
func (d *Dashboard) ShortHelp() []key.Binding {
	return []key.Binding{
		d.keys.Nav1, d.keys.Nav2, d.keys.Nav3,
		d.keys.Nav4, d.keys.Nav5, d.keys.Nav6,
	}
}

// FullHelp implements ViewModel.
func (d *Dashboard) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{d.keys.Nav1, d.keys.Nav2, d.keys.Nav3},
		{d.keys.Nav4, d.keys.Nav5, d.keys.Nav6},
		{d.keys.Help, d.keys.Quit},
	}
}

type dashboardCard struct {
	key   string
	title string
	desc  string
	count string
}

func (d *Dashboard) loadData() tea.Cmd {
	return func() tea.Msg {
		if d.app == nil {
			return DashboardLoadedMsg{Err: errors.New("dashboard requires app")}
		}

		ctx := d.context()

		data := &DashboardData{
			DrinkCount:      dashboardUnknown,
			IngredientCount: dashboardUnknown,
			InventoryCount:  dashboardUnknown,
			MenuCount:       dashboardUnknown,
			DraftMenus:      dashboardUnknown,
			PublishedMenus:  dashboardUnknown,
			LowStockCount:   dashboardUnknown,
			OrderCount:      dashboardUnknown,
			PendingOrders:   dashboardUnknown,
			AuditCount:      dashboardUnknown,
		}
		var loadErr error

		if count, err := d.app.Drinks.Count(ctx, drinks.ListRequest{}); err != nil {
			loadErr = firstErr(loadErr, err)
		} else {
			data.DrinkCount = count
		}

		if count, err := d.app.Ingredients.Count(ctx, ingredients.ListRequest{}); err != nil {
			loadErr = firstErr(loadErr, err)
		} else {
			data.IngredientCount = count
		}

		if count, err := d.app.Inventory.Count(ctx, inventory.ListRequest{}); err != nil {
			loadErr = firstErr(loadErr, err)
		} else {
			data.InventoryCount = count
		}

		if count, err := d.app.Inventory.Count(ctx, inventory.ListRequest{LowStock: optional.Some(0.0)}); err != nil {
			loadErr = firstErr(loadErr, err)
		} else {
			data.LowStockCount = count
		}

		if count, err := d.app.Menu.Count(ctx, menus.ListRequest{}); err != nil {
			loadErr = firstErr(loadErr, err)
		} else {
			data.MenuCount = count
		}

		if count, err := d.app.Menu.Count(ctx, menus.ListRequest{Status: menumodels.MenuStatusDraft}); err != nil {
			loadErr = firstErr(loadErr, err)
		} else {
			data.DraftMenus = count
		}

		if count, err := d.app.Menu.Count(ctx, menus.ListRequest{Status: menumodels.MenuStatusPublished}); err != nil {
			loadErr = firstErr(loadErr, err)
		} else {
			data.PublishedMenus = count
		}

		if count, err := d.app.Orders.Count(ctx, orders.ListRequest{}); err != nil {
			loadErr = firstErr(loadErr, err)
		} else {
			data.OrderCount = count
		}

		pendingCount := 0
		if count, err := d.app.Orders.Count(ctx, orders.ListRequest{Status: ordersmodels.OrderStatusPending}); err != nil {
			loadErr = firstErr(loadErr, err)
			pendingCount = dashboardUnknown
		} else {
			pendingCount = count
		}
		if count, err := d.app.Orders.Count(ctx, orders.ListRequest{Status: ordersmodels.OrderStatusPreparing}); err != nil {
			loadErr = firstErr(loadErr, err)
			if pendingCount >= 0 {
				pendingCount = dashboardUnknown
			}
		} else if pendingCount >= 0 {
			pendingCount += count
		}
		if pendingCount >= 0 {
			data.PendingOrders = pendingCount
		}

		if count, err := d.app.Audit.Count(ctx, audit.ListRequest{}); err != nil {
			loadErr = firstErr(loadErr, err)
		} else {
			data.AuditCount = count
			data.AuditCountCapped = false
		}

		if entries, err := d.app.Audit.List(ctx, audit.ListRequest{Limit: dashboardRecentMax}); err != nil {
			loadErr = firstErr(loadErr, err)
		} else {
			data.RecentActivity = make([]AuditSummary, 0, len(entries))
			for _, entry := range entries {
				ts := entry.CompletedAt
				if ts.IsZero() {
					ts = entry.StartedAt
				}
				data.RecentActivity = append(data.RecentActivity, AuditSummary{
					Timestamp: ts.Format("15:04"),
					Actor:     entry.Principal.String(),
					Action:    entry.Action,
				})
			}
		}

		return DashboardLoadedMsg{Data: data, Err: loadErr}
	}
}

func (d *Dashboard) context() *middleware.Context {
	return d.app.Context(context.Background(), d.principal)
}

func (d *Dashboard) renderLoading() string {
	content := d.spinner.View()
	if d.width > 0 && d.height > 0 {
		return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}

func (d *Dashboard) renderCountCards() []dashboardCard {
	data := d.data
	if data == nil {
		data = &DashboardData{
			DrinkCount:      dashboardUnknown,
			IngredientCount: dashboardUnknown,
			InventoryCount:  dashboardUnknown,
			MenuCount:       dashboardUnknown,
			DraftMenus:      dashboardUnknown,
			PublishedMenus:  dashboardUnknown,
			LowStockCount:   dashboardUnknown,
			OrderCount:      dashboardUnknown,
			PendingOrders:   dashboardUnknown,
			AuditCount:      dashboardUnknown,
		}
	}

	return []dashboardCard{
		{key: "1", title: "Drinks", desc: "Manage drink recipes", count: formatCount(data.DrinkCount)},
		{key: "2", title: "Ingredients", desc: "Catalog ingredients", count: formatCount(data.IngredientCount)},
		{key: "3", title: "Inventory", desc: d.inventorySubtitle(data), count: formatCount(data.InventoryCount)},
		{key: "4", title: "Menus", desc: d.menuSubtitle(data), count: formatCount(data.MenuCount)},
		{key: "5", title: "Orders", desc: d.ordersSubtitle(data), count: formatCount(data.OrderCount)},
		{key: "6", title: "Audit", desc: "Inspect audit logs", count: d.auditCountLabel(data)},
	}
}

func (d *Dashboard) inventorySubtitle(data *DashboardData) string {
	if data.LowStockCount >= 0 {
		return "Low stock: " + formatCount(data.LowStockCount)
	}
	return "Track stock levels"
}

func (d *Dashboard) menuSubtitle(data *DashboardData) string {
	if data.DraftMenus >= 0 && data.PublishedMenus >= 0 {
		return fmt.Sprintf("Draft %s • Published %s", formatCount(data.DraftMenus), formatCount(data.PublishedMenus))
	}
	return "Build drink menus"
}

func (d *Dashboard) ordersSubtitle(data *DashboardData) string {
	if data.PendingOrders >= 0 {
		return "Pending: " + formatCount(data.PendingOrders)
	}
	return "Review orders"
}

func (d *Dashboard) auditCountLabel(data *DashboardData) string {
	if data.AuditCount < 0 {
		return "?"
	}
	if data.AuditCountCapped {
		return fmt.Sprintf("%d+", data.AuditCount)
	}
	return strconv.Itoa(data.AuditCount)
}

func (d *Dashboard) renderRecentActivity() string {
	if d.data == nil || len(d.data.RecentActivity) == 0 {
		return d.styles.Subtitle.Render("No recent activity")
	}

	rows := make([]string, 0, len(d.data.RecentActivity))
	for _, entry := range d.data.RecentActivity {
		rows = append(rows, fmt.Sprintf("%s  %s  %s", entry.Timestamp, entry.Actor, entry.Action))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (d *Dashboard) layoutConfig() (int, int) {
	if d.width <= 0 {
		return 34, 2
	}

	gap := 2
	minCardWidth := 28
	availableWidth := d.width - (dashboardEdgeMargin * 2)
	if availableWidth < 0 {
		availableWidth = 0
	}
	available := availableWidth - gap
	if available >= minCardWidth*2 {
		return available / 2, 2
	}

	return availableWidth, 1
}

func (d *Dashboard) renderCards(cards []dashboardCard, width int, columns int) string {
	if columns <= 1 {
		rows := make([]string, 0, len(cards))
		for _, card := range cards {
			rows = append(rows, d.renderCard(card, width))
		}
		return lipgloss.JoinVertical(lipgloss.Left, rows...)
	}

	gap := lipgloss.NewStyle().Width(2).Render("")
	rows := make([]string, 0, (len(cards)+1)/2)
	for i := 0; i < len(cards); i += 2 {
		left := d.renderCard(cards[i], width)
		if i+1 >= len(cards) {
			rows = append(rows, left)
			break
		}
		right := d.renderCard(cards[i+1], width)
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (d *Dashboard) renderCard(card dashboardCard, width int) string {
	label := d.styles.HelpKey.Render("[" + card.key + "]")
	titleText := card.title
	if card.count != "" {
		titleText = fmt.Sprintf("%s (%s)", card.title, card.count)
	}
	title := d.styles.Title.Render(titleText)
	desc := d.styles.Subtitle.Render(card.desc)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Left, label, " ", title),
		desc,
	)

	style := d.styles.Card
	if width > 0 {
		style = style.Width(width)
	}

	return style.Render(content)
}

func firstErr(existing error, next error) error {
	if existing == nil {
		return next
	}
	return existing
}

func formatCount(count int) string {
	if count < 0 {
		return "?"
	}
	return strconv.Itoa(count)
}

func navigateTo(view View) tea.Cmd {
	return func() tea.Msg {
		return NavigateMsg{To: view}
	}
}
