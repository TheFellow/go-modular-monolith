package middleware

import (
	"log/slog"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
)

func Logging() Middleware {
	return func(ctx *Context, op Operation, next Next) error {
		attrs := []any{log.Action(op.Action)}
		if op.HasResource() {
			attrs = append(attrs, log.Resource(op.Resource.UID))
		}
		ctx.Context = log.ToContext(ctx.Context, log.FromContext(ctx).With(attrs...))

		logger := log.FromContext(ctx)
		start := time.Now()
		logger.Debug(string(op.Kind) + " started")

		err := next(ctx)
		duration := time.Since(start)

		if op.Kind == OperationKindCommand {
			if activity, ok := ctx.Activity(); ok && !activity.Resource.IsZero() {
				ctx.Context = log.ToContext(ctx.Context, log.FromContext(ctx).With(log.Resource(activity.Resource)))
				logger = log.FromContext(ctx)
			}
		}

		switch {
		case err == nil:
			logOperationCompleted(logger, op.Kind, duration)
		case errors.IsPermission(err):
			logger.Info(string(op.Kind)+" denied", slog.Duration("duration", duration), log.Err(err))
		default:
			logOperationFailed(logger, op.Kind, duration, err)
		}
		return err
	}
}

func logOperationCompleted(logger *slog.Logger, kind OperationKind, duration time.Duration) {
	switch kind {
	case OperationKindCommand:
		logger.Info("command completed", slog.Duration("duration", duration))
	case OperationKindQuery:
		logger.Debug("query completed", slog.Duration("duration", duration))
	}
}

func logOperationFailed(logger *slog.Logger, kind OperationKind, duration time.Duration, err error) {
	switch kind {
	case OperationKindCommand:
		logger.Error("command failed", slog.Duration("duration", duration), log.Err(err))
	case OperationKindQuery:
		logger.Warn("query failed", slog.Duration("duration", duration), log.Err(err))
	}
}
