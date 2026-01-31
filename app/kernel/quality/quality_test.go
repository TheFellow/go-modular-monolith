package quality_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/quality"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func TestQualityValidate(t *testing.T) {
	t.Parallel()

	if err := quality.Equivalent.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := quality.Quality("bad").Validate(); !errors.IsInvalid(err) {
		t.Fatalf("expected invalid, got %v", err)
	}
}
