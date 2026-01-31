# Sprint 037: Currency as Kernel Package

## Goal

Extract currency into its own kernel package using a sealed interface pattern where each currency type owns its formatting behavior.

## Problem

Currency is currently a raw string in the money package:

```go
// app/kernel/money/price.go
type Price struct {
    Amount   decimal.Decimal `json:"amount"`
    Currency string          `json:"currency"`  // Just a string!
}
```

This leads to leaked domain knowledge throughout the codebase:

```go
// BAD: Formatting logic scattered everywhere
func (p Price) String() string {
    if p.Currency == "USD" {
        return fmt.Sprintf("$%s", amount)  // USD: symbol prefix
    }
    return fmt.Sprintf("%s %s", amount, p.Currency)  // Others: symbol suffix
}
```

Every place that formats a price needs to know USD uses prefix symbols while EUR uses suffix. This is domain knowledge that belongs to the currency itself.

## Solution

Create `app/kernel/currency` package with a **sealed interface** where each currency type owns its behavior:

```go
// app/kernel/currency/currency.go

package currency

import "encoding/json"

// Currency represents a monetary currency.
// This is a sealed interface - only types in this package can implement it.
type Currency interface {
    Code() string
    Symbol() string
    Name() string
    Format(amount string) string

    // sealed prevents external implementations
    sealed()
}

// currency is the base implementation embedded by all currency types.
type currency struct {
    code   string
    symbol string
    name   string
}

func (c currency) Code() string   { return c.code }
func (c currency) Symbol() string { return c.symbol }
func (c currency) Name() string   { return c.name }
func (c currency) String() string { return c.code }
func (c currency) sealed()        {}

// MarshalJSON serializes to the currency code string.
func (c currency) MarshalJSON() ([]byte, error) {
    return json.Marshal(c.code)
}
```

### Currency Types

Each currency is its own type with specific formatting:

```go
// app/kernel/currency/usd.go

package currency

// usd represents the US Dollar.
type usd struct{ currency }

// Format returns the amount with $ prefix: "$12.50"
func (u usd) Format(amount string) string {
    return u.symbol + amount
}

// USD is the US Dollar currency.
var USD Currency = usd{currency{
    code:   "USD",
    symbol: "$",
    name:   "US Dollar",
}}
```

```go
// app/kernel/currency/eur.go

package currency

// eur represents the Euro.
type eur struct{ currency }

// Format returns the amount with € suffix: "12.50 €"
func (e eur) Format(amount string) string {
    return amount + " " + e.symbol
}

// EUR is the Euro currency.
var EUR Currency = eur{currency{
    code:   "EUR",
    symbol: "€",
    name:   "Euro",
}}
```

### Registry and Parsing

```go
// app/kernel/currency/registry.go

package currency

import (
    "strings"

    "github.com/TheFellow/go-modular-monolith/pkg/errors"
)

// all registered currencies for lookup by code
var all = map[string]Currency{
    "USD": USD,
    "EUR": EUR,
}

// Parse returns a Currency from a code string.
func Parse(code string) (Currency, error) {
    code = strings.ToUpper(strings.TrimSpace(code))
    if c, ok := all[code]; ok {
        return c, nil
    }
    return nil, errors.Invalidf("unknown currency: %s", code)
}

// MustParse returns a Currency or panics. Use only for static initialization.
func MustParse(code string) Currency {
    c, err := Parse(code)
    if err != nil {
        panic(err)
    }
    return c
}

// All returns all supported currencies.
func All() []Currency {
    result := make([]Currency, 0, len(all))
    for _, c := range all {
        result = append(result, c)
    }
    return result
}
```

### Updated Price Type

```go
// app/kernel/money/price.go

package money

import (
    "encoding/json"

    "github.com/TheFellow/go-modular-monolith/app/kernel/currency"
    // ...
)

type Price struct {
    Amount   decimal.Decimal   `json:"amount"`
    Currency currency.Currency `json:"currency"`
}

func NewPrice(amount string, curr currency.Currency) (Price, error) {
    if curr == nil {
        return Price{}, errors.Invalidf("currency is required")
    }
    d, err := decimal.Parse(strings.TrimSpace(amount))
    if err != nil {
        return Price{}, errors.Invalidf("invalid amount: %w", err)
    }
    p := Price{Amount: d, Currency: curr}
    return p, p.Validate()
}

func (p Price) Validate() error {
    if p.Amount.IsNeg() {
        return errors.Invalidf("amount must be >= 0")
    }
    if p.Currency == nil {
        return errors.Invalidf("currency is required")
    }
    return nil
}

func (p Price) Add(other Price) (Price, error) {
    // ...
    if p.Currency.Code() != other.Currency.Code() {
        return Price{}, errors.Invalidf("currency mismatch: %s vs %s",
            p.Currency.Code(), other.Currency.Code())
    }
    // ...
}

// String delegates formatting entirely to the currency.
func (p Price) String() string {
    s, err := p.displayAmount(2)
    if err != nil {
        return p.Currency.Code() + " ?"
    }
    return p.Currency.Format(s)
}

// UnmarshalJSON handles deserializing the currency code back to a Currency.
func (p *Price) UnmarshalJSON(data []byte) error {
    var raw struct {
        Amount   decimal.Decimal `json:"amount"`
        Currency string          `json:"currency"`
    }
    if err := json.Unmarshal(data, &raw); err != nil {
        return err
    }
    curr, err := currency.Parse(raw.Currency)
    if err != nil {
        return err
    }
    p.Amount = raw.Amount
    p.Currency = curr
    return nil
}
```

## Usage Examples

```go
// Creating prices - currency owns its formatting
usdPrice, _ := money.NewPrice("9.99", currency.USD)
eurPrice, _ := money.NewPrice("19.99", currency.EUR)

fmt.Println(usdPrice.String())  // "$9.99"     - USD formats with prefix
fmt.Println(eurPrice.String())  // "19.99 €"   - EUR formats with suffix

// No if-statements needed - polymorphism handles it
func displayPrice(p money.Price) {
    fmt.Println(p.Currency.Format(amount))  // Each currency knows its format
}

// Parse from user input
curr, err := currency.Parse("EUR")
if err != nil {
    return err
}
price, err := money.NewPrice("19.99", curr)

// Currency metadata via interface methods
fmt.Println(currency.USD.Code())    // "USD"
fmt.Println(currency.USD.Symbol())  // "$"
fmt.Println(currency.USD.Name())    // "US Dollar"

// Sealed interface - cannot implement outside package
type fakeCurrency struct{}
func (f fakeCurrency) Code() string { return "FAKE" }
// ERROR: fakeCurrency doesn't implement Currency (missing sealed method)
```

## File Structure

```
app/kernel/
├── currency/
│   ├── currency.go      # Currency interface and currency base type (with MarshalJSON)
│   ├── usd.go           # USD type with prefix formatting
│   ├── eur.go           # EUR type with suffix formatting
│   ├── registry.go      # Parse, MustParse, All functions
│   └── currency_test.go # Unit tests
└── money/
    └── price.go         # Updated with custom UnmarshalJSON for Currency
```

## Migration

Update callers from string to `currency.Currency`:

```go
// Before
price, err := money.NewPrice("9.99", "USD")
if p.Currency == "USD" {
    fmt.Printf("$%s", amount)
}

// After
price, err := money.NewPrice("9.99", currency.USD)
fmt.Println(price.String())  // Currency handles formatting
```

## Tasks

### Phase 1: Create Currency Package

- [x] Create `app/kernel/currency/currency.go` with sealed `Currency` interface
- [x] Create `currency` base struct with shared implementation and `MarshalJSON`
- [x] Create `app/kernel/currency/usd.go` with `usd` type and `USD` constant
- [x] Create `app/kernel/currency/eur.go` with `eur` type and `EUR` constant
- [x] Create `app/kernel/currency/registry.go` with `Parse()`, `MustParse()`, `All()`
- [x] Add unit tests for formatting and parsing

### Phase 2: Update Money Package

- [x] Update `Price.Currency` field to `currency.Code` (implements `currency.Currency`) for persistence
- [x] Update `NewPrice()` to accept `currency.Currency`
- [x] Add `NewPriceFromString()` for string input
- [x] Update `Validate()` to enforce a required/known currency
- [x] Update `Add()` to compare by `Code()`
- [x] Update `String()` to delegate to `Currency.Format()`
- [x] Add `UnmarshalJSON` to Price for deserializing currency codes
- [x] Remove all currency-specific if-statements
- [x] Update tests

### Phase 3: Update Callers

- [x] Update CLI helpers (`parsePrice` function)
- [x] Update any domain code using `Price`
- [x] Update DAO conversions if needed

### Phase 4: Verify

- [x] Run all tests
- [x] Test CLI price parsing with USD and EUR
- [x] Verify JSON serialization round-trips correctly

## Acceptance Criteria

- [x] `currency.USD` and `currency.EUR` are available as interface values
- [x] Cannot implement `Currency` outside the package (sealed)
- [x] Each currency type owns its `Format()` implementation
- [x] `Price.String()` delegates formatting to the currency implementation
- [x] JSON marshaling writes currency codes like `"USD"`
- [x] JSON unmarshaling resolves codes via `currency.ParseCode()`
- [x] All tests pass

## Result

```go
// Polymorphic formatting - no leaked domain knowledge
usd, _ := money.NewPrice("12.50", currency.USD)
eur, _ := money.NewPrice("12.50", currency.EUR)

fmt.Println(usd.String())  // "$12.50"   - USD.Format() adds prefix
fmt.Println(eur.String())  // "12.50 €"  - EUR.Format() adds suffix

// Price.String() is simply:
func (p Price) String() string {
    return p.Currency.Format(amount)  // No if-statements!
}

// Each currency owns its behavior
type usd struct{ currency }
func (u usd) Format(amount string) string { return u.symbol + amount }

type eur struct{ currency }
func (e eur) Format(amount string) string { return amount + " " + e.symbol }
```

## Future: Adding Currencies

To add a new currency (e.g., GBP):

```go
// app/kernel/currency/gbp.go

package currency

// gbp represents the British Pound.
type gbp struct{ currency }

// Format returns the amount with £ prefix: "£12.50"
func (g gbp) Format(amount string) string {
    return g.symbol + amount
}

// GBP is the British Pound currency.
var GBP Currency = gbp{currency{
    code:   "GBP",
    symbol: "£",
    name:   "British Pound",
}}

// Then add to registry.go:
var all = map[string]Currency{
    "USD": USD,
    "EUR": EUR,
    "GBP": GBP,  // Add
}
```

Each new currency defines its own formatting behavior. No other code needs to change.
