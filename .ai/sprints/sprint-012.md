# Sprint 012: Event Handlers & Validation

## Goal

Wire up event handlers to validate the dispatcher pattern. Create handlers that react to `StockAdjusted` events, demonstrating cross-context communication without cascading.

## Tasks

- [x] Create `app/menu/handlers/stock_adjusted.go` - update menu availability
- [x] Run `go generate ./pkg/dispatcher` to pick up new handlers
- [x] Add integration tests for event flows
- [x] Verify no cascading (handlers don't emit events)
- [x] Verify handlers only read from events (fat event pattern)

## Handler Pattern: Leaf Nodes Only

**Critical design constraint**: Handlers do NOT emit new events. They are leaf nodes in the event tree.

Handlers can:
- Update their own context's state (via DAO)
- Write to logs/audit trails

Handlers cannot:
- Call commands (which would emit events)
- Add events to the context
- Trigger other handlers
- Query other contexts (fat events carry required data)

## Example Handler

Handlers derive threshold states from `StockAdjusted` rather than subscribing to separate events. It is, after all, the business logic of that module to do the thing it needs to do.

```go
// app/menu/handlers/stock_adjusted.go
package handlers

import (
    "log"

    "github.com/TheFellow/go-modular-monolith/app/inventory/events"
    "github.com/TheFellow/go-modular-monolith/app/menu/internal/dao"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type StockAdjustedMenuUpdater struct {
    menuDAO *dao.MenuDAO
}

func NewStockAdjustedMenuUpdater(menuDAO *dao.MenuDAO) *StockAdjustedMenuUpdater {
    return &StockAdjustedMenuUpdater{menuDAO: menuDAO}
}

func (h *StockAdjustedMenuUpdater) Handle(ctx *middleware.Context, e events.StockAdjusted) error {
    // Derive threshold states from the event
    depleted := e.NewQty == 0
    restocked := e.PreviousQty == 0 && e.NewQty > 0

    if depleted {
        log.Printf("ingredient depleted: %s", e.IngredientID)
        // Mark menu items using this ingredient as unavailable
        // h.menuDAO.MarkIngredientUnavailable(ctx, e.IngredientID)
    }

    if restocked {
        log.Printf("ingredient restocked: %s (qty: %.2f)", e.IngredientID, e.NewQty)
        // Recalculate availability for menu items using this ingredient
        // h.menuDAO.RecalculateAvailability(ctx, e.IngredientID)
    }

    // Or simply: any stock change might affect availability
    // h.menuDAO.RecalculateAvailability(ctx, e.IngredientID)

    return nil
}
```

Handlers are discovered by the dispatcher generator by convention:
- Constructor: `New<HandlerName>(...deps)`
- Method: `Handle(ctx *middleware.Context, e events.X) error`

## Event Flow Diagram

```
Command: AdjustStock(vodka, -10)
         │
         ├─► Updates stock
         ├─► Emits StockAdjusted (prev=10, new=0, delta=-10)
         │
         ▼
    ┌─────────┐
    │Dispatcher│  (after commit)
    └─────────┘
         │
         ▼
    Handler: StockAdjustedMenuUpdater
         │
         ├─► Checks: NewQty == 0? → depleted
         ├─► Updates menu availability via DAO
         │
         ▼
      (leaf - no events emitted)
```

## Why One Event, Not Three

Earlier designs had separate `IngredientDepleted` and `IngredientRestocked` events. These were removed because:

1. **Redundant**: `StockAdjusted` already carries `PreviousQty` and `NewQty`
2. **Simpler command**: No conditional event emission logic
3. **Explicit handlers**: Handlers clearly show what threshold logic they care about
4. **Flexible**: Handlers can react to any stock change, not just thresholds

```go
// Handler decides what matters to it:
depleted := e.NewQty == 0
restocked := e.PreviousQty == 0 && e.NewQty > 0
lowStock := e.NewQty > 0 && e.NewQty < 5.0  // Custom threshold
```

## Success Criteria

- Adjusting stock via `inventory adjust` triggers `StockAdjusted` event
- Handler logs show event was processed with derived state
- No cascading events in logs
- Handler errors are logged but don't fail command
- `go test ./...` passes with integration tests

## Dependencies

- Sprint 010 (Dispatcher)
- Sprint 010b (Fat events; no cache)
- Sprint 011 (Inventory context with `StockAdjusted` event)
