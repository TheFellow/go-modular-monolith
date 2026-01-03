# Sprint 010: Dispatcher & Query Cache Infrastructure

## Goal

Implement the core event-driven infrastructure: a code-generated dispatcher that routes events to handlers, and the query cache for handler consistency.

## Tasks

- [ ] Create `pkg/dispatcher/gen.go` - generator that scans events and handlers
- [ ] Add `//go:generate` directive to `pkg/dispatcher/dispatcher.go`
- [ ] Add `QueryCache` to `middleware.Context`
- [ ] Create `middleware.Cached[T]` helper for cache-aware queries
- [ ] Write tests for dispatcher and cache behavior
- [ ] Create example handler to validate the pattern

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
    "github.com/TheFellow/go-modular-monolith/app/drinks/events"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type DrinkDeleted struct {
    // Dependencies injected via constructor
}

func NewDrinkDeleted( /* dependencies */ ) *DrinkDeleted {
    return &DrinkDeleted{ /* ... */ }
}

func (h *DrinkDeleted) Handle(ctx *middleware.Context, event events.DrinkDeleted) error {
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
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
    drinks_events "github.com/TheFellow/go-modular-monolith/app/drinks/events"
    drinks_handlers "github.com/TheFellow/go-modular-monolith/app/drinks/handlers"
    // ... other modules
)

type Dispatcher struct {
    drinkDeleted *drinks_handlers.DrinkDeleted
    // ... other handlers
}

func New(
    drinkDeleted *drinks_handlers.DrinkDeleted,
    // ... other handlers
) *Dispatcher {
    return &Dispatcher{
        drinkDeleted: drinkDeleted,
        // ...
    }
}

func (d *Dispatcher) Dispatch(ctx *middleware.Context, event any) error {
    switch e := event.(type) {
    case drinks_events.DrinkDeleted:
        if d.drinkDeleted != nil {
            return d.drinkDeleted.Handle(ctx, e)
        }
    // ... other cases
    }
    return nil
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

## Query Cache Implementation

```go
// pkg/middleware/cache.go
package middleware

type QueryKey string

type QueryCache struct {
    cache map[QueryKey]any
}

func newQueryCache() *QueryCache {
    return &QueryCache{cache: make(map[QueryKey]any)}
}

func (qc *QueryCache) Get(key QueryKey) (any, bool) {
    v, ok := qc.cache[key]
    return v, ok
}

func (qc *QueryCache) Set(key QueryKey, value any) {
    qc.cache[key] = value
}

// Cached wraps a query function with transparent caching
func Cached[T any](ctx *Context, key string, query func() (T, error)) (T, error) {
    var zero T

    qkey := QueryKey(key)
    if cached, ok := ctx.queryCache.Get(qkey); ok {
        return cached.(T), nil
    }

    result, err := query()
    if err == nil {
        ctx.queryCache.Set(qkey, result)
    }
    return result, err
}
```

## Context Updates

```go
// pkg/middleware/context.go additions
type Context struct {
    context.Context
    events     []any
    queryCache *QueryCache  // ADD THIS
}

func NewContext(parent context.Context, opts ...ContextOpt) *Context {
    c := &Context{
        Context:    parent,
        events:     make([]any, 0, 4),
        queryCache: newQueryCache(),  // ADD THIS
    }
    // ...
}
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
3. **Query cache is per-execution** - Fresh cache for each command
4. **Events dispatch after success** - Ensures data is persisted before handlers run
5. **Handler errors are logged, not fatal** - Best-effort processing

## Success Criteria

- `go generate ./pkg/dispatcher` produces dispatcher_gen.go
- Dispatcher routes events to matching handlers
- Query cache returns cached results for repeated queries
- Handler errors are logged but don't fail the command
- `go test ./...` passes

## Dependencies

- Sprint 009 (Ingredients context - provides events to test with)
