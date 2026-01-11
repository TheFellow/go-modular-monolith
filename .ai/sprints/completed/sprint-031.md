# Sprint 031: Cross-Domain Event Handlers for Cascading Operations

## Status

- Started: 2026-01-10
- Completed: 2026-01-10

## Goal

Ensure all CRUD operations that impact adjacent domains have appropriate event handlers to maintain consistency across the system.

## Outcome

- Added `Ingredients.Delete` (soft delete) and emitted `IngredientDeleted`.
- Added cross-domain handlers:
  - `IngredientDeleted` → soft-delete drinks that reference it (and emit `DrinkDeleted`)
  - `IngredientDeleted` → delete inventory stock record
  - `DrinkRecipeUpdated` → mark published menu items unavailable when new required ingredients are out of stock
  - `MenuPublished` → recompute item availability from current inventory/drink state
- Enabled event cascades by letting `DispatchEvents` process events added by handlers.

## Current State

### Domain Operations

| Domain | Create | Get | List | Update | Delete |
|--------|--------|-----|------|--------|--------|
| Ingredients | ✓ | ✓ | ✓ | ✓ | ✗ |
| Drinks | ✓ | ✓ | ✓ | ✓ | ✓ |
| Inventory | Set | ✓ | ✓ | Adjust | ✗ |
| Menu | ✓ | ✓ | ✓ | AddDrink/RemoveDrink | ✗ |
| Orders | Place | ✓ | ✓ | - | Cancel |

### Existing Handlers

| Event | Handler | Action |
|-------|---------|--------|
| `DrinkDeleted` | `DrinkDeletedMenuUpdater` | Remove drink from all menus |
| `StockAdjusted` | `StockAdjustedMenuUpdater` | Update menu item availability |
| `OrderCompleted` | `OrderCompletedMenuUpdater` | Mark items unavailable if depleted |
| `OrderCompleted` | `OrderCompletedStockUpdater` | Reduce stock quantities |

### Domain Relationships

```
Ingredients ←── Drinks ←── Menu ←── Orders
     │              │         │
     └── Inventory  │         └── (references MenuID)
                    │
                    └── Recipe.Ingredients[] references IngredientID
```

## Missing Events & Handlers

### 1. IngredientDeleted (Event + Handler)

**Event:** `IngredientDeleted` - does not exist
**Handler:** When ingredient deleted → cascade delete drinks using it

```
Ingredient deleted
  → Find all drinks with Recipe.Ingredients containing IngredientID
  → Delete those drinks (triggers DrinkDeleted)
  → DrinkDeleted handler removes from menus
```

**Also requires:** Add `Delete` operation to ingredients module

### 2. IngredientDeleted → Inventory Handler

**Handler:** When ingredient deleted → remove stock record

```
Ingredient deleted
  → Delete inventory stock for that ingredient
```

### 3. DrinkRecipeUpdated Handler

**Event:** `DrinkRecipeUpdated` - exists
**Handler:** Does not exist

When drink recipe changes (new ingredients added):
```
DrinkRecipeUpdated
  → For each added ingredient, check inventory stock
  → If any new required ingredient has 0 stock, mark menu items unavailable
```

### 4. IngredientUpdated → Drinks Handler

**Event:** `IngredientUpdated` - exists
**Handler:** Does not exist

When ingredient unit changes:
```
IngredientUpdated (unit changed)
  → Find all drinks using that ingredient
  → Log warning or update recipe units (depends on business rule)
```

This may be overly complex - consider if unit changes should be blocked if ingredient is in use.

### 5. MenuDeleted (Event + Handler)

**Event:** `MenuDeleted` - does not exist (menu has no delete)

If menu delete is added:
```
MenuDeleted
  → Check for open orders referencing menu
  → Either block delete or cancel orders
```

### 6. MenuPublished Handler

**Event:** `MenuPublished` - exists
**Handler:** Does not exist

When menu published:
```
MenuPublished
  → Validate all drinks on menu exist
  → Check inventory for required ingredients
  → Set initial availability status for each item
```

## Priority Analysis

| Missing Handler | Impact | Priority |
|-----------------|--------|----------|
| IngredientDeleted → Delete drinks | **Critical** - orphaned drink references | P0 |
| IngredientDeleted → Delete stock | **Critical** - orphaned stock records | P0 |
| DrinkRecipeUpdated → Menu availability | Medium - stale availability | P1 |
| MenuPublished → Validate/availability | Medium - published menu may have issues | P1 |
| IngredientUpdated → Unit validation | Low - edge case | P2 |

## Implementation

### Phase 1: Add Ingredient Delete with Cascades

#### 1.1 Add IngredientDeleted Event

```go
// ingredients/events/ingredient_deleted.go
type IngredientDeleted struct {
    Ingredient models.Ingredient
    DeletedAt  time.Time
}
```

#### 1.2 Add Delete to Ingredients Module

```go
// ingredients/delete.go
func (m *Module) Delete(ctx *middleware.Context, id cedar.EntityUID) (*models.Ingredient, error)
```

#### 1.3 Add IngredientDeleted → Drinks Handler

```go
// drinks/handlers/ingredient_deleted.go
type IngredientDeletedDrinkCascader struct {
    drinkDAO     *dao.DAO
    drinkQueries *queries.Queries
}

func (h *IngredientDeletedDrinkCascader) Handle(ctx *middleware.Context, e ingredientsevents.IngredientDeleted) error {
    // Find all drinks using this ingredient
    drinks, err := h.drinkQueries.ListByIngredient(ctx, e.Ingredient.ID)
    if err != nil {
        return err
    }

    // Delete each drink (this will trigger DrinkDeleted → menu cleanup)
    for _, drink := range drinks {
        if err := h.drinkDAO.Delete(ctx, drink.ID); err != nil {
            return err
        }
        ctx.AddEvent(drinksevents.DrinkDeleted{Drink: *drink, DeletedAt: time.Now()})
    }

    return nil
}
```

#### 1.4 Add IngredientDeleted → Inventory Handler

```go
// inventory/handlers/ingredient_deleted.go
type IngredientDeletedStockCleaner struct {
    stockDAO *dao.DAO
}

func (h *IngredientDeletedStockCleaner) Handle(ctx *middleware.Context, e ingredientsevents.IngredientDeleted) error {
    return h.stockDAO.DeleteByIngredient(ctx, e.Ingredient.ID)
}
```

### Phase 2: Recipe Update Availability

#### 2.1 Add DrinkRecipeUpdated Handler

```go
// menu/handlers/drink_recipe_updated.go
type DrinkRecipeUpdatedMenuUpdater struct {
    menuDAO        *dao.DAO
    inventoryQuery *inventoryq.Queries
}

func (h *DrinkRecipeUpdatedMenuUpdater) Handle(ctx *middleware.Context, e drinksevents.DrinkRecipeUpdated) error {
    // For newly added ingredients, check if any are out of stock
    for _, ingredientID := range e.AddedIngredients {
        stock, err := h.inventoryQuery.Get(ctx, ingredientID)
        if err != nil || stock == nil || stock.Quantity == 0 {
            // Mark drink as unavailable on all menus
            return h.markDrinkUnavailable(ctx, e.Current.ID)
        }
    }
    return nil
}
```

### Phase 3: Menu Publish Validation

#### 3.1 Add MenuPublished Handler

```go
// menu/handlers/menu_published.go
type MenuPublishedValidator struct {
    drinkQueries     *drinksq.Queries
    inventoryQueries *inventoryq.Queries
    menuDAO          *dao.DAO
}

func (h *MenuPublishedValidator) Handle(ctx *middleware.Context, e menuevents.MenuPublished) error {
    menu := e.Menu

    for i, item := range menu.Items {
        // Verify drink exists
        drink, err := h.drinkQueries.Get(ctx, item.DrinkID)
        if err != nil {
            // Drink doesn't exist - mark unavailable
            menu.Items[i].Availability = models.AvailabilityUnavailable
            continue
        }

        // Check ingredient availability
        available := h.checkIngredientsAvailable(ctx, drink.Recipe.Ingredients)
        if !available {
            menu.Items[i].Availability = models.AvailabilityUnavailable
        }
    }

    return h.menuDAO.Update(ctx, menu)
}
```

### Phase 4: Query Helpers

Add query methods needed by handlers:

```go
// drinks/queries/list_by_ingredient.go
func (q *Queries) ListByIngredient(ctx context.Context, ingredientID cedar.EntityUID) ([]*models.Drink, error)

// inventory/internal/dao/delete.go
func (d *DAO) DeleteByIngredient(ctx context.Context, ingredientID cedar.EntityUID) error
```

## Tasks

### Phase 1: Ingredient Delete Cascade

- [x] Create `IngredientDeleted` event
- [x] Add `Delete` operation to ingredients module
- [x] Add `ListByIngredient` query to drinks
- [x] Create `IngredientDeletedDrinkCascader` handler in drinks
- [x] Add `DeleteByIngredient` to inventory DAO
- [x] Create `IngredientDeletedStockCleaner` handler in inventory
- [x] Register handlers in app wiring
- [x] Add CLI command for ingredient delete

### Phase 2: Recipe Update Handler

- [x] Create `DrinkRecipeUpdatedMenuUpdater` handler
- [x] Register handler in app wiring

### Phase 3: Menu Publish Handler

- [x] Create `MenuPublishedValidator` handler
- [x] Register handler in app wiring

### Phase 4: Tests

- [x] Test: Delete ingredient → drinks using it are deleted
- [x] Test: Delete ingredient → stock record removed
- [x] Test: Delete ingredient → drinks removed from menus (cascade through DrinkDeleted)
- [x] Test: Update drink recipe with out-of-stock ingredient → menu item unavailable
- [x] Test: Publish menu → items with missing/unavailable ingredients marked unavailable
- [x] Verify `go test ./...` passes

## Event Flow Diagram

```
IngredientDeleted
  ├── IngredientDeletedDrinkCascader (drinks)
  │     └── For each drink using ingredient:
  │           └── Delete drink → DrinkDeleted event
  │                 └── DrinkDeletedMenuUpdater (menu)
  │                       └── Remove drink from all menus
  │
  └── IngredientDeletedStockCleaner (inventory)
        └── Delete stock record

DrinkRecipeUpdated
  └── DrinkRecipeUpdatedMenuUpdater (menu)
        └── Check new ingredients → update availability

MenuPublished
  └── MenuPublishedValidator (menu)
        └── Validate drinks & ingredients → set availability
```

## Acceptance Criteria

- Deleting an ingredient cascades to delete all drinks using it
- Deleting an ingredient removes its stock record
- Drinks deleted via cascade trigger menu cleanup
- Recipe updates that add out-of-stock ingredients mark menu items unavailable
- Publishing a menu validates and sets initial availability for all items
- All tests pass
- No orphaned references across domains
