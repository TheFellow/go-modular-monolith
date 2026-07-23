package measurement_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestUnitValidate(t *testing.T) {
	t.Parallel()

	testutil.Ok(t, measurement.UnitOz.Validate())
	testutil.Ok(t, measurement.UnitCl.Validate())
	testutil.ErrorIsInvalid(t, measurement.Unit("").Validate())
	testutil.ErrorIsInvalid(t, measurement.Unit("unknown").Validate())
}
