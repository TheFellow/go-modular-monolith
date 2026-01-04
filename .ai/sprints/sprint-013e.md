# Sprint 013e: Remove Command Request/Response Wrappers (Intermezzo)

## Goal

Eliminate unnecessary Request/Response wrapper types in commands. Commands should accept and return public models directly.

## Problem

Commands currently define thin wrapper types that add no value:

```go
// app/drinks/internal/commands/create.go (current - wasteful)
type CreateRequest struct {
    Name        string
    Description string
    Glass       string
    Recipe      models.Recipe
}

type CreateResponse struct {
    Drink models.Drink
}

func (c *Commands) Create(ctx *middleware.Context, req CreateRequest) (CreateResponse, error) {
    // ... create drink ...
    return CreateResponse{Drink: drink}, nil
}
```

The `CreateRequest` is just unpacked `models.Drink` fields. The `CreateResponse` is just a wrapped `models.Drink`. These wrappers:
1. Add boilerplate with no semantic value
2. Require translation to/from public models
3. Create maintenance burden when models change

## Solution

Commands accept/return public models directly:

```go
// app/drinks/internal/commands/create.go (simplified)
func (c *Commands) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
    // Validate, assign ID, save
    drink.ID = uuid.New().String()

    if err := c.dao.Save(ctx, drink); err != nil {
        return models.Drink{}, err
    }

    ctx.AddEvent(events.DrinkCreated{...})
    return drink, nil
}
```

## Where Translation Belongs

Translation happens at two boundaries, neither of which is Request/Response types:

### 1. CLI → Models (surface boundary)

```go
// main/cli/drinks.go
func createDrinkAction(ctx context.Context, cmd *cli.Command) error {
    drink := models.Drink{
        Name:        cmd.String("name"),
        Description: cmd.String("description"),
        Glass:       models.Glass(cmd.String("glass")),
        Recipe:      parseRecipe(cmd.String("recipe")),
    }

    created, err := drinksModule.Create(mwCtx, drink)
    // ...
}
```

### 2. Commands → DAO (persistence boundary)

If the DAO uses different internal types (e.g., for database mapping), translation happens there:

```go
// app/drinks/internal/dao/dao.go
type drinkRow struct {  // internal to DAO
    ID          string `json:"id"`
    Name        string `json:"name"`
    RecipeJSON  string `json:"recipe"`  // serialized differently
}

func (d *DAO) Save(ctx *middleware.Context, drink models.Drink) error {
    row := toRow(drink)  // translation here
    // persist row
}
```

## Tasks

- [ ] Remove Request/Response types from all command files
- [ ] Update command methods to accept/return public models
- [ ] Update Module methods to pass models through
- [ ] Update CLI to construct models directly
- [ ] Verify `go test ./...` passes

## Before/After

### Before (sprint-013d)

```go
// commands/create.go
type CreateRequest struct {
    Name        string
    Description string
    Glass       string
    Recipe      models.Recipe
}

type CreateResponse struct {
    Drink models.Drink
}

func (c *Commands) Create(ctx *middleware.Context, req CreateRequest) (CreateResponse, error)

// module.go
func (m *Module) Create(ctx *middleware.Context, req CreateRequest) (CreateResponse, error) {
    // AuthZ
    return m.commands.Create(ctx, req)
}
```

### After (sprint-013e)

```go
// commands/create.go
func (c *Commands) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error)

// module.go
func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
    // AuthZ
    return m.commands.Create(ctx, drink)
}
```

## Command Signatures

| Command | Before | After |
|---------|--------|-------|
| drinks.Create | `(CreateRequest) (CreateResponse, error)` | `(models.Drink) (models.Drink, error)` |
| drinks.UpdateRecipe | `(UpdateRecipeRequest) error` | `(drinkID string, recipe models.Recipe) error` |
| ingredients.Create | `(CreateRequest) (CreateResponse, error)` | `(models.Ingredient) (models.Ingredient, error)` |
| ingredients.Update | `(UpdateRequest) error` | `(models.Ingredient) error` |
| inventory.Adjust | `(AdjustRequest) error` | `(ingredientID string, delta float64, reason string) error` |
| inventory.Set | `(SetRequest) error` | `(models.Stock) error` |

## When Wrappers Are Justified

Request/Response types make sense when:
1. The command needs fields not on the model (e.g., `reason` for audit)
2. The response includes computed data beyond the model
3. The API is public and needs stability guarantees

For internal commands, these rarely apply. Use simple parameters or models.

## Success Criteria

- No Request/Response wrapper types in commands
- Commands accept public models or simple parameters
- Module signatures match command signatures
- `go test ./...` passes

## Dependencies

- Sprint 013d (Unified Commands object)
