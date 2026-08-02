package tui

import (
	"fmt"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"slices"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	drinks "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredients "github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	tuikeys "github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type listMode int

const (
	listModeBrowsing listMode = iota
	listModeCreating
	listModeEditing
	listModeTagging
	listModeConfirmingDelete
	listModeFiltering
)

// ListViewModel renders the ingredients list and detail panes.
type ListViewModel struct {
	app    *app.Session
	styles tui.ListViewStyles
	keys   listViewKeys

	formStyles   forms.FormStyles
	formKeys     forms.FormKeys
	dialogStyles dialog.DialogStyles
	dialogKeys   dialog.DialogKeys

	shell     *tui.ListDetail
	detail    *DetailViewModel
	mode      listMode
	create    *CreateIngredientVM
	edit      *EditIngredientVM
	tags      *components.TagEditor[cedar.EntityUID, tag.Tags]
	dialog    *dialog.ConfirmDialog
	filter    *filterVM
	request   ingredients.ListRequest
	next      paging.Cursor
	history   []paging.Cursor
	loadToken uint64
	items     []list.Item

	deleteTarget *models.Ingredient

	width       int
	height      int
	detailWidth int
}

func NewListViewModel(app *app.Session) *ListViewModel {
	vm := &ListViewModel{
		app:          app,
		styles:       tuistyles.Standard.ListView,
		keys:         newListViewKeys(),
		formStyles:   tuistyles.Standard.Form,
		formKeys:     tuikeys.Standard.Form,
		dialogStyles: tuistyles.Standard.Dialog,
		dialogKeys:   tuikeys.Standard.Dialog,
		shell:        tui.NewListDetail("Ingredients", "Loading ingredients...", tuistyles.Standard.ListView),
		detail:       NewDetailViewModel(tuistyles.Standard.ListView),
	}
	vm.shell.SetLocalFiltering(false)
	vm.shell.SetLocalPagination(false)
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	return tea.Batch(m.shell.BeginLoading(), m.loadIngredients(""))
}

func (m *ListViewModel) Interaction() tui.Interaction {
	return tui.Interaction{
		HandlesBack:  m.mode != listModeBrowsing,
		CapturesText: m.mode == listModeFiltering || m.mode == listModeCreating || m.mode == listModeEditing || m.mode == listModeTagging,
	}
}

func (m *ListViewModel) Update(msg tea.Msg) (tui.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		switch m.mode {
		case listModeBrowsing:
		case listModeCreating:
			m.create.SetWidth(m.detailWidth)
		case listModeEditing:
			m.edit.SetWidth(m.detailWidth)
		case listModeTagging:
			m.tags.SetWidth(m.width)
		case listModeConfirmingDelete:
			m.dialog.SetWidth(m.width)
		case listModeFiltering:
			m.filter.form.SetWidth(m.detailWidth)
		}
		return m, nil
	case IngredientCreatedMsg:
		m.mode = listModeBrowsing
		m.create = nil
		return m, tea.Batch(m.shell.BeginLoading(), m.loadIngredients(m.request.Cursor))
	case IngredientUpdatedMsg:
		m.mode = listModeBrowsing
		m.edit = nil
		return m, tea.Batch(m.shell.BeginLoading(), m.loadIngredients(m.request.Cursor))
	case components.TagsSavedMsg[cedar.EntityUID, tag.Tags]:
		if m.mode != listModeTagging || m.tags == nil || !m.tags.Owns(msg.Target) {
			return m, nil
		}
		m.mode, m.tags = listModeBrowsing, nil
		return m, tea.Batch(m.shell.BeginLoading(), m.loadIngredients(m.request.Cursor))
	case IngredientDeletedMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.deleteTarget = nil
		return m, tea.Batch(m.shell.BeginLoading(), m.loadIngredients(m.request.Cursor))
	case DeleteErrorMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.deleteTarget = nil
		m.shell.SetError(msg.Err)
		return m, nil
	case showDeleteDialogMsg:
		m.mode = listModeConfirmingDelete
		m.dialog = msg.dialog
		m.deleteTarget = &msg.target
		m.dialog.SetWidth(m.width)
		return m, nil
	case dialog.ConfirmMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		return m, m.performDelete()
	case dialog.CancelMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.deleteTarget = nil
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case listModeBrowsing:
		case listModeConfirmingDelete:
		case listModeCreating:
			if key.Matches(msg, m.keys.Back) && !m.create.form.IsEditing() {
				m.mode = listModeBrowsing
				m.create = nil
				return m, nil
			}
		case listModeEditing:
			if key.Matches(msg, m.keys.Back) && !m.edit.form.IsEditing() {
				m.mode = listModeBrowsing
				m.edit = nil
				return m, nil
			}
		case listModeTagging:
			if key.Matches(msg, m.keys.Back) && !m.tags.FormEditing() {
				if m.tags.Saving() {
					return m, nil
				}
				m.mode, m.tags = listModeBrowsing, nil
				return m, nil
			}
		case listModeFiltering:
			if key.Matches(msg, m.keys.Back) && !m.filter.form.IsEditing() {
				m.mode, m.filter = listModeBrowsing, nil
				return m, nil
			}
			if filterSubmit(msg) {
				req, err := m.filter.Request()
				if err != nil {
					return m, nil
				}
				m.request, m.history, m.next = req, nil, ""
				m.mode, m.filter = listModeBrowsing, nil
				return m, tea.Batch(m.shell.BeginLoading(), m.loadIngredients(""))
			}
		}
		if m.mode != listModeBrowsing {
			break
		}
		switch {
		case key.Matches(msg, m.keys.Refresh):
			return m, tea.Batch(m.shell.BeginLoading(), m.loadIngredients(m.request.Cursor))
		case msg.String() == "f":
			m.mode, m.filter = listModeFiltering, newFilterVM(m.request)
			m.filter.form.SetWidth(m.detailWidth)
			return m, m.filter.Init()
		case msg.String() == "]" && m.next != "":
			m.history = append(m.history, m.request.Cursor)
			m.request.Cursor = m.next
			return m, tea.Batch(m.shell.BeginLoading(), m.loadIngredients(m.request.Cursor))
		case msg.String() == "[" && len(m.history) > 0:
			i := len(m.history) - 1
			m.request.Cursor = m.history[i]
			m.history = m.history[:i]
			return m, tea.Batch(m.shell.BeginLoading(), m.loadIngredients(m.request.Cursor))
		case key.Matches(msg, m.keys.Create):
			return m, m.startCreate()
		case key.Matches(msg, m.keys.Edit), key.Matches(msg, m.keys.Enter):
			return m, m.startEdit()
		case key.Matches(msg, m.keys.Delete):
			return m, m.startDelete()
		case key.Matches(msg, m.keys.Tags):
			return m, m.startTags()
		}
	case IngredientsLoadedMsg:
		if msg.Token != m.loadToken {
			return m, nil
		}
		if msg.Err != nil {
			m.shell.SetResult(m.items, msg.Err)
			return m, nil
		}
		m.next = msg.Next
		selected := selectedIngredientID(m.selectedIngredient())
		items := make([]list.Item, 0, len(msg.Ingredients))
		for _, ingredient := range msg.Ingredients {
			items = append(items, newIngredientItem(ingredient))
		}
		m.items = items
		m.shell.SetResult(items, msg.Err)
		m.selectIngredient(selected)
		m.updateTitle()
		m.syncDetail()
		return m, nil
	}

	switch m.mode {
	case listModeBrowsing:
	case listModeConfirmingDelete:
		var cmd tea.Cmd
		m.dialog, cmd = m.dialog.Update(msg)
		return m, cmd
	case listModeEditing:
		var cmd tea.Cmd
		m.edit, cmd = m.edit.Update(msg)
		return m, cmd
	case listModeTagging:
		var cmd tea.Cmd
		m.tags, cmd = m.tags.Update(msg)
		return m, cmd
	case listModeCreating:
		var cmd tea.Cmd
		m.create, cmd = m.create.Update(msg)
		return m, cmd
	case listModeFiltering:
		return m, m.filter.Update(msg)
	}

	cmd := m.shell.Update(msg)
	m.syncDetail()
	return m, cmd
}

func (m *ListViewModel) View() string {
	if m.mode == listModeFiltering {
		return m.filter.View()
	}
	if m.mode == listModeConfirmingDelete {
		dialogView := m.dialog.View()
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialogView)
		}
		return dialogView
	}
	if m.mode == listModeTagging {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.tags.View())
	}

	detailView := m.detail.View()
	switch m.mode {
	case listModeBrowsing, listModeConfirmingDelete:
	case listModeTagging:
	case listModeCreating:
		detailView = m.create.View()
	case listModeEditing:
		detailView = m.edit.View()
	case listModeFiltering:
	}
	return m.shell.View(detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	switch m.mode {
	case listModeConfirmingDelete:
		return []key.Binding{m.dialogKeys.Confirm, m.keys.Back, m.dialogKeys.Switch}
	case listModeTagging:
		return []key.Binding{m.formKeys.Submit, m.keys.Back}
	case listModeCreating, listModeEditing:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Edit, m.keys.Enter, m.formKeys.Submit, m.keys.Back}
	case listModeBrowsing:
		return []key.Binding{
			m.keys.Up, m.keys.Down,
			m.shell.KeyMap().PrevPage, m.shell.KeyMap().NextPage,
			m.keys.Create, m.keys.Edit, m.keys.Delete, m.keys.Tags,
			m.keys.Refresh, m.keys.Back,
		}
	case listModeFiltering:
	}
	return nil
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	switch m.mode {
	case listModeConfirmingDelete:
		return [][]key.Binding{
			{m.dialogKeys.Confirm, m.keys.Back},
			{m.dialogKeys.Switch},
		}
	case listModeTagging:
		return [][]key.Binding{{m.formKeys.Submit, m.keys.Back}}
	case listModeCreating, listModeEditing:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down, m.keys.Edit, m.keys.Enter, m.formKeys.Submit},
			{m.keys.Back},
		}
	case listModeBrowsing:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down, m.keys.Enter},
			{m.shell.KeyMap().PrevPage, m.shell.KeyMap().NextPage},
			{m.keys.Create, m.keys.Edit, m.keys.Delete, m.keys.Tags},
			{m.keys.Refresh, m.keys.Back},
		}
	case listModeFiltering:
	}
	return nil
}

func (m *ListViewModel) loadIngredients(cursor paging.Cursor) tea.Cmd {
	m.loadToken++
	token := m.loadToken
	req := m.request
	req.Cursor = cursor
	return func() tea.Msg {
		ingredientsList, err := m.app.Ingredients.List(m.context(), req)
		if err != nil {
			return IngredientsLoadedMsg{Err: err, Token: token}
		}

		items := make([]models.Ingredient, 0, len(ingredientsList.Items))
		for i, ingredient := range ingredientsList.Items {
			if ingredient == nil {
				return IngredientsLoadedMsg{Err: errors.Internalf("ingredient %d missing", i), Token: token}
			}
			items = append(items, *ingredient)
		}

		return IngredientsLoadedMsg{Ingredients: items, Next: ingredientsList.Next, Token: token}
	}
}

func (m *ListViewModel) startCreate() tea.Cmd {
	m.mode = listModeCreating
	m.create = NewCreateIngredientVM(m.app)
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
	m.mode = listModeEditing
	m.edit = NewEditIngredientVM(m.app, ingredient)
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
		drinks, err := paging.Collect(func(cursor paging.Cursor) (paging.Page[*drinksmodels.Drink], error) {
			return m.app.Drinks.List(m.context(), drinks.ListRequest{Cursor: cursor})
		})
		if err != nil {
			return DeleteErrorMsg{Err: err}
		}
		drinkCount := countDrinksUsingIngredient(drinks, ingredient.ID)
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
	target := *m.deleteTarget
	return func() tea.Msg {
		deleted, err := m.app.Ingredients.Delete(m.context(), target.ID)
		if err != nil {
			return DeleteErrorMsg{Err: err}
		}
		return IngredientDeletedMsg{Ingredient: deleted}
	}
}

func (m *ListViewModel) context() *middleware.Context {
	return m.app.Context()
}

func (m *ListViewModel) selectedIngredient() *models.Ingredient {
	item, ok := m.shell.SelectedItem().(ingredientItem)
	if !ok {
		return nil
	}
	ingredient := item.Value
	return &ingredient
}

func selectedIngredientID(value *models.Ingredient) entity.IngredientID {
	if value == nil {
		return entity.IngredientID{}
	}
	return value.ID
}

func (m *ListViewModel) selectIngredient(id entity.IngredientID) {
	if id.IsZero() {
		return
	}
	for i, item := range m.shell.Items() {
		if value, ok := item.(ingredientItem); ok && value.Value.ID == id {
			m.shell.Select(i)
			return
		}
	}
}

func (m *ListViewModel) updateTitle() {
	parts := []string{"Ingredients"}
	if m.request.Category != "" {
		parts = append(parts, "category="+string(m.request.Category))
	}
	if m.request.Filter != "" {
		parts = append(parts, "filter="+m.request.Filter)
	}
	parts = append(parts, fmt.Sprintf("page size=%d", effectiveLimit(m.request.Limit)))
	if len(m.history) > 0 {
		parts = append(parts, fmt.Sprintf("page=%d", len(m.history)+1))
	}
	m.shell.SetTitle(strings.Join(parts, " • "))
}

func effectiveLimit(limit int) int {
	if limit <= 0 {
		return paging.DefaultLimit
	}
	return limit
}

func (m *ListViewModel) startTags() tea.Cmd {
	ingredient := m.selectedIngredient()
	if ingredient == nil {
		return nil
	}
	m.mode = listModeTagging
	m.tags = components.NewTagEditor(m.app.ReplaceTags, tag.ParseCollection, ingredient.EntityUID(), ingredient.Name, ingredient.Tags.Canonical().String())
	m.tags.SetWidth(m.width)
	return m.tags.Init()
}

func (m *ListViewModel) setSize(width, height int) {
	m.width = width
	m.height = height

	if width <= 0 {
		return
	}

	_, detailWidth := m.shell.SetSize(width, height)
	m.detail.SetSize(detailWidth, height)
	m.detailWidth = detailWidth
}

func (m *ListViewModel) syncDetail() {
	item, ok := m.shell.SelectedItem().(ingredientItem)
	if !ok {
		m.detail.SetIngredient(optional.None[models.Ingredient]())
		return
	}
	m.detail.SetIngredient(optional.Some(item.Value))
}

func countDrinksUsingIngredient(drinks []*drinksmodels.Drink, ingredientID entity.IngredientID) int {
	count := 0
	for _, drink := range drinks {
		if drink == nil {
			continue
		}
		if drinkUsesIngredient(drink, ingredientID) {
			count++
		}
	}
	return count
}

func drinkUsesIngredient(drink *drinksmodels.Drink, ingredientID entity.IngredientID) bool {
	for _, recipeIngredient := range drink.Recipe.Ingredients {
		if recipeIngredient.IngredientID == ingredientID {
			return true
		}
		if slices.Contains(recipeIngredient.Substitutes, ingredientID) {
			return true
		}
	}
	return false
}
