# Sprint 014: Menu Curation Context

## Goal

Create the Menu bounded context for curating drink menus with availability tracking. This is the first context that heavily consumes events from other contexts.

## Tasks

- [ ] Create `app/menu/models/menu.go` with Menu, MenuItem models
- [ ] Create `app/menu/internal/dao/dao.go` with file-based DAO
- [ ] Create `app/menu/authz/` with actions and policies
- [ ] Create `app/menu/queries/queries.go` with Get, List methods
- [ ] Create `app/menu/internal/commands/commands.go` with Create, AddDrink, RemoveDrink, Publish methods
- [ ] Create `app/menu/handlers/stock_adjusted.go` for inventory events
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

Menu context handles `StockAdjusted` from Inventory. Handlers:
- Read from the event
- Query other modules via their **queries package** (not Module - no AuthZ re-check)
- Update their own module's state via DAO
- Do NOT emit events or call commands

**Important:** Handlers import queries packages directly, not Module types. Modules enforce AuthZ at the surface boundary (CLI, API). Handlers execute within an already-authorized context.

```go
// app/menu/handlers/stock_adjusted.go
package handlers

import (
    "log"

    drinksq "github.com/TheFellow/go-modular-monolith/app/drinks/queries"
    "github.com/TheFellow/go-modular-monolith/app/inventory/events"
    "github.com/TheFellow/go-modular-monolith/app/menu/internal/dao"
    "github.com/TheFellow/go-modular-monolith/app/menu/models"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type StockAdjustedMenuUpdater struct {
    menuDAO      *dao.DAO
    drinkQueries *drinksq.Queries
}

func New() *StockAdjustedMenuUpdater {
    return &StockAdjustedMenuUpdater{
        menuDAO:      dao.New(),
        drinkQueries: drinksq.New(),
    }
}

func (h *StockAdjustedMenuUpdater) Handle(ctx *middleware.Context, e events.StockAdjusted) error {
    ingredientID := string(e.IngredientID.ID)
    depleted := e.NewQty == 0
    restocked := e.PreviousQty == 0 && e.NewQty > 0

    if !depleted && !restocked {
        return nil  // Only care about threshold crossings
    }

    menus, err := h.menuDAO.List(ctx)
    if err != nil {
        return err
    }

    for _, menu := range menus {
        if menu.Status != models.MenuStatusPublished {
            continue
        }

        changed := false
        for i, item := range menu.Items {
            // Query drinks queries package directly
            if !h.drinkUsesIngredient(ctx, item.DrinkID, ingredientID) {
                continue
            }

            if depleted && item.Availability != models.AvailabilityUnavailable {
                menu.Items[i].Availability = models.AvailabilityUnavailable
                changed = true
                log.Printf("menu %s: drink %s now unavailable (ingredient %s depleted)",
                    menu.ID, item.DrinkID, ingredientID)
            }

            if restocked && item.Availability == models.AvailabilityUnavailable {
                menu.Items[i].Availability = models.AvailabilityAvailable
                changed = true
                log.Printf("menu %s: drink %s now available (ingredient %s restocked)",
                    menu.ID, item.DrinkID, ingredientID)
            }
        }

        if changed {
            if err := h.menuDAO.Save(ctx, menu); err != nil {
                return err
            }
        }
    }
    return nil
}

func (h *StockAdjustedMenuUpdater) drinkUsesIngredient(ctx *middleware.Context, drinkID, ingredientID string) bool {
    drink, err := h.drinkQueries.Get(ctx, drinkID)
    if err != nil {
        return false
    }
    for _, ri := range drink.Recipe.Ingredients {
        if string(ri.IngredientID.ID) == ingredientID {
            return true
        }
    }
    return false
}
```

## Events

- `MenuCreated{MenuID, Name}` - new menu created
- `MenuPublished{MenuID}` - menu made active
- `DrinkAddedToMenu{MenuID, DrinkID}` - drink added
- `DrinkRemovedFromMenu{MenuID, DrinkID}` - drink removed

## Availability Calculation

Used when adding drinks or showing menus. Uses queries packages directly:

```go
// app/menu/internal/availability.go
package internal

import (
    drinksq "github.com/TheFellow/go-modular-monolith/app/drinks/queries"
    inventoryq "github.com/TheFellow/go-modular-monolith/app/inventory/queries"
    "github.com/TheFellow/go-modular-monolith/app/menu/models"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type AvailabilityCalculator struct {
    drinks    *drinksq.Queries
    inventory *inventoryq.Queries
}

func NewAvailabilityCalculator() *AvailabilityCalculator {
    return &AvailabilityCalculator{
        drinks:    drinksq.New(),
        inventory: inventoryq.New(),
    }
}

func (c *AvailabilityCalculator) Calculate(ctx *middleware.Context, drinkID string) models.Availability {
    drink, err := c.drinks.Get(ctx, drinkID)
    if err != nil {
        return models.AvailabilityUnavailable
    }

    for _, ri := range drink.Recipe.Ingredients {
        stock, err := c.inventory.Get(ctx, string(ri.IngredientID.ID))
        if err != nil || stock.Quantity < ri.Amount {
            return models.AvailabilityUnavailable
        }
        if stock.Quantity < ri.Amount*3 { // Low threshold
            return models.AvailabilityLimited
        }
    }
    return models.AvailabilityAvailable
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
- `StockAdjusted` event updates menu availability via handler
- Handler queries Drinks module to find affected items
- `go test ./...` passes

## Dependencies

- Sprint 011 (Inventory for availability)
- Sprint 012 (Event handlers pattern)
- Sprint 013 (Rich drink recipes)
- Sprint 013c (Simplified constructors)
- Sprint 013d (Unified Commands object)
- Sprint 013e (No Request/Response wrappers)
