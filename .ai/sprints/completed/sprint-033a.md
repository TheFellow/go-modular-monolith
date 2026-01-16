# Sprint 033a: Simplify Command Middleware Patterns

## Status

- Started: 2026-01-15
- Completed: 2026-01-15

## Problem

The current `RunCommand` implementation has several issues:

### 1. Value vs Pointer Mismatch

Commands and queries return pointers (`*Drink`), but the middleware expects value types:

```go
// Current - awkward dereferencing
func (m *Module) Update(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionUpdate,
        func(ctx *middleware.Context) (models.Drink, error) {
            current, err := m.queries.Get(ctx, drink.ID)  // Returns *Drink
            if err != nil {
                return models.Drink{}, err
            }
            return *current, nil  // Awkward dereference
        },
        // ...
    )
}
```

This prevents using `m.queries.Get` directly as a loader reference.

### 2. Context Storage Anti-Pattern

Input and output entities are stored in context via `setInputEntity`/`setOutputEntity`:

```go
// command_entities.go
func (c *Context) setInputEntity(entity CedarEntity) {
    c.Context = context.WithValue(c.Context, commandInputKey{}, entity)
}

func (c *Context) setOutputEntity(entity CedarEntity) {
    c.Context = context.WithValue(c.Context, commandOutputKey{}, entity)
}
```

These are actual parameters and return values—they should flow through the call stack, not be stored in context.

### 3. reflect.DeepEqual Usage

`reflect.DeepEqual` is used to conditionally emit events:

```go
// drinks/update.go
if !reflect.DeepEqual(current.Recipe, updated.Recipe) {
    ctx.AddEvent(events.DrinkRecipeUpdated{...})
}
```

Reflection is slow and shouldn't be used in application code. Commands should emit events unconditionally—handlers decide what to do.

### 4. Loader Stored in Context

The command loader is stored in context and retrieved by the authz middleware:

```go
func WithCommandLoader(loader commandLoader) ContextOpt {
    c.Context = context.WithValue(c.Context, commandLoaderKey{}, loader)
}
```

This indirection makes the flow hard to follow.

## Solution

### Authorization in RunCommand, Not Middleware

Move authorization logic out of the middleware chain and into `RunCommand` itself. Authorization isn't a cross-cutting concern like logging or transactions—it's specific to the command operation and needs typed access to input/output.

**Current flow (convoluted):**
```
RunCommand
  └─> stores loader in context
  └─> Command.Execute
        └─> CommandAuthorize middleware
              └─> retrieves loader from context
              └─> calls loader, stores input in context
              └─> authorizes input
              └─> calls next
              └─> retrieves output from context
              └─> authorizes output
        └─> final handler
              └─> retrieves input from context
              └─> calls execute
              └─> stores output in context
```

**Proposed flow (direct):**
```
RunCommand
  └─> Command.Execute (logging, metrics, UoW, events)
        └─> final handler
              └─> calls load(), gets *Input
              └─> authorizes input
              └─> calls execute(input), gets *Output
              └─> authorizes output
              └─> returns output
```

### Updated RunCommand

```go
func RunCommand[In, Out CedarEntity](
    ctx *Context,
    action cedar.EntityUID,
    load func(*Context) (In, error),
    execute func(*Context, In) (Out, error),
) (Out, error) {
    var out Out

    err := Command.Execute(ctx, action, func(c *Context) error {
        // Load (inside transaction)
        input, err := load(c)
        if err != nil {
            return err
        }

        // Authorize IN
        if err := authz.AuthorizeWithEntity(c.Principal(), action, input.CedarEntity()); err != nil {
            return err
        }

        // Execute
        output, err := execute(c, input)
        if err != nil {
            return err
        }

        // Authorize OUT
        if err := authz.AuthorizeWithEntity(c.Principal(), action, output.CedarEntity()); err != nil {
            return err
        }

        out = output
        return nil
    })
    return out, err
}
```

### Pointer Types Throughout

With this approach, the type constraints work with pointers:

```go
func RunCommand[In, Out CedarEntity](
    ctx *Context,
    action cedar.EntityUID,
    load func(*Context) (In, error),      // In = *Drink
    execute func(*Context, In) (Out, error), // Out = *Drink
) (Out, error)
```

Module methods become cleaner:

```go
func (m *Module) Delete(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionDelete,
        func(ctx *middleware.Context) (*models.Drink, error) {
            return m.queries.Get(ctx, id)  // Direct call, no dereference
        },
        m.commands.Delete,  // Can pass method reference directly
    )
}
```

### Simplified Middleware Chain

Remove `CommandAuthorize` from the chain:

```go
// Current
var Command = NewCommandChain(
    CommandLogging(),
    CommandMetrics(),
    TrackActivity(),
    UnitOfWork(),
    CommandAuthorize(),  // Remove this
    DispatchEvents(),
)

// Simplified
var Command = NewCommandChain(
    CommandLogging(),
    CommandMetrics(),
    TrackActivity(),
    UnitOfWork(),
    DispatchEvents(),
)
```

### Simplified CommandChain Signature

The chain no longer needs to pass `resource`:

```go
// Current
type CommandNext func(*Context) error
type CommandMiddleware func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error

// Simplified
type CommandNext func(*Context) error
type CommandMiddleware func(ctx *Context, action cedar.EntityUID, next CommandNext) error
```

### Remove reflect.DeepEqual

Commands emit events unconditionally:

```go
// Before
if !reflect.DeepEqual(current.Recipe, updated.Recipe) {
    ctx.AddEvent(events.DrinkRecipeUpdated{...})
}

// After
ctx.AddEvent(events.DrinkUpdated{
    Previous: *current,
    Current:  *updated,
})
```

If a handler needs to check whether something actually changed, it can compare the relevant fields itself. If that comparison is expensive, use custom event types that carry only the relevant data.

## Files to Remove

- `pkg/middleware/command_entities.go` - all of it (loader/input/output context storage)

## Files to Modify

- `pkg/middleware/run.go` - inline authorization in `RunCommand`
- `pkg/middleware/command.go` - simplify signature (remove `resource` parameter)
- `pkg/middleware/chains.go` - remove `CommandAuthorize` from chain
- `pkg/middleware/authz.go` - remove `CommandAuthorize` function
- `app/domains/drinks/update.go` - remove reflect.DeepEqual, emit event unconditionally
- All module entry points - update to use pointer types consistently

## Tasks

### Phase 1: Simplify RunCommand

- [x] Update `RunCommand` to inline authorization (load → authz IN → execute → authz OUT)
- [x] Change type constraints to work with pointer types
- [x] Remove `CommandAuthorize` from middleware chain
- [x] Simplify `CommandMiddleware` signature (remove `resource` parameter)

### Phase 2: Clean Up Context Storage

- [x] Delete `pkg/middleware/command_entities.go`
- [x] Remove `WithCommandLoader`, `commandLoaderFromContext`
- [x] Remove `setInputEntity`, `InputEntity`, `setOutputEntity`, `OutputEntity`
- [x] Remove `CommandAuthorize` function from `authz.go`

### Phase 3: Remove reflect.DeepEqual

- [x] Update `drinks/update.go` to emit events unconditionally
- [x] Remove `reflect` import from domain code
- [x] Audit for any other reflect.DeepEqual usage in application code

### Phase 4: Update Module Entry Points

- [x] Update drinks module to use pointer types in loaders
- [x] Update ingredients module to use pointer types in loaders
- [x] Update menu module to use pointer types in loaders
- [x] Update orders module to use pointer types in loaders
- [x] Update inventory module to use pointer types in loaders

### Phase 5: Verify

- [x] Run `go test ./...` and fix any issues
- [x] Verify authorization still works correctly (IN and OUT checks)

## Acceptance Criteria

- [x] `RunCommand` handles authorization inline, not via middleware
- [x] No context storage for input/output entities
- [x] Pointer types work naturally (`*Drink`, not `Drink`)
- [x] Can pass `m.queries.Get` style references without wrapper closures (where signatures match)
- [x] No `reflect.DeepEqual` in application code
- [x] Commands emit events unconditionally
- [x] `pkg/middleware/command_entities.go` deleted
- [x] All tests pass
