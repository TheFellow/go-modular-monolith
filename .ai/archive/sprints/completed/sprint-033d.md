# Sprint 033d: RunCommand Helper Functions

## Status

- Started: 2026-01-15
- Completed: 2026-01-15

## Goal

Eliminate inline closures in module handlers with composable helper functions for common patterns.

## Problem

Current module handlers have inline closures that obscure intent:

```go
func (m *Module) Update(ctx *middleware.Context, drink *models.Drink) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionUpdate,
        func(ctx *middleware.Context) (*models.Drink, error) {
            return m.queries.Get(ctx, drink.ID)
        },
        func(ctx *middleware.Context, _ *models.Drink) (*models.Drink, error) {
            return m.commands.Update(ctx, drink)
        },
    )
}
```

## Solution

Three helper functions for common patterns. Anything beyond these is a custom func.

### Helper Functions

```go
// Entity returns a loader that returns the given entity directly.
// Use for Create operations where there's nothing to load.
func Entity[T CedarEntity](e T) func(*Context) (T, error) {
    return func(*Context) (T, error) { return e, nil }
}

// Get returns a loader that fetches an entity by ID.
// Use for Update/Delete operations that need to load existing state for authz.
func Get[T CedarEntity](
    get func(context.Context, cedar.EntityUID) (T, error),
    id cedar.EntityUID,
) func(*Context) (T, error) {
    return func(ctx *Context) (T, error) { return get(ctx, id) }
}

// Update wraps an execute function to use a specific entity
// instead of the loaded one.
// Use for Update operations where you save desired state, not loaded state.
func Update[In, Out CedarEntity](
    execute func(*Context, In) (Out, error),
    entity In,
) func(*Context, In) (Out, error) {
    return func(ctx *Context, _ In) (Out, error) {
        return execute(ctx, entity)
    }
}
```

### Usage

Module methods take pointers:

```go
// Create - entity provided, execute with it
func (m *Module) Create(ctx *middleware.Context, drink *models.Drink) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionCreate,
        middleware.Entity(drink),
        m.commands.Create,
    )
}

// Update - get current for authz, execute with desired
func (m *Module) Update(ctx *middleware.Context, drink *models.Drink) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionUpdate,
        middleware.Get(m.queries.Get, drink.ID),
        middleware.Update(m.commands.Update, drink),
    )
}

// Delete - get current, execute with it
func (m *Module) Delete(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionDelete,
        middleware.Get(m.queries.Get, id),
        m.commands.Delete,
    )
}

// Patch - custom func, ID is in the request
func (m *Module) Patch(ctx *middleware.Context, patch *PatchRequest) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionPatch,
        middleware.Get(m.queries.Get, patch.ID),
        func(ctx *middleware.Context, current *models.Drink) (*models.Drink, error) {
            return m.commands.Patch(ctx, current, patch)
        },
    )
}
```

### Why These Three

| Helper | Pattern | Use Case |
|--------|---------|----------|
| `Entity(e)` | Return entity directly | Create |
| `Get(fn, id)` | Fetch by ID | Update/Delete/Patch |
| `Update(fn, e)` | Call fn with e, ignore loaded | Update |

Anything else (like Patch needing both current and changes) is a custom func. These helpers cover the common cases; they're syntax sugar, not a framework.

### Argument Order

Consistent pattern: function first, then arguments.

- `Get(m.queries.Get, id)` - function, then ID
- `Update(m.commands.Update, drink)` - function, then entity

## Tasks

- [x] Add `Entity` helper to `pkg/middleware/run.go`
- [x] Add `Get` helper to `pkg/middleware/run.go`
- [x] Add `Update` helper to `pkg/middleware/run.go`
- [x] Update module method signatures to take pointers
- [x] Update drinks module handlers to use helpers
- [x] Update ingredients module handlers to use helpers
- [x] Update menu module handlers to use helpers
- [x] Update orders module handlers to use helpers
- [x] Update inventory module handlers to use helpers
- [x] Run `go test ./...` and fix any issues

## Acceptance Criteria

- [x] `Entity`, `Get`, `Update` helpers implemented
- [x] Module methods take pointers (no `&` needed at call sites)
- [x] Create/Delete have no closures
- [x] Update uses `Update` helper instead of closure
- [x] Custom patterns (like Patch) use inline funcs
- [x] All tests pass

## Result

Clean, readable module handlers:

```go
// Create
middleware.RunCommand(ctx, action, middleware.Entity(drink), m.commands.Create)

// Update
middleware.RunCommand(ctx, action, middleware.Get(m.queries.Get, drink.ID), middleware.Update(m.commands.Update, drink))

// Delete
middleware.RunCommand(ctx, action, middleware.Get(m.queries.Get, id), m.commands.Delete)
```
