# Sprint 014: Menu Curation Context

## Goal

Create the Menu bounded context for curating drink menus with availability tracking. This is the first context that heavily consumes events from other contexts.

## Tasks

- [ ] Create `app/menu/models/menu.go` with Menu, MenuItem models
- [ ] Create `app/menu/internal/dao/` with file-based DAO
- [ ] Create `app/menu/authz/` with actions and policies
- [ ] Create `app/menu/queries/` with ListMenus, GetMenu queries
- [ ] Create `app/menu/internal/commands/` with CreateMenu, AddDrink, RemoveDrink, Publish commands
- [ ] Create `app/menu/handlers/` for inventory and drink events
- [ ] Create `app/menu/events/` with menu-related events
- [ ] Create `app/menu/module.go` exposing public API
- [ ] Add menu subcommands to CLI

## Domain Model

```go
type Menu struct {
    ID          string
    Name        string
    Description string
    Items       []MenuItem
    Status      MenuStatus
    CreatedAt   time.Time
    PublishedAt *time.Time
}

type MenuItem struct {
    DrinkID      string
    DisplayName  string       // Optional override of drink name
    Price        *Price       // Optional pricing
    Featured     bool
    Availability Availability // Calculated from inventory
    SortOrder    int
}

type Availability string
const (
    AvailabilityAvailable    Availability = "available"     // All ingredients in stock
    AvailabilityLimited      Availability = "limited"       // In stock but low
    AvailabilitySubstitution Availability = "substitution"  // Available with substitutes
    AvailabilityUnavailable  Availability = "unavailable"   // Missing required ingredients
)

type MenuStatus string
const (
    MenuStatusDraft     MenuStatus = "draft"
    MenuStatusPublished MenuStatus = "published"
    MenuStatusArchived  MenuStatus = "archived"
)

type Price struct {
    Amount   int    // In cents
    Currency string // USD, EUR, etc.
}
```

## Event Handlers

Menu context handles events from Inventory and Drinks:

```go
// app/menu/handlers/inventory_handlers.go
func HandleIngredientDepleted(menuDAO *dao.MenuDAO, drinkQueries *drinks.Queries) dispatcher.Handler {
    return func(ctx *middleware.Context, event any) error {
        e := event.(inventory.IngredientDepleted)

        // Find all menus with drinks using this ingredient
        menus, err := menuDAO.List(ctx)
        if err != nil {
            return err
        }

        for _, menu := range menus {
            if menu.Status != models.MenuStatusPublished {
                continue
            }

            for i, item := range menu.Items {
                if usesIngredient(ctx, drinkQueries, item.DrinkID, e.IngredientID) {
                    menu.Items[i].Availability = models.AvailabilityUnavailable
                    log.Printf("marked unavailable: menu=%s drink=%s", menu.ID, item.DrinkID)
                }
            }
            _ = menuDAO.Save(ctx, menu)
        }
        return nil
    }
}
```

## Events

- `MenuCreated{MenuID, Name}` - new menu created
- `MenuPublished{MenuID}` - menu made active
- `DrinkAvailabilityChanged{MenuID, DrinkID, OldStatus, NewStatus}` - availability recalculated

## Availability Calculation

```go
func calculateAvailability(ctx context.Context, drinkID string,
    inventoryQueries *inventory.Queries, drinkQueries *drinks.Queries) Availability {

    drink, err := drinkQueries.Get(ctx, drinkID)
    if err != nil {
        return AvailabilityUnavailable
    }

    for _, ri := range drink.Recipe.Ingredients {
        stock, err := inventoryQueries.GetStock(ctx, ri.IngredientID)
        if err != nil || stock.Quantity < ri.Amount {
            return AvailabilityUnavailable
        }
    }
    return AvailabilityAvailable
}
```

## CLI Commands

```
mixology menu list
mixology menu create "Happy Hour"
mixology menu add-drink <menu-id> <drink-id>
mixology menu remove-drink <menu-id> <drink-id>
mixology menu publish <menu-id>
mixology menu show <menu-id>
```

## Success Criteria

- `go run ./main/cli menu create "Happy Hour"` creates menu
- Adding drinks to menu calculates availability
- Depleting ingredient updates menu availability via handler
- `go test ./...` passes

## Dependencies

- Sprint 011 (Inventory for availability)
- Sprint 012 (Event handlers)
- Sprint 013 (Rich drink recipes)
