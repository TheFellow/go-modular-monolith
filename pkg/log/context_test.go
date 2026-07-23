package log_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestContextRoundTrip(t *testing.T) {
	t.Parallel()

	want := slog.Default()
	ctx := log.ToContext(context.Background(), want)
	testutil.IsTrue(t, log.FromContext(ctx) == want)
}

func TestFromContextPanicsWithoutLogger(t *testing.T) {
	t.Parallel()
	testutil.ExpectPanic(t, "no logger in context", func() {
		log.FromContext(context.Background())
	})
}
