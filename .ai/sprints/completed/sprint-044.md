# Sprint 044: Cascade Entity Touches on Domain Events

## Goal

Remove artificial handlers and implement proper entity touch cascading so that updates to ingredients propagate touches
to affected drinks and menus in audit entries.

## Problem

### Artificial Handlers

The `ingredients/handlers/ingredient-created-counter.go` file contains two no-op handlers:

```go
// Does nothing - artificial demonstration code
func (h *IngredientCreatedCounter) Handle(_ *middleware.Context, e events.IngredientCreated) error {
_ = e.Ingredient
return nil
}

func (h *IngredientCreatedAudit) Handle(_ *middleware.Context, e events.IngredientCreated) error {
_ = e.Ingredient
return nil
}
```

These serve no purpose and clutter the codebase.

### Missing Touch Cascades

When an ingredient is updated, drinks that use it are affected. The audit entry for `ingredient:update` should include
touches to:

1. The ingredient itself (already happens via command)
2. All drinks using that ingredient
3. All menus containing those drinks

Currently, `IngredientUpdated` has no handlers. Compare to `IngredientDeleted` which properly cascades:

```
IngredientDeleted
  → IngredientDeletedDrinkCascader (deletes drinks, touches them)
  → IngredientDeletedMenuCascader (removes from menus, touches them)
  → IngredientDeletedStockCleaner (removes stock)
```

### Audit Trail Gap

Without touch cascades, the audit trail is incomplete. If someone asks "what changes affected Menu X?", updates to
ingredients used in Menu X's drinks won't appear.

## Solution

### Remove Artificial Handlers

Delete `ingredients/handlers/ingredient-created-counter.go` and remove from dispatcher.

### Add Touch Cascade Handlers

Create handlers that touch (but don't modify) related entities. Handlers are named after the event they handle:

| Event               | Domain  | Handler             | Touches                          |
|---------------------|---------|---------------------|----------------------------------|
| `IngredientUpdated` | drinks  | `IngredientUpdated` | Drinks using ingredient          |
| `IngredientUpdated` | menu    | `IngredientUpdated` | Menus containing affected drinks |

Note: `IngredientCreated` and `IngredientDeleted` don't need touch handlers:

- Created: No drinks use it yet
- Deleted: Existing cascaders already delete/touch affected entities

### Handler Implementation Pattern

```go
// drinks/handlers/ingredient-updated.go
type IngredientUpdated struct {
	drinkQueries *queries.Queries
}

func NewIngredientUpdated() *IngredientUpdated {
	return &IngredientUpdated{drinkQueries: queries.New()}
}

func (h *IngredientUpdated) Handle(ctx *middleware.Context, e ingredientsevents.IngredientUpdated) error {
	drinks, err := h.drinkQueries.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}
	for _, drink := range drinks {
		ctx.TouchEntity(drink.ID.EntityUID())
	}
	return nil
}
```

```go
// menu/handlers/ingredient-updated.go
type IngredientUpdated struct {
	menuDAO      *dao.DAO
	drinkQueries *drinksq.Queries
}

func NewIngredientUpdated() *IngredientUpdated {
	return &IngredientUpdated{
		menuDAO:      dao.New(),
		drinkQueries: drinksq.New(),
	}
}

func (h *IngredientUpdated) Handle(ctx *middleware.Context, e ingredientsevents.IngredientUpdated) error {
	drinks, err := h.drinkQueries.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{})
	for _, drink := range drinks {
		menus, err := h.menuDAO.ListByDrink(ctx, drink.ID)
		if err != nil {
			return err
		}
		for _, menu := range menus {
			if _, ok := seen[menu.ID.String()]; ok {
				continue
			}
			seen[menu.ID.String()] = struct{}{}
			ctx.TouchEntity(menu.ID.EntityUID())
		}
	}
	return nil
}
```

## Tasks

### Phase 1: Remove Artificial Handlers

- [ ] Delete `app/domains/ingredients/handlers/ingredient-created-counter.go`
- [ ] Run `go generate ./...` to update dispatcher
- [ ] Verify `IngredientCreated` case is removed from dispatcher (no handlers remain)

### Phase 2: Add IngredientUpdated Touch Handlers

- [ ] Create `app/domains/drinks/handlers/ingredient-updated.go` with `IngredientUpdated`
- [ ] Create `app/domains/menu/handlers/ingredient-updated.go` with `IngredientUpdated`
- [ ] Run `go generate ./...` to register handlers

### Phase 3: Verify Touch Cascade

- [ ] Write test: Update ingredient → audit entry touches affected drinks
- [ ] Write test: Update ingredient → audit entry touches affected menus
- [ ] Existing tests pass

### Phase 4: Cleanup

- [ ] Run `go test ./...`
- [ ] Verify audit tests in `audit_test.go` still pass
- [ ] Manual verification: update ingredient, check audit entry touches

## Acceptance Criteria

- [ ] `ingredient-created-counter.go` removed
- [ ] `IngredientUpdated` event triggers touch of all drinks using the ingredient
- [ ] `IngredientUpdated` event triggers touch of all menus containing affected drinks
- [ ] Audit entries for `ingredient:update` include touched drinks and menus
- [ ] All existing tests pass
- [ ] `go generate ./...` succeeds

## Event/Handler Matrix (After Sprint)

| Event               | Domain    | Handler                          | Behavior                        |
|---------------------|-----------|----------------------------------|---------------------------------|
| `IngredientCreated` | -         | (none)                           | No downstream dependencies      |
| `IngredientUpdated` | drinks    | `IngredientUpdated` (new)        | Touch drinks using ingredient   |
| `IngredientUpdated` | menu      | `IngredientUpdated` (new)        | Touch menus containing drinks   |
| `IngredientDeleted` | drinks    | `IngredientDeletedDrinkCascader` | Delete drinks, touch them       |
| `IngredientDeleted` | inventory | `IngredientDeletedStockCleaner`  | Clean stock                     |
| `IngredientDeleted` | menu      | `IngredientDeletedMenuCascader`  | Remove from menus, touch them   |
| `DrinkCreated`      | -         | (none)                           | No downstream dependencies      |
| `DrinkUpdated`      | menu      | `DrinkUpdatedMenuUpdater`        | Recalculate availability, touch |
| `DrinkDeleted`      | menu      | `DrinkDeletedMenuUpdater`        | Remove from menus, touch them   |

## Notes

### Why Not Touch on Create?

When an ingredient or drink is created, nothing depends on it yet. Touches are only meaningful when existing
relationships are affected.

### Touch vs Modify

The new handlers only touch entities - they don't modify them. This is different from existing handlers like:

- `IngredientDeletedDrinkCascader` which deletes drinks
- `DrinkUpdatedMenuUpdater` which updates availability

(Note: Existing handlers use longer names; new handlers follow the simpler event-name convention.)

Touching records that an entity was indirectly affected by an operation, enabling comprehensive audit trails.

### Handler Ordering

The drinks domain `IngredientUpdated` handler should run before the menu domain's to maintain logical ordering, but
since both only touch (no data dependencies), order doesn't affect correctness.

### Naming Convention

Handlers are named after the event they handle. Since each handler lives in its own domain package, the package
provides disambiguation:

- `drinks_handlers.IngredientUpdated`
- `menu_handlers.IngredientUpdated`
