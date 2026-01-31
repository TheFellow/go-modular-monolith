# Task 004: Dashboard Enhancement with Live Data

## Goal

Update Dashboard to display live counts and recent activity from the application layer.

## Files to Modify

- `main/tui/views/dashboard.go` - Add app access, async loading, counts display
- `main/tui/app.go` - Pass app to Dashboard constructor

## Pattern Reference

Follow the async loading pattern from the sprint-002 plan's ListViewModel example.

## Current State

Dashboard currently shows static navigation cards without any data:

```go
// main/tui/views/dashboard.go
type Dashboard struct {
    styles DashboardStyles
    keys   DashboardKeys
    width  int
    height int
}
```

## Implementation

### 1. Update Dashboard struct

```go
type Dashboard struct {
    app    *app.App  // NEW: application access
    styles DashboardStyles
    keys   DashboardKeys
    width  int
    height int

    // Data state
    loading bool
    data    *DashboardData
    err     error
}

type DashboardData struct {
    DrinkCount      int
    IngredientCount int
    MenuCount       int
    DraftMenus      int
    PublishedMenus  int
    LowStockCount   int
    PendingOrders   int
    RecentActivity  []AuditSummary
}

type AuditSummary struct {
    Timestamp string
    Actor     string
    Action    string
}
```

### 2. Add async data loading

```go
type DashboardLoadedMsg struct {
    Data *DashboardData
}

func (d *Dashboard) Init() tea.Cmd {
    return d.loadData()
}

func (d *Dashboard) loadData() tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        data := &DashboardData{}

        // Load counts from each domain
        if drinks, err := d.app.Drinks.List(ctx); err == nil {
            data.DrinkCount = len(drinks)
        }
        if ingredients, err := d.app.Ingredients.List(ctx); err == nil {
            data.IngredientCount = len(ingredients)
        }
        // ... similar for menus, inventory, orders

        // Load recent audit entries
        if entries, err := d.app.Audit.List(ctx); err == nil {
            for i, e := range entries {
                if i >= 10 { break }
                data.RecentActivity = append(data.RecentActivity, AuditSummary{
                    Timestamp: e.Timestamp.Format("15:04"),
                    Actor:     e.Actor,
                    Action:    e.Action,
                })
            }
        }

        return DashboardLoadedMsg{Data: data}
    }
}
```

### 3. Update View to show counts

Replace static cards with data-driven cards showing counts:

```go
func (d *Dashboard) View() string {
    if d.loading {
        return d.renderLoading()
    }

    header := d.styles.Title.Render("Dashboard")

    cards := d.renderCountCards()
    activity := d.renderRecentActivity()

    return lipgloss.JoinVertical(lipgloss.Left,
        header,
        cards,
        "",
        d.styles.Subtitle.Render("Recent Activity"),
        activity,
    )
}

func (d *Dashboard) renderCountCards() string {
    // Show counts in each navigation card
    // e.g., "[1] Drinks (42)"
}
```

### 4. Update App to pass app to Dashboard

```go
// main/tui/app.go - in currentViewModel()
case ViewDashboard:
    vm = views.NewDashboard(a.app, a.dashboardStyles(), a.dashboardKeys())
```

Update `NewDashboard` signature:
```go
func NewDashboard(app *app.App, styles DashboardStyles, keys DashboardKeys) *Dashboard
```

## Notes

- Dashboard now requires app access for data queries
- Loading is async to keep UI responsive
- Counts update on Init() - could add refresh support later
- Recent activity shows last 10 audit entries
- Error handling: show counts as "?" if query fails

## Checklist

- [ ] Add app field to Dashboard struct
- [ ] Add DashboardData and DashboardLoadedMsg types
- [ ] Implement loadData() command
- [ ] Handle DashboardLoadedMsg in Update()
- [ ] Update View() to show counts in cards
- [ ] Add recent activity section
- [ ] Update NewDashboard to accept *app.App
- [ ] Update App.currentViewModel() to pass app
- [ ] `go build ./main/tui/...` passes
- [ ] `go test ./...` passes
