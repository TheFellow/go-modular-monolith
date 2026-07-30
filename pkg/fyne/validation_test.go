package fyne_test

import (
	"errors"
	"testing"

	fyneui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
)

func TestValidateReturnsFirstFailure(t *testing.T) {
	t.Parallel()
	first := errors.New("first")
	secondCalled := false
	err := fyneui.Validate("", func(string) error { return first }, func(string) error {
		secondCalled = true
		return nil
	})
	if !errors.Is(err, first) || secondCalled {
		t.Fatalf("err=%v secondCalled=%v", err, secondCalled)
	}
}

func TestRequiredUsesSurfaceSpecificMessage(t *testing.T) {
	t.Parallel()
	if err := fyneui.Required("name is required")(``); err == nil || err.Error() != "name is required" {
		t.Fatalf("error = %v", err)
	}
}
