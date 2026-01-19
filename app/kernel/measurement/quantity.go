package measurement

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

// Quantity pairs a Volume with its preferred display unit.
type Quantity struct {
	Volume Volume
	Unit   Unit
}

func NewQuantity(value float64, unit Unit) (Quantity, error) {
	unit = Unit(strings.TrimSpace(string(unit)))
	if err := unit.Validate(); err != nil {
		return Quantity{}, err
	}

	var vol Volume
	switch unit {
	case UnitMl:
		vol = Milliliters(value)
	case UnitOz:
		vol = Ounces(value)
	case UnitCl:
		vol = Centiliters(value)
	case UnitDash, UnitPiece, UnitSplash:
		return Quantity{}, errors.Invalidf("unit %q is not a volume", unit)
	default:
		return Quantity{}, errors.Invalidf("unknown unit: %s", unit)
	}

	return Quantity{Volume: vol, Unit: unit}, nil
}

func MustQuantity(value float64, unit Unit) Quantity {
	q, err := NewQuantity(value, unit)
	if err != nil {
		panic(err)
	}
	return q
}

func (q Quantity) Value() float64 {
	switch q.Unit {
	case UnitOz:
		return q.Volume.Oz()
	case UnitCl:
		return q.Volume.Cl()
	default:
		return q.Volume.Ml()
	}
}

func (q Quantity) String() string {
	return fmt.Sprintf("%.2f %s", q.Value(), q.Unit)
}

func (q Quantity) Convert(unit Unit) (Quantity, error) {
	unit = Unit(strings.TrimSpace(string(unit)))
	if err := unit.Validate(); err != nil {
		return Quantity{}, err
	}
	switch unit {
	case UnitMl, UnitOz, UnitCl:
		return Quantity{Volume: q.Volume, Unit: unit}, nil
	case UnitDash, UnitPiece, UnitSplash:
		return Quantity{}, errors.Invalidf("unit %q is not a volume", unit)
	default:
		return Quantity{}, errors.Invalidf("unknown unit: %s", unit)
	}
}

func (q Quantity) Add(other Quantity) Quantity {
	return Quantity{Volume: q.Volume.Add(other.Volume), Unit: q.Unit}
}

func (q Quantity) Sub(other Quantity) Quantity {
	return Quantity{Volume: q.Volume.Sub(other.Volume), Unit: q.Unit}
}

func (q Quantity) Mul(n float64) Quantity {
	return Quantity{Volume: q.Volume.Mul(n), Unit: q.Unit}
}

func (q Quantity) Div(n float64) Quantity {
	return Quantity{Volume: q.Volume.Div(n), Unit: q.Unit}
}

func (q Quantity) IsZero() bool {
	return q.Volume.IsZero()
}

func (q Quantity) LessThan(other Quantity) bool {
	return q.Volume.LessThan(other.Volume)
}
