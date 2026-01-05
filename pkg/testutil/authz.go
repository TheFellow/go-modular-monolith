package testutil

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func RequireDenied(t testing.TB, err error) {
	t.Helper()
	if err == nil || !errors.IsPermission(err) {
		t.Fatalf("expected authz denied error, got %v", err)
	}
}

func RequireNotDenied(t testing.TB, err error) {
	t.Helper()
	if errors.IsPermission(err) {
		t.Fatalf("unexpected authz denied error: %v", err)
	}
}
