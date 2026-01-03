# Sprint 012: Event Handlers & Validation

## Goal

Wire up event handlers to validate the dispatcher pattern. Create handlers that react to inventory events, demonstrating cross-context communication without cascading.

## Tasks

- [ ] Create `app/drinks/handlers/inventory_handlers.go` - react to ingredient events
- [ ] Run `go generate ./pkg/dispatcher` to pick up new handlers
- [ ] Add integration tests for event flows
- [ ] Verify no cascading (handlers don't emit events)
- [ ] Verify handlers only read from events

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

```go
// app/drinks/handlers/inventory_handlers.go
package handlers

import (
    "log"

    "github.com/TheFellow/go-modular-monolith/app/inventory/events"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type IngredientDepletedLogger struct{}

func NewIngredientDepletedLogger() *IngredientDepletedLogger { return &IngredientDepletedLogger{} }

func (h *IngredientDepletedLogger) Handle(ctx *middleware.Context, e events.IngredientDepleted) error {
    _ = ctx
    log.Printf("ingredient depleted: %s", e.IngredientID)
    return nil
}

type IngredientRestockedLogger struct{}

func NewIngredientRestockedLogger() *IngredientRestockedLogger { return &IngredientRestockedLogger{} }

func (h *IngredientRestockedLogger) Handle(ctx *middleware.Context, e events.IngredientRestocked) error {
    _ = ctx
    log.Printf("ingredient restocked: %s (qty: %.2f)", e.IngredientID, e.NewQty)
    return nil
}
```

Handlers are discovered by the dispatcher generator by convention (`New<HandlerName>()` + `Handle(ctx *middleware.Context, e events.X) error`).

## Event Flow Diagram

```
Command: AdjustStock(vodka, -10)
         │
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
- No cascading events in logs
- Handler errors are logged but don't fail command
- `go test ./...` passes with integration tests

## Dependencies

- Sprint 010 (Dispatcher)
- Sprint 010b (Fat events; no cache)
- Sprint 011 (Inventory context with events)
