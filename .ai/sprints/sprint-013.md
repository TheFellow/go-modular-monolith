# Sprint 013: Event-Driven Cross-Context Handlers

## Goal

Wire up the dispatcher to route events between contexts. Handlers react to events but do not emit new events (no cascading).

## Tasks

- [ ] Implement real `pkg/dispatcher` (replace stub)
- [ ] Create handler registration mechanism
- [ ] Create `app/menu/handlers/inventory_handlers.go` for inventory events
- [ ] Create `app/menu/handlers/drinks_handlers.go` for drink events
- [ ] Wire handlers in dispatcher initialization
- [ ] Add integration tests for event flows
- [ ] Update codegen to generate handler bindings

## Dispatcher Implementation

```go
// pkg/dispatcher/dispatcher.go
type Handler func(ctx context.Context, event any) error

type Dispatcher struct {
    handlers map[reflect.Type][]Handler
}

func (d *Dispatcher) Register(eventType any, handler Handler) {
    t := reflect.TypeOf(eventType)
    d.handlers[t] = append(d.handlers[t], handler)
}

func (d *Dispatcher) Flush(ctx context.Context, events []any) error {
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

## Handler Pattern: Leaf Nodes Only

**Critical design constraint**: Handlers do NOT emit new events. They are leaf nodes in the event tree.

Handlers can:
- Update their own context's state (via DAO)
- Query other contexts for information (via query cache for consistency)
- Write to logs/audit trails
- Send external notifications

Handlers cannot:
- Call commands (which would emit events)
- Add events to the context
- Trigger other handlers

This prevents cycles and keeps event flow predictable.

## Query Cache: Handler Consistency

Handlers receive the same `middleware.Context` that the command used. Queries are transparently cached, so:
- If the command queried it → handlers see the same cached result
- If the command didn't query it → handlers get fresh data

This ensures handlers see consistent "as-of-command" state without artificial pre-fetching. See Sprint 016 for implementation details.

## Event Handlers

### Inventory → Menu: Stock Changes

When stock is adjusted via command (not handler), Menu reacts:

```go
// app/menu/handlers/inventory_handlers.go
func HandleIngredientDepleted(menuDAO *dao.MenuDAO, drinkQueries *drinks.Queries, inventoryQueries *inventory.Queries) dispatcher.Handler {
    return func(ctx context.Context, event any) error {
        e := event.(inventory.IngredientDepleted)

        // Find all menus (could optimize with index)
        menus, err := menuDAO.List(ctx)
        if err != nil {
            return err
        }

        for _, menu := range menus {
            if menu.Status != models.MenuStatusPublished {
                continue
            }

            updated := false
            for i, item := range menu.Items {
                if usesIngredient(ctx, drinkQueries, item.DrinkID, e.IngredientID) {
                    // Recalculate availability
                    newAvail := calculateAvailability(ctx, item.DrinkID, inventoryQueries, drinkQueries)
                    if menu.Items[i].Availability != newAvail {
                        menu.Items[i].Availability = newAvail
                        updated = true
                        log.Printf("availability changed: menu=%s drink=%s old=%s new=%s",
                            menu.ID, item.DrinkID, menu.Items[i].Availability, newAvail)
                    }
                }
            }

            if updated {
                if err := menuDAO.Save(ctx, menu); err != nil {
                    return err
                }
            }
        }
        return nil
    }
}

func HandleIngredientRestocked(menuDAO *dao.MenuDAO, drinkQueries *drinks.Queries, inventoryQueries *inventory.Queries) dispatcher.Handler {
    // Similar pattern - recalculate availability for affected drinks
    // Only check drinks that were previously unavailable
}
```

### Drinks → Menu: Recipe Changes

```go
// app/menu/handlers/drinks_handlers.go
func HandleDrinkRecipeUpdated(menuDAO *dao.MenuDAO, inventoryQueries *inventory.Queries, drinkQueries *drinks.Queries) dispatcher.Handler {
    return func(ctx context.Context, event any) error {
        e := event.(drinks.DrinkRecipeUpdated)

        // Find menus containing this drink
        menus, err := menuDAO.FindWithDrink(ctx, e.DrinkID)
        if err != nil {
            return err
        }

        for _, menu := range menus {
            for i, item := range menu.Items {
                if item.DrinkID == e.DrinkID {
                    newAvail := calculateAvailability(ctx, e.DrinkID, inventoryQueries, drinkQueries)
                    if menu.Items[i].Availability != newAvail {
                        menu.Items[i].Availability = newAvail
                        log.Printf("availability changed (recipe update): menu=%s drink=%s status=%s",
                            menu.ID, e.DrinkID, newAvail)
                    }
                }
            }
            if err := menuDAO.Save(ctx, menu); err != nil {
                return err
            }
        }
        return nil
    }
}
```

## Handler Registration (Codegen)

```go
// pkg/dispatcher/handlers_gen.go (generated)
func RegisterAllHandlers(d *Dispatcher,
    menuDAO *menu_dao.MenuDAO,
    drinkQueries *drinks.Queries,
    inventoryQueries *inventory.Queries,
) {
    // Inventory events → Menu handlers
    d.Register(inventory.IngredientDepleted{},
        menu_handlers.HandleIngredientDepleted(menuDAO, drinkQueries, inventoryQueries))
    d.Register(inventory.IngredientRestocked{},
        menu_handlers.HandleIngredientRestocked(menuDAO, drinkQueries, inventoryQueries))

    // Drinks events → Menu handlers
    d.Register(drinks.DrinkRecipeUpdated{},
        menu_handlers.HandleDrinkRecipeUpdated(menuDAO, inventoryQueries, drinkQueries))
}
```

## Event Flow (No Cascade)

```
Command emits event(s)
         │
         ▼
    ┌─────────┐
    │Dispatcher│
    └─────────┘
         │
    ┌────┴────┐
    ▼         ▼
Handler A  Handler B
    │         │
    ▼         ▼
 (leaf)    (leaf)

Each handler is a leaf node.
No handler emits events.
No chaining. No cycles.
```

## When Events Are Emitted

Events are ONLY emitted by commands:

| Context | Command | Event |
|---------|---------|-------|
| Inventory | AdjustStock | StockAdjusted, (IngredientDepleted), (IngredientRestocked) |
| Inventory | SetStock | StockAdjusted, ... |
| Drinks | Create | DrinkCreated |
| Drinks | UpdateRecipe | DrinkRecipeUpdated |
| Menu | Publish | MenuPublished |
| Orders | Complete | OrderCompleted |

Handlers react to these but never emit their own events.

## Notes

This design trades some flexibility for predictability:

**Pros**:
- Easy to reason about event flow
- No hidden chains or cycles
- Each handler testable in isolation
- Clear audit: events = command executions

**Cons**:
- Some derived state changes aren't captured as events
- Less granular audit trail for handler-driven updates
- Handlers must be idempotent (may run multiple times on retry)

## Success Criteria

- Depleting vodka via `inventory adjust` marks Martini as unavailable
- Restocking vodka marks Martini as available again
- Changing Margarita recipe triggers availability recalc
- No cascading events in logs
- `go test ./...` passes with integration tests

## Dependencies

- Sprint 012 (Menu context)
