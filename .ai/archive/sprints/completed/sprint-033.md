# Sprint 033: Fine-Grained Authorization with Resource Attributes

## Goal

Enable attribute-based access control (ABAC) by populating Cedar entity attributes from domain models, allowing policies like:

```cedar
permit(
    principal == Mixology::Actor::"sommelier",
    action == Mixology::Drink::Action::"update",
    resource is Mixology::Drink
) when {
    resource.Category == "wine"
};
```

## Status

- Started: 2026-01-15
- Completed: 2026-01-15

## Problem

Current `CedarEntity()` implementations return empty attributes:

```go
func (d Drink) CedarEntity() cedar.Entity {
    return cedar.Entity{
        UID:        d.ID,
        Parents:    cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(nil),  // Empty!
        Tags:       cedar.NewRecord(nil),
    }
}
```

This means Cedar policies cannot access resource attributes like `resource.Category`, `resource.Name`, or any domain-specific fields.

## Solution

Populate `Attributes` in `CedarEntity()` with the model's relevant fields:

```go
func (d Drink) CedarEntity() cedar.Entity {
    return cedar.Entity{
        UID:     d.ID,
        Parents: cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(cedar.RecordMap{
            "Name":        cedar.String(d.Name),
            "Category":    cedar.String(string(d.Category)),
            "Glass":       cedar.String(string(d.Glass)),
            "Description": cedar.String(d.Description),
        }),
        Tags: cedar.NewRecord(nil),
    }
}
```

## Race Condition: TOCTOU

Loading before the transaction creates a Time-of-Check to Time-of-Use vulnerability:

```
Request A                    Request B
─────────                    ─────────
Load drink (Category=wine)
                             Update drink (Category=cocktail)
                             Commit
Authorize (sees wine) ✓
Start transaction
Execute delete              ← Deleting a cocktail, not wine!
```

**Solution**: Authorize inside the transaction, after loading.

## Pattern: Commands Take and Return Full Models

### Core Insight

Commands always take a full model IN and return a full model OUT. The authorization middleware automatically checks both:

- **IN**: Authorize the entity being operated on (before mutation)
- **OUT**: Authorize the resulting entity (after mutation)

This elegantly solves dual-state authorization for updates without special handling.

### Why This Works

The same policy is checked twice—once for input, once for output:

```cedar
permit(
    principal == Mixology::Actor::"sommelier",
    action == Mixology::Drink::Action::"update",
    resource is Mixology::Drink
) when {
    resource.Category == "wine"
};
```

If a sommelier tries to change Category from "wine" to "cocktail":
- **IN check**: Current state is wine ✓ (can modify this drink)
- **OUT check**: Resulting state is cocktail ✗ (cannot create/own this drink)

Authorization fails. The sommelier can only update wine drinks AND the result must still be wine.

### Command Signatures

All commands take the full model and return the full model:

```go
// Commands layer
Create(ctx, Drink) (*Drink, error)   // IN = new drink, OUT = created drink
Update(ctx, Drink) (*Drink, error)   // IN = updated drink, OUT = saved drink
Delete(ctx, Drink) (*Drink, error)   // IN = drink to delete, OUT = deleted drink
```

### Middleware Chain

Move `CommandAuthorize` inside `UnitOfWork`:

```go
// Current (race condition):
var Command = NewCommandChain(
    CommandLogging(),
    CommandMetrics(),
    TrackActivity(),
    CommandAuthorize(),  // Outside transaction!
    UnitOfWork(),
    DispatchEvents(),
)

// Fixed:
var Command = NewCommandChain(
    CommandLogging(),
    CommandMetrics(),
    TrackActivity(),
    UnitOfWork(),        // Transaction starts here
    CommandAuthorize(),  // Now inside transaction - checks IN and OUT
    DispatchEvents(),
)
```

### RunCommand with Loader

`RunCommand` takes a loader function that runs inside the transaction:

```go
// pkg/middleware/run.go

func RunCommand[T CedarEntityProvider, R CedarEntityProvider](
    ctx *Context,
    action cedar.EntityUID,
    load func(ctx *Context) (T, error),
    fn func(ctx *Context, loaded T) (R, error),
) (R, error) {
    return runCommandChain(ctx, action, load, fn)
}
```

The middleware:
1. Calls `load()` inside the transaction
2. Authorizes the loaded entity (IN)
3. Executes `fn()` with the loaded entity
4. Authorizes the returned entity (OUT)
5. Returns the result

### Create Operations

For create, the loader returns the model to be created:

```go
func (m *Module) Create(ctx *middleware.Context, drink Drink) (*Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionCreate,
        func(ctx *middleware.Context) (Drink, error) {
            return drink, nil  // IN = the drink we want to create
        },
        func(ctx *middleware.Context, toCreate Drink) (*Drink, error) {
            return m.commands.Create(ctx, toCreate)  // OUT = the created drink
        },
    )
}
```

Both IN and OUT are essentially the same entity, so policies are consistent.

### Update Operations

For update, the loader fetches the current entity, then the command saves the modified version:

```go
func (m *Module) Update(ctx *middleware.Context, id cedar.EntityUID, changes DrinkChanges) (*Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionUpdate,
        func(ctx *middleware.Context) (Drink, error) {
            // Load current state (IN = what we're authorized to modify)
            return m.queries.Get(ctx, id)
        },
        func(ctx *middleware.Context, current Drink) (*Drink, error) {
            // Apply changes and save
            updated := Drink{
                ID:          current.ID,
                Name:        changes.Name.OrDefault(current.Name),
                Category:    changes.Category.OrDefault(current.Category),
                Glass:       changes.Glass.OrDefault(current.Glass),
                Recipe:      changes.Recipe.OrDefault(current.Recipe),
                Description: changes.Description.OrDefault(current.Description),
            }
            return m.commands.Update(ctx, updated)  // OUT = what we created
        },
    )
}
```

- **IN**: The current drink (loaded from DB) — must be authorized to modify
- **OUT**: The updated drink (returned by command) — must be authorized to own

### Delete Operations

For delete, the loader fetches the entity to delete:

```go
func (m *Module) Delete(ctx *middleware.Context, id cedar.EntityUID) (*Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionDelete,
        func(ctx *middleware.Context) (Drink, error) {
            return m.queries.Get(ctx, id)  // IN = what we're deleting
        },
        func(ctx *middleware.Context, toDelete Drink) (*Drink, error) {
            return m.commands.Delete(ctx, toDelete)  // OUT = what we deleted
        },
    )
}
```

Both IN and OUT are the same entity, so authorization is consistent.

### Authorization Middleware

The middleware authorizes both input and output:

```go
func CommandAuthorize() CommandMiddleware {
    return func(next CommandHandler) CommandHandler {
        return func(ctx *Context) error {
            // Get input entity (set by RunCommand after calling loader)
            input := ctx.InputEntity()

            // Authorize input (current state / what we're operating on)
            if err := authorize(ctx, ctx.Action(), input.CedarEntity()); err != nil {
                return err
            }

            // Execute the command
            if err := next(ctx); err != nil {
                return err
            }

            // Get output entity (set by RunCommand after command execution)
            output := ctx.OutputEntity()

            // Authorize output (resulting state / what we created)
            if err := authorize(ctx, ctx.Action(), output.CedarEntity()); err != nil {
                return err
            }

            return nil
        }
    }
}
```

## Updated Model Pattern

Each model's `CedarEntity()` populates relevant attributes:

### Drink

```go
func (d Drink) CedarEntity() cedar.Entity {
    return cedar.Entity{
        UID:     d.ID,
        Parents: cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(cedar.RecordMap{
            "Name":        cedar.String(d.Name),
            "Category":    cedar.String(string(d.Category)),
            "Glass":       cedar.String(string(d.Glass)),
            "Description": cedar.String(d.Description),
        }),
        Tags: cedar.NewRecord(nil),
    }
}
```

### Ingredient

```go
func (i Ingredient) CedarEntity() cedar.Entity {
    return cedar.Entity{
        UID:     i.ID,
        Parents: cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(cedar.RecordMap{
            "Name":     cedar.String(i.Name),
            "Category": cedar.String(string(i.Category)),
            "Unit":     cedar.String(string(i.Unit)),
        }),
        Tags: cedar.NewRecord(nil),
    }
}
```

### Menu

```go
func (m Menu) CedarEntity() cedar.Entity {
    return cedar.Entity{
        UID:     m.ID,
        Parents: cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(cedar.RecordMap{
            "Name":   cedar.String(m.Name),
            "Status": cedar.String(string(m.Status)),
        }),
        Tags: cedar.NewRecord(nil),
    }
}
```

## Example Policies

With attributes populated and dual IN/OUT authorization, these policies work naturally:

```cedar
// Only sommelier can manage wine drinks
// (checked on IN and OUT, so they can only create/update TO wine as well)
permit(
    principal == Mixology::Actor::"sommelier",
    action in [
        Mixology::Drink::Action::"create",
        Mixology::Drink::Action::"update",
        Mixology::Drink::Action::"delete"
    ],
    resource is Mixology::Drink
) when {
    resource.Category == "wine"
};

// Bartenders can only modify cocktails
// (IN must be cocktail, OUT must be cocktail)
permit(
    principal == Mixology::Actor::"bartender",
    action == Mixology::Drink::Action::"update",
    resource is Mixology::Drink
) when {
    resource.Category == "cocktail"
};

// Only owner can delete published menus
permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Menu::Action::"delete",
    resource is Mixology::Menu
) when {
    resource.Status == "published"
};

// Owner can change any drink to any category
// (no attribute constraints, so IN and OUT both pass)
permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Drink::Action::"update",
    resource is Mixology::Drink
);
```

### How Dual Authorization Works in Practice

**Scenario**: Sommelier tries to change a wine drink to a cocktail.

Policy:
```cedar
permit(
    principal == Mixology::Actor::"sommelier",
    action == Mixology::Drink::Action::"update",
    resource is Mixology::Drink
) when {
    resource.Category == "wine"
};
```

Authorization flow:
1. **IN check**: resource = current drink (Category="wine") → ✓ Policy matches
2. Execute update: change Category to "cocktail"
3. **OUT check**: resource = updated drink (Category="cocktail") → ✗ Policy doesn't match

Result: **Forbidden**. The sommelier cannot change wine to cocktail.

**Scenario**: Owner changes a wine drink to a cocktail.

Policy:
```cedar
permit(
    principal == Mixology::Actor::"owner",
    action == Mixology::Drink::Action::"update",
    resource is Mixology::Drink
);
```

Authorization flow:
1. **IN check**: resource = current drink → ✓ Policy matches (no constraints)
2. Execute update: change Category to "cocktail"
3. **OUT check**: resource = updated drink → ✓ Policy matches (no constraints)

Result: **Allowed**. The owner can change any drink to any category.

## Tasks

### Phase 1: Middleware Changes

- [x] Reorder command chain: move `CommandAuthorize` inside `UnitOfWork`
- [x] Update `RunCommand` signature to take loader and pass loaded entity to fn
- [x] Update `CommandAuthorize` to check both IN (before) and OUT (after)
- [x] Add `InputEntity()` and `OutputEntity()` to Context

### Phase 2: Model Changes

- [x] Update `Drink.CedarEntity()` to populate attributes
- [x] Update `Ingredient.CedarEntity()` to populate attributes
- [x] Update `Menu.CedarEntity()` to populate attributes
- [x] Update `Order.CedarEntity()` to populate attributes
- [x] Update `Inventory.CedarEntity()` to populate attributes

### Phase 3: Command Layer Changes

- [x] Update drinks commands to take/return full models
- [x] Update ingredients commands to take/return full models
- [x] Update menu commands to take/return full models
- [x] Update orders commands to take/return full models
- [x] Update inventory commands to take/return full models

### Phase 4: Module Entry Points

- [x] Update drinks module to use loader pattern
- [x] Update ingredients module to use loader pattern
- [x] Update menu module to use loader pattern
- [x] Update orders module to use loader pattern
- [x] Update inventory module to use loader pattern

### Phase 5: Example Policies

- [x] Add example ABAC policy to drinks authz
- [x] Test that attribute-based policy works
- [x] Test dual IN/OUT authorization for updates

### Phase 6: Testing

- [x] Test authorization happens inside transaction
- [x] Test authorization with populated attributes
- [x] Test policy that uses `resource.Category`
- [x] Test sommelier cannot change wine to cocktail
- [x] Test owner can change any drink category
- [x] Verify `go test ./...` passes

## Acceptance Criteria

- [x] Authorization happens inside the transaction (no TOCTOU race)
- [x] `CedarEntity()` returns populated attributes for all domain models
- [x] Commands take full model IN and return full model OUT
- [x] `RunCommand` uses loader function pattern
- [x] Authorization middleware checks both IN and OUT automatically
- [x] Same policy naturally enforces both current and resulting state
- [x] Example ABAC policy demonstrates dual authorization
- [x] All tests pass
