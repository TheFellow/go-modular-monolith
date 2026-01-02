# Sprint 010: Dispatcher & Query Cache Infrastructure

## Goal

Implement the core event-driven infrastructure: the dispatcher for routing events to handlers, and the query cache for handler consistency. This validates the architectural pattern early.

## Tasks

- [ ] Implement real `pkg/dispatcher` (replace stub)
- [ ] Add `QueryCache` to `middleware.Context`
- [ ] Create `middleware.Cached[T]` helper for cache-aware queries
- [ ] Update dispatcher to pass `*middleware.Context` to handlers
- [ ] Write tests for dispatcher and cache behavior
- [ ] Wire dispatcher into command middleware (flush events on commit)

## Dispatcher Implementation

```go
// pkg/dispatcher/dispatcher.go
package dispatcher

import (
    "reflect"
    "log"

    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type Handler func(ctx *middleware.Context, event any) error

type Dispatcher struct {
    handlers map[reflect.Type][]Handler
}

func New() *Dispatcher {
    return &Dispatcher{
        handlers: make(map[reflect.Type][]Handler),
    }
}

func (d *Dispatcher) Register(eventType any, handler Handler) {
    t := reflect.TypeOf(eventType)
    d.handlers[t] = append(d.handlers[t], handler)
}

func (d *Dispatcher) Flush(ctx *middleware.Context) error {
    events := ctx.Events()
    for _, event := range events {
        handlers := d.handlers[reflect.TypeOf(event)]
        for _, h := range handlers {
            if err := h(ctx, event); err != nil {
                // Log but don't fail - handlers are best-effort
                log.Printf("handler error for %T: %v", event, err)
            }
        }
    }
    return nil
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
    // ...
    c := &Context{
        Context:    parent,
        events:     make([]any, 0, 4),
        queryCache: newQueryCache(),  // ADD THIS
    }
    // ...
}

func (c *Context) QueryCache() *QueryCache {
    return c.queryCache
}
```

## Command Middleware Integration

```go
// pkg/middleware/uow.go - update to flush events
func (m *UoWMiddleware) Execute(ctx *Context, next func(*Context) error) error {
    tx, err := m.manager.Begin(ctx)
    if err != nil {
        return err
    }

    ctx = NewContext(ctx, WithUnitOfWork(tx))

    if err := next(ctx); err != nil {
        tx.Rollback()
        return err
    }

    if err := tx.Commit(); err != nil {
        return err
    }

    // Flush events after successful commit
    return m.dispatcher.Flush(ctx)
}
```

## Key Design Points

1. **Handlers are leaf nodes** - they don't emit new events
2. **Query cache is per-execution** - fresh cache for each command
3. **Events flush after commit** - ensures data is persisted before handlers run
4. **Handler errors are logged, not fatal** - best-effort processing

## Success Criteria

- Dispatcher routes events to registered handlers
- Query cache returns cached results for repeated queries
- Handler errors are logged but don't fail the command
- `go test ./...` passes

## Dependencies

- Sprint 009 (Ingredients context - provides events to test with)
