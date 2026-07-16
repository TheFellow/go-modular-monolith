package authn_test

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/authn"
)

func TestContextRoundTrip(t *testing.T) {
	t.Parallel()

	want := authn.Owner()
	ctx := authn.ToContext(context.Background(), want)
	if got := authn.FromContext(ctx); got != want {
		t.Fatalf("principal = %s, want %s", got, want)
	}
}

func TestFromContextPanicsWithoutPrincipal(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatal("expected missing principal to panic")
		}
	}()
	authn.FromContext(context.Background())
}
