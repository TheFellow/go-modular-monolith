package testutil

import (
	"context"
	"errors"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func ActorContext(t testing.TB, actor string) *middleware.Context {
	t.Helper()

	p, err := authn.ParseActor(actor)
	Ok(t, err)
	return middleware.NewContext(context.Background(), middleware.WithPrincipal(p))
}

func IsDenied(err error) bool {
	return errors.Is(err, authz.ErrDenied)
}

func RequireDenied(t testing.TB, err error) {
	t.Helper()
	if err == nil || !IsDenied(err) {
		t.Fatalf("expected authz denied error, got %v", err)
	}
}

func RequireNotDenied(t testing.TB, err error) {
	t.Helper()
	if IsDenied(err) {
		t.Fatalf("unexpected authz denied error: %v", err)
	}
}
