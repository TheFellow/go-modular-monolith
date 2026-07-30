package main

import (
	"fmt"
	"strconv"
	"sync"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	application "github.com/TheFellow/go-modular-monolith/app"
	fyneui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
)

const dashboardRecentMax = application.DashboardRecentLimit

type dashboardData = application.DashboardAggregate
type dashboardLoader interface {
	LoadDashboard() (dashboardData, error)
}

type sessionDashboardLoader struct{ session *application.Session }

func (l sessionDashboardLoader) LoadDashboard() (dashboardData, error) {
	return l.session.Dashboard()
}

func unknownDashboardData() dashboardData {
	return application.UnknownDashboardAggregate()
}

type dashboardState struct {
	Status fyneui.LoadStatus
	Data   dashboardData
	Err    error
}

// dashboardViewModel is the shell-level presentation model. It exposes no
// widgets, and its executor and dispatcher make every transition deterministic
// in tests while production work remains off Fyne's UI goroutine.
type dashboardViewModel struct {
	loader  dashboardLoader
	request *fyneui.LatestRequest[dashboardLoadResult]

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
	data dashboardData
	err  error
}

func newDashboardViewModel(loader dashboardLoader, executor fyneui.Executor, dispatcher fyneui.Dispatcher) *dashboardViewModel {
	return &dashboardViewModel{
		loader: loader, request: fyneui.NewLatestRequest[dashboardLoadResult](executor, dispatcher),
		state: dashboardState{Status: fyneui.Idle, Data: unknownDashboardData()},
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
	m.request.Load(func() (dashboardLoadResult, error) {
		defer m.work.Done()
		data, err := m.loader.LoadDashboard()
		return dashboardLoadResult{data: data, err: err}, nil
	}, m.publish)
}

func (m *dashboardViewModel) publish(result fyneui.LoadState[dashboardLoadResult]) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	state := dashboardState{Status: result.Status, Data: result.Value.data, Err: result.Value.err}
	if result.Err != nil {
		state.Err = result.Err
	}
	if result.Status == fyneui.Loading {
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
}

type dashboardCard struct{ route, title, description string }

var dashboardCards = []dashboardCard{
	{"drinks", "Drinks", "Manage drink recipes"},
	{"ingredients", "Ingredients", "Catalog ingredients"},
	{"inventory", "Inventory", "Track stock levels"},
	{"menus", "Menus", "Build drink menus"},
	{"orders", "Orders", "Review orders"},
	{"audit", "Audit", "Inspect audit logs"},
	{"tags", "Tags", "Tag any entity"},
}

func newDashboardView(model *dashboardViewModel, navigate func(string) error) *dashboardView {
	v := &dashboardView{
		model: model, status: widget.NewLabel(""),
		cards: container.NewAdaptiveGrid(3), activity: container.NewVBox(), navigate: navigate,
	}
	refresh := fyneui.NewButton("dashboard-refresh", "Refresh", model.Refresh)
	header := container.NewBorder(nil, nil,
		widget.NewLabel("Overview of the Mixology application"), refresh,
	)
	v.content = container.NewBorder(
		container.NewVBox(header, v.status), nil, nil, nil,
		container.NewVScroll(container.NewVBox(v.cards, widget.NewSeparator(),
			widget.NewLabelWithStyle("Recent Activity", framework.TextAlignLeading, framework.TextStyle{Bold: true}),
			v.activity)),
	)
	model.Observe(v.render)
	return v
}

func (v *dashboardView) Title() string                   { return "Dashboard" }
func (v *dashboardView) Content() framework.CanvasObject { return v.content }
func (v *dashboardView) Activate()                       { v.model.Refresh() }

func (v *dashboardView) render(state dashboardState) {
	switch state.Status {
	case fyneui.Idle:
		v.status.SetText("Dashboard has not been loaded")
	case fyneui.Loading:
		v.status.SetText("Loading dashboard…")
	case fyneui.Failed:
		v.status.SetText("Dashboard could not be loaded: " + state.Err.Error())
	case fyneui.Loaded:
		if state.Err != nil {
			v.status.SetText("Some dashboard information is unavailable: " + state.Err.Error())
		} else {
			v.status.SetText("Dashboard is up to date")
		}
	}

	v.cards.RemoveAll()
	for _, definition := range dashboardCards {
		count, detail := dashboardCardText(definition.route, state.Data)
		button := fyneui.NewButton("dashboard-open-"+definition.route, "Open", func() { _ = v.navigate(definition.route) })
		v.cards.Add(widget.NewCard(definition.title+"  "+count, detail, button))
	}
	v.activity.RemoveAll()
	if len(state.Data.RecentActivity) == 0 {
		message := "No recent activity"
		if state.Status == fyneui.Loading {
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

func dashboardCardText(route string, data dashboardData) (string, string) {
	switch route {
	case "drinks":
		return formatDashboardCount(data.DrinkCount), "Manage drink recipes"
	case "ingredients":
		return formatDashboardCount(data.IngredientCount), "Catalog ingredients"
	case "inventory":
		return formatDashboardCount(data.InventoryCount), "Low stock: " + formatDashboardCount(data.LowStockCount)
	case "menus":
		return formatDashboardCount(data.MenuCount), fmt.Sprintf("Draft %s • Published %s", formatDashboardCount(data.DraftMenus), formatDashboardCount(data.PublishedMenus))
	case "orders":
		return formatDashboardCount(data.OrderCount), "Pending: " + formatDashboardCount(data.PendingOrders)
	case "audit":
		return formatDashboardCount(data.AuditCount), "Inspect audit logs"
	default:
		return "", "Tag any entity"
	}
}

func formatDashboardCount(count int) string {
	if count < 0 {
		return "?"
	}
	return strconv.Itoa(count)
}
