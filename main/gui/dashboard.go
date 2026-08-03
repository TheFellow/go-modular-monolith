package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	application "github.com/TheFellow/go-modular-monolith/app"
	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const (
	dashboardRecentMax         = application.DashboardRecentLimit
	controlDashboardRefresh    = "dashboard-refresh"
	controlDashboardOpenPrefix = "dashboard-open-"
)

type dashboardLoader interface {
	LoadDashboard(context.Context) (application.Dashboard, error)
}

type sessionDashboardLoader struct{ session *application.Session }

func (l sessionDashboardLoader) LoadDashboard(ctx context.Context) (application.Dashboard, error) {
	return l.session.DashboardContext(ctx)
}

func unknownDashboardData() application.Dashboard {
	return application.UnknownDashboard()
}

type dashboardState struct {
	Status gui.LoadStatus
	Data   application.Dashboard
	Err    error
}

// dashboardViewModel is the shell-level presentation model. It exposes no
// widgets, and its executor and dispatcher make every transition deterministic
// in tests while production work remains off Fyne's UI goroutine.
type dashboardViewModel struct {
	loader  dashboardLoader
	request *gui.LatestRequest[dashboardLoadResult]

	mu      sync.RWMutex
	work    sync.WaitGroup
	state   dashboardState
	changed func(dashboardState)
	closed  bool
	// beforeCloseWait is a deterministic test seam for observing the point at
	// which Close has invalidated publication and is about to await loader work.
	beforeCloseWait func()
}

type dashboardLoadResult struct {
	data application.Dashboard
	err  error
}

func newDashboardViewModel(loader dashboardLoader, executor gui.Executor, dispatcher gui.Dispatcher) *dashboardViewModel {
	return &dashboardViewModel{
		loader: loader, request: gui.NewLatestRequest[dashboardLoadResult](executor, dispatcher),
		state: dashboardState{Status: gui.Idle, Data: unknownDashboardData()},
	}
}

func (m *dashboardViewModel) Observe(changed func(dashboardState)) {
	m.mu.Lock()
	m.changed = changed
	state := m.state
	m.mu.Unlock()
	if changed != nil {
		changed(state)
	}
}

func (m *dashboardViewModel) Refresh() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.work.Add(1)
	m.mu.Unlock()
	m.request.LoadContext(context.Background(), func(ctx context.Context) (dashboardLoadResult, error) {
		defer m.work.Done()
		data, err := m.loader.LoadDashboard(ctx)
		return dashboardLoadResult{data: data, err: err}, nil
	}, m.publish)
}

func (m *dashboardViewModel) publish(result gui.LoadState[dashboardLoadResult]) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	state := dashboardState{Status: result.Status, Data: result.Value.data, Err: result.Value.err}
	if result.Err != nil {
		state.Err = result.Err
	}
	if result.Status == gui.Loading {
		state.Data = m.state.Data
	}
	m.state = state
	changed := m.changed
	m.mu.Unlock()
	if changed != nil {
		changed(state)
	}
}

func (m *dashboardViewModel) Snapshot() dashboardState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *dashboardViewModel) Close() {
	m.request.Invalidate()
	m.mu.Lock()
	m.closed = true
	m.changed = nil
	beforeWait := m.beforeCloseWait
	m.mu.Unlock()
	if beforeWait != nil {
		beforeWait()
	}
	m.work.Wait()
}

type dashboardView struct {
	model    *dashboardViewModel
	content  framework.CanvasObject
	status   *widget.Label
	cards    *framework.Container
	activity *framework.Container
	navigate func(string) error
	visible  map[workspace]bool
}

type dashboardCard struct {
	route              workspace
	title, description string
}

var dashboardCards = []dashboardCard{
	{workspaceDrinks, "Drinks", "Manage drink recipes"},
	{workspaceIngredients, "Ingredients", "Catalog ingredients"},
	{workspaceInventory, "Inventory", "Track stock levels"},
	{workspaceMenus, "Menus", "Build drink menus"},
	{workspaceOrders, "Orders", "Review orders"},
	{workspaceAudit, "Audit", "Inspect audit logs"},
	{workspaceTags, "Tags", "Tag any entity"},
}

func newDashboardView(model *dashboardViewModel, navigate func(string) error, visible ...map[workspace]bool) *dashboardView {
	v := &dashboardView{
		model: model, status: widget.NewLabel(""),
		cards: container.NewAdaptiveGrid(3), activity: container.NewVBox(), navigate: navigate,
	}
	if len(visible) != 0 {
		v.visible = visible[0]
	}
	refresh := gui.NewButton(controlDashboardRefresh, "Refresh", model.Refresh)
	v.content = gui.StandardPage(
		"Dashboard", "Overview of the Mixology application", []framework.CanvasObject{refresh},
		container.NewVScroll(container.NewVBox(v.cards, widget.NewSeparator(),
			widget.NewLabelWithStyle("Recent Activity", framework.TextAlignLeading, framework.TextStyle{Bold: true}),
			v.activity)), v.status,
	).(*framework.Container)
	model.Observe(v.render)
	return v
}

func (v *dashboardView) Title() string                   { return "Dashboard" }
func (v *dashboardView) Content() framework.CanvasObject { return v.content }
func (v *dashboardView) Activate()                       { v.model.Refresh() }

func (v *dashboardView) render(state dashboardState) {
	switch state.Status {
	case gui.Idle:
		v.status.SetText("Dashboard has not been loaded")
	case gui.Loading:
		v.status.SetText("Loading dashboard…")
	case gui.Failed:
		v.status.SetText("Dashboard could not be loaded: " + state.Err.Error())
	case gui.Loaded:
		if state.Err != nil {
			v.status.SetText("Some dashboard information is unavailable: " + state.Err.Error())
		} else {
			v.status.SetText("Dashboard is up to date")
		}
	}

	v.cards.RemoveAll()
	for _, definition := range dashboardCards {
		if v.visible != nil && !v.visible[definition.route] {
			continue
		}
		count, detail := dashboardCardText(definition.route, state.Data)
		button := gui.NewButton(controlDashboardOpenPrefix+definition.route.routeID(), "Open", func() { _ = v.navigate(definition.route.routeID()) })
		v.cards.Add(widget.NewCard(definition.title+"  "+count, detail, button))
	}
	v.activity.RemoveAll()
	if len(state.Data.RecentActivity) == 0 {
		message := "No recent activity"
		if state.Status == gui.Loading {
			message = "Loading recent activity…"
		}
		v.activity.Add(widget.NewLabel(message))
	} else {
		for _, item := range state.Data.RecentActivity {
			v.activity.Add(widget.NewLabel(fmt.Sprintf("%s   %s   %s",
				item.Timestamp.Format("2006-01-02 15:04"), item.Actor, item.Action)))
		}
	}
	v.cards.Refresh()
	v.activity.Refresh()
}

func dashboardCardText(route workspace, data application.Dashboard) (string, string) {
	switch route {
	case workspaceDashboard:
		return "", "Overview of the Mixology application"
	case workspaceDrinks:
		return formatDashboardCount(data.DrinkCount), "Manage drink recipes"
	case workspaceIngredients:
		return formatDashboardCount(data.IngredientCount), "Catalog ingredients"
	case workspaceInventory:
		return formatDashboardCount(data.InventoryCount), "Low stock: " + formatDashboardCount(data.LowStockCount)
	case workspaceMenus:
		return formatDashboardCount(data.MenuCount), fmt.Sprintf("Draft %s • Published %s", formatDashboardCount(data.DraftMenus), formatDashboardCount(data.PublishedMenus))
	case workspaceOrders:
		return formatDashboardCount(data.OrderCount), "Pending: " + formatDashboardCount(data.PendingOrders)
	case workspaceAudit:
		return formatDashboardCount(data.AuditCount), "Inspect audit logs"
	case workspaceTags:
		return "", "Tag any entity"
	}
	return "", ""
}

func formatDashboardCount(count int) string {
	if count < 0 {
		return "?"
	}
	return strconv.Itoa(count)
}
