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

func TestCommonAssertionsAcceptExpectedValues(t *testing.T) {
	t.Parallel()

	var typedNil *int
	testutil.NotEquals(t, "got", "want")
	testutil.IsTrue(t, true)
	testutil.IsFalse(t, false)
	testutil.Nil(t, nil)
	testutil.Nil(t, typedNil)
	testutil.NotNil(t, 0)
	testutil.Equals(t, testutil.Cast[string](t, any("value")), "value")
	testutil.ErrorContains(t, errors.New("wrapped detail"), "detail")
	testutil.StringContains(t, "filter expression", "expression")
	testutil.ExpectPanic(t, "boom", func() { panic("boom") })
}
