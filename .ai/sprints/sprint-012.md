# Sprint 012: Event Handlers & Validation

## Goal

Wire up event handlers to validate the dispatcher pattern. Create handlers that react to inventory events, demonstrating cross-context communication without cascading.

## Tasks

- [ ] Create handler registration mechanism
- [ ] Create `app/drinks/handlers/inventory_handlers.go` - react to ingredient events
- [ ] Wire handlers in app initialization
- [ ] Add integration tests for event flows
- [ ] Verify no cascading (handlers don't emit events)
- [ ] Test query cache consistency in handlers

## Handler Pattern: Leaf Nodes Only

**Critical design constraint**: Handlers do NOT emit new events. They are leaf nodes in the event tree.

Handlers can:
- Update their own context's state (via DAO)
- Query other contexts for information (via query cache for consistency)
- Write to logs/audit trails

Handlers cannot:
- Call commands (which would emit events)
- Add events to the context
- Trigger other handlers

## Example Handler

```go
// app/drinks/handlers/inventory_handlers.go
package handlers

import (
    "log"

    "github.com/TheFellow/go-modular-monolith/app/inventory/events"
    "github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func HandleIngredientDepleted() dispatcher.Handler {
    return func(ctx *middleware.Context, event any) error {
        e := event.(events.IngredientDepleted)

        // For now, just log - richer behavior comes with Menu context
        log.Printf("ingredient depleted: %s", e.IngredientID)

        return nil
    }
}

func HandleIngredientRestocked() dispatcher.Handler {
    return func(ctx *middleware.Context, event any) error {
        e := event.(events.IngredientRestocked)

        log.Printf("ingredient restocked: %s (qty: %.2f)", e.IngredientID, e.NewQty)

        return nil
    }
}
```

## Handler Registration

```go
// app/app.go or separate registration file
func registerHandlers(d *dispatcher.Dispatcher) {
    // Inventory events -> Drinks handlers
    d.Register(inventory_events.IngredientDepleted{},
        drinks_handlers.HandleIngredientDepleted())
    d.Register(inventory_events.IngredientRestocked{},
        drinks_handlers.HandleIngredientRestocked())
}
```

## Query Cache Validation Test

```go
func TestHandlerSeesCachedState(t *testing.T) {
    // 1. Command queries ingredient
    // 2. Command emits event
    // 3. Handler queries same ingredient
    // 4. Handler should see cached result from step 1
}
```

## Event Flow Diagram

```
Command: AdjustStock(vodka, -10)
         │
         ├─► Queries ingredient (cached)
         ├─► Updates stock
         ├─► Emits StockAdjusted
         ├─► Emits IngredientDepleted (if qty=0)
         │
         ▼
    ┌─────────┐
    │Dispatcher│  (after commit)
    └─────────┘
         │
    ┌────┴────┐
    ▼         ▼
Handler A  Handler B
    │         │
    ▼         ▼
 (leaf)    (leaf)

No handler emits events.
No chaining. No cycles.
```

## Success Criteria

- Depleting vodka via `inventory adjust` triggers handler
- Handler logs show events were processed
- Handlers see cached query results from command
- No cascading events in logs
- Handler errors are logged but don't fail command
- `go test ./...` passes with integration tests

## Dependencies

- Sprint 010 (Dispatcher & Query Cache)
- Sprint 011 (Inventory context with events)
