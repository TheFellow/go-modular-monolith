package currency

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

// Code is a concrete currency value that can be persisted.
type Code string

func (c Code) Code() string {
	return strings.ToUpper(strings.TrimSpace(string(c)))
}

func (c Code) Symbol() string {
	base, err := baseForCode(c.Code())
	if err != nil {
		return ""
	}
	return base.symbol
}

func (c Code) Name() string {
	base, err := baseForCode(c.Code())
	if err != nil {
		return ""
	}
	return base.name
}

func (c Code) Format(amount string) string {
	curr, err := Parse(c.Code())
	if err != nil {
		code := c.Code()
		if code == "" {
			return amount
		}
		return code + " " + amount
	}
	return curr.Format(amount)
}

func (c Code) String() string {
	return c.Code()
}

func (c Code) Validate() error {
	_, err := ParseCode(c.Code())
	return err
}

func (c Code) IsZero() bool {
	return c.Code() == ""
}

func (c Code) sealed() {}

func (c Code) GobEncode() ([]byte, error) {
	return []byte(c.Code()), nil
}

func (c *Code) GobDecode(data []byte) error {
	if c == nil {
		return errors.Invalidf("currency is required")
	}
	parsed, err := ParseCode(string(data))
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
