package measurement

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type Unit string

const (
	UnitOz     Unit = "oz"
	UnitMl     Unit = "ml"
	UnitCl     Unit = "cl"
	UnitDash   Unit = "dash"
	UnitPiece  Unit = "piece"
	UnitSplash Unit = "splash"
)

func AllUnits() []Unit {
	return []Unit{
		UnitOz,
		UnitMl,
		UnitCl,
		UnitDash,
		UnitPiece,
		UnitSplash,
	}
}

func (u Unit) Validate() error {
	u = Unit(strings.TrimSpace(string(u)))
	if u == "" {
		return errors.Invalidf("unit is required")
	}
	switch u {
	case UnitOz, UnitMl, UnitCl, UnitDash, UnitPiece, UnitSplash:
		return nil
	default:
		return errors.Invalidf("invalid unit %q", string(u))
	}
}

// Canonical returns the stable storage unit for this dimension. Display units
// never determine the scale of persisted inventory or reservations.
func (u Unit) Canonical() Unit {
	switch u {
	case UnitOz, UnitCl, UnitMl:
		return UnitMl
	case UnitDash, UnitPiece, UnitSplash:
		return u
	default:
		return u
	}
}
