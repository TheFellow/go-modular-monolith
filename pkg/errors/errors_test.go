package errors_test

import (
	"fmt"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestConstructorsAndCheckers(t *testing.T) {
	t.Parallel()

	inv := errors.Invalidf("name is required")
	if !errors.IsInvalid(inv) || errors.IsNotFound(inv) || errors.IsInternal(inv) {
		t.Fatalf("unexpected kind: %T %v", inv, inv)
	}

	if ex, ok := inv.(interface{ ExitCode() int }); !ok || ex.ExitCode() != errors.ExitInvalid {
		t.Fatalf("expected ExitCode %d, got %T %#v", errors.ExitInvalid, inv, inv)
	}

	wrapped := fmt.Errorf("outer: %w", inv)
	if !errors.IsInvalid(wrapped) {
		t.Fatalf("expected IsInvalid to match wrapped error")
	}

	var invErr *errors.InvalidError
	if !errors.As(wrapped, &invErr) || invErr.HTTPCode() != 400 || invErr.GRPCCode() != 3 {
		t.Fatalf("expected As(*InvalidError) with codes, got %+v", invErr)
	}
}

func TestConstructorUnwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("root")
	err := errors.Internalf("boom: %w", cause)

	testutil.ErrorIsInternal(t, err)
	testutil.ErrorIf(t, errors.Unwrap(err) != cause, "got %v, want %v", errors.Unwrap(err), cause)
}

func TestWrapOnlyStillUnwraps(t *testing.T) {
	t.Parallel()

	cause := errors.New("root")
	err := errors.Internalf("%w", cause)

	testutil.ErrorIsInternal(t, err)
	testutil.ErrorIf(t, errors.Unwrap(err) != cause, "got %v, want %v", errors.Unwrap(err), cause)
}
