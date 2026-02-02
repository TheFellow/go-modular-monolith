package log

import (
	"context"
	"log/slog"
)

type loggerKey struct{}
type rootLoggerKey struct{}

// ToContext attaches a logger to the context.
func ToContext(ctx context.Context, l *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(rootLoggerKey{}).(*slog.Logger); !ok {
		ctx = context.WithValue(ctx, rootLoggerKey{}, l)
	}
	return context.WithValue(ctx, loggerKey{}, l)
}

// WithLogger is kept for backwards-compatibility; prefer ToContext.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context { return ToContext(ctx, l) }

func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return slog.Default()
}

func RootFromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(rootLoggerKey{}).(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return FromContext(ctx)
}

// ResetLogger replaces the current logger with the root logger.
func ResetLogger(ctx context.Context) context.Context {
	return ToContext(ctx, RootFromContext(ctx))
}

// With enriches the context's logger with additional attributes.
func WithLogAttrs(ctx context.Context, attrs ...any) context.Context {
	return ToContext(ctx, FromContext(ctx).With(attrs...))
}
