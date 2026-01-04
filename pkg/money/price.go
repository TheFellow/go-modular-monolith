package money

import (
	"fmt"
	"math"
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type Price struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

func (p Price) Validate() error {
	if p.Amount < 0 {
		return errors.Invalidf("amount must be >= 0")
	}
	if strings.TrimSpace(p.Currency) == "" {
		return errors.Invalidf("currency is required")
	}
	return nil
}

func (p Price) Add(other Price) (Price, error) {
	if err := p.Validate(); err != nil {
		return Price{}, err
	}
	if err := other.Validate(); err != nil {
		return Price{}, err
	}
	if p.Currency != other.Currency {
		return Price{}, errors.Invalidf("currency mismatch: %s vs %s", p.Currency, other.Currency)
	}
	return Price{Amount: p.Amount + other.Amount, Currency: p.Currency}, nil
}

func (p Price) MulFloat(f float64) (Price, error) {
	if err := p.Validate(); err != nil {
		return Price{}, err
	}
	if f < 0 {
		return Price{}, errors.Invalidf("factor must be >= 0")
	}
	return Price{Amount: int(math.Round(float64(p.Amount) * f)), Currency: p.Currency}, nil
}

func (p Price) SuggestedPrice(targetMargin float64) (Price, error) {
	if err := p.Validate(); err != nil {
		return Price{}, err
	}
	if targetMargin <= 0 || targetMargin >= 1 {
		return Price{}, errors.Invalidf("target margin must be between 0 and 1")
	}
	return Price{Amount: int(math.Ceil(float64(p.Amount) / (1 - targetMargin))), Currency: p.Currency}, nil
}

func (p Price) String() string {
	if strings.EqualFold(p.Currency, "USD") {
		return fmt.Sprintf("$%.2f", float64(p.Amount)/100.0)
	}
	return fmt.Sprintf("%s %.2f", strings.ToUpper(p.Currency), float64(p.Amount)/100.0)
}
