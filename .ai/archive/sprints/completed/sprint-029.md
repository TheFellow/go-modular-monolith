# Sprint 029: Remove Test-Only Abstractions, Use Real Code in Tests

## Status

- Started: 2026-01-10
- Completed: 2026-01-10

## Goal

Remove interface abstractions and `NewWithDependencies` constructors that exist solely for testing. Module tests should use the test fixture and exercise real code through the module's public API.

## Problem

The codebase has test-only abstractions:

```go
// drinks/internal/commands/commands.go
type ingredientReader interface {
    Get(ctx context.Context, id cedar.EntityUID) (ingredientsmodels.Ingredient, error)
}

func NewWithDependencies(d *dao.DAO, ingredients ingredientReader) *Commands {
    return &Commands{dao: d, ingredients: ingredients}
}
```

Tests then use fake implementations:

```go
// drinks/internal/commands/create_test.go
type fakeIngredients struct{}

func (f fakeIngredients) Get(_ context.Context, _ cedar.EntityUID) (ingredientsmodels.Ingredient, error) {
    return ingredientsmodels.Ingredient{}, nil
}

func TestCreate_PersistsOnCommit(t *testing.T) {
    cmds := commands.NewWithDependencies(d, fakeIngredients{})
    // ...
}
```

**Why this is wrong:**

1. **Production code polluted with test concerns**: Interfaces and constructors that never run in production
2. **Faux mocking**: Mocking internal module dependencies is excessive - these aren't external services
3. **Tests don't test real behavior**: Fake ingredients always return success, hiding real integration issues
4. **Test isolation is fixture's job**: Transaction rollback, temp databases - not fake dependencies
5. **Inconsistent with existing patterns**: `permissions_test.go` files already use fixture + real modules correctly

## Solution

1. **Remove test-only interfaces**: Depend on concrete `*queries.Queries` types directly
2. **Remove `NewWithDependencies`**: Only `New()` constructor needed
3. **Move tests to module level**: Test through public module API using fixture
4. **Use real dependencies**: Real ingredients, real inventory, real everything
5. **Unit tests where appropriate**: Kernel packages, pure functions, isolated logic

## The Right Pattern

Already exists in `permissions_test.go`:

```go
func TestPermissions_Drinks(t *testing.T) {
    f := testutil.NewFixture(t)

    // Uses real module through fixture
    _, err := f.App.Drinks.Create(owner, models.Drink{...})
    testutil.RequireNotDenied(t, err)
}
```

Tests should:
- Use `fix.Drinks.Create()` not `cmds.Create()`
- Bootstrap test data via fixture: `fix.Bootstrap().WithIngredient(...)`
- Let fixture handle isolation (temp DB per test)

## Audit Results

### Test-Only Interfaces to Remove

| File | Interface |
|------|-----------|
| `app/domains/drinks/internal/commands/commands.go` | `ingredientReader` |
| `app/domains/inventory/internal/commands/commands.go` | `ingredientReader` |

### `NewWithDependencies` to Remove

| File |
|------|
| `app/domains/drinks/internal/commands/commands.go` |
| `app/domains/inventory/internal/commands/commands.go` |
| `app/domains/orders/internal/commands/commands.go` |

### Tests Using Fake Dependencies

| Test File | Issue |
|-----------|-------|
| `drinks/internal/commands/create_test.go` | `fakeIngredients{}`, direct command calls |
| `drinks/internal/commands/delete_test.go` | `fakeIngredients{}`, direct command calls |
| `drinks/internal/commands/update_test.go` | `fakeIngredientsOK{}`, direct command calls |
| `drinks/queries/list_test.go` | `fakeIngredients{}`, direct command calls |
| `inventory/internal/commands/adjust_test.go` | `fakeIngredients{}`, direct command calls |

### Tests Already Correct (Keep as-is)

| Test File | Pattern |
|-----------|---------|
| `drinks/permissions_test.go` | Uses fixture + real modules |
| `ingredients/permissions_test.go` | Uses fixture + real modules |
| `inventory/permissions_test.go` | Uses fixture + real modules |
| `menu/permissions_test.go` | Uses fixture + real modules |
| `orders/permissions_test.go` | Uses fixture + real modules |
| `dispatcher/dispatcher_integration_test.go` | Uses fixture + real modules |

## Changes

### drinks/internal/commands/commands.go

```go
// Before
type Commands struct {
    dao         *dao.DAO
    ingredients ingredientReader
}

type ingredientReader interface {
    Get(ctx context.Context, id cedar.EntityUID) (ingredientsmodels.Ingredient, error)
}

func New() *Commands {
    return &Commands{
        dao:         dao.New(),
        ingredients: ingredientsqueries.New(),
    }
}

func NewWithDependencies(d *dao.DAO, ingredients ingredientReader) *Commands {
    return &Commands{dao: d, ingredients: ingredients}
}

// After
type Commands struct {
    dao         *dao.DAO
    ingredients *ingredientsqueries.Queries
}

func New() *Commands {
    return &Commands{
        dao:         dao.New(),
        ingredients: ingredientsqueries.New(),
    }
}
```

### inventory/internal/commands/commands.go

Same pattern - remove `ingredientReader` interface, use concrete type, delete `NewWithDependencies`.

### orders/internal/commands/commands.go

Already uses concrete types. Just delete `NewWithDependencies`.

### Test Migration Example

```go
// Before: drinks/internal/commands/create_test.go
func TestCreate_PersistsOnCommit(t *testing.T) {
    fix := testutil.NewFixture(t)
    d := dao.New()
    cmds := commands.NewWithDependencies(d, fakeIngredients{})

    err := fix.Store.Write(context.Background(), func(tx *bstore.Tx) error {
        ctx := middleware.NewContext(fix.Ctx, middleware.WithTransaction(tx))
        created, err = cmds.Create(ctx, drinksmodels.Drink{
            Recipe: drinksmodels.Recipe{
                Ingredients: []drinksmodels.RecipeIngredient{{
                    IngredientID: entity.IngredientID("lime-juice"),  // Fake!
                    // ...
                }},
            },
        })
        return err
    })
    // ...
}

// After: drinks/drinks_test.go (module-level)
func TestCreate_PersistsOnCommit(t *testing.T) {
    fix := testutil.NewFixture(t)

    // Bootstrap real ingredient
    lime := fix.Bootstrap().WithIngredient("lime-juice", ingredientsmodels.UnitOz)

    // Use real module API
    created, err := fix.Drinks.Create(fix.Ctx, drinksmodels.Drink{
        Name:     "Margarita",
        Category: drinksmodels.DrinkCategoryCocktail,
        Glass:    drinksmodels.GlassTypeCoupe,
        Recipe: drinksmodels.Recipe{
            Ingredients: []drinksmodels.RecipeIngredient{{
                IngredientID: lime.ID,  // Real!
                Amount:       1.0,
                Unit:         ingredientsmodels.UnitOz,
            }},
            Steps: []string{"Shake with ice"},
        },
    })
    testutil.Ok(t, err)

    // Verify via real query
    got, err := fix.Drinks.Get(fix.Ctx, drinks.GetRequest{ID: created.ID})
    testutil.Ok(t, err)
    testutil.ErrorIf(t, got.Name != "Margarita", "expected Margarita")
}
```

## Tasks

### Phase 1: Remove Test-Only Code from Production

- [x] `drinks/internal/commands/commands.go` - remove `ingredientReader`, use `*ingredientsqueries.Queries`
- [x] `drinks/internal/commands/commands.go` - delete `NewWithDependencies`
- [x] `inventory/internal/commands/commands.go` - remove `ingredientReader`, use `*ingredientsqueries.Queries`
- [x] `inventory/internal/commands/commands.go` - delete `NewWithDependencies`
- [x] `orders/internal/commands/commands.go` - delete `NewWithDependencies`

### Phase 2: Migrate Tests to Module Level

- [x] `drinks/internal/commands/create_test.go` → `drinks/drinks_test.go`
- [x] `drinks/internal/commands/delete_test.go` → `drinks/drinks_test.go`
- [x] `drinks/internal/commands/update_test.go` → `drinks/drinks_test.go`
- [x] `drinks/queries/list_test.go` → `drinks/drinks_test.go`
- [x] `drinks/queries/get_test.go` → `drinks/drinks_test.go`
- [x] `inventory/internal/commands/adjust_test.go` → `inventory/inventory_test.go`
- [x] `menu/internal/commands/create_test.go` → `menu/menu_test.go`
- [x] `ingredients/internal/commands/create_test.go` → `ingredients/ingredients_test.go`
- [x] `orders/internal/commands/place_test.go` → `orders/orders_test.go`

### Phase 3: Enhance Fixture Bootstrap

- [x] Add `Bootstrap().WithIngredient(name, unit)` helper
- [x] Add `Bootstrap().WithDrink(name, ...)` helper
- [x] Add `Bootstrap().WithMenu(name, ...)` helper
- [ ] Add other bootstrap helpers as needed for test setup

### Phase 4: Delete Old Test Files

- [x] Delete `drinks/internal/commands/*_test.go`
- [x] Delete `drinks/queries/*_test.go`
- [x] Delete `inventory/internal/commands/*_test.go`
- [x] Delete `menu/internal/commands/*_test.go`
- [x] Delete `ingredients/internal/commands/*_test.go`
- [x] Delete `orders/internal/commands/*_test.go`

### Phase 5: Verify

- [x] `go test ./...` passes
- [x] No `fakeXxx` types remain
- [x] No `NewWithDependencies` remains
- [x] No test-only interfaces remain

## When Mocking IS Appropriate

- External HTTP APIs
- Third-party services
- Time-sensitive operations (use `clock` interface)
- Non-deterministic behavior

These don't apply to internal module dependencies.

## Acceptance Criteria

- No `NewWithDependencies` constructors
- No interfaces that exist solely for testing
- All module tests use fixture and real code
- Tests bootstrap real data, not fakes
- Kernel unit tests remain (pure functions, isolated logic)
- `go test ./...` passes
