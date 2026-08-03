package money

import (
	"strings"
	"unicode"

	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

// ParsePrice accepts the user-facing forms $1.23, USD 1.23, and 1.23 USD.
func ParsePrice(raw string) (Price, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Price{}, errors.Invalidf("price is required")
	}
	if amount, ok := strings.CutPrefix(raw, "$"); ok {
		return NewPrice(strings.TrimSpace(amount), currency.USD)
	}
	parts := strings.Fields(raw)
	if len(parts) != 2 {
		return Price{}, invalidPrice(raw)
	}
	code, amount := parts[0], parts[1]
	if !currencyCode(code) {
		code, amount = parts[1], parts[0]
	}
	if !currencyCode(code) {
		return Price{}, invalidPrice(raw)
	}
	curr, err := currency.Parse(strings.ToUpper(code))
	if err != nil {
		return Price{}, err
	}
	return NewPrice(amount, curr)
}

func currencyCode(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, r := range value {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func invalidPrice(raw string) error {
	return errors.Invalidf("invalid price %q (expected \"$1.23\" or \"USD 1.23\" or \"1.23 USD\")", raw)
}
