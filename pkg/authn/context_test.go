package authn_test

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestContextRoundTrip(t *testing.T) {
	t.Parallel()

	want := authn.Owner()
	ctx := authn.ToContext(context.Background(), want)
	testutil.Equals(t, authn.FromContext(ctx), want)
}

func TestFromContextPanicsWithoutPrincipal(t *testing.T) {
	t.Parallel()
	testutil.ExpectPanic(t, "no principal in context", func() {
		authn.FromContext(context.Background())
	})
}
