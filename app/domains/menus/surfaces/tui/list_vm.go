package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
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
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListViewModel renders the menus list and detail panes.
type ListViewModel struct {
	app    *app.App
	styles tui.ListViewStyles
	keys   tui.ListViewKeys

	formStyles   forms.FormStyles
	formKeys     forms.FormKeys
	dialogStyles dialog.DialogStyles
	dialogKeys   dialog.DialogKeys

	queries *queries.Queries

	list    list.Model
	detail  *DetailViewModel
	create  *CreateMenuVM
	rename  *RenameMenuVM
	dialog  *dialog.ConfirmDialog
	spinner components.Spinner
	loading bool
	err     error

	deleteTarget  *menusmodels.Menu
	publishTarget *menusmodels.Menu
	draftTarget   *menusmodels.Menu

	width       int
	height      int
	listWidth   int
	detailWidth int
}

func NewListViewModel(app *app.App) *ListViewModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = tuistyles.App.ListView.Selected
	delegate.Styles.SelectedDesc = tuistyles.App.ListView.Selected

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Menus"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.Paginator.Type = paginator.Arabic
	l.SetFilteringEnabled(true)

	vm := &ListViewModel{
		app:          app,
		styles:       tuistyles.App.ListView,
		keys:         tuikeys.App.ListView,
		formStyles:   tuistyles.App.Form,
		formKeys:     tuikeys.App.Form,
		dialogStyles: tuistyles.App.Dialog,
		dialogKeys:   tuikeys.App.Dialog,
		queries:      queries.New(),
		list:         l,
		detail:       NewDetailViewModel(tuistyles.App.ListView, app),
		loading:      true,
	}
	vm.spinner = components.NewSpinner("Loading menus...", vm.styles.Subtitle)
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Init(), m.loadMenus())
}

func (m *ListViewModel) Update(msg tea.Msg) (views.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		if m.create != nil {
			m.create.SetWidth(m.detailWidth)
		}
		if m.rename != nil {
			m.rename.SetWidth(m.detailWidth)
		}
		if m.dialog != nil {
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case MenuCreatedMsg:
		m.create = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus())
	case MenuRenamedMsg:
		m.rename = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus())
	case MenuDeletedMsg:
		m.dialog = nil
		m.deleteTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus())
	case MenuPublishedMsg:
		m.dialog = nil
		m.publishTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus())
	case MenuDraftedMsg:
		m.dialog = nil
		m.draftTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus())
	case DeleteErrorMsg:
		m.dialog = nil
		m.deleteTarget = nil
		m.err = msg.Err
		return m, nil
	case PublishErrorMsg:
		m.dialog = nil
		m.publishTarget = nil
		m.err = msg.Err
		return m, nil
	case DraftErrorMsg:
		m.dialog = nil
		m.draftTarget = nil
		m.err = msg.Err
		return m, nil
	case showDeleteDialogMsg:
		m.dialog = msg.dialog
		m.deleteTarget = &msg.target
		m.publishTarget = nil
		m.draftTarget = nil
		if m.dialog != nil {
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case showPublishDialogMsg:
		m.dialog = msg.dialog
		m.publishTarget = &msg.target
		m.deleteTarget = nil
		m.draftTarget = nil
		if m.dialog != nil {
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case showDraftDialogMsg:
		m.dialog = msg.dialog
		m.draftTarget = &msg.target
		m.deleteTarget = nil
		m.publishTarget = nil
		if m.dialog != nil {
			m.dialog.SetWidth(m.width)
		}
		return m, nil
	case dialog.ConfirmMsg:
		m.dialog = nil
		if m.deleteTarget != nil {
			return m, m.performDelete()
		}
		if m.publishTarget != nil {
			return m, m.performPublish()
		}
		if m.draftTarget != nil {
			return m, m.performDraft()
		}
		return m, nil
	case dialog.CancelMsg:
		m.dialog = nil
		m.deleteTarget = nil
		m.publishTarget = nil
		m.draftTarget = nil
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
		if m.rename != nil {
			if key.Matches(msg, m.keys.Back) {
				m.rename = nil
				return m, nil
			}
			break
		}
		switch {
		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Init(), m.loadMenus())
		case key.Matches(msg, m.keys.Create):
			return m, m.startCreate()
		case key.Matches(msg, m.keys.Edit), key.Matches(msg, m.keys.Enter):
			return m, m.startRename()
		case key.Matches(msg, m.keys.Delete):
			return m, m.startDelete()
		case key.Matches(msg, m.keys.Publish):
			return m, m.startPublish()
		case key.Matches(msg, m.keys.Draft):
			return m, m.startDraft()
		}
	case MenusLoadedMsg:
		m.loading = false
		m.err = msg.Err
		items := make([]list.Item, 0, len(msg.Menus))
		for _, menu := range msg.Menus {
			items = append(items, newMenuItem(menu, m.styles))
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

	if m.rename != nil {
		var cmd tea.Cmd
		m.rename, cmd = m.rename.Update(msg)
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
	} else if m.rename != nil {
		detailView = m.rename.View()
	}
	detailView = m.styles.DetailPane.Width(m.detailWidth).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	if m.dialog != nil {
		return []key.Binding{m.dialogKeys.Confirm, m.keys.Back, m.dialogKeys.Switch}
	}
	if m.create != nil || m.rename != nil {
		return []key.Binding{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit, m.keys.Back}
	}
	return []key.Binding{
		m.keys.Up, m.keys.Down,
		m.list.KeyMap.PrevPage, m.list.KeyMap.NextPage,
		m.keys.Create, m.keys.Edit, m.keys.Delete, m.keys.Publish, m.keys.Draft,
		m.keys.Refresh, m.keys.Back,
	}
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	if m.dialog != nil {
		return [][]key.Binding{
			{m.dialogKeys.Confirm, m.keys.Back},
			{m.dialogKeys.Switch},
		}
	}
	if m.create != nil || m.rename != nil {
		return [][]key.Binding{
			{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit},
			{m.keys.Back},
		}
	}
	return [][]key.Binding{
		{m.keys.Up, m.keys.Down, m.keys.Enter},
		{m.list.KeyMap.PrevPage, m.list.KeyMap.NextPage},
		{m.keys.Create, m.keys.Edit, m.keys.Delete, m.keys.Publish, m.keys.Draft},
		{m.keys.Refresh, m.keys.Back},
	}
}

func (m *ListViewModel) loadMenus() tea.Cmd {
	return func() tea.Msg {
		menusList, err := m.queries.List(m.context(), queries.ListFilter{})
		if err != nil {
			return MenusLoadedMsg{Err: err}
		}

		menus := make([]menusmodels.Menu, 0, len(menusList))
		for i, menu := range menusList {
			if menu == nil {
				return MenusLoadedMsg{Err: errors.Internalf("menu %d missing", i)}
			}
			menus = append(menus, *menu)
		}

		return MenusLoadedMsg{Menus: menus}
	}
}

func (m *ListViewModel) startCreate() tea.Cmd {
	m.create = NewCreateMenuVM(m.app)
	m.create.SetWidth(m.detailWidth)
	return m.create.Init()
}

type showDeleteDialogMsg struct {
	dialog *dialog.ConfirmDialog
	target menusmodels.Menu
}

type showPublishDialogMsg struct {
	dialog *dialog.ConfirmDialog
	target menusmodels.Menu
}

type showDraftDialogMsg struct {
	dialog *dialog.ConfirmDialog
	target menusmodels.Menu
}

func (m *ListViewModel) startRename() tea.Cmd {
	menu := m.selectedMenu()
	if menu == nil {
		return nil
	}
	m.rename = NewRenameMenuVM(m.app, menu)
	m.rename.SetWidth(m.detailWidth)
	return m.rename.Init()
}

func (m *ListViewModel) startDelete() tea.Cmd {
	menu := m.selectedMenu()
	if menu == nil {
		return nil
	}
	return m.showDeleteConfirm(menu)
}

func (m *ListViewModel) showDeleteConfirm(menu *menusmodels.Menu) tea.Cmd {
	if menu == nil {
		return nil
	}
	return func() tea.Msg {
		if menu.Status != menusmodels.MenuStatusDraft {
			return DeleteErrorMsg{Err: errors.Invalidf("only draft menus can be deleted")}
		}
		itemCount := len(menu.Items)
		message := fmt.Sprintf("Delete %q?", menu.Name)
		if itemCount > 0 {
			message = fmt.Sprintf(
				"Delete %q?\n\nThis menu contains %d item(s).",
				menu.Name,
				itemCount,
			)
		}
		confirm := dialog.NewConfirmDialog(
			"Delete Menu",
			message,
			dialog.WithDangerous(),
			dialog.WithFocusCancel(),
			dialog.WithConfirmText("Delete"),
			dialog.WithStyles(m.dialogStyles),
			dialog.WithKeys(m.dialogKeys),
		)
		return showDeleteDialogMsg{dialog: confirm, target: *menu}
	}
}

func (m *ListViewModel) performDelete() tea.Cmd {
	if m.deleteTarget == nil {
		return nil
	}
	target := m.deleteTarget
	return func() tea.Msg {
		deleted, err := m.app.Menu.Delete(m.context(), target.ID)
		if err != nil {
			return DeleteErrorMsg{Err: err}
		}
		return MenuDeletedMsg{Menu: deleted}
	}
}

func (m *ListViewModel) startPublish() tea.Cmd {
	menu := m.selectedMenu()
	if menu == nil {
		return nil
	}
	return m.showPublishConfirm(menu)
}

func (m *ListViewModel) showPublishConfirm(menu *menusmodels.Menu) tea.Cmd {
	if menu == nil {
		return nil
	}
	return func() tea.Msg {
		if menu.Status != menusmodels.MenuStatusDraft {
			return PublishErrorMsg{Err: errors.Invalidf("only draft menus can be published")}
		}
		if len(menu.Items) == 0 {
			return PublishErrorMsg{Err: errors.Invalidf("cannot publish empty menu")}
		}
		message := fmt.Sprintf(
			"Publish menu %q?\n\nThis will make the menu available for orders.\nPublished menus cannot be modified.",
			menu.Name,
		)
		confirm := dialog.NewConfirmDialog(
			"Publish Menu",
			message,
			dialog.WithConfirmText("Publish"),
			dialog.WithStyles(m.dialogStyles),
			dialog.WithKeys(m.dialogKeys),
		)
		return showPublishDialogMsg{dialog: confirm, target: *menu}
	}
}

func (m *ListViewModel) performPublish() tea.Cmd {
	if m.publishTarget == nil {
		return nil
	}
	target := m.publishTarget
	return func() tea.Msg {
		published, err := m.app.Menu.Publish(m.context(), &menusmodels.Menu{ID: target.ID})
		if err != nil {
			return PublishErrorMsg{Err: err}
		}
		return MenuPublishedMsg{Menu: published}
	}
}

func (m *ListViewModel) startDraft() tea.Cmd {
	menu := m.selectedMenu()
	if menu == nil {
		return nil
	}
	return m.showDraftConfirm(menu)
}

func (m *ListViewModel) showDraftConfirm(menu *menusmodels.Menu) tea.Cmd {
	if menu == nil {
		return nil
	}
	return func() tea.Msg {
		if menu.Status != menusmodels.MenuStatusPublished {
			return DraftErrorMsg{Err: errors.Invalidf("only published menus can be drafted")}
		}
		message := fmt.Sprintf(
			"Return %q to draft?\n\nThis will remove the menu from active service.\nCustomers will not be able to order from this menu.",
			menu.Name,
		)
		confirm := dialog.NewConfirmDialog(
			"Draft Menu",
			message,
			dialog.WithDangerous(),
			dialog.WithConfirmText("Draft"),
			dialog.WithStyles(m.dialogStyles),
			dialog.WithKeys(m.dialogKeys),
		)
		return showDraftDialogMsg{dialog: confirm, target: *menu}
	}
}

func (m *ListViewModel) performDraft() tea.Cmd {
	if m.draftTarget == nil {
		return nil
	}
	target := m.draftTarget
	return func() tea.Msg {
		drafted, err := m.app.Menu.Draft(m.context(), &menusmodels.Menu{ID: target.ID})
		if err != nil {
			return DraftErrorMsg{Err: err}
		}
		return MenuDraftedMsg{Menu: drafted}
	}
}

func (m *ListViewModel) selectedMenu() *menusmodels.Menu {
	item, ok := m.list.SelectedItem().(menuItem)
	if !ok {
		return nil
	}
	menu := item.menu
	return &menu
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
	item, ok := m.list.SelectedItem().(menuItem)
	if !ok {
		m.detail.SetMenu(optional.None[menusmodels.Menu]())
		return
	}
	m.detail.SetMenu(optional.Some(item.menu))
}

func (m *ListViewModel) context() *middleware.Context {
	return m.app.Context()
}
