package tui

import (
	"context"
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	drinksqueries "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListViewModel renders the ingredients list and detail panes.
type ListViewModel struct {
	app       *app.App
	principal cedar.EntityUID
	styles    tui.ListViewStyles
	keys      tui.ListViewKeys

	formStyles   forms.FormStyles
	formKeys     forms.FormKeys
	dialogStyles dialog.DialogStyles
	dialogKeys   dialog.DialogKeys

	queries      *queries.Queries
	drinkQueries *drinksqueries.Queries

	list    list.Model
	detail  *DetailViewModel
	create  *CreateIngredientVM
	edit    *EditIngredientVM
	dialog  *dialog.ConfirmDialog
	spinner components.Spinner
	loading bool
	err     error

	deleteTarget *models.Ingredient

	width       int
	height      int
	listWidth   int
	detailWidth int
}

func NewListViewModel(app *app.App, principal cedar.EntityUID) *ListViewModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = tuistyles.ListView.Selected
	delegate.Styles.SelectedDesc = tuistyles.ListView.Selected

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Ingredients"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)

	vm := &ListViewModel{
		app:          app,
		principal:    principal,
		styles:       tuistyles.ListView,
		keys:         tuikeys.ListView,
		formStyles:   tuistyles.Form,
		formKeys:     tuikeys.Form,
		dialogStyles: tuistyles.Dialog,
		dialogKeys:   tuikeys.Dialog,
		queries:      queries.New(),
		drinkQueries: drinksqueries.New(),
		list:         l,
		detail:       NewDetailViewModel(tuistyles.ListView),
		loading:      true,
	}
	vm.spinner = components.NewSpinner("Loading ingredients...", vm.styles.Subtitle)
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
		if m.edit != nil {
			m.edit.SetWidth(m.detailWidth)
		}
		if m.dialog != nil {
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case IngredientCreatedMsg:
		m.create = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadIngredients())
	case IngredientUpdatedMsg:
		m.edit = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadIngredients())
	case IngredientDeletedMsg:
		m.dialog = nil
		m.deleteTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadIngredients())
	case DeleteErrorMsg:
		m.dialog = nil
		m.deleteTarget = nil
		m.err = msg.Err
		return m, nil
	case showDeleteDialogMsg:
		m.dialog = msg.dialog
		m.deleteTarget = &msg.target
		if m.dialog != nil {
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case dialog.ConfirmMsg:
		m.dialog = nil
		return m, m.performDelete()
	case dialog.CancelMsg:
		m.dialog = nil
		m.deleteTarget = nil
		return m, nil
	case tea.KeyMsg:
		if m.dialog != nil {
			break
		}
		if m.create != nil {
			if key.Matches(msg, m.keys.Back) {
				m.create = nil
				return m, nil
			}
			break
		}
		if m.edit != nil {
			if key.Matches(msg, m.keys.Back) {
				m.edit = nil
				return m, nil
			}
			break
		}
		switch {
		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Init(), m.loadIngredients())
		case key.Matches(msg, m.keys.Create):
			return m, m.startCreate()
		case key.Matches(msg, m.keys.Edit), key.Matches(msg, m.keys.Enter):
			return m, m.startEdit()
		case key.Matches(msg, m.keys.Delete):
			return m, m.startDelete()
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

	if m.dialog != nil {
		var cmd tea.Cmd
		m.dialog, cmd = m.dialog.Update(msg)
		return m, cmd
	}

	if m.edit != nil {
		var cmd tea.Cmd
		m.edit, cmd = m.edit.Update(msg)
		return m, cmd
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
	if m.create != nil {
		detailView = m.create.View()
	} else if m.edit != nil {
		detailView = m.edit.View()
	}
	detailView = m.styles.DetailPane.Width(m.detailWidth).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	if m.dialog != nil {
		return []key.Binding{m.dialogKeys.Confirm, m.keys.Back, m.dialogKeys.Switch}
	}
	if m.create != nil || m.edit != nil {
		return []key.Binding{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit, m.keys.Back}
	}
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Create, m.keys.Edit, m.keys.Delete, m.keys.Refresh, m.keys.Back}
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	if m.dialog != nil {
		return [][]key.Binding{
			{m.dialogKeys.Confirm, m.keys.Back},
			{m.dialogKeys.Switch},
		}
	}
	if m.create != nil || m.edit != nil {
		return [][]key.Binding{
			{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit},
			{m.keys.Back},
		}
	}
	return [][]key.Binding{
		{m.keys.Up, m.keys.Down, m.keys.Enter},
		{m.keys.Create, m.keys.Edit, m.keys.Delete},
		{m.keys.Refresh, m.keys.Back},
	}
}

func (m *ListViewModel) loadIngredients() tea.Cmd {
	return func() tea.Msg {
		ingredientsList, err := m.queries.List(m.context(), queries.ListFilter{})
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
	m.create = NewCreateIngredientVM(m.app, m.principal)
	m.create.SetWidth(m.detailWidth)
	return m.create.Init()
}

type showDeleteDialogMsg struct {
	dialog *dialog.ConfirmDialog
	target models.Ingredient
}

func (m *ListViewModel) startEdit() tea.Cmd {
	ingredient := m.selectedIngredient()
	if ingredient == nil {
		return nil
	}
	m.edit = NewEditIngredientVM(m.app, m.principal, ingredient)
	m.edit.SetWidth(m.detailWidth)
	return m.edit.Init()
}

func (m *ListViewModel) startDelete() tea.Cmd {
	ingredient := m.selectedIngredient()
	if ingredient == nil {
		return nil
	}
	return m.showDeleteConfirm(ingredient)
}

func (m *ListViewModel) showDeleteConfirm(ingredient *models.Ingredient) tea.Cmd {
	if ingredient == nil {
		return nil
	}
	return func() tea.Msg {
		drinks, err := m.drinkQueries.ListByIngredient(m.context(), ingredient.ID)
		if err != nil {
			return DeleteErrorMsg{Err: err}
		}
		drinkCount := len(drinks)
		message := fmt.Sprintf("Delete %q?", ingredient.Name)
		if drinkCount > 0 {
			message = fmt.Sprintf(
				"Delete %q?\n\nThis will also delete %d drink(s) that use this ingredient.",
				ingredient.Name,
				drinkCount,
			)
		}
		confirm := dialog.NewConfirmDialog(
			"Delete Ingredient",
			message,
			dialog.WithDangerous(),
			dialog.WithFocusCancel(),
			dialog.WithConfirmText("Delete"),
			dialog.WithStyles(m.dialogStyles),
			dialog.WithKeys(m.dialogKeys),
		)
		return showDeleteDialogMsg{dialog: confirm, target: *ingredient}
	}
}

func (m *ListViewModel) performDelete() tea.Cmd {
	if m.deleteTarget == nil {
		return nil
	}
	target := m.deleteTarget
	return func() tea.Msg {
		deleted, err := m.app.Ingredients.Delete(m.context(), target.ID)
		if err != nil {
			return DeleteErrorMsg{Err: err}
		}
		return IngredientDeletedMsg{Ingredient: deleted}
	}
}

func (m *ListViewModel) context() *middleware.Context {
	return m.app.Context(context.Background(), m.principal)
}

func (m *ListViewModel) selectedIngredient() *models.Ingredient {
	item, ok := m.list.SelectedItem().(ingredientItem)
	if !ok {
		return nil
	}
	ingredient := item.ingredient
	return &ingredient
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
