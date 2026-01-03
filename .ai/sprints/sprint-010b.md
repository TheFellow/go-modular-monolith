# Sprint 010b: Event Architecture Refinement (Intermezzo)

## Goal

Address architectural shortcomings discovered during implementation:
1. The entity cache is clumsy to use
2. Threshold events (`IngredientDepleted`) create implicit cascading when handlers modify inventory

## Problems Identified

### Problem 1: Cache Ergonomics

The current `EntityCache` requires handlers to:
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

### Problem 2: Cascading Events via Commands

The sprint-016 design shows handlers updating stock directly. But `IngredientDepleted` is emitted by the `Adjust` command when stock hits zero. This creates a tension:

**Scenario A: Handler calls Adjust command**
```
OrderCompleted → handler → Inventory.Adjust() → IngredientDepleted → more handlers
```
This IS cascading, violating our constraint.

**Scenario B: Handler uses DAO directly (sprint-016 design)**
```
OrderCompleted → handler → stockDAO.Set() → (no events)
```
Stock depletion goes unnoticed by other modules.

Neither option is satisfying:
- Option A breaks the "no cascading" rule
- Option B means some state changes are "silent" - Menu can't react to depletion

## Proposed Solutions

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
- No need for query cache at all (or it becomes optional optimization)
- Event is a complete record of what happened
- Handlers can't accidentally query stale data

**Trade-offs:**
- Events are larger (but still small compared to network traffic)
- Command must know what handlers need (acceptable coupling)
- Event schema becomes richer

### Solution 2: Distinguish Operator Events from System Effects

`IngredientDepleted` means different things in different contexts:
- **Operator scenario**: "We ran out of vodka" - newsworthy, Menu should react
- **Order scenario**: "Making drinks used up the last of the vodka" - consequence, not news

**Proposal**: Split into two concepts:

1. **Threshold events** (operator-initiated): Emitted only when an operator directly adjusts stock
   - `IngredientDepleted` - operator adjusted stock to zero
   - `IngredientRestocked` - operator replenished from zero

2. **Included in fat events** (system effects): Depletion caused by orders is captured in the event itself
   - `OrderCompleted.DepletedIngredients []cedar.EntityUID` - ingredients that hit zero

**This means**:
- `Inventory.Adjust` with `Reason=used` (from order fulfillment) does NOT emit `IngredientDepleted`
- `Inventory.Adjust` with `Reason=spilled/expired/corrected` DOES emit `IngredientDepleted` if threshold crossed
- Handlers for `OrderCompleted` can check `DepletedIngredients` if they care

### Solution 3: Remove Entity Cache

With fat events, the entity cache becomes unnecessary for correctness. Remove it to reduce complexity.

The cache was solving: "How do handlers see consistent state during event dispatch?"

Fat events solve this better: "Events carry the state handlers need, captured at emission time."

## Revised Event Design

### OrderCompleted (from sprint-016)

```go
type OrderCompleted struct {
    OrderID   string
    MenuID    string
    Items     []OrderItemCompleted

    // Enrichment: what the command computed
    IngredientUsage     []IngredientUsage  // What was consumed
    DepletedIngredients []cedar.EntityUID  // Ingredients that hit zero
}

type OrderItemCompleted struct {
    DrinkID  cedar.EntityUID
    Name     string           // Denormalized for logging/audit
    Quantity int
}

type IngredientUsage struct {
    IngredientID cedar.EntityUID
    Name         string   // Denormalized
    Amount       float64
    Unit         string
}
```

### IngredientDepleted (refined)

```go
// Only emitted for OPERATOR-initiated depletion, not order-driven
type IngredientDepleted struct {
    IngredientID cedar.EntityUID
    Name         string  // Denormalized
    Reason       string  // spilled, expired, corrected - NOT "used"
}
```

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
            usage.Name, usage.Amount, e.OrderID)
    }

    // Menu handler can use DepletedIngredients to update availability
    for _, depleted := range e.DepletedIngredients {
        log.Printf("ingredient depleted by order: %s", depleted.ID)
    }

    return nil
}
```

## Tasks

- [x] Document "fat events" pattern in architecture docs
- [x] Remove `EntityCache` from `middleware.Context`
- [x] Remove `pkg/middleware/cache.go`
- [x] Update `Inventory.Adjust` to only emit `IngredientDepleted` for operator actions (not `Reason=used`)
- [x] Update sprint-016 event definitions to use fat events
- [ ] Add `DepletedIngredients` computation to order completion logic (deferred to sprint-016 implementation)

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

## Migration Notes

- Existing events (`DrinkCreated`, `IngredientCreated`, etc.) are already reasonably "fat" - they carry the essential data
- `StockAdjusted` is fat (carries prev/new/delta)
- `IngredientDepleted` is lean (just ID) - but given the change above, it now only fires for operator actions where the ID is sufficient

## Success Criteria

- Entity cache removed from middleware
- `IngredientDepleted` only emitted for operator-initiated adjustments
- Sprint-016 examples updated with fat event pattern
- `go test ./...` passes
- Architecture decision documented

## Dependencies

- Sprint 010 (Dispatcher infrastructure - completed)
- Sprint 011 (Inventory - provides Adjust command to modify)
