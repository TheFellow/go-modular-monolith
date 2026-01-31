package testutil

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func PermissionTestFail(t testing.TB, err error) {
	t.Helper()
	if err == nil || !errors.IsPermission(err) {
		t.Fatalf("expected authz denied error, got %v", err)
	}
}

func PermissionTestPass(t testing.TB, err error) {
	t.Helper()
	if errors.IsPermission(err) {
		t.Fatalf("unexpected authz denied error: %v", err)
	}
}
