package currency

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

var all = map[string]Currency{
	"USD": USD,
	"EUR": EUR,
}

func baseForCode(code string) (currency, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return currency{}, errors.Invalidf("currency is required")
	}
	switch code {
	case "USD":
		return currency{code: "USD", symbol: "$", name: "US Dollar"}, nil
	case "EUR":
		return currency{code: "EUR", symbol: "€", name: "Euro"}, nil
	default:
		return currency{}, errors.Invalidf("unknown currency: %s", code)
	}
}

// ParseCode returns a validated, normalized currency code.
func ParseCode(code string) (Code, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	if normalized == "" {
		return "", errors.Invalidf("currency is required")
	}
	if _, ok := all[normalized]; ok {
		return Code(normalized), nil
	}
	return "", errors.Invalidf("unknown currency: %s", normalized)
}

// MustParseCode returns a Code or panics. Use only for static initialization.
func MustParseCode(code string) Code {
	c, err := ParseCode(code)
	if err != nil {
		panic(err)
	}
	return c
}

// Parse returns a Currency from a code string.
func Parse(code string) (Currency, error) {
	parsed, err := ParseCode(code)
	if err != nil {
		return nil, err
	}
	return all[parsed.Code()], nil
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
