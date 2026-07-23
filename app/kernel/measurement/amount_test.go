package measurement_test

import (
	"math"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestVolumeConversions(t *testing.T) {
	t.Parallel()

	v := measurement.Ounces(1)
	testutil.ErrorIf(t, math.Abs(v.Ml()-29.5735) > 0.0001, "expected 1 oz to be 29.5735 ml, got %.6f", v.Ml())

	v = measurement.Milliliters(29.5735)
	testutil.ErrorIf(t, math.Abs(v.Oz()-1.0) > 0.0001, "expected 29.5735 ml to be 1 oz, got %.6f", v.Oz())

	v = measurement.Centiliters(10)
	testutil.ErrorIf(t, math.Abs(v.Ml()-100.0) > 0.0001, "expected 10 cl to be 100 ml, got %.6f", v.Ml())
}

func TestQuantityConvert(t *testing.T) {
	t.Parallel()

	q, err := measurement.NewQuantity(1.5, measurement.UnitOz)
	testutil.Ok(t, err)

	q2, err := q.Convert(measurement.UnitMl)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, math.Abs(q2.Value()-44.36025) > 0.01, "expected 1.5 oz to be ~44.36 ml, got %.6f", q2.Value())
}

func TestAmountAddDiscreteMismatch(t *testing.T) {
	t.Parallel()

	a := measurement.MustAmount(1, measurement.UnitDash)
	b := measurement.MustAmount(1, measurement.UnitPiece)
	_, err := a.Add(b)
	testutil.ErrorIsInvalid(t, err)
}
