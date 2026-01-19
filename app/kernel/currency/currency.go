package currency

import (
	"encoding/json"
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

// Currency represents a monetary currency.
// This is a sealed interface - only types in this package can implement it.
type Currency interface {
	Code() string
	Symbol() string
	Name() string
	Format(amount string) string

	// sealed prevents external implementations.
	sealed()
}

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

func (c currency) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.code)
}

func (c currency) GobEncode() ([]byte, error) {
	return []byte(c.code), nil
}

func (c *currency) GobDecode(data []byte) error {
	if c == nil {
		return errors.Invalidf("currency is required")
	}
	code := strings.ToUpper(strings.TrimSpace(string(data)))
	base, err := baseForCode(code)
	if err != nil {
		return err
	}
	*c = base
	return nil
}
