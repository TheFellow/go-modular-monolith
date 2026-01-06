# Sprint 026: Decimal Money with Accounting-Style Rounding

## Goal

Replace the current integer-cents `Price` implementation with proper decimal arithmetic using `github.com/govalues/decimal` to ensure accurate financial calculations with accounting-style rounding.

## Status

- Started: 2026-01-06
- Completed: 2026-01-06

## Problem

The current `Price` type stores amounts as integer cents and uses `math.Round()` for arithmetic:

```go
type Price struct {
    Amount   int    // cents: 1234 = $12.34
    Currency string
}

func (p Price) MulFloat(f float64) (Price, error) {
    // Uses math.Round() - accumulates rounding errors
    return Price{Amount: int(math.Round(float64(p.Amount) * f)), Currency: p.Currency}, nil
}
```

Problems:
- **Rounding errors accumulate** in multi-step calculations (cost × quantity × margin)
- **Precision loss** when converting between float and int
- **Non-standard rounding** - `math.Round()` uses banker's rounding, not accounting half-up
- **Limited precision** - integer cents can't represent sub-cent values during intermediate calculations

## Solution

Use `github.com/govalues/decimal` for arbitrary-precision decimal arithmetic:

```go
import "github.com/govalues/decimal"

type Price struct {
    Amount   decimal.Decimal `json:"amount"`
    Currency string          `json:"currency"`
}

func (p Price) Mul(d decimal.Decimal) (Price, error) {
    if p.Currency == "" {
        return Price{}, errors.Invalidf("currency required")
    }
    return Price{Amount: p.Amount.Mul(d), Currency: p.Currency}, nil
}
```

### govalues/decimal Features

- **Arbitrary precision** - No floating-point errors
- **Configurable rounding** - Supports half-up (accounting standard), half-even (banker's), floor, ceiling
- **Immutable** - Thread-safe, no mutation bugs
- **JSON support** - Marshals to string for precision preservation

### Rounding Strategy

Use **half-up rounding** (standard accounting) for final amounts:
- 2.345 → 2.35
- 2.344 → 2.34
- 2.5 → 3 (rounds away from zero)

Preserve full precision during intermediate calculations, round only at display/storage boundaries.

## Changes Required

### Core Price Type

**`app/kernel/money/price.go`:**

```go
package money

import (
    "github.com/govalues/decimal"
    "github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type Price struct {
    Amount   decimal.Decimal `json:"amount"`
    Currency string          `json:"currency"`
}

// NewPrice creates a price from a string amount (e.g., "12.34")
func NewPrice(amount string, currency string) (Price, error) {
    d, err := decimal.Parse(amount)
    if err != nil {
        return Price{}, errors.Invalidf("invalid amount: %w", err)
    }
    return Price{Amount: d, Currency: currency}, nil
}

// NewPriceFromCents creates a price from integer cents (for backwards compatibility)
func NewPriceFromCents(cents int, currency string) Price {
    return Price{
        Amount:   decimal.New(int64(cents), -2), // 1234 → 12.34
        Currency: currency,
    }
}

// Cents returns the amount as integer cents, rounded half-up
func (p Price) Cents() int {
    rounded := p.Amount.Round(2) // Round to 2 decimal places
    cents, _ := rounded.Mul(decimal.New(100, 0)).Int64()
    return int(cents)
}

func (p Price) Add(other Price) (Price, error) {
    if p.Currency != other.Currency {
        return Price{}, errors.Invalidf("currency mismatch: %s vs %s", p.Currency, other.Currency)
    }
    return Price{Amount: p.Amount.Add(other.Amount), Currency: p.Currency}, nil
}

func (p Price) Mul(d decimal.Decimal) (Price, error) {
    if p.Currency == "" {
        return Price{}, errors.Invalidf("currency required")
    }
    return Price{Amount: p.Amount.Mul(d), Currency: p.Currency}, nil
}

func (p Price) Div(d decimal.Decimal) (Price, error) {
    if p.Currency == "" {
        return Price{}, errors.Invalidf("currency required")
    }
    if d.IsZero() {
        return Price{}, errors.Invalidf("division by zero")
    }
    return Price{Amount: p.Amount.Quo(d), Currency: p.Currency}, nil
}

func (p Price) SuggestedPrice(targetMargin float64) (Price, error) {
    if targetMargin >= 1 || targetMargin < 0 {
        return Price{}, errors.Invalidf("margin must be between 0 and 1")
    }
    divisor := decimal.New(int64((1-targetMargin)*10000), -4) // e.g., 0.30 → 0.7000
    suggested := p.Amount.Quo(divisor)
    // Round up to nearest cent for suggested retail
    return Price{Amount: suggested.Ceil(2), Currency: p.Currency}, nil
}

func (p Price) String() string {
    if p.Currency == "" {
        return p.Amount.Round(2).String()
    }
    return fmt.Sprintf("%s %s", p.Currency, p.Amount.Round(2).String())
}

func (p Price) IsZero() bool {
    return p.Amount.IsZero()
}

func (p Price) IsNegative() bool {
    return p.Amount.IsNeg()
}

func (p Price) Validate() error {
    if p.Currency == "" {
        return errors.Invalidf("currency is required")
    }
    if p.Amount.IsNeg() {
        return errors.Invalidf("amount cannot be negative")
    }
    return nil
}
```

### Method Signature Changes

| Old | New |
|-----|-----|
| `MulFloat(f float64)` | `Mul(d decimal.Decimal)` |
| `Amount int` field | `Amount decimal.Decimal` field |

### JSON Serialization

The `decimal.Decimal` type serializes to JSON string by default, preserving precision:

```json
{"amount": "12.34", "currency": "USD"}
```

For backwards compatibility with integer cents, provide custom marshaling if needed.

### CLI Input Parsing

**`main/cli/helpers.go`:**

```go
func parsePrice(s string, currency string) (money.Price, error) {
    // s is like "12.34" or "1234" (cents)
    return money.NewPrice(s, currency)
}
```

### Cost Calculation Updates

**`app/domains/menu/queries/cost.go`:**

```go
// Before
ingredientCost, err = ingredientCost.MulFloat(required)

// After
requiredQty, _ := decimal.Parse(fmt.Sprintf("%f", required))
ingredientCost, err = ingredientCost.Mul(requiredQty)
```

### Analytics Updates

**`app/domains/menu/queries/analytics.go`:**

```go
// Before
m := float64(menuPrice.Amount-cost.IngredientCost.Amount) / float64(menuPrice.Amount)

// After
profit := menuPrice.Amount.Sub(cost.IngredientCost.Amount)
margin := profit.Quo(menuPrice.Amount)
m, _ := margin.Float64() // Convert to float for percentage display
```

### Database Persistence

DAO models can store `decimal.Decimal` directly (implements `BinaryMarshaler`) or store as string for readability:

```go
type MenuItemRow struct {
    // Option A: Store decimal directly
    Price decimal.Decimal

    // Option B: Store as string
    Price string // "12.34"
}
```

## Tasks

- [x] Add `github.com/govalues/decimal` dependency
- [x] Rewrite `Price` struct to use `decimal.Decimal`
- [x] Implement new arithmetic methods (`Add`, `Mul`, `SuggestedPrice`)
- [x] Add `NewPrice()` and `NewPriceFromCents()` constructors
- [x] Add `Cents()` method for backwards compatibility
- [x] Update `String()` method for proper formatting (half-up for display)
- [x] Update `Validate()` for decimal type
- [x] Update CLI `parsePrice` helper
- [x] Update menu cost calculations to use decimal
- [x] Update menu analytics calculations
- [x] Update all tests
- [x] Add precision/rounding tests
- [x] Verify `go test ./...` passes
- [x] Verify CLI operations work end-to-end (requires deleting `data/mixology.db` after schema changes)

## Acceptance Criteria

- `Price.Amount` is `decimal.Decimal`, not `int`
- Arithmetic operations use decimal math with no floating-point conversion
- Rounding uses half-up (accounting standard) at display/storage boundaries
- Multi-step calculations preserve precision: `(cost × qty) + (cost × qty) + ...` is exact
- `SuggestedPrice` rounds up to nearest cent
- All tests pass
- CLI displays prices correctly (e.g., "USD 12.34")
- JSON serialization preserves precision
