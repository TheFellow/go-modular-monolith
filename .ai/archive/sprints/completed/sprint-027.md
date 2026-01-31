# Sprint 027: Rich Domain Models with Enforced Identity Invariants

## Status

- Started: 2026-01-08
- Completed: 2026-01-09

## Goal

Remove defensive guards that mask invalid domain state. Business logic enforces identity rules; persistence returns NotFound for invalid IDs.

## Problem

The codebase contains defensive guards in `CedarEntity()` implementations:

```go
func (d Drink) CedarEntity() cedar.Entity {
    uid := d.ID
    if string(uid.ID) == "" {
        uid = cedar.NewEntityUID(DrinkEntityType, cedar.String(""))  // Masks the problem!
    }
    return cedar.Entity{UID: uid, ...}
}
```

**Why this is wrong:**

1. **Masks invalid state**: Silently "fixes" empty IDs instead of letting downstream logic handle it
2. **Redundant**: An invalid ID (empty, wrong type) results in NotFound from the DAO anyway
3. **Cedar authorization leakage**: Creates a CedarEntity with fabricated ID

## Solution

1. **CedarEntity() methods**: Use the ID directly - no guards
2. **Create commands**: Return `Invalid` error if ID is provided (only meaningful check)
3. **Other operations**: Let NotFound handle invalid/missing IDs naturally

## Implementation

### CedarEntity() - Use ID Directly

```go
// Before
func (d Drink) CedarEntity() cedar.Entity {
    uid := d.ID
    if string(uid.ID) == "" {
        uid = cedar.NewEntityUID(DrinkEntityType, cedar.String(""))
    }
    return cedar.Entity{UID: uid, ...}
}

// After
func (d Drink) CedarEntity() cedar.Entity {
    return cedar.Entity{UID: d.ID, ...}
}
```

### Create Commands - Reject If ID Provided

```go
func (c *Commands) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
    if string(drink.ID.ID) != "" {
        return models.Drink{}, errors.Invalidf("id must be empty for create")
    }
    // ... assign ID, persist, return
}
```

## Affected Files

### CedarEntity() Guards to Remove

| File | Entity |
|------|--------|
| `app/domains/drinks/models/drink.go` | `Drink` |
| `app/domains/ingredients/models/ingredient.go` | `Ingredient` |
| `app/domains/menu/models/menu.go` | `Menu` |
| `app/domains/menu/models/drink_change.go` | `MenuDrinkChange` |
| `app/domains/orders/models/order.go` | `Order` |
| `app/domains/orders/get.go` | `GetRequest` |
| `app/domains/inventory/models/stock.go` | `Stock` |
| `app/domains/inventory/models/patch.go` | `StockPatch` |
| `app/domains/inventory/models/update.go` | `StockUpdate` |

### Create Commands to Update

| Domain | Command |
|--------|---------|
| drinks | `Create` |
| ingredients | `Create` |
| menu | `Create` |
| orders | `Place` |

### Redundant ID Checks to Remove

| File | Check |
|------|-------|
| `app/domains/orders/queries/get.go` | `string(id.ID) == ""` |
| `app/domains/orders/internal/commands/complete.go` | `string(order.ID.ID) == ""` |
| `app/domains/orders/internal/commands/cancel.go` | `string(order.ID.ID) == ""` |
| `app/domains/menu/internal/commands/publish.go` | `string(menu.ID.ID) == ""` |
| `app/domains/menu/internal/commands/add_drink.go` | MenuID/DrinkID checks |
| `app/domains/menu/internal/commands/remove_drink.go` | MenuID/DrinkID checks |
| `app/domains/inventory/internal/commands/set.go` | `string(update.IngredientID.ID) == ""` |
| `app/domains/inventory/internal/commands/adjust.go` | `string(patch.IngredientID.ID) == ""` |

### Event Handler Guard to Remove

| File | Issue |
|------|-------|
| `app/domains/menu/handlers/order_completed.go` | Silent skip on empty ID |

## Tasks

### Phase 1: CedarEntity() Methods

- [x] `Drink.CedarEntity()` - remove guard
- [x] `Ingredient.CedarEntity()` - remove guard
- [x] `Menu.CedarEntity()` - remove guard
- [x] `MenuDrinkChange.CedarEntity()` - remove guard
- [x] `Order.CedarEntity()` - remove guard
- [x] `GetRequest.CedarEntity()` - remove guard
- [x] `Stock.CedarEntity()` - remove guard
- [x] `StockPatch.CedarEntity()` - remove guard
- [x] `StockUpdate.CedarEntity()` - remove guard

### Phase 2: Create Commands

- [x] drinks `Create` - reject if ID provided
- [x] ingredients `Create` - reject if ID provided
- [x] menu `Create` - reject if ID provided
- [x] orders `Place` - reject if ID provided

### Phase 3: Remove Redundant Checks

- [x] Remove ID checks from commands that aren't Create (let NotFound handle it)
- [x] Remove silent skip in `order_completed.go` handler

### Phase 4: Tests

- [x] Add tests: Create with ID returns error
- [x] Verify `go test ./...` passes

## Acceptance Criteria

- `CedarEntity()` methods use ID directly with no guards
- Create commands reject entities that have an ID
- No redundant ID checks - NotFound handles invalid IDs
- All tests pass
