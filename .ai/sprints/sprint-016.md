# Sprint 016: Orders & Inventory Consumption

## Goal

Create an Orders context that records drink orders. Handlers in other contexts react to order events but do not produce new events (no cascading).

## Tasks

- [ ] Create `app/orders/models/order.go` with Order, OrderItem models
- [ ] Create `app/orders/internal/dao/` with file-based DAO
- [ ] Create `app/orders/authz/` with actions and policies
- [ ] Create `app/orders/queries/` with ListOrders, GetOrder queries
- [ ] Create `app/orders/internal/commands/` with PlaceOrder, CompleteOrder, CancelOrder
- [ ] Create `app/orders/events/` with order events
- [ ] Create `app/inventory/handlers/order_handlers.go` - updates stock directly (no events)
- [ ] Create `app/menu/handlers/order_handlers.go` - recalculates availability directly (no events)
- [ ] Add order subcommands to CLI

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
// app/inventory/handlers/order_handlers.go
func HandleOrderCompleted(stockDAO *dao.StockDAO) dispatcher.Handler {
    return func(ctx *middleware.Context, event any) error {
        e := event.(orders.OrderCompleted)

        for _, usage := range e.IngredientUsage {
            stock, err := stockDAO.Get(ctx, string(usage.IngredientID.ID))
            if err != nil {
                return err
            }

            stock.Quantity -= usage.Amount
            stock.LastUpdated = time.Now()

            if err := stockDAO.Save(ctx, stock); err != nil {
                return err
            }
        }
        return nil
    }
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

- Sprint 014 (Menu curation)
- Sprint 015 (Cost/substitution logic)
