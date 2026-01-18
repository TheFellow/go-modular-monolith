# Sprint 033f: Strongly Typed Entity IDs

## Problem

Currently all entity IDs use `cedar.EntityUID`:

```go
type Drink struct {
    ID cedar.EntityUID  // Could accidentally be an IngredientID!
    ...
}

func (d *DAO) Get(ctx dao.Context, id cedar.EntityUID) (*models.Drink, error)
```

This is weakly typed - the compiler can't catch if you pass an `IngredientID` where a `DrinkID` is expected:

```go
ingredient := getIngredient()
drink, err := drinkDAO.Get(ctx, ingredient.ID)  // Compiles! But wrong.
```

## Solution

Define distinct ID types per entity:

```go
type DrinkID cedar.EntityUID

func (d *DAO) Get(ctx dao.Context, id entity.DrinkID) (*models.Drink, error)
```

Now the compiler catches mistakes:

```go
ingredient := getIngredient()
drink, err := drinkDAO.Get(ctx, ingredient.ID)  // Compile error!
```

## Implementation

### ID Type Definition

```go
// app/kernel/entity/drink.go
package entity

import "github.com/cedar-policy/cedar-go"

const (
    TypeDrink   = cedar.EntityType("Mixology::Drink")
    PrefixDrink = "drk"
)

// DrinkID is a strongly-typed ID for Drink entities.
type DrinkID cedar.EntityUID

// NewDrinkID generates a new DrinkID with a KSUID.
func NewDrinkID() DrinkID {
    return DrinkID(NewID(TypeDrink, PrefixDrink))
}

// ParseDrinkID creates a DrinkID from a string (for existing IDs).
func ParseDrinkID(id string) (DrinkID, error) {
	if id == "" {
		return DrinkID(cedar.NewEntityUID(TypeDrink,"")), nil
    }
    if !strings.HasPrefix(id, PrefixDrink+"-") {
        return DrinkID{}, errors.Invalidf("invalid drink ID prefix: %s", id)
    }
    return DrinkID(cedar.NewEntityUID(TypeDrink, cedar.String(id))), nil
}

// EntityUID converts to cedar.EntityUID for Cedar API interop.
func (id DrinkID) EntityUID() cedar.EntityUID {
    return cedar.EntityUID(id)
}

// String returns the ID portion as a string.
func (id DrinkID) String() string {
    return string(cedar.EntityUID(id).ID)
}

// IsZero returns true if the ID is unset.
func (id DrinkID) IsZero() bool {
    return cedar.EntityUID(id).ID == ""
}
```

### Usage Pattern

```go
// Creating new IDs
id := entity.NewDrinkID()

// Parsing existing IDs
id, err := entity.ParseDrinkID("drk-abc123")
if err != nil {
    return err
}

// Converting to cedar.EntityUID when needed
cedarUID := id.EntityUID()
```

### Model Updates

```go
// app/domains/drinks/models/drink.go
type Drink struct {
    ID          entity.DrinkID  // Strongly typed!
    Name        string
    Category    DrinkCategory
    Glass       GlassType
    Recipe      Recipe
    Description string
    DeletedAt   optional.Value[time.Time]
}

func (d Drink) EntityUID() cedar.EntityUID {
    return d.ID.EntityUID()
}

func (d Drink) CedarEntity() cedar.Entity {
    return cedar.Entity{
        UID:        d.ID.EntityUID(),
        Parents:    cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(cedar.RecordMap{...}),
        Tags:       cedar.NewRecord(nil),
    }
}
```

### DAO Updates

```go
// app/domains/drinks/internal/dao/get.go
func (d *DAO) Get(ctx dao.Context, id entity.DrinkID) (*models.Drink, error) {
    var row DrinkRow
    err := dao.Read(ctx, func(tx *bstore.Tx) error {
        row = DrinkRow{ID: id.String()}
        return tx.Get(&row)
    })
    ...
}
```

### Module Updates

```go
// app/domains/drinks/get.go
func (m *Module) Get(ctx *middleware.Context, id entity.DrinkID) (*models.Drink, error) {
    return middleware.RunQuery(ctx, authz.ActionGet, m.dao.Get, id)
}

// app/domains/drinks/delete.go
func (m *Module) Delete(ctx *middleware.Context, id entity.DrinkID) (*models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionDelete,
        middleware.Get(m.dao.Get, id),
        m.commands.Delete,
    )
}
```

### Cross-Domain References

Recipe references ingredients by ID:

```go
// app/domains/drinks/models/recipe.go
type RecipeIngredient struct {
    IngredientID entity.IngredientID  // Strongly typed!
    Amount       decimal.Decimal
    Unit         string
    Optional     bool
    Substitutes  []entity.IngredientID
}
```

Compile-time safety: can't accidentally use a DrinkID as an IngredientID.

## All Entity ID Types

```go
// app/kernel/entity/

// Types
type DrinkID cedar.EntityUID
type IngredientID cedar.EntityUID
type MenuID cedar.EntityUID
type OrderID cedar.EntityUID
type InventoryID cedar.EntityUID
type ActivityID cedar.EntityUID

// Constructors (package-level functions)
func NewDrinkID() DrinkID
func ParseDrinkID(id string) (DrinkID, error)

func NewIngredientID() IngredientID
func ParseIngredientID(id string) (IngredientID, error)

// ... etc for each type
```

Each type has instance methods:
- `EntityUID()` - convert to cedar.EntityUID
- `String()` - get ID string
- `IsZero()` - check if unset

## Guards That Become Unnecessary

With strongly typed IDs, some runtime checks become compile-time guarantees:

```go
// Before - runtime check needed
func (d *DAO) Get(ctx context.Context, id cedar.EntityUID) (*models.Drink, error) {
    if id.Type != entity.TypeDrink {
        return nil, errors.Invalidf("expected drink ID, got %s", id.Type)
    }
    ...
}

// After - compiler enforces it
func (d *DAO) Get(ctx dao.Context, id entity.DrinkID) (*models.Drink, error) {
    // No check needed - can only receive DrinkID
    ...
}
```

## Tasks

### Phase 1: Define ID Types

- [ ] Update `entity/drink.go` with `DrinkID` type and methods
- [ ] Update `entity/ingredient.go` with `IngredientID` type and methods
- [ ] Update `entity/menu.go` with `MenuID` type and methods
- [ ] Update `entity/order.go` with `OrderID` type and methods
- [ ] Update `entity/inventory.go` with `InventoryID` type and methods
- [ ] Update `entity/audit.go` with `ActivityID` type and methods

### Phase 2: Update Models

- [ ] Update `drinks/models/drink.go` to use `entity.DrinkID`
- [ ] Update `drinks/models/recipe.go` to use `entity.IngredientID`
- [ ] Update `ingredients/models/ingredient.go` to use `entity.IngredientID`
- [ ] Update `menu/models/menu.go` to use `entity.MenuID`
- [ ] Update `orders/models/order.go` to use `entity.OrderID`
- [ ] Update `inventory/models/stock.go` to use `entity.InventoryID`
- [ ] Update `audit/models/activity.go` to use `entity.ActivityID`

### Phase 3: Update DAOs

- [ ] Update drinks DAO method signatures
- [ ] Update ingredients DAO method signatures
- [ ] Update menu DAO method signatures
- [ ] Update orders DAO method signatures
- [ ] Update inventory DAO method signatures
- [ ] Update audit DAO method signatures

### Phase 4: Update Module Methods

- [ ] Update drinks module method signatures
- [ ] Update ingredients module method signatures
- [ ] Update menu module method signatures
- [ ] Update orders module method signatures
- [ ] Update inventory module method signatures

### Phase 5: Update Commands

- [ ] Update drinks commands
- [ ] Update ingredients commands
- [ ] Update menu commands
- [ ] Update orders commands
- [ ] Update inventory commands

### Phase 6: Remove Unnecessary Guards

- [ ] Audit for runtime type checks that are now compile-time guarantees
- [ ] Remove redundant guards

### Phase 7: Verify

- [ ] Run `go test ./...` and fix any issues

## Acceptance Criteria

- [ ] Each entity has a distinct ID type
- [ ] Models use strongly-typed IDs
- [ ] DAOs accept strongly-typed IDs
- [ ] Module methods accept strongly-typed IDs
- [ ] Cross-domain references use correct ID types
- [ ] Compiler catches ID type mismatches
- [ ] Runtime type guards removed where redundant
- [ ] All tests pass

## Result

```go
// Compile-time type safety
drink, _ := drinkModule.Get(ctx, drinkID)      // ✓
drink, _ := drinkModule.Get(ctx, ingredientID) // Compile error!

// Self-documenting signatures
func (m *Module) Get(ctx *middleware.Context, id entity.DrinkID) (*models.Drink, error)

// Clear cross-domain references
type RecipeIngredient struct {
    IngredientID entity.IngredientID
    ...
}
```
