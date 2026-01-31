# Sprint 010: Dispatcher & Query Cache Infrastructure

## Goal

Implement the core event-driven infrastructure: a code-generated dispatcher that routes events to handlers, and per-execution caching so handlers see consistent read results within a command execution.

**Update**: The per-execution entity cache was removed in Sprint 010b in favor of fat events (handlers read only from events; no query/cache coupling).

## Tasks

- [x] Create `pkg/dispatcher/gen.go` - generator that scans events and handlers
- [x] Add `//go:generate` directive to `pkg/dispatcher/dispatcher.go`
- [x] Add per-execution entity cache to `middleware.Context` (removed in Sprint 010b)
- [x] Provide cache primitives keyed by `cedar.EntityUID` (removed in Sprint 010b)
- [x] Write tests for dispatcher and cache behavior (removed in Sprint 010b)
- [x] Create example handler to validate the pattern

## Generated Dispatcher Design

The dispatcher is generated via `go generate`. It:
1. Scans `app/*/events/*.go` for event types
2. Scans `app/*/handlers/*.go` for handler structs
3. Matches handlers to events by inspecting the event type argument of the Handle method
4. Generates a type switch in `Dispatch()`
   1. Note: There may be many handlers for any one event type. Ensure code generation sorts the events and handlers first to get deterministic code generation.

### Handler Conventions

```go
// app/drinks/handlers/drink_deleted.go
package handlers

import (
    "context"

    "github.com/TheFellow/go-modular-monolith/app/drinks/events"
)

type DrinkDeleted struct {
    // Dependencies injected via constructor
}

// Required by the generator: parameterless constructor.
func NewDrinkDeleted() *DrinkDeleted {
    return &DrinkDeleted{ /* ... */ }
}

func (h *DrinkDeleted) Handle(ctx context.Context, event events.DrinkDeleted) error {
    // React to event - update state, log, etc.
    // Does NOT emit new events (leaf node)
    return nil
}
```

### Generated Dispatcher

```go
// pkg/dispatcher/dispatcher_gen.go (GENERATED - DO NOT EDIT)
package dispatcher

import (
    drinks_events "github.com/TheFellow/go-modular-monolith/app/drinks/events"
    drinks_handlers "github.com/TheFellow/go-modular-monolith/app/drinks/handlers"
    // ... other modules
)

type Dispatcher struct {
    // note: no stored handler references; constructed per dispatch call
}

func New() *Dispatcher { return &Dispatcher{} }

// Hook points (implemented in dispatcher.go)
func (d *Dispatcher) handlerError(ctx context.Context, event any, err error) error
func (d *Dispatcher) unhandledEvent(ctx context.Context, event any) error

func (d *Dispatcher) Dispatch(ctx context.Context, event any) error {
    switch e := event.(type) {
    case drinks_events.DrinkDeleted:
        if err := drinks_handlers.NewDrinkDeleted().Handle(ctx, e); err != nil {
            if herr := d.handlerError(ctx, e, err); herr != nil {
                return herr
            }
        }
        return nil
    default:
        return d.unhandledEvent(ctx, event)
    }
}
```

### Generator Implementation

```go
// pkg/dispatcher/gen.go
//go:build ignore

package main

import (
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "path/filepath"
    "text/template"
)

func main() {
    // 1. Find all event types in app/*/events/
    // 2. Find all handler structs in app/*/handlers/
    // 3. Match by handler argument type matching event type
    // 4. Generate dispatcher_gen.go
}
```

## Entity Cache Implementation

**Deprecated**: This cache was removed in Sprint 010b; the current architecture uses fat events instead.

```go
// pkg/middleware/cache.go
package middleware

// Cache is per-execution and keyed by cedar.EntityUID.
// DAOs populate it automatically so queries stay thin.
type EntityCache struct{ /* ... */ }

func (c *EntityCache) Set(entity CedarEntity)
func (c *EntityCache) Get(uid cedar.EntityUID) (any, bool)
```

## Context Updates

```go
// pkg/middleware/context.go additions
type Context struct {
    context.Context
    events []any
    cache  *EntityCache
}

func NewContext(parent context.Context, opts ...ContextOpt) *Context {
    c := &Context{
        Context:    parent,
        events:     make([]any, 0, 4),
        cache:      newCache(),
    }
    // ...
}

func (c *Context) Cache() *EntityCache { return c.cache }
```

## Middleware Integration

The existing dispatcher middleware calls `d.Dispatch()` after the command succeeds:

```go
// pkg/middleware/dispatcher.go - update signature
func Dispatcher(d *dispatcher.Dispatcher) CommandMiddleware {
    return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
        if err := next(ctx); err != nil {
            return err
        }
        // Dispatch each event
        for _, event := range ctx.Events() {
            if err := d.Dispatch(ctx, event); err != nil {
                log.Printf("handler error for %T: %v", event, err)
            }
        }
        return nil
    }
}
```

## Key Design Points

1. **Code generation** - No manual wiring, just follow naming conventions
2. **Handlers are leaf nodes** - They don't emit new events
3. **Entity cache is per-execution** - Fresh cache for each command
4. **Events dispatch after success** - Ensures data is persisted before handlers run
5. **Handler errors are logged, not fatal** - Best-effort processing

## Success Criteria

- `go generate ./pkg/dispatcher` produces dispatcher_gen.go
- Dispatcher routes events to matching handlers
- DAOs populate the per-execution entity cache by `cedar.EntityUID`
- Handler errors are logged but don't fail the command
- `go test ./...` passes

## Dependencies

- Sprint 009 (Ingredients context - provides events to test with)

## Status

✅ Completed
