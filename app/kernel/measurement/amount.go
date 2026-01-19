package measurement

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

// Amount represents either a volume or discrete quantity.
type Amount interface {
	Unit() Unit
	Value() float64
	String() string
	IsZero() bool
	Convert(Unit) (Amount, error)
	Add(Amount) (Amount, error)
	Sub(Amount) (Amount, error)
	Mul(float64) Amount
	LessThan(Amount) (bool, error)
	isAmount()
}

// VolumeAmount wraps a volume quantity.
type VolumeAmount struct {
	Quantity Quantity
}

func (VolumeAmount) isAmount() {}

// DiscreteAmount wraps a discrete quantity.
type DiscreteAmount struct {
	Quantity DiscreteQuantity
}

func (DiscreteAmount) isAmount() {}

func NewVolumeAmount(q Quantity) Amount {
	return VolumeAmount{Quantity: q}
}

func NewDiscreteAmount(d DiscreteQuantity) Amount {
	return DiscreteAmount{Quantity: d}
}

func NewAmount(value float64, unit Unit) (Amount, error) {
	unit = Unit(strings.TrimSpace(string(unit)))
	if err := unit.Validate(); err != nil {
		return nil, err
	}
	switch unit {
	case UnitMl, UnitOz, UnitCl:
		q, err := NewQuantity(value, unit)
		if err != nil {
			return nil, err
		}
		return NewVolumeAmount(q), nil
	case UnitDash, UnitPiece, UnitSplash:
		d, err := NewDiscreteQuantity(value, unit)
		if err != nil {
			return nil, err
		}
		return NewDiscreteAmount(d), nil
	default:
		panic("unreachable")
	}
}

func MustAmount(value float64, unit Unit) Amount {
	a, err := NewAmount(value, unit)
	if err != nil {
		panic(err)
	}
	return a
}

func (v VolumeAmount) Unit() Unit {
	return v.Quantity.Unit
}

func (v VolumeAmount) Value() float64 {
	return v.Quantity.Value()
}

func (v VolumeAmount) String() string {
	return v.Quantity.String()
}

func (v VolumeAmount) IsZero() bool {
	return v.Quantity.IsZero()
}

func (v VolumeAmount) Convert(unit Unit) (Amount, error) {
	q, err := v.Quantity.Convert(unit)
	if err != nil {
		return nil, err
	}
	return NewVolumeAmount(q), nil
}

func (v VolumeAmount) Add(other Amount) (Amount, error) {
	if other == nil {
		return nil, errors.Invalidf("amount is empty")
	}
	switch o := other.(type) {
	case VolumeAmount:
		return NewVolumeAmount(v.Quantity.Add(o.Quantity)), nil
	case *VolumeAmount:
		return NewVolumeAmount(v.Quantity.Add(o.Quantity)), nil
	default:
		return nil, errors.Invalidf("unit mismatch: %s vs %s", v.Unit(), other.Unit())
	}
}

func (v VolumeAmount) Sub(other Amount) (Amount, error) {
	if other == nil {
		return nil, errors.Invalidf("amount is empty")
	}
	switch o := other.(type) {
	case VolumeAmount:
		return NewVolumeAmount(v.Quantity.Sub(o.Quantity)), nil
	case *VolumeAmount:
		return NewVolumeAmount(v.Quantity.Sub(o.Quantity)), nil
	default:
		return nil, errors.Invalidf("unit mismatch: %s vs %s", v.Unit(), other.Unit())
	}
}

func (v VolumeAmount) Mul(n float64) Amount {
	return NewVolumeAmount(v.Quantity.Mul(n))
}

func (v VolumeAmount) LessThan(other Amount) (bool, error) {
	if other == nil {
		return false, errors.Invalidf("amount is empty")
	}
	switch o := other.(type) {
	case VolumeAmount:
		return v.Quantity.LessThan(o.Quantity), nil
	case *VolumeAmount:
		return v.Quantity.LessThan(o.Quantity), nil
	default:
		return false, errors.Invalidf("unit mismatch: %s vs %s", v.Unit(), other.Unit())
	}
}

func (d DiscreteAmount) Unit() Unit {
	return d.Quantity.Unit
}

func (d DiscreteAmount) Value() float64 {
	return d.Quantity.Count
}

func (d DiscreteAmount) String() string {
	return d.Quantity.String()
}

func (d DiscreteAmount) IsZero() bool {
	return d.Quantity.IsZero()
}

func (d DiscreteAmount) Convert(unit Unit) (Amount, error) {
	if d.Quantity.Unit != unit {
		return nil, errors.Invalidf("unit mismatch: %s vs %s", d.Quantity.Unit, unit)
	}
	return NewDiscreteAmount(d.Quantity), nil
}

func (d DiscreteAmount) Add(other Amount) (Amount, error) {
	if other == nil {
		return nil, errors.Invalidf("amount is empty")
	}
	switch o := other.(type) {
	case DiscreteAmount:
		q, err := d.Quantity.Add(o.Quantity)
		if err != nil {
			return nil, err
		}
		return NewDiscreteAmount(q), nil
	case *DiscreteAmount:
		q, err := d.Quantity.Add(o.Quantity)
		if err != nil {
			return nil, err
		}
		return NewDiscreteAmount(q), nil
	default:
		return nil, errors.Invalidf("unit mismatch: %s vs %s", d.Unit(), other.Unit())
	}
}

func (d DiscreteAmount) Sub(other Amount) (Amount, error) {
	if other == nil {
		return nil, errors.Invalidf("amount is empty")
	}
	switch o := other.(type) {
	case DiscreteAmount:
		q, err := d.Quantity.Sub(o.Quantity)
		if err != nil {
			return nil, err
		}
		return NewDiscreteAmount(q), nil
	case *DiscreteAmount:
		q, err := d.Quantity.Sub(o.Quantity)
		if err != nil {
			return nil, err
		}
		return NewDiscreteAmount(q), nil
	default:
		return nil, errors.Invalidf("unit mismatch: %s vs %s", d.Unit(), other.Unit())
	}
}

func (d DiscreteAmount) Mul(n float64) Amount {
	return NewDiscreteAmount(d.Quantity.Mul(n))
}

func (d DiscreteAmount) LessThan(other Amount) (bool, error) {
	if other == nil {
		return false, errors.Invalidf("amount is empty")
	}
	switch o := other.(type) {
	case DiscreteAmount:
		if d.Quantity.Unit != o.Quantity.Unit {
			return false, errors.Invalidf("unit mismatch: %s vs %s", d.Quantity.Unit, o.Quantity.Unit)
		}
		return d.Quantity.Count < o.Quantity.Count, nil
	case *DiscreteAmount:
		if d.Quantity.Unit != o.Quantity.Unit {
			return false, errors.Invalidf("unit mismatch: %s vs %s", d.Quantity.Unit, o.Quantity.Unit)
		}
		return d.Quantity.Count < o.Quantity.Count, nil
	default:
		return false, errors.Invalidf("unit mismatch: %s vs %s", d.Unit(), other.Unit())
	}
}
