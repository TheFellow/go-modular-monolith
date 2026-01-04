# Sprint 015: Advanced Business Rules & Cost Analysis

## Goal

Implement sophisticated business rules including ingredient substitutions, cost calculations, and menu analytics.

## Tasks

- [ ] Create `app/ingredients/models/substitution.go` with substitution rules
- [ ] Add substitution queries to Ingredients context
- [ ] Implement substitution-aware availability in Menu context
- [ ] Add cost tracking to Inventory (cost per unit)
- [ ] Create `app/menu/queries/cost.go` for drink cost calculation
- [ ] Create `app/menu/queries/analytics.go` for menu analytics
- [ ] Add cost/margin display to CLI menu commands

## Substitution Rules

```go
// app/ingredients/models/substitution.go
type SubstitutionRule struct {
    IngredientID    string
    SubstituteID    string
    Ratio           float64  // 1.0 = equal, 0.5 = use half as much
    QualityImpact   Quality  // Does substitution affect drink quality?
    Notes           string
}

type Quality string
const (
    QualityEquivalent Quality = "equivalent"  // No noticeable difference
    QualitySimilar    Quality = "similar"     // Minor difference
    QualityDifferent  Quality = "different"   // Noticeably different
)
```

Example substitutions:
- Lime juice ↔ Lemon juice (similar, 1:1)
- Simple syrup ↔ Honey syrup (different, 0.75:1)
- Bourbon → Rye whiskey (equivalent, 1:1)
- Fresh mint → Dried mint (different, 0.5:1)

## Enhanced Availability Calculation

```go
// app/menu/internal/availability.go
type AvailabilityDetail struct {
    Status         Availability
    Missing        []MissingIngredient
    Substitutions  []AppliedSubstitution
    EstimatedCost  Price
}

type MissingIngredient struct {
    IngredientID   string
    Required       float64
    Available      float64
    HasSubstitute  bool
}

type AppliedSubstitution struct {
    Original       string
    Substitute     string
    Ratio          float64
    QualityImpact  Quality
}
```

## Cost Tracking

```go
// app/inventory/models/stock.go (extended)
type Stock struct {
    IngredientID   string
    Quantity       float64
    Unit           string
    CostPerUnit    Price    // What we paid
    LastUpdated    time.Time
}
```

## Drink Cost Calculation

```go
// app/menu/queries/cost.go
package queries

import (
    drinksq "github.com/TheFellow/go-modular-monolith/app/drinks/queries"
    inventoryq "github.com/TheFellow/go-modular-monolith/app/inventory/queries"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type CostCalculator struct {
    drinks    *drinksq.Queries
    inventory *inventoryq.Queries
}

func NewCostCalculator() *CostCalculator {
    return &CostCalculator{
        drinks:    drinksq.New(),
        inventory: inventoryq.New(),
    }
}

type DrinkCost struct {
    DrinkID        string
    IngredientCost Price
    Breakdown      []IngredientCost
    SuggestedPrice Price // Based on target margin
}

func (c *CostCalculator) Calculate(ctx *middleware.Context, drinkID string, targetMargin float64) (DrinkCost, error)
```

## CLI Output Example

```
$ go run ./main/cli menu show happy-hour --costs

Happy Hour Menu
===============
Status: Published

Drinks:
  Margarita          $12.00  (cost: $2.85, margin: 76%)  [AVAILABLE]
  Old Fashioned      $14.00  (cost: $4.20, margin: 70%)  [AVAILABLE]
  Moscow Mule        $11.00  (cost: $2.40, margin: 78%)  [LIMITED - lime low]
  Whiskey Sour       $13.00  (cost: $3.10, margin: 76%)  [SUBSTITUTION - lemon for lime]
  Mai Tai            $15.00  (cost: $5.50, margin: 63%)  [UNAVAILABLE - missing orgeat]

Analytics:
  Available: 3/5 (60%)
  Average margin: 73%
```

## Success Criteria

- Drinks with substitutions show as "substitution" availability
- Cost calculations are accurate
- Menu analytics display correctly
- `go test ./...` passes

## Dependencies

- Sprint 013c (Simplified constructors)
- Sprint 013d (Unified Commands object)
- Sprint 013e (No Request/Response wrappers)
- Sprint 013g (CedarEntity interface)
- Sprint 014 (Menu curation)
