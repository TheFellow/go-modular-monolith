# Sprint 016: Query Cache for Handler Consistency

## Goal

Add a query cache to the middleware context so handlers see consistent "as-of-command" state without artificial pre-fetching.

## Problem

When a command emits events, handlers react. If handlers query state:
1. They might see different state than what existed when the command ran
2. Handler execution order could affect outcomes (inconsistent)
3. Pre-fetching data in the command "just for handlers" feels artificial

## Solution

The execution context IS the world snapshot. Queries made during command execution are cached. Handlers receive the same context, so their queries return cached results.

```
Command executes
    │
    ├─► Queries menu (result cached)
    ├─► Queries stock (result cached)
    ├─► Emits OrderCompleted event
    │
    ▼
Dispatcher runs with SAME context
    │
    ├─► Handler A queries menu → returns CACHED result
    └─► Handler B queries stock → returns CACHED result

All see the same world state. No artificial pre-fetching.
```

## Tasks

- [ ] Add `QueryCache` to `middleware.Context`
- [ ] Create `QueryKey` type for cache keys
- [ ] Create `Cached[T]` wrapper for query methods
- [ ] Update dispatcher to pass context to handlers
- [ ] Update existing queries to use cache-aware pattern
- [ ] Write tests verifying cache consistency

## Implementation

### Query Cache on Context

```go
// pkg/middleware/context.go

type QueryKey string

type QueryCache struct {
    cache map[QueryKey]any
}

func NewQueryCache() *QueryCache {
    return &QueryCache{cache: make(map[QueryKey]any)}
}

func (qc *QueryCache) Get(key QueryKey) (any, bool) {
    v, ok := qc.cache[key]
    return v, ok
}

func (qc *QueryCache) Set(key QueryKey, value any) {
    qc.cache[key] = value
}

type Context struct {
    context.Context
    events     []any
    queryCache *QueryCache
}

func NewContext(parent context.Context, opts ...ContextOpt) *Context {
    c := &Context{
        Context:    parent,
        queryCache: NewQueryCache(),
    }
    // ...
}

func (c *Context) QueryCache() *QueryCache {
    return c.queryCache
}
```

### Cache-Aware Queries

Queries transparently check the cache. No special API needed.

```go
// app/drinks/queries/get.go

func (q *Queries) Get(ctx context.Context, id string) (models.Drink, error) {
    // Check if we're in a middleware context with cache
    if mctx, ok := ctx.(*middleware.Context); ok {
        key := middleware.QueryKey(fmt.Sprintf("drinks:get:%s", id))
        if cached, ok := mctx.QueryCache().Get(key); ok {
            return cached.(models.Drink), nil
        }

        // Query and cache
        result, err := q.dao.Get(ctx, id)
        if err == nil {
            mctx.QueryCache().Set(key, result)
        }
        return result, err
    }

    // Not in middleware context, just query
    return q.dao.Get(ctx, id)
}
```

### Helper for Cleaner Implementation

```go
// pkg/middleware/cache.go

func Cached[T any](ctx context.Context, key string, query func() (T, error)) (T, error) {
    var zero T

    mctx, ok := ctx.(*Context)
    if !ok {
        return query()
    }

    qkey := QueryKey(key)
    if cached, ok := mctx.QueryCache().Get(qkey); ok {
        return cached.(T), nil
    }

    result, err := query()
    if err == nil {
        mctx.QueryCache().Set(qkey, result)
    }
    return result, err
}
```

Usage in queries:

```go
func (q *Queries) Get(ctx context.Context, id string) (models.Drink, error) {
    return middleware.Cached(ctx, fmt.Sprintf("drinks:get:%s", id), func() (models.Drink, error) {
        return q.dao.Get(ctx, id)
    })
}
```

### Dispatcher Passes Context

```go
// pkg/dispatcher/dispatcher.go

func (d *Dispatcher) Flush(ctx *middleware.Context, events []any) error {
    for _, event := range events {
        handlers := d.handlers[reflect.TypeOf(event)]
        for _, h := range handlers {
            // Pass the SAME context - handlers see cached state
            if err := h(ctx, event); err != nil {
                log.Printf("handler error for %T: %v", event, err)
            }
        }
    }
    return nil
}
```

### Handler Signature

```go
// pkg/dispatcher/dispatcher.go

type Handler func(ctx *middleware.Context, event any) error
```

Handlers receive the middleware context, not plain `context.Context`. Their queries use the cache.

## Example Flow

```go
// Command execution
func (c *CompleteOrder) Execute(ctx *middleware.Context, req Request) (*Order, error) {
    // These queries are cached
    menu, _ := c.menuQueries.Get(ctx, req.MenuID)

    for _, item := range req.Items {
        drink, _ := c.drinkQueries.Get(ctx, item.DrinkID)
        // Calculate ingredients from drink.Recipe
    }

    // ... complete order ...

    ctx.AddEvent(events.OrderCompleted{
        OrderID: order.ID,
        MenuID:  req.MenuID,
        Items:   items,
    })
    return order, nil
}

// Handler execution (receives same ctx)
func HandleOrderCompleted(drinkQueries *drinks.Queries, invQueries *inventory.Queries) Handler {
    return func(ctx *middleware.Context, event any) error {
        e := event.(orders.OrderCompleted)

        // Returns CACHED drink from command execution
        // No need for command to pre-fetch "for handlers"
        for _, item := range e.Items {
            drink, _ := drinkQueries.Get(ctx, item.DrinkID)
            // drink.Recipe has ingredients
        }

        // Stock queries during handler are fresh (not cached by command)
        // But that's fine - we're reading current stock to update it
        for _, ingredientID := range ingredientIDs {
            stock, _ := invQueries.GetStock(ctx, ingredientID)
            // update stock...
        }
    }
}
```

## Key Insight

The cache makes queries **idempotent within an execution**. If the command queried it, handlers see the same result. If the command didn't query it, handlers get fresh data.

This is exactly what you want:
- Shared data (menu, drinks, recipes) → cached, consistent
- Handler-specific data (current stock to update) → fresh, current

## Cache Key Strategy

Keys should be deterministic and unique per query:

```go
// Convention: "{module}:{query}:{params...}"
"drinks:get:drk_123"
"drinks:list"
"inventory:stock:ing_456"
"menu:get:menu_789"
```

## Notes

- Cache is per-execution, not global (each command gets fresh cache)
- Cache is read-through (misses query and populate)
- No cache invalidation needed (execution is short-lived)
- Write operations don't use cache (they modify state)

## Success Criteria

- Queries during command execution are cached
- Handler queries return cached results for same keys
- Handler execution order doesn't affect query results
- No artificial pre-fetching in commands
- `go test ./...` passes

## Dependencies

- Sprint 015 (Orders context)
