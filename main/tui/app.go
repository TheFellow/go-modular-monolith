package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
)

// App is the root model for the TUI application.
type App struct {
	// Navigation
	currentView View
	prevViews   []View

	// Application layer
	app *app.App

	// UI State
	styles    Styles
	keys      KeyMap
	help      help.Model
	width     int
	height    int
	showHelp  bool
	lastError error

	// Child views (lazy initialized)
	views map[View]views.ViewModel
}

// NewApp creates a new App with the given application and initial view.
func NewApp(application *app.App, initialView View) *App {
	if !isValidView(initialView) {
		initialView = ViewDashboard
	}

	helpModel := help.New()
	helpModel.ShowAll = false

	return &App{
		currentView: initialView,
		app:         application,
		styles:      NewStyles(),
		keys:        NewKeyMap(),
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
		if key.Matches(msg, a.keys.Quit) {
			return a, tea.Quit
		}
		if key.Matches(msg, a.keys.Help) {
			a.showHelp = !a.showHelp
			return a, nil
		}
		if key.Matches(msg, a.keys.Back) {
			return a, a.navigateBack()
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.help.Width = msg.Width

	case NavigateMsg:
		return a, a.navigateTo(msg.To)

	case ErrorMsg:
		a.lastError = msg.Err
		return a, nil
	}

	vm, cmd := a.currentViewModel().Update(msg)
	a.views[a.currentView] = vm
	return a, cmd
}

// View implements tea.Model.
func (a *App) View() string {
	content := a.currentViewModel().View()
	status := a.statusBarView()

	parts := []string{content, status}
	if a.showHelp {
		a.help.ShowAll = true
		parts = append(parts, a.help.View(a.currentViewModel()))
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
		vm = views.NewDashboard(a.dashboardStyles(), a.dashboardKeys())
	case ViewDrinks, ViewIngredients, ViewInventory, ViewMenus, ViewOrders, ViewAudit:
		vm = views.NewPlaceholder(viewTitle(a.currentView))
	default:
		a.currentView = ViewDashboard
		vm = views.NewDashboard(a.dashboardStyles(), a.dashboardKeys())
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

	if _, ok := a.views[target]; ok {
		return a.syncWindowCmd()
	}

	initCmd := a.currentViewModel().Init()
	return tea.Batch(initCmd, a.syncWindowCmd())
}

// navigateBack pops the previous view from the stack.
func (a *App) navigateBack() tea.Cmd {
	if len(a.prevViews) == 0 {
		if a.currentView != ViewDashboard {
			a.currentView = ViewDashboard
			return a.syncWindowCmd()
		}
		return nil
	}

	idx := len(a.prevViews) - 1
	a.currentView = a.prevViews[idx]
	a.prevViews = a.prevViews[:idx]
	return nil
}

func (a *App) syncWindowCmd() tea.Cmd {
	if a.width == 0 && a.height == 0 {
		return nil
	}

	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: a.width, Height: a.height}
	}
}

func (a *App) statusBarView() string {
	var content string
	if a.lastError != nil {
		content = a.styles.ErrorText.Render("Error: " + a.lastError.Error())
	} else {
		content = a.styles.HelpDesc.Render("View: " + viewTitle(a.currentView) + "  •  Press ? for help")
	}

	style := a.styles.StatusBar
	if a.width > 0 {
		style = style.Width(a.width)
	}

	return style.Render(content)
}

func (a *App) dashboardStyles() views.DashboardStyles {
	return views.DashboardStyles{
		Title:    a.styles.Title,
		Subtitle: a.styles.Subtitle,
		Card:     a.styles.Card,
		HelpKey:  a.styles.HelpKey,
	}
}

func (a *App) dashboardKeys() views.DashboardKeys {
	return views.DashboardKeys{
		Nav1: a.keys.Nav1,
		Nav2: a.keys.Nav2,
		Nav3: a.keys.Nav3,
		Nav4: a.keys.Nav4,
		Nav5: a.keys.Nav5,
		Nav6: a.keys.Nav6,
		Help: a.keys.Help,
		Quit: a.keys.Quit,
	}
}

func isValidView(view View) bool {
	switch view {
	case ViewDashboard, ViewDrinks, ViewIngredients, ViewInventory, ViewMenus, ViewOrders, ViewAudit:
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
	default:
		return "Unknown"
	}
}
