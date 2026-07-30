package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TheFellow/go-modular-monolith/app"
	auditui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/tui"
	drinksui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/tui"
	ingredientsui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/tui"
	inventoryui "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/tui"
	menusui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/tui"
	ordersui "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/surfaces/tui/keys"
	"github.com/TheFellow/go-modular-monolith/app/surfaces/tui/styles"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

const (
	MinWidth        = 80
	MinHeight       = 24
	titleBarHeight  = 2
	statusBarHeight = 1
)

type viewSizeMsg struct {
	width  int
	height int
}

// App is the root model for the TUI application.
type App struct {
	// Navigation
	currentView View
	prevViews   []View

	// Application layer
	app *app.Session

	// UI State
	styles    styles.Styles
	keys      keys.KeyMap
	help      help.Model
	width     int
	height    int
	showHelp  bool
	lastError error

	// Child views (lazy initialized)
	views map[View]views.ViewModel
}

// NewApp creates a new App with the given application.
func NewApp(application *app.Session) *App {
	helpModel := help.New()
	helpModel.ShowAll = false

	return &App{
		currentView: ViewDashboard,
		app:         application,
		styles:      styles.App,
		keys:        keys.App,
		help:        helpModel,
		views:       make(map[View]views.ViewModel),
	}
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return a.currentViewModel().Init()
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		vm := a.currentViewModel()
		interaction := vm.Interaction()
		if msg.Type != tea.KeyRunes || !interaction.CapturesText {
			if key.Matches(msg, a.keys.Quit) {
				return a, tea.Quit
			}
			if key.Matches(msg, a.keys.Help) {
				a.showHelp = !a.showHelp
				vm, cmd := vm.Update(tea.WindowSizeMsg{
					Width: a.width, Height: a.availableHeight(),
				})
				a.views[a.currentView] = vm
				return a, cmd
			}
		}
		if key.Matches(msg, a.keys.Back) {
			if a.showHelp {
				a.showHelp = false
				vm, cmd := vm.Update(tea.WindowSizeMsg{
					Width: a.width, Height: a.availableHeight(),
				})
				a.views[a.currentView] = vm
				return a, cmd
			}
			if interaction.HandlesBack {
				vm, cmd := vm.Update(msg)
				a.views[a.currentView] = vm
				return a, cmd
			}
			return a, a.navigateBack()
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.help.Width = msg.Width
		vm, cmd := a.currentViewModel().Update(tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: a.availableHeight(),
		})
		a.views[a.currentView] = vm
		return a, cmd

	case NavigateMsg:
		return a, a.navigateTo(msg.To)

	case ErrorMsg:
		a.lastError = msg.Err
		return a, nil

	case viewSizeMsg:
		vm, cmd := a.currentViewModel().Update(tea.WindowSizeMsg{
			Width:  msg.width,
			Height: msg.height,
		})
		a.views[a.currentView] = vm
		return a, cmd
	}

	vm, cmd := a.currentViewModel().Update(msg)
	a.views[a.currentView] = vm
	return a, cmd
}

// View implements tea.Model.
func (a *App) View() string {
	if a.width > 0 && a.height > 0 && (a.width < MinWidth || a.height < MinHeight) {
		return a.renderTooSmallWarning()
	}

	title := a.titleBarView()
	content := a.currentViewModel().View()
	status := a.statusBarView()

	parts := []string{title, content, status}
	if a.showHelp {
		a.help.ShowAll = true
		parts = append(parts, a.help.View(a.currentViewModel()))
	} else {
		a.help.ShowAll = false
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// currentViewModel returns the ViewModel for the current view, lazy initializing if needed.
func (a *App) currentViewModel() views.ViewModel {
	if a.views == nil {
		a.views = make(map[View]views.ViewModel)
	}

	if vm, ok := a.views[a.currentView]; ok {
		return vm
	}

	var vm views.ViewModel
	switch a.currentView {
	case ViewDashboard:
		vm = views.NewDashboard(a.app)
	case ViewDrinks:
		vm = drinksui.NewListViewModel(a.app)
	case ViewIngredients:
		vm = ingredientsui.NewListViewModel(a.app)
	case ViewInventory:
		vm = inventoryui.NewListViewModel(a.app)
	case ViewMenus:
		vm = menusui.NewListViewModel(a.app)
	case ViewOrders:
		vm = ordersui.NewListViewModel(a.app)
	case ViewAudit:
		vm = auditui.NewListViewModel(a.app)
	case ViewTags:
		vm = views.NewTags(a.app)
	default:
		a.currentView = ViewDashboard
		vm = views.NewDashboard(a.app)
	}

	a.views[a.currentView] = vm
	return vm
}

// navigateTo pushes current view to stack and switches to target.
func (a *App) navigateTo(target View) tea.Cmd {
	if !isValidView(target) || target == a.currentView {
		return nil
	}

	a.prevViews = append(a.prevViews, a.currentView)
	a.currentView = target

	if a.currentView == ViewDashboard {
		delete(a.views, ViewDashboard)
	}

	if _, ok := a.views[target]; ok {
		return a.syncWindowCmd()
	}

	return a.initializeCurrentView()
}

// navigateBack pops the previous view from the stack.
func (a *App) navigateBack() tea.Cmd {
	if len(a.prevViews) == 0 {
		if a.currentView != ViewDashboard {
			a.currentView = ViewDashboard
			delete(a.views, ViewDashboard)
			return a.initializeCurrentView()
		}
		return nil
	}

	idx := len(a.prevViews) - 1
	a.currentView = a.prevViews[idx]
	a.prevViews = a.prevViews[:idx]
	if a.currentView == ViewDashboard {
		delete(a.views, ViewDashboard)
		return a.initializeCurrentView()
	}
	return a.syncWindowCmd()
}

// initializeCurrentView sizes a newly created child before its Init command can
// produce a renderable result. A resize command batched with Init may arrive
// later, briefly exposing an unbounded frame to Bubble Tea's renderer.
func (a *App) initializeCurrentView() tea.Cmd {
	vm := a.currentViewModel()
	var sizeCmd tea.Cmd
	if a.width > 0 || a.height > 0 {
		vm, sizeCmd = vm.Update(tea.WindowSizeMsg{Width: a.width, Height: a.availableHeight()})
		a.views[a.currentView] = vm
	}
	return tea.Batch(sizeCmd, vm.Init())
}

func (a *App) syncWindowCmd() tea.Cmd {
	if a.width == 0 && a.height == 0 {
		return nil
	}

	return func() tea.Msg {
		return viewSizeMsg{width: a.width, height: a.availableHeight()}
	}
}

func (a *App) availableHeight() int {
	height := a.height - titleBarHeight - statusBarHeight - a.helpHeight()
	if height < 0 {
		return 0
	}
	return height
}

func (a *App) titleBarView() string {
	title := "Mixology > " + viewTitle(a.currentView)
	style := a.styles.TitleBar
	if a.width > 0 {
		style = style.Width(a.width)
	}
	return style.Render(title)
}

func (a *App) helpHeight() int {
	if !a.showHelp {
		return 0
	}
	a.help.ShowAll = true
	return lipgloss.Height(a.help.View(a.currentViewModel()))
}

func (a *App) statusBarView() string {
	var content string
	if a.lastError != nil {
		tuiErr := errors.ToTUIError(a.lastError)
		var style lipgloss.Style
		switch tuiErr.Style {
		case errors.TUIStyleWarning:
			style = a.styles.WarningText
		case errors.TUIStyleInfo:
			style = a.styles.InfoText
		case errors.TUIStyleError:
			style = a.styles.ErrorText
		default:
			style = a.styles.ErrorText
		}
		content = style.Render(tuiErr.Message)
	} else {
		content = a.styles.HelpDesc.Render("View: " + viewTitle(a.currentView) + "  •  Press ? for help")
	}

	style := a.styles.StatusBar
	if a.width > 0 {
		style = style.Width(a.width)
	}

	return style.Render(content)
}

func (a *App) renderTooSmallWarning() string {
	title := a.styles.ErrorText.Render("Terminal too small")
	minimum := a.styles.HelpDesc.Render(fmt.Sprintf("Minimum: %dx%d", MinWidth, MinHeight))
	current := a.styles.HelpDesc.Render(fmt.Sprintf("Current: %dx%d", a.width, a.height))
	content := lipgloss.JoinVertical(lipgloss.Center, title, minimum, current)

	if a.width > 0 && a.height > 0 {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, content)
	}

	return content
}

func isValidView(view View) bool {
	switch view {
	case ViewDashboard, ViewDrinks, ViewIngredients, ViewInventory, ViewMenus, ViewOrders, ViewAudit, ViewTags:
		return true
	default:
		return false
	}
}

func viewTitle(view View) string {
	switch view {
	case ViewDashboard:
		return "Dashboard"
	case ViewDrinks:
		return "Drinks"
	case ViewIngredients:
		return "Ingredients"
	case ViewInventory:
		return "Inventory"
	case ViewMenus:
		return "Menus"
	case ViewOrders:
		return "Orders"
	case ViewAudit:
		return "Audit"
	case ViewTags:
		return "Tags"
	default:
		return "Unknown"
	}
}
