# Sprint 017c: Test Fixtures with Isolated Databases (Intermezzo)

## Goal

Create test fixtures that bootstrap isolated databases per test, enabling full integration tests against the real system without tests interfering with each other.

## Problem

Current tests either:
1. Mock the database (doesn't test real behavior)
2. Use shared state that can leak between tests
3. Require manual cleanup (error-prone)

## Solution

Leverage bstore's file-based nature: each test gets its own database file via `t.TempDir()`. The fixture bootstraps a complete mixology store, and cleanup happens automatically when the test completes.

```go
func TestCreateDrink(t *testing.T) {
    fix := testutil.NewFixture(t)

    // Full system available - real DB, real modules
    drink, err := fix.Drinks.Create(fix.Ctx, models.Drink{
        Name: "Margarita",
    })
    require.NoError(t, err)

    // Verify it was saved
    found, err := fix.Drinks.Get(fix.Ctx, drink.ID)
    require.NoError(t, err)
    assert.Equal(t, "Margarita", found.Name)

    // Database file deleted after test - no cleanup needed
}
```

## Tasks

- [x] Create `pkg/testutil/fixture.go` with `NewFixture(t)`
- [x] Add fixture struct with initialized modules and context
- [x] Create bootstrap helpers for common starting states
- [x] Create module-specific helpers (e.g., `CreateDrink`)
- [x] Add example integration tests
- [x] Verify `go test ./...` passes

## Architecture

### Test Fixture

```go
// pkg/testutil/fixture.go
package testutil

import (
    "testing"

    "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
    "github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
    "github.com/TheFellow/go-modular-monolith/app/domains/inventory"
    "github.com/TheFellow/go-modular-monolith/app/domains/menu"
    "github.com/TheFellow/go-modular-monolith/app/domains/orders"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

// Fixture provides a complete test environment with isolated database.
type Fixture struct {
    T   testing.TB
    Ctx *middleware.Context

    // All domain modules, ready to use
    Drinks      *drinks.Module
    Ingredients *ingredients.Module
    Inventory   *inventory.Module
    Menu        *menu.Module
    Orders      *orders.Module
}

// NewFixture creates an isolated test environment.
// Each test gets its own database file via t.TempDir().
func NewFixture(t testing.TB) *Fixture {
    t.Helper()

    // Create isolated database in temp directory
    OpenStore(t)  // Uses t.TempDir(), registers cleanup

    // Create context with test principal
    ctx := ActorContext(t, "owner")

    return &Fixture{
        T:           t,
        Ctx:         ctx,
        Drinks:      drinks.NewModule(),
        Ingredients: ingredients.NewModule(),
        Inventory:   inventory.NewModule(),
        Menu:        menu.NewModule(),
        Orders:      orders.NewModule(),
    }
}

// AsActor returns a new context with the specified actor.
func (f *Fixture) AsActor(actor string) *middleware.Context {
    return ActorContext(f.T, actor)
}
```

### Bootstrap Helpers

Common starting states that tests can build upon:

```go
// pkg/testutil/bootstrap.go
package testutil

import (
    drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
    ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
)

// Bootstrap provides pre-configured starting states.
type Bootstrap struct {
    fix *Fixture
}

func (f *Fixture) Bootstrap() *Bootstrap {
    return &Bootstrap{fix: f}
}

// WithBasicIngredients creates common bar ingredients.
func (b *Bootstrap) WithBasicIngredients() *Bootstrap {
    basics := []ingredientsmodels.Ingredient{
        {Name: "Tequila", Category: ingredientsmodels.CategorySpirit, Unit: ingredientsmodels.UnitOz},
        {Name: "Lime Juice", Category: ingredientsmodels.CategoryJuice, Unit: ingredientsmodels.UnitOz},
        {Name: "Triple Sec", Category: ingredientsmodels.CategoryLiqueur, Unit: ingredientsmodels.UnitOz},
        {Name: "Simple Syrup", Category: ingredientsmodels.CategorySyrup, Unit: ingredientsmodels.UnitOz},
        {Name: "Vodka", Category: ingredientsmodels.CategorySpirit, Unit: ingredientsmodels.UnitOz},
        {Name: "Gin", Category: ingredientsmodels.CategorySpirit, Unit: ingredientsmodels.UnitOz},
    }
    for _, ing := range basics {
        _, err := b.fix.Ingredients.Create(b.fix.Ctx, ing)
        Ok(b.fix.T, err)
    }
    return b
}

// WithStock adds inventory for all ingredients.
func (b *Bootstrap) WithStock(quantity float64) *Bootstrap {
    ingredients, err := b.fix.Ingredients.List(b.fix.Ctx)
    Ok(b.fix.T, err)

    for _, ing := range ingredients {
        _, err := b.fix.Inventory.Set(b.fix.Ctx, ing.ID, quantity)
        Ok(b.fix.T, err)
    }
    return b
}
```

### Module-Specific Helpers

Helpers that handle entity creation with all dependencies:

```go
// pkg/testutil/drinks.go
package testutil

import (
    drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
    ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
)

// DrinkBuilder creates drinks with their required ingredients.
type DrinkBuilder struct {
    fix  *Fixture
    name string
    ingredients []struct {
        name   string
        amount float64
    }
}

func (f *Fixture) CreateDrink(name string) *DrinkBuilder {
    return &DrinkBuilder{fix: f, name: name}
}

func (b *DrinkBuilder) With(ingredientName string, amount float64) *DrinkBuilder {
    b.ingredients = append(b.ingredients, struct {
        name   string
        amount float64
    }{ingredientName, amount})
    return b
}

// Build creates the drink and any missing ingredients.
func (b *DrinkBuilder) Build() drinksmodels.Drink {
    var recipeIngredients []drinksmodels.RecipeIngredient

    for _, ing := range b.ingredients {
        // Find or create ingredient
        ingredient := b.fix.findOrCreateIngredient(ing.name)
        recipeIngredients = append(recipeIngredients, drinksmodels.RecipeIngredient{
            IngredientID: ingredient.ID,
            Amount:       ing.amount,
            Unit:         ingredient.Unit,
        })
    }

    drink, err := b.fix.Drinks.Create(b.fix.Ctx, drinksmodels.Drink{
        Name:     b.name,
        Category: drinksmodels.DrinkCategoryCocktail,
        Glass:    drinksmodels.GlassTypeCoupe,
        Recipe: drinksmodels.Recipe{
            Ingredients: recipeIngredients,
        },
    })
    Ok(b.fix.T, err)
    return drink
}

func (f *Fixture) findOrCreateIngredient(name string) ingredientsmodels.Ingredient {
    // Try to find existing
    ing, err := f.Ingredients.FindByName(f.Ctx, name)
    if err == nil {
        return ing
    }

    // Create new
    ing, err = f.Ingredients.Create(f.Ctx, ingredientsmodels.Ingredient{
        Name:     name,
        Category: ingredientsmodels.CategorySpirit,
        Unit:     ingredientsmodels.UnitOz,
    })
    Ok(f.T, err)
    return ing
}
```

## Usage Examples

### Basic Test

```go
func TestCreateDrink(t *testing.T) {
    fix := testutil.NewFixture(t)

    drink, err := fix.Drinks.Create(fix.Ctx, models.Drink{
        Name:     "Margarita",
        Category: models.DrinkCategoryCocktail,
    })

    require.NoError(t, err)
    assert.NotEmpty(t, drink.ID)
    assert.Equal(t, "Margarita", drink.Name)
}
```

### Test with Bootstrap

```go
func TestMenuWithAvailability(t *testing.T) {
    fix := testutil.NewFixture(t)
    fix.Bootstrap().
        WithBasicIngredients().
        WithStock(10.0)

    // Create drink using bootstrapped ingredients
    drink := fix.CreateDrink("Margarita").
        With("Tequila", 2.0).
        With("Lime Juice", 1.0).
        With("Triple Sec", 0.5).
        Build()

    // Create menu with drink
    menu, err := fix.Menu.Create(fix.Ctx, models.Menu{
        Name: "Happy Hour",
        Items: []models.MenuItem{{DrinkID: drink.ID}},
    })
    require.NoError(t, err)

    // Verify availability calculated correctly
    assert.Equal(t, models.AvailabilityAvailable, menu.Items[0].Availability)
}
```

### Test with Different Actors

```go
func TestAuthorizationDenied(t *testing.T) {
    fix := testutil.NewFixture(t)

    // Owner creates a drink
    drink := fix.CreateDrink("Secret Recipe").Build()

    // Anonymous user cannot update it
    anonCtx := fix.AsActor("anonymous")
    _, err := fix.Drinks.Update(anonCtx, drink.ID, models.DrinkPatch{
        Name: optional.Some("Stolen Recipe"),
    })

    testutil.RequireDenied(t, err)
}
```

### Parallel Tests

```go
func TestDrinks(t *testing.T) {
    t.Run("Create", func(t *testing.T) {
        t.Parallel()
        fix := testutil.NewFixture(t)
        // Each parallel test gets its own database file
        drink := fix.CreateDrink("Mojito").Build()
        assert.Equal(t, "Mojito", drink.Name)
    })

    t.Run("List", func(t *testing.T) {
        t.Parallel()
        fix := testutil.NewFixture(t)
        // Completely isolated from other tests
        fix.CreateDrink("Daiquiri").Build()
        fix.CreateDrink("Cosmopolitan").Build()

        drinks, err := fix.Drinks.List(fix.Ctx)
        require.NoError(t, err)
        assert.Len(t, drinks, 2)
    })
}
```

## How Isolation Works

```
Test starts
    ↓
testutil.NewFixture(t)
    ↓
t.TempDir() ──→ /tmp/TestFoo123/
    ↓
store.Open(tmpDir + "/mixology.db")
    ↓
Modules initialized with isolated store
    ↓
Test executes (all writes go to temp DB)
    ↓
Test ends
    ↓
t.Cleanup() ──→ store.Close()
    ↓
OS deletes temp directory (database gone)
```

**Key insight**: bstore uses a single file per database. By using `t.TempDir()`, each test gets a completely isolated database that is automatically cleaned up by the testing framework.

## Success Criteria

- `testutil.NewFixture(t)` creates isolated test environment
- Each test has its own database file
- Bootstrap helpers for common starting states
- Module helpers like `CreateDrink` handle dependencies
- Parallel tests are isolated from each other
- No test data persists after test completion
- `go test ./...` passes

## Dependencies

- Sprint 017b (DAO separation)
