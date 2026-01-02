# Sprint 012: Menu Curation Context

## Goal

Create the Menu bounded context for curating drink menus with availability tracking.

## Tasks

- [ ] Create `app/menu/models/menu.go` with Menu, MenuItem models
- [ ] Create `app/menu/internal/dao/` with file-based DAO
- [ ] Create `app/menu/authz/` with actions and policies
- [ ] Create `app/menu/queries/` with ListMenus, GetMenu, GetAvailableDrinks queries
- [ ] Create `app/menu/internal/commands/` with CreateMenu, AddDrink, RemoveDrink, Publish commands
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

## Availability Calculation

Menu queries both Drinks and Inventory to calculate availability:

```go
// app/menu/internal/availability.go
func (s *AvailabilityService) CalculateForDrink(ctx context.Context, drinkID string) (Availability, error) {
    // 1. Get drink recipe from Drinks context
    drink, err := s.drinkQueries.Get(ctx, drinkID)
    if err != nil {
        return AvailabilityUnavailable, err
    }

    // 2. Check inventory for each required ingredient
    ingredientIDs := extractIngredientIDs(drink.Recipe)
    amounts := extractAmounts(drink.Recipe)

    availability, err := s.inventoryQueries.CheckAvailability(ctx, inventory.AvailabilityRequest{
        IngredientIDs: ingredientIDs,
        Amounts:       amounts,
    })
    if err != nil {
        return AvailabilityUnavailable, err
    }

    // 3. Determine availability status
    if !availability.Available {
        // Check if substitutes available for missing ingredients
        if hasSubstitutes(drink.Recipe, availability.Missing) {
            return AvailabilitySubstitution, nil
        }
        return AvailabilityUnavailable, nil
    }

    if len(availability.LowStock) > 0 {
        return AvailabilityLimited, nil
    }

    return AvailabilityAvailable, nil
}
```

## Events

- `MenuCreated{MenuID, Name}` - new menu created
- `MenuPublished{MenuID}` - menu made active
- `MenuArchived{MenuID}` - menu deactivated
- `DrinkAddedToMenu{MenuID, DrinkID}` - drink added to menu
- `DrinkRemovedFromMenu{MenuID, DrinkID}` - drink removed from menu
- `DrinkAvailabilityChanged{MenuID, DrinkID, OldStatus, NewStatus}` - availability recalculated

## Notes

Menu is a "downstream" context - it consumes data from Drinks and Inventory. It doesn't own drink or ingredient data, only the curation and availability status.

Availability is recalculated:
1. When a drink is added to a menu
2. When inventory events indicate stock changes
3. On-demand via explicit refresh

## Success Criteria

- `go run ./main/cli menu create "Happy Hour"`
- `go run ./main/cli menu add-drink happy-hour margarita`
- Availability status calculated correctly based on inventory
- `go test ./...` passes

## Dependencies

- Sprint 010 (Rich drink recipes)
- Sprint 011 (Inventory for availability checks)
