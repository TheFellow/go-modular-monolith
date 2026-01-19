package currency

import (
	"encoding/json"
	"fmt"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type Code string

// Currency represents a monetary currency.
// Zero value is invalid - use the provided constants or Parse().
type Currency struct {
	Code          Code
	Symbol        string
	Name          string
	FormatPattern string
}

const (
	CodeUSD Code = "USD"
	CodeEUR Code = "EUR"
)

var (
	USD = Currency{Code: CodeUSD, Symbol: "$", Name: "US Dollar", FormatPattern: "$%s"}
	EUR = Currency{Code: CodeEUR, Symbol: "€", Name: "Euro", FormatPattern: "%s €"}
)

// Parse returns a Currency from a string
func Parse(code string) (Currency, error) {
	return FromCode(Code(code))
}

// FromCode returns a Currency from a Code
func FromCode(code Code) (Currency, error) {
	switch code {
	case CodeUSD:
		return USD, nil
	case CodeEUR:
		return EUR, nil
	}
	return Currency{}, errors.Invalidf("unknown currency: %s", code)
}

// Format returns the amount formatted with the currency symbol.
func (c Currency) Format(amount string) string {
	return fmt.Sprintf(c.FormatPattern, amount)
}

// String implements fmt.Stringer.
func (c Currency) String() string {
	return string(c.Code)
}

// IsZero reports whether the currency is unset.
func (c Currency) IsZero() bool {
	return string(c.Code) == ""
}

func (c Currency) Validate() error {
	_, err := FromCode(c.Code)
	return err
}

// MarshalJSON serializes to the currency code.
func (c Currency) MarshalJSON() ([]byte, error) {
	if c.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(c.Code)
}

// UnmarshalJSON deserializes from a currency code.
func (c *Currency) UnmarshalJSON(data []byte) error {
	var code *string
	if err := json.Unmarshal(data, &code); err != nil {
		return err
	}
	if code == nil {
		return errors.Invalidf("currency code is required")
	}
	parsed, err := Parse(*code)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
