package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Dashboard is the main navigation hub of the TUI.
type Dashboard struct {
	styles DashboardStyles
	keys   DashboardKeys
	width  int
	height int
}

// DashboardStyles contains the lipgloss styles used by the dashboard.
type DashboardStyles struct {
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Card     lipgloss.Style
	HelpKey  lipgloss.Style
}

// DashboardKeys defines the key bindings used by the dashboard.
type DashboardKeys struct {
	Nav1 key.Binding
	Nav2 key.Binding
	Nav3 key.Binding
	Nav4 key.Binding
	Nav5 key.Binding
	Nav6 key.Binding
	Help key.Binding
	Quit key.Binding
}

// NewDashboard creates a new Dashboard view.
func NewDashboard(styles DashboardStyles, keys DashboardKeys) *Dashboard {
	return &Dashboard{
		styles: styles,
		keys:   keys,
	}
}

// Init implements ViewModel.
func (d *Dashboard) Init() tea.Cmd {
	return nil
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
	}
	return d, nil
}

// View implements ViewModel.
func (d *Dashboard) View() string {
	header := d.styles.Title.Render("Dashboard")
	subtitle := d.styles.Subtitle.Render("Select a workspace to continue")

	cards := []dashboardCard{
		{key: "1", title: "Drinks", desc: "Manage drink recipes"},
		{key: "2", title: "Ingredients", desc: "Catalog ingredients"},
		{key: "3", title: "Inventory", desc: "Track stock levels"},
		{key: "4", title: "Menus", desc: "Build drink menus"},
		{key: "5", title: "Orders", desc: "Review orders"},
		{key: "6", title: "Audit", desc: "Inspect audit logs"},
	}

	cardWidth, columnCount := d.layoutConfig()
	content := d.renderCards(cards, cardWidth, columnCount)

	body := lipgloss.JoinVertical(lipgloss.Left, header, subtitle, "", content)
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
}

func (d *Dashboard) layoutConfig() (int, int) {
	if d.width <= 0 {
		return 34, 2
	}

	gap := 2
	minCardWidth := 28
	available := d.width - gap
	if available >= minCardWidth*2 {
		return available / 2, 2
	}

	return d.width, 1
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
	title := d.styles.Title.Render(card.title)
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

func navigateTo(view View) tea.Cmd {
	return func() tea.Msg {
		return NavigateMsg{To: view}
	}
}
