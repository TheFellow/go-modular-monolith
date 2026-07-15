package testutil_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestEqualsSupportsOptionalValues(t *testing.T) {
	t.Parallel()

	type document struct {
		Name optional.Value[string]
	}

	testutil.Equals(t,
		document{Name: optional.Some("mixology")},
		document{Name: optional.Some("mixology")},
	)
	testutil.Equals(t,
		document{Name: optional.None[string]()},
		document{Name: optional.None[string]()},
	)
}

func TestEqualsEquatesWrappedErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel")
	testutil.Equals[error](t, fmt.Errorf("context: %w", sentinel), sentinel)
}
