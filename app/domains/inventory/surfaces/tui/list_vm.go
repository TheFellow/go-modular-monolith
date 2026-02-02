package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientsqueries "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	inventoryqueries "github.com/TheFellow/go-modular-monolith/app/domains/inventory/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	lowStockThreshold  = 10.0
	inventoryColumnGap = 1
)

// ListViewModel renders the inventory list and detail panes.
type ListViewModel struct {
	app    *app.App
	ctx    *middleware.Context
	styles tui.ListViewStyles
	keys   tui.ListViewKeys

	formStyles forms.FormStyles
	formKeys   forms.FormKeys

	inventoryQueries  *inventoryqueries.Queries
	ingredientQueries *ingredientsqueries.Queries
	rows              []InventoryRow
	table             table.Model
	detail            *DetailViewModel
	adjust            *AdjustInventoryVM
	set               *SetInventoryVM
	spinner           components.Spinner
	loading           bool
	err               error
	width             int
	height            int
	listWidth         int
	detailWidth       int
}

func NewListViewModel(app *app.App, ctx *middleware.Context, styles tui.ListViewStyles, keys tui.ListViewKeys, formStyles forms.FormStyles, formKeys forms.FormKeys) *ListViewModel {
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
		formStyles:        formStyles,
		formKeys:          formKeys,
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
		if m.adjust != nil {
			m.adjust.SetWidth(m.detailWidth)
		}
		if m.set != nil {
			m.set.SetWidth(m.detailWidth)
		}
		return m, nil
	case InventoryAdjustedMsg:
		m.adjust = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadInventory())
	case InventorySetMsg:
		m.set = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadInventory())
	case tea.KeyMsg:
		if m.adjust != nil {
			if key.Matches(msg, m.keys.Back) {
				m.adjust = nil
				return m, nil
			}
			break
		}
		if m.set != nil {
			if key.Matches(msg, m.keys.Back) {
				m.set = nil
				return m, nil
			}
			break
		}
		switch {
		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Init(), m.loadInventory())
		case key.Matches(msg, m.keys.Adjust):
			return m, m.startAdjust()
		case key.Matches(msg, m.keys.Set):
			return m, m.startSet()
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

	if m.adjust != nil {
		var cmd tea.Cmd
		m.adjust, cmd = m.adjust.Update(msg)
		return m, cmd
	}

	if m.set != nil {
		var cmd tea.Cmd
		m.set, cmd = m.set.Update(msg)
		return m, cmd
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
	if m.adjust != nil {
		detailView = m.adjust.View()
	} else if m.set != nil {
		detailView = m.set.View()
	}
	detailView = m.styles.DetailPane.Width(m.detailWidth).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	if m.adjust != nil || m.set != nil {
		return []key.Binding{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit, m.keys.Back}
	}
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Adjust, m.keys.Set, m.keys.Refresh, m.keys.Back}
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	if m.adjust != nil || m.set != nil {
		return [][]key.Binding{
			{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit},
			{m.keys.Back},
		}
	}
	return [][]key.Binding{
		{m.keys.Up, m.keys.Down, m.keys.Enter},
		{m.keys.Adjust, m.keys.Set},
		{m.keys.Refresh, m.keys.Back},
	}
}

func (m *ListViewModel) loadInventory() tea.Cmd {
	return func() tea.Msg {
		inventoryList, err := m.inventoryQueries.List(m.ctx, inventoryqueries.ListFilter{})
		if err != nil {
			return InventoryLoadedMsg{Err: err}
		}

		ingredientIDs := make(map[entity.IngredientID]struct{}, len(inventoryList))
		for _, item := range inventoryList {
			if item.IngredientID.IsZero() {
				return InventoryLoadedMsg{Err: errors.Internalf("inventory %s missing ingredient", item.ID.String())}
			}
			ingredientIDs[item.IngredientID] = struct{}{}
		}

		ids := make([]entity.IngredientID, 0, len(ingredientIDs))
		for id := range ingredientIDs {
			ids = append(ids, id)
		}

		ingredientList, err := m.ingredientQueries.List(m.ctx, ingredientsqueries.ListFilter{IDs: ids})
		if err != nil {
			return InventoryLoadedMsg{Err: errors.Internalf("load ingredients: %w", err)}
		}

		ingredientByID := make(map[entity.IngredientID]*ingredientsmodels.Ingredient, len(ingredientList))
		for _, ingredient := range ingredientList {
			if ingredient == nil {
				continue
			}
			ingredientByID[ingredient.ID] = ingredient
		}

		rows := make([]InventoryRow, 0, len(inventoryList))
		for _, item := range inventoryList {
			ingredient, ok := ingredientByID[item.IngredientID]
			if !ok {
				return InventoryLoadedMsg{Err: errors.Internalf("ingredient %s missing", item.IngredientID.String())}
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

func (m *ListViewModel) startAdjust() tea.Cmd {
	row, ok := m.selectedRow()
	if !ok {
		return nil
	}
	m.adjust = NewAdjustInventoryVM(row, AdjustDeps{
		FormStyles: m.formStyles,
		FormKeys:   m.formKeys,
		Ctx:        m.ctx,
		AdjustFunc: m.app.Inventory.Adjust,
	})
	m.adjust.SetWidth(m.detailWidth)
	return m.adjust.Init()
}

func (m *ListViewModel) startSet() tea.Cmd {
	row, ok := m.selectedRow()
	if !ok {
		return nil
	}
	m.set = NewSetInventoryVM(row, SetDeps{
		FormStyles: m.formStyles,
		FormKeys:   m.formKeys,
		Ctx:        m.ctx,
		SetFunc:    m.app.Inventory.Set,
	})
	m.set.SetWidth(m.detailWidth)
	return m.set.Init()
}

func (m *ListViewModel) selectedRow() (InventoryRow, bool) {
	if len(m.rows) == 0 {
		return InventoryRow{}, false
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.rows) {
		return InventoryRow{}, false
	}
	return m.rows[idx], true
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

func inventoryTableStyles(styles tui.ListViewStyles) table.Styles {
	tableStyles := table.DefaultStyles()
	tableStyles.Header = styles.Subtitle.Bold(true).PaddingRight(inventoryColumnGap)
	tableStyles.Cell = lipgloss.NewStyle().PaddingRight(inventoryColumnGap)
	tableStyles.Selected = styles.Selected
	return tableStyles
}

func buildInventoryTableRows(rows []InventoryRow, styles tui.ListViewStyles) []table.Row {
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

func renderStatus(status string, styles tui.ListViewStyles) string {
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
