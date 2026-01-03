# Sprint 010b: Event Architecture Refinement (Intermezzo)

## Goal

Address architectural shortcomings discovered during implementation:
1. The entity cache is clumsy to use
2. Separate threshold events add complexity without value

## Problems Identified

### Problem 1: Cache Ergonomics

The `EntityCache` requires handlers to:
1. Know which entities they need
2. Call `CacheGet` with the correct `cedar.EntityUID`
3. Handle cache misses by fetching from queries

This creates coupling between handlers and query APIs, and requires handlers to understand caching semantics.

**Current pattern (clumsy):**
```go
func (h *OrderCompletedHandler) Handle(ctx *middleware.Context, e events.OrderCompleted) error {
    for _, item := range e.Items {
        // Handler must query for drink recipe
        drink, err := h.drinkQueries.Get(ctx, item.DrinkID)
        if err != nil {
            return err
        }

        // Then iterate ingredients
        for _, ri := range drink.Recipe.Ingredients {
            stock, err := h.stockDAO.Get(ctx, ri.IngredientID)
            // ... update stock
        }
    }
}
```

### Problem 2: Redundant Threshold Events

Early designs had `StockAdjusted`, `IngredientDepleted`, and `IngredientRestocked` as separate events. But `StockAdjusted` already carries `PreviousQty` and `NewQty` - handlers can derive threshold states:

```go
depleted := e.NewQty == 0
restocked := e.PreviousQty == 0 && e.NewQty > 0
```

Separate threshold events add:
- More types to maintain
- Conditional emission logic in commands
- No additional information

## Solutions

### Solution 1: Fat Events (Event Enrichment)

Events carry all the context handlers need. The command that emits the event queries relevant data upfront and embeds it.

**Before (lean event):**
```go
type OrderCompleted struct {
    OrderID string
    Items   []OrderItemCompleted  // Just DrinkID + Quantity
}
```

**After (fat event):**
```go
type OrderCompleted struct {
    OrderID string
    Items   []OrderItemCompleted

    // Pre-computed by the command:
    IngredientUsage []IngredientUsage
}

type IngredientUsage struct {
    IngredientID cedar.EntityUID
    Amount       float64
    Unit         string
}
```

**Benefits:**
- Handlers become trivially simple - just read from event
- No need for entity cache
- Event is a complete record of what happened
- Handlers can't accidentally query stale data

**Trade-offs:**
- Events are larger (but still small compared to network traffic)
- Command must know what handlers need (acceptable coupling)

### Solution 2: Single Stock Event

Remove `IngredientDepleted` and `IngredientRestocked`. Keep only `StockAdjusted`:

```go
type StockAdjusted struct {
    IngredientID cedar.EntityUID
    PreviousQty  float64
    NewQty       float64
    Delta        float64
    Reason       string
}
```

Handlers derive what they need:

```go
func (h *MenuUpdater) Handle(ctx *middleware.Context, e events.StockAdjusted) error {
    if e.NewQty == 0 {
        // Ingredient depleted - mark unavailable
    }
    if e.PreviousQty == 0 && e.NewQty > 0 {
        // Ingredient restocked - recalculate availability
    }
    // Or: any change might affect availability
    return nil
}
```

### Solution 3: Remove Entity Cache

With fat events, the entity cache becomes unnecessary. Remove it.

The cache was solving: "How do handlers see consistent state during event dispatch?"

Fat events solve this better: "Events carry the state handlers need, captured at emission time."

## Handler Simplification

With fat events, handlers become trivial:

```go
func (h *OrderCompletedHandler) Handle(ctx *middleware.Context, e events.OrderCompleted) error {
    // All data is on the event - no queries needed
    for _, usage := range e.IngredientUsage {
        stock, err := h.stockDAO.Get(ctx, string(usage.IngredientID.ID))
        if err != nil {
            return err
        }

        stock.Quantity -= usage.Amount
        stock.LastUpdated = time.Now()

        if err := h.stockDAO.Set(ctx, stock); err != nil {
            return err
        }

        log.Printf("stock adjusted: %s -= %.2f (order %s)",
            usage.IngredientID, usage.Amount, e.OrderID)
    }
    return nil
}
```

## Tasks

- [x] Document "fat events" pattern in architecture docs
- [x] Remove `EntityCache` from `middleware.Context`
- [x] Remove `pkg/middleware/cache.go`
- [x] Remove `IngredientDepleted` and `IngredientRestocked` events
- [x] Simplify `Inventory.Adjust` to emit only `StockAdjusted`
- [x] Update sprint-011 and sprint-012 to reflect single-event design

## Handler Constraints (Formalized)

Handlers MUST:
1. Only read from the event (fat event pattern)
2. Use DAOs directly for state updates (not commands)
3. NOT emit new events (leaf nodes)
4. NOT call commands

Handlers MAY:
1. Write to logs/audit trails
2. Update their own module's state via DAO
3. Return errors (logged but don't fail the originating command)

## Success Criteria

- Entity cache removed from middleware
- Only `StockAdjusted` event for inventory changes
- Handlers derive threshold states from event data
- `go test ./...` passes
- Architecture decision documented

## Dependencies

- Sprint 010 (Dispatcher infrastructure - completed)
- Sprint 011 (Inventory - provides Adjust command)
