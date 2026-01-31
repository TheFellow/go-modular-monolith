# Sprint 028: Fat Events with Embedded Domain Models

## Status

- Started: 2026-01-09
- Completed: 2026-01-10

## Goal

Refactor domain events to embed concrete domain models instead of duplicating their fields. When a model changes, events automatically carry the new fields without modification.

## Problem

Some events duplicate model fields instead of embedding the model:

```go
// Bad - duplicates Drink fields
type DrinkCreated struct {
    DrinkID     cedar.EntityUID
    Name        string
    Category    models.DrinkCategory
    Glass       models.GlassType
    Recipe      models.Recipe
    Description string
}

// Good - embeds the model
type DrinkDeleted struct {
    Drink models.Drink
}
```

**Why duplication is wrong:**

1. **Maintenance burden**: When the model changes, the event must change too
2. **Drift risk**: Easy to forget to update events, leading to missing fields
3. **Boilerplate**: Requires mapping functions like `OrderPlacedFromDomain()`
4. **Inconsistency**: Some events embed models, others don't

## Solution

Events embed domain models directly. Handlers access whatever fields they need.

```go
// After
type DrinkCreated struct {
    Drink models.Drink
}

type OrderPlaced struct {
    Order models.Order
}
```

For events that include derived/diff data beyond the model, embed the model plus keep the derived fields:

```go
type OrderCompleted struct {
    Order               models.Order
    IngredientUsage     []IngredientUsage
    DepletedIngredients []cedar.EntityUID
}
```

## Audit Results

### Good (already embed models)

| Event | Embedding |
|-------|-----------|
| `DrinkDeleted` | `Drink models.Drink` |
| `IngredientCreated` | `Ingredient models.Ingredient` |
| `IngredientUpdated` | `Previous, Current models.Ingredient` |
| `MenuCreated` | `Menu models.Menu` |
| `MenuPublished` | `Menu models.Menu` |
| `DrinkAddedToMenu` | `Menu models.Menu`, `Item models.MenuItem` |
| `DrinkRemovedFromMenu` | `Menu models.Menu`, `Item models.MenuItem` |
| `OrderCancelled` | `Order models.Order` |

### Needs Refactoring

| Event | Current | Target |
|-------|---------|--------|
| `DrinkCreated` | Duplicates: DrinkID, Name, Category, Glass, Recipe, Description | `Drink models.Drink` |
| `DrinkRecipeUpdated` | DrinkID, Name, PreviousRecipe, NewRecipe, Added/Removed | `Previous, Current models.Drink` |
| `StockAdjusted` | IngredientID, PreviousQty, NewQty, Delta, Reason | `Previous, Current models.Stock` |
| `OrderPlaced` | OrderID, MenuID, Items, At, Notes + helper types | `Order models.Order` |
| `OrderCompleted` | OrderID, MenuID, Items + derived data + helper types | `Order models.Order` + derived fields |

### Types to Delete

| Type | Reason |
|------|--------|
| `OrderItemPlaced` | Use `models.OrderItem` |
| `OrderItemCompleted` | Use `models.OrderItem` |
| `OrderPlacedFromDomain()` | No longer needed |

## Changes

### drinks/events/drink-created.go

```go
// Before
type DrinkCreated struct {
    DrinkID     cedar.EntityUID
    Name        string
    Category    models.DrinkCategory
    Glass       models.GlassType
    Recipe      models.Recipe
    Description string
}

// After
type DrinkCreated struct {
    Drink models.Drink
}
```

### drinks/events/drink-recipe-updated.go

```go
// Before
type DrinkRecipeUpdated struct {
    DrinkID            cedar.EntityUID
    Name               string
    PreviousRecipe     models.Recipe
    NewRecipe          models.Recipe
    AddedIngredients   []cedar.EntityUID
    RemovedIngredients []cedar.EntityUID
}

// After
type DrinkRecipeUpdated struct {
    Previous models.Drink
    Current  models.Drink
}
```

Handlers can compute added/removed ingredients if needed - that's derived data.

### inventory/events/stock_adjusted.go

```go
// Before
type StockAdjusted struct {
    IngredientID cedar.EntityUID
    PreviousQty  float64
    NewQty       float64
    Delta        float64
    Reason       string
}

// After
type StockAdjusted struct {
    Previous models.Stock
    Current  models.Stock
    Reason   string
}
```

Delta can be computed: `Current.Quantity - Previous.Quantity`

### orders/events/order_placed.go

```go
// Before
type OrderPlaced struct {
    OrderID cedar.EntityUID
    MenuID  cedar.EntityUID
    Items   []OrderItemPlaced
    At      time.Time
    Notes   string
}

type OrderItemPlaced struct {
    DrinkID   cedar.EntityUID
    Quantity  int
    ItemNotes string
}

func OrderPlacedFromDomain(o models.Order) OrderPlaced { ... }

// After
type OrderPlaced struct {
    Order models.Order
}
```

Delete `OrderItemPlaced` and `OrderPlacedFromDomain()`.

### orders/events/order_placed.go (OrderCompleted)

```go
// Before
type OrderCompleted struct {
    OrderID             cedar.EntityUID
    MenuID              cedar.EntityUID
    Items               []OrderItemCompleted
    IngredientUsage     []IngredientUsage
    DepletedIngredients []cedar.EntityUID
    At                  time.Time
}

type OrderItemCompleted struct {
    DrinkID   cedar.EntityUID
    Name      string
    Quantity  int
    ItemNotes string
}

// After
type OrderCompleted struct {
    Order               models.Order
    IngredientUsage     []IngredientUsage     // Derived data - keep
    DepletedIngredients []cedar.EntityUID     // Derived data - keep
}
```

Delete `OrderItemCompleted`. Keep `IngredientUsage` - it's genuinely derived data not in the Order model.

## Tasks

### Phase 1: Refactor Events

- [x] `DrinkCreated` - embed `Drink models.Drink`
- [x] `DrinkRecipeUpdated` - embed `Previous, Current models.Drink`
- [x] `StockAdjusted` - embed `Previous, Current models.Stock`, keep `Reason`
- [x] `OrderPlaced` - embed `Order models.Order`
- [x] `OrderCompleted` - embed `Order models.Order`, keep derived fields

### Phase 2: Delete Helper Types

- [x] Delete `OrderItemPlaced`
- [x] Delete `OrderItemCompleted`
- [x] Delete `OrderPlacedFromDomain()`

### Phase 3: Update Event Publishers

- [x] drinks `Create` command - publish `DrinkCreated{Drink: created}`
- [x] drinks `Update` command - publish `DrinkRecipeUpdated{Previous: before, Current: after}`
- [x] inventory `Adjust`/`Set` commands - publish `StockAdjusted{Previous, Current, Reason}`
- [x] orders `Place` command - publish `OrderPlaced{Order: order}`
- [x] orders `Complete` command - publish `OrderCompleted{Order: order, ...}`

### Phase 4: Update Event Handlers

- [x] Update handlers to access `event.Drink.ID` instead of `event.DrinkID`, etc.
- [x] Remove any computed fields handlers were using (they can compute from models)

### Phase 5: Tests

- [x] Update event-related tests
- [x] Verify `go test ./...` passes

## Acceptance Criteria

- All events embed domain models instead of duplicating fields
- No helper mapping functions (e.g., `OrderPlacedFromDomain`)
- No duplicate type definitions (e.g., `OrderItemPlaced` vs `models.OrderItem`)
- Handlers access model fields directly
- All tests pass
