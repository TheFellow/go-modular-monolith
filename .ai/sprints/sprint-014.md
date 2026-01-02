# Sprint 014: Advanced Business Rules & Cost Analysis

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

func (s *AvailabilityService) CalculateDetailed(ctx context.Context, drinkID string) (AvailabilityDetail, error) {
    // ... check primary ingredients

    for _, missing := range missingIngredients {
        // Check for substitutes
        subs, err := s.ingredientQueries.GetSubstitutes(ctx, missing.IngredientID)
        if err != nil {
            continue
        }

        for _, sub := range subs {
            // Check if substitute is in stock
            stock, err := s.inventoryQueries.GetStock(ctx, sub.SubstituteID)
            if err != nil || stock.Quantity < (missing.Required * sub.Ratio) {
                continue
            }

            // Found viable substitute
            detail.Substitutions = append(detail.Substitutions, AppliedSubstitution{
                Original:      missing.IngredientID,
                Substitute:    sub.SubstituteID,
                Ratio:         sub.Ratio,
                QualityImpact: sub.QualityImpact,
            })
            break
        }
    }

    // ... determine final status
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
    LowThreshold   float64
    ReorderPoint   float64
    LastUpdated    time.Time
    LastCostUpdate time.Time
}
```

## Drink Cost Calculation

```go
// app/menu/queries/cost.go
type DrinkCost struct {
    DrinkID        string
    IngredientCost Price           // Sum of ingredient costs
    Breakdown      []IngredientCost
    SuggestedPrice Price           // Based on target margin
}

type IngredientCost struct {
    IngredientID string
    Amount       float64
    UnitCost     Price
    TotalCost    Price
}

func (q *Queries) CalculateDrinkCost(ctx context.Context, drinkID string, targetMargin float64) (DrinkCost, error) {
    drink, _ := q.drinkQueries.Get(ctx, drinkID)

    var totalCost int
    var breakdown []IngredientCost

    for _, ri := range drink.Recipe.Ingredients {
        stock, _ := q.inventoryQueries.GetStock(ctx, ri.IngredientID)

        ingredientCost := int(float64(stock.CostPerUnit.Amount) * ri.Amount)
        totalCost += ingredientCost

        breakdown = append(breakdown, IngredientCost{
            IngredientID: ri.IngredientID,
            Amount:       ri.Amount,
            UnitCost:     stock.CostPerUnit,
            TotalCost:    Price{Amount: ingredientCost, Currency: "USD"},
        })
    }

    // Calculate suggested price for target margin
    // margin = (price - cost) / price
    // price = cost / (1 - margin)
    suggestedPrice := int(float64(totalCost) / (1 - targetMargin))

    return DrinkCost{
        DrinkID:        drinkID,
        IngredientCost: Price{Amount: totalCost, Currency: "USD"},
        Breakdown:      breakdown,
        SuggestedPrice: Price{Amount: suggestedPrice, Currency: "USD"},
    }, nil
}
```

## Menu Analytics

```go
// app/menu/queries/analytics.go
type MenuAnalytics struct {
    MenuID            string
    TotalDrinks       int
    AvailableCount    int
    UnavailableCount  int
    LimitedCount      int
    AverageCost       Price
    AveragePrice      Price
    AverageMargin     float64
    MostExpensive     MenuItem
    LeastExpensive    MenuItem
    HighestMargin     MenuItem
    LowestMargin      MenuItem
}

func (q *Queries) GetMenuAnalytics(ctx context.Context, menuID string) (MenuAnalytics, error)
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
  Total potential revenue: $65.00
```

## Events

- `SubstitutionApplied{MenuID, DrinkID, Original, Substitute}` - logged when drink served with substitute
- `CostThresholdExceeded{DrinkID, Cost, Threshold}` - ingredient costs exceeded target

## Notes

This sprint demonstrates:
1. Complex cross-context queries (Drinks + Ingredients + Inventory → Menu)
2. Business rule engines (substitution logic)
3. Derived calculations (cost analysis)
4. Analytics/reporting patterns

## Success Criteria

- Drinks with substitutions show as "substitution" availability
- Cost calculations are accurate
- Menu analytics display correctly
- `go test ./...` passes

## Dependencies

- Sprint 013 (Event handlers wired up)
