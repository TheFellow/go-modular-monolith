# Task 003: Remove app.go Boilerplate

## Goal

Remove the 12 domain-specific style/key mapping methods from `main/tui/app.go` and use the shared `ListViewStylesFrom()` and `ListViewKeysFrom()` functions.

## Files to Modify

- `main/tui/app.go`
- `main/tui/viewmodel_types.go`

## Current State

`main/tui/app.go` has 12 nearly identical methods (lines 306-442):

```go
func (a *App) drinksListStyles() drinksui.ListViewStyles { ... }
func (a *App) drinksListKeys() drinksui.ListViewKeys { ... }
func (a *App) ingredientsListStyles() ingredientsui.ListViewStyles { ... }
func (a *App) ingredientsListKeys() ingredientsui.ListViewKeys { ... }
func (a *App) inventoryListStyles() inventoryui.ListViewStyles { ... }
func (a *App) inventoryListKeys() inventoryui.ListViewKeys { ... }
func (a *App) menusListStyles() menusui.ListViewStyles { ... }
func (a *App) menusListKeys() menusui.ListViewKeys { ... }
func (a *App) ordersListStyles() ordersui.ListViewStyles { ... }
func (a *App) ordersListKeys() ordersui.ListViewKeys { ... }
func (a *App) auditListStyles() auditui.ListViewStyles { ... }
func (a *App) auditListKeys() auditui.ListViewKeys { ... }
```

## Implementation

1. Update `main/tui/viewmodel_types.go`:
   - Change `ListViewStylesFrom()` to return `pkgtui.ListViewStyles`
   - Change `ListViewKeysFrom()` to return `pkgtui.ListViewKeys`
   - Remove local type definitions (now in `pkg/tui`)

2. Update `main/tui/app.go`:
   - Delete all `*ListStyles()` and `*ListKeys()` methods for domains
   - Update `currentViewModel()` to use `ListViewStylesFrom(a.styles)` and `ListViewKeysFrom(a.keys)` directly

Before:
```go
case ViewDrinks:
    vm = drinksui.NewListViewModel(a.app, a.ctx, a.drinksListStyles(), a.drinksListKeys())
```

After:
```go
case ViewDrinks:
    vm = drinksui.NewListViewModel(a.app, a.ctx, ListViewStylesFrom(a.styles), ListViewKeysFrom(a.keys))
```

## Notes

- Keep `dashboardStyles()` and `dashboardKeys()` - dashboard has different types
- The `ListViewStylesFrom` and `ListViewKeysFrom` functions remain in `main/tui` because they convert from `main/tui.Styles` and `main/tui.KeyMap`
- This removes ~130 lines of boilerplate

## Checklist

- [ ] Update `viewmodel_types.go` to use `pkg/tui` types
- [ ] Delete domain-specific style/key methods from `app.go`
- [ ] Update `currentViewModel()` to use shared conversion functions
- [ ] `go build ./main/tui/...` passes
- [ ] `go test ./...` passes
