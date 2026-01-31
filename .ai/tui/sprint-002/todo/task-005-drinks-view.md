# Task 005: Drinks View Implementation

## Goal

Create the Drinks domain ListViewModel and DetailViewModel, replacing the placeholder view.

## Files to Create/Modify

- `app/domains/drinks/surfaces/tui/messages.go` (new)
- `app/domains/drinks/surfaces/tui/list_vm.go` (new)
- `app/domains/drinks/surfaces/tui/detail_vm.go` (new)
- `app/domains/drinks/surfaces/tui/items.go` (new) - list.Item implementation
- `main/tui/app.go` - Wire DrinksListViewModel

## Pattern Reference

Follow the ListViewModel pattern from sprint-002 plan. Reference `app/domains/drinks/surfaces/cli/views.go` for field access patterns.

## Implementation

### 1. Create `app/domains/drinks/surfaces/tui/messages.go`

```go
package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"

// DrinksLoadedMsg is sent when drinks have been loaded
type DrinksLoadedMsg struct {
    Drinks []models.Drink
}
```

### 2. Create `app/domains/drinks/surfaces/tui/items.go`

```go
package tui

import (
    "fmt"
    "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
)

// drinkItem implements list.Item for drinks
type drinkItem struct {
    drink models.Drink
}

func (i drinkItem) Title() string       { return i.drink.Name }
func (i drinkItem) Description() string { return fmt.Sprintf("%s • %s", i.drink.Category, i.drink.Glass) }
func (i drinkItem) FilterValue() string { return i.drink.Name }
```

### 3. Create `app/domains/drinks/surfaces/tui/list_vm.go`

Implement ListViewModel following the pattern in sprint-002 plan:
- Use bubbles/list component
- Implement ViewModel interface (Init, Update, View, ShortHelp, FullHelp)
- Load drinks via app.Drinks.List()
- Embed DetailViewModel for right pane
- Handle selection changes to update detail view

Key methods:
- `NewListViewModel(app, styles, keys)` - constructor
- `Init()` - returns loadDrinks command
- `loadDrinks()` - async query
- `Update()` - handle DrinksLoadedMsg, WindowSizeMsg, KeyMsg
- `View()` - render list + detail pane side by side

### 4. Create `app/domains/drinks/surfaces/tui/detail_vm.go`

Implement DetailViewModel:
- Display: Name, ID, Category, Glass, Description
- Display recipe ingredients with quantities
- Handle nil drink (show "Select a drink...")

### 5. Wire in `main/tui/app.go`

```go
// In currentViewModel() switch
case ViewDrinks:
    vm = drinks.NewListViewModel(a.app, ListViewStylesFrom(a.styles), ListViewKeysFrom(a.keys))
```

Add import:
```go
drinks "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/tui"
```

## Notes

- Check `app/domains/drinks/models/drink.go` for actual Drink struct fields
- Check `app/domains/drinks/models/recipe.go` for Recipe struct
- The list component from bubbles handles filtering automatically
- Split pane layout: 60% list, 40% detail (adjust based on width)

## Checklist

- [ ] Create surfaces/tui/ directory under drinks domain
- [ ] Create messages.go with DrinksLoadedMsg
- [ ] Create items.go with drinkItem list.Item impl
- [ ] Create list_vm.go with ListViewModel
- [ ] Create detail_vm.go with DetailViewModel
- [ ] Wire ListViewModel in App.currentViewModel()
- [ ] Test navigation: Dashboard -> Drinks -> back
- [ ] Test data loading and display
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
