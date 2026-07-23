package quality_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/quality"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestQualityValidate(t *testing.T) {
	t.Parallel()

	testutil.Ok(t, quality.Equivalent.Validate())
	testutil.ErrorIsInvalid(t, quality.Quality("bad").Validate())
}
