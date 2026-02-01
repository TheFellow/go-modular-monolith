package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	ingredientsqueries "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	inventorydao "github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	inventoryqueries "github.com/TheFellow/go-modular-monolith/app/domains/inventory/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	lowStockThreshold  = 10.0
	inventoryColumnGap = 1
)

// ListViewStyles contains styles needed by the inventory list view.
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

// ListViewKeys contains key bindings needed by the inventory list view.
type ListViewKeys struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Refresh key.Binding
	Back    key.Binding
}

// ListViewModel renders the inventory list and detail panes.
type ListViewModel struct {
	app    *app.App
	ctx    *middleware.Context
	styles ListViewStyles
	keys   ListViewKeys

	inventoryQueries  *inventoryqueries.Queries
	ingredientQueries *ingredientsqueries.Queries
	rows              []InventoryRow
	table             table.Model
	detail            *DetailViewModel
	spinner           components.Spinner
	loading           bool
	err               error
	width             int
	height            int
	listWidth         int
	detailWidth       int
}

func NewListViewModel(app *app.App, ctx *middleware.Context, styles ListViewStyles, keys ListViewKeys) *ListViewModel {
	columns := inventoryColumns(0)
	model := table.New(
		table.WithColumns(columns),
		table.WithRows(nil),
		table.WithFocused(true),
	)
	model.SetStyles(inventoryTableStyles(styles))

	vm := &ListViewModel{
		app:               app,
		ctx:               ctx,
		styles:            styles,
		keys:              keys,
		inventoryQueries:  inventoryqueries.New(),
		ingredientQueries: ingredientsqueries.New(),
		table:             model,
		detail:            NewDetailViewModel(styles),
		loading:           true,
	}
	vm.spinner = components.NewSpinner("Loading inventory...", styles.Subtitle)
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Init(), m.loadInventory())
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
			return m, tea.Batch(m.spinner.Init(), m.loadInventory())
		}
	case InventoryLoadedMsg:
		m.loading = false
		m.err = msg.Err
		m.rows = msg.Rows
		m.table.SetRows(buildInventoryTableRows(msg.Rows, m.styles))
		m.table.SetCursor(0)
		m.syncDetail()
		return m, nil
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	m.syncDetail()
	return m, cmd
}

func (m *ListViewModel) View() string {
	if m.loading {
		return m.renderLoading()
	}

	listView := m.table.View()
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

func (m *ListViewModel) loadInventory() tea.Cmd {
	return func() tea.Msg {
		inventoryList, err := m.inventoryQueries.List(m.ctx, inventorydao.ListFilter{})
		if err != nil {
			return InventoryLoadedMsg{Err: err}
		}

		rows := make([]InventoryRow, 0, len(inventoryList))
		for _, item := range inventoryList {
			if item.IngredientID.IsZero() {
				return InventoryLoadedMsg{Err: errors.Internalf("inventory %s missing ingredient", item.ID.String())}
			}

			ingredient, err := m.ingredientQueries.Get(m.ctx, item.IngredientID)
			if err != nil {
				return InventoryLoadedMsg{Err: errors.Internalf("load ingredient %s: %w", item.IngredientID.String(), err)}
			}

			quantity := item.Amount.String()
			cost := "N/A"
			if price, ok := item.CostPerUnit.Unwrap(); ok {
				cost = price.String()
			}
			status := stockStatus(item.Amount)

			rows = append(rows, InventoryRow{
				Inventory:  *item,
				Ingredient: *ingredient,
				Quantity:   quantity,
				Cost:       cost,
				Status:     status,
			})
		}

		return InventoryLoadedMsg{Rows: rows}
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
	listPadLeft, listPadRight := m.styles.ListPane.GetPaddingLeft(), m.styles.ListPane.GetPaddingRight()
	innerListWidth := listWidth - listPadLeft - listPadRight
	if innerListWidth < 0 {
		innerListWidth = 0
	}
	m.table.SetColumns(inventoryColumns(innerListWidth))
	m.table.SetWidth(innerListWidth)
	tableHeight := height
	if tableHeight > 0 {
		tableHeight--
	}
	m.table.SetHeight(tableHeight)
	m.listWidth = listWidth
	m.detailWidth = detailWidth
	m.detail.SetSize(detailWidth, height)
}

func (m *ListViewModel) syncDetail() {
	if len(m.rows) == 0 {
		m.detail.SetRow(optional.None[InventoryRow]())
		return
	}

	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.rows) {
		m.detail.SetRow(optional.None[InventoryRow]())
		return
	}

	m.detail.SetRow(optional.Some(m.rows[idx]))
}

func inventoryColumns(width int) []table.Column {
	const (
		categoryWidth = 8
		quantityWidth = 10
		costWidth     = 8
		statusWidth   = 6
		defaultWidth  = 48
		columnCount   = 5
	)

	if width <= 0 {
		width = defaultWidth
	}

	contentWidth := width - (inventoryColumnGap * columnCount)
	if contentWidth < 0 {
		contentWidth = 0
	}

	nameWidth := contentWidth - (categoryWidth + quantityWidth + costWidth + statusWidth)
	if nameWidth < 0 {
		nameWidth = 0
	}

	return []table.Column{
		{Title: "Ingredient", Width: nameWidth},
		{Title: "Category", Width: categoryWidth},
		{Title: "Quantity", Width: quantityWidth},
		{Title: "Cost", Width: costWidth},
		{Title: "Status", Width: statusWidth},
	}
}

func inventoryTableStyles(styles ListViewStyles) table.Styles {
	tableStyles := table.DefaultStyles()
	tableStyles.Header = styles.Subtitle.Bold(true).PaddingRight(inventoryColumnGap)
	tableStyles.Cell = lipgloss.NewStyle().PaddingRight(inventoryColumnGap)
	tableStyles.Selected = styles.Selected
	return tableStyles
}

func buildInventoryTableRows(rows []InventoryRow, styles ListViewStyles) []table.Row {
	out := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		status := renderStatus(row.Status, styles)
		out = append(out, table.Row{
			row.Ingredient.Name,
			string(row.Ingredient.Category),
			row.Quantity,
			row.Cost,
			status,
		})
	}
	return out
}

func renderStatus(status string, styles ListViewStyles) string {
	switch status {
	case "OUT":
		return styles.ErrorText.Render(status)
	case "LOW":
		return styles.WarningText.Render(status)
	default:
		return status
	}
}

func formatCost(cost optional.Value[money.Price]) (string, error) {
	price, ok := cost.Unwrap()
	if !ok {
		return "N/A", nil
	}
	if err := price.Validate(); err != nil {
		return "", err
	}
	return price.String(), nil
}

func stockStatus(amount measurement.Amount) string {
	value := amount.Value()
	if value <= 0 {
		return "OUT"
	}
	if value < lowStockThreshold {
		return "LOW"
	}
	return "OK"
}
