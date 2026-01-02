# Sprint 015: Orders & Inventory Consumption

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
    OrderID string
    MenuID  string
    Items   []OrderItemCompleted
}

type OrderItemCompleted struct {
    DrinkID  string
    Quantity int
}
```

Events carry minimal data. Handlers query what they need - the query cache (Sprint 016) ensures they see consistent "as-of-command" state.

## Handler Pattern: No Cascading Events

Handlers react to events but **do not emit new events**. They update their own state directly.

```go
// app/inventory/handlers/order_handlers.go
func HandleOrderCompleted(stockDAO *dao.StockDAO, drinkQueries *drinks.Queries) dispatcher.Handler {
    return func(ctx *middleware.Context, event any) error {
        e := event.(orders.OrderCompleted)

        for _, item := range e.Items {
            // Query drink recipe - returns CACHED result from command execution
            drink, err := drinkQueries.Get(ctx, item.DrinkID)
            if err != nil {
                return err
            }

            // Calculate and deduct ingredients
            for _, ri := range drink.Recipe.Ingredients {
                amount := ri.Amount * float64(item.Quantity)

                stock, err := stockDAO.Get(ctx, ri.IngredientID)
                if err != nil {
                    return err
                }

                stock.Quantity -= amount
                stock.LastUpdated = time.Now()

                if err := stockDAO.Save(ctx, stock); err != nil {
                    return err
                }

                log.Printf("stock adjusted: %s -= %.2f (order %s)",
                    ri.IngredientID, amount, e.OrderID)
            }
        }
        return nil
    }
}
```

```go
// app/menu/handlers/order_handlers.go
func HandleOrderCompleted(menuDAO *dao.MenuDAO, inventoryQueries *inventory.Queries, drinkQueries *drinks.Queries) dispatcher.Handler {
    return func(ctx *middleware.Context, event any) error {
        e := event.(orders.OrderCompleted)

        // Query menu - returns CACHED result from command execution
        menu, err := menuDAO.Get(ctx, e.MenuID)
        if err != nil {
            return err
        }

        // Check each drink's availability and update directly
        for i, item := range menu.Items {
            // drinkQueries.Get returns CACHED result
            // inventoryQueries.GetStock returns FRESH data (wasn't queried by command)
            availability := calculateAvailability(ctx, item.DrinkID, inventoryQueries, drinkQueries)
            if menu.Items[i].Availability != availability {
                menu.Items[i].Availability = availability
                log.Printf("availability changed: menu=%s drink=%s status=%s",
                    menu.ID, item.DrinkID, availability)
            }
        }

        return menuDAO.Save(ctx, menu)
    }
}
```

## Why No Cascading?

The dispatcher explicitly does not support cascading events because:

1. **Prevents cycles**: A → B → C → A would cause infinite loops
2. **Explicit flow**: All reactions to a command are visible in the handler registrations
3. **Simpler reasoning**: Each event has a fixed set of handlers, no hidden chains
4. **Testability**: Handlers can be tested in isolation

## Trade-offs

**Lost**: Granular audit trail via events (no `StockAdjusted`, `IngredientDepleted`, `DrinkAvailabilityChanged` from handler-driven changes)

**Gained**:
- Simple, predictable event flow
- No risk of cycles or infinite loops
- Each handler is a leaf node

**Mitigation**: If audit trail is needed:
- Handlers can write to an audit log directly
- Or the originating command can orchestrate synchronously and emit all events itself

## Alternative: Command Orchestration

If we need events for audit, the command itself does the work:

```go
// app/orders/internal/commands/complete.go
func (c *CompleteOrder) Execute(ctx *middleware.Context, req CompleteOrderRequest) (*Order, error) {
    order := // ... mark complete

    // Synchronously adjust inventory (this WILL emit StockAdjusted events)
    for _, usage := range order.IngredientsUsed {
        c.inventoryModule.AdjustStock(ctx, inventory.AdjustStockRequest{
            IngredientID: usage.IngredientID,
            Delta:        -usage.Amount,
            Reason:       inventory.ReasonUsed,
        })
    }

    // The AdjustStock commands emit their own events
    // Those events trigger Menu handlers
    // But this creates tight coupling between Orders and Inventory

    ctx.AddEvent(events.OrderCompleted{...})
    return order, nil
}
```

This trades loose coupling for audit trail. Choose based on requirements.

## Recommended Approach

For this project, we'll use the **handler-only approach** (no cascading):
- Handlers update state directly
- Audit via logs, not events
- Clean separation between contexts

This demonstrates the "handlers as leaf nodes" pattern that prevents complexity.

## Success Criteria

- `go run ./main/cli order place happy-hour margarita:2` creates order
- `go run ./main/cli order complete <id>` triggers handlers
- Inventory stock is reduced (check via `inventory list`)
- Menu availability is recalculated (check via `menu show`)
- No cascading events in dispatcher logs
- `go test ./...` passes

## Dependencies

- Sprint 014 (Cost/substitution logic)
