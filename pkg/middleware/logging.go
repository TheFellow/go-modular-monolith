package middleware

import (
	"log/slog"
	"time"

	"github.com/cedar-policy/cedar-go"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
)

func CommandLogging() CommandMiddleware {
	return func(ctx *Context, action cedar.EntityUID, next CommandNext) error {
		ctx.Context = log.WithLogAttrs(ctx.Context, log.Action(action))

		logger := log.FromContext(ctx)
		start := time.Now()
		logger.Debug("command started")

		err := next(ctx)
		duration := time.Since(start)

		if activity, ok := ActivityFromContext(ctx.Context); ok && !activity.Resource.IsZero() {
			ctx.Context = log.WithLogAttrs(ctx.Context, log.Resource(activity.Resource))
			logger = log.FromContext(ctx)
		}

		switch {
		case err == nil:
			logger.Info("command completed", slog.Duration("duration", duration))
		case errors.IsPermission(err):
			logger.Info("command denied", slog.Duration("duration", duration), log.Err(err))
		default:
			logger.Error("command failed", slog.Duration("duration", duration), log.Err(err))
		}
		return err
	}
}

func QueryLogging() QueryMiddleware {
	return func(ctx *Context, action cedar.EntityUID, next QueryNext) error {
		ctx.Context = log.WithLogAttrs(ctx.Context, log.Action(action))

		logger := log.FromContext(ctx)
		start := time.Now()
		logger.Debug("query started")

		err := next(ctx)
		duration := time.Since(start)

		switch {
		case err == nil:
			logger.Debug("query completed", slog.Duration("duration", duration))
		case errors.IsPermission(err):
			logger.Info("query denied", slog.Duration("duration", duration), log.Err(err))
		default:
			logger.Warn("query failed", slog.Duration("duration", duration), log.Err(err))
		}
		return err
	}
}

func QueryWithResourceLogging() QueryWithResourceMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next QueryWithResourceNext) error {
		ctx.Context = log.WithLogAttrs(ctx.Context, log.Action(action), log.Resource(resource.UID))

		logger := log.FromContext(ctx)
		start := time.Now()
		logger.Debug("query started")

		err := next(ctx)
		duration := time.Since(start)

		switch {
		case err == nil:
			logger.Debug("query completed", slog.Duration("duration", duration))
		case errors.IsPermission(err):
			logger.Info("query denied", slog.Duration("duration", duration), log.Err(err))
		default:
			logger.Warn("query failed", slog.Duration("duration", duration), log.Err(err))
		}
		return err
	}
}
