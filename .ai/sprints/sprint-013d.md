# Sprint 013d: Unified Commands Object (Intermezzo)

## Goal

Mirror the Queries pattern for Commands: a single `Commands` object per module that encapsulates all command operations and shares one DAO instance.

## Problem

Sprint 013c introduced `New()` constructors, but created asymmetry:

```go
// Queries: single object, single DAO
type Queries struct {
    dao *dao.DAO
}

func New() *Queries {
    return &Queries{dao: dao.New()}
}

func (q *Queries) Get(ctx, id) (Drink, error)
func (q *Queries) List(ctx) ([]Drink, error)

// Commands: multiple objects, multiple DAOs (wasteful)
type Create struct {
    dao         *dao.DAO  // DAO instance 1
    ingredients *ingredientsq.Queries
}

type UpdateRecipe struct {
    dao         *dao.DAO  // DAO instance 2 (duplicate!)
    ingredients *ingredientsq.Queries
}
```

This creates:
1. Multiple DAO instances per module (wasteful)
2. Asymmetric API between queries and commands
3. More constructor wiring in Module

## Solution

Single `Commands` object per module. The struct and constructor live in `commands.go`, while individual methods live in separate files:

```go
// app/drinks/internal/commands/commands.go
package commands

import (
    "github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
    ingredientsq "github.com/TheFellow/go-modular-monolith/app/ingredients/queries"
)

type Commands struct {
    dao         *dao.DAO
    ingredients *ingredientsq.Queries
}

func New() *Commands {
    return &Commands{
        dao:         dao.New(),
        ingredients: ingredientsq.New(),
    }
}
```

```go
// app/drinks/internal/commands/create.go
package commands

func (c *Commands) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
    // implementation
}
```

```go
// app/drinks/internal/commands/update_recipe.go
package commands

func (c *Commands) UpdateRecipe(ctx *middleware.Context, drinkID string, recipe models.Recipe) error {
    // implementation
}
```

## Tasks

- [x] Create `commands/commands.go` with `Commands` struct and `New()` for each module
- [x] Update individual command files to define methods on `Commands` instead of separate structs
- [x] Update modules to use single `commands.New()`
- [x] Remove individual command constructors (keep methods in their own files)
- [x] Verify `go test ./...` passes

## Before/After Comparison

### Before (sprint-013c)

```go
// app/drinks/module.go
type Module struct {
    queries   *queries.Queries
    createCmd *commands.Create       // separate object
    updateCmd *commands.UpdateRecipe // separate object
}

func NewModule() *Module {
    return &Module{
        queries:   queries.New(),
        createCmd: commands.NewCreate(),      // creates DAO
        updateCmd: commands.NewUpdateRecipe(), // creates another DAO
    }
}
```

### After (sprint-013d)

```go
// app/drinks/module.go
type Module struct {
    queries  *queries.Queries
    commands *commands.Commands  // single object
}

func NewModule() *Module {
    return &Module{
        queries:  queries.New(),
        commands: commands.New(),  // single DAO
    }
}

func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
    // AuthZ check
    return m.commands.Create(ctx, drink)
}

func (m *Module) UpdateRecipe(ctx *middleware.Context, drinkID string, recipe models.Recipe) error {
    // AuthZ check
    return m.commands.UpdateRecipe(ctx, drinkID, recipe)
}
```

## Package Structure

```
app/drinks/
├── module.go              # Public API with AuthZ
├── queries/
│   └── queries.go         # Queries struct + New() + all read methods
├── internal/
│   ├── commands/
│   │   ├── commands.go    # Commands struct + New() only
│   │   ├── create.go      # Create method on Commands
│   │   └── update_recipe.go # UpdateRecipe method on Commands
│   └── dao/
│       └── dao.go         # Single DAO implementation
```

## Pattern Summary

| Layer | Constructor | Contains |
|-------|-------------|----------|
| Module | `NewModule()` | queries + commands, adds AuthZ |
| Queries | `queries.New()` | single DAO, read methods |
| Commands | `commands.New()` | single DAO + cross-module queries, write methods |
| DAO | `dao.New()` | hardcoded path (temporary) |

## Example: Drinks Module

```go
// app/drinks/internal/commands/commands.go
package commands

import (
    "github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
    ingredientsq "github.com/TheFellow/go-modular-monolith/app/ingredients/queries"
)

type Commands struct {
    dao         *dao.DAO
    ingredients *ingredientsq.Queries
}

func New() *Commands {
    return &Commands{
        dao:         dao.New(),
        ingredients: ingredientsq.New(),
    }
}
```

```go
// app/drinks/internal/commands/create.go
package commands

// Note: Uses public models directly, no Request/Response wrappers (see sprint-013e)

func (c *Commands) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
    // Validate ingredients exist
    for _, ri := range drink.Recipe.Ingredients {
        if _, err := c.ingredients.Get(ctx, string(ri.IngredientID.ID)); err != nil {
            return models.Drink{}, errors.Wrap(err, "invalid ingredient")
        }
    }

    drink.ID = uuid.New().String()

    if err := c.dao.Save(ctx, drink); err != nil {
        return models.Drink{}, err
    }

    ctx.AddEvent(events.DrinkCreated{
        DrinkID: cedar.NewEntityUID("Mixology::Drink", drink.ID),
        Name:    drink.Name,
    })

    return drink, nil
}
```

```go
// app/drinks/internal/commands/update_recipe.go
package commands

func (c *Commands) UpdateRecipe(ctx *middleware.Context, drinkID string, recipe models.Recipe) error {
    // implementation
}
```

## Success Criteria

- Each module has exactly one `Commands` object
- Each module has exactly one `Queries` object
- DAOs are effectively singletons within their module
- Symmetric API: `queries.New()` and `commands.New()`
- `go test ./...` passes

## Dependencies

- Sprint 013c (simplified constructors)
