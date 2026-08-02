package views

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/main/tui/routes"
	contracts "github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
)

// Dashboard is the main navigation hub of the TUI.
type Dashboard struct {
	app    *app.Session
	styles styles.DashboardStyles
	keys   dashboardKeys
	width  int
	height int

	loading bool
	spinner contracts.Spinner
	data    *app.Dashboard
	err     error
}

const (
	dashboardEdgeMargin = 2
	dashboardRecentMax  = app.DashboardRecentLimit
)

type DashboardLoadedMsg struct {
	Data *app.Dashboard
	Err  error
}

// NewDashboard creates a new Dashboard view.
func NewDashboard(app *app.Session) *Dashboard {
	d := &Dashboard{
		app:     app,
		styles:  styles.Standard.Dashboard,
		keys:    newDashboardKeys(),
		loading: true,
	}
	d.spinner = contracts.NewSpinner("Loading dashboard...", d.styles.Subtitle)
	return d
}

// Init implements ViewModel.
func (d *Dashboard) Init() tea.Cmd {
	d.loading = true
	return tea.Batch(d.spinner.Init(), d.loadData())
}

func (d *Dashboard) Interaction() contracts.Interaction { return contracts.Interaction{} }

// Update implements ViewModel.
func (d *Dashboard) Update(msg tea.Msg) (contracts.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.Nav1):
			return d, navigateTo(routes.ViewDrinks)
		case key.Matches(msg, d.keys.Nav2):
			return d, navigateTo(routes.ViewIngredients)
		case key.Matches(msg, d.keys.Nav3):
			return d, navigateTo(routes.ViewInventory)
		case key.Matches(msg, d.keys.Nav4):
			return d, navigateTo(routes.ViewMenus)
		case key.Matches(msg, d.keys.Nav5):
			return d, navigateTo(routes.ViewOrders)
		case key.Matches(msg, d.keys.Nav6):
			return d, navigateTo(routes.ViewAudit)
		case key.Matches(msg, d.keys.Nav7):
			return d, navigateTo(routes.ViewTags)
		case key.Matches(msg, d.keys.Refresh):
			d.loading = true
			d.err = nil
			return d, tea.Batch(d.spinner.Init(), d.loadData())
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

	fixed := d.fixedContent(header, subtitle, content)
	if d.height > 0 && lipgloss.Height(fixed) > d.height {
		content = d.renderCompactCards(cards, cardWidth, columnCount)
		fixed = d.fixedContent(header, subtitle, content)
	}
	activity := d.renderRecentActivity(d.recentActivityLimit(fixed))
	body := lipgloss.JoinVertical(lipgloss.Left, fixed, activity)
	if d.width > 0 && d.height > 0 {
		return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, body)
	}

	return body
}

func (d *Dashboard) fixedContent(header, subtitle, cards string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		subtitle,
		"",
		cards,
		"",
		d.styles.Subtitle.Render("Recent Activity"),
	)
}

// ShortHelp implements ViewModel.
func (d *Dashboard) ShortHelp() []key.Binding {
	return []key.Binding{
		d.keys.Nav1, d.keys.Nav2, d.keys.Nav3,
		d.keys.Nav4, d.keys.Nav5, d.keys.Nav6, d.keys.Nav7,
		d.keys.Refresh,
	}
}

// FullHelp implements ViewModel.
func (d *Dashboard) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{d.keys.Nav1, d.keys.Nav2, d.keys.Nav3},
		{d.keys.Nav4, d.keys.Nav5, d.keys.Nav6, d.keys.Nav7},
		{d.keys.Refresh, d.keys.Help, d.keys.Quit},
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
		aggregate, err := d.app.Dashboard()
		return DashboardLoadedMsg{Data: &aggregate, Err: err}
	}
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
		unknown := app.UnknownDashboard()
		data = &unknown
	}

	return []dashboardCard{
		{key: "1", title: "Drinks", desc: "Manage drink recipes", count: formatCount(data.DrinkCount)},
		{key: "2", title: "Ingredients", desc: "Catalog ingredients", count: formatCount(data.IngredientCount)},
		{key: "3", title: "Inventory", desc: d.inventorySubtitle(data), count: formatCount(data.InventoryCount)},
		{key: "4", title: "Menus", desc: d.menuSubtitle(data), count: formatCount(data.MenuCount)},
		{key: "5", title: "Orders", desc: d.ordersSubtitle(data), count: formatCount(data.OrderCount)},
		{key: "6", title: "Audit", desc: "Inspect audit logs", count: d.auditCountLabel(data)},
		{key: "7", title: "Tags", desc: "Tag any entity", count: ""},
	}
}

func (d *Dashboard) inventorySubtitle(data *app.Dashboard) string {
	if data.LowStockCount >= 0 {
		return "Low stock: " + formatCount(data.LowStockCount)
	}
	return "Track stock levels"
}

func (d *Dashboard) menuSubtitle(data *app.Dashboard) string {
	if data.DraftMenus >= 0 && data.PublishedMenus >= 0 {
		return fmt.Sprintf("Draft %s • Published %s", formatCount(data.DraftMenus), formatCount(data.PublishedMenus))
	}
	return "Build drink menus"
}

func (d *Dashboard) ordersSubtitle(data *app.Dashboard) string {
	if data.PendingOrders >= 0 {
		return "Pending: " + formatCount(data.PendingOrders)
	}
	return "Review orders"
}

func (d *Dashboard) auditCountLabel(data *app.Dashboard) string {
	if data.AuditCount < 0 {
		return "?"
	}
	return strconv.Itoa(data.AuditCount)
}

func (d *Dashboard) renderRecentActivity(limit int) string {
	if limit <= 0 {
		return ""
	}
	if d.data == nil || len(d.data.RecentActivity) == 0 {
		return d.styles.Subtitle.Render("No recent activity")
	}

	limit = min(limit, len(d.data.RecentActivity))
	rows := make([]string, 0, limit)
	for _, entry := range d.data.RecentActivity[:limit] {
		rows = append(rows, fmt.Sprintf("%s  %s  %s", entry.Timestamp.Format("15:04"), entry.Actor, entry.Action))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (d *Dashboard) recentActivityLimit(fixed string) int {
	if d.height <= 0 {
		return dashboardRecentMax
	}
	return min(max(d.height-lipgloss.Height(fixed), 0), dashboardRecentMax)
}

func (d *Dashboard) layoutConfig() (int, int) {
	if d.width <= 0 {
		return 34, 2
	}

	gap := 2
	minCardWidth := 28
	availableWidth := max(d.width-(dashboardEdgeMargin*2), 0)
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

func (d *Dashboard) renderCompactCards(cards []dashboardCard, width int, columns int) string {
	rows := make([]string, 0, len(cards))
	for _, card := range cards {
		title := card.title
		if card.count != "" {
			title = fmt.Sprintf("%s (%s)", card.title, card.count)
		}
		rows = append(rows, lipgloss.NewStyle().Width(width).Render("["+card.key+"] "+title))
	}
	if columns <= 1 {
		return lipgloss.JoinVertical(lipgloss.Left, rows...)
	}
	paired := make([]string, 0, (len(rows)+1)/2)
	gap := lipgloss.NewStyle().Width(2).Render("")
	for i := 0; i < len(rows); i += 2 {
		if i+1 == len(rows) {
			paired = append(paired, rows[i])
			break
		}
		paired = append(paired, lipgloss.JoinHorizontal(lipgloss.Top, rows[i], gap, rows[i+1]))
	}
	return lipgloss.JoinVertical(lipgloss.Left, paired...)
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

func formatCount(count int) string {
	if count < 0 {
		return "?"
	}
	return strconv.Itoa(count)
}

func navigateTo(view routes.View) tea.Cmd {
	return func() tea.Msg {
		return routes.NavigateMsg{To: view}
	}
}
