package measurement_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func TestUnitValidate(t *testing.T) {
	t.Parallel()

	if err := measurement.UnitOz.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := measurement.UnitCl.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := measurement.Unit("").Validate(); !errors.IsInvalid(err) {
		t.Fatalf("expected invalid for empty, got %v", err)
	}
	if err := measurement.Unit("unknown").Validate(); !errors.IsInvalid(err) {
		t.Fatalf("expected invalid for unknown, got %v", err)
	}
}
