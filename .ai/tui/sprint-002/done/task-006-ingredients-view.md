# Task 006: Ingredients View Implementation

## Goal

Create the Ingredients domain ListViewModel and DetailViewModel, replacing the placeholder view.

## Design Principles

- **Keep it simple and direct** - Query data from domain queries, render it
- **No fallback logic** - If data should exist and doesn't, that's an internal error
- **Surface errors** - Return/display errors, never silently hide them
- **Self-consistent data** - The application guarantees referential integrity; trust it

## Files to Create/Modify

- `app/domains/ingredients/surfaces/tui/messages.go` (new)
- `app/domains/ingredients/surfaces/tui/list_vm.go` (new)
- `app/domains/ingredients/surfaces/tui/detail_vm.go` (new)
- `app/domains/ingredients/surfaces/tui/items.go` (new)
- `main/tui/app.go` - Wire IngredientsListViewModel

## Pattern Reference

Follow task-005 (Drinks View) as the template. Reference `app/domains/ingredients/surfaces/cli/views.go` for field access.

## Implementation

### 1. Create messages.go

```go
package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"

type IngredientsLoadedMsg struct {
    Ingredients []models.Ingredient
}
```

### 2. Create items.go

```go
type ingredientItem struct {
    ingredient models.Ingredient
}

func (i ingredientItem) Title() string       { return i.ingredient.Name }
func (i ingredientItem) Description() string { return fmt.Sprintf("%s • %s", i.ingredient.Category, i.ingredient.Unit) }
func (i ingredientItem) FilterValue() string { return i.ingredient.Name }
```

### 3. Create list_vm.go

Implement ListViewModel:
- Load ingredients via app.Ingredients.List()
- Display: Name, Category, Unit
- Handle selection to update detail view

### 4. Create detail_vm.go

Display for selected ingredient:
- Name, ID, Category, Unit
- Optionally: Current stock level (query inventory)
- Optionally: Drinks using this ingredient (query drinks by ingredient)

Note: Cross-domain queries (inventory, drinks) can be deferred if complex.
Start with basic ingredient details only.

### 5. Wire in app.go

```go
case ViewIngredients:
    vm = ingredients.NewListViewModel(a.app, ListViewStylesFrom(a.styles), ListViewKeysFrom(a.keys))
```

## Notes

- Check `app/domains/ingredients/models/` for Ingredient struct
- Category is likely an enum type - use .String() method
- Unit field shows measurement unit (oz, ml, count, etc.)

## Checklist

- [x] Create surfaces/tui/ directory under ingredients domain
- [x] Create messages.go with IngredientsLoadedMsg
- [x] Create items.go with ingredientItem
- [x] Create list_vm.go with ListViewModel
- [x] Create detail_vm.go with DetailViewModel
- [x] Wire ListViewModel in App.currentViewModel()
- [ ] Test navigation and data display
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
