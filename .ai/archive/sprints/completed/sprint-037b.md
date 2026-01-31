# Sprint 037b: Currency Package (Simplified)

## Goal

Create `app/kernel/currency` package with a simple struct type that supports comparison and owns its formatting behavior.

## Revision

Sprint 037 proposed a sealed interface for polymorphic formatting. This revision simplifies to a struct with a format string - data-driven rather than code-driven.

## Solution

```go
// app/kernel/currency/currency.go

package currency

import (
    "encoding/json"
    "fmt"
    "strings"

    "github.com/TheFellow/go-modular-monolith/pkg/errors"
)

// Currency represents a monetary currency.
// Zero value is invalid - use the provided constants.
type Currency struct {
    Code          string
    Symbol        string
    Name          string
    FormatPattern string // e.g., "$%s" or "%s €"
}

// Supported currencies
var (
    USD = Currency{Code: "USD", Symbol: "$", Name: "US Dollar", FormatPattern: "$%s"}
    EUR = Currency{Code: "EUR", Symbol: "€", Name: "Euro", FormatPattern: "%s €"}
)

// all registered currencies for lookup
var all = map[string]Currency{
    "USD": USD,
    "EUR": EUR,
}

// Parse returns a Currency from a code string.
func Parse(code string) (Currency, error) {
    code = strings.ToUpper(strings.TrimSpace(code))
    if code == "" {
        return Currency{}, errors.Invalidf("currency is required")
    }
    if c, ok := all[code]; ok {
        return c, nil
    }
    return Currency{}, errors.Invalidf("unknown currency: %s", code)
}

// MustParse returns a Currency or panics.
func MustParse(code string) Currency {
    c, err := Parse(code)
    if err != nil {
        panic(err)
    }
    return c
}

// String implements fmt.Stringer.
func (c Currency) String() string { return strings.ToUpper(strings.TrimSpace(c.Code)) }

// IsZero returns true if this is the zero value.
func (c Currency) IsZero() bool { return strings.TrimSpace(c.Code) == "" }

// Format returns the amount formatted with the currency symbol.
func (c Currency) Format(amount string) string {
    if c.FormatPattern != "" {
        return fmt.Sprintf(c.FormatPattern, amount)
    }
    return strings.ToUpper(strings.TrimSpace(c.Code)) + " " + amount
}

// MarshalJSON serializes to the currency code.
func (c Currency) MarshalJSON() ([]byte, error) {
    if c.IsZero() {
        return []byte("null"), nil
    }
    return json.Marshal(strings.ToUpper(strings.TrimSpace(c.Code)))
}

// UnmarshalJSON deserializes from a currency code.
func (c *Currency) UnmarshalJSON(data []byte) error {
    var code string
    if err := json.Unmarshal(data, &code); err != nil {
        return err
    }
    if code == "" {
        *c = Currency{}
        return nil
    }
    parsed, err := Parse(code)
    if err != nil {
        return err
    }
    *c = parsed
    return nil
}

func (c Currency) Validate() error {
    _, err := Parse(c.Code)
    return err
}
```

## Usage

```go
// Simple comparison
if price.Currency != currency.USD {
    // handle non-USD
}

// Owned formatting behavior
fmt.Println(currency.USD.Format("12.50"))  // "$12.50"
fmt.Println(currency.EUR.Format("12.50"))  // "12.50 €"

// Public fields for simple access; validate if needed
c := currency.Currency{Code: "FAKE"}
if err := c.Validate(); err != nil {
    // handle invalid currency
}

// Parse from user input
curr, err := currency.Parse("EUR")

// Metadata
fmt.Println(currency.USD.Symbol)  // "$"
fmt.Println(currency.EUR.Name)    // "Euro"
```

## Updated Price Type

```go
// app/kernel/money/price.go

type Price struct {
    Amount   decimal.Decimal   `json:"amount"`
    Currency currency.Currency `json:"currency"`
}

// String delegates formatting to the currency.
func (p Price) String() string {
    s, err := p.displayAmount(2)
    if err != nil {
        return p.Currency.Code + " ?"
    }
    return p.Currency.Format(s)
}
```

No custom `UnmarshalJSON` needed on Price - the `Currency` struct handles its own JSON.

## File Structure

```
app/kernel/currency/
├── currency.go      # Currency type, constants, Parse, JSON
└── currency_test.go # Unit tests
```

## Adding New Currencies

```go
var (
    USD = Currency{Code: "USD", Symbol: "$", Name: "US Dollar", FormatPattern: "$%s"}
    EUR = Currency{Code: "EUR", Symbol: "€", Name: "Euro", FormatPattern: "%s €"}
    GBP = Currency{Code: "GBP", Symbol: "£", Name: "British Pound", FormatPattern: "£%s"}  // Add
)

var all = map[string]Currency{
    "USD": USD,
    "EUR": EUR,
    "GBP": GBP,  // Add
}
```

One line for the constant, one line in the registry. Done.

## Tasks

- [x] Create `app/kernel/currency/currency.go`
- [x] Add `USD` and `EUR` constants with format strings
- [x] Add `Parse()`, `MustParse()` functions
- [x] Expose `Code`, `Symbol`, `Name`, `FormatPattern` and add `Format()`, `String()`, `IsZero()`, `Validate()` methods
- [x] Add `MarshalJSON()` and `UnmarshalJSON()` methods
- [x] Add unit tests
- [x] Update `money.Price` to use `currency.Currency`
- [x] Update `Price.String()` to use `Currency.Format()`
- [x] Remove currency if-statements from money package
- [x] Update CLI helpers
- [x] Run all tests

## Acceptance Criteria

- [x] `currency.USD != currency.EUR` works for comparison
- [x] `currency.USD.Format("12.50")` returns `"$12.50"`
- [x] `currency.EUR.Format("12.50")` returns `"12.50 €"`
- [x] `Currency.Validate()` rejects unknown codes
- [x] JSON round-trips correctly
- [x] `Price.String()` has no currency-specific if-statements
- [x] All tests pass

## Comparison: 037 vs 037b

| Aspect | 037 (Interface) | 037b (Struct) |
|--------|-----------------|---------------|
| Comparison | `c.Code() == other.Code()` | `c == other` |
| Formatting | Polymorphic `Format()` method | Format string + `fmt.Sprintf` |
| Adding currency | New file with type | One line |
| JSON unmarshal | Custom on Price | Built into Currency |
| Complexity | Higher | Lower |
