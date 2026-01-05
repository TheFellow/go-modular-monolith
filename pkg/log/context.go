package log

import (
	"context"
	"log/slog"
)

type loggerKey struct{}

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerKey{}, l)
}

func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return slog.Default()
}

func With(ctx context.Context, attrs ...slog.Attr) context.Context {
	logger := FromContext(ctx).With(Args(attrs...)...)
	return WithLogger(ctx, logger)
}

func Args(attrs ...slog.Attr) []any {
	args := make([]any, 0, len(attrs))
	for _, a := range attrs {
		if a.Key == "" {
			continue
		}
		args = append(args, a)
	}
	return args
}
