# Sprint 016: Orders & Inventory Consumption

## Goal

Create an Orders context that records drink orders. Handlers in other contexts react to order events but do not produce new events (no cascading).

## Tasks

- [x] Create `app/domains/orders/models/order.go` with Order, OrderItem models
- [x] Create `app/domains/orders/internal/dao/dao.go` with file-based DAO
- [x] Create `app/domains/orders/authz/` with actions and policies
- [x] Create `app/domains/orders/queries/queries.go` with Get, List methods
- [x] Create `app/domains/orders/internal/commands/commands.go` with Place, Complete, Cancel methods
- [x] Create `app/domains/orders/events/` with order events
- [x] Create `app/domains/inventory/handlers/order_completed.go` - updates stock directly (no events)
- [x] Create `app/domains/menu/handlers/order_completed.go` - recalculates availability directly (no events)
- [x] Add order subcommands to CLI

## Domain Model

```go
type Order struct {
    ID          string
    MenuID      string
    Items       []OrderItem
    Status      OrderStatus
    CreatedAt   time.Time
    CompletedAt *time.Time
    Notes       string
}

type OrderItem struct {
    DrinkID       string
    Quantity      int
    Substitutions []AppliedSubstitution
    Notes         string
}

type OrderStatus string
const (
    OrderStatusPending    OrderStatus = "pending"
    OrderStatusPreparing  OrderStatus = "preparing"
    OrderStatusCompleted  OrderStatus = "completed"
    OrderStatusCancelled  OrderStatus = "cancelled"
)
```

## Events

```go
type OrderCompleted struct {
    OrderID cedar.EntityUID
    MenuID  cedar.EntityUID
    Items   []OrderItemCompleted

    // Enrichment computed by the command at completion time.
    IngredientUsage     []IngredientUsage
    DepletedIngredients []cedar.EntityUID
}

type OrderItemCompleted struct {
    DrinkID  cedar.EntityUID
    Name     string
    Quantity int
}

type IngredientUsage struct {
    IngredientID cedar.EntityUID
    Name         string
    Amount       float64
    Unit         string
}
```

Events are intentionally fat: handlers only read from the event and never query/compute business logic.

## Handler Pattern: No Cascading Events

Handlers react to events but **do not emit new events**. They update their own state directly.

```go
// app/domains/inventory/handlers/order_completed.go
package handlers

import (
    "time"

    "github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
    "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type OrderCompletedStockUpdater struct {
    stockDAO *dao.DAO
}

func New() *OrderCompletedStockUpdater {
    return &OrderCompletedStockUpdater{
        stockDAO: dao.New(),
    }
}

func (h *OrderCompletedStockUpdater) Handle(ctx *middleware.Context, e events.OrderCompleted) error {
    for _, usage := range e.IngredientUsage {
        stock, err := h.stockDAO.Get(ctx, string(usage.IngredientID.ID))
        if err != nil {
            return err
        }

        stock.Quantity -= usage.Amount
        stock.LastUpdated = time.Now()

        if err := h.stockDAO.Save(ctx, stock); err != nil {
            return err
        }
    }
    return nil
}
```

## Why No Cascading?

The dispatcher explicitly does not support cascading events because:

1. **Prevents cycles**: A → B → C → A would cause infinite loops
2. **Explicit flow**: All reactions to a command are visible in the handler registrations
3. **Simpler reasoning**: Each event has a fixed set of handlers, no hidden chains
4. **Testability**: Handlers can be tested in isolation

## CLI Commands

```
mixology order place <menu-id> <drink-id>:<qty> [<drink-id>:<qty>...]
mixology order list
mixology order get <order-id>
mixology order complete <order-id>
mixology order cancel <order-id>
```

## Success Criteria

- `go run ./main/cli order place happy-hour margarita:2` creates order
- `go run ./main/cli order complete <id>` triggers handlers
- Inventory stock is reduced (check via `inventory list`)
- Menu availability is recalculated (check via `menu show`)
- No cascading events in dispatcher logs
- `go test ./...` passes

## Dependencies

- Sprint 013c (Simplified constructors)
- Sprint 013d (Unified Commands object)
- Sprint 013e (No Request/Response wrappers)
- Sprint 013g (CedarEntity interface)
- Sprint 014 (Menu curation)
- Sprint 015 (Cost/substitution logic)
- Sprint 015b (Optional package)
- Sprint 015c (Domain structure reorganization)
