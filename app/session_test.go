package app_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestSessionContextFromPreservesIdentityAndAcceptsCancellation(t *testing.T) {
	t.Parallel()
	root, cancelSession := context.WithCancel(context.Background())
	base := authn.ToContext(root, authn.Sommelier())
	base = pkglog.ToContext(base, slog.Default())
	session := app.NewSession(base, nil)
	lifetime, cancel := context.WithCancel(context.Background())
	ctx := session.ContextFrom(lifetime)
	testutil.Equals(t, ctx.Principal(), authn.Sommelier())
	cancel()
	<-ctx.Done()
	testutil.ErrorIf(t, !errors.Is(ctx.Err(), context.Canceled), "context error = %v", ctx.Err())
	sessionCtx := session.ContextFrom(context.Background())
	cancelSession()
	<-sessionCtx.Done()
	testutil.ErrorIf(t, !errors.Is(sessionCtx.Err(), context.Canceled), "session context error = %v", sessionCtx.Err())
}
