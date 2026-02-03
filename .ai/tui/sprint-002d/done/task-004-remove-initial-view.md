# Task 004: Remove initialView + Add Title Bar

## Goal

Simplify the TUI API by always starting at the dashboard, and add a title bar so users know where they are.

## Files to Modify

```
main/tui/main.go      # Remove initialView param from Run()
main/tui/app.go       # Remove initialView param from NewApp(), add title bar to View()
main/tui/styles.go    # Add TitleBar style
main/cli/cli.go       # Remove initialView parsing and passing
```

## Implementation

### 1. Update `main/tui/main.go`

```go
// Before
func Run(principal cedar.EntityUID, application *app.App, initialView View) error {
    model := NewApp(principal, application, initialView)
    // ...
}

// After
func Run(principal cedar.EntityUID, application *app.App) error {
    model := NewApp(principal, application)
    // ...
}
```

### 2. Update `main/tui/app.go`

```go
// Before
func NewApp(principal cedar.EntityUID, application *app.App, initialView View) *App {
    if !isValidView(initialView) {
        initialView = ViewDashboard
    }
    return &App{
        currentView: initialView,
        // ...
    }
}

// After
func NewApp(principal cedar.EntityUID, application *app.App) *App {
    return &App{
        currentView: ViewDashboard,  // Always start here
        // ...
    }
}
```

### 3. Update `main/cli/cli.go`

Remove the initialView parsing logic:

```go
// Before
if cmd != nil && cmd.Bool("tui") {
    // ...
    initialView := tui.ViewDashboard
    args := cmd.Args().Slice()
    if len(args) > 0 {
        var ok bool
        initialView, ok = tui.ParseView(args[0])
        if !ok {
            return ctx, cli.Exit(fmt.Errorf("unknown view: %s", args[0]), apperrors.ExitUsage)
        }
    }
    if len(args) > 1 {
        return ctx, cli.Exit(fmt.Errorf("too many arguments for --tui"), apperrors.ExitUsage)
    }

    if err := tui.Run(p, c.app, initialView); err != nil {
        // ...
    }
}

// After
if cmd != nil && cmd.Bool("tui") {
    // ...
    if err := tui.Run(p, c.app); err != nil {
        // ...
    }
}
```

### 4. Remove ParseView if unused

Check if `tui.ParseView()` is used elsewhere. If not, remove it from `main/tui/views.go` or wherever it's defined.

### 5. Add Title Bar Style

**In `main/tui/styles.go`:**

```go
type Styles struct {
    // ...
    TitleBar lipgloss.Style  // NEW
}

func newStyles() Styles {
    // ...
    styles.TitleBar = lipgloss.NewStyle().
        Bold(true).
        Foreground(statusForeground).
        Background(styles.Primary).
        Padding(0, 1).
        MarginBottom(1)

    return styles
}
```

### 6. Add Title Bar to App.View()

**In `main/tui/app.go`:**

```go
func (a *App) View() string {
    if a.width > 0 && a.height > 0 && (a.width < MinWidth || a.height < MinHeight) {
        return a.renderTooSmallWarning()
    }

    // Title bar showing current view
    titleBar := a.titleBarView()

    content := a.currentViewModel().View()
    status := a.statusBarView()

    parts := []string{titleBar, content, status}
    if a.showHelp {
        // ...
    }

    return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (a *App) titleBarView() string {
    title := "Mixology › " + viewTitle(a.currentView)
    style := appStyles.TitleBar
    if a.width > 0 {
        style = style.Width(a.width)
    }
    return style.Render(title)
}
```

### 7. Update availableHeight()

Account for title bar height:

```go
const (
    // ...
    titleBarHeight  = 2  // 1 line + 1 margin
)

func (a *App) availableHeight() int {
    height := a.height - titleBarHeight - statusBarHeight - a.helpHeight()
    if height < 0 {
        return 0
    }
    return height
}
```

### 8. Fix Dashboard Refresh Bug

The dashboard's recent activity list doesn't refresh when returning to the view or when pressing `r`.

**Investigate `main/tui/views/dashboard.go`:**

1. Check if `Refresh` key binding is handled in `Update()`
2. Check if returning to dashboard triggers a data reload
3. The dashboard may be cached in `App.views` map and never re-initialized

**Likely fixes:**

```go
// In dashboard.go Update()
case tea.KeyMsg:
    if key.Matches(msg, d.keys.Refresh) {
        return d, d.loadData()  // Trigger refresh
    }
```

**For stale data on return**, either:
- Clear dashboard from `App.views` cache when navigating away, or
- Send a refresh command when navigating back to dashboard, or
- Don't cache the dashboard at all (always create fresh)

```go
// Option: In app.go navigateBack() or when switching to dashboard
if a.currentView == ViewDashboard {
    delete(a.views, ViewDashboard)  // Force re-init
}
```

## Notes

- The dashboard provides a natural entry point with navigation to all views
- Users can quickly navigate to any view from the dashboard
- Removes edge case handling for invalid view names
- Simplifies the API surface
- Title bar provides clear visual context of current location
- Title format: "Mixology › Dashboard", "Mixology › Drinks", etc.
- Dashboard must show fresh audit data on return and on refresh

## Checklist

### Remove initialView
- [x] Remove `initialView` param from `tui.Run()`
- [x] Remove `initialView` param from `NewApp()`
- [x] Always set `currentView: ViewDashboard` in NewApp
- [x] Remove view parsing logic from `main/cli/cli.go`
- [x] Remove `ParseView()` function if no longer used

### Add Title Bar
- [x] Add `TitleBar` style to `main/tui/styles.go`
- [x] Add `titleBarView()` method to App
- [x] Update `View()` to include title bar
- [x] Update `availableHeight()` to account for title bar
- [x] Add `titleBarHeight` constant

### Fix Dashboard Refresh
- [x] Handle `Refresh` key in dashboard `Update()` to reload data
- [x] Ensure fresh data when returning to dashboard (clear cache or send refresh)
- [ ] Verify recent activity list updates after CRUD operations

### Verification
- [x] `go build ./...` passes
- [x] `go test ./...` passes
- [x] Manual test: title bar shows correct view name when navigating
