package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListViewModel renders the ingredients list and detail panes.
type ListViewModel struct {
	app    *app.App
	ctx    *middleware.Context
	styles tui.ListViewStyles
	keys   tui.ListViewKeys

	formStyles forms.FormStyles
	formKeys   forms.FormKeys

	queries *queries.Queries

	list    list.Model
	detail  *DetailViewModel
	create  *CreateIngredientVM
	spinner components.Spinner
	loading bool
	err     error

	width       int
	height      int
	listWidth   int
	detailWidth int
}

func NewListViewModel(app *app.App, ctx *middleware.Context, styles tui.ListViewStyles, keys tui.ListViewKeys, formStyles forms.FormStyles, formKeys forms.FormKeys) *ListViewModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = styles.Selected
	delegate.Styles.SelectedDesc = styles.Selected

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Ingredients"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)

	vm := &ListViewModel{
		app:        app,
		ctx:        ctx,
		styles:     styles,
		keys:       keys,
		formStyles: formStyles,
		formKeys:   formKeys,
		queries:    queries.New(),
		list:       l,
		detail:     NewDetailViewModel(styles),
		loading:    true,
	}
	vm.spinner = components.NewSpinner("Loading ingredients...", styles.Subtitle)
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Init(), m.loadIngredients())
}

func (m *ListViewModel) Update(msg tea.Msg) (views.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		if m.create != nil {
			m.create.SetWidth(m.detailWidth)
		}
		return m, nil
	case IngredientCreatedMsg:
		m.create = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadIngredients())
	case tea.KeyMsg:
		if m.create != nil {
			if key.Matches(msg, m.keys.Back) {
				m.create = nil
				return m, nil
			}
		} else {
			switch {
			case key.Matches(msg, m.keys.Refresh):
				m.loading = true
				m.err = nil
				return m, tea.Batch(m.spinner.Init(), m.loadIngredients())
			case msg.String() == "c":
				return m, m.startCreate()
			}
		}
	case IngredientsLoadedMsg:
		m.loading = false
		m.err = msg.Err
		items := make([]list.Item, 0, len(msg.Ingredients))
		for _, ingredient := range msg.Ingredients {
			items = append(items, ingredientItem{ingredient: ingredient})
		}
		m.list.SetItems(items)
		m.syncDetail()
		return m, nil
	}

	if m.create != nil {
		var cmd tea.Cmd
		m.create, cmd = m.create.Update(msg)
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

	listView := m.list.View()
	if m.err != nil {
		listView = m.styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.err))
	}
	listView = m.styles.ListPane.Width(m.listWidth).Render(listView)

	detailView := m.detail.View()
	if m.create != nil {
		detailView = m.create.View()
	}
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

func (m *ListViewModel) loadIngredients() tea.Cmd {
	return func() tea.Msg {
		ingredientsList, err := m.queries.List(m.ctx, queries.ListFilter{})
		if err != nil {
			return IngredientsLoadedMsg{Err: err}
		}

		items := make([]models.Ingredient, 0, len(ingredientsList))
		for i, ingredient := range ingredientsList {
			if ingredient == nil {
				return IngredientsLoadedMsg{Err: errors.Internalf("ingredient %d missing", i)}
			}
			items = append(items, *ingredient)
		}

		return IngredientsLoadedMsg{Ingredients: items}
	}
}

func (m *ListViewModel) startCreate() tea.Cmd {
	m.create = NewCreateIngredientVM(CreateDeps{
		FormStyles: m.formStyles,
		FormKeys:   m.formKeys,
		Ctx:        m.ctx,
		CreateFunc: m.app.Ingredients.Create,
	})
	m.create.SetWidth(m.detailWidth)
	return m.create.Init()
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
	item, ok := m.list.SelectedItem().(ingredientItem)
	if !ok {
		m.detail.SetIngredient(optional.None[models.Ingredient]())
		return
	}
	m.detail.SetIngredient(optional.Some(item.ingredient))
}
