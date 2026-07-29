package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventory "github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	lowStockThreshold  = inventory.DefaultLowStockThreshold
	inventoryColumnGap = 1
)

var (
	previousInventoryPage = key.NewBinding(key.WithKeys("["), key.WithHelp("[", "previous page"))
	nextInventoryPage     = key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next page"))
)

type listMode int

const (
	listModeBrowsing listMode = iota
	listModeAdjusting
	listModeSetting
	listModeTagging
	listModeFiltering
)

// ListViewModel renders the inventory list and detail panes.
type ListViewModel struct {
	app    *app.Session
	styles tui.ListViewStyles
	keys   tuikeys.ListViewKeys

	formStyles forms.FormStyles
	formKeys   forms.FormKeys

	rows        []InventoryRow
	table       table.Model
	detail      *DetailViewModel
	mode        listMode
	adjust      *AdjustInventoryVM
	set         *SetInventoryVM
	tags        *components.TagEditor
	filter      *filterVM
	request     inventory.ListRequest
	next        paging.Cursor
	history     []paging.Cursor
	loadToken   uint64
	spinner     tui.Spinner
	loading     bool
	err         error
	width       int
	height      int
	listWidth   int
	detailWidth int
}

func NewListViewModel(app *app.Session) *ListViewModel {
	columns := inventoryColumns(0)
	model := table.New(
		table.WithColumns(columns),
		table.WithRows(nil),
		table.WithFocused(true),
	)
	model.SetStyles(inventoryTableStyles(tuistyles.App.ListView))

	vm := &ListViewModel{
		app:        app,
		styles:     tuistyles.App.ListView,
		keys:       tuikeys.App.ListView,
		formStyles: tuistyles.App.Form,
		formKeys:   tuikeys.App.Form,
		table:      model,
		detail:     NewDetailViewModel(tuistyles.App.ListView),
		loading:    true,
	}
	vm.spinner = tui.NewSpinner("Loading inventory...", vm.styles.Subtitle)
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Init(), m.loadInventory())
}

func (m *ListViewModel) Interaction() views.Interaction {
	return views.Interaction{
		HandlesBack:  m.mode != listModeBrowsing,
		CapturesText: m.mode == listModeAdjusting || m.mode == listModeSetting || m.mode == listModeTagging || m.mode == listModeFiltering,
	}
}

func (m *ListViewModel) Update(msg tea.Msg) (views.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		switch m.mode {
		case listModeBrowsing:
		case listModeAdjusting:
			m.adjust.SetWidth(m.detailWidth)
		case listModeSetting:
			m.set.SetWidth(m.detailWidth)
		case listModeTagging:
			m.tags.SetWidth(m.width)
		case listModeFiltering:
			m.filter.form.SetWidth(m.detailWidth)
		}
		return m, nil
	case InventoryAdjustedMsg:
		m.mode = listModeBrowsing
		m.adjust = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadInventory())
	case InventorySetMsg:
		m.mode = listModeBrowsing
		m.set = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadInventory())
	case components.TagsSavedMsg:
		if m.mode != listModeTagging || m.tags == nil || !m.tags.Owns(msg.Target) {
			return m, nil
		}
		m.mode, m.tags, m.loading, m.err = listModeBrowsing, nil, true, nil
		return m, tea.Batch(m.spinner.Init(), m.loadInventory())
	case tea.KeyMsg:
		switch m.mode {
		case listModeBrowsing:
		case listModeAdjusting:
			if key.Matches(msg, m.keys.Back) {
				m.mode = listModeBrowsing
				m.adjust = nil
				return m, nil
			}
		case listModeSetting:
			if key.Matches(msg, m.keys.Back) {
				m.mode = listModeBrowsing
				m.set = nil
				return m, nil
			}
		case listModeTagging:
			if key.Matches(msg, m.keys.Back) {
				if m.tags.Saving() {
					return m, nil
				}
				m.mode, m.tags = listModeBrowsing, nil
				return m, nil
			}
		case listModeFiltering:
			if key.Matches(msg, m.keys.Back) {
				m.mode, m.filter = listModeBrowsing, nil
				return m, nil
			}
			if filterSubmit(msg) {
				req, err := m.filter.Request()
				if err != nil {
					return m, nil
				}
				m.request, m.history, m.next = req, nil, ""
				m.mode, m.filter, m.loading = listModeBrowsing, nil, true
				return m, tea.Batch(m.spinner.Init(), m.loadInventory())
			}
		}
		if m.mode != listModeBrowsing {
			break
		}
		switch {
		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Init(), m.loadInventory())
		case msg.String() == "f":
			m.mode, m.filter = listModeFiltering, newFilterVM(m.request)
			m.filter.form.SetWidth(m.detailWidth)
			return m, m.filter.Init()
		case msg.String() == "]" && m.next != "":
			m.history = append(m.history, m.request.Cursor)
			m.request.Cursor, m.loading = m.next, true
			return m, tea.Batch(m.spinner.Init(), m.loadInventory())
		case msg.String() == "[" && len(m.history) > 0:
			i := len(m.history) - 1
			m.request.Cursor, m.history, m.loading = m.history[i], m.history[:i], true
			return m, tea.Batch(m.spinner.Init(), m.loadInventory())
		case key.Matches(msg, m.keys.Adjust):
			return m, m.startAdjust()
		case key.Matches(msg, m.keys.Set):
			return m, m.startSet()
		case key.Matches(msg, m.keys.Tags):
			return m, m.startTags()
		}
	case InventoryLoadedMsg:
		if msg.Token != m.loadToken {
			return m, nil
		}
		m.loading = false
		m.err = msg.Err
		if msg.Err != nil {
			return m, nil
		}
		m.next = msg.Next
		selected := selectedInventoryID(m.selectedRow())
		m.rows = msg.Rows
		m.table.SetRows(buildInventoryTableRows(msg.Rows, m.styles))
		m.selectInventory(selected)
		m.syncDetail()
		return m, nil
	}

	switch m.mode {
	case listModeBrowsing:
	case listModeAdjusting:
		var cmd tea.Cmd
		m.adjust, cmd = m.adjust.Update(msg)
		return m, cmd
	case listModeSetting:
		var cmd tea.Cmd
		m.set, cmd = m.set.Update(msg)
		return m, cmd
	case listModeTagging:
		var cmd tea.Cmd
		m.tags, cmd = m.tags.Update(msg)
		return m, cmd
	case listModeFiltering:
		return m, m.filter.Update(msg)
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
	if m.mode == listModeFiltering {
		return m.filter.View()
	}
	if m.loading {
		return m.renderLoading()
	}
	if m.mode == listModeTagging {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.tags.View())
	}

	listView := m.table.View()
	if m.err != nil {
		listView = m.styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.err))
	}
	listView = m.styles.ListPane.Width(views.PaneStyleWidth(m.styles.ListPane, m.listWidth)).Render(listView)

	detailView := m.detail.View()
	switch m.mode {
	case listModeBrowsing:
	case listModeTagging:
	case listModeAdjusting:
		detailView = m.adjust.View()
	case listModeSetting:
		detailView = m.set.View()
	}
	detailView = m.styles.DetailPane.Width(views.PaneStyleWidth(m.styles.DetailPane, m.detailWidth)).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	switch m.mode {
	case listModeTagging:
		return []key.Binding{m.formKeys.Submit, m.keys.Back}
	case listModeAdjusting, listModeSetting:
		return []key.Binding{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit, m.keys.Back}
	case listModeBrowsing:
		return []key.Binding{m.keys.Up, m.keys.Down, previousInventoryPage, nextInventoryPage, m.keys.Adjust, m.keys.Set, m.keys.Tags, m.keys.Refresh, m.keys.Back}
	}
	return nil
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	switch m.mode {
	case listModeTagging:
		return [][]key.Binding{{m.formKeys.Submit, m.keys.Back}}
	case listModeAdjusting, listModeSetting:
		return [][]key.Binding{
			{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit},
			{m.keys.Back},
		}
	case listModeBrowsing:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down, m.keys.Enter},
			{previousInventoryPage, nextInventoryPage},
			{m.keys.Adjust, m.keys.Set, m.keys.Tags},
			{m.keys.Refresh, m.keys.Back},
		}
	}
	return nil
}

func (m *ListViewModel) loadInventory() tea.Cmd {
	m.loadToken++
	token := m.loadToken
	req := m.request
	return func() tea.Msg {
		inventoryList, err := m.app.Inventory.List(m.context(), req)
		if err != nil {
			return InventoryLoadedMsg{Err: err, Token: token}
		}

		ingredientIDs := make(map[entity.IngredientID]struct{}, len(inventoryList.Items))
		for i, item := range inventoryList.Items {
			if item == nil {
				return InventoryLoadedMsg{Err: errors.Internalf("inventory %d missing", i), Token: token}
			}
			if item.IngredientID.IsZero() {
				return InventoryLoadedMsg{Err: errors.Internalf("inventory %s missing ingredient", item.ID.String()), Token: token}
			}
			ingredientIDs[item.IngredientID] = struct{}{}
		}

		ids := make([]entity.IngredientID, 0, len(ingredientIDs))
		for id := range ingredientIDs {
			ids = append(ids, id)
		}

		ingredientByID, err := m.loadIngredients(ids)
		if err != nil {
			return InventoryLoadedMsg{Err: errors.Internalf("load ingredients: %w", err), Token: token}
		}

		rows := make([]InventoryRow, 0, len(inventoryList.Items))
		for _, item := range inventoryList.Items {
			ingredient, ok := ingredientByID[item.IngredientID]
			if !ok {
				return InventoryLoadedMsg{Err: errors.Internalf("ingredient %s missing", item.IngredientID.String()), Token: token}
			}

			quantity := item.Amount.String()
			cost := "N/A"
			if price, ok := item.CostPerUnit.Unwrap(); ok {
				cost = price.String()
			}
			threshold := inventory.DefaultLowStockThreshold
			if value, ok := req.LowStock.Unwrap(); ok {
				threshold = value
			}
			status := stockStatus(item.Amount, threshold)

			rows = append(rows, InventoryRow{
				Inventory:  *item,
				Ingredient: *ingredient,
				Quantity:   quantity,
				Cost:       cost,
				Status:     status,
			})
		}

		return InventoryLoadedMsg{Rows: rows, Next: inventoryList.Next, Token: token}
	}
}

func (m *ListViewModel) loadIngredients(ids []entity.IngredientID) (map[entity.IngredientID]*ingredientsmodels.Ingredient, error) {
	ingredientByID := make(map[entity.IngredientID]*ingredientsmodels.Ingredient, len(ids))
	for _, id := range ids {
		ingredient, err := m.app.Ingredients.Get(m.context(), id)
		if err != nil {
			return nil, err
		}
		if ingredient == nil {
			continue
		}
		ingredientByID[ingredient.ID] = ingredient
	}
	return ingredientByID, nil
}

func (m *ListViewModel) startAdjust() tea.Cmd {
	row, ok := m.selectedRow()
	if !ok {
		return nil
	}
	m.mode = listModeAdjusting
	m.adjust = NewAdjustInventoryVM(m.app, row)
	m.adjust.SetWidth(m.detailWidth)
	return m.adjust.Init()
}

func (m *ListViewModel) startSet() tea.Cmd {
	row, ok := m.selectedRow()
	if !ok {
		return nil
	}
	m.mode = listModeSetting
	m.set = NewSetInventoryVM(m.app, row)
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

func selectedInventoryID(row InventoryRow, ok bool) entity.InventoryID {
	if !ok {
		return entity.InventoryID{}
	}
	return row.Inventory.ID
}

func (m *ListViewModel) selectInventory(id entity.InventoryID) {
	m.table.SetCursor(0)
	if id.IsZero() {
		return
	}
	for i := range m.rows {
		if m.rows[i].Inventory.ID == id {
			m.table.SetCursor(i)
			return
		}
	}
}

func (m *ListViewModel) startTags() tea.Cmd {
	row, ok := m.selectedRow()
	if !ok {
		return nil
	}
	m.mode = listModeTagging
	m.tags = components.NewTagEditor(m.app, row.Inventory.EntityUID(), row.Ingredient.Name, row.Inventory.Tags)
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

	listWidth, detailWidth := views.SplitListDetailWidths(width)
	innerListWidth := views.PaneContentWidth(m.styles.ListPane, listWidth)
	detailWidth = views.PaneContentWidth(m.styles.DetailPane, detailWidth)
	m.table.SetColumns(inventoryColumns(innerListWidth))
	m.table.SetWidth(innerListWidth)
	tableHeight := height
	if tableHeight > 0 {
		tableHeight--
	}
	m.table.SetHeight(tableHeight)
	m.listWidth = innerListWidth
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

func (m *ListViewModel) context() *middleware.Context {
	return m.app.Context()
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

	contentWidth := max(width-(inventoryColumnGap*columnCount), 0)

	nameWidth := max(contentWidth-(categoryWidth+quantityWidth+costWidth+statusWidth), 0)

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

func stockStatus(amount measurement.Amount, thresholds ...float64) string {
	threshold := lowStockThreshold
	if len(thresholds) > 0 {
		threshold = thresholds[0]
	}
	value := amount.Value()
	if value <= 0 {
		return "OUT"
	}
	if value <= threshold {
		return "LOW"
	}
	return "OK"
}
