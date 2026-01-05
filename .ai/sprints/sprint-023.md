# Sprint 023: Fat Events Pattern

## Goal

Ensure all domain events follow the "fat events" pattern - include complete domain models when the command already has them, allowing handlers to inspect any property they need without additional queries.

## Problem

Several events are "thin" - they contain only IDs or partial data:

| Event | Current Fields | Issue |
|-------|---------------|-------|
| `IngredientCreated` | `IngredientID, Name, Category` | Missing Unit, Description |
| `IngredientUpdated` | `IngredientID, Name, Category` | Missing Unit, Description; no previous values |
| `MenuCreated` | `MenuID, Name` | Missing Status, Description, CreatedAt |
| `MenuPublished` | `MenuID, PublishedAt` | Missing full Menu with items |
| `DrinkAddedToMenu` | `MenuID, DrinkID` | Missing MenuItem details (price, availability) |
| `DrinkRemovedFromMenu` | `MenuID, DrinkID` | Missing removed item details |
| `OrderCancelled` | `OrderID, MenuID, At` | Missing full Order with items |

Meanwhile, some events already follow the fat pattern well:

- `DrinkDeleted` - contains full `models.Drink`
- `DrinkCreated` - contains full drink data including Recipe
- `DrinkRecipeUpdated` - contains both previous and new recipes
- `OrderCompleted` - contains ingredient usage data for handlers

## Solution

Update thin events to include the full domain model. The command already has the model loaded, so this adds no extra queries - just include it in the event payload.

### IngredientCreated

```go
// Before
type IngredientCreated struct {
    IngredientID cedar.EntityUID
    Name         string
    Category     models.IngredientCategory
}

// After
type IngredientCreated struct {
    Ingredient models.Ingredient
}
```

### IngredientUpdated

```go
// Before
type IngredientUpdated struct {
    IngredientID cedar.EntityUID
    Name         string
    Category     models.IngredientCategory
}

// After
type IngredientUpdated struct {
    Previous models.Ingredient
    Current  models.Ingredient
}
```

### MenuCreated

```go
// Before
type MenuCreated struct {
    MenuID cedar.EntityUID
    Name   string
}

// After
type MenuCreated struct {
    Menu models.Menu
}
```

### MenuPublished

```go
// Before
type MenuPublished struct {
    MenuID      cedar.EntityUID
    PublishedAt time.Time
}

// After
type MenuPublished struct {
    Menu models.Menu // Contains PublishedAt, items, status, etc.
}
```

### DrinkAddedToMenu / DrinkRemovedFromMenu

```go
// Before
type DrinkAddedToMenu struct {
    MenuID  cedar.EntityUID
    DrinkID cedar.EntityUID
}

// After
type DrinkAddedToMenu struct {
    Menu models.Menu      // Full menu after change
    Item models.MenuItem  // The added item with price, availability, etc.
}

type DrinkRemovedFromMenu struct {
    Menu models.Menu      // Full menu after change
    Item models.MenuItem  // The removed item (for audit/undo)
}
```

### OrderCancelled

```go
// Before
type OrderCancelled struct {
    OrderID cedar.EntityUID
    MenuID  cedar.EntityUID
    At      time.Time
}

// After
type OrderCancelled struct {
    Order models.Order // Full order with items, notes, timestamps
}
```

### StockAdjusted (keep as-is)

`StockAdjusted` is intentionally lean but adequate - it provides `PreviousQty`, `NewQty`, `Delta`, and `Reason`. The handler only needs `IngredientID` to check menu items. No change needed.

## Command Updates

Each command that emits these events needs to populate the fat event with the model it already has:

- `ingredients/internal/commands/create.go` - has the created Ingredient
- `ingredients/internal/commands/update.go` - has both previous and updated Ingredient
- `menu/internal/commands/create.go` - has the created Menu
- `menu/internal/commands/publish.go` - has the published Menu
- `menu/internal/commands/add_drink.go` - has the updated Menu and added MenuItem
- `menu/internal/commands/remove_drink.go` - has the updated Menu and removed MenuItem
- `orders/internal/commands/cancel.go` - has the cancelled Order

## Handler Updates

Update any handlers that reference the old event field names:

- `IngredientCreatedAudit` - access `e.Ingredient.Name` instead of `e.Name`
- `IngredientCreatedCounter` - access `e.Ingredient.Category` instead of `e.Category`

## Tasks

- [ ] Update `IngredientCreated` to fat event with full `models.Ingredient`
- [ ] Update `IngredientUpdated` to fat event with `Previous` and `Current` models
- [ ] Update ingredient handlers to use new event structure
- [ ] Update `MenuCreated` to fat event with full `models.Menu`
- [ ] Update `MenuPublished` to fat event with full `models.Menu`
- [ ] Update `DrinkAddedToMenu` to fat event with Menu and MenuItem
- [ ] Update `DrinkRemovedFromMenu` to fat event with Menu and removed MenuItem
- [ ] Update `OrderCancelled` to fat event with full `models.Order`
- [ ] Update all commands that emit these events
- [ ] Verify `go test ./...` passes

## Acceptance Criteria

- All events contain complete domain models (no ID-only events except where explicitly justified)
- Commands populate events with models they already have loaded
- Handlers can access any model property without additional queries
- Pattern matches `DrinkDeleted` and `OrderCompleted` exemplars
- `go test ./...` passes
