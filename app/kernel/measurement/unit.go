package measurement

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type Unit string

const (
	UnitOz     Unit = "oz"
	UnitMl     Unit = "ml"
	UnitDash   Unit = "dash"
	UnitPiece  Unit = "piece"
	UnitSplash Unit = "splash"
)

func AllUnits() []Unit {
	return []Unit{
		UnitOz,
		UnitMl,
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
	for _, v := range AllUnits() {
		if u == v {
			return nil
		}
	}
	return errors.Invalidf("invalid unit %q", string(u))
}
