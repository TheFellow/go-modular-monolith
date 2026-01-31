# Sprint 033b: Remove Business Logic from Module Handlers

## Status

- Started: 2026-01-15
- Completed: 2026-01-15

## Problem

Module handlers contain business logic that belongs in commands. The module layer should be a thin orchestration layer that:
1. Wires up middleware (authorization, transactions, events)
2. Delegates to commands/queries

It should NOT contain any business logic, validation, or ID generation.

### Current Anti-Pattern

Module handlers generate IDs:

```go
// drinks/create.go
func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionCreate,
        func(*middleware.Context) (*models.Drink, error) {
            toCreate := drink
            if toCreate.ID.Type == "" {
                toCreate.ID = models.NewDrinkID(string(toCreate.ID.ID))  // Business logic!
            }
            return &toCreate, nil
        },
        m.commands.Create,
    )
}
```

This is problematic because:
1. **Duplicated responsibility**: The command already generates IDs (`created.ID = entity.NewDrinkID()`)
2. **Validation confusion**: The command validates `ID.ID` must be empty, but the handler populates it
3. **Inconsistent flow**: Business rules are split across layers

### Same Pattern in Other Domains

```go
// ingredients/create.go
if toCreate.ID.Type == "" {
    toCreate.ID = entity.IngredientID(string(toCreate.ID.ID))
}

// menu/create.go
if toCreate.ID.Type == "" {
    toCreate.ID = models.NewMenuID(string(toCreate.ID.ID))
}
```

## Solution

Module handlers should be pure pass-through to commands. For create operations:

```go
// drinks/create.go - AFTER
func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionCreate,
        func(*middleware.Context) (*models.Drink, error) {
            return &drink, nil  // Just pass through
        },
        m.commands.Create,
    )
}
```

The command is responsible for:
- Validating the input (ID must be empty for create)
- Generating the ID
- All other business logic

## Principle

**Module handlers are orchestration, not logic.**

| Layer | Responsibility |
|-------|---------------|
| Module Handler | Wire up middleware, delegate to command/query |
| Command | Validation, business logic, ID generation, persistence, events |
| Query | Data retrieval, projection |

If you find yourself writing `if` statements or calling domain functions in a module handler, that logic belongs in the command.

## Tasks

### Phase 1: Clean Up Create Handlers

- [x] Remove ID generation logic from `drinks/create.go`
- [x] Remove ID generation logic from `ingredients/create.go`
- [x] Remove ID generation logic from `menu/create.go`
- [x] Audit other create handlers (orders, inventory) for similar patterns

### Phase 2: Verify Commands Handle Everything

- [x] Verify `drinks/internal/commands/create.go` validates and generates ID
- [x] Verify `ingredients/internal/commands/create.go` validates and generates ID
- [x] Verify `menu/internal/commands/create.go` validates and generates ID
- [x] Audit other command create handlers

### Phase 3: Audit Other Module Handlers

- [x] Audit update handlers for misplaced logic
- [x] Audit delete handlers for misplaced logic
- [x] Audit any other module handlers

### Phase 4: Verify

- [x] Run `go test ./...` and fix any issues

## Acceptance Criteria

- [x] Module handlers contain NO business logic (no `if` statements beyond nil checks)
- [x] Module handlers contain NO ID generation
- [x] Module handlers contain NO validation
- [x] All business logic is in commands
- [x] All tests pass

## After Changes

Module create handlers become trivially simple:

```go
func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionCreate,
        func(*middleware.Context) (*models.Drink, error) {
            return &drink, nil
        },
        m.commands.Create,
    )
}
```

All the interesting work happens in the command where it belongs.
