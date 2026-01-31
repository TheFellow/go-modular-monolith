# Sprint 036: Protect Ingredient Unit Changes

## Goal

Prevent ingredient unit changes when the ingredient is in use by recipes, protecting data integrity.

## Problem

Currently, ingredient units can be changed freely:

```go
// ingredients/internal/commands/update.go
func (c *Command) Update(ctx *middleware.Context, ing *models.Ingredient) (*models.Ingredient, error) {
    // Unit change is allowed without checks!
    ing.Unit = models.Unit(ing.Unit)  // validated but not protected
    // ...
}
```

But recipes store their own copy of the unit:

```go
// drinks/models/recipe.go
type RecipeIngredient struct {
    IngredientID entity.IngredientID
    Amount       float64
    Unit         measurement.Unit  // Stored separately from ingredient!
    // ...
}
```

**The danger:**

1. Create ingredient "Vodka" with unit "oz"
2. Create recipe "Martini" using 2 oz Vodka
3. Update ingredient "Vodka" to unit "ml"
4. Recipe still says "2 oz" but ingredient is now "ml"
5. Inventory, costing, and display are all inconsistent

## Solution

**Strategy A (Blocking):** Reject unit changes if the ingredient is used in any recipe.

This is the simplest, safest approach. If someone needs to change units, they must:
1. Update all recipes using that ingredient first
2. Then change the ingredient's unit

### Implementation

Add a validation check in the ingredient update command:

```go
// ingredients/internal/commands/update.go

func (c *Command) Update(ctx *middleware.Context, ing *models.Ingredient) (*models.Ingredient, error) {
    // Fetch current state
    current, err := c.dao.Get(ctx, ing.ID)
    if err != nil {
        return nil, err
    }

    // Check if unit is changing
    if ing.Unit != "" && ing.Unit != current.Unit {
        // Query for recipes using this ingredient
        usageCount, err := c.recipeChecker.CountUsages(ctx, ing.ID)
        if err != nil {
            return nil, err
        }
        if usageCount > 0 {
            return nil, errors.Conflictf(
                "cannot change unit: ingredient is used in %d recipe(s)",
                usageCount,
            )
        }
    }

    // ... rest of update logic
}
```

### Cross-Domain Query

The ingredients domain needs to query the drinks domain to check recipe usage. Options:

**Option 1: Direct DAO Query (simple)**

Add a query method to drinks domain that ingredients can call:

```go
// drinks/queries/ingredient_usage.go

type IngredientUsageQuery struct {
    dao *dao.DAO
}

func (q *IngredientUsageQuery) CountRecipesUsingIngredient(
    ctx *middleware.Context,
    ingredientID entity.IngredientID,
) (int, error) {
    return q.dao.CountRecipesWithIngredient(ctx, ingredientID)
}
```

```go
// drinks/internal/dao/queries.go

func (d *DAO) CountRecipesWithIngredient(
    ctx dao.Context,
    ingredientID entity.IngredientID,
) (int, error) {
    var count int
    err := dao.Read(ctx, func(tx *bstore.Tx) error {
        // Query recipe_ingredients table for this ingredient
        rows, err := tx.QueryIndices(RecipeIngredientRow{}, "IngredientID", ingredientID.String())
        if err != nil {
            return err
        }
        count = len(rows)
        return nil
    })
    return count, err
}
```

**Option 2: Interface in Kernel (decoupled)**

Define an interface in the kernel that drinks implements:

```go
// app/kernel/queries/ingredient_usage.go

package queries

import (
    "github.com/TheFellow/go-modular-monolith/app/kernel/entity"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

// IngredientUsageChecker checks if an ingredient is in use.
type IngredientUsageChecker interface {
    CountRecipesUsingIngredient(ctx *middleware.Context, id entity.IngredientID) (int, error)
}
```

Drinks module implements this interface, ingredients module depends on the interface.

**Recommendation:** Option 1 for simplicity. Cross-domain queries are acceptable for read operations.

### Wire Up

```go
// app/domains/ingredients/module.go

type Module struct {
    // ...
    recipeChecker *drinks.IngredientUsageQuery  // Add dependency
}

func New(db *bstore.DB, drinksModule *drinks.Module) *Module {
    return &Module{
        // ...
        recipeChecker: drinksModule.IngredientUsageQuery(),
    }
}
```

### Error Response

When blocked, return a clear error:

```
$ mixology ingredients update --id ing-abc123 --unit ml
Error: cannot change unit: ingredient is used in 3 recipe(s)

To change the unit:
1. Update recipes using this ingredient
2. Then change the ingredient unit
```

## Alternative: Strategy B (Flagging)

For reference, the alternative approach would:
1. Allow the unit change
2. Have an `IngredientUpdated` handler in drinks domain
3. Handler finds all recipes using the ingredient
4. Marks affected drinks as `Draft` or adds a warning flag
5. Requires manual intervention to fix recipes

This is more complex and risks data inconsistency during the "fix" window. Not recommended for this project's strictness level.

## Tasks

### Phase 1: Add Usage Query

- [x] Add `CountRecipesWithIngredient` method to drinks DAO
- [x] Add `IngredientUsageQuery` to drinks module public API
- [x] Add unit test for the query

### Phase 2: Add Validation

- [x] Update ingredients `Update` command to check unit changes
- [x] Inject drinks query dependency into ingredients module
- [x] Return `Conflict` error when unit change is blocked

### Phase 3: Update Wiring

- [x] Update `app/app.go` to wire drinks → ingredients dependency
- [x] Ensure no circular dependencies

### Phase 4: Verify

- [x] Test: Update unit when ingredient not in use → succeeds
- [x] Test: Update unit when ingredient in recipe → returns conflict error
- [x] Test: Update other fields (name, category) → succeeds regardless of usage
- [x] Run all tests

## Acceptance Criteria

- [x] Unit changes blocked when ingredient is used in recipes
- [x] Clear error message explains why and what to do
- [x] Other ingredient updates (name, category, description) unaffected
- [x] No circular dependencies between modules
- [x] All tests pass

## Result

```bash
# Ingredient in use
$ mixology ingredients update --id ing-vodka --unit ml
Error: cannot change unit: ingredient is used in 5 recipe(s)

# Ingredient not in use
$ mixology ingredients update --id ing-unused --unit ml
ing-unused  Unused Ingredient  spirit  ml

# Other updates always work
$ mixology ingredients update --id ing-vodka --name "Premium Vodka"
ing-vodka  Premium Vodka  spirit  oz
```
