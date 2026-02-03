# Task 002: ViewModel Architecture Update

## Goal

Change TUI to store principal instead of context, simplify ViewModel constructors, and create fresh context per command/query.

## Files to Modify

```
main/cli/cli.go                              # Pass principal to tui.Run
main/tui/main.go                             # Update Run signature
main/tui/app.go                              # Store principal, update currentViewModel
main/tui/views/dashboard.go                  # Update constructor, fresh context
app/domains/drinks/surfaces/tui/
├── list_vm.go                               # Simplify constructor, fresh context
├── create_vm.go                             # Update deps, fresh context
└── edit_vm.go                               # Update deps, fresh context
app/domains/ingredients/surfaces/tui/
├── list_vm.go
├── create_vm.go
└── edit_vm.go
app/domains/inventory/surfaces/tui/
├── list_vm.go
├── adjust_vm.go
└── set_vm.go
app/domains/menus/surfaces/tui/
├── list_vm.go
├── create_vm.go
└── rename_vm.go
app/domains/orders/surfaces/tui/
└── list_vm.go
app/domains/audit/surfaces/tui/
└── list_vm.go
```

## Implementation

### 1. Update `main/cli/cli.go`

Pass principal to tui.Run instead of full context:

```go
// Before
if err := tui.Run(mctx, c.app, initialView); err != nil {

// After
if err := tui.Run(p, c.app, initialView); err != nil {
```

Where `p` is the `cedar.EntityUID` parsed from `c.actor`.

### 2. Update `main/tui/main.go`

```go
import "github.com/cedar-policy/cedar-go"

// Run starts the TUI with the given application and principal.
func Run(principal cedar.EntityUID, application *app.App, initialView View) error {
    model := NewApp(principal, application, initialView)
    p := tea.NewProgram(model, tea.WithAltScreen())
    _, err := p.Run()
    return err
}
```

### 3. Update `main/tui/app.go`

```go
import "github.com/cedar-policy/cedar-go"

type App struct {
    currentView View
    prevViews   []View

    app       *app.App
    principal cedar.EntityUID  // Changed from *middleware.Context

    styles    Styles
    keys      KeyMap
    // ...
}

func NewApp(principal cedar.EntityUID, application *app.App, initialView View) *App {
    // ...
    return &App{
        currentView: initialView,
        app:         application,
        principal:   principal,
        styles:      appStyles,
        keys:        appKeys,
        // ...
    }
}

func (a *App) currentViewModel() views.ViewModel {
    // ...
    switch a.currentView {
    case ViewDashboard:
        vm = views.NewDashboard(a.app, a.principal)
    case ViewDrinks:
        vm = drinksui.NewListViewModel(a.app, a.principal)
    case ViewIngredients:
        vm = ingredientsui.NewListViewModel(a.app, a.principal)
    case ViewInventory:
        vm = inventoryui.NewListViewModel(a.app, a.principal)
    case ViewMenus:
        vm = menusui.NewListViewModel(a.app, a.principal)
    case ViewOrders:
        vm = ordersui.NewListViewModel(a.app, a.principal)
    case ViewAudit:
        vm = auditui.NewListViewModel(a.app, a.principal)
    // ...
    }
}
```

### 4. Update ViewModel Pattern (example: drinks)

```go
// app/domains/drinks/surfaces/tui/list_vm.go
package tui

import (
    "context"

    maintui "github.com/TheFellow/go-modular-monolith/main/tui"
    "github.com/cedar-policy/cedar-go"
    // ...
)

type ListViewModel struct {
    app       *app.App
    principal cedar.EntityUID

    styles     tui.ListViewStyles
    keys       tui.ListViewKeys
    formStyles forms.FormStyles
    formKeys   forms.FormKeys
    dialogStyles dialog.DialogStyles
    dialogKeys   dialog.DialogKeys

    // ... rest unchanged
}

// Before: 8 parameters
// After: 2 parameters
func NewListViewModel(app *app.App, principal cedar.EntityUID) *ListViewModel {
    // ... list setup unchanged ...

    return &ListViewModel{
        app:          app,
        principal:    principal,
        styles:       maintui.ListViewStyles,   // Import from main/tui
        keys:         maintui.ListViewKeys,
        formStyles:   maintui.FormStyles,
        formKeys:     maintui.FormKeys,
        dialogStyles: maintui.DialogStyles,
        dialogKeys:   maintui.DialogKeys,
        // ... rest unchanged
    }
}

// Fresh context per command/query
func (m *ListViewModel) context() *middleware.Context {
    return m.app.Context(context.Background(), m.principal)
}

func (m *ListViewModel) loadDrinks() tea.Cmd {
    return func() tea.Msg {
        // Before: m.ctx
        // After: m.context() - fresh each time
        drinksList, err := m.drinksQueries.List(m.context(), queries.ListFilter{})
        // ...
    }
}
```

### 5. Update child ViewModels

Child ViewModels (CreateDrinkVM, EditDrinkVM, etc.) receive app and principal from parent:

```go
// In list_vm.go
func (m *ListViewModel) startCreate() tea.Cmd {
    m.create = NewCreateDrinkVM(m.app, m.principal, m.loadIngredients())
    return m.create.Init()
}

// In create_vm.go
func NewCreateDrinkVM(app *app.App, principal cedar.EntityUID, ingredients []*ingredientsmodels.Ingredient) *CreateDrinkVM {
    return &CreateDrinkVM{
        app:        app,
        principal:  principal,
        styles:     maintui.FormStyles,
        keys:       maintui.FormKeys,
        // ...
    }
}
```

### 6. Update Dashboard

```go
// main/tui/views/dashboard.go
func NewDashboard(app *app.App, principal cedar.EntityUID) *Dashboard {
    return &Dashboard{
        app:       app,
        principal: principal,
        styles:    dashboardStylesFrom(tui.AppStyles()),
        keys:      dashboardKeysFrom(tui.AppKeys()),
        // ...
    }
}

func (d *Dashboard) context() *middleware.Context {
    return d.app.Context(context.Background(), d.principal)
}
```

## Notes

- This is a large atomic change - all ViewModels must be updated together
- The `m.context()` helper creates fresh context for each call
- Child ViewModels receive app and principal from parent, not from constructor params
- Import alias `maintui` avoids conflict with `pkg/tui`

## Checklist

### Core Infrastructure
- [ ] Update `main/cli/cli.go` to pass principal to tui.Run
- [ ] Update `main/tui/main.go` Run signature
- [ ] Update `main/tui/app.go` to store principal, update currentViewModel

### ViewModels
- [ ] Update `main/tui/views/dashboard.go`
- [ ] Update `app/domains/drinks/surfaces/tui/` (list_vm, create_vm, edit_vm)
- [ ] Update `app/domains/ingredients/surfaces/tui/` (list_vm, create_vm, edit_vm)
- [ ] Update `app/domains/inventory/surfaces/tui/` (list_vm, adjust_vm, set_vm)
- [ ] Update `app/domains/menus/surfaces/tui/` (list_vm, create_vm, rename_vm)
- [ ] Update `app/domains/orders/surfaces/tui/` (list_vm)
- [ ] Update `app/domains/audit/surfaces/tui/` (list_vm)

### Fresh Context
- [ ] Add `context()` helper to each ViewModel
- [ ] Replace all `m.ctx` usages with `m.context()`

### Verification
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Manual test: TUI session with multiple operations, verify no log accumulation
