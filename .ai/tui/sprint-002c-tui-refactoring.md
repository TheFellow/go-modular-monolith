# Sprint 002c: TUI Refactoring

## Overview

Address architectural issues discovered during sprint-002 implementation to maintain a clean modular monolith demo.

## Issues

### 1. Duplicate Struct Definitions

`ListViewStyles` and `ListViewKeys` are defined in 6 places:
- `main/tui/viewmodel_types.go` (intended shared types)
- `app/domains/drinks/surfaces/tui/list_vm.go`
- `app/domains/ingredients/surfaces/tui/list_vm.go`
- `app/domains/inventory/surfaces/tui/list_vm.go`
- `app/domains/menus/surfaces/tui/list_vm.go`
- `app/domains/orders/surfaces/tui/list_vm.go`

**Problem:** Violates DRY, makes changes error-prone.

**Solution:** Move shared types to `pkg/tui/types.go` so both `main/tui` and domain packages can import them without circular dependencies.

### 2. Boilerplate Mapping Methods in app.go

10 nearly identical methods map global styles/keys to domain-specific types:
- `drinksListStyles()`, `drinksListKeys()`
- `ingredientsListStyles()`, `ingredientsListKeys()`
- `inventoryListStyles()`, `inventoryListKeys()`
- `menusListStyles()`, `menusListKeys()`
- `ordersListStyles()`, `ordersListKeys()`

**Problem:** ~100 lines of repetitive code that must be copied for each new domain.

**Solution:** With shared types, domains accept `tui.ListViewStyles` directly. One `ListViewStylesFrom()` function suffices.

### 3. ListFilter Types in internal/dao

`ListFilter` structs are defined in `internal/dao` for each domain, but:
- The `queries` package exposes them in its public API signature
- TUI surfaces import `internal/dao` directly to access them

**Problem:** `internal/dao` should be truly internal. Having query-related types there forces consumers to reach into internal packages.

**Solution:** Re-export filter types from the `queries` package:

```go
// app/domains/drinks/queries/filters.go
type ListFilter = dao.ListFilter
```

TUI surfaces and other consumers then import `queries.ListFilter` instead of `dao.ListFilter`. This keeps query concerns in the queries package where they belong.

### 4. N+1 Ingredient Query Problem

When displaying drinks or inventory, each ingredient name is fetched individually:
- `drinks/surfaces/tui/detail_vm.go`: `ingredientName()` calls `Get()` per ingredient
- `inventory/surfaces/tui/list_vm.go`: Fetches ingredient per inventory item

**Problem:** A drink with 5 ingredients makes 5 database queries. Inventory list with 20 items makes 20 queries.

**Solution:** Add batch lookup to ingredients queries:

```go
// app/domains/ingredients/queries/queries.go
func (q *Queries) GetMany(ctx *middleware.Context, ids []entity.IngredientID) (map[entity.IngredientID]*models.Ingredient, error)
```

Or preload pattern:
```go
// Collect all IDs first, then batch fetch
ids := collectIngredientIDs(drink.Recipe.Ingredients)
ingredients, err := q.GetMany(ctx, ids)
// Then lookup from map
```

---

## Tasks

### Task 1: Create pkg/tui/types.go

Move shared types to a neutral package:
- `ListViewStyles`
- `ListViewKeys`
- `ListViewStylesFrom()`
- `ListViewKeysFrom()`

### Task 2: Update domains to use shared types

For each domain (drinks, ingredients, inventory, menus, orders):
- Delete local `ListViewStyles` and `ListViewKeys` definitions
- Import from `pkg/tui`
- Update `NewListViewModel()` signature

### Task 3: Remove boilerplate from app.go

- Delete domain-specific style/key mapping methods
- Use single `ListViewStylesFrom()` and `ListViewKeysFrom()`

### Task 4: Re-export ListFilter from queries

For each domain (drinks, ingredients, inventory, menus, orders, audit):
- Create `queries/filters.go` with `type ListFilter = dao.ListFilter`
- Update TUI surfaces to import `queries.ListFilter` instead of `dao.ListFilter`
- Remove direct `internal/dao` imports from TUI surfaces

### Task 5: Implement ViewModel registry

- Create `pkg/tui/registry.go` with `Register()` and `Create()`
- Define View constants in `pkg/tui/views.go`
- Update `main/tui/app.go` to use registry
- Add `register.go` to each domain's TUI surface

### Task 6: Add batch ingredient lookup

- Add `GetMany(ctx, ids)` to ingredients queries
- Update drinks detail_vm to batch fetch ingredients
- Update inventory list_vm to batch fetch ingredients

### Task 7: Update tests

- Update test helpers to use shared types
- Verify all existing tests pass

---

## Success Criteria

- [ ] `ListViewStyles`/`ListViewKeys` defined once in `pkg/tui/`
- [ ] Zero duplicate struct definitions across domains
- [ ] `ListFilter` re-exported from `queries` package (no `internal/dao` imports in TUI surfaces)
- [ ] app.go uses registry pattern, no domain imports
- [ ] Adding a new domain doesn't require modifying app.go
- [ ] Ingredient lookups are batched (1 query for N ingredients)
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
