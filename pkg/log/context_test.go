package log_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/log"
)

func TestContextRoundTrip(t *testing.T) {
	t.Parallel()

	want := slog.Default()
	ctx := log.ToContext(context.Background(), want)
	if got := log.FromContext(ctx); got != want {
		t.Fatalf("logger = %p, want %p", got, want)
	}
}

func TestFromContextPanicsWithoutLogger(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatal("expected missing logger to panic")
		}
	}()
	log.FromContext(context.Background())
}
