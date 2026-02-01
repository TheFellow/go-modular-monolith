# Task 005: Implement ViewModel Registry

## Goal

Replace the large switch statement in `main/tui/app.go` with a registry pattern where domains self-register their ViewModel factories.

## Files to Create

- `pkg/tui/registry.go`
- `pkg/tui/views.go`
- `app/domains/drinks/surfaces/tui/register.go`
- `app/domains/ingredients/surfaces/tui/register.go`
- `app/domains/inventory/surfaces/tui/register.go`
- `app/domains/menus/surfaces/tui/register.go`
- `app/domains/orders/surfaces/tui/register.go`
- `app/domains/audit/surfaces/tui/register.go`

## Files to Modify

- `main/tui/app.go`
- `main/tui/views.go`

## Current State

`main/tui/app.go` imports every domain TUI package and has a switch statement:

```go
import (
    drinksui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/tui"
    ingredientsui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/tui"
    // ... all domains
)

func (a *App) currentViewModel() views.ViewModel {
    switch a.currentView {
    case ViewDrinks:
        vm = drinksui.NewListViewModel(...)
    case ViewIngredients:
        vm = ingredientsui.NewListViewModel(...)
    // ... all domains
    }
}
```

## Implementation

### 1. Create View constants in `pkg/tui/views.go`:

```go
package tui

type View int

const (
    ViewDashboard View = iota
    ViewDrinks
    ViewIngredients
    ViewInventory
    ViewMenus
    ViewOrders
    ViewAudit
)
```

### 2. Create registry in `pkg/tui/registry.go`:

```go
package tui

import (
    "github.com/TheFellow/go-modular-monolith/app"
    "github.com/TheFellow/go-modular-monolith/main/tui/views"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

// ViewModelFactory creates a ViewModel for a view.
type ViewModelFactory func(
    ctx *middleware.Context,
    application *app.App,
    styles ListViewStyles,
    keys ListViewKeys,
) views.ViewModel

var registry = make(map[View]ViewModelFactory)

// Register adds a ViewModel factory to the registry.
func Register(view View, factory ViewModelFactory) {
    registry[view] = factory
}

// Create instantiates a ViewModel from the registry.
func Create(
    view View,
    ctx *middleware.Context,
    application *app.App,
    styles ListViewStyles,
    keys ListViewKeys,
) views.ViewModel {
    if factory, ok := registry[view]; ok {
        return factory(ctx, application, styles, keys)
    }
    return nil
}
```

### 3. Create register.go in each domain:

```go
// app/domains/drinks/surfaces/tui/register.go
package tui

import (
    "github.com/TheFellow/go-modular-monolith/app"
    "github.com/TheFellow/go-modular-monolith/main/tui/views"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
    pkgtui "github.com/TheFellow/go-modular-monolith/pkg/tui"
)

func init() {
    pkgtui.Register(pkgtui.ViewDrinks, func(
        ctx *middleware.Context,
        application *app.App,
        styles pkgtui.ListViewStyles,
        keys pkgtui.ListViewKeys,
    ) views.ViewModel {
        return NewListViewModel(application, ctx, styles, keys)
    })
}
```

### 4. Update `main/tui/app.go`:

```go
func (a *App) currentViewModel() views.ViewModel {
    // ...
    vm := pkgtui.Create(
        pkgtui.View(a.currentView),
        a.ctx,
        a.app,
        ListViewStylesFrom(a.styles),
        ListViewKeysFrom(a.keys),
    )
    if vm == nil {
        // Fallback for dashboard or unknown views
        vm = views.NewDashboard(a.app, a.ctx, a.dashboardStyles(), a.dashboardKeys())
    }
    // ...
}
```

### 5. Ensure registration via imports:

In `main/tui/` or `main/main.go`, add blank imports to trigger init():

```go
import (
    _ "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/tui"
    _ "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/tui"
    // ... etc
)
```

## Notes

- Dashboard has different styles/keys and may remain special-cased
- The registry uses `init()` for self-registration
- This pattern allows adding new domains without modifying `app.go`
- Consider if `views.ViewModel` interface should move to `pkg/tui` to avoid import cycles

## Checklist

- [ ] Create `pkg/tui/views.go` with View constants
- [ ] Create `pkg/tui/registry.go` with Register/Create
- [ ] Create `register.go` for drinks
- [ ] Create `register.go` for ingredients
- [ ] Create `register.go` for inventory
- [ ] Create `register.go` for menus
- [ ] Create `register.go` for orders
- [ ] Create `register.go` for audit
- [ ] Update `main/tui/app.go` to use registry
- [ ] Remove domain TUI imports from `app.go`
- [ ] Add blank imports to trigger registration
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
