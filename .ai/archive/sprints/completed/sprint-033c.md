# Sprint 033c: Simplify Command Middleware and Module Handlers

## Status

- Started: 2026-01-15
- Completed: 2026-01-15

## Problems

### 1. Update Command Has Bizarre 3-Parameter Signature

```go
func (c *Commands) Update(ctx *middleware.Context, current *models.Drink, drink *models.Drink) (*models.Drink, error)
```

### 2. Events Carry Previous State (Unnecessary)

```go
type DrinkRecipeUpdated struct {
    Previous Drink  // Why?
    Current  Drink
}
```

### 3. Context Storage Anti-Pattern

Input/output entities stored in context via `setInputEntity`/`setOutputEntity` instead of flowing as parameters.

### 4. reflect.DeepEqual in Application Code

```go
if !reflect.DeepEqual(current.Recipe, updated.Recipe) {
    ctx.AddEvent(events.DrinkRecipeUpdated{...})
}
```

### 5. Business Logic in Module Handlers

ID generation and other logic in module handlers instead of commands.

## Solution: KISS

One simple, uniform `RunCommand` function that works for all operations.

### RunCommand Signature

```go
func RunCommand[In, Out CedarEntity](
    ctx *Context,
    action cedar.EntityUID,
    load func(*Context) (In, error),
    execute func(*Context, In) (Out, error),
) (Out, error) {
    err := Command.Execute(ctx, action, func(c *Context) error {
        // 1. Load entity for IN authz
        input, err := load(c)
        if err != nil {
            return err
        }

        // 2. Authorize IN
        if err := authz.AuthorizeWithEntity(c.Principal(), action, input.CedarEntity()); err != nil {
            return err
        }

        // 3. Execute (receives loaded entity)
        result, err := execute(c, input)
        if err != nil {
            return err
        }

        // 4. Authorize OUT
        if err := authz.AuthorizeWithEntity(c.Principal(), action, result.CedarEntity()); err != nil {
            return err
        }

        out = result
        return nil
    })
    return out, err
}
```

### Helper Functions

```go
// FromModel returns a loader that returns the given entity (for Create)
func FromModel[T CedarEntity](entity T) func(*Context) (T, error) {
    return func(*Context) (T, error) { return entity, nil }
}

// ByID returns a loader that fetches by ID (for Update/Delete/Patch)
func ByID[T CedarEntity](
    id cedar.EntityUID,
    get func(context.Context, cedar.EntityUID) (T, error),
) func(*Context) (T, error) {
    return func(ctx *Context) (T, error) { return get(ctx, id) }
}
```

### All CRUD Operations - Same Pattern

```go
// Create
func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionCreate,
        middleware.FromModel(&drink),
        m.commands.Create,
    )
}

// Update
func (m *Module) Update(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionUpdate,
        middleware.ByID(drink.ID, m.queries.Get),
        func(ctx *middleware.Context, _ *models.Drink) (*models.Drink, error) {
            return m.commands.Update(ctx, &drink)
        },
    )
}

// Delete
func (m *Module) Delete(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionDelete,
        middleware.ByID(id, m.queries.Get),
        m.commands.Delete,
    )
}

// Patch (if needed)
func (m *Module) Patch(ctx *middleware.Context, id cedar.EntityUID, changes PatchRequest) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionPatch,
        middleware.ByID(id, m.queries.Get),
        func(ctx *middleware.Context, current *models.Drink) (*models.Drink, error) {
            return m.commands.Patch(ctx, current, changes)
        },
    )
}
```

### Why This Works

| Operation | Load Returns | Execute Receives | Execute Does |
|-----------|--------------|------------------|--------------|
| Create | Input entity | Input entity | Save it |
| Update | Current (for IN authz) | Current (ignored via `_`) | Save desired |
| Delete | Current | Current | Delete it |
| Patch | Current | Current | Merge & save |

Same function, same pattern. Closures capture what they need.

## Principles

### Events Represent Current State

```go
// Before - handler diffs previous vs current
type DrinkRecipeUpdated struct {
    Previous Drink
    Current  Drink
}

// After - handler reacts to current state
type DrinkUpdated struct {
    Drink Drink
}
```

If you need previous state, you don't. That vertical maintains its own state.

### Commands Own Their Logic

- Validation in commands
- ID generation in commands
- Events emitted unconditionally (no reflect.DeepEqual)

### Module Handlers Are Thin Orchestration

- Wire up middleware
- Delegate to commands
- No business logic

## Files to Delete

- `pkg/middleware/command_entities.go` (context storage for entities)

## Files to Modify

- `pkg/middleware/run.go` - new uniform `RunCommand`, add `FromModel`/`ByID` helpers
- `pkg/middleware/command.go` - simplify signature (remove unused `resource` param)
- `pkg/middleware/chains.go` - remove `CommandAuthorize` from chain
- `pkg/middleware/authz.go` - remove `CommandAuthorize` function
- All domain events - remove `Previous` field
- All update commands - single-param signature
- All module handlers - use helpers, no business logic

## Tasks

### Phase 1: Simplify RunCommand

- [x] Update `RunCommand` with uniform signature (load + execute)
- [x] Add `FromModel` helper for Create operations
- [x] Add `ByID` helper for Update/Delete/Patch operations
- [x] Authorization happens in `RunCommand`, not middleware
- [x] Remove `CommandAuthorize` from chain and authz.go
- [x] Delete `pkg/middleware/command_entities.go`

### Phase 2: Simplify Events

- [x] Update `DrinkRecipeUpdated` → `DrinkUpdated` with single `Drink` field
- [x] Audit all events for Previous/Current patterns
- [x] Update handlers that reference `Previous`
- [x] Remove `reflect.DeepEqual` from application code

### Phase 3: Simplify Commands

- [x] Update commands take single model parameter (no `current`)
- [x] Remove business logic from module handlers (ID generation, etc.)
- [x] Commands emit events unconditionally

### Phase 4: Update All Module Handlers

- [x] drinks: Create, Update, Delete using helpers
- [x] ingredients: Create, Update, Delete using helpers
- [x] menu: Create, Update, Delete using helpers
- [x] orders: Create using helpers
- [x] inventory: Update using helpers

### Phase 5: Verify

- [x] Run `go test ./...` and fix any issues
- [x] Verify IN/OUT authorization works correctly

## Acceptance Criteria

- [x] Single `RunCommand` function for all operations
- [x] `FromModel` and `ByID` helpers eliminate boilerplate
- [x] Authorization in `RunCommand`, not middleware
- [x] No context storage for entities
- [x] Events carry current state only
- [x] Commands take single model parameter
- [x] No `reflect.DeepEqual` in application code
- [x] No business logic in module handlers
- [x] All tests pass

## Result

Every module handler follows the same simple pattern:

```go
// Create - direct references
func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionCreate,
        middleware.FromModel(&drink),
        m.commands.Create,
    )
}

// Delete - direct references
func (m *Module) Delete(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionDelete,
        middleware.ByID(id, m.queries.Get),
        m.commands.Delete,
    )
}

// Update - one closure (captures desired state)
func (m *Module) Update(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionUpdate,
        middleware.ByID(drink.ID, m.queries.Get),
        func(ctx *middleware.Context, _ *models.Drink) (*models.Drink, error) {
            return m.commands.Update(ctx, &drink)
        },
    )
}
```

KISS: One function, helper utilities, same pattern everywhere.
