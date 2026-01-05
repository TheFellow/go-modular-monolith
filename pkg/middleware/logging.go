package middleware

import (
	"log/slog"
	"time"

	cedar "github.com/cedar-policy/cedar-go"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
)

func CommandLogging() CommandMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error {
		ctx.Context = log.WithLogAttrs(ctx.Context, log.Action(action), log.Resource(resource.UID))

		logger := log.FromContext(ctx)
		start := time.Now()
		logger.Debug("command started")

		err := next(ctx)

		if errors.IsPermission(err) {
			return err
		}

		if err != nil {
			logger.Error("command failed", slog.Duration("duration", time.Since(start)), log.Err(err))
		} else {
			logger.Info("command completed", slog.Duration("duration", time.Since(start)))
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

		if errors.IsPermission(err) {
			return err
		}

		if err != nil {
			logger.Warn("query failed", slog.Duration("duration", time.Since(start)), log.Err(err))
		} else {
			logger.Debug("query completed", slog.Duration("duration", time.Since(start)))
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

		if errors.IsPermission(err) {
			return err
		}

		if err != nil {
			logger.Warn("query failed", slog.Duration("duration", time.Since(start)), log.Err(err))
		} else {
			logger.Debug("query completed", slog.Duration("duration", time.Since(start)))
		}
		return err
	}
}
