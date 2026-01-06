package money

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/govalues/decimal"
)

type Price struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency string          `json:"currency"`
}

func NewPrice(amount string, currency string) (Price, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return Price{}, errors.Invalidf("currency is required")
	}
	d, err := decimal.Parse(strings.TrimSpace(amount))
	if err != nil {
		return Price{}, errors.Invalidf("invalid amount: %w", err)
	}
	p := Price{Amount: d, Currency: currency}
	return p, p.Validate()
}

func NewPriceFromCents(cents int, currency string) Price {
	d, err := decimal.New(int64(cents), 2)
	if err != nil {
		return Price{}
	}
	return Price{Amount: d, Currency: strings.ToUpper(strings.TrimSpace(currency))}
}

func (p Price) Validate() error {
	if p.Amount.IsNeg() {
		return errors.Invalidf("amount must be >= 0")
	}
	if strings.TrimSpace(p.Currency) == "" {
		return errors.Invalidf("currency is required")
	}
	return nil
}

func (p Price) Cents() (int, error) {
	if err := p.Validate(); err != nil {
		return 0, err
	}
	rounded, err := roundHalfUp(p.Amount, 2)
	if err != nil {
		return 0, err
	}
	whole, frac, ok := rounded.Pad(2).Int64(2)
	if !ok {
		return 0, errors.Invalidf("amount out of range")
	}
	return int(whole*100 + frac), nil
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
	sum, err := p.Amount.Add(other.Amount)
	if err != nil {
		return Price{}, err
	}
	return Price{Amount: sum, Currency: p.Currency}, nil
}

func (p Price) Mul(factor decimal.Decimal) (Price, error) {
	if err := p.Validate(); err != nil {
		return Price{}, err
	}
	if factor.IsNeg() {
		return Price{}, errors.Invalidf("factor must be >= 0")
	}
	amt, err := p.Amount.Mul(factor)
	if err != nil {
		return Price{}, err
	}
	return Price{Amount: amt, Currency: p.Currency}, nil
}

func (p Price) SuggestedPrice(targetMargin float64) (Price, error) {
	if err := p.Validate(); err != nil {
		return Price{}, err
	}
	if targetMargin <= 0 || targetMargin >= 1 {
		return Price{}, errors.Invalidf("target margin must be between 0 and 1")
	}
	// Use 4 decimal places for the divisor to keep the calculation stable and exact-ish.
	bp := int64(math.Floor(targetMargin*10000.0 + 0.5)) // basis points with 2 extra decimals
	if bp <= 0 || bp >= 10000 {
		return Price{}, errors.Invalidf("target margin must be between 0 and 1")
	}
	divisor, err := decimal.New(10000-bp, 4) // 0.7000, etc.
	if err != nil {
		return Price{}, errors.Internalf("decimal divisor: %w", err)
	}
	suggested, err := p.Amount.Quo(divisor)
	if err != nil {
		return Price{}, err
	}
	return Price{Amount: suggested.Ceil(2).Pad(2), Currency: p.Currency}, nil
}

func (p Price) String() string {
	if strings.EqualFold(p.Currency, "USD") {
		s, err := p.displayAmount(2)
		if err != nil {
			return "$?"
		}
		return fmt.Sprintf("$%s", s)
	}
	s, err := p.displayAmount(2)
	if err != nil {
		return fmt.Sprintf("%s ?", strings.ToUpper(p.Currency))
	}
	return fmt.Sprintf("%s %s", strings.ToUpper(p.Currency), s)
}

func (p Price) displayAmount(scale int) (string, error) {
	rounded, err := roundHalfUp(p.Amount, scale)
	if err != nil {
		return "", err
	}
	return rounded.Pad(scale).String(), nil
}

func (p Price) IsZero() bool { return p.Amount.IsZero() }

func (p Price) IsNegative() bool { return p.Amount.IsNeg() }

func roundHalfUp(d decimal.Decimal, scale int) (decimal.Decimal, error) {
	if scale < 0 {
		scale = 0
	}
	if scale >= d.Scale() {
		return d.Pad(scale), nil
	}
	ulp, err := decimal.New(5, scale+1) // 0.5 * 10^-scale
	if err != nil {
		return decimal.Decimal{}, err
	}
	if d.IsNeg() {
		// Should not happen for Price amounts, but keep behavior consistent:
		// half-up for negatives == half-away-from-zero.
		n, err := d.Sub(ulp)
		if err != nil {
			return decimal.Decimal{}, err
		}
		return n.Trunc(scale), nil
	}
	n, err := d.Add(ulp)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return n.Trunc(scale), nil
}

func decimalFromFloat64(v float64) (decimal.Decimal, error) {
	// Avoid binary float math during money operations: represent the float as a decimal string first.
	return decimal.Parse(strconv.FormatFloat(v, 'f', -1, 64))
}
