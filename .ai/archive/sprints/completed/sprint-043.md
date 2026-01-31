# Sprint 043 (Intermezzo): Organize Domain Tests by Method

## Goal

Partition domain tests into method-correlated test files (e.g., `create_test.go`, `update_test.go`, `delete_test.go`, `list_test.go`) to improve test discoverability and maintainability.

## Problem

Tests are currently organized by concern (ABAC, permissions, general) rather than by the method being exercised:

```
drinks/
├── drinks_test.go       # Mixed: Create, Get, Update, Delete, List
├── abac_test.go         # Mixed: Create, Update
└── permissions_test.go  # Mixed: List, Get, Create, Update, Delete

ingredients/
├── ingredients_test.go  # Create validation
├── update_test.go       # Update tests (good!)
├── delete_cascade_test.go # Delete tests (good!)
└── permissions_test.go  # Mixed: List, Get, Create, Update, Delete
```

When investigating a bug in `Update`, developers must search across multiple files. The `ingredients` domain partially follows the pattern already (`update_test.go`, `delete_cascade_test.go`), but most domains don't.

## Solution

Reorganize tests so each test file contains tests for a single method or closely related method group:

```
drinks/
├── create_test.go       # All Create tests (ABAC, permissions, validation)
├── update_test.go       # All Update tests
├── delete_test.go       # All Delete tests
├── get_test.go          # Get tests
└── list_test.go         # List/filter tests
```

### Guiding Principles

1. **Method-first organization**: Test file name matches the primary method being tested
2. **ABAC/permission tests merge into method files**: `TestDrinks_ABAC_SommelierCanCreateWine` goes to `create_test.go`
3. **Table-driven permission tests stay unified**: The comprehensive `TestPermissions_*` table-driven tests remain in `permissions_test.go` as they systematically verify all operations across all roles
4. **Shared helpers in common file**: Test fixtures and helpers that span multiple files go in `helpers_test.go`

## Current State by Domain

### drinks

| File | Tests | Target |
|------|-------|--------|
| `drinks_test.go` | `TestDrinks_CreateGetUpdateDelete` | Split or keep as integration test |
| | `TestDrinks_CreateRejectsIDProvided` | `create_test.go` |
| | `TestDrinks_ListFiltersByName` | `list_test.go` |
| `abac_test.go` | `TestDrinks_ABAC_SommelierCanCreateWine` | `create_test.go` |
| | `TestDrinks_ABAC_SommelierCannotChangeWineToCocktail` | `update_test.go` |
| | `TestDrinks_ABAC_BartenderCanUpdateCocktail` | `update_test.go` |
| `permissions_test.go` | `TestPermissions_Drinks` | Keep (table-driven, comprehensive) |

### ingredients

| File | Tests | Target |
|------|-------|--------|
| `ingredients_test.go` | `TestIngredients_CreateRejectsIDProvided` | `create_test.go` |
| `update_test.go` | All Update tests | Keep |
| `delete_cascade_test.go` | Delete cascade test | Rename to `delete_test.go` |
| `permissions_test.go` | `TestPermissions_Ingredients` | Keep (table-driven) |

### inventory

| File | Tests | Target |
|------|-------|--------|
| `inventory_test.go` | `TestInventory_SetAndAdjust` | Split: `set_test.go`, `adjust_test.go` |
| `permissions_test.go` | `TestPermissions_Inventory` | Keep (table-driven) |

### orders

| File | Tests | Target |
|------|-------|--------|
| `orders_test.go` | `TestOrders_PlaceRejectsIDProvided` | `place_test.go` |
| `permissions_test.go` | `TestPermissions_Orders` | Keep (table-driven) |

### menu

| File | Tests | Target |
|------|-------|--------|
| `menu_test.go` | `TestMenu_CreateRejectsIDProvided` | `create_test.go` |
| `handlers_test.go` | Handler/event tests | Keep (cross-cutting event handlers) |
| `permissions_test.go` | `TestPermissions_Menu` | Keep (table-driven) |

### audit

| File | Tests | Target |
|------|-------|--------|
| `audit_test.go` | `TestAudit_RecordsActivityForCommand` | `list_test.go` (audit is read-only) |
| | `TestAudit_TouchesIncludeHandlerUpdates` | `list_test.go` |
| | `TestAudit_ListFilters` | `list_test.go` |
| | `TestAudit_ListFiltersByTime` | `list_test.go` |
| `permissions_test.go` | `TestPermissions_Audit` | Keep (table-driven) |

## Tasks

### Phase 1: drinks domain

- [ ] Create `drinks/create_test.go` with:
  - `drinkForPolicy` helper
  - `TestDrinks_CreateRejectsIDProvided`
  - `TestDrinks_ABAC_SommelierCanCreateWine`
- [ ] Create `drinks/update_test.go` with:
  - `drinkForPolicy` helper (or move to `helpers_test.go`)
  - `TestDrinks_ABAC_SommelierCannotChangeWineToCocktail`
  - `TestDrinks_ABAC_BartenderCanUpdateCocktail`
- [ ] Create `drinks/list_test.go` with:
  - `TestDrinks_ListFiltersByName`
- [ ] Update `drinks/drinks_test.go` to only contain `TestDrinks_CreateGetUpdateDelete` (integration test)
- [ ] Remove `drinks/abac_test.go` (tests moved)

### Phase 2: ingredients domain

- [ ] Create `ingredients/create_test.go` with:
  - `TestIngredients_CreateRejectsIDProvided`
- [ ] Rename `delete_cascade_test.go` to `delete_test.go`
- [ ] Remove empty `ingredients_test.go`

### Phase 3: inventory domain

- [ ] Create `inventory/set_test.go` with Set tests
- [ ] Create `inventory/adjust_test.go` with Adjust tests
- [ ] Remove `inventory_test.go` (tests moved)

### Phase 4: orders domain

- [ ] Rename `orders_test.go` to `place_test.go`

### Phase 5: menu domain

- [ ] Rename `menu_test.go` to `create_test.go`

### Phase 6: audit domain

- [ ] Rename `audit_test.go` to `list_test.go`

### Phase 7: Verification

- [ ] Run `go test ./...` - all tests pass
- [ ] Verify no duplicate test names
- [ ] Verify test helpers are accessible where needed

## Acceptance Criteria

- [ ] Each domain has tests organized by method (create, update, delete, list, etc.)
- [ ] No orphaned `abac_test.go` files - ABAC tests live with the method they exercise
- [ ] Table-driven `permissions_test.go` files remain for comprehensive role coverage
- [ ] Shared test helpers are in `helpers_test.go` or accessible via common packages
- [ ] All tests pass
- [ ] No test name collisions

## Notes

### Why Keep permissions_test.go?

The table-driven `TestPermissions_*` tests are valuable as-is because they:
1. Systematically verify all roles against all operations in one place
2. Make it easy to add new roles or operations
3. Serve as documentation of the permission matrix

These complement the method-specific tests rather than replacing them.

### Test Helper Placement

If a helper like `drinkForPolicy` is needed in multiple files, move it to `helpers_test.go`:

```go
// drinks/helpers_test.go
package drinks_test

func drinkForPolicy(name string, category models.DrinkCategory, ingredientID entity.IngredientID) models.Drink {
    // ...
}
```

### File Naming Convention

| Method | File Name |
|--------|-----------|
| Create | `create_test.go` |
| Update | `update_test.go` |
| Delete | `delete_test.go` |
| Get | `get_test.go` |
| List | `list_test.go` |
| Set (inventory) | `set_test.go` |
| Adjust (inventory) | `adjust_test.go` |
| Place (orders) | `place_test.go` |
| Complete (orders) | `complete_test.go` |
| Cancel (orders) | `cancel_test.go` |
| AddDrink (menu) | `add_drink_test.go` |
| RemoveDrink (menu) | `remove_drink_test.go` |
| Publish (menu) | `publish_test.go` |
