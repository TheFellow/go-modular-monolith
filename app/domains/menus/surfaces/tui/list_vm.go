package tui

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"

	"github.com/TheFellow/go-modular-monolith/app"
	menus "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type listMode int

type detailLoadedMsg struct {
	menuID    entity.MenuID
	token     uint64
	readiness *menusmodels.ReadinessReport
	err       error
}

const (
	listModeBrowsing listMode = iota
	listModeCreating
	listModeRenaming
	listModeTagging
	listModeConfirmingDelete
	listModeConfirmingPublish
	listModeConfirmingDraft
	listModeAddingDrink
	listModeRemovingDrink
	listModeConfirmingRemoveDrink
	listModeAnalyzing
	listModeFiltering
)

var (
	filterMenusKey = key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter"))
	prevMenusKey   = key.NewBinding(key.WithKeys("["), key.WithHelp("[", "previous page"))
	nextMenusKey   = key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next page"))
)

func (m listMode) isConfirming() bool {
	switch m {
	case listModeConfirmingDelete, listModeConfirmingPublish, listModeConfirmingDraft, listModeConfirmingRemoveDrink:
		return true
	case listModeBrowsing, listModeCreating, listModeRenaming, listModeTagging, listModeAddingDrink, listModeRemovingDrink, listModeAnalyzing, listModeFiltering:
		return false
	}
	return false
}

// ListViewModel renders the menus list and detail panes.
type ListViewModel struct {
	app    *app.Session
	styles tui.ListViewStyles
	keys   listViewKeys

	formStyles   forms.FormStyles
	formKeys     forms.FormKeys
	dialogStyles dialog.DialogStyles
	dialogKeys   dialog.DialogKeys

	list         list.Model
	detail       *DetailViewModel
	detailPane   tui.DetailViewport
	mode         listMode
	create       *CreateMenuVM
	rename       *RenameMenuVM
	tags         *components.TagEditor[cedar.EntityUID, tag.Tags]
	dialog       *dialog.ConfirmDialog
	taggedDialog *components.TaggedConfirm[tag.Tags]
	drinkPicker  *drinkPicker
	analysis     *analysisVM
	filter       *filterVM
	request      menus.ListRequest
	next         paging.Cursor
	history      []paging.Cursor
	loadToken    uint64
	detailToken  uint64
	workflowID   uint64
	spinner      tui.Spinner
	loading      bool
	err          error
	actionErr    error
	projector    menus.ActionProjector
	actions      map[actions.ID]actions.State

	deleteTarget      *menusmodels.Menu
	publishTarget     *menusmodels.Menu
	draftTarget       *menusmodels.Menu
	removeDrinkTarget *drinkChoice

	width       int
	height      int
	listWidth   int
	detailWidth int
}

func NewListViewModel(app *app.Session) *ListViewModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = styles.Standard.ListView.Selected
	delegate.Styles.SelectedDesc = styles.Standard.ListView.Selected

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Menus"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.Paginator.Type = paginator.Arabic
	l.SetFilteringEnabled(false)

	vm := &ListViewModel{
		app:          app,
		styles:       styles.Standard.ListView,
		keys:         newListViewKeys(),
		formStyles:   styles.Standard.Form,
		formKeys:     keys.Standard.Form,
		dialogStyles: styles.Standard.Dialog,
		dialogKeys:   keys.Standard.Dialog,
		list:         l,
		detail:       NewDetailViewModel(styles.Standard.ListView, app),
		detailPane:   tui.NewDetailViewport(),
		loading:      true,
		projector:    menus.NewActionProjector(),
	}
	if app != nil {
		vm.syncActions()
	}
	vm.spinner = tui.NewSpinner("Loading menus...", vm.styles.Subtitle)
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	if !m.actionEnabled(menus.ControlList) {
		m.loading = false
		return nil
	}
	m.loading = true
	return tea.Batch(m.spinner.Init(), m.loadMenus(""))
}

func (m *ListViewModel) Interaction() tui.Interaction {
	return tui.Interaction{
		HandlesBack:  m.mode != listModeBrowsing,
		CapturesText: m.mode == listModeFiltering || m.mode == listModeCreating || m.mode == listModeRenaming || m.mode == listModeTagging || m.mode == listModeAddingDrink || m.mode == listModeRemovingDrink || m.mode == listModeAnalyzing,
	}
}

func (m *ListViewModel) Update(msg tea.Msg) (tui.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.DataInvalidatedMsg:
		if m.mode != listModeBrowsing || !m.actionEnabled(menus.ControlList) {
			return m, nil
		}
		m.loading, m.err = true, nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		switch m.mode {
		case listModeBrowsing:
		case listModeCreating:
			m.create.SetWidth(m.detailWidth)
		case listModeRenaming:
			m.rename.SetWidth(m.detailWidth)
		case listModeTagging:
			m.tags.SetWidth(m.width)
		case listModeConfirmingDelete, listModeConfirmingRemoveDrink:
			m.dialog.SetWidth(m.width)
		case listModeConfirmingPublish, listModeConfirmingDraft:
			m.taggedDialog.SetWidth(m.width)
		case listModeAddingDrink, listModeRemovingDrink, listModeAnalyzing:
		case listModeFiltering:
			m.filter.form.SetWidth(m.detailWidth)
		}
		return m, nil
	case MenuCreatedMsg:
		m.mode = listModeBrowsing
		m.create = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
	case MenuRenamedMsg:
		m.mode = listModeBrowsing
		m.rename = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
	case components.TagsSavedMsg[cedar.EntityUID, tag.Tags]:
		if m.mode != listModeTagging || m.tags == nil || !m.tags.Owns(msg.Target) {
			return m, nil
		}
		m.mode, m.tags, m.loading, m.err = listModeBrowsing, nil, true, nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
	case MenuDeletedMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.deleteTarget = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
	case MenuPublishedMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.publishTarget = nil
		m.taggedDialog = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
	case MenuDraftedMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.draftTarget = nil
		m.taggedDialog = nil
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
	case DeleteErrorMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.deleteTarget = nil
		m.err = msg.Err
		return m, nil
	case PublishErrorMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.publishTarget = nil
		m.taggedDialog = nil
		m.err = msg.Err
		return m, nil
	case DraftErrorMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.draftTarget = nil
		m.taggedDialog = nil
		m.err = msg.Err
		return m, nil
	case drinkChoicesLoadedMsg:
		if msg.workflowID == m.workflowID && m.drinkPicker != nil {
			m.drinkPicker.setChoices(msg.choices, msg.err)
		}
		return m, nil
	case drinkAddedMsg:
		if msg.workflowID != m.workflowID || m.mode != listModeAddingDrink || m.drinkPicker == nil {
			return m, nil
		}
		if msg.err != nil {
			m.drinkPicker.saving = false
			m.err = msg.err
			if m.drinkPicker != nil {
				m.drinkPicker.err = msg.err
			}
			return m, nil
		}
		m.mode, m.drinkPicker, m.loading, m.err = listModeBrowsing, nil, true, nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
	case drinkRemovedMsg:
		if msg.workflowID != m.workflowID {
			return m, nil
		}
		if msg.err != nil {
			m.mode, m.dialog, m.removeDrinkTarget, m.err = listModeBrowsing, nil, nil, msg.err
			return m, nil
		}
		m.mode, m.dialog, m.removeDrinkTarget, m.drinkPicker, m.loading, m.err = listModeBrowsing, nil, nil, nil, true, nil
		return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
	case analysisLoadedMsg:
		if msg.workflowID == m.workflowID && m.mode == listModeAnalyzing && m.analysis != nil {
			m.analysis.loading, m.analysis.err = false, msg.err
			if msg.err == nil {
				value := msg.value
				m.analysis.result = &value
			}
		}
		return m, nil
	case showDeleteDialogMsg:
		m.mode = listModeConfirmingDelete
		m.dialog = msg.dialog
		m.deleteTarget = &msg.target
		m.publishTarget = nil
		m.draftTarget = nil
		m.dialog.SetWidth(m.width)
		return m, nil
	case showPublishDialogMsg:
		m.mode = listModeConfirmingPublish
		m.taggedDialog = msg.dialog
		m.publishTarget = &msg.target
		m.deleteTarget = nil
		m.draftTarget = nil
		m.taggedDialog.SetWidth(m.width)
		return m, m.taggedDialog.Init()
	case showDraftDialogMsg:
		m.mode = listModeConfirmingDraft
		m.taggedDialog = msg.dialog
		m.draftTarget = &msg.target
		m.deleteTarget = nil
		m.publishTarget = nil
		m.taggedDialog.SetWidth(m.width)
		return m, m.taggedDialog.Init()
	case dialog.ConfirmMsg:
		mode := m.mode
		m.mode = listModeBrowsing
		m.dialog = nil
		switch mode {
		case listModeConfirmingDelete:
			return m, m.performDelete()
		case listModeConfirmingPublish:
			return m, m.performPublish()
		case listModeConfirmingDraft:
			return m, m.performDraft()
		case listModeConfirmingRemoveDrink:
			return m, m.performRemoveDrink()
		case listModeBrowsing, listModeCreating, listModeRenaming, listModeTagging, listModeAddingDrink, listModeRemovingDrink, listModeAnalyzing, listModeFiltering:
			panic(fmt.Sprintf("confirm message received in %v mode", m.mode))
		}
		return m, nil
	case dialog.CancelMsg:
		m.mode = listModeBrowsing
		m.dialog = nil
		m.deleteTarget = nil
		m.publishTarget = nil
		m.draftTarget = nil
		m.removeDrinkTarget = nil
		m.taggedDialog = nil
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case listModeBrowsing:
		case listModeConfirmingDelete, listModeConfirmingPublish, listModeConfirmingDraft, listModeConfirmingRemoveDrink:
		case listModeCreating:
			if key.Matches(msg, m.keys.Back) && !m.create.form.IsEditing() {
				m.mode = listModeBrowsing
				m.create = nil
				return m, nil
			}
		case listModeRenaming:
			if key.Matches(msg, m.keys.Back) && !m.rename.form.IsEditing() {
				m.mode = listModeBrowsing
				m.rename = nil
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
		case listModeAddingDrink, listModeRemovingDrink, listModeAnalyzing:
			if key.Matches(msg, m.keys.Back) {
				if (m.drinkPicker != nil && m.drinkPicker.saving) || (m.analysis != nil && m.analysis.loading) {
					return m, nil
				}
				m.workflowID++
				m.mode, m.drinkPicker, m.analysis = listModeBrowsing, nil, nil
				return m, nil
			}
		case listModeFiltering:
			if key.Matches(msg, m.keys.Back) && !m.filter.form.IsEditing() {
				m.mode, m.filter = listModeBrowsing, nil
				return m, nil
			}
			if key.Matches(msg, m.formKeys.Submit) {
				if !m.actionEnabled(menus.ControlList) {
					return m, nil
				}
				req, err := m.filter.Request()
				if err != nil {
					return m, nil
				}
				m.request, m.history, m.next = req, nil, ""
				m.mode, m.filter, m.loading, m.err = listModeBrowsing, nil, true, nil
				return m, tea.Batch(m.spinner.Init(), m.loadMenus(""))
			}
		}
		if m.mode != listModeBrowsing {
			break
		}
		switch {
		case key.Matches(msg, m.keys.Refresh):
			if !m.actionEnabled(menus.ControlList) {
				return m, nil
			}
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
		case key.Matches(msg, filterMenusKey):
			if !m.actionEnabled(menus.ControlList) {
				return m, nil
			}
			m.mode, m.filter = listModeFiltering, newFilterVM(m.request)
			m.filter.form.SetWidth(m.detailWidth)
			return m, m.filter.Init()
		case key.Matches(msg, nextMenusKey) && m.next != "" && m.actionEnabled(menus.ControlList):
			m.history = append(m.history, m.request.Cursor)
			m.request.Cursor = m.next
			m.loading = true
			return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
		case key.Matches(msg, prevMenusKey) && len(m.history) > 0 && m.actionEnabled(menus.ControlList):
			i := len(m.history) - 1
			m.request.Cursor = m.history[i]
			m.history = m.history[:i]
			m.loading = true
			return m, tea.Batch(m.spinner.Init(), m.loadMenus(m.request.Cursor))
		case key.Matches(msg, m.keys.Create):
			if !m.actionEnabled(menus.ControlCreate) {
				return m, nil
			}
			return m, m.startCreate()
		case key.Matches(msg, m.keys.Edit), key.Matches(msg, m.keys.Enter):
			if !m.actionEnabled(menus.ControlEdit) {
				return m, nil
			}
			return m, m.startRename()
		case key.Matches(msg, m.keys.Delete):
			if !m.actionEnabled(menus.ControlDelete) {
				return m, nil
			}
			return m, m.startDelete()
		case key.Matches(msg, m.keys.Publish):
			if !m.actionEnabled(menus.ControlPublish) {
				return m, nil
			}
			return m, m.startPublish()
		case key.Matches(msg, m.keys.Draft):
			if !m.actionEnabled(menus.ControlDraft) {
				return m, nil
			}
			return m, m.startDraft()
		case key.Matches(msg, m.keys.Tags):
			if !m.actionEnabled(menus.ControlTags) {
				return m, nil
			}
			return m, m.startTags()
		case key.Matches(msg, addDrinkKey):
			if !m.actionEnabled(menus.ControlAddDrink) {
				return m, nil
			}
			return m, m.startDrinkPicker(false)
		case key.Matches(msg, removeDrinkKey):
			if !m.actionEnabled(menus.ControlRemoveDrink) {
				return m, nil
			}
			return m, m.startDrinkPicker(true)
		case key.Matches(msg, analyzeKey):
			return m, m.startAnalysis()
		}
	case MenusLoadedMsg:
		if msg.Token != m.loadToken {
			return m, nil
		}
		m.loading = false
		m.err = msg.Err
		if msg.Err != nil {
			return m, nil
		}
		m.next = msg.Next
		selected := selectedMenuID(m.selectedMenu())
		items := make([]list.Item, 0, len(msg.Menus))
		for _, menu := range msg.Menus {
			items = append(items, newMenuItem(menu, m.styles))
		}
		m.list.SetItems(items)
		selectMenuID(&m.list, selected)
		m.syncDetail()
		m.detail.SetDrinkNames(msg.Names)
		m.syncActions()
		return m, m.loadDetail()
	case detailLoadedMsg:
		selected := m.selectedMenu()
		if selected == nil || selected.ID != msg.menuID || msg.token != m.detailToken {
			return m, nil
		}
		m.detail.SetReadiness(msg.readiness, msg.err)
		if msg.err != nil {
			m.err = msg.err
		} else if msg.readiness != nil {
			m.actions = menus.ApplyReadiness(m.actions, *msg.readiness)
			m.detail.SetActions(m.actions)
		}
		return m, nil
	}

	switch m.mode {
	case listModeBrowsing:
	case listModeConfirmingDelete, listModeConfirmingRemoveDrink:
		var cmd tea.Cmd
		m.dialog, cmd = m.dialog.Update(msg)
		return m, cmd
	case listModeConfirmingPublish, listModeConfirmingDraft:
		var cmd tea.Cmd
		m.taggedDialog, cmd = m.taggedDialog.Update(msg)
		return m, cmd
	case listModeRenaming:
		var cmd tea.Cmd
		m.rename, cmd = m.rename.Update(msg)
		return m, cmd
	case listModeCreating:
		var cmd tea.Cmd
		m.create, cmd = m.create.Update(msg)
		return m, cmd
	case listModeTagging:
		var cmd tea.Cmd
		m.tags, cmd = m.tags.Update(msg)
		return m, cmd
	case listModeAddingDrink:
		if typed, ok := msg.(tea.KeyMsg); ok && typed.Type == tea.KeyEnter {
			if m.drinkPicker.saving {
				return m, nil
			}
			choice, exists := m.drinkPicker.choice()
			if !exists {
				return m, nil
			}
			menu := m.selectedMenu()
			desired, err := m.drinkPicker.desiredTags()
			if err != nil {
				m.drinkPicker.err = err
				return m, nil
			}
			m.drinkPicker.saving = true
			workflowID := m.workflowID
			return m, func() tea.Msg {
				_, err := app.RunTaggedMutation(m.app.App, m.context(), desired, func(ctx *middleware.Context) (*menusmodels.Menu, error) {
					return m.app.Menus.AddDrink(ctx, &menusmodels.MenuPatch{MenuID: menu.ID, DrinkID: choice.id})
				})
				return drinkAddedMsg{workflowID: workflowID, err: err}
			}
		}
		return m, m.drinkPicker.update(msg)
	case listModeRemovingDrink:
		if typed, ok := msg.(tea.KeyMsg); ok && typed.Type == tea.KeyEnter {
			return m, m.confirmRemoveDrink()
		}
		return m, m.drinkPicker.update(msg)
	case listModeAnalyzing:
		if typed, ok := msg.(tea.KeyMsg); ok && typed.Type == tea.KeyEnter {
			return m, m.runAnalysis()
		}
		var cmd tea.Cmd
		m.analysis.input, cmd = m.analysis.input.Update(msg)
		return m, cmd
	case listModeFiltering:
		return m, m.filter.Update(msg)
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	if m.detailPane.Update(msg) {
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.syncDetail()
	m.syncActions()
	return m, tea.Batch(cmd, m.loadDetail())
}

func (m *ListViewModel) View() string {
	if m.loading {
		return m.renderLoading()
	}

	if m.mode.isConfirming() {
		dialogView := ""
		if m.mode == listModeConfirmingPublish || m.mode == listModeConfirmingDraft {
			dialogView = m.taggedDialog.View()
		} else {
			dialogView = m.dialog.View()
		}
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialogView)
		}
		return dialogView
	}
	if m.mode == listModeTagging {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.tags.View())
	}
	if m.mode == listModeAddingDrink || m.mode == listModeRemovingDrink {
		title := "Add drink to menu"
		if m.mode == listModeRemovingDrink {
			title = "Remove drink from menu"
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.drinkPicker.view(title))
	}
	if m.mode == listModeAnalyzing {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.analysis.view())
	}
	if m.mode == listModeFiltering {
		return m.filter.View()
	}

	listView := m.list.View()
	if m.err != nil {
		listView = m.styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.err))
	}
	listView = m.styles.ListPane.Width(tui.PaneStyleWidth(m.styles.ListPane, m.listWidth)).Render(listView)

	detailView := m.detailPane.View(m.detail.View())
	switch m.mode {
	case listModeBrowsing, listModeConfirmingDelete, listModeConfirmingPublish, listModeConfirmingDraft, listModeConfirmingRemoveDrink:
	case listModeTagging:
	case listModeAddingDrink, listModeRemovingDrink, listModeAnalyzing:
	case listModeCreating:
		detailView = m.create.View()
	case listModeRenaming:
		detailView = m.rename.View()
	case listModeFiltering:
	}
	detailView = m.styles.DetailPane.Width(tui.PaneStyleWidth(m.styles.DetailPane, m.detailWidth)).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	switch m.mode {
	case listModeConfirmingDelete, listModeConfirmingPublish, listModeConfirmingDraft, listModeConfirmingRemoveDrink:
		return []key.Binding{m.dialogKeys.Confirm, m.keys.Back, m.dialogKeys.Switch}
	case listModeTagging:
		return []key.Binding{m.formKeys.Submit, m.keys.Back}
	case listModeCreating, listModeRenaming:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Edit, m.keys.Enter, m.formKeys.Submit, m.keys.Back}
	case listModeFiltering:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Edit, m.keys.Enter, m.formKeys.Submit, m.keys.Back}
	case listModeBrowsing:
		bindings := []key.Binding{}
		if m.actionEnabled(menus.ControlList) {
			bindings = append(bindings, m.keys.Up, m.keys.Down, prevMenusKey, nextMenusKey, filterMenusKey)
		}
		bindings = m.appendVisibleBindings(bindings)
		if m.actionEnabled(menus.ControlList) {
			bindings = append(bindings, analyzeKey, m.keys.Refresh)
		}
		return append(bindings, tui.DetailScrollHelp, m.keys.Back)
	case listModeAddingDrink, listModeRemovingDrink, listModeAnalyzing:
	}
	if m.mode == listModeAddingDrink || m.mode == listModeRemovingDrink || m.mode == listModeAnalyzing {
		return []key.Binding{m.keys.Enter, m.keys.Back}
	}
	return nil
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	switch m.mode {
	case listModeConfirmingDelete, listModeConfirmingPublish, listModeConfirmingDraft, listModeConfirmingRemoveDrink:
		return [][]key.Binding{
			{m.dialogKeys.Confirm, m.keys.Back},
			{m.dialogKeys.Switch},
		}
	case listModeTagging:
		return [][]key.Binding{{m.formKeys.Submit, m.keys.Back}}
	case listModeCreating, listModeRenaming:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down, m.keys.Edit, m.keys.Enter, m.formKeys.Submit},
			{m.keys.Back},
		}
	case listModeFiltering:
		return [][]key.Binding{{m.keys.Up, m.keys.Down, m.keys.Edit, m.keys.Enter, m.formKeys.Submit}, {m.keys.Back}}
	case listModeBrowsing:
		actionsHelp := m.visibleBindings()
		navigation := []key.Binding{}
		collection := []key.Binding{}
		analysisHelp := []key.Binding{}
		footer := []key.Binding{m.keys.Back}
		if m.actionEnabled(menus.ControlList) {
			navigation = append(navigation, m.keys.Up, m.keys.Down)
			collection = append(collection, prevMenusKey, nextMenusKey, filterMenusKey)
			analysisHelp = append(analysisHelp, analyzeKey)
			footer = append([]key.Binding{m.keys.Refresh}, footer...)
		}
		collection = append(collection, tui.DetailScrollHelp)
		if m.actionEnabled(menus.ControlEdit) {
			navigation = append(navigation, m.keys.Enter)
		}
		return [][]key.Binding{
			navigation,
			collection,
			actionsHelp,
			analysisHelp,
			footer,
		}
	case listModeAddingDrink, listModeRemovingDrink, listModeAnalyzing:
	}
	if m.mode == listModeAddingDrink || m.mode == listModeRemovingDrink || m.mode == listModeAnalyzing {
		return [][]key.Binding{{m.keys.Enter, m.keys.Back}}
	}
	return nil
}

func (m *ListViewModel) loadMenus(cursor paging.Cursor) tea.Cmd {
	if !m.actionEnabled(menus.ControlList) {
		return nil
	}
	m.loadToken++
	token := m.loadToken
	req := m.request
	req.Cursor = cursor
	return func() tea.Msg {
		page, err := m.app.Menus.List(m.context(), req)
		if err != nil {
			return MenusLoadedMsg{Err: err, Token: token}
		}

		menus := make([]menusmodels.Menu, 0, len(page.Items))
		names := make(map[entity.DrinkID]string)
		for i, menu := range page.Items {
			if menu == nil {
				return MenusLoadedMsg{Err: errors.Internalf("menu %d missing", i), Token: token}
			}
			menus = append(menus, *menu)
			for _, item := range menu.Items {
				if name, ok := item.DisplayName.Unwrap(); ok && strings.TrimSpace(name) != "" {
					names[item.DrinkID] = strings.TrimSpace(name)
					continue
				}
				drink, getErr := m.app.Drinks.Get(m.context(), item.DrinkID)
				if getErr != nil {
					return MenusLoadedMsg{Err: getErr, Token: token}
				}
				names[item.DrinkID] = drink.Name
			}
		}

		return MenusLoadedMsg{Menus: menus, Next: page.Next, Token: token, Names: names}
	}
}

func (m *ListViewModel) startCreate() tea.Cmd {
	m.mode = listModeCreating
	m.create = NewCreateMenuVM(m.app)
	m.create.SetWidth(m.detailWidth)
	return m.create.Init()
}

type showDeleteDialogMsg struct {
	dialog *dialog.ConfirmDialog
	target menusmodels.Menu
}

type showPublishDialogMsg struct {
	dialog *components.TaggedConfirm[tag.Tags]
	target menusmodels.Menu
}

type showDraftDialogMsg struct {
	dialog *components.TaggedConfirm[tag.Tags]
	target menusmodels.Menu
}

func (m *ListViewModel) startRename() tea.Cmd {
	menu := m.selectedMenu()
	if menu == nil {
		return nil
	}
	m.mode = listModeRenaming
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
	target := *m.deleteTarget
	return func() tea.Msg {
		deleted, err := m.app.Menus.Delete(m.context(), target.ID)
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
		return showPublishDialogMsg{dialog: components.NewTaggedConfirm(menu.Tags.Canonical().String(), tag.ParseCollection, confirm), target: *menu}
	}
}

func (m *ListViewModel) performPublish() tea.Cmd {
	target := *m.publishTarget
	desired, err := m.taggedDialog.DesiredTags()
	if err != nil {
		return func() tea.Msg { return PublishErrorMsg{Err: err} }
	}
	return func() tea.Msg {
		published, err := app.RunTaggedMutation(m.app.App, m.context(), desired, func(ctx *middleware.Context) (*menusmodels.Menu, error) {
			return m.app.Menus.Publish(ctx, &menusmodels.Menu{ID: target.ID})
		})
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
		return showDraftDialogMsg{dialog: components.NewTaggedConfirm(menu.Tags.Canonical().String(), tag.ParseCollection, confirm), target: *menu}
	}
}

func (m *ListViewModel) performDraft() tea.Cmd {
	target := *m.draftTarget
	desired, err := m.taggedDialog.DesiredTags()
	if err != nil {
		return func() tea.Msg { return DraftErrorMsg{Err: err} }
	}
	return func() tea.Msg {
		drafted, err := app.RunTaggedMutation(m.app.App, m.context(), desired, func(ctx *middleware.Context) (*menusmodels.Menu, error) {
			return m.app.Menus.Draft(ctx, &menusmodels.Menu{ID: target.ID})
		})
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
	menu := item.Value
	return &menu
}

func selectedMenuID(value *menusmodels.Menu) entity.MenuID {
	if value == nil {
		return entity.MenuID{}
	}
	return value.ID
}

func selectMenuID(values *list.Model, id entity.MenuID) {
	if id.IsZero() {
		return
	}
	for i, item := range values.Items() {
		if menu, ok := item.(menuItem); ok && menu.Value.ID == id {
			values.Select(i)
			return
		}
	}
}

func (m *ListViewModel) startTags() tea.Cmd {
	menu := m.selectedMenu()
	if menu == nil {
		return nil
	}
	m.mode = listModeTagging
	m.tags = components.NewTagEditor(m.app.ReplaceTags, tag.ParseCollection, menu.EntityUID(), menu.Name, menu.Tags.Canonical().String())
	m.tags.SetWidth(m.width)
	return m.tags.Init()
}

func (m *ListViewModel) startDrinkPicker(removing bool) tea.Cmd {
	menu := m.selectedMenu()
	if menu == nil {
		return nil
	}
	if removing && len(menu.Items) == 0 {
		return nil
	}
	m.err = nil
	m.workflowID++
	m.drinkPicker = newDrinkPicker(menu.Tags)
	if removing {
		m.mode = listModeRemovingDrink
	} else {
		m.mode = listModeAddingDrink
	}
	return tea.Batch(m.drinkPicker.query.Focus(), loadDrinkChoices(m.app, *menu, removing, m.workflowID))
}

func (m *ListViewModel) confirmRemoveDrink() tea.Cmd {
	choice, ok := m.drinkPicker.choice()
	menu := m.selectedMenu()
	if !ok || menu == nil {
		return nil
	}
	m.removeDrinkTarget = &choice
	m.mode = listModeConfirmingRemoveDrink
	m.dialog = dialog.NewConfirmDialog(
		"Remove Drink",
		fmt.Sprintf("Remove %q from %q?", choice.name, menu.Name),
		dialog.WithDangerous(), dialog.WithFocusCancel(), dialog.WithConfirmText("Remove"),
		dialog.WithStyles(m.dialogStyles), dialog.WithKeys(m.dialogKeys),
	)
	m.dialog.SetWidth(m.width)
	return nil
}

func (m *ListViewModel) performRemoveDrink() tea.Cmd {
	menu := m.selectedMenu()
	choice := m.removeDrinkTarget
	if menu == nil || choice == nil {
		return func() tea.Msg {
			return drinkRemovedMsg{workflowID: m.workflowID, err: errors.Invalidf("menu drink is not selected")}
		}
	}
	menuID, drinkID := menu.ID, choice.id
	desired, err := m.drinkPicker.desiredTags()
	if err != nil {
		return func() tea.Msg { return drinkRemovedMsg{workflowID: m.workflowID, err: err} }
	}
	workflowID := m.workflowID
	return func() tea.Msg {
		_, err := app.RunTaggedMutation(m.app.App, m.context(), desired, func(ctx *middleware.Context) (*menusmodels.Menu, error) {
			return m.app.Menus.RemoveDrink(ctx, &menusmodels.MenuPatch{MenuID: menuID, DrinkID: drinkID})
		})
		return drinkRemovedMsg{workflowID: workflowID, err: err}
	}
}

func (m *ListViewModel) startAnalysis() tea.Cmd {
	if m.selectedMenu() == nil {
		return nil
	}
	m.workflowID++
	m.mode, m.analysis, m.err = listModeAnalyzing, newAnalysisVM(), nil
	return m.analysis.input.Focus()
}

func (m *ListViewModel) runAnalysis() tea.Cmd {
	if m.analysis == nil || m.analysis.loading {
		return nil
	}
	target, err := m.analysis.target()
	if err != nil {
		m.analysis.err = err
		m.analysis.result = nil
		return nil
	}
	menu := m.selectedMenu()
	if menu == nil {
		m.analysis.err = errors.Invalidf("menu is not selected")
		return nil
	}
	m.analysis.loading, m.analysis.err, m.analysis.result = true, nil, nil
	value := *menu
	workflowID := m.workflowID
	return func() tea.Msg {
		analysis, err := m.app.Menus.Analyze(m.context(), value, target)
		return analysisLoadedMsg{workflowID: workflowID, value: analysis, err: err}
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

	listWidth, detailWidth := tui.SplitListDetailWidths(width)
	listWidth = tui.PaneContentWidth(m.styles.ListPane, listWidth)
	detailWidth = tui.PaneContentWidth(m.styles.DetailPane, detailWidth)

	m.list.SetSize(listWidth, height)
	m.detail.SetSize(detailWidth, height)
	_, frameHeight := m.styles.DetailPane.GetFrameSize()
	m.detailPane.SetSize(detailWidth, max(height-frameHeight, 1))
	m.listWidth = listWidth
	m.detailWidth = detailWidth
}

func (m *ListViewModel) syncDetail() {
	m.detail.SetReadiness(nil, nil)
	item, ok := m.list.SelectedItem().(menuItem)
	if !ok {
		m.detail.SetMenu(optional.None[menusmodels.Menu]())
		return
	}
	m.detail.SetMenu(optional.Some(item.Value))
}

func (m *ListViewModel) syncActions() {
	states, err := m.projector.Project(m.context(), m.context().Principal(), m.selectedMenu())
	if err != nil {
		m.actions = nil
		m.actionErr = err
		m.err = err
		m.detail.SetActions(nil)
		return
	}
	if m.actionErr != nil && errors.Is(m.err, m.actionErr) {
		m.err = nil
	}
	m.actionErr = nil
	m.actions = make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		m.actions[state.ID] = state
	}
	m.detail.SetReadiness(nil, nil)
	m.detail.SetActions(m.actions)
	if !m.actionEnabled(menus.ControlList) {
		m.list.SetItems(nil)
		m.syncDetail()
	}
}

func (m *ListViewModel) loadDetail() tea.Cmd {
	m.detailToken++
	token := m.detailToken
	menu := m.selectedMenu()
	if menu == nil {
		m.detail.SetDrinkNames(nil)
		m.detail.SetReadiness(nil, nil)
		return nil
	}
	target := *menu
	readiness := m.actions[menus.ControlReadiness]
	return func() tea.Msg {
		var report *menusmodels.ReadinessReport
		if readiness.Visible && readiness.Enabled {
			value, err := m.app.Menus.Readiness(m.context(), target.ID)
			if err != nil {
				return detailLoadedMsg{menuID: target.ID, token: token, err: err}
			}
			report = &value
		}
		return detailLoadedMsg{menuID: target.ID, token: token, readiness: report}
	}
}

func (m *ListViewModel) actionEnabled(id actions.ID) bool {
	state, ok := m.actions[id]
	return ok && state.Visible && state.Enabled
}

func (m *ListViewModel) appendVisibleBindings(dst []key.Binding) []key.Binding {
	return append(dst, m.visibleBindings()...)
}

func (m *ListViewModel) visibleBindings() []key.Binding {
	pairs := []struct {
		id      actions.ID
		binding key.Binding
	}{
		{menus.ControlCreate, m.keys.Create}, {menus.ControlEdit, m.keys.Edit},
		{menus.ControlDelete, m.keys.Delete}, {menus.ControlPublish, m.keys.Publish},
		{menus.ControlDraft, m.keys.Draft}, {menus.ControlTags, m.keys.Tags},
		{menus.ControlAddDrink, addDrinkKey}, {menus.ControlRemoveDrink, removeDrinkKey},
	}
	bindings := make([]key.Binding, 0, len(pairs))
	for _, pair := range pairs {
		if state, ok := m.actions[pair.id]; ok && state.Visible && state.Enabled {
			bindings = append(bindings, pair.binding)
		}
	}
	return bindings
}

func (m *ListViewModel) context() *middleware.Context {
	return m.app.Context()
}
