package testutil

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func ActorContext(t testing.TB, actor string) *middleware.Context {
	t.Helper()

	p, err := authn.ParseActor(actor)
	Ok(t, err)
	return middleware.NewContext(context.Background(), middleware.WithPrincipal(p))
}

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
