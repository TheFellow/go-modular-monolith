package gui_test

import (
	"errors"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestValidateReturnsFirstFailure(t *testing.T) {
	t.Parallel()
	first := errors.New("first")
	secondCalled := false
	err := gui.Validate("", func(string) error { return first }, func(string) error {
		secondCalled = true
		return nil
	})
	if !errors.Is(err, first) || secondCalled {
		testutil.ErrorIf(t, true, "err=%v secondCalled=%v", err, secondCalled)
	}
}

func TestRequiredUsesSurfaceSpecificMessage(t *testing.T) {
	t.Parallel()
	if err := gui.Required("name is required")(``); err == nil || err.Error() != "name is required" {
		testutil.ErrorIf(t, true, "error = %v", err)
	}
}
