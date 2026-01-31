# Sprint 035: Unit Value Objects with Conversion

## Goal

Replace primitive `float64` + `string` unit representations with strongly-typed value objects that support unit conversion.

## Problem

Current unit handling uses primitives:

```go
// inventory/models/inventory.go
type Inventory struct {
    Quantity float64
    Unit     measurement.Unit  // just a validated string
}

// drinks/models/recipe.go
type RecipeIngredient struct {
    Amount float64
    Unit   measurement.Unit
}
```

This has several limitations:

1. **No conversion logic** - Can't convert 1 oz to 29.57 ml
2. **No type safety for arithmetic** - Can accidentally add ml to oz
3. **Recipe costing is fragile** - If inventory is in ml but recipe is in oz, calculations fail silently

```go
// Current: hope units match
stock := 500.0  // ml
recipe := 1.5   // oz - oops, different units!
servings := stock / recipe  // Wrong! 333 servings instead of ~11
```

## Solution

Create value objects that encapsulate quantity + unit with conversion:

```go
// pkg/measurement/volume.go

package measurement

import "github.com/TheFellow/go-modular-monolith/pkg/errors"

// Volume represents a liquid measurement with automatic conversion.
type Volume struct {
    ml float64 // internal canonical unit
}

// Volume constructors
func Milliliters(v float64) Volume { return Volume{ml: v} }
func Ounces(v float64) Volume      { return Volume{ml: v * 29.5735} }
func Centiliters(v float64) Volume { return Volume{ml: v * 10} }

// Accessors - get value in any unit
func (v Volume) Ml() float64 { return v.ml }
func (v Volume) Oz() float64 { return v.ml / 29.5735 }
func (v Volume) Cl() float64 { return v.ml / 10 }

// Arithmetic
func (v Volume) Add(other Volume) Volume { return Volume{ml: v.ml + other.ml} }
func (v Volume) Sub(other Volume) Volume { return Volume{ml: v.ml - other.ml} }
func (v Volume) Mul(n float64) Volume    { return Volume{ml: v.ml * n} }
func (v Volume) Div(n float64) Volume    { return Volume{ml: v.ml / n} }

// Comparison
func (v Volume) IsZero() bool            { return v.ml == 0 }
func (v Volume) LessThan(other Volume) bool { return v.ml < other.ml }

// String returns a human-readable representation
func (v Volume) String() string {
    if v.ml >= 100 {
        return fmt.Sprintf("%.0f ml", v.ml)
    }
    return fmt.Sprintf("%.1f oz", v.Oz())
}
```

### Unit-Aware Quantity

For storage and display, track the preferred display unit:

```go
// pkg/measurement/quantity.go

// VolumeUnit is a display unit preference
type VolumeUnit string

const (
    UnitMl VolumeUnit = "ml"
    UnitOz VolumeUnit = "oz"
    UnitCl VolumeUnit = "cl"
)

// Quantity pairs a Volume with its preferred display unit
type Quantity struct {
    Volume Volume
    Unit   VolumeUnit
}

// NewQuantity creates a quantity from a value and unit
func NewQuantity(value float64, unit VolumeUnit) (Quantity, error) {
    var vol Volume
    switch unit {
    case UnitMl:
        vol = Milliliters(value)
    case UnitOz:
        vol = Ounces(value)
    case UnitCl:
        vol = Centiliters(value)
    default:
        return Quantity{}, errors.Invalidf("unknown unit: %s", unit)
    }
    return Quantity{Volume: vol, Unit: unit}, nil
}

// Value returns the quantity in its preferred unit
func (q Quantity) Value() float64 {
    switch q.Unit {
    case UnitOz:
        return q.Volume.Oz()
    case UnitCl:
        return q.Volume.Cl()
    default:
        return q.Volume.Ml()
    }
}

// String returns "1.5 oz" format
func (q Quantity) String() string {
    return fmt.Sprintf("%.2f %s", q.Value(), q.Unit)
}

// Convert returns a new Quantity with a different display unit
func (q Quantity) Convert(unit VolumeUnit) Quantity {
    return Quantity{Volume: q.Volume, Unit: unit}
}
```

### Discrete Units (Non-Convertible)

Some units aren't convertible (dash, piece, splash):

```go
// pkg/measurement/discrete.go

// DiscreteUnit represents non-convertible units
type DiscreteUnit string

const (
    UnitDash   DiscreteUnit = "dash"
    UnitPiece  DiscreteUnit = "piece"
    UnitSplash DiscreteUnit = "splash"
)

// DiscreteQuantity is a count of discrete units
type DiscreteQuantity struct {
    Count float64
    Unit  DiscreteUnit
}

func (d DiscreteQuantity) String() string {
    if d.Count == 1 {
        return fmt.Sprintf("1 %s", d.Unit)
    }
    return fmt.Sprintf("%.0f %ss", d.Count, d.Unit)
}
```

### Combined Amount Type

For recipes that mix volume and discrete:

```go
// pkg/measurement/amount.go

// Amount represents either a Volume or Discrete quantity
type Amount struct {
    volume   *Quantity
    discrete *DiscreteQuantity
}

func VolumeAmount(q Quantity) Amount {
    return Amount{volume: &q}
}

func DiscreteAmount(d DiscreteQuantity) Amount {
    return Amount{discrete: &d}
}

func (a Amount) IsVolume() bool   { return a.volume != nil }
func (a Amount) IsDiscrete() bool { return a.discrete != nil }

func (a Amount) Volume() (Quantity, bool) {
    if a.volume == nil {
        return Quantity{}, false
    }
    return *a.volume, true
}

func (a Amount) Discrete() (DiscreteQuantity, bool) {
    if a.discrete == nil {
        return DiscreteQuantity{}, false
    }
    return *a.discrete, true
}

func (a Amount) String() string {
    if a.volume != nil {
        return a.volume.String()
    }
    if a.discrete != nil {
        return a.discrete.String()
    }
    return "0"
}
```

## Usage Examples

### Recipe Costing

```go
// Before: hope units match
func calculateCost(stock Inventory, recipe RecipeIngredient) float64 {
    // BUG: stock.Unit might be "ml", recipe.Unit might be "oz"
    return (recipe.Amount / stock.Quantity) * stock.CostPerUnit
}

// After: automatic conversion
func calculateCost(stock Inventory, recipe RecipeIngredient) money.Price {
    // Both are Quantity - conversion happens automatically
    ratio := recipe.Amount.Volume.Ml() / stock.Amount.Volume.Ml()
    return stock.CostPerUnit.Mul(ratio)
}
```

### Inventory Adjustment

```go
// Before
stock.Quantity += delta  // what if units differ?

// After
stock.Amount = stock.Amount.Add(delta)  // type-safe, auto-converts
```

### Recipe Display

```go
// Display in preferred unit
fmt.Printf("Add %s of vodka\n", recipe.Amount.String())
// Output: "Add 1.50 oz of vodka"

// Convert for display
metric := recipe.Amount.Convert(measurement.UnitMl)
fmt.Printf("Add %s of vodka\n", metric.String())
// Output: "Add 44.36 ml of vodka"
```

## Model Updates

### Inventory

```go
// Before
type Inventory struct {
    IngredientID entity.IngredientID
    Quantity     float64
    Unit         measurement.Unit
    CostPerUnit  optional.Value[money.Price]
}

// After
type Inventory struct {
    IngredientID entity.IngredientID
    Amount       measurement.Quantity  // or measurement.Amount for discrete support
    CostPerUnit  optional.Value[money.Price]
}
```

### Recipe

```go
// Before
type RecipeIngredient struct {
    IngredientID entity.IngredientID
    Amount       float64
    Unit         measurement.Unit
    // ...
}

// After
type RecipeIngredient struct {
    IngredientID entity.IngredientID
    Amount       measurement.Amount  // supports both volume and discrete
    // ...
}
```

## Migration Strategy

1. **Create new types** in `pkg/measurement/`
2. **Add conversion helpers** for existing code
3. **Update models** one at a time
4. **Update DAO conversions** to serialize/deserialize properly
5. **Keep CLI interface stable** (still accept "1.5 oz" strings)

## Tasks

### Phase 1: Create Value Objects

- [x] Create `pkg/measurement/volume.go` with `Volume` type
- [x] Create `pkg/measurement/quantity.go` with `Quantity` type
- [x] Create `pkg/measurement/discrete.go` with `DiscreteQuantity` type
- [x] Create `pkg/measurement/amount.go` with `Amount` type (optional, if needed)
- [x] Add unit tests for conversions

### Phase 2: Update Inventory Domain

- [x] Update `inventory/models/inventory.go` to use `Quantity`
- [x] Update `inventory/internal/dao` conversions
- [x] Update inventory commands and queries
- [x] Update CLI output

### Phase 3: Update Drinks Domain (Recipes)

- [x] Update `drinks/models/recipe.go` to use `Amount`
- [x] Update `drinks/internal/dao` conversions
- [x] Update recipe JSON serialization in CLI surface

### Phase 4: Update Ingredients Domain

- [x] Update `ingredients/models/ingredient.go` default unit
- [x] Ensure consistency with inventory

### Phase 5: Enable Cross-Unit Operations

- [x] Update menu analytics to use proper conversions
- [x] Update cost calculations to handle unit conversion
- [x] Verify inventory checks work across units

### Phase 6: Verify

- [x] Run all tests
- [x] Test CLI with various unit inputs
- [x] Verify cost calculations are correct

## Acceptance Criteria

- [x] `Volume` type with ml/oz/cl conversion
- [x] `Quantity` type preserving display unit preference
- [x] `DiscreteQuantity` for non-convertible units (dash, piece, splash)
- [x] Inventory uses `Quantity` instead of `float64` + `Unit`
- [x] Recipes use `Amount` supporting both volume and discrete
- [x] Cost calculations work correctly across units
- [x] CLI accepts same input formats (backwards compatible)
- [x] All tests pass

## Result

```go
// Type-safe unit handling
stock := measurement.Milliliters(500)
recipe := measurement.Ounces(1.5)

// Automatic conversion
servings := stock.Ml() / recipe.Ml()  // ~11 servings (correct!)

// Arithmetic is safe
remaining := stock.Sub(recipe.Mul(5))  // 500ml - (5 × 1.5oz) = ~278ml

// Display in any unit
fmt.Println(remaining.String())  // "278 ml"
fmt.Println(measurement.Quantity{Volume: remaining, Unit: measurement.UnitOz}.String())  // "9.40 oz"
```
