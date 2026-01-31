package measurement

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

// DiscreteQuantity represents non-convertible unit counts.
type DiscreteQuantity struct {
	Count float64
	Unit  Unit
}

func NewDiscreteQuantity(count float64, unit Unit) (DiscreteQuantity, error) {
	unit = Unit(strings.TrimSpace(string(unit)))
	if err := unit.Validate(); err != nil {
		return DiscreteQuantity{}, err
	}
	switch unit {
	case UnitDash, UnitPiece, UnitSplash:
		return DiscreteQuantity{Count: count, Unit: unit}, nil
	case UnitMl, UnitOz, UnitCl:
		return DiscreteQuantity{}, errors.Invalidf("unit %q is not discrete", unit)
	default:
		return DiscreteQuantity{}, errors.Invalidf("unknown unit: %s", unit)
	}
}

func MustDiscreteQuantity(count float64, unit Unit) DiscreteQuantity {
	d, err := NewDiscreteQuantity(count, unit)
	if err != nil {
		panic(err)
	}
	return d
}

func (d DiscreteQuantity) Add(other DiscreteQuantity) (DiscreteQuantity, error) {
	if d.Unit != other.Unit {
		return DiscreteQuantity{}, errors.Invalidf("unit mismatch: %s vs %s", d.Unit, other.Unit)
	}
	return DiscreteQuantity{Count: d.Count + other.Count, Unit: d.Unit}, nil
}

func (d DiscreteQuantity) Sub(other DiscreteQuantity) (DiscreteQuantity, error) {
	if d.Unit != other.Unit {
		return DiscreteQuantity{}, errors.Invalidf("unit mismatch: %s vs %s", d.Unit, other.Unit)
	}
	return DiscreteQuantity{Count: d.Count - other.Count, Unit: d.Unit}, nil
}

func (d DiscreteQuantity) Mul(n float64) DiscreteQuantity {
	return DiscreteQuantity{Count: d.Count * n, Unit: d.Unit}
}

func (d DiscreteQuantity) IsZero() bool {
	return d.Count == 0
}

func (d DiscreteQuantity) String() string {
	if d.Count == 1 {
		return fmt.Sprintf("1 %s", d.Unit)
	}
	return fmt.Sprintf("%.0f %ss", d.Count, d.Unit)
}
