package log

import (
	"context"
	"log/slog"
)

type loggerKey struct{}

// ToContext returns a new context with the logger attached.
func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext returns the logger from the context, or panics if there is none.
func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey{}).(*slog.Logger)
	if !ok {
		panic("no logger in context")
	}
	return logger
}
